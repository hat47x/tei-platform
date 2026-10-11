#!/usr/bin/env bash
# Compare complete Go package/test qualification on an exact private checkout
# and an independently verified, history-free public source candidate.
set -euo pipefail
if [[ "$#" -ne 4 ]]; then
  echo 'Usage: check-public-test-parity.sh SOURCE_CHECKOUT EXACT_COMMIT_SHA APPROVED_PATHS_FILE CANDIDATE_DIRECTORY' >&2
  exit 2
fi
source_arg=$1
sha=$2
paths=$3
candidate_arg=$4
here=$(cd "$(dirname "$0")" && pwd)
[[ "$sha" =~ ^[0-9a-f]{40}$ ]] || { echo 'invalid exact source SHA' >&2; exit 2; }
[[ -f "$paths" ]] || { echo 'reviewed path list missing' >&2; exit 2; }
[[ -f "$here/verify-candidate.sh" ]] || { echo 'independent verifier not installed; Draft PR #1195 prerequisite' >&2; exit 2; }
source=$(realpath -e "$(git -C "$source_arg" rev-parse --show-toplevel)")
candidate=$(realpath -e "$candidate_arg")
[[ -d "$candidate" && ! -L "$candidate" ]] || { echo 'candidate must be an ordinary directory' >&2; exit 2; }
[[ "$source" != "$candidate" ]] || { echo 'candidate is the source checkout' >&2; exit 2; }
case "$candidate/" in "$source/"*) echo 'candidate must be outside source' >&2; exit 2;; esac
[[ "$(git -C "$source" rev-parse HEAD)" == "$sha" ]] || { echo 'source checkout HEAD is not requested SHA' >&2; exit 2; }
[[ -z "$(git -C "$source" status --porcelain=v1 --ignored --untracked-files=all)" ]] || {
  echo 'source checkout must have clean tracked/untracked/ignored state' >&2
  exit 2
}
umask 077
logs=$(mktemp -d)
trap 'rm -rf -- "$logs"' EXIT
export GOENV=off GOWORK=off GOFLAGS=

verify_candidate() {
  bash "$here/verify-candidate.sh" "$source" "$sha" "$paths" "$candidate" > "$logs/candidate-integrity"
}

run_go_suite() {
  local root=$1
  local id=$2
  (
    cd "$root"
    go list ./... | LC_ALL=C sort > "$logs/$id.packages" || return
    go vet ./... > "$logs/$id.vet" 2>&1 || return
    go test -count=1 -timeout 60m ./... > "$logs/$id.tests" 2>&1 || return
    go build -o "$logs/$id-tei" ./cmd/tei > "$logs/$id.build" 2>&1 || return
    go run ./cmd/tei validate -workspace reference/workspace > "$logs/$id.validate" 2>&1 || return
    tail -n 1 "$logs/$id.validate" | grep -qx 'ok' || return
  )
}

verify_candidate
echo '==> exact source checkout: full go list / vet / test / CLI build / workspace validate'
if ! run_go_suite "$source" private; then
  echo 'FAIL: source checkout Go suite failed; no parity qualification' >&2
  exit 1
fi
echo '==> history-free public candidate: same suite'
if ! run_go_suite "$candidate" selected; then
  echo 'FAIL: public source candidate Go suite failed; no parity qualification' >&2
  exit 1
fi
if ! diff -u "$logs/private.packages" "$logs/selected.packages"; then
  echo 'FAIL: public candidate Go package set differs from source' >&2
  exit 1
fi
[[ -z "$(git -C "$source" status --porcelain=v1 --ignored --untracked-files=all)" ]] || {
  echo 'FAIL: source checkout mutated during suite' >&2
  exit 1
}
verify_candidate
echo 'TEST_PARITY_AT_FIXED_SHA: PASS (source and selected candidate)'
echo 'NOT_PUBLICATION_APPROVAL: historical Git-dependent tests, CI, secrets, license and public rights require independent review.'
