#!/usr/bin/env bash
# 固定したクリーンなLinuxチェックアウトから、実出力と上書き禁止を確認する。
# 検証結果は既存のビルド証拠JSONを標準出力へ返し、作業ファイルは回収する。
set -euo pipefail
if [[ "$#" != 1 ]]; then
  echo "usage: bash tools/reference-transform-runtime-engine-build/verify-export.sh EXACT_HEAD" >&2
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
temporary=""
finish() {
  local result="$?"
  trap - EXIT
  if [[ -n "$temporary" ]]; then rm -rf -- "$temporary"; fi
  if [[ "$(git rev-parse HEAD)" != "$expected" || -n "$(git status --porcelain=v1 --untracked-files=all --ignored)" ]]; then
    echo 'post-clean-or-head=FAIL' >&2
    exit 1
  fi
  printf 'post-clean=PASS\ncandidate=%s\n' "$expected" >&2
  exit "$result"
}
trap finish EXIT
go test -count=1 ./tools/reference-transform-runtime-engine-build >&2
temporary="$(mktemp -d "${TMPDIR:-/tmp}/tei-export-verification.XXXXXXXX")"
temporary="$(cd "$temporary" && pwd -P)"
if [[ "$temporary" = "$root" || "$temporary" = "$root/"* ]]; then
  echo 'verification directory must be outside repository' >&2
  exit 1
fi
artifact="$temporary/engine"
go run ./tools/reference-transform-runtime-engine-build -repo . -artifact-out "$artifact" > "$temporary/build-evidence.json"
before_digest="$(sha256sum "$artifact")"
before_mode="$(stat -c '%a' "$artifact")"
if go run ./tools/reference-transform-runtime-engine-build -repo . -artifact-out "$artifact" > "$temporary/second-evidence.json" 2> "$temporary/second-error.txt"; then
  echo 'existing destination unexpectedly replaced' >&2
  exit 1
fi
test "$before_digest" = "$(sha256sum "$artifact")"
test "$before_mode" = "$(stat -c '%a' "$artifact")"
"$artifact" -source tools/reference-transform-runtime-engine/testdata/echo-string.transform.json < tools/reference-transform-runtime-engine/testdata/echo-string.request.json > "$temporary/response.json"
python3 tools/reference-transform-runtime-engine-build/verify-export.py "$temporary" "$expected" >&2
result="$(cat "$temporary/build-evidence.json")"
rm -rf -- "$temporary"
temporary=""
test "$(git rev-parse HEAD)" = "$expected"
test -z "$(git status --porcelain=v1 --untracked-files=all --ignored)"
printf '%s\n' "$result"
