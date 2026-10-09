#!/usr/bin/env bash
set -euo pipefail

SCRIPT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd -P)"
INTEGRATION_ROOT=""
RECEIPT_FILE=""
CANDIDATE_REF=""
EXPECTED_MAIN=""
REMOTE="origin"

usage() {
  cat <<'EOF'
usage: bash tools/reference-tool-neutral-development/integration-preflight.sh [options]

Read-only preflight for integrating an RQ-034-qualified candidate.

Options:
  --receipt-file PATH   qualification receipt emitted by qualify.sh
  --candidate REF       candidate branch, tag, or commit to integrate
  --expected-main SHA   exact fetched remote main qualified by the receipt
  --remote NAME         integration remote to verify live; default: origin
  --integration-root PATH
                        required clean checkout whose HEAD/remote main are the integration base
  -h, --help            show this help

Run this script from the exact candidate checkout and point --integration-root
at a separate clean checkout of the exact expected main. Keeping verifier source
and integration base separate makes the bootstrap case explicit and prevents
the candidate checkout from being mistaken for the integration base. The script
does not fetch, merge, commit, push, or change refs.
EOF
}

while [[ "$#" -gt 0 ]]; do
  case "$1" in
    --receipt-file)
      [[ "$#" -ge 2 ]] || { echo "error: --receipt-file requires PATH" >&2; exit 2; }
      RECEIPT_FILE="$2"
      shift 2
      ;;
    --candidate)
      [[ "$#" -ge 2 ]] || { echo "error: --candidate requires REF" >&2; exit 2; }
      CANDIDATE_REF="$2"
      shift 2
      ;;
    --expected-main)
      [[ "$#" -ge 2 ]] || { echo "error: --expected-main requires SHA" >&2; exit 2; }
      EXPECTED_MAIN="$2"
      shift 2
      ;;
    --remote)
      [[ "$#" -ge 2 ]] || { echo "error: --remote requires NAME" >&2; exit 2; }
      REMOTE="$2"
      shift 2
      ;;
    --integration-root)
      [[ "$#" -ge 2 ]] || { echo "error: --integration-root requires PATH" >&2; exit 2; }
      INTEGRATION_ROOT="$2"
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

if [[ "$CANDIDATE_REF" == -* ]]; then
  echo "error: --candidate must not begin with '-'" >&2
  exit 2
fi

if [[ -z "$RECEIPT_FILE" || -z "$CANDIDATE_REF" || -z "$EXPECTED_MAIN" || -z "$INTEGRATION_ROOT" ]]; then
  echo "error: --receipt-file, --candidate, --expected-main, and --integration-root are required" >&2
  exit 2
fi

if ! git -C "$SCRIPT_ROOT" rev-parse --is-inside-work-tree >/dev/null 2>&1; then
  echo "error: integration preflight must be executed from a TEI candidate checkout" >&2
  exit 2
fi

INTEGRATION_ROOT="$(cd "$INTEGRATION_ROOT" 2>/dev/null && pwd -P)" || {
  echo "error: integration root does not exist" >&2
  exit 2
}

if [[ "$INTEGRATION_ROOT" == "$SCRIPT_ROOT" ]]; then
  echo "error: candidate checkout and integration root must be separate checkouts" >&2
  exit 2
fi

if ! git -C "$INTEGRATION_ROOT" rev-parse --is-inside-work-tree >/dev/null 2>&1; then
  echo "error: integration root is not a Git checkout" >&2
  exit 2
fi

