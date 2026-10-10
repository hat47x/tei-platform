#!/usr/bin/env bash
set -euo pipefail

# Trusted local qualification for Draft PR #1172. Produces an evidence packet,
# not merge authority: parent PRs and the exact main integration must still be
# reviewed under AGENTS.md / CONTRIBUTING.md.
#
# Usage:
#   tools/verify-cli-c1-local.sh <expected-pr-head-sha> <expected-main-sha> <outside-repo-receipt-dir>
#
# Requires a clean, detached checkout of the exact PR head and authenticated
# read-only access to origin. Does not fetch or mutate Git refs.

if [[ "$#" -ne 3 || ! "$1" =~ ^[0-9a-f]{40}$ || ! "$2" =~ ^[0-9a-f]{40}$ ]]; then
  echo "usage: $0 <40-char-pr-head-sha> <40-char-main-sha> <outside-repo-receipt-dir>" >&2
  exit 2
fi
expected_head="$1"
expected_main="$2"
receipt_path="$3"
repo_root="$(git rev-parse --show-toplevel)" || exit 2
repo_root="$(cd "$repo_root" && pwd -P)"
if [[ "$(pwd -P)" != "$repo_root" ]]; then
  echo "run the verifier from the repository root" >&2
  exit 2
fi
if [[ "$(git symbolic-ref -q HEAD || true)" != "" ]]; then
  echo "a detached checkout of the exact candidate SHA is required" >&2
  exit 2
fi
if [[ "$(git rev-parse HEAD)" != "$expected_head" ]]; then
  echo "candidate HEAD differs from supplied expected SHA" >&2
  exit 2
fi
if [[ -n "$(git status --porcelain --untracked-files=all)" ]]; then
  echo "candidate checkout is not clean" >&2
  exit 2
fi

# Receipt files must never dirty the checkout being qualified.
receipt_abs="$(realpath -m "$receipt_path")"
case "$receipt_abs/" in
  "$repo_root/"* )
    echo "receipt directory must be outside the repository" >&2
    exit 2
    ;;
esac

remote_sha() {
  local ref="$1" output
  output="$(git ls-remote --exit-code origin "$ref")" || return 1
  printf '%s\n' "${output%%[[:space:]]*}"
}
check_remotes() {
  local head main
  head="$(remote_sha refs/heads/feat/cli-c1-typed-trace-integration)" || return 1
  main="$(remote_sha refs/heads/main)" || return 1
  if [[ "$head" != "$expected_head" || "$main" != "$expected_main" ]]; then
    echo "stale PR candidate or main; obtain fresh SHAs and requalify" >&2
    echo "remote candidate=$head expected=$expected_head; remote main=$main expected=$expected_main" >&2
    return 1
  fi
}
check_remotes

mkdir -p "$receipt_abs"
chmod 700 "$receipt_abs"
{
  printf 'qualification=cli-c1-pr1172\n'
  printf 'start_utc=%s\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
  printf 'candidate_head=%s\n' "$expected_head"
  printf 'expected_main=%s\n' "$expected_main"
  printf 'go_version=%s\n' "$(go version)"
  printf 'git_version=%s\n' "$(git --version)"
  printf 'checkout_clean_before=true\n'
  printf 'command=tools/repository-health-check.sh\n'
} > "$receipt_abs/receipt.txt"

status=0
tools/repository-health-check.sh > "$receipt_abs/repository-health.log" 2>&1 || status=$?

after_head="$(git rev-parse HEAD)"
after_status="$(git status --porcelain --untracked-files=all)"
remote_status=0
check_remotes >> "$receipt_abs/repository-health.log" 2>&1 || remote_status=$?
{
  printf 'end_utc=%s\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
  printf 'checkout_head_after=%s\n' "$after_head"
  printf 'checkout_clean_after=%s\n' "$([[ -z "$after_status" ]] && echo true || echo false)"
  printf 'repository_health_exit=%s\n' "$status"
  printf 'remote_head_main_recheck_exit=%s\n' "$remote_status"
} >> "$receipt_abs/receipt.txt"

if [[ "$status" -ne 0 || "$after_head" != "$expected_head" ||
      -n "$after_status" || "$remote_status" -ne 0 ]]; then
  echo "FAIL/STALE: no merge authority; see $receipt_abs" >&2
  exit 1
fi
echo "PASS: exact-head Repository Health passed with clean checkout; prerequisite only."
echo "Receipt: $receipt_abs"
echo "Parent PR and current-main integration checks are still required."
