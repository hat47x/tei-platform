#!/bin/bash
# A catalog entry whose version does not parse is filtered out by
# CompatibleMajor, so the resolver reports it exactly as a missing plugin.
# The management layer distinguishes the two. This script pins the resolver's
# current answers and records the anchors the difference rests on.
#
# git is not used here: a worktree's .git is a file naming a host path, which a
# POSIX shell under Windows cannot resolve. The caller records the revision.
set -eu

[ "$#" -ge 1 ] || { echo "usage: run.sh <outdir> [head]" >&2; exit 2; }
OUT=$(mkdir -p "$1" && (cd "$1" && pwd))
REPO=$(cd "$(dirname "$0")/../../.." && pwd)
HEAD_DESC=${2:-"(not supplied)"}

export PATH=/usr/local/go/bin:/usr/bin:/bin:$PATH
cd "$REPO"

printf '%s\n' "$HEAD_DESC" > "$OUT/head.txt"
go version >> "$OUT/head.txt"

{
  echo "== the acceptance gate: a parse error becomes false =="
  sed -n '/^func CompatibleMajor/,/^}/p' internal/plugin/model.go | sed 's/^/  /'
  echo ""
  echo "== the comparison drops the parse error =="
  sed -n '/^func CompareVersion/,/^}/p' internal/plugin/model.go | sed 's/^/  /'
  echo ""
  echo "== what the parser refuses =="
  sed -n '/^func ParseVersion/,/^}/p' internal/plugin/model.go | sed 's/^/  /'
  echo ""
  echo "== the management layer already separates the two cases =="
  grep -n 'DocumentSelectionVersionUnknown\|DocumentSelectionVersionMismatch' \
    internal/plugin/management/document_preflight.go | sed 's/^/  /'
} > "$OUT/anchors.txt"

go test ./internal/plugin -run \
  'TestAMalformedPluginVersionIsReportedAsNoCompatiblePlugin|TestCompareVersionTreatsUnparseableVersionsAsZero|TestAMalformedVersionDoesNotShadowAUsableOne|TestCompatibleMajorRefusesAMalformedVersion' \
  -count=1 -v > "$OUT/tests.txt" 2>&1 || true
grep -E '^(=== RUN|--- (PASS|FAIL)|    --- (PASS|FAIL)|ok|FAIL)' "$OUT/tests.txt" \
  > "$OUT/tests-summary.txt" || true

cat "$OUT/anchors.txt" | sed 's/^/  /'
echo ""
head -2 "$OUT/head.txt" | sed 's/^/  /'
echo ""
cat "$OUT/tests-summary.txt" | sed 's/^/  /'
