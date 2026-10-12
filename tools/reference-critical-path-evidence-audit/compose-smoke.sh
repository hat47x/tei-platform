#!/usr/bin/env bash
# Run cross-lane smoke checks in a disposable detached worktree:
# compose pinned Draft tips (historical mode), or check integrated current main.
# Never merge to main or alter owner refs.
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

mode="${TEI_CP_COMPOSE_MODE:-draft-tips}"
case "$mode" in
  draft-tips|integrated-main) ;;
  *) die "TEI_CP_COMPOSE_MODE must be draft-tips or integrated-main" ;;
esac
expected_main="${TEI_CP_COMPOSE_MAIN:-}"
expected_1119="${TEI_CP_COMPOSE_1119:-}"
expected_1120="${TEI_CP_COMPOSE_1120:-}"
expected_1124="${TEI_CP_COMPOSE_1124:-}"
expected_1121="${TEI_CP_COMPOSE_1121:-}"
[[ "$expected_main" =~ ^[0-9a-f]{40}$ ]] ||
  die "supply TEI_CP_COMPOSE_MAIN as the exact 40-character current main SHA"
if [[ "$mode" == draft-tips ]]; then
  for value in "$expected_1119" "$expected_1120" "$expected_1124" "$expected_1121"; do
    [[ "$value" =~ ^[0-9a-f]{40}$ ]] ||
      die "draft-tips mode requires four exact TEI_CP_COMPOSE_* branch tip SHAs"
  done
else
  [[ -z "$expected_1119$expected_1120$expected_1124$expected_1121" ]] ||
    die "integrated-main mode refuses stale Draft branch pins"
fi
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
git fetch --no-tags "$remote" --prune || die "fetch failed"
check_tip() {
  local branch="$1" expected="$2" actual
  actual="$(git rev-parse "refs/remotes/$remote/$branch" 2>/dev/null)" ||
    die "remote branch not found: $branch"
  [[ "$actual" == "$expected" ]] || die "stale pin for $branch: $actual"
}
check_tip main "$expected_main"
if [[ "$mode" == draft-tips ]]; then
  check_tip work/issue-1119-critical-path-product-gate "$expected_1119"
  check_tip issue-1120-typed-reference-contract "$expected_1120"
  check_tip work/issue-1124-studio-authoring-projection "$expected_1124"
  check_tip work/issue-1121-lowering-contract "$expected_1121"
fi

# Merge only in a disposable detached worktree. Branch refs remain untouched.
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
if [[ "$mode" == draft-tips ]]; then
  for head in "$expected_1119" "$expected_1120" "$expected_1124" "$expected_1121"; do
    git -C "$wt" -c user.name="TEI Integration Test" -c user.email="integration.invalid@example.invalid" \
      -c commit.gpgsign=false merge --no-ff --no-edit "$head" ||
      die "candidate branches conflict or cannot be combined (no owner branch altered)"
  done
fi
clean "$wt"
integration_head="$(git -C "$wt" rev-parse HEAD)"
integration_tree="$(git -C "$wt" rev-parse HEAD^{tree})"
printf 'CROSS_LANE_SMOKE_BEGIN mode=%s main=%s tested-head=%s tree=%s\n' \
  "$mode" "$expected_main" "$integration_head" "$integration_tree"

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
  # Only code on the pinned tested HEAD can count. As of the original main
  # Reference this is live R1 HTTP positive, synthetic Provider only. Later
  # changes to the smoke runner must be separately inspected for their scope.
  TEI_CP_LIVE_R1_SMOKE=1 node tools/reference-critical-path-evidence-audit/live-r1-smoke.mjs
  bash tools/repository-health-check.sh
)
clean "$wt"
[[ "$(git -C "$wt" rev-parse HEAD)" == "$integration_head" ]] ||
  die "temporary merged HEAD moved"
git fetch --no-tags "$remote" --prune || die "post-test fetch failed"
check_tip main "$expected_main"
if [[ "$mode" == draft-tips ]]; then
  check_tip work/issue-1119-critical-path-product-gate "$expected_1119"
  check_tip issue-1120-typed-reference-contract "$expected_1120"
  check_tip work/issue-1124-studio-authoring-projection "$expected_1124"
  check_tip work/issue-1121-lowering-contract "$expected_1121"
fi
[[ "$(git rev-parse HEAD)" == "$original_head" ]] ||
  die "original checkout HEAD moved"
clean "$root"
if [[ "$mode" == integrated-main ]]; then
  printf 'CURRENT_MAIN_CROSS_LANE_SMOKE_ONLY_NO_PRODUCT_GATE\n'
else
  printf 'CROSS_LANE_CANDIDATE_SMOKE_ONLY_NO_PRODUCT_GATE\n'
fi
printf 'mode=%s\nmain=%s\nintegrationHead=%s\nintegrationTree=%s\n' \
  "$mode" "$expected_main" "$integration_head" "$integration_tree"
printf 'limits=NOT_OWNER_PR_QUALIFICATION_NOT_EXTERNAL_DB_NOT_INDEPENDENT_READ_NOT_HUMAN_APPROVAL\n'
