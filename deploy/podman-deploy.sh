#!/bin/bash
set -e

# =============================================================================
# Retail POS System - Podman Deployment Script (Refactored)
# =============================================================================
# Usage:
#   ./deploy/podman-deploy.sh build
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
HOST_FRONTEND_PORT="${HOST_FRONTEND_PORT:-5173}"

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

# Browser origin allowed by the API. Must be a SINGLE origin.
#
# The CORS middleware gets it as []string{cfg.CORSOrigin} with no comma
# splitting, and the WebSocket upgrade check does an exact string comparison
# (pkg/websocket/hub.go). A comma-separated list therefore matches neither.
CORS_ORIGIN="${CORS_ORIGIN:-http://localhost:${HOST_FRONTEND_PORT}}"

# libpq sslmode for the backend's connection to PostgreSQL.
#
# The backend defaults to `require` in production, which the stock
# postgres:18-alpine image cannot satisfy: it ships with ssl=off unless a
# certificate and key are mounted, so `require` fails with
# "server refused TLS connection" and the backend crash-loops.
#
#   require       - correct when TLS certs are mounted (see PRODUCTION-DEPLOYMENT.md).
#                   This is the default and the recommended posture.
#   disable       - only valid when the database is unreachable from outside the
#                   pod's internal network. It is a decision, not a fallback: the
#                   backend logs a warning at startup when it sees this in production.
DB_SSLMODE="${DB_SSLMODE:-require}"

# Volume names
POSTGRES_VOLUME="retail-pos-postgres-data"

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

ensure_pod() {
    if ! pod_exists; then
        log_info "Creating pod '$POD_NAME'..."
        podman pod create \
            --name "$POD_NAME" \
            --network bridge \
            -p "${HOST_FRONTEND_PORT}:8081" \
            -p "5432:5432" \
            -p "8080:8080"
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

    # pg_isready connects over the unix socket and never negotiates TLS, so it
    # reports ready even when the server holds no certificate and the backend's
    # sslmode connection is about to be refused. Confirm the server can actually
    # satisfy the mode the backend was given.
    case "$DB_SSLMODE" in
        require|verify-ca|verify-full)
            local ssl_on
            ssl_on=$(podman exec postgres psql -U "$DB_USER" -tAc "SHOW ssl;" 2>/dev/null | tr -d '[:space:]')
            if [ "$ssl_on" != "on" ]; then
                log_error "PostgreSQL has ssl=off, but the backend is configured with DB_SSLMODE=$DB_SSLMODE."
                log_error "The backend would crash-loop with 'server refused TLS connection'."
                log_error "Either mount a server certificate and key on the postgres container (see"
                log_error "deploy/PRODUCTION-DEPLOYMENT.md), or set DB_SSLMODE=disable in the secret"
                log_error "file and record that the database is confined to the pod network."
                return 1
            fi
            log_info "PostgreSQL TLS is enabled (ssl=on), satisfying DB_SSLMODE=$DB_SSLMODE"
            ;;
    esac

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

# Service Management
build_image() {
    local service=$1
    log_info "Building latest $service image..."
    case "$service" in
        backend)
            podman build -t "$BACKEND_IMAGE" -f deploy/backend/Dockerfile .
            ;;
        frontend)
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
    # COOKIE_SECURE=true is required, not hardening: the 7-day HttpOnly refresh_token
    # cookie is set with secure=(COOKIE_SECURE == "true") in internal/user/auth_handler.go,
    # so leaving it unset ships that token without the Secure flag. Browsers still accept
    # Secure cookies over http://localhost, so this is safe on a plain-HTTP test deploy.
    # COOKIE_DOMAIN is deliberately left unset: host-only is the correct default, and
    # setting it too broadly would share the refresh token across subdomains.
    podman run -d \
        --pod "$POD_NAME" \
        --name backend \
        --env-file "$ENV_FILE" \
        -e DB_HOST=localhost \
        -e DB_PORT="$DB_PORT" \
        -e DB_USER="$DB_USER" \
        -e DB_NAME="$DB_NAME" \
        -e DB_SSLMODE="$DB_SSLMODE" \
        -e PORT=8080 \
        -e ENV=production \
        -e LOG_LEVEL=info \
        -e CORS_ORIGIN="$CORS_ORIGIN" \
        -e COOKIE_SECURE=true \
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

    # Bootstrap prerequisites that 000_squash.sql depends on (fresh-DB spin-up).
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
            # Record the applied file for the audit trail. Migrations that
            # self-register keep their own row; 000_squash clears 00*.sql rows
            # on each run, so every migration is re-applied idempotently. The
            # ON CONFLICT guard makes recording safe for both cases.
            podman exec postgres psql -U "$DB_USER" -d "$DB_NAME" -q \
                -c "INSERT INTO schema_migrations (filename) VALUES ('$(basename "$sql_file")') ON CONFLICT (filename) DO NOTHING" >/dev/null
        fi
    done
    log_info "Migrations applied."
}

seed() {
    log_info "Running dummy data injection via Go..."
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
        build_image backend
        build_image frontend
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
