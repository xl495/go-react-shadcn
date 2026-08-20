#!/bin/sh
set -eu
ROOT="$(cd "$(dirname "$0")/.." && pwd)"

if [ -x "${HOME}/go/bin/go1.24.0" ]; then
  GO="${HOME}/go/bin/go1.24.0"
elif command -v go1.24.0 >/dev/null 2>&1; then
  GO="$(command -v go1.24.0)"
else
  GO="$(command -v go)"
fi

for port in 8080 5173; do
  pids="$(lsof -ti tcp:$port 2>/dev/null || true)"
  if [ -n "$pids" ]; then
    echo "freeing :$port ($pids)"
    kill $pids 2>/dev/null || true
    sleep 0.3
  fi
done

cd "$ROOT/server"
"$GO" run ./cmd/server &
API_PID=$!

cd "$ROOT/admin"
npm run dev -- --host 127.0.0.1 --port 5173 &
WEB_PID=$!

cleanup() {
  kill "$API_PID" "$WEB_PID" 2>/dev/null || true
}
trap cleanup INT TERM EXIT

echo "server  http://127.0.0.1:8080"
echo "admin   http://127.0.0.1:5173"
wait
