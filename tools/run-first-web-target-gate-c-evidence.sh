#!/usr/bin/env bash
set -euo pipefail

repo_root="$(git rev-parse --show-toplevel 2>/dev/null || true)"
if [[ -z "$repo_root" || "$PWD" != "$repo_root" ]]; then
  printf 'Run from the TEI repository root.\n' >&2
  exit 1
fi

baseline_id="${TEI_GATE_C_BASELINE_ID:-B}"
baseline_revision="${TEI_GATE_C_BASELINE_REVISION:-e5231839a5dcc62cce1181a2731fcb7c70d16736}"
target_revision="$(git rev-parse HEAD)"
run_id="${TEI_GATE_C_RUN_ID:-first-web-target-${target_revision:0:12}}"
port="${TEI_GATE_C_PORT:-$((18080 + ($$ % 1000)))}"
listen="127.0.0.1:${port}"
base_url="http://127.0.0.1:${port}"

if [[ ! "$run_id" =~ ^[A-Za-z0-9._-]+$ ]]; then
  printf 'TEI_GATE_C_RUN_ID must contain only A-Z, a-z, 0-9, dot, underscore, or hyphen.\n' >&2
  exit 1
fi

if ! command -v curl >/dev/null 2>&1; then
  printf 'curl is required for live Target Evidence acquisition.\n' >&2
  exit 1
fi

status_before="$(git status --porcelain --untracked-files=all --ignored=matching)"
if [[ -n "$status_before" ]]; then
  printf 'Repository is not clean before Gate C acquisition:\n%s\n' "$status_before" >&2
  exit 1
fi

nonstandard_index="$(git ls-files -v | awk 'substr($0,1,1)!="H"{print}')"
if [[ -n "$nonstandard_index" ]]; then
  printf 'Repository has non-standard tracked index state:\n%s\n' "$nonstandard_index" >&2
  exit 1
fi

if [[ -n "${TEI_GATE_C_ARTIFACT_DIR:-}" ]]; then
  artifact_dir="$TEI_GATE_C_ARTIFACT_DIR"
  if [[ -e "$artifact_dir" ]]; then
    printf 'Artifact directory already exists: %s\n' "$artifact_dir" >&2
    exit 1
  fi
  mkdir -m 700 -p "$artifact_dir"
else
  artifact_dir="$(mktemp -d "${TMPDIR:-/tmp}/tei-gate-c-first-web.XXXXXX")"
fi
artifact_dir="$(cd "$artifact_dir" && pwd -P)"

