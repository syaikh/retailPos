#!/bin/bash
set -e

# =============================================================================
# Retail POS System - Podman Deployment Script (Refactored)
# =============================================================================
# Usage:
#   ./deploy/podman-deploy.sh build [backend|frontend]
#   ./deploy/podman-deploy.sh start [postgres|backend|frontend|all]
#   ./deploy/podman-deploy.sh stop [postgres|backend|frontend|all]
#   ./deploy/podman-deploy.sh migrate
#   ./deploy/podman-deploy.sh seed
#   ./deploy/podman-deploy.sh status
#   ./deploy/podman-deploy.sh logs [backend|frontend|postgres|all]
# =============================================================================

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$SCRIPT_DIR"

# Configuration
POD_NAME="retail-pos-pod"
NETWORK_NAME="retail-pos-network"
# 8000, not 5173: 5173 is the Vite dev server's port (FRONTEND_PORT in the root
# .env.example), and a developer running `npm run dev` on this machine would
# collide with a production stack published there. 8000 is unprivileged, so
# rootless podman can bind it, and nothing else in the stack claims it — the
# backend holds 8080 and postgres holds 5432, both on loopback.
HOST_FRONTEND_PORT="${HOST_FRONTEND_PORT:-8000}"

# Address the frontend's nginx proxies /api/, /ws/ and /health to. It is
# substituted into deploy/nginx/default.conf.template at container start.
# 127.0.0.1 is correct for the single-pod layout, where nginx and the Go backend
# share the pod's network namespace. For a split deployment, set BACKEND_HOST to
# the backend machine's address and run the frontend outside this pod.
BACKEND_HOST="${BACKEND_HOST:-127.0.0.1}"
BACKEND_PORT="${BACKEND_PORT:-8080}"

# Secret file. Holds DB_PASSWORD, JWT_SECRET and (optionally) JWT_SECRET_REFRESH.
#
# Sourced below so this script can use those values, and passed to the backend
# container with --env-file so they never appear as `-e NAME=value` arguments,
# which would expose them in the process table.
#
# Note the limit of this mechanism: --env-file moves values out of argv but not
# out of the container's configuration. Anyone able to run `podman inspect
# backend` can still read them. The file must stay mode 600, and if that is not
# sufficient isolation, use podman secrets instead.
#
# Create it before the first start:
#   sudo mkdir -p /etc/retail-pos
#   sudo tee /etc/retail-pos/backend.env >/dev/null <<EOF
#   DB_PASSWORD=$(openssl rand -hex 24)
#   JWT_SECRET=$(openssl rand -hex 32)
#   JWT_SECRET_REFRESH=$(openssl rand -hex 32)
#   EOF
#   sudo chmod 600 /etc/retail-pos/backend.env
#
# Keep a separate JWT_SECRET_REFRESH so the two token types can be rotated
# independently; the backend falls back to reusing JWT_SECRET if it is unset.
ENV_FILE="${ENV_FILE:-/etc/retail-pos/backend.env}"
if [ -f "$ENV_FILE" ]; then
    set -a
    # shellcheck disable=SC1090
    . "$ENV_FILE"
    set +a
fi

# Image names
BACKEND_IMAGE="${BACKEND_IMAGE:-localhost/retail-pos-backend:latest}"
FRONTEND_IMAGE="${FRONTEND_IMAGE:-localhost/retail-pos-frontend:latest}"
POSTGRES_IMAGE="${POSTGRES_IMAGE:-docker.io/library/postgres:18-alpine}"

# Database configuration
DB_NAME="${DB_NAME:-retail_pos}"
DB_USER="${DB_USER:-pos}"
# No default on purpose: a shipped default would silently become the production
# database password. Supplied by $ENV_FILE and checked in validate_backend_config.
DB_PASSWORD="${DB_PASSWORD:-}"
DB_PORT="${DB_PORT:-5432}"
POSTGRES_PASSWORD="${POSTGRES_PASSWORD:-${DB_PASSWORD}}"
# `podman run -e POSTGRES_PASSWORD` (no "=value") copies an *exported* variable
# from the caller's environment. The line above is a plain shell assignment made
# after `set +a`, so without this export a fresh database init fails with
# "Database is uninitialized and superuser password is not specified". Exporting
# keeps the value out of argv (unlike `-e POSTGRES_PASSWORD="$..."`).
export POSTGRES_PASSWORD

