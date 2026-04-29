#!/bin/bash
set -e

cd /home/my-excellency/Projects/retail-pos-system

# Clean up any existing processes
pkill -f 'go run.*server' || true
pkill -f 'python3.*http.server' || true
sleep 2

# Start backend
echo "Starting backend..."
export DB_NAME=devdb
go run ./cmd/server/main.go > /tmp/backend.log 2>&1 &
BACKEND_PID=$!
sleep 3

# Start frontend
echo "Starting frontend..."
cd web
python3 -m http.server 5173 --bind 0.0.0.0 > /tmp/frontend.log 2>&1 &
FRONTEND_PID=$!
cd ..
sleep 2

# Verify both are running
echo "Verifying servers..."
if ! curl -s http://localhost:8080/api/stats > /dev/null 2>&1; then
  echo "Backend not responding!"
  kill $BACKEND_PID 2>/dev/null || true
  kill $FRONTEND_PID 2>/dev/null || true
  exit 1
fi

if ! curl -s http://localhost:5173/ > /dev/null 2>&1; then
  echo "Frontend not responding!"
  kill $BACKEND_PID 2>/dev/null || true
  kill $FRONTEND_PID 2>/dev/null || true
  exit 1
fi

echo "Both servers are running. Running E2E tests..."
# Run tests
cd /home/my-excellency/Projects/retail-pos-system
npx playwright test tests/e2e/login.spec.ts --reporter=list

TEST_RESULT=$?

# Cleanup
echo "Cleaning up..."
kill $BACKEND_PID 2>/dev/null || true
kill $FRONTEND_PID 2>/dev/null || true

exit $TEST_RESULT
