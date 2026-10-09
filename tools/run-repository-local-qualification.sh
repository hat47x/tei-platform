#!/usr/bin/env bash
set -euo pipefail

repo_root="$(git rev-parse --show-toplevel 2>/dev/null || true)"
if [[ -z "$repo_root" || "$PWD" != "$repo_root" ]]; then
  printf 'Run from the TEI repository root.\n' >&2
  exit 2
fi

remote="origin"
expected_head=""
expected_main=""
receipt_path=""

usage() {
  cat <<'EOF'
usage: bash tools/run-repository-local-qualification.sh \
  --expected-head SHA \
  --expected-main SHA \
  [--remote NAME] \
  [--receipt PATH]

Run the full repository health gate against one exact candidate HEAD and one
already-known current main SHA. The script refreshes remote refs before and
after the health run and fails if HEAD, main, tracked/untracked/ignored state,
or tracked index state changes.

The optional receipt must be outside the repository. It is published only
after the qualification has passed.
EOF
}

while [[ "$#" -gt 0 ]]; do
  case "$1" in
    --expected-head)
      [[ "$#" -ge 2 ]] || { printf '%s\n' 'error: --expected-head requires SHA' >&2; exit 2; }
      expected_head="$2"
      shift 2
      ;;
    --expected-main)
      [[ "$#" -ge 2 ]] || { printf '%s\n' 'error: --expected-main requires SHA' >&2; exit 2; }
      expected_main="$2"
      shift 2
      ;;
    --remote)
      [[ "$#" -ge 2 ]] || { printf '%s\n' 'error: --remote requires NAME' >&2; exit 2; }
      remote="$2"
      shift 2
      ;;
    --receipt)
      [[ "$#" -ge 2 ]] || { printf '%s\n' 'error: --receipt requires PATH' >&2; exit 2; }
      receipt_path="$2"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      printf 'error: unknown argument %s\n' "$1" >&2
      usage >&2
      exit 2
      ;;
  esac
done

if [[ -z "$expected_head" || -z "$expected_main" ]]; then
  printf '%s\n' 'error: --expected-head and --expected-main are required' >&2
  exit 2
fi

for command in git go bash awk mktemp rm ln date; do
  if ! command -v "$command" >/dev/null 2>&1; then
    printf '%s is required for repository qualification.\n' "$command" >&2
    exit 2
  fi
done

assert_exact_clean() {
  local phase="$1"
  local status nonstandard_index

  status="$(git status --porcelain --untracked-files=all --ignored=matching)"
  if [[ -n "$status" ]]; then
    printf 'Repository is not exact-clean during %s (including ignored paths):\n%s\n' "$phase" "$status" >&2
    exit 1
  fi

  nonstandard_index="$(git ls-files -v | awk 'substr($0,1,1)!="H"{print}')"
  if [[ -n "$nonstandard_index" ]]; then
    printf 'Repository has non-standard tracked index state during %s:\n%s\n' "$phase" "$nonstandard_index" >&2
    exit 1
  fi
}