# Browser origin allowed by the API. Must be a SINGLE origin.
#
# The CORS middleware gets it as []string{cfg.CORSOrigin} with no comma
# splitting, and the WebSocket upgrade check does an exact string comparison
# (pkg/websocket/hub.go). A comma-separated list therefore matches neither.
CORS_ORIGIN="${CORS_ORIGIN:-http://localhost:${HOST_FRONTEND_PORT}}"

# Secure flag on the 7-day HttpOnly refresh_token cookie, which the backend sets with
# secure=(COOKIE_SECURE == "true") in internal/user/auth_handler.go.
#
# Defaults to false because the deployment this script ships serves PLAIN HTTP:
# deploy/nginx/default.conf.template has only `listen 8081;` with no ssl, and
# start_frontend publishes that container on the LAN. A browser refuses to store a
# Secure cookie received over a non-localhost http origin, so with `true` the
# refresh cookie is silently dropped for every remote register. The access token
# lives in sessionStorage (web/src/shared/api/http-client.ts), which makes cookie
# refresh the only path back from an expired access token -- so the session simply
# dies at expiry on every till except a browser sitting on that same machine.
#
# Set `true` ONLY when TLS is terminated in front of nginx. Hardening this flag
# without also terminating TLS is what breaks auth, so the scheme decides, not the
# flag. See PRODUCTION-DEPLOYMENT.md for the TLS-fronted layout.
COOKIE_SECURE="${COOKIE_SECURE:-false}"

# libpq sslmode for the backend's connection to PostgreSQL.
#
# `require` is the preferred posture, but the stock postgres:18-alpine image
# cannot satisfy it: it ships with ssl=off unless a certificate and key are
# mounted, so a hard default of `require` makes a fresh `start all` abort
# before anything is deployed. wait_for_postgres therefore treats `require` as
# the preferred *default* rather than a guaranteed value: when the database
# turns out to have ssl=off and the operator never chose a mode, it downgrades
# to `disable` with a warning. An explicitly configured mode is never
# downgraded, so a deliberate `require` still fails loudly.
#
#   require       - correct when TLS certs are mounted (see PRODUCTION-DEPLOYMENT.md).
#                   The preferred posture, and the default when one can be met.
#   disable       - valid here because 5432 is bound to 127.0.0.1 only and the
#                   backend reaches the database over the pod's private network.
#                   The backend logs a warning at startup when it sees this in production.
#
# An invalid value is rejected up front: the backend otherwise falls back to its
# own default of `require`, which is exactly the ssl=off crash-loop that the
# probe in wait_for_postgres exists to prevent.
DB_SSLMODE_SET_EXPLICITLY=0
if [ -n "${DB_SSLMODE:-}" ]; then
    DB_SSLMODE_SET_EXPLICITLY=1
fi
DB_SSLMODE="${DB_SSLMODE:-require}"

# Volume names
POSTGRES_VOLUME="retail-pos-postgres-data"
# Store logo uploads. The backend image sets WORKDIR /app and resolves the logo
# directory as the relative path "uploads/logos" (see internal/appsettings/handler.go),
# so without this volume the shop's logo is lost on every image rebuild. Created by
# start_backend, which is the only component that mounts it.
UPLOADS_VOLUME="retail-pos-uploads"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# Checks
pod_exists() { podman pod exists "$POD_NAME" 2>/dev/null || return 1; }
container_exists() { podman container exists "$1" 2>/dev/null || return 1; }

