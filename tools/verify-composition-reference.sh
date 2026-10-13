#!/usr/bin/env bash
# G1/G2と項目対応の参照実装を検査する。製品移行や無停止性の判定は含まない。
set -euo pipefail
if [[ "$#" != 1 ]]; then
  echo "usage: bash tools/verify-composition-reference.sh EXACT_HEAD" >&2
  exit 2
fi
expected="$1"
root="$(git rev-parse --show-toplevel)"
test "$PWD" = "$root"
test "$(git rev-parse HEAD)" = "$expected"
test -z "$(git status --porcelain=v1 --untracked-files=all --ignored)"
export GOENV=off GOWORK=off GOFLAGS= GOTOOLCHAIN=local
printf 'candidate=%s\npre-clean=PASS\n' "$expected"
go version
finish() {
  local result="$?"
  trap - EXIT
  if [[ "$(git rev-parse HEAD)" != "$expected" || -n "$(git status --porcelain=v1 --untracked-files=all --ignored)" ]]; then
    echo 'post-clean-or-head=FAIL' >&2
    exit 1
  fi
  printf 'post-clean=PASS\ncandidate=%s\n' "$expected"
  exit "$result"
}
trap finish EXIT
go test -race -count=20 ./internal/composition ./reference/composition-evidence-handoff ./reference/composition-io-mapping
go test -count=1 ./tools/reference-plugin-capability-contract ./tools/reference-composition-process-bridge
bash tools/check-core-isolation.sh
printf 'scope=reference-only\nG1/G2/field-mapping=PASS\nproduction-migration/continuous-service=NOT-VERIFIED\n'