if [[ -n "$receipt_path" ]]; then
  receipt_parent="$(dirname -- "$receipt_path")"
  receipt_name="$(basename -- "$receipt_path")"
  if ! receipt_dir="$(cd "$receipt_parent" 2>/dev/null && pwd -P)"; then
    printf 'Qualification receipt directory must already exist: %s\n' "$receipt_parent" >&2
    exit 2
  fi
  receipt_path="$receipt_dir/$receipt_name"
  case "$receipt_path" in
    "$repo_root"|"$repo_root"/*)
      printf 'Qualification receipt must remain outside the repository: %s\n' "$receipt_path" >&2
      exit 2
      ;;
  esac
  if [[ -e "$receipt_path" || -L "$receipt_path" ]]; then
    printf 'Qualification receipt path already exists: %s\n' "$receipt_path" >&2
    exit 2
  fi
fi

git fetch "$remote" --prune

candidate_head="$(git rev-parse HEAD)"
candidate_tree="$(git rev-parse HEAD^{tree})"
remote_main_before="$(git rev-parse "$remote/main")"

if [[ "$candidate_head" != "$expected_head" ]]; then
  printf 'Candidate HEAD mismatch: expected=%s actual=%s\n' "$expected_head" "$candidate_head" >&2
  exit 1
fi
if [[ "$remote_main_before" != "$expected_main" ]]; then
  printf 'Current %s/main mismatch: expected=%s actual=%s\n' "$remote" "$expected_main" "$remote_main_before" >&2
  exit 1
fi
if ! git merge-base --is-ancestor "$expected_main" "$candidate_head"; then
  printf 'Expected main is not an ancestor of candidate HEAD: main=%s candidate=%s\n'     "$expected_main" "$candidate_head" >&2
  exit 1
fi

assert_exact_clean "pre-state"

qualification_path="tools/run-repository-local-qualification.sh"
health_path="tools/repository-health-check.sh"
qualification_mode="$(git ls-files -s -- "$qualification_path" | awk 'NR == 1 {print $1}')"
qualification_blob="$(git hash-object "$qualification_path")"
health_mode="$(git ls-files -s -- "$health_path" | awk 'NR == 1 {print $1}')"
health_blob="$(git hash-object "$health_path")"
git_version="$(git --version)"
go_version="$(GOENV=off GOWORK=off GOFLAGS= go env GOVERSION)"
go_os="$(GOENV=off GOWORK=off GOFLAGS= go env GOOS)"
go_arch="$(GOENV=off GOWORK=off GOFLAGS= go env GOARCH)"

bash "$health_path"

head_after_health="$(git rev-parse HEAD)"
if [[ "$head_after_health" != "$candidate_head" ]]; then
  printf 'HEAD changed during repository health: before=%s after=%s\n'     "$candidate_head" "$head_after_health" >&2
  exit 1
fi
assert_exact_clean "post-health"

git fetch "$remote" --prune

head_after_fetch="$(git rev-parse HEAD)"
remote_main_after="$(git rev-parse "$remote/main")"
if [[ "$head_after_fetch" != "$candidate_head" ]]; then
  printf 'HEAD changed before receipt publication: before=%s after=%s\n'     "$candidate_head" "$head_after_fetch" >&2
  exit 1
fi
if [[ "$remote_main_after" != "$remote_main_before" ]]; then
  printf '%s/main moved during qualification: before=%s after=%s\n'     "$remote" "$remote_main_before" "$remote_main_after" >&2
  exit 1
fi
assert_exact_clean "post-fetch"

qualified_at_utc="$(date -u '+%Y-%m-%dT%H:%M:%SZ')"

receipt_json="$(
  cat <<EOF
{
  "schema": "tei-repository-merge-qualification-v1",
  "qualifiedAtUTC": "$qualified_at_utc",
  "candidateHead": "$candidate_head",
  "candidateTree": "$candidate_tree",
  "expectedMain": "$expected_main",
  "remoteMainBefore": "$remote_main_before",
  "remoteMainAfter": "$remote_main_after",
  "candidateContainsExpectedMain": true,
  "gitVersion": "$git_version",
  "goVersion": "$go_version",
  "goOS": "$go_os",
  "goArch": "$go_arch",
  "qualificationHelperMode": "$qualification_mode",
  "qualificationHelperBlob": "$qualification_blob",
  "repositoryHealthMode": "$health_mode",
  "repositoryHealthBlob": "$health_blob",
  "repositoryHealthPassed": true,
  "headStable": true,
  "mainStable": true,
  "cleanPre": true,
  "cleanPost": true,
  "cleanIncludesIgnored": true,
  "nonstandardIndexStateAbsent": true
}
EOF
)"

if [[ -n "$receipt_path" ]]; then
  (
    cd "$receipt_dir"
    if [[ -e "$receipt_name" || -L "$receipt_name" ]]; then
      printf 'Qualification receipt path appeared during qualification: %s\n' "$receipt_path" >&2
      exit 1
    fi

    receipt_tmp="$(mktemp '.tei-repository-qualification-receipt.XXXXXX')"
    receipt_published=false
    cleanup_receipt() {
      status="$?"
      trap - EXIT
      if [[ "$status" -ne 0 && "$receipt_published" == true ]]; then
        if [[ ( -e "$receipt_name" || -L "$receipt_name" ) && "$receipt_tmp" -ef "$receipt_name" ]]; then
          rm -f -- "$receipt_name" || true
        fi
      fi
      rm -f -- "$receipt_tmp" || true
      exit "$status"
    }
    trap cleanup_receipt EXIT

    printf '%s\n' "$receipt_json" >"$receipt_tmp"
    if ! ln -- "$receipt_tmp" "$receipt_name"; then
      printf 'Qualification receipt path appeared before atomic publish: %s\n' "$receipt_path" >&2
      exit 1
    fi
    receipt_published=true

    if [[ "$(git -C "$repo_root" rev-parse HEAD)" != "$candidate_head" ]]; then
      printf '%s\n' 'HEAD changed while publishing qualification receipt.' >&2
      exit 1
    fi
    assert_exact_clean "receipt-publication"

    trap - EXIT
    rm -f -- "$receipt_tmp" || true
  )
fi

printf '%s\n' 'TEI_REPOSITORY_MERGE_QUALIFICATION_V1'
printf '%s\n' "$receipt_json"
printf '%s\n' 'TEI_REPOSITORY_MERGE_QUALIFICATION_END'
if [[ -n "$receipt_path" ]]; then
  printf 'receipt=%s\n' "$receipt_path"
fi