# Fails fast before any container is created. Without this the backend starts,
# panics on the missing JWT_SECRET, and --restart unless-stopped turns a missing
# setting into a silent crash loop whose only symptom is a failing healthcheck.
validate_backend_config() {
    local missing=0

    if [ -z "$JWT_SECRET" ]; then
        log_error "JWT_SECRET is not set."
        missing=1
    elif [ "$JWT_SECRET" = "dev-jwt-secret-change-in-production" ]; then
        log_error "JWT_SECRET is still the development placeholder from .env.example."
        log_error "Generate a real one: openssl rand -hex 32"
        missing=1
    fi

    if [ -z "$DB_PASSWORD" ]; then
        log_error "DB_PASSWORD is not set."
        missing=1
    fi

    # Reject an unusable sslmode here. Left unchecked it reaches the backend,
    # which silently falls back to its own default of `require` in production —
    # the very mode settle_db_sslmode refuses to let pass against an ssl=off
    # database, so the operator would get a crash loop with no mention of the
    # typo that caused it.
    #
    # Reported separately from the missing-secret block: a typo needs the operator
    # to fix a value they already set, and printing the secret-creation recipe
    # alongside it sends them to the wrong file.
    case "$DB_SSLMODE" in
        disable|require|verify-ca|verify-full) ;;
        *)
            log_error "DB_SSLMODE=$DB_SSLMODE is not a valid libpq sslmode."
            log_error "Valid values: disable, require, verify-ca, verify-full"
            log_error "Fix the value in $ENV_FILE (or unset it to let the script choose)."
            return 1
            ;;
    esac

    if [ "$missing" -ne 0 ]; then
        log_error ""
        log_error "Add the missing values to $ENV_FILE and re-run. Create it with:"
        log_error "  sudo mkdir -p $(dirname "$ENV_FILE")"
        log_error "  sudo tee $ENV_FILE >/dev/null <<EOF"
        log_error "  DB_PASSWORD=\$(openssl rand -hex 24)"
        log_error "  JWT_SECRET=\$(openssl rand -hex 32)"
        log_error "  JWT_SECRET_REFRESH=\$(openssl rand -hex 32)"
        log_error "  EOF"
        log_error "  sudo chmod 600 $ENV_FILE"
        return 1
    fi

    log_info "Backend configuration validated (secrets from $ENV_FILE)"
}

# settle_db_sslmode decides the sslmode the backend will actually be given.
#
# It must be callable from start_backend as well as start_postgres: `start
# backend` is a documented command that runs without start_postgres, so a probe
# that lived only in the readiness wait left that path handing the backend a
# mode it had never checked. The result is a crash loop whose logs blame TLS and
# never mention the missing probe.
#
# pg_isready connects over the unix socket and never negotiates TLS, so it
# reports ready even when the server holds no certificate. That makes the
# following check, not readiness, the thing that actually prevents the loop.
settle_db_sslmode() {
    case "$DB_SSLMODE" in
        require|verify-ca|verify-full)
            local ssl_on
            ssl_on=$(podman exec postgres psql -U "$DB_USER" -tAc "SHOW ssl;" 2>/dev/null | tr -d '[:space:]')
            if [ "$ssl_on" != "on" ]; then
                if [ "$DB_SSLMODE_SET_EXPLICITLY" -eq 1 ]; then
                    # The operator asked for TLS, so honour it and fail. Silently
                    # downgrading here would ship plaintext database traffic under
                    # a setting that reads as encrypted.
                    log_error "PostgreSQL has ssl=off, but DB_SSLMODE=$DB_SSLMODE was set explicitly."
                    log_error "The backend would crash-loop with 'server refused TLS connection'."
                    log_error "Either mount a server certificate and key on the postgres container (see"
                    log_error "deploy/PRODUCTION-DEPLOYMENT.md), or change DB_SSLMODE to disable in"
                    log_error "the secret file and record that the database is confined to the pod network."
                    return 1
                fi
                # No mode was chosen, so there is no decision to contradict. The
                # database is published on 127.0.0.1 only and reached over the
                # pod's private network, which is the confinement `disable`
                # requires, so fall back and say so.
                log_warn "PostgreSQL has ssl=off and no DB_SSLMODE was set, so TLS cannot be used."
                log_warn "Continuing with DB_SSLMODE=disable. This is acceptable here because the"
                log_warn "database is bound to 127.0.0.1 and reached over the pod's private network."
                log_warn "To encrypt it instead, mount a certificate on the postgres container and set"
                log_warn "DB_SSLMODE=require (see deploy/PRODUCTION-DEPLOYMENT.md)."
                DB_SSLMODE=disable
            else
                log_info "PostgreSQL TLS is enabled (ssl=on), satisfying DB_SSLMODE=$DB_SSLMODE"
            fi
            ;;
    esac
}

