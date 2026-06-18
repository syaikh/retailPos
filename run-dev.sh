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
  podman start postgres-dev
  echo "Waiting for postgres-dev to be ready..."
  sleep 3
fi

start_server() {
  # Graceful shutdown server lama via SIGTERM
  if [ -n "$SERVER_PID" ]; then
    echo "Shutting down server (PID: $SERVER_PID)..."
    kill $SERVER_PID 2>/dev/null || true
    sleep 2
  fi

  # Force kill jika masih ada yang menempati port
  PID=$(lsof -ti :$PORT 2>/dev/null) || true
  if [ -n "$PID" ]; then
    echo "Port $PORT still in use by process $PID, force killing..."
    kill -9 $PID 2>/dev/null || true
  fi

  go run ./cmd/server/main.go "$@" &
  SERVER_PID=$!
}

cleanup() {
  if [ -n "$SERVER_PID" ]; then
    kill $SERVER_PID 2>/dev/null || true
    wait $SERVER_PID 2>/dev/null || true
  fi
  exit 0
}

trap cleanup SIGINT SIGTERM

start_server "$@"
echo "Server is running. Press 'r' + Enter to restart, 'q' + Enter to quit."

while true; do
  read key
  case "$key" in
    r|R) start_server "$@" ;;
    q|Q) cleanup ;;
  esac
done