#!/usr/bin/env bash
# Exact-head, local qualification for #1119. This does not qualify Product Gates.
set -euo pipefail

die() { printf 'critical-path qualification blocked: %s\n' "$*" >&2; exit 1; }
for cmd in git go node bash awk gofmt; do
  command -v "$cmd" >/dev/null 2>&1 || die "missing required command: $cmd"
done

repo_root="$(git rev-parse --show-toplevel 2>/dev/null)" || die "not a Git checkout"
[[ "$PWD" == "$repo_root" ]] || die "run at the repository root"
[[ "$(git branch --show-current)" == "work/issue-1119-critical-path-product-gate" ]] ||
  die "not on the #1119 work branch"

expected_head="${TEI_CP_EXPECTED_HEAD:-}"
expected_main="${TEI_CP_EXPECTED_MAIN:-}"
remote="${TEI_CP_REMOTE:-origin}"
[[ "$expected_head" =~ ^[0-9a-f]{40}$ ]] || die "TEI_CP_EXPECTED_HEAD must be an exact 40-hex commit"
[[ "$expected_main" =~ ^[0-9a-f]{40}$ ]] || die "TEI_CP_EXPECTED_MAIN must be an exact 40-hex commit"
[[ "$remote" =~ ^[A-Za-z0-9._/-]+$ ]] || die "unsafe TEI_CP_REMOTE"
[[ "$remote" != -* ]] || die "TEI_CP_REMOTE must not start with a dash"

# Do not allow a local/forked remote to masquerade as the authoritative main.
# Only canonical GitHub transport URLs for this repository are allowed.
remote_url="$(git remote get-url "$remote" 2>/dev/null)" ||
  die "remote not configured: $remote"
case "$remote_url" in
  https://github.com/hat47x/tei-platform|https://github.com/hat47x/tei-platform.git|\
  git@github.com:hat47x/tei-platform|git@github.com:hat47x/tei-platform.git|\
  ssh://git@github.com/hat47x/tei-platform|ssh://git@github.com/hat47x/tei-platform.git) ;;
  *) die "unexpected remote repository URL; require canonical hat47x/tei-platform" ;;
esac

clean() {
  local stage="$1" status unusual
  status="$(git status --porcelain=v1 --untracked-files=all --ignored=matching)" ||
    die "$stage: git status failed"
  [[ -z "$status" ]] || die "$stage: untracked/ignored/tracked workspace content exists"
  unusual="$(git ls-files -v | awk 'substr($0,1,1)!="H" {print}')" ||
    die "$stage: tracked index inspection failed"
  [[ -z "$unusual" ]] || die "$stage: assume-unchanged/skip-worktree or other nonstandard index entry exists"
}

clean before
head="$(git rev-parse HEAD)"
[[ "$head" == "$expected_head" ]] || die "HEAD is not the requested exact candidate"
git fetch --no-tags "$remote" --prune || die "cannot refresh remote main"
remote_head="$(git rev-parse "refs/remotes/$remote/work/issue-1119-critical-path-product-gate" 2>/dev/null)" ||
  die "remote work branch is missing"
[[ "$remote_head" == "$head" ]] || die "candidate changed on remote"
main="$(git rev-parse "refs/remotes/$remote/main" 2>/dev/null)" ||
  die "remote main is missing"
[[ "$main" == "$expected_main" ]] || die "main moved: refresh and specify its new exact SHA"
git merge-base --is-ancestor "$main" "$head" ||
  die "candidate is not based on current main; integrate current main safely, then rerun"

# Never silently qualify code from another parallel work lane.
changed="$(git diff --name-only "$main...$head")"
while IFS= read -r path; do
  [[ -z "$path" ]] && continue
  case "$path" in
    plan/workstreams/TEI_PRODUCT_CRITICAL_PATH_2026-10.md|tools/reference-critical-path-evidence-audit/*) ;;
    *) die "out-of-scope changed path: $path" ;;
  esac
done <<< "$changed"

bash -n tools/reference-critical-path-evidence-audit/qualify.sh
bash -n tools/reference-critical-path-evidence-audit/compose-smoke.sh
node --check tools/reference-critical-path-evidence-audit/live-r1-smoke.mjs

unformatted="$(gofmt -l tools/reference-critical-path-evidence-audit/*.go)"
[[ -z "$unformatted" ]] || die "gofmt discrepancy: $unformatted"

GOENV=off GOWORK=off GOFLAGS= go test -race -count=1 ./tools/reference-critical-path-evidence-audit
GOENV=off GOWORK=off GOFLAGS= go vet ./tools/reference-critical-path-evidence-audit
# Reuse the existing acceptance/Driver conformance suites; no second runtime.
GOENV=off GOWORK=off GOFLAGS= go test -count=1 ./tools/reference-shared-acceptance \
  -run '^TestPracticalRuleChange(PreviewsAndAcceptsBothMeaningAndApplicationSelection|RejectsBrokenOrRepeatedDeltaWithoutChangingSource|RefusesStaleReviewBeforeWritingEitherSource)$'
GOENV=off GOWORK=off GOFLAGS= go test -count=1 ./tools/reference-first-web-target \
  -run '^(TestPracticalBusinessRuleChangeReachesDriverAndWebWithoutTargetRuleList|TestReferenceCommandScenarioRejectsMissingValuesAcrossAllLines)$'
bash tools/repository-health-check.sh

[[ "$(git rev-parse HEAD)" == "$head" ]] || die "HEAD moved while testing"
clean after
git fetch --no-tags "$remote" --prune || die "cannot recheck remote tips"
[[ "$(git rev-parse "refs/remotes/$remote/main")" == "$main" ]] ||
  die "main moved during qualification"
[[ "$(git rev-parse "refs/remotes/$remote/work/issue-1119-critical-path-product-gate")" == "$head" ]] ||
  die "candidate moved during qualification"

printf 'CRITICAL_PATH_LOCAL_SOURCE_QUALIFIED_ONLY\n'
printf 'head=%s\nmain=%s\n' "$head" "$main"
printf 'go_version=%s\n' "$(GOENV=off GOWORK=off GOFLAGS= go env GOVERSION)"
printf 'health_runner_blob=%s\n' "$(git hash-object tools/repository-health-check.sh)"
printf 'qualifier_blob=%s\n' "$(git hash-object tools/reference-critical-path-evidence-audit/qualify.sh)"
printf 'limits=NO_STUDIO_DRAFT_INTEGRATION_NO_ACTUAL_WEB_R1_NO_DB_NO_PRODUCT_GATE\n'
