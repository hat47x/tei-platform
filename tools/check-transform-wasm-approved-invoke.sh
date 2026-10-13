#!/usr/bin/env bash
# Extension to the shared native/js-wasm validation: require a signed
# Reference Host approval before launching the exact measured Wasm CLI image.
# Production browser Host authority and loaded-image attestation are out of scope.
set -euo pipefail

if [[ $# -ne 1 || "$1" != /* ]]; then
  echo "usage: $0 /absolute/output-directory-outside-repository" >&2
  exit 2
fi
output="$1"
repo="$(git rev-parse --show-toplevel)"
case "$output/" in
  "$repo/"*) echo "output directory must be outside repository" >&2; exit 2 ;;
esac
if [[ -n "$(git status --porcelain=v1 --untracked-files=all)" ]]; then
  echo "source repository is not clean" >&2
  exit 1
fi

# Reuse the existing target conformance and real Runtime Engine Lock checks.
# This invocation intentionally depends on its public output files rather than
# editing its source, which is separately owned by issue #973 PR #1605.
bash tools/check-transform-wasm-conformance.sh "$output"
test -s "$output/conformance-evidence.json"
test -s "$output/native-engine"
test -s "$output/engine-first.wasm"
native_digest="sha256:$(sha256sum "$output/native-engine" | cut -d' ' -f1)"
wasm_digest="sha256:$(sha256sum "$output/engine-first.wasm" | cut -d' ' -f1)"
jq -e --arg native "$native_digest" --arg wasm "$wasm_digest" '
  .nativeArtifactDigest == $native
  and .wasmArtifactDigest == $wasm
  and .realArtifactBindingVerified == true
  and .crossTargetResultEqual == true
' "$output/conformance-evidence.json" >/dev/null

# Export the exact Go-oracle test Source and typed-wire request as CLI fixtures.
TEI_TARGET_CLI_FIXTURE_OUT="$output" \
  go test -count=1 -run '^TestExportTargetCLIConformanceFixture$' -v \
    ./tools/reference-transform-runtime-engine > "$output/cli-fixture-test.log"
test -s "$output/cli-source.json"
test -s "$output/cli-request.json"
test -s "$output/cli-expected-response.json"

# The Reference Stage2 verifier checks independently signed Host registration,
# pinned Ed25519 public key, Host ID, Engine digest/target/contract and negatives.
TEI_WASM_ACCEPT_IMAGE="$output/engine-first.wasm" \
TEI_WASM_ACCEPT_DIGEST="$wasm_digest" \
  go test -count=1 -run '^TestReferenceWasmImageRequiresSignedHostAcceptance$' -v \
    ./tools/reference-plugin-self-build > "$output/wasm-host-approval.log" || {
      cat "$output/wasm-host-approval.log" >&2
      exit 1
    }
grep -qF "WASM_HOST_APPROVED_DIGEST=$wasm_digest" "$output/wasm-host-approval.log"
test "sha256:$(sha256sum "$output/engine-first.wasm" | cut -d' ' -f1)" = "$wasm_digest"

# Invoke the actual measured production CLI images, not go test's test wasm.
go_root="$(go env GOROOT)"
wasm_exec="$go_root/lib/wasm/go_js_wasm_exec"
if [[ ! -f "$wasm_exec" ]]; then wasm_exec="$go_root/misc/wasm/go_js_wasm_exec"; fi
test -f "$wasm_exec"
"$output/native-engine" -source "$output/cli-source.json" \
  < "$output/cli-request.json" > "$output/native-cli-response.json"
"$wasm_exec" "$output/engine-first.wasm" -source "$output/cli-source.json" \
  < "$output/cli-request.json" > "$output/wasm-cli-response.json"

for file in native-cli-response.json wasm-cli-response.json cli-expected-response.json; do
  jq -e 'type == "object" and .schema == "tei.reference.transform-engine-response/v1"' \
    "$output/$file" > /dev/null
done
cmp <(jq -cS . "$output/native-cli-response.json") \
    <(jq -cS . "$output/wasm-cli-response.json")
cmp <(jq -cS . "$output/cli-expected-response.json") \
    <(jq -cS . "$output/wasm-cli-response.json")
native_sha="$(jq -c . "$output/native-cli-response.json" | tr -d '\n' | sha256sum | cut -d' ' -f1)"
wasm_sha="$(jq -c . "$output/wasm-cli-response.json" | tr -d '\n' | sha256sum | cut -d' ' -f1)"
expected_sha="$(jq -r '.typedWireResponseDigest' "$output/conformance-evidence.json")"
[[ "$native_sha" == "$wasm_sha" && "$native_sha" == "$expected_sha" ]] || {
  echo "accepted exact wasm CLI changed the canonical typed response" >&2
  exit 1
}
test "sha256:$(sha256sum "$output/engine-first.wasm" | cut -d' ' -f1)" = "$wasm_digest"

jq -n \
  --arg revision "$(git rev-parse HEAD)" \
  --arg native "$native_digest" \
  --arg wasm "$wasm_digest" \
  --arg response "$wasm_sha" \
  '{
    schema: "tei.reference.transform-signed-wasm-cli-invocation/v1",
    sourceRevision: $revision,
    nativeArtifactDigest: $native,
    approvedWasmArtifactDigest: $wasm,
    actualCLIResponseDigest: $response,
    signedReferenceHostAccepted: true,
    exactImageInvokedAfterAcceptance: true,
    crossTargetCLIEqual: true,
    productionAuthorityVerified: false
  }' > "$output/approved-wasm-cli-evidence.json"
cat "$output/approved-wasm-cli-evidence.json"
test -z "$(git status --porcelain=v1 --untracked-files=all)"
