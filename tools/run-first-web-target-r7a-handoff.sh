#!/usr/bin/env bash
set -euo pipefail

repo_root="$(git rev-parse --show-toplevel 2>/dev/null || true)"
if [[ -z "$repo_root" || "$PWD" != "$repo_root" ]]; then
  printf 'Run from the TEI repository root.\n' >&2
  exit 1
fi

readiness_path="${TEI_R7A_GATE_C_READINESS:-}"
semantic_path="${TEI_R7A_SEMANTIC_FUNCTIONAL_SUMMARY:-}"
output_dir="${TEI_R7A_OUTPUT_DIR:-}"

if [[ -z "$readiness_path" || -z "$semantic_path" || -z "$output_dir" ]]; then
  cat >&2 <<'EOF'
TEI_R7A_GATE_C_READINESS,
TEI_R7A_SEMANTIC_FUNCTIONAL_SUMMARY,
and TEI_R7A_OUTPUT_DIR are required.
EOF
  exit 1
fi

if [[ ! -f "$readiness_path" ]]; then
  printf 'Gate C readiness artifact not found: %s\n' "$readiness_path" >&2
  exit 1
fi
if [[ ! -f "$semantic_path" ]]; then
  printf 'Semantic / Functional summary not found: %s\n' "$semantic_path" >&2
  exit 1
fi

case "$output_dir" in
  "$repo_root"|"$repo_root"/*)
    printf 'R7A output directory must be outside the repository: %s\n' "$output_dir" >&2
    exit 1
    ;;
esac
if [[ -e "$output_dir" ]]; then
  printf 'R7A output directory already exists: %s\n' "$output_dir" >&2
  exit 1
fi
mkdir -m 700 -p "$output_dir"

status_before="$(git status --porcelain --untracked-files=all --ignored=matching)"
if [[ -n "$status_before" ]]; then
  printf 'Repository is not clean before R7A handoff acquisition:\n%s\n' "$status_before" >&2
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

expected_readiness_sha="b796e38db9a249ef98f2cb600eb354afc70f9adf57d2c83d47116f9968a0ff6c"
expected_semantic_sha="78c4c3b1031a70eb62683233fb5f1794cf7f71fd8b420ffa741d5ebe3f649d6b"
recorded_gate_c_revision="d34edb9273f5f138abcbc23f4dcb000dcb5cdf20"
recorded_target_revision="0ba87a4e0a96f70a198b03ff7652fb00e3826abe"

actual_readiness_sha="$(sha256_file "$readiness_path")"
actual_semantic_sha="$(sha256_file "$semantic_path")"

if [[ "$actual_readiness_sha" != "$expected_readiness_sha" ]]; then
  printf 'Gate C readiness exact-byte SHA-256 mismatch.\nexpected: %s\nactual:   %s\n' \
    "$expected_readiness_sha" "$actual_readiness_sha" >&2
  exit 1
fi
if [[ "$actual_semantic_sha" != "$expected_semantic_sha" ]]; then
  printf 'Semantic / Functional summary exact-byte SHA-256 mismatch.\nexpected: %s\nactual:   %s\n' \
    "$expected_semantic_sha" "$actual_semantic_sha" >&2
  exit 1
fi

go_files=(tools/reference-first-web-target-r7a-handoff/*.go)
unformatted="$(gofmt -l "${go_files[@]}")"
if [[ -n "$unformatted" ]]; then
  printf 'Unformatted R7A handoff Go files:\n%s\n' "$unformatted" >&2
  exit 1
fi

GOENV=off GOWORK=off GOFLAGS= \
  go test -count=1 ./tools/reference-first-web-target-r7a-handoff \
  >"$output_dir/source-tests.log" 2>&1

handoff_path="$output_dir/r7a-replayable-behavior-handoff.json"
GOENV=off GOWORK=off GOFLAGS= \
  go run ./tools/reference-first-web-target-r7a-handoff \
    -gate-c-readiness "$readiness_path" \
    -semantic-functional-summary "$semantic_path" \
  >"$handoff_path"

if [[ ! -s "$handoff_path" ]]; then
  printf 'R7A handoff artifact is empty.\n' >&2
  exit 1
fi

handoff_sha="$(sha256_file "$handoff_path")"

post_revision="$(git rev-parse HEAD)"
status_after="$(git status --porcelain --untracked-files=all --ignored=matching)"
post_clean=true
if [[ "$post_revision" != "$tool_revision" || -n "$status_after" ]]; then
  post_clean=false
fi

cat >"$output_dir/r7a-handoff-receipt.json" <<EOF
{
  "evidenceNature": "first-web-target-r7a-handoff-acquisition-receipt",
  "gateCJudgmentRecordRepositoryRevision": "$recorded_gate_c_revision",
  "recordedTargetBuildRevision": "$recorded_target_revision",
  "handoffToolRepositoryRevision": "$tool_revision",
  "repositoryPostRunRevision": "$post_revision",
  "repositoryPostRunClean": $post_clean,
  "sourceQualificationPassed": true,
  "gateCReadinessArtifactSHA256": "$actual_readiness_sha",
  "semanticFunctionalSummarySHA256": "$actual_semantic_sha",
  "handoffArtifact": "r7a-replayable-behavior-handoff.json",
  "handoffArtifactSHA256": "$handoff_sha"
}
EOF

if [[ "$post_clean" != true ]]; then
  printf 'Repository changed during R7A handoff acquisition.\n%s\n' "$status_after" >&2
  exit 1
fi

printf 'R7A handoff acquired.\n' >&2
printf '  tool revision: %s\n' "$tool_revision" >&2
printf '  handoff SHA-256: %s\n' "$handoff_sha" >&2
printf '  output: %s\n' "$output_dir" >&2
cat "$output_dir/r7a-handoff-receipt.json"
