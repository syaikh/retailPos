#!/bin/bash
# Seed dummy data to postgres-dev
# Usage: ./seed-dev.sh [flags]
#   Flags passed to seeder: -products=100 -days=30 -truncate=false

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