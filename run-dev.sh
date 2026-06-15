#!/bin/bash
# Run backend server in development mode connected to postgres-dev
# Usage: ./run-dev.sh [flags passed to server]
set -e

# Load environment variables from .env file
if [ -f .env ]; then
  set -a
  source .env
  set +a
fi

# Default values (can be overridden by .env)
export ENV="${ENV:-development}"
export PORT="${BACKEND_PORT:-9095}"
export DATABASE_URL="${DATABASE_URL:-postgres://pos:admin123@localhost:${DATABASE_PORT:-5433}/retail_pos?sslmode=disable&timezone=Asia/Jakarta}"

echo "Starting server in $ENV mode on port $PORT"
echo "Connecting to database: $DATABASE_URL"

# Check if postgres-dev container is running, start if not
if ! podman inspect --format '{{.State.Running}}' postgres-dev 2>/dev/null | grep -q "true"; then
  echo "postgres-dev container is not running, starting it..."
  podman start postgres-dev 2>/dev/null || podman run -d --name postgres-dev -e POSTGRES_USER=pos -e POSTGRES_PASSWORD=admin123 -e POSTGRES_DB=retail_pos -p 5433:5432 docker.io/library/postgres:18
  echo "Waiting for postgres-dev to be ready..."
  sleep 3
fi

# Check if port is in use and kill the process
PID=$(lsof -ti :$PORT 2>/dev/null) || true
if [ -n "$PID" ]; then
  echo "Port $PORT is in use by process $PID, killing it..."
  kill -9 $PID
fi

go run ./cmd/server/main.go "$@"