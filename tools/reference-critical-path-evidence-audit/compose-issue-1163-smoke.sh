#!/usr/bin/env bash
# Issue #1163-specific qualification probe (reuses #1119 contracts). Main already contains the
# historical #1119/#1120/#1124/#1121 PRs. Compose only a pinned #1163
# candidate on current main inside a disposable detached worktree.
# Never change main, the candidate branch, or the caller's working tree.
set -euo pipefail
die() { printf 'cross-lane integration blocked: %s\n' "$*" >&2; exit 1; }

for cmd in git go node bash mktemp; do
  command -v "$cmd" >/dev/null 2>&1 || die "missing command: $cmd"
done
root="$(git rev-parse --show-toplevel 2>/dev/null)" || die "not a git checkout"
[[ "$PWD" == "$root" ]] || die "run from the repository root"
remote="${TEI_CP_COMPOSE_REMOTE:-origin}"
[[ "$remote" =~ ^[A-Za-z0-9._/-]+$ && "$remote" != -* ]] ||
  die "invalid remote name"
url="$(git remote get-url "$remote" 2>/dev/null)" || die "remote missing"
case "$url" in
  https://github.com/hat47x/tei-platform|https://github.com/hat47x/tei-platform.git|\
  git@github.com:hat47x/tei-platform|git@github.com:hat47x/tei-platform.git|\
  ssh://git@github.com/hat47x/tei-platform|ssh://git@github.com/hat47x/tei-platform.git) ;;
  *) die "remote is not canonical hat47x/tei-platform" ;;
esac

expected_main="${TEI_CP_COMPOSE_MAIN:-}"
expected_1163="${TEI_CP_COMPOSE_1163:-}"
for value in "$expected_main" "$expected_1163"; do
  [[ "$value" =~ ^[0-9a-f]{40}$ ]] ||
    die "supply exact 40-character TEI_CP_COMPOSE_MAIN and TEI_CP_COMPOSE_1163 SHAs"
done
for obsolete in TEI_CP_COMPOSE_1119 TEI_CP_COMPOSE_1120 TEI_CP_COMPOSE_1124 TEI_CP_COMPOSE_1121; do
  [[ -z "${!obsolete:-}" ]] ||
    die "$obsolete is obsolete: its producer was merged into main; remove it"
done
clean() {
  local where="$1" status unusual
  status="$(git -C "$where" status --porcelain=v1 --untracked-files=all --ignored=matching)" ||
    die "cannot inspect worktree cleanliness"
  [[ -z "$status" ]] || die "worktree is not clean: $where"
  unusual="$(git -C "$where" ls-files -v | awk 'substr($0,1,1)!="H" {print}')" ||
    die "cannot inspect tracked index flags"
  [[ -z "$unusual" ]] || die "nonstandard tracked index state: $where"
}
clean "$root"
original_head="$(git rev-parse HEAD)"
[[ "$original_head" == "$expected_1163" ]] ||
  die "run from the exact #1163 candidate HEAD, not an unrelated checkout"

git fetch --no-tags "$remote" --prune || die "fetch failed"
check_tip() {
  local name="$1" expected="$2" actual
  actual="$(git rev-parse "refs/remotes/$remote/$name" 2>/dev/null)" ||
    die "remote branch not found: $name"
  [[ "$actual" == "$expected" ]] || die "stale pin for $name: $actual"
}
check_tip main "$expected_main"
check_tip work/issue-1163-review-evidence-link "$expected_1163"

# Merge only in a disposable detached worktree, never the owner branches.
scratch="$(mktemp -d "${TMPDIR:-/tmp}/tei-cp-compose.XXXXXX")"
wt="$scratch/checkout"
has_worktree=false
cleanup() {
  if [[ "$has_worktree" == true ]]; then
    git -C "$root" worktree remove --force "$wt" >/dev/null 2>&1 || true
  fi
  rm -rf -- "$scratch"
}
trap cleanup EXIT
git worktree add --detach "$wt" "$expected_main" >/dev/null ||
  die "cannot create isolated worktree"
has_worktree=true
git -C "$wt" -c user.name="TEI Integration Test" \
  -c user.email="integration.invalid@example.invalid" \
  -c commit.gpgsign=false merge --no-ff --no-edit "$expected_1163" ||
  die "candidate cannot be combined with current main (no owner branch altered)"
clean "$wt"
integration_head="$(git -C "$wt" rev-parse HEAD)"
integration_tree="$(git -C "$wt" rev-parse HEAD^{tree})"

# An outdated or competing copy of the test must never masquerade as this PR.
for p in \
  tools/reference-critical-path-evidence-audit/main.go \
  tools/reference-critical-path-evidence-audit/main_test.go \
  tools/reference-critical-path-evidence-audit/live-r1-smoke.mjs; do
  candidate_blob="$(git -C "$root" rev-parse "$expected_1163:$p")" ||
    die "candidate lacks $p"
  tested_blob="$(git -C "$wt" rev-parse "HEAD:$p")" ||
    die "integration worktree lacks $p"
  [[ "$candidate_blob" == "$tested_blob" ]] ||
    die "integration would test a different $p than #1163 candidate"
done
printf 'CANDIDATE_COMPOSITION_BEGIN main=%s candidate=%s integration=%s tree=%s\n' \
  "$expected_main" "$expected_1163" "$integration_head" "$integration_tree"
(
  cd "$wt"
  export GOENV=off GOWORK=off GOFLAGS=
  export TEI_CP_ORIGINAL_REPOSITORY_ROOT="$root"
  export TEI_CP_INTEGRATION_TEMP_DIR="$scratch"
  node --check tools/reference-critical-path-evidence-audit/live-r1-smoke.mjs
  node --test reference/studio-authoring/bridge.test.mjs
  TEI_STUDIO_E2E=1 node reference/studio-authoring/e2e.mjs
  go test -race -count=1 ./internal/modelref/... ./internal/workspace
  go test -count=1 ./tools/reference-critical-lowering
  go test -count=1 ./tools/reference-critical-path-evidence-audit
  go test -count=1 ./tools/reference-first-web-target \
    -run '^(TestPracticalBusinessRuleChangeReachesDriverAndWebWithoutTargetRuleList|TestReferenceCommandScenarioRejectsMissingValuesAcrossAllLines)$'
  # Test-only acceptance and three synthetic Web binaries; not a Product Gate.
  TEI_CP_LIVE_R1_SMOKE=1 node tools/reference-critical-path-evidence-audit/live-r1-smoke.mjs
  bash tools/repository-health-check.sh
)
clean "$wt"
[[ "$(git -C "$wt" rev-parse HEAD)" == "$integration_head" ]] ||
  die "temporary merged HEAD moved"
git fetch --no-tags "$remote" --prune || die "post-test fetch failed"
check_tip main "$expected_main"
check_tip work/issue-1163-review-evidence-link "$expected_1163"
[[ "$(git rev-parse HEAD)" == "$original_head" ]] ||
  die "original checkout HEAD moved"
clean "$root"
printf 'ISSUE_1163_CANDIDATE_SMOKE_ONLY_NO_PRODUCT_GATE\n'
printf 'main=%s\ncandidate=%s\nintegrationHead=%s\nintegrationTree=%s\n' \
  "$expected_main" "$expected_1163" "$integration_head" "$integration_tree"
printf 'limits=TEST_ONLY_ACCEPTANCE_SYNTHETIC_PROVIDER_THREE_BINARIES_N11_SYNTHETIC_ONLY_NOT_DB_NOT_HUMAN_APPROVAL\n'
