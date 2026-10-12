#!/usr/bin/env bash
# Demonstrate that real js/wasm execution preserves the native Transform wire
# contract. Keep logical identity distinct from target-specific binary bytes.
set -euo pipefail

if [[ $# -ne 1 || "$1" != /* ]]; then
  echo "usage: $0 /absolute/output-directory" >&2
  exit 2
fi
output="$1"
repo="$(git rev-parse --show-toplevel)"
case "$output/" in
  "$repo/"*) echo "output directory must be outside repository" >&2; exit 2 ;;
esac
if [[ -n "$(git status --porcelain=v1 --untracked-files=all)" ]]; then
  echo "source repository must be clean" >&2
  exit 1
fi

mkdir -p "$output"
command -v node >/dev/null || { echo "Node.js is needed to execute js/wasm" >&2; exit 1; }
go_root="$(go env GOROOT)"
wasm_exec="$go_root/lib/wasm/go_js_wasm_exec"
if [[ ! -f "$wasm_exec" ]]; then
  wasm_exec="$go_root/misc/wasm/go_js_wasm_exec"
fi
test -f "$wasm_exec" || { echo "Go js/wasm Node runner unavailable" >&2; exit 1; }

go version > "$output/go-version.txt"
node --version > "$output/node-version.txt"

go test -count=1 -run '^TestTargetConformanceWireDigest$' -v \
  ./tools/reference-transform-runtime-engine > "$output/native-test.log"
GOOS=js GOARCH=wasm CGO_ENABLED=0 \
  go test -count=1 -exec="$wasm_exec" -run '^TestTargetConformanceWireDigest$' -v \
  ./tools/reference-transform-runtime-engine > "$output/wasm-test.log"

get_digest() {
  local file="$1"
  local count
  count="$(grep -c 'TARGET_CONFORMANCE_SHA256=' "$file")"
  test "$count" = 1 || { echo "expected exactly one conformance digest in $file" >&2; exit 1; }
  sed -n 's/^.*TARGET_CONFORMANCE_SHA256=\([0-9a-f]\{64\}\).*$/\1/p' "$file"
}
native_output="$(get_digest "$output/native-test.log")"
wasm_output="$(get_digest "$output/wasm-test.log")"
[[ "$native_output" =~ ^[0-9a-f]{64}$ && "$native_output" == "$wasm_output" ]] || {
  echo "native and js/wasm Transform wire responses differ" >&2
  exit 1
}

# The artifacts are physically distinct; their business-level wire result is
# byte-identical. Build the wasm image twice without a repository edit.
go build -trimpath -buildvcs=false -o "$output/native-engine" \
  ./tools/reference-transform-runtime-engine
GOOS=js GOARCH=wasm CGO_ENABLED=0 \
  go build -trimpath -buildvcs=false -o "$output/engine-first.wasm" \
  ./tools/reference-transform-runtime-engine
GOOS=js GOARCH=wasm CGO_ENABLED=0 \
  go build -trimpath -buildvcs=false -o "$output/engine-second.wasm" \
  ./tools/reference-transform-runtime-engine
cmp "$output/engine-first.wasm" "$output/engine-second.wasm"
native_digest="sha256:$(sha256sum "$output/native-engine" | cut -d' ' -f1)"
wasm_digest="sha256:$(sha256sum "$output/engine-first.wasm" | cut -d' ' -f1)"
test "$native_digest" != "$wasm_digest" || { echo "target binaries unexpectedly identical" >&2; exit 1; }

jq -n \
  --arg source "$(git rev-parse HEAD)" \
  --arg native "$native_digest" \
  --arg wasm "$wasm_digest" \
  --arg result "$native_output" \
  --arg go "$(cat "$output/go-version.txt")" \
  --arg node "$(cat "$output/node-version.txt")" \
  '{
    schema: "tei.reference.transform-native-wasm-conformance/v1",
    sourceRevision: $source,
    engineLogicalId: "tei.runtime.engine.deterministic_transform.reference",
    engineContract: "tei.runtime-engine/deterministic-transform-v1",
    nativeTarget: "host/linux/amd64",
    wasmTarget: "js/wasm",
    nativeArtifactDigest: $native,
    wasmArtifactDigest: $wasm,
    typedWireResponseDigest: $result,
    goVersion: $go,
    nodeVersion: $node,
    wasmReproducible: true,
    crossTargetResultEqual: true
  }' > "$output/conformance-evidence.json"

cat "$output/conformance-evidence.json"
test -z "$(git status --porcelain=v1 --untracked-files=all)"