ensure_pod() {
    if ! pod_exists; then
        log_info "Creating pod '$POD_NAME'..."
        # 5432 and 8080 are bound to 127.0.0.1 deliberately. Unqualified
        # `-p PORT:PORT` publishes on every host interface, which puts the
        # database and the unauthenticated API in reach of anything that can
        # route to this host. The backend reaches the database over the pod's
        # own network namespace, and nothing outside this machine should reach
        # either port; the only client that needs a host-side listener is
        # `seed`, which connects to 127.0.0.1:5432.
        #
        # The frontend keeps its public port, because a browser on the store
        # LAN has to reach it.
        podman pod create \
            --name "$POD_NAME" \
            --network bridge \
            -p "${HOST_FRONTEND_PORT}:8081" \
            -p "127.0.0.1:5432:5432" \
            -p "127.0.0.1:8080:8080"
    fi
}

wait_for_postgres() {
    log_info "Waiting for PostgreSQL to be ready..."
    local max_attempts=30
    local attempt=0
    while [ $attempt -lt $max_attempts ]; do
        if podman exec postgres pg_isready -U "$DB_USER" >/dev/null 2>&1; then
            break
        fi
        attempt=$((attempt + 1))
        echo -n "."
        sleep 2
    done
    if [ "$attempt" -ge "$max_attempts" ]; then
        log_error "PostgreSQL did not become ready in time"
        return 1
    fi

    settle_db_sslmode

    log_info "PostgreSQL is ready!"
    return 0
}

wait_for_backend() {
    log_info "Waiting for backend API to be ready..."
    local max_attempts=30
    local attempt=0
    while [ $attempt -lt $max_attempts ]; do
        if podman exec backend curl -s -o /dev/null http://localhost:8080/health 2>/dev/null; then
            log_info "Backend API is ready!"
            return 0
        fi
        attempt=$((attempt + 1))
        echo -n "."
        sleep 2
    done
    log_error "Backend API did not become ready in time"
    # Without this the operator sees a timeout and no cause, which is how the
    # missing-JWT_SECRET and refused-TLS failures were both hard to diagnose.
    log_error "Last 20 lines of backend output:"
    podman logs --tail 20 backend 2>&1 | sed 's/^/    /'
    return 1
}

# The frontend image is a `COPY web/dist/` of a bundle the Dockerfile never
# builds, so `podman build` silently packages whatever dist happens to be on
# disk. A dist left over from an earlier commit ships the old JavaScript while
# reporting a perfectly successful build, which is how a source fix reaches a
# deployment as a no-op. Rebuild the bundle first whenever anything the build
# reads is newer than the last build.
frontend_dist_is_stale() {
    local stamp="web/dist/index.html"
    [ -f "$stamp" ] || return 0

    local watched=(web/src web/index.html web/vite.config.js web/package.json)
    # The env file is the ROOT .env, not web/.env: web/vite.config.js loads it with
    # dotenv.config({ path: '../.env' }), and it is the only source of the build-time
    # VITE_* values (VITE_PRINT_MODE, VITE_PRINT_AGENT_URL) baked into the bundle by
    # web/src/shared/stores/printConfig.svelte.ts. web/.env exists nowhere in the tree,
    # so watching it meant a root-.env print-agent change left dist looking fresh and
    # the previous bundle shipped behind a successful-looking build.
    [ -f .env ] && watched+=(.env)
    [ -d web/public ] && watched+=(web/public)

    [ -n "$(find "${watched[@]}" -newer "$stamp" -print -quit 2>/dev/null)" ]
}

build_frontend_assets() {
    if [ "${SKIP_FRONTEND_BUILD:-0}" = "1" ]; then
        log_warn "SKIP_FRONTEND_BUILD=1 set; not checking whether web/dist is stale"
        return 0
    fi

    if ! frontend_dist_is_stale; then
        log_info "web/dist is up to date"
        return 0
    fi

    if ! command -v npm >/dev/null 2>&1; then
        log_error "web/dist is stale or missing and npm was not found."
        log_error "Build the bundle where Node is available, then retry: (cd web && npm ci && npm run build)"
        return 1
    fi
    if [ ! -d web/node_modules ]; then
        log_error "web/node_modules is missing. Run: (cd web && npm ci)"
        return 1
    fi

    log_info "web/dist is stale or missing; building the frontend bundle..."
    (cd web && npm run build) || { log_error "npm run build failed"; return 1; }
}

