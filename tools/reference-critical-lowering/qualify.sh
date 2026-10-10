#!/usr/bin/env bash
# Local qualification driver for #1121. This does NOT merge or close a PR.
# It requires an independently prepared, trusted, complete clean checkout.
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$repo_root"

: "${EXPECTED_HEAD:?set EXPECTED_HEAD to the exact PR commit SHA}"
: "${EXPECTED_MAIN:?set EXPECTED_MAIN to the reviewed main commit SHA}"
if [[ ! "$EXPECTED_HEAD" =~ ^[[:xdigit:]]{40}$ ]] || [[ ! "$EXPECTED_MAIN" =~ ^[[:xdigit:]]{40}$ ]]; then
    echo "Both expected revisions must be complete 40-character SHAs" >&2
    exit 2
fi

check_checkout() {
    local stage="$1"
    local head main status
    head="$(git rev-parse HEAD)"
    main="$(git rev-parse refs/remotes/origin/main)"
    if [[ "$head" != "$EXPECTED_HEAD" || "$main" != "$EXPECTED_MAIN" ]]; then
        printf '%s: stale revision (HEAD=%s origin/main=%s)\n' "$stage" "$head" "$main" >&2
        exit 1
    fi
    # Health evidence must cover the exact target main, not an outdated
    # feature branch that predates later main changes.
    if ! git merge-base --is-ancestor "$EXPECTED_MAIN" "$EXPECTED_HEAD"; then
        printf '%s: current main is not an ancestor of tested PR HEAD\n' "$stage" >&2
        exit 1
    fi
    status="$(git status --porcelain --untracked-files=all)"
    if [[ -n "$status" ]]; then
        printf '%s: checkout is not clean; refusing to qualify\n' "$stage" >&2
        exit 1
    fi
}

check_checkout before
echo "qualification inputs: HEAD=$EXPECTED_HEAD origin/main=$EXPECTED_MAIN"
echo "go: $(go version)"

# Include all Go source files in this lane, including opt-in Web E2E.
mapfile -t reference_go < <(find tools/reference-critical-lowering -maxdepth 1 -type f -name '*.go' | LC_ALL=C sort)
if (( ${#reference_go[@]} == 0 )); then
    echo "reference Go files missing" >&2
    exit 1
fi
unformatted="$(gofmt -l "${reference_go[@]}")"
if [[ -n "$unformatted" ]]; then
    printf 'gofmt required:\n%s\n' "$unformatted" >&2
    exit 1
fi

go vet ./tools/reference-critical-lowering
go test -count=1 ./tools/reference-critical-lowering

# Reuse existing Source Acceptance and Web Driver contracts; do not
# introduce duplicate acceptance or web runtime implementations in #1121.
go test -count=1 -run '^TestPracticalRuleChangePreviewsAndAcceptsBothMeaningAndApplicationSelection$' ./tools/reference-shared-acceptance
go test -count=1 -run '^TestPracticalBusinessRuleChangeReachesDriverAndWebWithoutTargetRuleList$' ./tools/reference-first-web-target

# This actually builds and serves the EXISTING Web command over HTTP.
TEI_LOWERING_WEB_SMOKE=1 go test -count=1 -run '^TestWebTargetHTTPBlackBoxTrace$' -v ./tools/reference-critical-lowering

# This is the full repository integration gate, not a focused-test shortcut.
bash tools/repository-health-check.sh
check_checkout after

echo "LOCAL_CHECKS_PASSED HEAD=$EXPECTED_HEAD MAIN=$EXPECTED_MAIN"
echo "Qualification still requires independent review, external provenance, current Host Authority/DB evidence, and explicit PR state checks."
