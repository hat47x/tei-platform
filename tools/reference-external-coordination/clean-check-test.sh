#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
RUNNER="$ROOT/tools/reference-external-coordination/run.sh"
TMP_ROOT="$(mktemp -d)"
trap 'rm -rf "$TMP_ROOT"' EXIT

make_fixture() {
  local repo="$1"
  mkdir -p "$repo/tools/reference-external-coordination"
  cp "$RUNNER" "$repo/tools/reference-external-coordination/run.sh"
  printf 'fixture\n' > "$repo/README.md"
  printf 'ignored.txt\n' > "$repo/.gitignore"
  git -C "$repo" init -q
  git -C "$repo" config user.name 'TEI Reference Test'
  git -C "$repo" config user.email 'reference-test@tei.invalid'
  git -C "$repo" add .
  git -C "$repo" commit -qm 'test: seed synthetic product checkout'
}

check_case() {
  local name="$1" expected="$2" repo output actual=0
  repo="$TMP_ROOT/$name"
  output="$TMP_ROOT/$name.log"
  make_fixture "$repo"
  case "$name" in
    clean) ;;
    tracked-dirty) printf 'changed\n' >> "$repo/README.md" ;;
    untracked) printf 'new\n' > "$repo/new.txt" ;;
    ignored) printf 'cache\n' > "$repo/ignored.txt" ;;
    assume-unchanged) git -C "$repo" update-index --assume-unchanged README.md; printf 'masked\n' >> "$repo/README.md" ;;
    skip-worktree) git -C "$repo" update-index --skip-worktree README.md; printf 'masked\n' >> "$repo/README.md" ;;
    *) echo "unknown fixture: $name" >&2; exit 1 ;;
  esac

  if [[ "$name" == assume-unchanged || "$name" == skip-worktree ]]; then
    # Ordinary porcelain output can be empty even when a tracked file was changed.
    test -z "$(git -C "$repo" status --porcelain=v1 --untracked-files=all)"
  fi

  bash "$repo/tools/reference-external-coordination/run.sh" > "$output" 2>&1 || actual=$?
  if [[ "$actual" -ne "$expected" ]]; then
    echo "$name: expected exit $expected, got $actual" >&2
    cat "$output" >&2
    exit 1
  fi
  if [[ "$expected" -eq 0 ]]; then
    grep -Fq 'external coordination git-text reference: PASS' "$output"
    test -z "$(git -C "$repo" status --porcelain=v1 --untracked-files=all)"
    test -z "$(git -C "$repo" ls-files --others --ignored --exclude-standard)"
  else
    grep -Fq 'requires a clean product checkout' "$output"
    if grep -Fq 'external coordination git-text reference: PASS' "$output"; then
      echo "$name: reported PASS despite a rejected checkout" >&2
      exit 1
    fi
  fi
  printf '%s=PASS (exit %s)\n' "$name" "$actual"
}

check_case clean 0
check_case tracked-dirty 2
check_case untracked 2
check_case ignored 2
check_case assume-unchanged 2
check_case skip-worktree 2

NON_GIT="$TMP_ROOT/non-git"
mkdir -p "$NON_GIT/tools/reference-external-coordination"
cp "$RUNNER" "$NON_GIT/tools/reference-external-coordination/run.sh"
exit_code=0
bash "$NON_GIT/tools/reference-external-coordination/run.sh" > "$TMP_ROOT/non-git.log" 2>&1 || exit_code=$?
test "$exit_code" -eq 2
grep -Fq 'requires a Git checkout' "$TMP_ROOT/non-git.log"
printf 'non-git=PASS (exit 2)\n'
printf 'external coordination clean-check regression: PASS\n'
