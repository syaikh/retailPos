#!/bin/bash
# Seed dummy data to postgres-dev
# Usage: ./seed-dev.sh [flags]
#   Flags passed to seeder: -products=100 -days=180 -truncate=false -stock-opnames=3
#   Note: -days=0 triggers interactive selection

set -e

# Load environment variables from .env file
if [ -f .env ]; then
  set -a
  source .env
  set +a
fi

# Allow overriding the database URL via env
DATABASE_URL="${DATABASE_URL:-postgres://pos:admin123@localhost:${DATABASE_PORT:-5433}/retail_pos?sslmode=disable&timezone=Asia/Jakarta}"

export DATABASE_URL

echo "Seeding dummy data to postgres-dev (port ${DATABASE_PORT:-5433})..."
go run ./cmd/dummy "$@"
# Ensure system users have correct bcrypt hashes
if command -v psql >/dev/null 2>&1; then
  PGPASSWORD="${DB_PASSWORD:-admin123}" psql -h "${DB_HOST:-localhost}" -p "${DB_PORT:-5433}" -U "${DB_USER:-pos}" -d "${DB_NAME:-retail_pos}" -c "
    UPDATE users SET password_hash = '\$2a\$14\$siHE.dJhi5basdsIKS8nXOjd/ETPAO1q7.ZNshHQnlhl.uxUmx.Rq'
    WHERE username IN ('superadmin','admin','manager','supervisor','cashier','finance','warehouse_manager')
  " >/dev/null 2>&1 || true
fi
