#!/bin/bash
# Observe whether an operation surface can be checked for drift at execution.
#
# Usage: run.sh <outdir>
#
# Questions:
#   1. Does `tei operations` accept an expected fingerprint (a way to bind a
#      stored surface to the current meaning)?
#   2. Does the execution guard that reads the surface compare any fingerprint
#      before invoking?
#   3. What happens when the meaning changes between a surface and a later run?
#
# Reads only. Workspaces are copied; the fixtures are never modified.
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
  | grep -v 'plan/evidence/OPERATION_DRIFT_CHECK_SURVEY_2026-11-07' > "$OUT/pre-status.txt" || true

# Q1: does `operations` take an expected fingerprint flag?
{
  echo "== flags defined for the operations command =="
  grep -n 'operations' cmd/tei/main.go | head -20 || true
  echo
  echo "== expected-fingerprint flags anywhere in cmd/tei =="
  grep -rn 'expected-fingerprint\|ExpectedFingerprint' cmd/tei --include='*.go' | head -20 || true
} > "$OUT/flag-scan.txt"

# Q2: does the code that reads the surface at execution compare a fingerprint?
{
  echo "== callers of the operation surface outside tests =="
  grep -rn --include='*.go' -e 'DeriveOperationSurface' -e 'SurfaceWorkspace' \
    internal cmd tools reference 2>/dev/null | grep -v '_test.go' || true
  echo
  echo "== fingerprint comparisons in internal/operations and internal/execution =="
  grep -rn --include='*.go' -e 'WorldFingerprint' -e 'ModelRevision' internal/operations internal/execution \
    | grep -v '_test.go' || true
  echo
  echo "== OPERATION_SURFACE_STALE anywhere =="
  grep -rn --include='*.go' 'OPERATION_SURFACE_STALE' internal cmd tools 2>/dev/null || echo "(no occurrences)"
} > "$OUT/drift-scan.txt"

# Q3: change the meaning and ask whether the surface or a run guard notices.
WORK=$(mktemp -d)
trap 'rm -rf "$WORK"' EXIT
cp -r reference/standard-crud/workspace "$WORK/before"
cp -r reference/standard-crud/workspace "$WORK/after"
# Add one attribute to the semantic data. This changes the World and the
# revision without touching any accepted baseline.
python3 - "$WORK/after/semantic/sales/order.tei" <<'PY'
import io
import sys

path = sys.argv[1]
text = io.open(path, encoding='utf-8', newline='').read()
if 'attribute note type std.text' not in text:
    lines = text.split('\n')
    for index, line in enumerate(lines):
        if 'attribute status' in line:
            lines.insert(index + 1, '    attribute note type std.text')
            break
    io.open(path, 'w', encoding='utf-8', newline='').write('\n'.join(lines))
print('attribute added: %s' % path)
PY

for name in before after; do
  go run ./cmd/tei operations -workspace "$WORK/$name" -application ledger \
    > "$OUT/$name-operations.json" 2> "$OUT/$name-operations.err.txt"
done

python3 - "$OUT" <<'PY'
import json
import pathlib
import sys

out = pathlib.Path(sys.argv[1])
lines = []
for name in ("before", "after"):
    surface = json.load(open(out / (name + "-operations.json"), encoding="utf-8"))
    lines.append("== %s ==" % name)
    lines.append("  worldFingerprint: %s" % surface["worldFingerprint"])
    lines.append("  modelRevision: %s" % surface["modelRevision"])
    lines.append("  operations: %d" % len(surface["operations"]))
    for op in surface["operations"]:
        lines.append("  %s | basis=%s" % (op["id"], op["basis"]))
    lines.append("")
before = json.load(open(out / "before-operations.json", encoding="utf-8"))
after = json.load(open(out / "after-operations.json", encoding="utf-8"))
lines.append("worldFingerprint changed: %s" % (before["worldFingerprint"] != after["worldFingerprint"]))
lines.append("modelRevision changed: %s" % (before["modelRevision"] != after["modelRevision"]))
shared = [op for op in after["operations"]
          if op["id"] in {b["id"] for b in before["operations"]}]
same_basis = all(
    op["basis"] == next(b for b in before["operations"] if b["id"] == op["id"])["basis"]
    for op in shared)
lines.append("shared operation ids: %d" % len(shared))
lines.append("shared basis unchanged: %s" % same_basis)
lines.append("")
lines.append("NOTE: nothing in the CLI compares these fingerprints before a run;")
lines.append("      the question is whether a caller CAN, not whether it did.")
(out / "summary.txt").write_text("\n".join(lines) + "\n", encoding="utf-8", newline="\n")
print("\n".join(lines))
PY
