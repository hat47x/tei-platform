#!/usr/bin/env bash
set -euo pipefail

# Read-only, exact-revision acceptance gate for the TEI Web reference.
# No checkout, fetch, rebase, merge, force-update, or GitHub Actions execution.
usage() {
  cat <<'HELP'
Usage: bash reference/web-composition/qualify.sh <candidate-head-sha> <current-main-sha>

In a clean, full local TEI checkout, verify that HEAD is the candidate SHA,
origin/main is the expected main SHA, and main is an ancestor of the candidate.
Then run Web Node/Go conformance (with race), repository-wide health, and verify
that the repository and both Git refs remain unchanged after the checks.
This produces *local* evidence; recheck current GitHub PR head and main before merge.
HELP
}
if [[ "${1:-}" == --help || "${1:-}" == -h ]]; then usage; exit 0; fi
if (($# != 2)); then usage >&2; exit 2; fi
candidate="$1"
base="$2"
for sha in "$candidate" "$base"; do
  if [[ ! "$sha" =~ ^[[:xdigit:]]{40}$ ]]; then
    echo "error: expected a complete 40-hex Git revision" >&2
    exit 2
  fi
done

root="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../.." && pwd -P)"
cd "$root"
if [[ "$(git rev-parse --show-toplevel 2>/dev/null || true)" != "$root" ]]; then
  echo "error: exact-head qualification requires repository root" >&2
  exit 2
fi
if [[ ! -f go.mod || ! -d reference/workspace || ! -f reference/web-composition/plan-check.test.mjs || ! -f tools/repository-health-check.sh ]]; then
  echo "error: expected complete TEI checkout" >&2
  exit 2
fi

head_now() { git rev-parse HEAD; }
main_now() { git rev-parse --verify refs/remotes/origin/main 2>/dev/null || true; }
clean() { [[ -z "$(git status --porcelain=v1 --untracked-files=all)" ]]; }
assert_refs() {
  if [[ "$(head_now)" != "$candidate" || "$(main_now)" != "$base" ]]; then
    echo "FAIL: candidate HEAD or origin/main no longer matches exact qualification inputs" >&2
    return 1
  fi
  if ! clean; then
    echo "FAIL: checkout is not clean" >&2
    git status --short >&2
    return 1
  fi
}

assert_refs
if ! git merge-base --is-ancestor "$base" "$candidate"; then
  echo "FAIL: candidate does not contain the specified current main revision" >&2
  exit 1
fi

# Even if a command exits nonzero, check for changes to the exact candidate.
finish() {
  local status=$?
  trap - EXIT
  if ! assert_refs; then status=1; fi
  if ((status == 0)); then
    echo "PASS: exact-head Web + full-repository qualification, HEAD=$candidate main=$base" >&2
  else
    echo "FAIL: exact-head Web + full-repository qualification, HEAD=$candidate main=$base" >&2
  fi
  exit "$status"
}
trap finish EXIT

printf 'Candidate HEAD: %s\nCurrent main:   %s\n' "$candidate" "$base"
bash reference/web-composition/verify.sh --race
bash tools/repository-health-check.sh
