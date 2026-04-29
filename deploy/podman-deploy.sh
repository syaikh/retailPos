#!/bin/bash
set -e

# =============================================================================
# Retail POS System - Podman Deployment Script
# =============================================================================
# This script deploys the entire Retail POS System using Podman containers:
#  - PostgreSQL database
#  - Go backend (API + WebSocket)
#  - Nginx frontend (static files + reverse proxy)
#
# Usage:
#   ./deploy/podman-deploy.sh [start|stop|restart|logs|status]
#
# Prerequisites:
#   - Podman installed and running
#   - Images built: retail-pos-backend:latest, retail-pos-frontend:latest
#   - Optionally: podman-docker package for docker-compatible CLI
# =============================================================================

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$SCRIPT_DIR"

# Configuration
POD_NAME="retail-pos-pod"
NETWORK_NAME="retail-pos-network"

# Host port mapping: external port (e.g., 5173) -> frontend container port 80
# Backend not exposed externally, only accessible within pod via localhost:8080
HOST_FRONTEND_PORT="${HOST_FRONTEND_PORT:-5173}"

# Image names (fully qualified with localhost for local images)
BACKEND_IMAGE="${BACKEND_IMAGE:-localhost/retail-pos-backend:latest}"
FRONTEND_IMAGE="${FRONTEND_IMAGE:-localhost/retail-pos-frontend:latest}"
POSTGRES_IMAGE="${POSTGRES_IMAGE:-docker.io/library/postgres:15-alpine}"

# Database configuration
DB_NAME="${DB_NAME:-retail_pos}"
DB_USER="${DB_USER:-pos}"
# Generate random password if not set
if [ -z "$DB_PASSWORD" ]; then
    if command -v openssl &>/dev/null; then
        DB_PASSWORD=$(openssl rand -base64 12 2>/dev/null || echo "pospass123")
    else
        DB_PASSWORD="pospass123"
    fi
fi
DB_PORT="${DB_PORT:-5432}"
POSTGRES_PASSWORD="${POSTGRES_PASSWORD:-${DB_PASSWORD}}"

# Volume names
POSTGRES_VOLUME="retail-pos-postgres-data"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if pod exists
pod_exists() {
    podman pod exists "$POD_NAME" 2>/dev/null || return 1
}

# Check if container exists
container_exists() {
    podman container exists "$1" 2>/dev/null || return 1
}

# Wait for PostgreSQL to be ready
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

