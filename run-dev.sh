#!/bin/bash
# Run backend server in development mode connected to postgres-dev
# Usage: ./run-dev.sh [flags passed to server]
set -e

# Default connection to postgres-dev (port 5433)
export ENV="${ENV:-development}"
export PORT="${PORT:-9095}"
export DATABASE_URL="${DATABASE_URL:-postgres://pos:admin123@localhost:5433/retail_pos?sslmode=disable&timezone=Asia/Jakarta}"

echo "Starting server in $ENV mode on port $PORT"
echo "Connecting to database: $DATABASE_URL"

go run ./cmd/server/main.go "$@"