# Service Management
build_image() {
    local service=$1
    log_info "Building latest $service image..."
    case "$service" in
        backend)
            podman build -t "$BACKEND_IMAGE" -f deploy/backend/Dockerfile .
            ;;
        frontend)
            build_frontend_assets
            podman build -t "$FRONTEND_IMAGE" -f deploy/frontend/Dockerfile .
            ;;
    esac
}

start_postgres() {
    ensure_pod
    # Guarded here too because `start postgres` runs without start_backend, and
    # an empty POSTGRES_PASSWORD would otherwise start a database that no
    # configured password can reach.
    if [ -z "$POSTGRES_PASSWORD" ]; then
        log_error "POSTGRES_PASSWORD is not set. Add DB_PASSWORD to $ENV_FILE (see validate_backend_config)."
        return 1
    fi
    podman volume create "$POSTGRES_VOLUME" 2>/dev/null || true

    if container_exists "postgres"; then
        local status=$(podman container inspect postgres --format '{{.State.Status}}')
        if [ "$status" == "running" ]; then
            log_info "PostgreSQL is already running"
        else
            log_info "Starting existing PostgreSQL container..."
            podman start postgres
        fi
    else
        log_info "Starting new PostgreSQL container..."
        podman run -d \
            --pod "$POD_NAME" \
            --name postgres \
            -e POSTGRES_USER="$DB_USER" \
            -e POSTGRES_PASSWORD \
            -e POSTGRES_DB="$DB_NAME" \
            -v "$POSTGRES_VOLUME:/var/lib/postgresql" \
            --restart unless-stopped \
            "$POSTGRES_IMAGE"
    fi
    wait_for_postgres
}

start_backend() {
    ensure_pod
    validate_backend_config || return 1
    # Not only in wait_for_postgres: `start backend` reaches this point without
    # the readiness probe ever running, and passing an unsatisfiable mode here is
    # what puts the backend into the crash loop the probe exists to prevent.
    settle_db_sslmode || return 1
    # Migrations must land BEFORE the new binary does, and this function replaces the
    # backend container unconditionally, so `start backend` (and `start all`, which has
    # no migrate step of its own) would otherwise put a new binary on an old schema.
    # cmd/server/main.go performs no startup migration, so nothing downstream would
    # catch it. Abort before touching the container: a migration such as
    # 059_store_fk_integrity.sql validates every existing row, so a failure part-way
    # through the file loop leaves a partially migrated database, and continuing would
    # serve it from a freshly started server.
    #
    # Re-running is safe by design -- every migration in database/migrations is
    # permanently re-runnable (guarded DDL), and the schema_migrations ledger makes the
    # bookkeeping idempotent.
    migrate || {
        log_error "Migrations failed; refusing to start a new backend binary against an unmigrated database"
        return 1
    }
    build_image backend
    if container_exists "backend"; then
        log_info "Replacing existing backend container..."
        podman stop backend 2>/dev/null || true
        podman rm backend 2>/dev/null || true
    fi
    log_info "Starting Go backend container..."
    # --env-file carries the secrets; -e carries the non-secret wiring. ENV=production
    # selects JSON logging and stops the debug default. CORS_ORIGIN replaces the old
    # FRONTEND_URL, which no Go code ever read.
    #
    # COOKIE_SECURE follows the scheme actually served -- it is derived near the top of
    # this script and defaults to false, because the nginx this script ships is plain
    # HTTP. The 7-day HttpOnly refresh_token cookie is set with
    # secure=(COOKIE_SECURE == "true") in internal/user/auth_handler.go, and a browser
    # refuses to store a Secure cookie from a non-localhost http origin, so pinning it
    # true here would silently drop the cookie and end the session at access-token
    # expiry on every remote register. Set it true only behind TLS termination.
    # COOKIE_DOMAIN is deliberately left unset: host-only is the correct default, and
    # setting it too broadly would share the refresh token across subdomains.
    podman volume create "$UPLOADS_VOLUME" 2>/dev/null || true
    podman run -d \
        --pod "$POD_NAME" \
        --name backend \
        --env-file "$ENV_FILE" \
        -v "$UPLOADS_VOLUME:/app/uploads" \
        -e DB_HOST=localhost \
        -e DB_PORT="$DB_PORT" \
        -e DB_USER="$DB_USER" \
        -e DB_NAME="$DB_NAME" \
        -e DB_SSLMODE="$DB_SSLMODE" \
        -e PORT=8080 \
        -e ENV=production \
        -e LOG_LEVEL=info \
        -e CORS_ORIGIN="$CORS_ORIGIN" \
        -e COOKIE_SECURE="$COOKIE_SECURE" \
        -e GIN_MODE=release \
        --restart unless-stopped \
        "$BACKEND_IMAGE"
    wait_for_backend
}

