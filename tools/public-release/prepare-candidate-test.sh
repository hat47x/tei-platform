#!/usr/bin/env bash
set -euo pipefail
here=$(cd "$(dirname "$0")" && pwd)
candidate="$here/prepare-candidate.sh"
verifier="$here/verify-candidate.sh"
work=$(mktemp -d)
trap 'rm -rf -- "$work"' EXIT

mkdir -p "$work/repo/internal"
git -C "$work/repo" init -q
printf '# TEI\n\n> 生成AIを用いて開発しています。人的レビューは未実施です。\n' > "$work/repo/README.md"
printf 'MIT License\n' > "$work/repo/LICENSE"
printf 'package sample\n' > "$work/repo/internal/sample.go"
printf '#!/bin/sh\nexit 0\n' > "$work/repo/internal/tool.sh"
chmod 755 "$work/repo/internal/tool.sh"
printf 'INTERNAL_ONLY\n' > "$work/repo/private-notes.txt"
ln -s ../private-notes.txt "$work/repo/internal/link"
git -C "$work/repo" add -A
git -C "$work/repo" -c user.name=Test -c user.email=test@example.invalid \
    -c commit.gpgsign=false commit -qm initial
sha=$(git -C "$work/repo" rev-parse HEAD)
printf 'README.md\nLICENSE\ninternal/sample.go\ninternal/tool.sh\n' > "$work/paths"

bash "$candidate" "$work/repo" "$sha" "$work/paths" "$work/export" > "$work/log"
[[ -f "$work/export/README.md" && -f "$work/export/LICENSE" ]]
[[ -f "$work/export/internal/sample.go" && ! -e "$work/export/private-notes.txt" ]]
[[ -x "$work/export/internal/tool.sh" ]]
[[ ! -e "$work/export/.git" && ! -e "$work/export/internal/link" ]]
cmp "$work/repo/internal/sample.go" "$work/export/internal/sample.go"
grep -q 'CANDIDATE_ONLY' "$work/log"
bash "$verifier" "$work/repo" "$sha" "$work/paths" "$work/export" > "$work/verify.log"
grep -q 'VERIFIED_CANDIDATE_ONLY' "$work/verify.log"
verify_must_fail() {
  if bash "$verifier" "$work/repo" "$sha" "$work/paths" "$work/export" > /dev/null 2>&1; then
    echo 'expected verifier refusal' >&2
    exit 1
  fi
}
printf 'modified\n' >> "$work/export/README.md"
verify_must_fail
cp "$work/repo/README.md" "$work/export/README.md"
chmod 600 "$work/export/LICENSE"
verify_must_fail
chmod 644 "$work/export/LICENSE"
chmod 644 "$work/export/internal/tool.sh"
verify_must_fail
chmod 755 "$work/export/internal/tool.sh"
printf 'unreviewed\n' > "$work/export/unreviewed.txt"
verify_must_fail
rm "$work/export/unreviewed.txt"
ln -s /dev/null "$work/export/rogue-link"
verify_must_fail
rm "$work/export/rogue-link"
ln "$work/export/LICENSE" "$work/external-hardlink"
verify_must_fail
rm "$work/external-hardlink"
bash "$verifier" "$work/repo" "$sha" "$work/paths" "$work/export" > /dev/null

must_fail() {
  local output=$1
  if bash "$candidate" "$work/repo" "$sha" "$work/paths" "$output" > /dev/null 2>&1; then
    echo "expected refusal: $output" >&2
    exit 1
  fi
  [[ ! -e "$output" ]] || { echo "incomplete output left behind" >&2; exit 1; }
}
printf 'README.md\nLICENSE\ninternal/link\n' > "$work/paths"
must_fail "$work/symlink"
printf 'README.md\nLICENSE\n../other.txt\n' > "$work/paths"
must_fail "$work/traversal"
printf 'README.md\nLICENSE\nnonexistent\n' > "$work/paths"
must_fail "$work/missing"
printf 'README.md\nLICENSE\nREADME.md\n' > "$work/paths"
must_fail "$work/duplicate"
printf 'README.md\nLICENSE\n.git/config\n' > "$work/paths"
must_fail "$work/gitdir"
printf 'README.md\nLICENSE\n' > "$work/paths"
mkdir "$work/reserved.lock"
must_fail "$work/reserved"
rmdir "$work/reserved.lock"
printf '# TEI\n' > "$work/repo/README.md"
# A fixed commit cannot be affected by later uncommitted working-tree changes.
bash "$candidate" "$work/repo" "$sha" "$work/paths" "$work/fixed" > /dev/null
grep -Fq '人的レビュー' "$work/fixed/README.md"
# A new commit without the notice must be refused.
git -C "$work/repo" add README.md
git -C "$work/repo" -c user.name=Test -c user.email=test@example.invalid \
    -c commit.gpgsign=false commit -qm no-notice
previous_sha=$sha
sha=$(git -C "$work/repo" rev-parse HEAD)
must_fail "$work/no-notice"
sha=$previous_sha
# Candidate output within the source repository must be refused.
if bash "$candidate" "$work/repo" "$sha" "$work/paths" "$work/repo/export" > /dev/null 2>&1; then
  echo "expected in-repository refusal" >&2
  exit 1
fi
echo "PASS: candidate builder isolated synthetic tests"
