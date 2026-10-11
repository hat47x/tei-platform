#!/usr/bin/env bash
set -euo pipefail
here=$(cd "$(dirname "$0")" && pwd)
work=$(mktemp -d)
trap 'rm -rf -- "$work"' EXIT
mkdir -p "$work/repo/cmd/tei" "$work/repo/tools/probe/testdata"
git -C "$work/repo" init -q
printf '# TEI\n\n> 生成AIで開発中、体系的な人的レビューは未実施\n' > "$work/repo/README.md"
printf 'Test license\n' > "$work/repo/LICENSE"
printf 'module tei\n\ngo 1.23.2\n' > "$work/repo/go.mod"
touch "$work/repo/go.sum"
printf 'package main\nfunc main() {}\n' > "$work/repo/cmd/tei/main.go"
printf 'package main\nfunc main() {}\n' > "$work/repo/tools/probe/main.go"
printf 'package main\n' > "$work/repo/tools/probe/main_test.go"
printf '{"value":1}\n' > "$work/repo/tools/probe/testdata/sample.json"
git -C "$work/repo" add -A
git -C "$work/repo" -c user.name=Test -c user.email=test@example.invalid -c commit.gpgsign=false commit -qm initial
sha=$(git -C "$work/repo" rev-parse HEAD)
{
  printf '# Based on exact private main revision: %s\n' "$sha"
  git -C "$work/repo" ls-tree -r --name-only "$sha"
} > "$work/full"
python3 "$here/audit-profile-coverage.py" "$work/repo" "$sha" "$work/full" --profile development > "$work/pass"
grep -q 'go_tests_selected=1/1' "$work/pass"
grep -q 'STATIC_FILE_COVERAGE_ONLY: PASS' "$work/pass"

must_fail() {
  if python3 "$here/audit-profile-coverage.py" "$work/repo" "$sha" "$1" --profile development > /dev/null 2>&1; then
    echo 'Expected a closed failure for incomplete profile' >&2
    exit 1
  fi
}
grep -v 'tools/probe/main_test.go' "$work/full" > "$work/without-test"
must_fail "$work/without-test"
python3 "$here/audit-profile-coverage.py" "$work/repo" "$sha" "$work/without-test" --profile distribution > "$work/distribution"
grep -q 'WARNING: Distribution is not equivalent' "$work/distribution"

grep -v 'tools/probe/main.go' "$work/full" > "$work/without-go"
must_fail "$work/without-go"
grep -v 'tools/probe/testdata/sample.json' "$work/full" > "$work/without-fixture"
must_fail "$work/without-fixture"
sed "s/$sha/0000000000000000000000000000000000000000/" "$work/full" > "$work/wrong-sha"
must_fail "$work/wrong-sha"
cat "$work/full" "$work/full" > "$work/duplicate"
must_fail "$work/duplicate"
printf '# Based on exact private main revision: %s\nREADME.md\n../private\n' "$sha" > "$work/path-traversal"
must_fail "$work/path-traversal"

printf '# No disclosure here\n' > "$work/repo/README.md"
git -C "$work/repo" add README.md
git -C "$work/repo" -c user.name=Test -c user.email=test@example.invalid -c commit.gpgsign=false commit -qm no-disclosure
badsha=$(git -C "$work/repo" rev-parse HEAD)
sed "s/$sha/$badsha/" "$work/full" > "$work/new-head"
if python3 "$here/audit-profile-coverage.py" "$work/repo" "$badsha" "$work/new-head" --profile development > /dev/null 2>&1; then
  echo 'Expected refusal for missing README disclosure' >&2
  exit 1
fi
echo 'PASS: source-pinned public profile coverage checks'
