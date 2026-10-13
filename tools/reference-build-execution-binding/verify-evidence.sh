#!/usr/bin/env bash
# 読取り専用の証拠出力を固定版の個別試験と照合し、元のJSONを標準出力へ返す。
set -euo pipefail
if [[ "$#" != 1 ]]; then
  echo "usage: bash tools/reference-build-execution-binding/verify-evidence.sh EXACT_HEAD" >&2
  exit 2
fi
expected="$1"
root="$(git rev-parse --show-toplevel)"
test "$PWD" = "$root"
test "$(git rev-parse HEAD)" = "$expected"
test -z "$(git status --porcelain=v1 --untracked-files=all --ignored)"
export GOENV=off GOWORK=off GOFLAGS= GOTOOLCHAIN=local
printf 'candidate=%s\npre-clean=PASS\n' "$expected" >&2
go version >&2
finish() {
  local result="$?"
  trap - EXIT
  if [[ "$(git rev-parse HEAD)" != "$expected" || -n "$(git status --porcelain=v1 --untracked-files=all --ignored)" ]]; then
    echo 'post-clean-or-head=FAIL' >&2
    exit 1
  fi
  printf 'post-clean=PASS\ncandidate=%s\n' "$expected" >&2
  exit "$result"
}
trap finish EXIT
go test -count=1 ./tools/reference-build-execution-binding >&2
evidence="$(go run ./tools/reference-build-execution-binding)"
printf '%s\n' "$evidence" | python3 tools/reference-build-execution-binding/verify-evidence.py >&2
test "$(git rev-parse HEAD)" = "$expected"
test -z "$(git status --porcelain=v1 --untracked-files=all --ignored)"
printf '%s\n' "$evidence"
