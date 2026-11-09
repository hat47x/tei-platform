#!/usr/bin/env bash
# Local qualification runner for #1121. Never merge/close from this script.
# Requires an independently prepared, full, clean, trusted Git checkout.
set -euo pipefail

# Fail before any tests if a later line was accidentally truncated or malformed.
bash -n "$0"

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$repo_root"

: "${EXPECTED_HEAD:?set EXPECTED_HEAD to the exact PR commit SHA}"
: "${EXPECTED_MAIN:?set EXPECTED_MAIN to the reviewed main commit SHA}"
if [[ ! "$EXPECTED_HEAD" =~ ^[[:xdigit:]]{40}$ ]] ||
   [[ ! "$EXPECTED_MAIN" =~ ^[[:xdigit:]]{40}$ ]]; then
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
    # origin/main is only a local cache. Check the actual remote ref both
    # before and after the run to reject a moving integration target.
    local remote_line
    if ! remote_line="$(git ls-remote --exit-code origin refs/heads/main)"; then
        printf '%s: could not verify remote main\n' "$stage" >&2
        exit 1
    fi
    if [[ "$remote_line" != "$EXPECTED_MAIN"$'\t'"refs/heads/main" ]]; then
        printf '%s: remote main has moved: %s\n' "$stage" "$remote_line" >&2
        exit 1
    fi
    if ! git merge-base --is-ancestor "$EXPECTED_MAIN" "$EXPECTED_HEAD"; then
        printf '%s: current main is not an ancestor of PR HEAD\n' "$stage" >&2
        exit 1
    fi
    status="$(git status --porcelain --untracked-files=all)"
    if [[ -n "$status" ]]; then
        printf '%s: checkout is not clean; refusing to qualify\n' "$stage" >&2
        exit 1
    fi
}

# Go reports exit 0 for an unmatched -run filter or skipped opt-in test.
# The explicit top-level PASS marker prevents those two false positives.
expect_test_pass() {
    local name="$1"
    shift
    local transcript
    transcript="$(mktemp)"
    if ! "$@" -v | tee "$transcript"; then
        rm -f "$transcript"
        printf 'Go test command failed: %s\n' "$name" >&2
        return 1
    fi
    if ! grep -Fq -- "--- PASS: ${name} (" "$transcript"; then
        rm -f "$transcript"
        printf 'Required test was missing or skipped: %s\n' "$name" >&2
        return 1
    fi
    rm -f "$transcript"
}

check_checkout before
echo "Qualification HEAD=$EXPECTED_HEAD main=$EXPECTED_MAIN"
go version

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

# Guard the qualification script itself against missing, stale or moved refs.
bash tools/reference-critical-lowering/qualify_gate_test.sh

go vet ./tools/reference-critical-lowering
go test -count=1 ./tools/reference-critical-lowering

expect_test_pass TestPracticalRuleChangePreviewsAndAcceptsBothMeaningAndApplicationSelection \
    go test -count=1 -run '^TestPracticalRuleChangePreviewsAndAcceptsBothMeaningAndApplicationSelection$' ./tools/reference-shared-acceptance

expect_test_pass TestPracticalBusinessRuleChangeReachesDriverAndWebWithoutTargetRuleList \
    go test -count=1 -run '^TestPracticalBusinessRuleChangeReachesDriverAndWebWithoutTargetRuleList$' ./tools/reference-first-web-target

# Uses a real running local HTTP server; the provider and read are synthetic.
expect_test_pass TestWebTargetHTTPBlackBoxTrace \
    env TEI_LOWERING_WEB_SMOKE=1 go test -count=1 -run '^TestWebTargetHTTPBlackBoxTrace$' ./tools/reference-critical-lowering

bash tools/repository-health-check.sh
check_checkout after
echo "LOCAL_CHECKS_PASSED HEAD=$EXPECTED_HEAD MAIN=$EXPECTED_MAIN"
echo "This alone does not qualify real DB/Host Authority or parent Gate review."