# Wait for backend to be ready
wait_for_backend() {
    log_info "Waiting for backend API to be ready..."
    local max_attempts=30
    local attempt=0
    while [ $attempt -lt $max_attempts ]; do
        # Use curl inside backend container; any response (even 401) means backend is alive
        if podman exec backend curl -s -o /dev/null http://localhost:8080/api/stats; then
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

init_db() {
    log_info "Initializing database..."

    # Wait for postgres to be ready inside container
    podman exec postgres pg_isready -U postgres >/dev/null 2>&1 || sleep 3

    # Create database if not exists
    if podman exec postgres psql -U "$DB_USER" -lqt | cut -d\| -f1 | grep -qw "$DB_NAME"; then
        log_info "Database '$DB_NAME' already exists"
    else
        log_info "Creating database '$DB_NAME'..."
        podman exec postgres createdb -U "$DB_USER" "$DB_NAME"
    fi

    # Run migrations (as application superuser)
    log_info "Running database migrations..."
    local migration_dir="$SCRIPT_DIR/database/migrations"
    if [ -d "$migration_dir" ]; then
        for sql_file in "$migration_dir"/*.sql; do
            if [ -f "$sql_file" ]; then
                log_info "  Migrating: $(basename "$sql_file")"
                podman exec -i postgres psql -U "$DB_USER" -d "$DB_NAME" < "$sql_file"
            fi
        done
    else
        log_warn "Migration directory not found: $migration_dir"
    fi

    # Run seed files
    log_info "Running database seeds..."
    local seed_dir="$SCRIPT_DIR/database/seeds"
    if [ -d "$seed_dir" ]; then
        for sql_file in "$seed_dir"/*.sql; do
            if [ -f "$sql_file" ]; then
                log_info "  Seeding: $(basename "$sql_file")"
                podman exec -i postgres psql -U "$DB_USER" -d "$DB_NAME" < "$sql_file"
            fi
        done
        log_info "Seeds applied successfully."
    else
        log_warn "Seed directory not found: $seed_dir"
    fi

    log_info "Database initialized"
}

start() {
    log_info "Starting Retail POS System..."

    if pod_exists; then
        log_warn "Pod '$POD_NAME' already exists. Use 'restart' to recreate."
        return 0
    fi

    # 1. Create pod with shared network
    # Only frontend port is exposed externally. Backend is only accessible within pod via localhost:8080
    log_info "Creating pod '$POD_NAME'..."
    podman pod create \
        --name "$POD_NAME" \
        --network bridge \
        -p "${HOST_FRONTEND_PORT}:8081"
        # SSL port 8443 can be added: -p 8443:8443

    # 2. Create volume for Postgres data (if not exists)
    log_info "Creating persistent volume for PostgreSQL..."
    podman volume create "$POSTGRES_VOLUME" 2>/dev/null || true

    # 3. Start PostgreSQL container in the pod
    log_info "Starting PostgreSQL container..."
    podman run -d \
        --pod "$POD_NAME" \
        --name postgres \
        -e POSTGRES_USER="$DB_USER" \
        -e POSTGRES_PASSWORD="$POSTGRES_PASSWORD" \
        -e POSTGRES_DB="$DB_NAME" \
        -e PGDATA=/var/lib/postgresql/data/pgdata \
        -v "$POSTGRES_VOLUME:/var/lib/postgresql/data" \
        --restart unless-stopped \
        "$POSTGRES_IMAGE"

    # Wait for Postgres
    sleep 5
    wait_for_postgres || exit 1

    # 4. Initialize database (run migrations + seeds)
    init_db

    # 5. Start backend container
    log_info "Starting Go backend container..."
    podman run -d \
        --pod "$POD_NAME" \
        --name backend \
        -e DB_HOST=localhost \
        -e DB_PORT=5432 \
        -e DB_USER="$DB_USER" \
        -e DB_PASSWORD="$DB_PASSWORD" \
        -e DB_NAME="$DB_NAME" \
        -e GIN_MODE=release \
        --restart unless-stopped \
        "$BACKEND_IMAGE"

    # 6. Start frontend container
    log_info "Starting Nginx frontend container..."
    podman run -d \
        --pod "$POD_NAME" \
        --name frontend \
        --restart unless-stopped \
        "$FRONTEND_IMAGE"

    # 7. Wait for services
    wait_for_backend || exit 1

    # 8. Show credentials
    log_info "============================================"
    log_info "Retail POS System is running!"
    log_info "  Frontend:  http://localhost:${HOST_FRONTEND_PORT}"
    log_info "  API (internal only):  http://localhost:8080"
    log_info "  Database:  ${DB_NAME}@localhost:5432"
    log_info "  DB User:   ${DB_USER}"
    log_info "  DB Pass:   ${POSTGRES_PASSWORD}"
    log_info "============================================"
}

stop() {
    log_info "Stopping Retail POS System..."

    if ! pod_exists; then
        log_warn "Pod '$POD_NAME' does not exist"
        return 0
    fi

    # Stop and remove pod (containers are removed automatically)
    podman pod stop "$POD_NAME"
    podman pod rm "$POD_NAME"

    # Remove volume (data will be lost - comment out if you want to keep data)
    # podman volume rm "$POSTGRES_VOLUME" 2>/dev/null || true

    log_info "All services stopped"
}

restart() {
    stop
    sleep 2
    start
}

status() {
    echo ""
    log_info "Pod status:"
    podman pod ls | grep "$POD_NAME" || echo "  Pod not found"

    echo ""
    log_info "Container status:"
    podman ps -a --format "table {{.Names}}\t{{.Status}}\t{{.Ports}}" | grep -E "postgres|backend|frontend" || echo "  No containers"

    echo ""
    log_info "Network connectivity:"
    # Backend is accessed via nginx proxy on HOST_FRONTEND_PORT
    if curl -s "http://localhost:${HOST_FRONTEND_PORT}/api/stats" >/dev/null 2>&1; then
        echo -e "  ${GREEN}✓ Backend API responding (via Nginx)${NC}"
    else
        echo -e "  ${RED}✗ Backend API not responding${NC}"
    fi

    if curl -s "http://localhost:${HOST_FRONTEND_PORT}/" >/dev/null 2>&1; then
        echo -e "  ${GREEN}✓ Frontend accessible on port ${HOST_FRONTEND_PORT}${NC}"
    else
        echo -e "  ${RED}✗ Frontend not accessible${NC}"
    fi
}

logs() {
    if ! pod_exists; then
        log_error "Pod '$POD_NAME' not running"
        return 1
    fi

    case "${1:-all}" in
        backend)
            podman logs -f backend
            ;;
        frontend)
            podman logs -f frontend
            ;;
        postgres)
            podman logs -f postgres
            ;;
        all|*)
            echo "=== Backend ==="
            podman logs backend 2>&1 | tail -20
            echo ""
            echo "=== Frontend ==="
            podman logs frontend 2>&1 | tail -20
            echo ""
            echo "=== PostgreSQL ==="
            podman logs postgres 2>&1 | tail -20
            ;;
    esac
}

# Main command dispatcher
case "${1:-status}" in
    start)
        start
        ;;
    stop)
        stop
        ;;
    restart)
        restart
        ;;
    status)
        status
        ;;
    logs)
        logs "${2:-all}"
        ;;
    init-db)
        # Helper to initialize DB manually
        init_db
        ;;
    *)
        echo "Usage: $0 {start|stop|restart|status|logs [backend|frontend|postgres|all]}"
        exit 1
        ;;
esac
