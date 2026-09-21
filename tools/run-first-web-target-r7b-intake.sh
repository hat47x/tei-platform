#!/usr/bin/env bash
set -euo pipefail

repo_root="$(git rev-parse --show-toplevel 2>/dev/null || true)"
if [[ -z "$repo_root" || "$PWD" != "$repo_root" ]]; then
  printf 'Run from the TEI repository root.\n' >&2
  exit 1
fi

handoff_path="${TEI_R7B_R7A_HANDOFF:-}"
receipt_path="${TEI_R7B_R7A_RECEIPT:-}"
output_dir="${TEI_R7B_OUTPUT_DIR:-}"

if [[ -z "$handoff_path" || -z "$receipt_path" || -z "$output_dir" ]]; then
  cat >&2 <<'EOF'
TEI_R7B_R7A_HANDOFF,
TEI_R7B_R7A_RECEIPT,
and TEI_R7B_OUTPUT_DIR are required.
EOF
  exit 1
fi

if [[ ! -f "$handoff_path" ]]; then
  printf 'R7A handoff artifact not found: %s\n' "$handoff_path" >&2
  exit 1
fi
if [[ ! -f "$receipt_path" ]]; then
  printf 'R7A acquisition receipt not found: %s\n' "$receipt_path" >&2
  exit 1
fi

canonical_file() {
  local path="$1"
  local dir
  dir="$(cd "$(dirname "$path")" && pwd -P)"
  printf '%s/%s\n' "$dir" "$(basename "$path")"
}

handoff_path="$(canonical_file "$handoff_path")"
receipt_path="$(canonical_file "$receipt_path")"

case "$handoff_path" in
  "$repo_root"|"$repo_root"/*)
    printf 'R7A handoff artifact must remain outside the repository: %s\n' "$handoff_path" >&2
    exit 1
    ;;
esac
case "$receipt_path" in
  "$repo_root"|"$repo_root"/*)
    printf 'R7A acquisition receipt must remain outside the repository: %s\n' "$receipt_path" >&2
    exit 1
    ;;
esac

if [[ "$(basename "$handoff_path")" != "r7a-replayable-behavior-handoff.json" ]]; then
  printf 'Unexpected R7A handoff filename: %s\n' "$(basename "$handoff_path")" >&2
  exit 1
fi
if [[ "$(basename "$receipt_path")" != "r7a-handoff-receipt.json" ]]; then
  printf 'Unexpected R7A receipt filename: %s\n' "$(basename "$receipt_path")" >&2
  exit 1
fi

case "$output_dir" in
  "$repo_root"|"$repo_root"/*)
    printf 'R7B output directory must be outside the repository: %s\n' "$output_dir" >&2
    exit 1
    ;;
esac
if [[ -e "$output_dir" ]]; then
  printf 'R7B output directory already exists: %s\n' "$output_dir" >&2
  exit 1
fi
mkdir -m 700 -p "$output_dir"
output_dir="$(cd "$output_dir" && pwd -P)"

status_before="$(git status --porcelain --untracked-files=all --ignored=matching)"
if [[ -n "$status_before" ]]; then
  printf 'Repository is not clean before R7B intake acquisition:\n%s\n' "$status_before" >&2
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

handoff_sha="$(sha256_file "$handoff_path")"
receipt_sha="$(sha256_file "$receipt_path")"

go_files=(tools/reference-first-web-target-r7b-intake/*.go)
unformatted="$(gofmt -l "${go_files[@]}")"
if [[ -n "$unformatted" ]]; then
  printf 'Unformatted R7B intake Go files:\n%s\n' "$unformatted" >&2
  exit 1
fi

GOENV=off GOWORK=off GOFLAGS= \
  go test -count=1 ./tools/reference-first-web-target-r7b-intake \
  >"$output_dir/source-tests.log" 2>&1

intake_path="$output_dir/r7b-execution-intake.json"
GOENV=off GOWORK=off GOFLAGS= \
  go run ./tools/reference-first-web-target-r7b-intake \
    -r7a-handoff "$handoff_path" \
    -r7a-receipt "$receipt_path" \
  >"$intake_path"

if [[ ! -s "$intake_path" ]]; then
  printf 'R7B execution intake artifact is empty.\n' >&2
  exit 1
fi

intake_sha="$(sha256_file "$intake_path")"

post_revision="$(git rev-parse HEAD)"
status_after="$(git status --porcelain --untracked-files=all --ignored=matching)"
post_clean=true
if [[ "$post_revision" != "$tool_revision" || -n "$status_after" ]]; then
  post_clean=false
fi

cat >"$output_dir/r7b-intake-receipt.json" <<EOF
{
  "evidenceNature": "first-web-target-r7b-intake-acquisition-receipt",
  "intakeToolRepositoryRevision": "$tool_revision",
  "repositoryPostRunRevision": "$post_revision",
  "repositoryPostRunClean": $post_clean,
  "sourceQualificationPassed": true,
  "sourceR7AHandoffSHA256": "$handoff_sha",
  "sourceR7AAcquisitionReceiptSHA256": "$receipt_sha",
  "executionIntakeArtifact": "r7b-execution-intake.json",
  "executionIntakeArtifactSHA256": "$intake_sha",
  "replayExecuted": false,
  "repeatabilityObserved": false
}
EOF

if [[ "$post_clean" != true ]]; then
  printf 'Repository changed during R7B intake acquisition.\n%s\n' "$status_after" >&2
  exit 1
fi

printf 'R7B execution intake acquired.\n' >&2
printf '  tool revision: %s\n' "$tool_revision" >&2
printf '  R7A handoff SHA-256: %s\n' "$handoff_sha" >&2
printf '  R7A receipt SHA-256: %s\n' "$receipt_sha" >&2
printf '  intake SHA-256: %s\n' "$intake_sha" >&2
printf '  output: %s\n' "$output_dir" >&2
cat "$output_dir/r7b-intake-receipt.json"
