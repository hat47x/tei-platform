#!/bin/bash
# D-4's first question is "where does the comparison value come from". This
# script shows what each candidate actually carries, so the choice can be made
# against facts instead of assumptions.
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
  | grep -v 'plan/evidence/DRIFT_COMPARISON_VALUE_2026-11-11' > "$OUT/pre-status.txt" || true

{
  echo "== candidate 1: the CLI surface (tei operations) identity fields =="
  sed -n '/^type OperationSurface struct/,/^}/p' internal/operations/surface.go | sed 's/^/  /'
  echo ""
  echo "== who fills those fields =="
  grep -n 'WorldFingerprint =\|ModelRevision =' internal/operations/surface.go | sed 's/^/  /'
  echo "  (DeriveOperationSurface does NOT fill them; SurfaceWorkspace does)"
  echo ""
  echo "== candidate 2: the artifact payload (what gets digested) =="
  sed -n '/^type operationSurfacePayload struct/,/^}/p' internal/artifact/surface.go | sed 's/^/  /'
  echo ""
  echo "== does the artifact payload carry the world identity? =="
  hits=$(sed -n '/^type operationSurfacePayload struct/,/^}/p' internal/artifact/surface.go | grep -c 'WorldFingerprint\|ModelRevision' || true)
  echo "  world-identity fields in the payload: $hits"
  echo ""
  echo "== can the CLI read a saved surface back for comparison? =="
  reads=$(grep -c 'os.ReadFile' cmd/tei/cli_operations_artifacts.go || true)
  echo "  os.ReadFile in the operations command: $reads"
  echo ""
  echo "== any drift-comparison flag on the operations command? =="
  flags=$(grep -c 'drift\|expected\|compare' cmd/tei/cli_operations_artifacts.go || true)
  echo "  such flags: $flags"
  echo ""
  echo "== what the artifact digest is computed over =="
  grep -n 'func (a Artifact) computeDigest\|func digestOf\|Digest =' internal/artifact/artifact.go | sed 's/^/  /'
} > "$OUT/anchors.txt"

cat "$OUT/anchors.txt"
