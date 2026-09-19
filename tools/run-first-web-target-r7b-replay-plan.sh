#!/usr/bin/env bash
set -euo pipefail

repo_root="$(git rev-parse --show-toplevel 2>/dev/null || true)"
if [[ -z "$repo_root" || "$PWD" != "$repo_root" ]]; then
  printf 'Run from the TEI repository root.\n' >&2
  exit 1
fi

intake_path="${TEI_R7B_EXECUTION_INTAKE:-}"
receipt_path="${TEI_R7B_INTAKE_RECEIPT:-}"
output_dir="${TEI_R7B_REPLAY_PLAN_OUTPUT_DIR:-}"

if [[ -z "$intake_path" || -z "$receipt_path" || -z "$output_dir" ]]; then
  cat >&2 <<'EOF'
TEI_R7B_EXECUTION_INTAKE,
TEI_R7B_INTAKE_RECEIPT,
and TEI_R7B_REPLAY_PLAN_OUTPUT_DIR are required.
EOF
  exit 1
fi

if [[ ! -f "$intake_path" ]]; then
  printf 'R7B execution intake not found: %s\n' "$intake_path" >&2
  exit 1
fi
if [[ ! -f "$receipt_path" ]]; then
  printf 'R7B intake receipt not found: %s\n' "$receipt_path" >&2
  exit 1
fi

canonical_file() {
  local path="$1"
  local dir
  dir="$(cd "$(dirname "$path")" && pwd -P)"
  printf '%s/%s\n' "$dir" "$(basename "$path")"
}

intake_path="$(canonical_file "$intake_path")"
receipt_path="$(canonical_file "$receipt_path")"

case "$intake_path" in
  "$repo_root"|"$repo_root"/*)
    printf 'R7B execution intake must remain outside the repository: %s\n' "$intake_path" >&2
    exit 1
    ;;
esac
case "$receipt_path" in
  "$repo_root"|"$repo_root"/*)
    printf 'R7B intake receipt must remain outside the repository: %s\n' "$receipt_path" >&2
    exit 1
    ;;
esac

if [[ "$(basename "$intake_path")" != "r7b-execution-intake.json" ]]; then
  printf 'Unexpected R7B execution-intake filename: %s\n' "$(basename "$intake_path")" >&2
  exit 1
fi
if [[ "$(basename "$receipt_path")" != "r7b-intake-receipt.json" ]]; then
  printf 'Unexpected R7B intake-receipt filename: %s\n' "$(basename "$receipt_path")" >&2
  exit 1
fi

case "$output_dir" in
  "$repo_root"|"$repo_root"/*)
    printf 'R7B replay-plan output directory must be outside the repository: %s\n' "$output_dir" >&2
    exit 1
    ;;
esac
if [[ -e "$output_dir" ]]; then
  printf 'R7B replay-plan output directory already exists: %s\n' "$output_dir" >&2
  exit 1
fi
mkdir -m 700 -p "$output_dir"
output_dir="$(cd "$output_dir" && pwd -P)"

status_before="$(git status --porcelain --untracked-files=all --ignored=matching)"
if [[ -n "$status_before" ]]; then
  printf 'Repository is not clean before R7B replay-plan acquisition:\n%s\n' "$status_before" >&2
  exit 1
fi

tool_revision="$(git rev-parse HEAD)"

sha256_file() {
  local path="$1"
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$path" | awk '{print $1}'
    return
  fi
  if command -v shasum >/dev/null 2>&1; then
    shasum -a 256 "$path" | awk '{print $1}'
    return
  fi
  printf 'sha256sum or shasum is required.\n' >&2
  exit 1
}

intake_sha="$(sha256_file "$intake_path")"
receipt_sha="$(sha256_file "$receipt_path")"

go_files=(tools/reference-first-web-target-r7b-replay-plan/*.go)
unformatted="$(gofmt -l "${go_files[@]}")"
if [[ -n "$unformatted" ]]; then
  printf 'Unformatted R7B replay-plan Go files:\n%s\n' "$unformatted" >&2
  exit 1
fi

GOENV=off GOWORK=off GOFLAGS= \
  go test -count=1 ./tools/reference-first-web-target-r7b-replay-plan \
  >"$output_dir/source-tests.log" 2>&1

plan_path="$output_dir/r7b-replay-plan.json"
GOENV=off GOWORK=off GOFLAGS= \
  go run ./tools/reference-first-web-target-r7b-replay-plan \
    -r7b-intake "$intake_path" \
    -r7b-intake-receipt "$receipt_path" \
  >"$plan_path"

if [[ ! -s "$plan_path" ]]; then
  printf 'R7B replay-plan artifact is empty.\n' >&2
  exit 1
fi

plan_sha="$(sha256_file "$plan_path")"

post_revision="$(git rev-parse HEAD)"
status_after="$(git status --porcelain --untracked-files=all --ignored=matching)"
post_clean=true
if [[ "$post_revision" != "$tool_revision" || -n "$status_after" ]]; then
  post_clean=false
fi

cat >"$output_dir/r7b-replay-plan-receipt.json" <<EOF
{
  "evidenceNature": "first-web-target-r7b-replay-plan-acquisition-receipt",
  "replayPlanToolRepositoryRevision": "$tool_revision",
  "repositoryPostRunRevision": "$post_revision",
  "repositoryPostRunClean": $post_clean,
  "sourceQualificationPassed": true,
  "sourceExecutionIntakeSHA256": "$intake_sha",
  "sourceExecutionIntakeReceiptSHA256": "$receipt_sha",
  "replayPlanArtifact": "r7b-replay-plan.json",
  "replayPlanArtifactSHA256": "$plan_sha",
  "replayExecuted": false,
  "repeatabilityObserved": false
}
EOF

if [[ "$post_clean" != true ]]; then
  printf 'Repository changed during R7B replay-plan acquisition.\n%s\n' "$status_after" >&2
  exit 1
fi

printf 'R7B replay plan acquired.\n' >&2
printf '  tool revision: %s\n' "$tool_revision" >&2
printf '  execution-intake SHA-256: %s\n' "$intake_sha" >&2
printf '  intake-receipt SHA-256: %s\n' "$receipt_sha" >&2
printf '  replay-plan SHA-256: %s\n' "$plan_sha" >&2
printf '  output: %s\n' "$output_dir" >&2
cat "$output_dir/r7b-replay-plan-receipt.json"
