#!/usr/bin/env bash
set -euo pipefail

repo_root="$(git rev-parse --show-toplevel 2>/dev/null || true)"
if [[ -z "$repo_root" || "$PWD" != "$repo_root" ]]; then
  printf 'Run from the TEI repository root.\n' >&2
  exit 1
fi

plan_path="${TEI_R7B_REPLAY_PLAN:-}"
plan_receipt_path="${TEI_R7B_REPLAY_PLAN_RECEIPT:-}"
origin_archive="${TEI_R7B_GATE_C_EVIDENCE_ARCHIVE:-}"
output_dir="${TEI_R7B_REPLAY_OUTPUT_DIR:-}"

if [[ -z "$plan_path" || -z "$plan_receipt_path" || -z "$origin_archive" || -z "$output_dir" ]]; then
  cat >&2 <<'EOF'
TEI_R7B_REPLAY_PLAN,
TEI_R7B_REPLAY_PLAN_RECEIPT,
TEI_R7B_GATE_C_EVIDENCE_ARCHIVE,
and TEI_R7B_REPLAY_OUTPUT_DIR are required.
EOF
  exit 1
fi

for command in git go curl tar sha256sum grep date mktemp awk cp rm mkdir dirname basename sleep cat; do
  if ! command -v "$command" >/dev/null 2>&1; then
    printf '%s is required for R7B replay execution.\n' "$command" >&2
    exit 1
  fi
done

canonical_file() {
  local path="$1"
  local dir
  dir="$(cd "$(dirname "$path")" && pwd -P)"
  printf '%s/%s\n' "$dir" "$(basename "$path")"
}

canonical_dir() {
  (cd "$1" && pwd -P)
}

sha256_file() {
  sha256sum "$1" | awk '{print $1}'
}

for path_name in plan_path plan_receipt_path origin_archive; do
  path="${!path_name}"
  if [[ ! -f "$path" ]]; then
    printf 'Required input not found: %s\n' "$path" >&2
    exit 1
  fi
  printf -v "$path_name" '%s' "$(canonical_file "$path")"
done

