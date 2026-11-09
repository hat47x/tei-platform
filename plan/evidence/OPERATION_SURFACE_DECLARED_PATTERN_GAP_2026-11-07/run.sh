#!/bin/bash
# Reproduction for OPERATION_SURFACE_DECLARED_PATTERN_GAP_2026-11-07.
# Run from the repository root:  bash plan/evidence/OPERATION_SURFACE_DECLARED_PATTERN_GAP_2026-11-07/run.sh <outdir>
set -eu
export PATH=/usr/local/go/bin:/usr/bin:/bin:$PATH
[ "$#" -ge 1 ] || { echo "usage: run.sh <outdir>" >&2; exit 2; }
OUT=$(mkdir -p "$1" && cd "$1" && pwd)
REPO=$(cd "$(dirname "$0")/../../.." && pwd)
WORK=$(mktemp -d)
trap 'rm -rf "$WORK"' EXIT

# A git worktree created on Windows stores an absolute Windows path in its
# .git pointer, which a WSL/Linux git cannot resolve. gitdir.sh exports
# GIT_DIR/GIT_WORK_TREE for that case; elsewhere it is a no-op.
# shellcheck disable=SC1091
. "$REPO/plan/evidence/OPERATION_SURFACE_DECLARED_PATTERN_GAP_2026-11-07/gitdir.sh" "$REPO"

git rev-parse HEAD > "$OUT/head.txt"
go version >> "$OUT/head.txt"
# The evidence directory itself is untracked while this runs; exclude it so the
# record shows only changes that this reproduction did not create.
git status --porcelain=v1 --untracked-files=all | grep -v "plan/evidence/OPERATION_SURFACE_DECLARED_PATTERN_GAP_2026-11-07" > "$OUT/pre-status.txt" || true

# The accepted reference Workspace is a historical baseline. Never edit it;
# copy it and overlay the candidate files in a temporary directory, exactly as
# reference/web-composition/go-host/canonical_quote_authorization_test.go does.
cp -r "$REPO/reference/workspace" "$WORK/accepted"
cp -r "$REPO/reference/workspace" "$WORK/overlay"
cp -r "$REPO/reference/candidates/order-quote/applications/order-entry/." "$WORK/overlay/applications/order-entry/"
cp -r "$REPO/reference/candidates/order-quote/semantic/sales/." "$WORK/overlay/semantic/sales/"

cd "$REPO"
go run ./cmd/tei validate -workspace "$WORK/accepted" > "$OUT/accepted-validate.txt" 2>&1 || true
go run ./cmd/tei validate -workspace "$WORK/overlay" > "$OUT/overlay-validate.txt" 2>&1 || true
go run ./cmd/tei operations -workspace "$WORK/accepted" -application order_entry > "$OUT/accepted-operations.json" 2> "$OUT/accepted-operations.err.txt" || true
go run ./cmd/tei operations -workspace "$WORK/overlay" -application order_entry > "$OUT/overlay-operations.json" 2> "$OUT/overlay-operations.err.txt" || true
go run ./cmd/tei artifacts -workspace "$WORK/overlay" -application order_entry > "$OUT/overlay-artifacts.json" 2> "$OUT/overlay-artifacts.err.txt" || true

python3 - "$OUT" <<'PY'
import json, sys, pathlib
out = pathlib.Path(sys.argv[1])
lines = []
for name in ("accepted", "overlay"):
    d = json.load(open(out / f"{name}-operations.json", encoding="utf-8"))
    summary = d.get("summary") or {}
    origins = {}
    for op in d.get("operations") or []:
        origins[op.get("origin")] = origins.get(op.get("origin"), 0) + 1
    lines.append(f"{name}:")
    lines.append(f"  schemaVersion={d.get('schemaVersion')} application={d.get('application')}")
    lines.append(f"  worldFingerprint={d.get('worldFingerprint')}")
    lines.append(f"  modelRevision={d.get('modelRevision')}")
    lines.append(f"  rule={d.get('rule')}")
    lines.append(f"  operations={len(d.get('operations') or [])} gaps={len(d.get('gaps') or [])} conflicts={len(d.get('conflicts') or [])}")
    lines.append(f"  summary={summary}")
    lines.append(f"  origins={origins}")
    for op in d.get("operations") or []:
        lines.append(f"   {op['id']} | kind={op.get('kind')} | target={op.get('target')} | origin={op.get('origin')} | binding={op.get('binding')}")
a = json.load(open(out / "overlay-artifacts.json", encoding="utf-8"))
fn = [x for x in a.get("artifacts") or [] if x.get("kind") == "function-definition"]
lines.append("overlay artifacts:")
lines.append(f"  manifest.complete={a['manifest']['complete']} findings={len(a.get('findings') or [])}")
if fn:
    pats = [l for l in fn[0]["content"].splitlines() if l.startswith("pattern ")]
    lines.append(f"  pattern lines in function-definition={len(pats)}")
    for p in pats:
        lines.append(f"   {p}")
(out / "summary.txt").write_text("\n".join(lines) + "\n", encoding="utf-8", newline="\n")
print("\n".join(lines))
PY