#!/bin/bash
set -e

# =============================================================================
# Retail POS System - Podman Deployment Script (Refactored)
# =============================================================================
# Usage:
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

# Image names
BACKEND_IMAGE="${BACKEND_IMAGE:-localhost/retail-pos-backend:latest}"
FRONTEND_IMAGE="${FRONTEND_IMAGE:-localhost/retail-pos-frontend:latest}"
POSTGRES_IMAGE="${POSTGRES_IMAGE:-docker.io/library/postgres:18-alpine}"

# Database configuration
DB_NAME="${DB_NAME:-retail_pos}"
DB_USER="${DB_USER:-pos}"
DB_PASSWORD="${DB_PASSWORD:-admin123}"
DB_PORT="${DB_PORT:-5432}"
POSTGRES_PASSWORD="${POSTGRES_PASSWORD:-${DB_PASSWORD}}"

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
            log_info "PostgreSQL is ready!"
            return 0
        fi
        attempt=$((attempt + 1))
        echo -n "."
        sleep 2
    done
    log_error "PostgreSQL did not become ready in time"
    return 1
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
            -e POSTGRES_PASSWORD="$POSTGRES_PASSWORD" \
            -e POSTGRES_DB="$DB_NAME" \
            -v "$POSTGRES_VOLUME:/var/lib/postgresql" \
            --restart unless-stopped \
            "$POSTGRES_IMAGE"
    fi
    wait_for_postgres
}

start_backend() {
    ensure_pod
    build_image backend
    if container_exists "backend"; then
        log_info "Replacing existing backend container..."
        podman stop backend 2>/dev/null || true
        podman rm backend 2>/dev/null || true
    fi
    log_info "Starting Go backend container..."
    podman run -d \
        --pod "$POD_NAME" \
        --name backend \
        -e DB_HOST=localhost \
        -e DB_PORT=5432 \
        -e DB_USER="$DB_USER" \
        -e DB_PASSWORD="$DB_PASSWORD" \
        -e DB_NAME="$DB_NAME" \
        -e PORT=8080 \
        -e FRONTEND_URL="${FRONTEND_URL:-http://localhost:5173,http://localhost:5174}" \
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
    migrate) migrate ;;
    seed)    seed ;;
    status)  status ;;
    logs)    logs "$2" ;;
    restart)
        stop "$2"
        $0 start "$2"
        ;;
    *)
        echo "Usage: $0 {start|stop|migrate|seed|status|logs|restart} [service]"
        exit 1
        ;;
esac
