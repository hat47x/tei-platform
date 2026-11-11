#!/bin/bash
# CapabilityKinds is what a report lists and what Valid() accepts, and the
# comment on it claims the two cannot drift. This script proves the invariant
# holds today, then shows the test catches each direction of drift.
set -eu

[ "$#" -ge 1 ] || { echo "usage: run.sh <outdir>" >&2; exit 2; }
OUT=$(mkdir -p "$1" && cd "$1" && pwd)
REPO=$(cd "$(dirname "$0")/../../.." && pwd)

export PATH=/usr/local/go/bin:/usr/bin:/bin:$PATH
cd "$REPO"

git rev-parse HEAD > "$OUT/head.txt"
go version >> "$OUT/head.txt"
git status --porcelain=v1 --untracked-files=all \
  | grep -v 'plan/evidence/CAPABILITY_KIND_VOCABULARY_2026-11-11' > "$OUT/pre-status.txt" || true

{
  echo "== the report list =="
  sed -n '/^var CapabilityKinds/,/^}/p' internal/plugin/model.go | sed 's/^/  /'
  echo ""
  echo "== what Valid() reads =="
  sed -n '/^func (k CapabilityKind) Valid/,/^}/p' internal/plugin/model.go | sed 's/^/  /'
  echo ""
  echo "== the validation diagnostic that names the kind =="
  grep -n 'invalid kind' internal/plugin/capability_structure.go | sed 's/^/  /'
  echo ""
  echo "== existing tests that write an invalid kind by hand =="
  grep -rn 'CapabilityKind("missing")' internal/plugin/*_test.go | sed 's/^/  /'
} > "$OUT/anchors.txt"

go test ./internal/plugin -run 'CapabilityKindIsInTheReportList|KindOutsideTheVocabulary|RuntimeKindIsInTheReportList' -count=1 -v \
  > "$OUT/tests.txt" 2>&1 || true
grep -E '^(=== RUN|--- (PASS|FAIL)|ok|FAIL)' "$OUT/tests.txt" > "$OUT/tests-summary.txt" || true

# Drift direction 1: a constant added without updating the list.
cp internal/plugin/model.go /tmp/model.backup
python3 - > "$OUT/drift1.txt" 2>&1 <<'PY' || true
import io
p = 'internal/plugin/model.go'
t = io.open(p, encoding='utf-8', newline='').read()
old = '\tCapabilityEditorIntelligence    CapabilityKind = "editor-intelligence"\n'
new = old + '\tCapabilityTargetGenerator      CapabilityKind = "target-generator"\n'
if old not in t:
    print('constant anchor not found; cannot re-prove')
    raise SystemExit(0)
io.open(p, 'w', encoding='utf-8', newline='').write(t.replace(old, new, 1))
print('added a constant without updating CapabilityKinds')
PY
go test ./internal/plugin -run 'EveryDeclaredCapabilityKindIsInTheReportList' -count=1 \
  >> "$OUT/drift1.txt" 2>&1 || true
cp /tmp/model.backup internal/plugin/model.go
echo "restored model.go" >> "$OUT/drift1.txt"

# Drift direction 2: an entry in the list that no constant names.
cp internal/plugin/model.go /tmp/model.backup2
python3 - > "$OUT/drift2.txt" 2>&1 <<'PY' || true
import io
p = 'internal/plugin/model.go'
t = io.open(p, encoding='utf-8', newline='').read()
old = '\tCapabilityEditorIntelligence,\n}'
new = '\tCapabilityEditorIntelligence, CapabilityKind("language-plugin"),\n}'
if old not in t:
    print('list anchor not found; cannot re-prove')
    raise SystemExit(0)
io.open(p, 'w', encoding='utf-8', newline='').write(t.replace(old, new, 1))
print('added a list entry that no constant names')
PY
go test ./internal/plugin -run 'EveryDeclaredCapabilityKindIsInTheReportList' -count=1 \
  >> "$OUT/drift2.txt" 2>&1 || true
cp /tmp/model.backup2 internal/plugin/model.go
echo "restored model.go" >> "$OUT/drift2.txt"

cat "$OUT/anchors.txt" | sed 's/^/  /'
echo ""
cat "$OUT/tests-summary.txt" | sed 's/^/  /'
echo ""
cat "$OUT/drift1.txt" | sed 's/^/  /'
echo ""
cat "$OUT/drift2.txt" | sed 's/^/  /'
