#!/usr/bin/env bash
# Diagnostic two-unit build: browser HTML derived from a running order_entry
# projection and a separate Linux/API Go binary. Not a Build Artifact Lock.
set -euo pipefail

if [[ $# -ne 3 || "$1" != /* ]]; then
  echo "usage: bash $0 /absolute/new/output-dir BASELINE_ID BASELINE_REVISION" >&2
  exit 2
fi
destination="$1"
baseline="$2"
revision="$3"
if [[ ! "$baseline" =~ ^[A-Za-z0-9._-]+$ || ! "$revision" =~ ^[0-9a-f]{40}$ ]]; then
  echo "explicit baseline ID and 40-character source revision are required" >&2
  exit 2
fi
for required in git go curl python3 sha256sum; do
  command -v "$required" >/dev/null || { echo "missing $required" >&2; exit 2; }
done
repo="$(git rev-parse --show-toplevel)"
repo="$(cd "$repo" && pwd -P)"
[[ "$PWD" == "$repo" ]] || { echo "run from repository root" >&2; exit 2; }
parent="$(cd "$(dirname "$destination")" && pwd -P)"
name="$(basename "$destination")"
if [[ "$name" == "." || "$name" == ".." || "$parent/$name" == "$repo" ||
      "$parent/$name" == "$repo/"* || -e "$parent/$name" || -L "$parent/$name" ]]; then
  echo "new output directory must be outside the checkout" >&2
  exit 2
fi
[[ -z "$(git status --porcelain=v1 --untracked-files=all)" ]] ||
  { echo "refuse dirty checkout" >&2; exit 1; }
source_rev="$(git rev-parse HEAD)"
export GOENV=off GOWORK=off GOFLAGS= GOTOOLCHAIN=local GOOS=linux GOARCH=amd64 GOAMD64=v1 CGO_ENABLED=0
temp="$(mktemp -d "$parent/.tei-e1-two-unit-XXXXXXXX")"
api_pid=""
published=false
cleanup() {
  if [[ -n "$api_pid" ]]; then
    kill "$api_pid" 2>/dev/null || :
    wait "$api_pid" 2>/dev/null || :
  fi
  rm -rf -- "$temp"
  if [[ "$published" == true ]]; then rm -rf -- "$parent/$name"; fi
}
trap cleanup EXIT
mkdir -p "$temp/browser" "$temp/api"
go build -trimpath -ldflags=-buildid= -tags=tei_order_only \
  -o "$temp/api/order-entry" ./tools/reference-first-web-target
# The port is allocated by the OS, and the server is loopback-only. This
# standalone diagnostic has a small port allocation race, not a deployable Host.
port="$(python3 -c 'import socket; s=socket.socket(); s.bind(("127.0.0.1",0)); print(s.getsockname()[1]); s.close()')"
"$temp/api/order-entry" \
  -workspace "$repo/reference/workspace" \
  -legacy-users "$repo/reference/legacy-auth/users.json" \
  -legacy-authorization "$repo/reference/legacy-auth/authorization.json" \
  -application order_entry \
  -baseline-id "$baseline" -baseline-revision "$revision" \
  -target-revision "$source_rev" -listen "127.0.0.1:$port" \
  > "$temp/api.log" 2>&1 &
api_pid="$!"
ready=false
for attempt in {1..100}; do
  if ! kill -0 "$api_pid" 2>/dev/null; then
    echo "API binary exited before HTTP startup" >&2; exit 1
  fi
  if curl --noproxy '*' -fsS --max-time 2 \
      "http://127.0.0.1:$port/manifest.json" -o "$temp/projection.json"; then
    ready=true; break
  fi
  sleep 0.1
done
[[ "$ready" == true ]] || { echo "API did not start" >&2; exit 1; }
curl --noproxy '*' -fsS --max-time 10 \
  "http://127.0.0.1:$port/custom-submit" -o "$temp/browser/index.html"
# An HTML build is derived from the real running Web Target, not a copy of
# Canonical declarations. Do not publish the full diagnostic JSON manifest.
python3 - "$temp/projection.json" "$temp/browser/index.html" "$baseline" "$revision" "$source_rev" <<'PY'
import json
import pathlib
import sys
manifest = json.loads(pathlib.Path(sys.argv[1]).read_bytes())
html = pathlib.Path(sys.argv[2]).read_text("utf-8")
for key, expected in (
    ("applicationId", "order_entry"),
    ("handoverBaselineId", sys.argv[3]),
    ("baselineRepositoryRevision", sys.argv[4]),
    ("targetBuildRevision", sys.argv[5]),
):
    if manifest.get(key) != expected:
        raise SystemExit(f"projection binding mismatch: {key}")
    if expected not in html:
        raise SystemExit(f"browser HTML omits projected binding {key}")
if '<form method="post" action="/custom-submit"' not in html:
    raise SystemExit("browser form not bound to the existing API contract")
if 'name="credential" type="password"' not in html:
    raise SystemExit("browser did not preserve the transient credential field")
for forbidden in ('editor-secret', 'reference/legacy-auth/', 'workspace.Load',
                  'driver.NewResolver', '<script', 'sourceMappingURL'):
    if forbidden in html:
        raise SystemExit(f"browser asset contains forbidden source/secret marker: {forbidden}")
PY
if [[ "$(git rev-parse HEAD)" != "$source_rev" ||
      -n "$(git status --porcelain=v1 --untracked-files=all)" ]]; then
  echo "checkout changed during build; discard artifacts" >&2
  exit 1
fi
browser_size="$(wc -c < "$temp/browser/index.html" | tr -d '[:space:]')"
api_size="$(wc -c < "$temp/api/order-entry" | tr -d '[:space:]')"
browser_sha="$(sha256sum "$temp/browser/index.html" | cut -d' ' -f1)"
api_sha="$(sha256sum "$temp/api/order-entry" | cut -d' ' -f1)"
cat > "$temp/measurement.json" <<EOF
{
  "kind": "diagnostic-first-web-two-unit-observation",
  "applicationId": "order_entry",
  "handoverBaselineId": "$baseline",
  "baselineRepositoryRevision": "$revision",
  "browserAndApiSourceRevision": "$source_rev",
  "goVersion": "$(go env GOVERSION)",
  "goos": "linux",
  "goarch": "amd64",
  "cgoEnabled": false,
  "browserBytes": $browser_size,
  "browserSHA256": "$browser_sha",
  "apiBytes": $api_size,
  "apiSHA256": "$api_sha",
  "browserAndApiPhysicallySeparate": true,
  "sameHandoverBusinessParityVerified": false,
  "productionAuthorizationVerified": false,
  "browserRuntimeVisualVerified": false,
  "fullE1BrowserAPIPackagingVerified": false,
  "artifactAuthority": false
}
EOF
kill "$api_pid" 2>/dev/null || :
wait "$api_pid" 2>/dev/null || :
api_pid=""
rm -f "$temp/projection.json" "$temp/api.log"
mkdir -- "$parent/$name"
published=true
mv "$temp/browser" "$temp/api" "$temp/measurement.json" "$parent/$name/"
published=false
cat "$parent/$name/measurement.json"