for input in "$plan_path" "$plan_receipt_path" "$origin_archive"; do
  case "$input" in
    "$repo_root"|"$repo_root"/*)
      printf 'R7B replay input must remain outside the repository: %s\n' "$input" >&2
      exit 1
      ;;
  esac
done

output_parent="$(dirname "$output_dir")"
output_base="$(basename "$output_dir")"
if [[ ! -d "$output_parent" || -z "$output_base" || "$output_base" == "." || "$output_base" == ".." ]]; then
  printf 'R7B replay output parent must already exist and the output basename must be ordinary: %s\n' "$output_dir" >&2
  exit 1
fi
output_dir="$(cd "$output_parent" && pwd -P)/$output_base"
case "$output_dir" in
  "$repo_root"|"$repo_root"/*)
    printf 'R7B replay output directory must remain outside the repository: %s\n' "$output_dir" >&2
    exit 1
    ;;
esac
if [[ -e "$output_dir" ]]; then
  printf 'R7B replay output directory already exists: %s\n' "$output_dir" >&2
  exit 1
fi
mkdir -m 700 -p "$output_dir/source"
output_dir="$(canonical_dir "$output_dir")"

status_before="$(git status --porcelain --untracked-files=all --ignored=matching)"
if [[ -n "$status_before" ]]; then
  printf 'Repository is not clean before R7B replay execution:\n%s\n' "$status_before" >&2
  exit 1
fi
nonstandard_index="$(git ls-files -v | awk 'substr($0,1,1)!="H"{print}')"
if [[ -n "$nonstandard_index" ]]; then
  printf 'Repository has non-standard tracked index state:\n%s\n' "$nonstandard_index" >&2
  exit 1
fi

tool_revision="$(git rev-parse HEAD)"
go_version="$(GOENV=off GOWORK=off GOFLAGS= go version)"
go_os="$(GOENV=off GOWORK=off GOFLAGS= go env GOOS)"
go_arch="$(GOENV=off GOWORK=off GOFLAGS= go env GOARCH)"

tools/repository-health-check.sh >"$output_dir/source/repository-health-check.log" 2>&1
post_health_revision="$(git rev-parse HEAD)"
post_health_status="$(git status --porcelain --untracked-files=all --ignored=matching)"
if [[ "$post_health_revision" != "$tool_revision" || -n "$post_health_status" ]]; then
  printf 'Repository changed during R7B source qualification:\n%s\n' "$post_health_status" >&2
  exit 1
fi

cp "$plan_path" "$output_dir/source/r7b-replay-plan.json"
cp "$plan_receipt_path" "$output_dir/source/r7b-replay-plan-receipt.json"
plan_copy="$output_dir/source/r7b-replay-plan.json"
receipt_copy="$output_dir/source/r7b-replay-plan-receipt.json"

plan_sha="$(sha256_file "$plan_path")"
receipt_sha="$(sha256_file "$plan_receipt_path")"
if [[ "$(sha256_file "$plan_copy")" != "$plan_sha" || "$(sha256_file "$receipt_copy")" != "$receipt_sha" ]]; then
  printf 'Copied replay plan / receipt bytes changed.\n' >&2
  exit 1
fi

expected_archive_sha="de5f587a038c56c0f4b5776953b2f6729e6521d4eda67ebcc246567638fa12a3"
archive_sha="$(sha256_file "$origin_archive")"
if [[ "$archive_sha" != "$expected_archive_sha" ]]; then
  printf 'Gate C Evidence archive SHA-256=%s, want %s\n' "$archive_sha" "$expected_archive_sha" >&2
  exit 1
fi

scratch="$(mktemp -d "${TMPDIR:-/tmp}/tei-r7b-replay.XXXXXX")"
origin_extract="$scratch/origin"
target_checkout="$scratch/target-T"
server_pid=""

cleanup_server() {
  if [[ -n "$server_pid" ]]; then
    kill "$server_pid" >/dev/null 2>&1 || true
    wait "$server_pid" >/dev/null 2>&1 || true
    server_pid=""
  fi
}

cleanup() {
  cleanup_server
  if [[ -d "$target_checkout" ]] && git worktree list --porcelain | grep -Fqx "worktree $target_checkout"; then
    git worktree remove --force "$target_checkout" >/dev/null 2>&1 || true
  fi
  rm -rf "$scratch"
}
trap cleanup EXIT

mkdir -m 700 -p "$origin_extract"
tar -xzf "$origin_archive" -C "$origin_extract"
origin_dir="$origin_extract/gate-c-T-0ba87a4e"
manifest="$origin_dir/MANIFEST.sha256"
if [[ ! -d "$origin_dir" || ! -f "$manifest" ]]; then
  printf 'Gate C Evidence archive does not contain the expected root / MANIFEST.sha256.\n' >&2
  exit 1
fi
(
  cd "$origin_dir"
  sha256sum -c MANIFEST.sha256
) >"$output_dir/source/origin-manifest-check.log" 2>&1
manifest_sha="$(sha256_file "$manifest")"

launch_line="$(
  GOENV=off GOWORK=off GOFLAGS= \
    go run ./tools/reference-first-web-target-r7b-replay-audit \
      -replay-plan "$plan_copy" \
      -replay-plan-receipt "$receipt_copy" \
      -preflight-only \
      -origin-artifact-dir "$origin_dir" \
      -emit-launch-provenance-tsv
)"
printf '%s\n' "$launch_line" >"$output_dir/source/validated-launch-provenance.tsv"

IFS=$'\t' read -r baseline_id baseline_revision target_revision target_adapter target_host extra <<<"$launch_line"
if [[ -z "$baseline_id" || -z "$baseline_revision" || -z "$target_revision" || -z "$target_adapter" || -z "$target_host" || -n "${extra:-}" ]]; then
  printf 'V2 preflight did not emit exactly five launch-provenance fields.\n' >&2
  exit 1
fi

if ! git cat-file -e "${target_revision}^{commit}" 2>/dev/null; then
  printf 'Replay Target revision is unavailable locally: %s\n' "$target_revision" >&2
  exit 1
fi

git worktree add --quiet --detach "$target_checkout" "$target_revision"
if [[ "$(git -C "$target_checkout" rev-parse HEAD)" != "$target_revision" ]]; then
  printf 'Isolated Target checkout is not exact replay Target T.\n' >&2
  exit 1
fi
target_status="$(git -C "$target_checkout" status --porcelain --untracked-files=all --ignored=matching)"
if [[ -n "$target_status" ]]; then
  printf 'Isolated Target checkout is not clean:\n%s\n' "$target_status" >&2
  exit 1
fi
target_nonstandard="$(git -C "$target_checkout" ls-files -v | awk 'substr($0,1,1)!="H"{print}')"
if [[ -n "$target_nonstandard" ]]; then
  printf 'Isolated Target checkout has non-standard tracked index state:\n%s\n' "$target_nonstandard" >&2
  exit 1
fi

for required_path in \
  reference/workspace \
  reference/legacy-auth/users.json \
  reference/legacy-auth/authorization.json \
  tools/reference-first-web-target; do
  if [[ ! -e "$target_checkout/$required_path" ]]; then
    printf 'Exact-T checkout is missing required path: %s\n' "$required_path" >&2
    exit 1
  fi
done

(
  cd "$target_checkout"
  GOENV=off GOWORK=off GOFLAGS= go test -count=1 ./tools/reference-first-web-target
) >"$output_dir/source/target-source-tests.log" 2>&1

target_binary="$scratch/reference-first-web-target"
(
  cd "$target_checkout"
  GOENV=off GOWORK=off GOFLAGS= go build -o "$target_binary" ./tools/reference-first-web-target
)

acquisition_label="${TEI_R7B_REPLAY_ACQUISITION_ID:-auto}"
if [[ ! "$acquisition_label" =~ ^[A-Za-z0-9._-]+$ ]]; then
  printf 'TEI_R7B_REPLAY_ACQUISITION_ID contains unsupported characters.\n' >&2
  exit 1
fi
acquisition_nonce="$(date -u +%Y%m%dT%H%M%SZ)-${BASHPID}"
run_prefix="r7b-${target_revision:0:12}-${acquisition_label}-${acquisition_nonce}"
base_port="${TEI_R7B_REPLAY_BASE_PORT:-$((19080 + ($$ % 700)))}"
if [[ ! "$base_port" =~ ^[0-9]+$ ]] || ((base_port < 1024 || base_port > 65534)); then
  printf 'TEI_R7B_REPLAY_BASE_PORT must be an integer from 1024 through 65534.\n' >&2
  exit 1
fi

run_one() {
  local ordinal="$1"
  local port="$2"
  local label
  printf -v label '%02d' "$ordinal"
  local run_id="${run_prefix}-run-${label}"
  local process_id="${run_prefix}-process-${label}"
  local run_dir="$output_dir/run-${label}"
  local listen="127.0.0.1:${port}"
  local base_url="http://127.0.0.1:${port}"

  mkdir -m 700 -p "$run_dir"

  local pre_revision pre_status
  pre_revision="$(git -C "$target_checkout" rev-parse HEAD)"
  pre_status="$(git -C "$target_checkout" status --porcelain --untracked-files=all --ignored=matching)"
  if [[ "$pre_revision" != "$target_revision" || -n "$pre_status" ]]; then
    printf 'Target checkout is not clean exact T before %s.\n' "$run_id" >&2
    exit 1
  fi

  if curl --silent --fail --max-time 0.2 "$base_url/healthz" >/dev/null 2>&1; then
    printf 'Replay port already serves a health endpoint before %s: %s\n' "$run_id" "$listen" >&2
    exit 1
  fi

  (
    cd "$target_checkout"
    exec "$target_binary" \
      -workspace reference/workspace \
      -application order_entry \
      -baseline-id "$baseline_id" \
      -baseline-revision "$baseline_revision" \
      -target-revision "$target_revision" \
      -legacy-users reference/legacy-auth/users.json \
      -legacy-authorization reference/legacy-auth/authorization.json \
      -listen "$listen"
  ) >"$run_dir/server.log" 2>&1 &
  server_pid="$!"

  local healthy=false
  local attempt
  for ((attempt = 0; attempt < 100; attempt++)); do
    if ! kill -0 "$server_pid" >/dev/null 2>&1; then
      break
    fi
    if curl --silent --show-error --fail "$base_url/healthz" >"$run_dir/health.txt" 2>/dev/null; then
      healthy=true
      break
    fi
    sleep 0.1
  done
  if [[ "$healthy" != true ]]; then
    printf 'First Web Target did not become healthy for %s.\n' "$run_id" >&2
    exit 1
  fi

  local read_status success_status authn_status authz_status validation_status execution_status

  read_status="$(
    curl --silent --show-error \
      -o "$run_dir/read.json" \
      -w '%{http_code}' \
      "$base_url/scenario/read?from=2026-09-18T08%3A30%3A00Z&row=0"
  )"
  success_status="$(
    printf '%s' '{"loginId":"legacy-order-editor","credential":"editor-secret","from":"2026-09-18T08:30:00Z","row":0}' |
      curl --silent --show-error \
        -o "$run_dir/submit-success.json" \
        -w '%{http_code}' \
        -H 'content-type: application/json' \
        --data-binary @- \
        "$base_url/scenario/submit"
  )"
  authn_status="$(
    printf '%s' '{"loginId":"legacy-order-editor","credential":"wrong-secret","from":"2026-09-18T08:30:00Z","row":0}' |
      curl --silent --show-error \
        -o "$run_dir/submit-authentication-denied.json" \
        -w '%{http_code}' \
        -H 'content-type: application/json' \
        --data-binary @- \
        "$base_url/scenario/submit"
  )"
  authz_status="$(
    printf '%s' '{"loginId":"legacy-order-viewer","credential":"viewer-secret","from":"2026-09-18T08:30:00Z","row":0}' |
      curl --silent --show-error \
        -o "$run_dir/submit-authorization-denied.json" \
        -w '%{http_code}' \
        -H 'content-type: application/json' \
        --data-binary @- \
        "$base_url/scenario/submit"
  )"
  validation_status="$(
    printf '%s' '{"loginId":"legacy-order-editor","credential":"editor-secret","from":"2026-09-18T08:00:00Z","row":1}' |
      curl --silent --show-error \
        -o "$run_dir/submit-validation-not-satisfied.json" \
        -w '%{http_code}' \
        -H 'content-type: application/json' \
        --data-binary @- \
        "$base_url/scenario/submit"
  )"
  execution_status="$(
    printf '%s' '{"loginId":"legacy-order-editor","credential":"editor-secret","from":"2026-09-18T07:00:00Z","row":2}' |
      curl --silent --show-error \
        -o "$run_dir/submit-execution-error.json" \
        -w '%{http_code}' \
        -H 'content-type: application/json' \
        --data-binary @- \
        "$base_url/scenario/submit"
  )"

  cleanup_server

  cat >"$run_dir/run-metadata.json" <<EOF
{
  "evidenceNature": "first-web-target-r7b-run-metadata",
  "runIdentity": "$run_id",
  "sourceReplayPlanSHA256": "$plan_sha",
  "sourceReplayPlanReceiptSHA256": "$receipt_sha",
  "replayTargetBuildRevision": "$target_revision",
  "targetProcessInstanceIdentity": "$process_id",
  "targetListenScope": "loopback",
  "captures": [
    {"behaviorId":"first-web/read-only","observedHTTPStatus":$read_status,"rawArtifactFile":"read.json"},
    {"behaviorId":"first-web/command-success","observedHTTPStatus":$success_status,"rawArtifactFile":"submit-success.json"},
    {"behaviorId":"first-web/authentication-denied","observedHTTPStatus":$authn_status,"rawArtifactFile":"submit-authentication-denied.json"},
    {"behaviorId":"first-web/authorization-denied","observedHTTPStatus":$authz_status,"rawArtifactFile":"submit-authorization-denied.json"},
    {"behaviorId":"first-web/validation-not-satisfied","observedHTTPStatus":$validation_status,"rawArtifactFile":"submit-validation-not-satisfied.json"},
    {"behaviorId":"first-web/execution-error","observedHTTPStatus":$execution_status,"rawArtifactFile":"submit-execution-error.json"}
  ]
}
EOF

  GOENV=off GOWORK=off GOFLAGS= \
    go run ./tools/reference-first-web-target-r7b-replay-audit \
      -replay-plan "$plan_copy" \
      -replay-plan-receipt "$receipt_copy" \
      -run-metadata "$run_dir/run-metadata.json" \
      -raw-dir "$run_dir" \
    >"$run_dir/run-observation.json"

  (
    cd "$run_dir"
    sha256sum \
      read.json \
      submit-success.json \
      submit-authentication-denied.json \
      submit-authorization-denied.json \
      submit-validation-not-satisfied.json \
      submit-execution-error.json
  ) >"$run_dir/raw-artifacts.sha256"

  local post_revision post_status
  post_revision="$(git -C "$target_checkout" rev-parse HEAD)"
  post_status="$(git -C "$target_checkout" status --porcelain --untracked-files=all --ignored=matching)"
  if [[ "$post_revision" != "$target_revision" || -n "$post_status" ]]; then
    printf 'Target checkout changed during %s:\n%s\n' "$run_id" "$post_status" >&2
    exit 1
  fi
}

run_one 1 "$base_port"
run_one 2 "$((base_port + 1))"

GOENV=off GOWORK=off GOFLAGS= \
  go run ./tools/reference-first-web-target-r7b-repeatability \
    -replay-plan "$plan_copy" \
    -replay-plan-receipt "$receipt_copy" \
    -run-observation "$output_dir/run-01/run-observation.json" \
    -run-observation "$output_dir/run-02/run-observation.json" \
  >"$output_dir/repeatability-summary.json"

git worktree remove --force "$target_checkout"
target_checkout=""

post_revision="$(git rev-parse HEAD)"
status_after="$(git status --porcelain --untracked-files=all --ignored=matching)"
if [[ "$post_revision" != "$tool_revision" || -n "$status_after" ]]; then
  printf 'Repository changed during R7B replay execution:\n%s\n' "$status_after" >&2
  exit 1
fi

repository_health_sha="$(sha256_file "$output_dir/source/repository-health-check.log")"
target_source_tests_sha="$(sha256_file "$output_dir/source/target-source-tests.log")"
origin_manifest_check_sha="$(sha256_file "$output_dir/source/origin-manifest-check.log")"
launch_sha="$(sha256_file "$output_dir/source/validated-launch-provenance.tsv")"
run_01_sha="$(sha256_file "$output_dir/run-01/run-observation.json")"
run_02_sha="$(sha256_file "$output_dir/run-02/run-observation.json")"
run_01_raw_manifest_sha="$(sha256_file "$output_dir/run-01/raw-artifacts.sha256")"
run_02_raw_manifest_sha="$(sha256_file "$output_dir/run-02/raw-artifacts.sha256")"
summary_sha="$(sha256_file "$output_dir/repeatability-summary.json")"

scan_retained_output() {
  local forbidden
  for forbidden in \
    editor-secret \
    viewer-secret \
    wrong-secret \
    locked-secret \
    disabled-secret \
    'credential-ref://legacy/'; do
    if grep -R -F -- "$forbidden" "$output_dir" >/dev/null 2>&1; then
      printf 'Credential material detected in retained R7B replay output: %s\n' "$forbidden" >&2
      return 1
    fi
  done
}

scan_retained_output
printf 'retained output credential scan: PASS\n' >"$output_dir/source/credential-scan.txt"
credential_scan_sha="$(sha256_file "$output_dir/source/credential-scan.txt")"

cat >"$output_dir/r7b-replay-receipt.json" <<EOF
{
  "evidenceNature": "first-web-target-r7b-replay-execution-receipt",
  "replayRunnerRepositoryRevision": "$tool_revision",
  "repositoryPostRunRevision": "$post_revision",
  "repositoryPostRunClean": true,
  "sourceQualificationPassed": true,
  "goVersion": "$go_version",
  "goOS": "$go_os",
  "goArch": "$go_arch",
  "replayPlanArtifactSHA256": "$plan_sha",
  "replayPlanReceiptSHA256": "$receipt_sha",
  "repositoryHealthCheckLogSHA256": "$repository_health_sha",
  "targetSourceTestsLogSHA256": "$target_source_tests_sha",
  "originGateCEvidenceArchiveSHA256": "$archive_sha",
  "originManifestSHA256": "$manifest_sha",
  "originManifestCheckLogSHA256": "$origin_manifest_check_sha",
  "originManifestVerified": true,
  "validatedLaunchProvenanceSHA256": "$launch_sha",
  "handoverBaselineId": "$baseline_id",
  "baselineRepositoryRevision": "$baseline_revision",
  "replayTargetBuildRevision": "$target_revision",
  "targetAdapter": "$target_adapter",
  "host": "$target_host",
  "run01ObservationSHA256": "$run_01_sha",
  "run02ObservationSHA256": "$run_02_sha",
  "run01RawArtifactsManifestSHA256": "$run_01_raw_manifest_sha",
  "run02RawArtifactsManifestSHA256": "$run_02_raw_manifest_sha",
  "repeatabilitySummarySHA256": "$summary_sha",
  "credentialScanSHA256": "$credential_scan_sha",
  "credentialScanPassed": true,
  "runIdentities": [
    "${run_prefix}-run-01",
    "${run_prefix}-run-02"
  ],
  "replayExecuted": true,
  "repeatabilityObserved": true,
  "coverageClass": "behavior-unit",
  "deepWorkflowCovered": false
}
EOF

scan_retained_output

printf 'R7B exact-T replay completed with two distinct validated runs.\n' >&2
printf '  runner revision: %s\n' "$tool_revision" >&2
printf '  replay Target T: %s\n' "$target_revision" >&2
printf '  output: %s\n' "$output_dir" >&2
