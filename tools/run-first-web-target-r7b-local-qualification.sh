#!/usr/bin/env bash
set -euo pipefail

repo_root="$(git rev-parse --show-toplevel 2>/dev/null || true)"
if [[ -z "$repo_root" || "$PWD" != "$repo_root" ]]; then
  printf 'Run from the TEI repository root.\n' >&2
  exit 2
fi

if [[ -v TEI_R7B_QUALIFICATION_REMOTE ]]; then
  remote="$TEI_R7B_QUALIFICATION_REMOTE"
else
  remote="origin"
fi
if [[ -v TEI_R7B_QUALIFICATION_BRANCH ]]; then
  qualification_branch="$TEI_R7B_QUALIFICATION_BRANCH"
else
  qualification_branch="research/selective-knowledge-routing-r6"
fi
if [[ -v TEI_R7B_EXPECTED_GO_VERSION ]]; then
  expected_go_version="$TEI_R7B_EXPECTED_GO_VERSION"
else
  expected_go_version="go1.23.2"
fi
receipt_path="${TEI_R7B_QUALIFICATION_RECEIPT:-}"

if [[ -n "$receipt_path" ]]; then
  case "$receipt_path" in
    "$repo_root"|"$repo_root"/*)
      printf 'Qualification receipt must remain outside the repository: %s\n' "$receipt_path" >&2
      exit 2
      ;;
  esac
  if [[ -e "$receipt_path" ]]; then
    printf 'Qualification receipt already exists: %s\n' "$receipt_path" >&2
    exit 2
  fi
  mkdir -p "$(dirname "$receipt_path")"
  receipt_dir="$(cd "$(dirname "$receipt_path")" && pwd -P)"
  receipt_path="$receipt_dir/$(basename "$receipt_path")"
fi

for command in git go bash awk sort diff grep mktemp; do
  if ! command -v "$command" >/dev/null 2>&1; then
    printf '%s is required for local R7B qualification.\n' "$command" >&2
    exit 2
  fi
done

git fetch "$remote" --prune

current_branch="$(git branch --show-current)"
if [[ "$current_branch" != "$qualification_branch" ]]; then
  printf 'Expected branch %s, got %s.\n' "$qualification_branch" "$current_branch" >&2
  exit 1
fi

status_before="$(git status --porcelain --untracked-files=all)"
if [[ -n "$status_before" ]]; then
  printf 'Repository is not clean before local R7B qualification:\n%s\n' "$status_before" >&2
  exit 1
fi

head_before="$(git rev-parse HEAD)"
remote_head="$(git rev-parse "$remote/$qualification_branch")"
base_before="$(git rev-parse "$remote/main")"

if [[ "$head_before" != "$remote_head" ]]; then
  printf 'Local qualification branch is not at its remote tip: local=%s remote=%s\n' \
    "$head_before" "$remote_head" >&2
  exit 1
fi
if ! git merge-base --is-ancestor "$base_before" "$head_before"; then
  printf 'Current %s/main is not an ancestor of the qualification HEAD.\n' "$remote" >&2
  exit 1
fi

actual_go_version="$(GOENV=off GOWORK=off GOFLAGS= go env GOVERSION)"
if [[ "$actual_go_version" != "$expected_go_version" ]]; then
  printf 'Go version=%s, want %s.\n' "$actual_go_version" "$expected_go_version" >&2
  exit 1
fi

scratch="$(mktemp -d "${TMPDIR:-/tmp}/tei-r7b-local-qualification.XXXXXX")"
trap 'rm -rf "$scratch"' EXIT

LC_ALL=C sort >"$scratch/expected-paths" <<'PATHS'
tools/internal/firstwebtargetevidence/semantics.go
tools/internal/firstwebtargetevidence/semantics_pressure_test.go
tools/internal/firstwebtargetevidence/semantics_test.go
tools/reference-first-web-target-evidence-audit/main.go
tools/reference-first-web-target-r7b-repeatability/main.go
tools/reference-first-web-target-r7b-repeatability/main_test.go
tools/reference-first-web-target-r7b-replay-audit/main.go
tools/reference-first-web-target-r7b-replay-audit/main_test.go
tools/reference-first-web-target-r7b-replay-plan/main.go
tools/reference-first-web-target-r7b-replay-plan/main_test.go
tools/run-first-web-target-r7b-replay.sh
PATHS

git diff --name-only "$base_before...$head_before" | LC_ALL=C sort >"$scratch/actual-paths"
if ! diff -u "$scratch/expected-paths" "$scratch/actual-paths"; then
  printf 'Qualification diff is not exactly the canonical 11 paths.\n' >&2
  exit 1
fi

check_entry() {
  path="$1"
  expected_mode="$2"
  expected_blob="$3"
  actual_mode="$(git ls-files -s -- "$path" | awk 'NR == 1 {print $1}')"
  actual_blob="$(git hash-object "$path")"
  if [[ "$actual_mode" != "$expected_mode" || "$actual_blob" != "$expected_blob" ]]; then
    printf '%s mode/blob mismatch: mode=%s blob=%s want mode=%s blob=%s\n' \
      "$path" "$actual_mode" "$actual_blob" "$expected_mode" "$expected_blob" >&2
    exit 1
  fi
}

check_entry tools/internal/firstwebtargetevidence/semantics.go 100644 9641e3539bb048b13e7e9a4bb245ee457fe1e7a5
check_entry tools/internal/firstwebtargetevidence/semantics_pressure_test.go 100644 95553ba10756399a50e52e639f7d860f4420f29f
check_entry tools/internal/firstwebtargetevidence/semantics_test.go 100644 ba17f69a7e375cfdeec0db544986d51f99b26192
check_entry tools/reference-first-web-target-evidence-audit/main.go 100644 3878dae15b06a3b57954cf196b108da1cc03bbc7
check_entry tools/reference-first-web-target-r7b-repeatability/main.go 100644 7b2cf09460319b13241fa44e59f865d3864734ff
check_entry tools/reference-first-web-target-r7b-repeatability/main_test.go 100644 30a907578836c9b1b33b79bf4d521fb809a49e74
check_entry tools/reference-first-web-target-r7b-replay-audit/main.go 100644 a370e0c5efc4399e324be7a0ad4801626f05a5da
check_entry tools/reference-first-web-target-r7b-replay-audit/main_test.go 100644 2fab7e44d49ddb2778a2bcb55cae67c0f48c1c99
check_entry tools/reference-first-web-target-r7b-replay-plan/main.go 100644 40a1b1b19fa8bf16c43dd2fd0d4f549070efe7ab
check_entry tools/reference-first-web-target-r7b-replay-plan/main_test.go 100644 d8fc5e8a1cadb82783c3804fa04d147c5954b3ed
check_entry tools/run-first-web-target-r7b-replay.sh 100755 2f55ec06f29bbca8f4a9e6ce04816bd311a8af03

bash -n tools/run-first-web-target-r7b-replay.sh
bash tools/repository-health-check.sh

head_after="$(git rev-parse HEAD)"
status_after="$(git status --porcelain --untracked-files=all)"
if [[ "$head_after" != "$head_before" || -n "$status_after" ]]; then
  printf 'Repository changed during local R7B qualification.\n' >&2
  printf 'HEAD before=%s after=%s\n' "$head_before" "$head_after" >&2
  if [[ -n "$status_after" ]]; then
    printf '%s\n' "$status_after" >&2
  fi
  exit 1
fi

git fetch "$remote" --prune
base_after="$(git rev-parse "$remote/main")"
main_moved=false
if [[ "$base_after" != "$base_before" ]]; then
  main_moved=true
  if ! git merge-base --is-ancestor "$base_before" "$base_after"; then
    printf '%s/main was rewritten during qualification; stop and reconcile.\n' "$remote" >&2
    exit 1
  fi
  git diff --name-only "$base_before..$base_after" | LC_ALL=C sort -u >"$scratch/main-movement-paths"
  if grep -Fxf "$scratch/expected-paths" "$scratch/main-movement-paths" >"$scratch/canonical-overlap"; then
    printf 'Canonical-path overlap appeared on %s/main during qualification:\n' "$remote" >&2
    cat "$scratch/canonical-overlap" >&2
    exit 1
  fi
fi

if [[ -n "$receipt_path" ]]; then
  cat >"$receipt_path" <<EOF
{
  "evidenceNature": "first-web-target-r7b-local-source-qualification-receipt",
  "qualificationRepositoryRevision": "$head_before",
  "baseMainBefore": "$base_before",
  "baseMainAfter": "$base_after",
  "mainMovedDuringQualification": $main_moved,
  "goVersion": "$actual_go_version",
  "canonicalPathCount": 11,
  "replayRunnerMode": "100755",
  "replayRunnerBlob": "2f55ec06f29bbca8f4a9e6ce04816bd311a8af03",
  "repositoryHealthPassed": true,
  "repositoryPostRunClean": true
}
EOF
fi

printf 'R7B local source qualification PASS\n'
printf '  base main before: %s\n' "$base_before"
printf '  candidate HEAD:   %s\n' "$head_before"
printf '  Go version:       %s\n' "$actual_go_version"
printf '  base main after:  %s\n' "$base_after"
printf '  main moved:       %s\n' "$main_moved"
if [[ -n "$receipt_path" ]]; then
  printf '  receipt:          %s\n' "$receipt_path"
fi
