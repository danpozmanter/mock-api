#!/usr/bin/env bash
# Boots a built mock-api binary and checks one answer of each kind.
# Usage: scripts/smoke.sh [BINARY] (run from the repository root)
set -euo pipefail

bin=${1:-target/release/mock-api}
port=${SMOKE_PORT:-18080}
base="http://127.0.0.1:$port"

# A FIFO rather than coproc keeps this runnable on the bash 3.2 macOS ships.
fifo=$(mktemp -u)
mkfifo "$fifo"
"$bin" --config scripts/smoke.yaml --port "$port" >"$fifo" 2>&1 &
server_pid=$!
trap 'kill "$server_pid" 2>/dev/null || true' EXIT
exec 3<"$fifo"
rm "$fifo"

# The server binds before it announces itself, so the announcement means
# connections are already being accepted.
ready=false
while IFS= read -r line <&3; do
    echo "$line"
    if [[ $line == "Starting server on"* ]]; then
        ready=true
        break
    fi
done
$ready || { echo "smoke: server exited before it started listening" >&2; exit 1; }
# Keep draining the server's log so a full pipe never stalls a request.
cat <&3 >&2 &

fail() { echo "smoke: $*" >&2; exit 1; }

expect_status() {
    local want=$1 method=$2 path=$3 got
    got=$(curl -s -m 10 -o /dev/null -w '%{http_code}' -X "$method" "$base$path")
    [[ $got == "$want" ]] || fail "$method $path answered $got, expected $want"
    echo "ok: $method $path -> $got"
}

expect_status 200 GET /v1/models
expect_status 200 GET /v1/models/
expect_status 200 GET /v1/models/gpt-4
expect_status 200 POST /v1/chat/completions
expect_status 405 DELETE /v1/models
expect_status 404 GET /v1/unknown

body=$(curl -s -m 10 "$base/v1/models")
[[ $body == *'"id": "gpt-4"'* ]] || fail "override body missing from: $body"
echo "ok: override body"

stream=$(curl -s -N -m 10 "$base/v1/models?stream=true")
[[ $(grep -c '^data: ' <<<"$stream") -ge 2 ]] || fail "stream has too few frames: $stream"
[[ $(grep '^data: ' <<<"$stream" | tail -n 1) == "data: [DONE]" ]] || fail "stream not terminated: $stream"
echo "ok: stream ends with [DONE]"
