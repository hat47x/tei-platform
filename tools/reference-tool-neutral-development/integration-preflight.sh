#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
RECEIPT_FILE=""
CANDIDATE_REF=""
EXPECTED_MAIN=""

usage() {
  cat <<'EOF'
usage: bash tools/reference-tool-neutral-development/integration-preflight.sh [options]

Read-only preflight for integrating an RQ-034-qualified candidate.

Options:
  --receipt-file PATH   qualification receipt emitted by qualify.sh
  --candidate REF       candidate branch, tag, or commit to integrate
  --expected-main SHA   exact fetched origin/main qualified by the receipt
  -h, --help            show this help

Run after refreshing origin/main. The script does not fetch, merge, commit, push,
or change refs.
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

if [[ -z "$RECEIPT_FILE" || -z "$CANDIDATE_REF" || -z "$EXPECTED_MAIN" ]]; then
  echo "error: --receipt-file, --candidate, and --expected-main are required" >&2
  exit 2
fi

cd "$ROOT"

if ! git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
  echo "error: run from a TEI Git checkout" >&2
  exit 2
fi

assert_clean_checkout() {
  local status ignored
  status="$(git status --porcelain=v1 --untracked-files=all)"
  ignored="$(git ls-files --others --ignored --exclude-standard)"
  if [[ -n "$status" || -n "$ignored" ]]; then
    echo "error: integration preflight requires a clean checkout" >&2
    if [[ -n "$status" ]]; then
      git status --short --untracked-files=all >&2
    fi
    if [[ -n "$ignored" ]]; then
      echo "ignored artifacts:" >&2
      printf '%s\n' "$ignored" >&2
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
  git rev-parse "$CANDIDATE_HEAD:$path"
}

[[ -f "$RECEIPT_FILE" ]] || { echo "error: receipt file not found: $RECEIPT_FILE" >&2; exit 2; }
assert_clean_checkout

CANDIDATE_HEAD="$(git rev-parse "$CANDIDATE_REF^{commit}")"
CURRENT_HEAD="$(git rev-parse HEAD)"

if ! git rev-parse --verify origin/main >/dev/null 2>&1; then
  echo "error: origin/main is unavailable; fetch remote state before preflight" >&2
  exit 1
fi
ORIGIN_MAIN="$(git rev-parse origin/main)"

if [[ "$ORIGIN_MAIN" != "$EXPECTED_MAIN" ]]; then
  echo "error: origin/main moved or expected main is stale" >&2
  echo "expected: $EXPECTED_MAIN" >&2
  echo "actual:   $ORIGIN_MAIN" >&2
  exit 1
fi

if [[ "$CURRENT_HEAD" != "$EXPECTED_MAIN" ]]; then
  echo "error: integration checkout HEAD is not the exact expected main" >&2
  echo "expected: $EXPECTED_MAIN" >&2
  echo "actual:   $CURRENT_HEAD" >&2
  exit 1
fi

if ! git merge-base --is-ancestor "$EXPECTED_MAIN" "$CANDIDATE_HEAD"; then
  echo "error: candidate does not contain the expected main" >&2
  echo "candidate: $CANDIDATE_HEAD" >&2
  echo "main:      $EXPECTED_MAIN" >&2
  exit 1
fi

require_receipt_value "format" "tei-rq034-qualification-v1"
require_receipt_value "candidate-head" "$CANDIDATE_HEAD"
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

assert_clean_checkout

printf 'candidate-head=%s\n' "$CANDIDATE_HEAD"
printf 'expected-main=%s\n' "$EXPECTED_MAIN"
printf 'receipt-binding=PASS\n'
printf 'runner-blob-binding=PASS\n'
printf 'integration-preflight=PASS\n'