assert_clean_checkout() {
  local repo="$1"
  local label="$2"
  local status ignored hidden_index
  status="$(git -C "$repo" status --porcelain=v1 --untracked-files=all)"
  ignored="$(git -C "$repo" ls-files --others --ignored --exclude-standard)"
  hidden_index="$(git -C "$repo" ls-files -v | grep -E '^[a-zS]' || true)"
  if [[ -n "$status" || -n "$ignored" || -n "$hidden_index" ]]; then
    echo "error: $label must be clean" >&2
    if [[ -n "$status" ]]; then
      git -C "$repo" status --short --untracked-files=all >&2
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

receipt_value() {
  local key="$1"
  local count line
  count="$(grep -c "^${key}=" "$RECEIPT_FILE" || true)"
  if [[ "$count" != "1" ]]; then
    echo "error: receipt must contain exactly one ${key}= entry" >&2
    exit 1
  fi
  line="$(grep "^${key}=" "$RECEIPT_FILE")"
  printf '%s' "${line#*=}"
}

require_receipt_value() {
  local key="$1"
  local expected="$2"
  local actual
  actual="$(receipt_value "$key")"
  if [[ "$actual" != "$expected" ]]; then
    echo "error: receipt $key mismatch" >&2
    echo "expected: $expected" >&2
    echo "actual:   $actual" >&2
    exit 1
  fi
}

candidate_blob() {
  local path="$1"
  git -C "$SCRIPT_ROOT" rev-parse "$CANDIDATE_HEAD:$path"
}

remote_main_live() {
  local line count
  line="$(git -C "$INTEGRATION_ROOT" ls-remote --exit-code "$REMOTE" refs/heads/main)" || {
    echo "error: cannot read live $REMOTE main" >&2
    exit 1
  }
  count="$(printf '%s\n' "$line" | awk '$2 == "refs/heads/main" { n++ } END { print n + 0 }')"
  if [[ "$count" != "1" ]]; then
    echo "error: live $REMOTE main did not resolve exactly once" >&2
    exit 1
  fi
  printf '%s\n' "$line" | awk '$2 == "refs/heads/main" { print $1 }'
}

[[ -f "$RECEIPT_FILE" ]] || { echo "error: receipt file not found: $RECEIPT_FILE" >&2; exit 2; }
RECEIPT_BLOB="$(git -C "$SCRIPT_ROOT" hash-object "$RECEIPT_FILE")"
assert_clean_checkout "$SCRIPT_ROOT" "candidate checkout"
assert_clean_checkout "$INTEGRATION_ROOT" "integration checkout"

CANDIDATE_HEAD="$(git -C "$SCRIPT_ROOT" rev-parse --verify "$CANDIDATE_REF^{commit}")"
CANDIDATE_TREE="$(git -C "$SCRIPT_ROOT" rev-parse "$CANDIDATE_HEAD^{tree}")"
SCRIPT_HEAD="$(git -C "$SCRIPT_ROOT" rev-parse HEAD)"
CURRENT_HEAD="$(git -C "$INTEGRATION_ROOT" rev-parse HEAD)"

if [[ "$SCRIPT_HEAD" != "$CANDIDATE_HEAD" ]]; then
  echo "error: preflight must execute from the exact candidate checkout" >&2
  echo "candidate: $CANDIDATE_HEAD" >&2
  echo "script checkout HEAD: $SCRIPT_HEAD" >&2
  exit 1
fi

if ! git -C "$INTEGRATION_ROOT" rev-parse --verify "$REMOTE/main" >/dev/null 2>&1; then
  echo "error: $REMOTE/main is unavailable; fetch remote state before preflight" >&2
  exit 1
fi
ORIGIN_MAIN="$(git -C "$INTEGRATION_ROOT" rev-parse "$REMOTE/main")"

if [[ "$ORIGIN_MAIN" != "$EXPECTED_MAIN" ]]; then
  echo "error: local $REMOTE/main moved or expected main is stale" >&2
  echo "expected: $EXPECTED_MAIN" >&2
  echo "actual:   $ORIGIN_MAIN" >&2
  exit 1
fi

LIVE_MAIN_BEFORE="$(remote_main_live)"
if [[ "$LIVE_MAIN_BEFORE" != "$EXPECTED_MAIN" ]]; then
  echo "error: live $REMOTE main moved or local remote-tracking state is stale" >&2
  echo "expected: $EXPECTED_MAIN" >&2
  echo "live:     $LIVE_MAIN_BEFORE" >&2
  exit 1
fi

if [[ "$CURRENT_HEAD" != "$EXPECTED_MAIN" ]]; then
  echo "error: integration checkout HEAD is not the exact expected main" >&2
  echo "expected: $EXPECTED_MAIN" >&2
  echo "actual:   $CURRENT_HEAD" >&2
  exit 1
fi

if ! git -C "$SCRIPT_ROOT" merge-base --is-ancestor "$EXPECTED_MAIN" "$CANDIDATE_HEAD"; then
  echo "error: candidate does not contain the expected main" >&2
  echo "candidate: $CANDIDATE_HEAD" >&2
  echo "main:      $EXPECTED_MAIN" >&2
  exit 1
fi

require_receipt_value "format" "tei-rq034-qualification-v3"
require_receipt_value "candidate-head" "$CANDIDATE_HEAD"
require_receipt_value "candidate-tree" "$CANDIDATE_TREE"
require_receipt_value "origin-main" "$EXPECTED_MAIN"
require_receipt_value "expected-main" "$EXPECTED_MAIN"
require_receipt_value "candidate-contains-expected-main" "true"
require_receipt_value "clean-pre" "PASS"
require_receipt_value "repository-health" "PASS"
require_receipt_value "tool-neutral-development-reference" "PASS"
require_receipt_value "clean-post" "PASS"
require_receipt_value "qualification" "PASS"

require_receipt_value \
  "repository-health-runner-blob" \
  "$(candidate_blob tools/repository-health-check.sh)"
require_receipt_value \
  "qualify-runner-blob" \
  "$(candidate_blob tools/reference-tool-neutral-development/qualify.sh)"
require_receipt_value \
  "reference-runner-blob" \
  "$(candidate_blob tools/reference-tool-neutral-development/run.sh)"
PREFLIGHT_BLOB="$(candidate_blob tools/reference-tool-neutral-development/integration-preflight.sh)"
require_receipt_value \
  "integration-preflight-runner-blob" \
  "$PREFLIGHT_BLOB"

EXECUTING_PREFLIGHT_BLOB="$(git -C "$SCRIPT_ROOT" hash-object "$SCRIPT_ROOT/tools/reference-tool-neutral-development/integration-preflight.sh")"
if [[ "$EXECUTING_PREFLIGHT_BLOB" != "$PREFLIGHT_BLOB" ]]; then
  echo "error: executing integration preflight does not match candidate blob" >&2
  echo "candidate:  $PREFLIGHT_BLOB" >&2
  echo "executing:  $EXECUTING_PREFLIGHT_BLOB" >&2
  exit 1
fi

if [[ "$(git -C "$SCRIPT_ROOT" rev-parse --verify "$CANDIDATE_REF^{commit}")" != "$CANDIDATE_HEAD" ]]; then
  echo "error: candidate ref moved during integration preflight" >&2
  exit 1
fi
if [[ "$(git -C "$SCRIPT_ROOT" rev-parse HEAD)" != "$CANDIDATE_HEAD" ]]; then
  echo "error: candidate checkout HEAD changed during integration preflight" >&2
  exit 1
fi
if [[ "$(git -C "$INTEGRATION_ROOT" rev-parse HEAD)" != "$EXPECTED_MAIN" ]]; then
  echo "error: integration checkout HEAD changed during integration preflight" >&2
  exit 1
fi
if [[ "$(git -C "$INTEGRATION_ROOT" rev-parse "$REMOTE/main")" != "$EXPECTED_MAIN" ]]; then
  echo "error: local $REMOTE/main changed during integration preflight" >&2
  exit 1
fi
LIVE_MAIN_AFTER="$(remote_main_live)"
if [[ "$LIVE_MAIN_AFTER" != "$EXPECTED_MAIN" ]]; then
  echo "error: live $REMOTE main changed during integration preflight" >&2
  echo "expected: $EXPECTED_MAIN" >&2
  echo "live:     $LIVE_MAIN_AFTER" >&2
  exit 1
fi

assert_clean_checkout "$SCRIPT_ROOT" "candidate checkout after preflight"
assert_clean_checkout "$INTEGRATION_ROOT" "integration checkout after preflight"

if [[ "$(git -C "$SCRIPT_ROOT" hash-object "$RECEIPT_FILE")" != "$RECEIPT_BLOB" ]]; then
  echo "error: qualification receipt changed during integration preflight" >&2
  exit 1
fi

printf 'candidate-head=%s\n' "$CANDIDATE_HEAD"
printf 'expected-main=%s\n' "$EXPECTED_MAIN"
printf 'remote=%s\n' "$REMOTE"
printf 'live-main-before=%s\n' "$LIVE_MAIN_BEFORE"
printf 'live-main-after=%s\n' "$LIVE_MAIN_AFTER"
printf 'integration-root=%s\n' "$INTEGRATION_ROOT"
printf 'receipt-blob=%s\n' "$RECEIPT_BLOB"
printf 'receipt-binding=PASS\n'
printf 'runner-blob-binding=PASS\n'
printf 'preflight-self-binding=PASS\n'
printf 'integration-preflight=PASS\n'
