#!/usr/bin/env bash
# Read-only composition smoke with the ACTUAL order_entry First Web Target Go API.
# Neither process implements a new command, authentication or Authorization.
set -euo pipefail
repo="$(git rev-parse --show-toplevel)"
cd "$repo"
tmp="$(mktemp -d)"
api_pid=""
advice_pid=""
cleanup() {
  if [[ -n "$advice_pid" ]]; then kill "$advice_pid" 2>/dev/null || true; wait "$advice_pid" 2>/dev/null || true; fi
  if [[ -n "$api_pid" ]]; then kill "$api_pid" 2>/dev/null || true; wait "$api_pid" 2>/dev/null || true; fi
  rm -rf -- "$tmp"
}
trap cleanup EXIT
api="http://127.0.0.1:18977"
host="http://127.0.0.1:8787"

go build -tags=tei_order_only,tei_api_only -o "$tmp/order-entry" ./tools/reference-first-web-target
go build -o "$tmp/advice-host" ./tools/reference-project-advice-lit

"$tmp/order-entry" \
  -workspace "$repo/reference/workspace" \
  -legacy-users "$repo/reference/legacy-auth/users.json" \
  -legacy-authorization "$repo/reference/legacy-auth/authorization.json" \
  -application order_entry \
  -baseline-id B \
  -baseline-revision e5231839a5dcc62cce1181a2731fcb7c70d16736 \
  -target-revision "$(git rev-parse HEAD)" \
  -listen 127.0.0.1:18977 >"$tmp/api.log" 2>&1 &
api_pid=$!

"$tmp/advice-host" -order-entry-api "$api" >"$tmp/advice.log" 2>&1 &
advice_pid=$!

for attempt in $(seq 1 100); do
  if ! kill -0 "$api_pid" 2>/dev/null || ! kill -0 "$advice_pid" 2>/dev/null; then
    echo "reference process exited" >&2
    exit 1
  fi
  if curl -fsS --max-time 1 "$api/scenario/read" -o "$tmp/read.json" &&
     curl -fsS --max-time 1 "$host/project/order-entry-context" -o "$tmp/context.json"; then
    break
  fi
  sleep 0.1
done
test -s "$tmp/context.json"
python3 - "$tmp/read.json" "$tmp/context.json" <<'PY'
import json
import pathlib
import sys
read = json.loads(pathlib.Path(sys.argv[1]).read_text())
context = json.loads(pathlib.Path(sys.argv[2]).read_text())
assert read["result"]["applicationId"] == "order_entry"
assert context["schema"] == "tei.reference.order-entry-context/v0"
for key in ("applicationId", "scenarioId", "selectedRowIndex"):
    assert context[key] == read["result"][key], key
assert context["targetBuildRevision"] == read["targetBuildRevision"]
# The project-owned context does not leak business rows, credentials or Action results.
assert set(context) == {"schema", "applicationId", "scenarioId",
                        "selectedRowIndex", "targetBuildRevision"}
print("PASS: actual order_entry read → project-owned Go Host → limited context")
PY
# Project-owned advisory remains independent of order_entry read/commands.
curl -fsS --max-time 2 -H "Content-Type: application/json" \
  -d '{"customerCode":"VIP-1","note":""}' \
  "$host/project/advice" > "$tmp/advice.json"
python3 - "$tmp/advice.json" <<'PY'
import json
import pathlib
import sys
result = json.loads(pathlib.Path(sys.argv[1]).read_text())
assert result["advisory"] is True and result["kind"] == "review_suggested"
print("PASS: project-owned advisory still isolated")
PY
