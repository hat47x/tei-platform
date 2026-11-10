#!/bin/bash
# D-2 of the operation-surface decision brief: operationKindRank ranks four
# kinds and falls back to a catch-all. This script runs the tests that pin the
# relation between the ranking vocabulary and the emitted kinds, then proves the
# tests fail when a derived kind loses its rank.
set -eu

[ "$#" -ge 1 ] || { echo "usage: run.sh <outdir>" >&2; exit 2; }
OUT=$(mkdir -p "$1" && cd "$1" && pwd)
REPO=$(cd "$(dirname "$0")/../../.." && pwd)

if [ -f "$REPO/.git" ]; then
  pointer=$(head -n 1 "$REPO/.git")
  case "$pointer" in
    gitdir:*[A-Za-z]:/*)
      win=${pointer#gitdir: }
      drive=$(printf '%s' "$win" | cut -c1 | tr '[:upper:]' '[:lower:]')
      GIT_DIR="/mnt/$drive${win#*:}"
      export GIT_DIR
      export GIT_WORK_TREE="$REPO"
      ;;
  esac
fi

export PATH=/usr/local/go/bin:/usr/bin:/bin:$PATH
cd "$REPO"

git rev-parse HEAD > "$OUT/head.txt"
go version >> "$OUT/head.txt"
git status --porcelain=v1 --untracked-files=all \
  | grep -v 'plan/evidence/KIND_RANK_VOCABULARY_2026-11-10' > "$OUT/pre-status.txt" || true

go test ./internal/operations -run 'Rank|Vocabulary' -count=1 -v \
  > "$OUT/tests.txt" 2>&1 || true
grep -E '^(=== RUN|--- (PASS|FAIL)|ok|FAIL)' "$OUT/tests.txt" > "$OUT/tests-summary.txt" || true

# Re-prove: drop "update" from the ranking and watch the invariant fail.
cp internal/operations/surface.go /tmp/surface.rkv.backup
python3 - > "$OUT/violation.txt" 2>&1 <<'PY' || true
import io
p = 'internal/operations/surface.go'
t = io.open(p, encoding='utf-8', newline='').read()
old = '\tcase "update":\n\t\treturn 3\n'
if old not in t:
    print('update rank not found; cannot re-prove')
    raise SystemExit(0)
t = t.replace(old, '', 1)
io.open(p, 'w', encoding='utf-8', newline='').write(t)
print('removed the rank for a kind the derivation emits (update)')
PY
go test ./internal/operations -run 'EveryDerivedKindHasItsOwnRank' -count=1 \
  >> "$OUT/violation.txt" 2>&1 || true
cp /tmp/surface.rkv.backup internal/operations/surface.go
echo "restored surface.go" >> "$OUT/violation.txt"

cat "$OUT/tests-summary.txt" | sed 's/^/  /'
echo ""
cat "$OUT/violation.txt" | sed 's/^/  /'
