#!/usr/bin/env bash
set -euo pipefail

if [[ "$#" -ne 0 ]]; then
  echo 'usage: bash tools/check-reference-plugin-wasi-smoke.sh' >&2
  exit 2
fi
root="$(git rev-parse --show-toplevel 2>/dev/null || true)"
if [[ -z "$root" || "$PWD" != "$root" ]]; then
  echo 'error: run from TEI repository root' >&2
  exit 2
fi
for executable in go node timeout cmp; do
  if ! command -v "$executable" >/dev/null 2>&1; then
    echo "error: $executable is required for WASI invocation parity" >&2
    exit 2
  fi
done
if [[ "$(go env GOOS)/$(go env GOARCH)" != "linux/amd64" ]]; then
  echo 'skip: actual native-vs-WASI invocation parity requires a linux/amd64 host' >&2
  exit 0
fi

probe="$(mktemp -d)"
trap 'rm -rf "$probe"' EXIT
mkdir -p "$probe/module" "$probe/home" "$probe/cache" "$probe/modcache"
cp -R "$root/plugins/reference-process-executor/." "$probe/module/"
if [[ -n "$(find "$probe/module" \( -name go.mod -o -name go.work -o -type l \) -print -quit)" ]]; then
  echo 'error: unexpected nested module/workspace or symlink in reference plugin' >&2
  exit 1
fi
printf 'module tei-independent-wasi-probe\n\ngo 1.23.2\n' > "$probe/module/go.mod"

(
  cd "$probe/module"
  export GOENV=off GOWORK=off GOFLAGS= GO111MODULE=on GOTOOLCHAIN=local
  export GOPROXY=off GOSUMDB=off
  export GOPATH="$probe/home" GOMODCACHE="$probe/modcache" GOCACHE="$probe/cache"
  GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -o "$probe/native" .
  GOOS=wasip1 GOARCH=wasm CGO_ENABLED=0 go build -trimpath -o "$probe/plugin.wasm" .
)

marker='--tei-reference-marker=argv value preserved'
cat > "$probe/valid.json" <<'JSON'
{"protocolVersion":1,"requestId":"wasi-parity-one","plugin":{"id":"tei.process.reference","version":"1.0.0"},"capabilityId":"process.reference","descriptorFingerprint":"wasi-parity-descriptor","operation":"execute","payload":{"id":"intent-one","actionId":"sales_expand/apply","idempotencyKey":"wasi-parity-key"}}
JSON
sed 's/"protocolVersion":1/"protocolVersion":2/' "$probe/valid.json" > "$probe/wrong-version.json"
sed 's/"operation":"execute"/"operation":"observe"/' "$probe/valid.json" > "$probe/wrong-operation.json"
sed 's/"idempotencyKey":"wasi-parity-key"/"idempotencyKey":""/' "$probe/valid.json" > "$probe/missing-key.json"

runner="$root/tools/reference-plugin-wasi-smoke/wasi-runner.mjs"
run_native() {
  timeout 30s "$probe/native" "$@" < "$probe/request.json" > "$probe/native.out" 2> "$probe/native.err"
}
run_wasi() {
  timeout 30s node "$runner" "$probe/plugin.wasm" "$@" < "$probe/request.json" > "$probe/wasi.out" 2> "$probe/wasi.err"
}
verify_case() {
  local name="$1" expected_status="$2" fixture="$3"
  shift 3
  cp "$fixture" "$probe/request.json"
  run_native "$@"
  run_wasi "$@"
  if ! cmp -s "$probe/native.out" "$probe/wasi.out"; then
    echo "error: different native/WASI response bytes for $name" >&2
    exit 1
  fi
  if ! grep -Fq "\"status\":\"$expected_status\"" "$probe/wasi.out"; then
    echo "error: wrong WASI response status for $name" >&2
    exit 1
  fi
  if [[ "$expected_status" == "ok" ]]; then
    grep -Fq '"outcome":"accepted"' "$probe/wasi.out"
    grep -Fq '"idempotencyKey":"wasi-parity-key"' "$probe/wasi.out"
    grep -Fq '"requestId":"wasi-parity-one"' "$probe/wasi.out"
  else
    if grep -Fq '"outcome":"accepted"' "$probe/wasi.out"; then
      echo "error: rejected request was acknowledged for $name" >&2
      exit 1
    fi
  fi
  echo "PASS native/WASI request parity: $name"
}
verify_case accepted ok "$probe/valid.json" "$marker"
verify_case protocol-rejected error "$probe/wrong-version.json" "$marker"
verify_case operation-rejected error "$probe/wrong-operation.json" "$marker"
verify_case payload-rejected error "$probe/missing-key.json" "$marker"
verify_case argv-rejected error "$probe/valid.json" '--wrong-argv'

# Both OS process boundaries reject malformed requests without echoing the
# original payload or returning a partial acknowledgment.
printf '%s\n' 'private-material-not-for-echo{' > "$probe/request.json"
native_exit=0
run_native "$marker" || native_exit=$?
wasi_exit=0
run_wasi "$marker" || wasi_exit=$?
if [[ "$native_exit" -eq 0 || "$wasi_exit" -eq 0 ||
      "$native_exit" -ne "$wasi_exit" ]]; then
  echo "error: malformed request exit mismatch native=$native_exit wasi=$wasi_exit" >&2
  exit 1
fi
if [[ -s "$probe/native.out" || -s "$probe/wasi.out" ]]; then
  echo 'error: malformed request produced response bytes' >&2
  exit 1
fi
if grep -Fq 'private-material-not-for-echo' "$probe/native.err" "$probe/wasi.err"; then
  echo 'error: stderr exposed malformed untrusted request' >&2
  exit 1
fi
echo 'PASS native/WASI malformed request rejection'
