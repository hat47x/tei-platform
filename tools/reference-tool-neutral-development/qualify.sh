#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
EXPECTED_HEAD=""
EXPECTED_MAIN=""
RECEIPT_FILE=""

usage() {
  cat <<'EOF'
usage: tools/reference-tool-neutral-development/qualify.sh [options]

Run the full repository health gate and the tool-neutral development reference
against one unchanged clean checkout.

Options:
  --expected-head SHA   fail unless current HEAD is exactly SHA
  --expected-main SHA   fail unless origin/main is exactly SHA and candidate contains it
  --receipt-file PATH   write a machine-readable PASS receipt outside the repository
  -h, --help            show this help

Before using --expected-main, refresh remote refs with the repository's normal
preflight (for example git fetch --prune). This script intentionally does not
perform network operations or update refs.
EOF
}

while [[ "$#" -gt 0 ]]; do
  case "$1" in
    --expected-head)
      [[ "$#" -ge 2 ]] || { echo "error: --expected-head requires SHA" >&2; exit 2; }
      EXPECTED_HEAD="$2"
      shift 2
      ;;
    --expected-main)
      [[ "$#" -ge 2 ]] || { echo "error: --expected-main requires SHA" >&2; exit 2; }
      EXPECTED_MAIN="$2"
      shift 2
      ;;
    --receipt-file)
      [[ "$#" -ge 2 ]] || { echo "error: --receipt-file requires PATH" >&2; exit 2; }
      RECEIPT_FILE="$2"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    *)
      echo "error: unknown argument '$1'" >&2
      usage >&2
      exit 2
      ;;
  esac
done

cd "$ROOT"

if [[ -n "$RECEIPT_FILE" && ( -z "$EXPECTED_HEAD" || -z "$EXPECTED_MAIN" ) ]]; then
  echo "error: --receipt-file requires both --expected-head and --expected-main" >&2
  exit 2
fi

