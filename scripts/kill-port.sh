#!/bin/bash

if [ -z "$1" ]; then
  echo "Usage: kill-port.sh <port>"
  exit 1
fi

PORT=$1
PID=$(lsof -ti :$PORT)

if [ -z "$PID" ]; then
  echo "No process found on port $PORT"
  exit 0
fi

echo "Killing process $PID on port $PORT"
kill -9 $PID