case "$artifact_dir" in
  "$repo_root"|"$repo_root"/*)
    printf 'Artifact directory must be outside the repository: %s\n' "$artifact_dir" >&2
    exit 1
    ;;
esac

printf 'Gate C raw artifact directory: %s\n' "$artifact_dir" >&2

server_pid=""
cleanup() {
  if [[ -n "$server_pid" ]]; then
    kill "$server_pid" >/dev/null 2>&1 || true
    wait "$server_pid" >/dev/null 2>&1 || true
    server_pid=""
  fi
}
trap cleanup EXIT

go_files=(
  tools/reference-first-web-target/*.go
  tools/reference-first-web-target-evidence-audit/*.go
  tools/reference-first-web-target-visual-evidence-audit/*.go
  tools/reference-first-web-target-architecture-evidence/*.go
)
unformatted="$(gofmt -l "${go_files[@]}")"
if [[ -n "$unformatted" ]]; then
  printf 'Unformatted First Web Target Go files:\n%s\n' "$unformatted" >&2
  exit 1
fi

GOENV=off GOWORK=off GOFLAGS= \
  go test -count=1 \
    ./tools/reference-first-web-target \
    ./tools/reference-first-web-target-evidence-audit \
    ./tools/reference-first-web-target-visual-evidence-audit \
    ./tools/reference-first-web-target-architecture-evidence \
  >"$artifact_dir/source-tests.log" 2>&1

GOENV=off GOWORK=off GOFLAGS= \
  go run ./tools/reference-first-web-target \
    -workspace reference/workspace \
    -application order_entry \
    -baseline-id "$baseline_id" \
    -baseline-revision "$baseline_revision" \
    -target-revision "$target_revision" \
    -legacy-users reference/legacy-auth/users.json \
    -legacy-authorization reference/legacy-auth/authorization.json \
    -listen "$listen" \
  >"$artifact_dir/server.log" 2>&1 &
server_pid="$!"

healthy=false
for _ in $(seq 1 100); do
  if curl --silent --show-error --fail "$base_url/healthz" >"$artifact_dir/health.txt" 2>/dev/null; then
    healthy=true
    break
  fi
  if ! kill -0 "$server_pid" >/dev/null 2>&1; then
    break
  fi
  sleep 0.1
done
if [[ "$healthy" != true ]]; then
  printf 'First Web Target did not become healthy. See %s/server.log\n' "$artifact_dir" >&2
  exit 1
fi

curl --silent --show-error --fail \
  "$base_url/manifest.json" \
  >"$artifact_dir/manifest.json"

curl --silent --show-error --fail \
  "$base_url/scenario/read?from=2026-09-18T08%3A30%3A00Z&row=0" \
  >"$artifact_dir/read.json"

printf '%s' '{"loginId":"legacy-order-editor","credential":"editor-secret","from":"2026-09-18T08:30:00Z","row":0}' | \
  curl --silent --show-error --fail \
    -H 'content-type: application/json' \
    --data-binary @- \
    "$base_url/scenario/submit" \
    >"$artifact_dir/submit-success.json"

printf '%s' '{"loginId":"legacy-order-editor","credential":"wrong-secret","from":"2026-09-18T08:30:00Z","row":0}' | \
  curl --silent --show-error --fail \
    -H 'content-type: application/json' \
    --data-binary @- \
    "$base_url/scenario/submit" \
    >"$artifact_dir/submit-authentication-denied.json"

printf '%s' '{"loginId":"legacy-order-viewer","credential":"viewer-secret","from":"2026-09-18T08:30:00Z","row":0}' | \
  curl --silent --show-error --fail \
    -H 'content-type: application/json' \
    --data-binary @- \
    "$base_url/scenario/submit" \
    >"$artifact_dir/submit-authorization-denied.json"

printf '%s' '{"loginId":"legacy-order-editor","credential":"editor-secret","from":"2026-09-18T08:00:00Z","row":1}' | \
  curl --silent --show-error --fail \
    -H 'content-type: application/json' \
    --data-binary @- \
    "$base_url/scenario/submit" \
    >"$artifact_dir/submit-validation-not-satisfied.json"

execution_status="$(
  printf '%s' '{"loginId":"legacy-order-editor","credential":"editor-secret","from":"2026-09-18T07:00:00Z","row":2}' | \
    curl --silent --show-error \
      -o "$artifact_dir/submit-execution-error.json" \
      -w '%{http_code}' \
      -H 'content-type: application/json' \
      --data-binary @- \
      "$base_url/scenario/submit"
)"
if [[ "$execution_status" != "422" ]]; then
  printf 'Expected Command Execution Error HTTP 422, got %s.\n' "$execution_status" >&2
  exit 1
fi

curl --silent --show-error --fail \
  "$base_url/custom-submit" \
  >"$artifact_dir/custom-submit-get.html"

printf '%s' 'loginId=legacy-order-editor&credential=editor-secret&from=2026-09-18T08%3A30%3A00Z&row=0' | \
  curl --silent --show-error --fail \
    -H 'content-type: application/x-www-form-urlencoded' \
    --data-binary @- \
    "$base_url/custom-submit" \
    >"$artifact_dir/custom-submit-success.html"

cleanup

post_revision="$(git rev-parse HEAD)"
status_after="$(git status --porcelain --untracked-files=all --ignored=matching)"
post_clean=true
if [[ -n "$status_after" || "$post_revision" != "$target_revision" ]]; then
  post_clean=false
fi

cat >"$artifact_dir/run-metadata.json" <<EOF
{
  "runIdentity": "$run_id",
  "targetBuildRevision": "$target_revision",
  "repositoryPreRunRevision": "$target_revision",
  "repositoryPreRunClean": true,
  "repositoryPostRunRevision": "$post_revision",
  "repositoryPostRunClean": $post_clean,
  "sourceQualificationPassed": true
}
EOF

if [[ "$post_clean" != true ]]; then
  printf 'Repository changed during Gate C acquisition.\n%s\n' "$status_after" >&2
  exit 1
fi

GOENV=off GOWORK=off GOFLAGS= \
  go run ./tools/reference-first-web-target-evidence-audit \
    -artifact-dir "$artifact_dir" \
    -run-id "$run_id" \
    -baseline-id "$baseline_id" \
    -baseline-revision "$baseline_revision" \
    -target-revision "$target_revision" \
  >"$artifact_dir/semantic-functional-summary.json"

GOENV=off GOWORK=off GOFLAGS= \
  go run ./tools/reference-first-web-target-architecture-evidence \
    -baseline-id "$baseline_id" \
    -baseline-revision "$baseline_revision" \
    -target-revision "$target_revision" \
  >"$artifact_dir/architecture-conformance.json"

printf 'Semantic / Functional Gate C raw Evidence acquired at T=%s\n' "$target_revision" >&2
printf 'Visual Fidelity remains pending by contract.\n' >&2
cat "$artifact_dir/semantic-functional-summary.json"
