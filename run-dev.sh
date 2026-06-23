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

SERVER_BINARY="/tmp/retail-pos-server"

echo "Starting server in $ENV mode on port $PORT"
echo "Connecting to database: $DATABASE_URL"

# Check if postgres-dev container exists and is running
if ! podman inspect --format '{{.State.Running}}' postgres-dev 2>/dev/null | grep -q "true"; then
  if podman ps -a --filter name="^/postgres-dev$" --format '{{.Names}}' 2>/dev/null | grep -q "postgres-dev"; then
    echo "postgres-dev container is not running, starting it..."
    podman start postgres-dev
  else
    echo "Error: postgres-dev container does not exist. Create it first."
    exit 1
  fi

  echo "Waiting for postgres-dev to be ready..."
  DB_PORT="${DATABASE_PORT:-5433}"
  for i in $(seq 1 30); do
    if pg_isready -h localhost -p "$DB_PORT" -U "${DB_USER:-pos}" >/dev/null 2>&1; then
      echo "postgres-dev is ready."
      break
    fi
    if [ "$i" -eq 30 ]; then
      echo "Timed out waiting for postgres-dev to be ready."
      exit 1
    fi
    sleep 1
  done
fi

start_server() {
  # Graceful shutdown server lama via SIGTERM
  if [ -n "$SERVER_PID" ]; then
    echo "Shutting down server (PID: $SERVER_PID)..."
    kill $SERVER_PID 2>/dev/null || true
    sleep 2
  fi

  # Force kill jika masih ada yang menempati port
  PID=$(ss -tlnp "sport = :$PORT" 2>/dev/null | grep -oP 'pid=\K[0-9]+' | head -1) || true
  if [ -n "$PID" ]; then
    echo "Port $PORT still in use by process $PID, force killing..."
    kill -9 $PID 2>/dev/null || true
    sleep 1
  fi

  if ! go build -o "$SERVER_BINARY" ./cmd/server/main.go; then
    echo "Build failed. Server not started."
    return 1
  fi

  "$SERVER_BINARY" "$@" &
  SERVER_PID=$!

  # Wait until server is actually listening on the port
  for i in $(seq 1 10); do
    if ss -tlnp "sport = :$PORT" 2>/dev/null | grep -q .; then
      echo "Server started successfully (PID: $SERVER_PID)."
      return 0
    fi
    sleep 0.5
  done

  echo "Warning: Server may not have started correctly (PID $SERVER_PID). Check logs above."
}

cleanup() {
  if [ -n "$SERVER_PID" ]; then
    echo "Shutting down server..."
    kill $SERVER_PID 2>/dev/null || true
    wait $SERVER_PID 2>/dev/null || true
  fi
  rm -f "$SERVER_BINARY"
  exit 0
}

trap cleanup SIGINT SIGTERM

start_server "$@" || true
echo "Server is running. Press 'r' + Enter to restart, 'q' + Enter to quit."

while true; do
  read -r key
  case "$key" in
    r|R) start_server "$@" ;;
    q|Q) cleanup ;;
  esac
done