if [[ -n "$RECEIPT_FILE" ]]; then
  RECEIPT_DIR="$(cd "$(dirname "$RECEIPT_FILE")" 2>/dev/null && pwd -P)" || {
    echo "error: receipt parent directory does not exist" >&2
    exit 2
  }
  RECEIPT_FILE="$RECEIPT_DIR/$(basename "$RECEIPT_FILE")"
  case "$RECEIPT_FILE" in
    "$ROOT"|"$ROOT"/*)
      echo "error: receipt file must be outside the TEI checkout" >&2
      exit 2
      ;;
  esac
  if [[ -e "$RECEIPT_FILE" || -L "$RECEIPT_FILE" ]]; then
    echo "error: receipt file already exists: $RECEIPT_FILE" >&2
    exit 2
  fi
fi

if ! git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
  echo "error: run from a TEI Git checkout" >&2
  exit 2
fi

assert_clean_checkout() {
  local phase="$1"
  local status ignored hidden_index

  status="$(git status --porcelain=v1 --untracked-files=all)"
  ignored="$(git ls-files --others --ignored --exclude-standard)"
  hidden_index="$(git ls-files -v | grep -E '^[a-zS]' || true)"

  if [[ -n "$status" || -n "$ignored" || -n "$hidden_index" ]]; then
    echo "error: qualification checkout is not clean during $phase" >&2
    if [[ -n "$status" ]]; then
      git status --short --untracked-files=all >&2
    fi
    if [[ -n "$ignored" ]]; then
      echo "ignored artifacts:" >&2
      printf '%s\n' "$ignored" >&2
    fi
    if [[ -n "$hidden_index" ]]; then
      echo "index flags that can hide tracked worktree content:" >&2
      printf '%s\n' "$hidden_index" >&2
    fi
    exit 1
  fi
}

CANDIDATE_HEAD="$(git rev-parse HEAD)"

if [[ -n "$EXPECTED_HEAD" && "$CANDIDATE_HEAD" != "$EXPECTED_HEAD" ]]; then
  echo "error: candidate HEAD mismatch" >&2
  echo "expected: $EXPECTED_HEAD" >&2
  echo "actual:   $CANDIDATE_HEAD" >&2
  exit 1
fi

assert_clean_checkout "pre-state"

ORIGIN_MAIN=""
if git rev-parse --verify origin/main >/dev/null 2>&1; then
  ORIGIN_MAIN="$(git rev-parse origin/main)"
fi

if [[ -n "$EXPECTED_MAIN" ]]; then
  if [[ -z "$ORIGIN_MAIN" ]]; then
    echo "error: --expected-main was supplied but origin/main is unavailable" >&2
    exit 1
  fi
  if [[ "$ORIGIN_MAIN" != "$EXPECTED_MAIN" ]]; then
    echo "error: origin/main mismatch" >&2
    echo "expected: $EXPECTED_MAIN" >&2
    echo "actual:   $ORIGIN_MAIN" >&2
    exit 1
  fi
  if ! git merge-base --is-ancestor "$EXPECTED_MAIN" "$CANDIDATE_HEAD"; then
    echo "error: candidate HEAD does not contain expected main" >&2
    echo "candidate: $CANDIDATE_HEAD" >&2
    echo "main:      $EXPECTED_MAIN" >&2
    exit 1
  fi
fi

assert_unchanged() {
  local phase="$1"
  local current_head
  current_head="$(git rev-parse HEAD)"

  if [[ "$current_head" != "$CANDIDATE_HEAD" ]]; then
    echo "error: HEAD changed during $phase" >&2
    echo "before: $CANDIDATE_HEAD" >&2
    echo "after:  $current_head" >&2
    exit 1
  fi

  assert_clean_checkout "$phase"

  if [[ -n "$EXPECTED_MAIN" ]]; then
    local current_main
    current_main="$(git rev-parse origin/main)"
    if [[ "$current_main" != "$EXPECTED_MAIN" ]]; then
      echo "error: origin/main changed during $phase" >&2
      echo "expected: $EXPECTED_MAIN" >&2
      echo "actual:   $current_main" >&2
      exit 1
    fi
  fi
}

write_receipt() {
  local path="$1"
  local tmp receipt_name
  local git_version go_version python_version
  local health_blob qualify_blob reference_blob preflight_blob candidate_tree

  receipt_name="$(basename "$path")"
  tmp="$(mktemp "$RECEIPT_DIR/.tei-rq034-qualification.XXXXXX")"
  git_version="$(git --version | tr '\n' ' ')"
  go_version="$(go version | tr '\n' ' ')"
  if command -v python3 >/dev/null 2>&1; then
    python_version="$(python3 --version 2>&1 | tr '\n' ' ')"
  else
    python_version="unavailable"
  fi
  candidate_tree="$(git rev-parse "$CANDIDATE_HEAD^{tree}")"
  health_blob="$(git rev-parse "$CANDIDATE_HEAD:tools/repository-health-check.sh")"
  qualify_blob="$(git rev-parse "$CANDIDATE_HEAD:tools/reference-tool-neutral-development/qualify.sh")"
  reference_blob="$(git rev-parse "$CANDIDATE_HEAD:tools/reference-tool-neutral-development/run.sh")"
  preflight_blob="$(git rev-parse "$CANDIDATE_HEAD:tools/reference-tool-neutral-development/integration-preflight.sh")"

  {
    printf 'format=tei-rq034-qualification-v3\n'
    printf 'candidate-head=%s\n' "$CANDIDATE_HEAD"
    printf 'candidate-tree=%s\n' "$candidate_tree"
    printf 'origin-main=%s\n' "$ORIGIN_MAIN"
    printf 'expected-main=%s\n' "$EXPECTED_MAIN"
    printf 'candidate-contains-expected-main=true\n'
    printf 'clean-pre=PASS\n'
    printf 'repository-health=PASS\n'
    printf 'tool-neutral-development-reference=PASS\n'
    printf 'clean-post=PASS\n'
    printf 'qualification=PASS\n'
    printf 'repository-health-runner-blob=%s\n' "$health_blob"
    printf 'qualify-runner-blob=%s\n' "$qualify_blob"
    printf 'reference-runner-blob=%s\n' "$reference_blob"
    printf 'integration-preflight-runner-blob=%s\n' "$preflight_blob"
    printf 'git-version=%s\n' "$git_version"
    printf 'go-version=%s\n' "$go_version"
    printf 'python-version=%s\n' "$python_version"
  } > "$tmp"

  assert_unchanged "qualification receipt preparation"

  if [[ -e "$path" || -L "$path" ]]; then
    rm -f -- "$tmp"
    echo "error: receipt file appeared during qualification: $path" >&2
    exit 1
  fi
  if ! (cd "$RECEIPT_DIR" && ln -- "$(basename "$tmp")" "$receipt_name"); then
    rm -f -- "$tmp"
    echo "error: receipt file appeared before atomic publish: $path" >&2
    exit 1
  fi
  rm -f -- "$tmp"
}
echo "==> full repository health" >&2
"$ROOT/tools/repository-health-check.sh"
assert_unchanged "repository health"

echo "==> tool-neutral development reference" >&2
"$ROOT/tools/reference-tool-neutral-development/run.sh"
assert_unchanged "tool-neutral development reference"

printf 'candidate-head=%s\n' "$CANDIDATE_HEAD"
if [[ -n "$ORIGIN_MAIN" ]]; then
  printf 'origin-main=%s\n' "$ORIGIN_MAIN"
fi
if [[ -n "$EXPECTED_MAIN" ]]; then
  printf 'candidate-contains-expected-main=true\n'
fi
printf 'repository-health=PASS\n'
printf 'tool-neutral-development-reference=PASS\n'
if [[ -n "$RECEIPT_FILE" ]]; then
  write_receipt "$RECEIPT_FILE"
  printf 'receipt-file=%s\n' "$RECEIPT_FILE"
  printf 'receipt-blob=%s\n' "$(git hash-object "$RECEIPT_FILE")"
fi
printf 'qualification=PASS\n'