start_frontend() {
    ensure_pod
    build_image frontend
    if container_exists "frontend"; then
        log_info "Replacing existing frontend container..."
        podman stop frontend 2>/dev/null || true
        podman rm frontend 2>/dev/null || true
    fi
    log_info "Starting Nginx frontend container..."
    podman run -d \
        --pod "$POD_NAME" \
        --name frontend \
        -e BACKEND_HOST="$BACKEND_HOST" \
        -e BACKEND_PORT="$BACKEND_PORT" \
        --restart unless-stopped \
        "$FRONTEND_IMAGE"
}

migrate() {
    log_info "Running database migrations..."
    if ! container_exists "postgres"; then
        log_error "PostgreSQL must be running to migrate. Run: $0 start postgres"
        return 1
    fi

    # Create database if not exists
    if podman exec postgres psql -U "$DB_USER" -lqt | cut -d\| -f1 | grep -qw "$DB_NAME"; then
        log_info "Database '$DB_NAME' already exists"
    else
        log_info "Creating database '$DB_NAME'..."
        podman exec postgres createdb -U "$DB_USER" "$DB_NAME"
    fi

    # Bootstrap prerequisites that 000_baseline.sql depends on (fresh-DB spin-up).
    # pgcrypto + invoice_seq are required by the schema; schema_migrations tracks
    # applied files and must exist before the first migration runs.
    log_info "Bootstrapping schema_migrations, pgcrypto, invoice_seq..."
    if ! podman exec postgres psql -U "$DB_USER" -d "$DB_NAME" -v ON_ERROR_STOP=1 \
        -c "CREATE EXTENSION IF NOT EXISTS pgcrypto;" \
        -c "CREATE SEQUENCE IF NOT EXISTS invoice_seq START 1;" \
        -c "CREATE TABLE IF NOT EXISTS schema_migrations (
               filename VARCHAR(255) PRIMARY KEY,
               applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
           )"; then
        log_error "Migration bootstrap failed"
        return 1
    fi

    local migration_dir="$SCRIPT_DIR/database/migrations"
    # P2-4 (026_shift_open_unique.sql): in dev/dummy-data environments allow the
    # migration to auto-close older duplicate open shifts; production (ENV=production
    # or unset) keeps the default fail-loud guard so no open shifts are silently lost.
    local pgoptions="-c app.shift_migration_mode=fail"
    if [ "$ENV" != "production" ] && [ -n "$ENV" ]; then
        pgoptions="-c app.shift_migration_mode=auto-close"
    fi
    for sql_file in "$migration_dir"/*.sql; do
        if [ -f "$sql_file" ]; then
            log_info "  Migrating: $(basename "$sql_file")"
            podman exec -i -e PGOPTIONS="$pgoptions" postgres psql -U "$DB_USER" -d "$DB_NAME" -v ON_ERROR_STOP=1 < "$sql_file" || {
                log_error "Migration failed: $(basename "$sql_file")"
                return 1
            }
            # Record the applied file for the audit trail. 000_baseline.sql
            # registers itself as its final step; any migration added after the
            # squash is recorded here instead. The ON CONFLICT guard keeps the
            # two paths from colliding, so re-running migrate is always safe.
            podman exec postgres psql -U "$DB_USER" -d "$DB_NAME" -q \
                -c "INSERT INTO schema_migrations (filename) VALUES ('$(basename "$sql_file")') ON CONFLICT (filename) DO NOTHING" >/dev/null
        fi
    done
    log_info "Migrations applied."
}

seed() {
    log_info "Running dummy data injection via Go..."
    # The bulk seeder defaults to -truncate=true and truncates 41 tables, so it
    # must never run against anything but the local dev setup — same hard-refuse
    # guards as scripts/reset-dev-db.sh (D2). The production volumes are never
    # named retail_pos.
    if [ "${ENV:-development}" = "production" ]; then
        log_error "refusing to seed: ENV=production"
        return 1
    fi
    if [ "$DB_NAME" != "retail_pos" ]; then
        log_error "refusing to seed: DB_NAME='$DB_NAME' (expected 'retail_pos')"
        return 1
    fi
    # Run from host (ensure DB is accessible)
    DB_HOST=localhost \
    DB_PORT=5432 \
    DB_USER="$DB_USER" \
    DB_PASSWORD="$DB_PASSWORD" \
    DB_NAME="$DB_NAME" \
    go run cmd/dummy/main.go
}

stop() {
    local target="${1:-all}"
    case "$target" in
        postgres|backend|frontend)
            log_info "Stopping $target..."
            podman stop "$target" 2>/dev/null || log_warn "$target not running"
            ;;
        all|*)
            log_info "Stopping all services in pod '$POD_NAME'..."
            if pod_exists; then
                podman pod stop "$POD_NAME"
                podman pod rm "$POD_NAME"
                log_info "Pod and containers removed"
            else
                log_warn "Pod '$POD_NAME' does not exist"
            fi
            ;;
    esac
}

status() {
    echo ""
    log_info "Pod status:"
    podman pod ls | grep "$POD_NAME" || echo "  Pod not found"
    echo ""
    log_info "Container status:"
    podman ps -a --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}" | grep -E "postgres|backend|frontend" || echo "  No containers"
    echo ""
    log_info "Connectivity:"
    if curl -s "http://localhost:${HOST_FRONTEND_PORT}/health" >/dev/null 2>&1; then
        echo -e "  ${GREEN}✓ Backend API responding${NC}"
    else
        echo -e "  ${RED}✗ Backend API not responding${NC}"
    fi
    if curl -s "http://localhost:${HOST_FRONTEND_PORT}/" >/dev/null 2>&1; then
        echo -e "  ${GREEN}✓ Frontend accessible on ${HOST_FRONTEND_PORT}${NC}"
    else
        echo -e "  ${RED}✗ Frontend not accessible${NC}"
    fi
}

logs() {
    if ! pod_exists; then log_error "Pod not running"; return 1; fi
    case "${1:-all}" in
        backend|frontend|postgres) podman logs -f "$1" ;;
        *)
            for s in backend frontend postgres; do
                echo "=== $s ==="
                podman logs "$s" 2>&1 | tail -20
                echo ""
            done
            ;;
    esac
}

# Main
case "$1" in
    start)
        case "$2" in
            postgres) start_postgres ;;
            backend)  start_backend ;;
            frontend) start_frontend ;;
            all|"")
                start_postgres
                start_backend
                start_frontend
                ;;
            *) log_error "Unknown service: $2"; exit 1 ;;
        esac
        ;;
    stop)
        stop "$2"
        ;;
    build)
        # Reachable on its own because the Quadlet units start containers without
        # ever calling this script, so nothing else would build their images.
        # The optional service argument is what the Makefile's build-backend and
        # build-frontend targets use, so every image build passes through here
        # and picks up the frontend staleness check.
        case "$2" in
            backend|frontend) build_image "$2" ;;
            "")              build_image backend; build_image frontend ;;
            *) log_error "Unknown service: $2"; exit 1 ;;
        esac
        ;;
    migrate) migrate ;;
    seed)    seed ;;
    status)  status ;;
    logs)    logs "$2" ;;
    restart)
        stop "$2"
        $0 start "$2"
        ;;
    *)
        echo "Usage: $0 {build|start|stop|migrate|seed|status|logs|restart} [service]"
        exit 1
        ;;
esac
