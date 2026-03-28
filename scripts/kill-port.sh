#!/usr/bin/env bash
# Kill any process listening on the given port.
# Usage: ./scripts/kill-port.sh <port>
set -euo pipefail

PORT="${1:?Usage: kill-port.sh <port>}"

PID=$(lsof -ti :"$PORT" 2>/dev/null || true)
if [ -n "$PID" ]; then
  echo "Killing process(es) on port $PORT: $PID"
  echo "$PID" | xargs kill -9 2>/dev/null || true
  sleep 0.5
else
  echo "Port $PORT is free."
fi
