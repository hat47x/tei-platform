#!/bin/bash
# status.md says Runtime has "3 paths that call InvokeWithProvenance directly,
# none using composition.Coordinator". This script separates, for each caller:
#   - is it inside the Runtime package (an internal wrapper)?
#   - is it a product construction of a Host-facing adapter, or a reference?
#   - is the adapter it uses constructed anywhere at all?
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
  | grep -v 'plan/evidence/UNPINNED_ADAPTER_SURVEY_2026-11-09' > "$OUT/pre-status.txt" || true

{
  echo "== all non-test InvokeWithProvenance callers =="
  grep -rn 'InvokeWithProvenance(' --include='*.go' internal/ cmd/ tools/ examples/ 2>/dev/null \
    | grep -v '_test.go' | grep -v 'func (r Registry) InvokeWithProvenance' \
    | sed 's/^/  /'
  echo ""
  echo "== all non-test Registry.Invoke callers inside internal/ =="
  grep -rn 'Registry\.Invoke(\|\.Registry\.Invoke(' --include='*.go' internal/ 2>/dev/null \
    | grep -v '_test.go' | sed 's/^/  /'
  echo ""
  echo "== is the Host-facing ObservationProvider adapter constructed anywhere? =="
  hits=$(grep -rn 'ObservationProvider{' --include='*.go' . 2>/dev/null || true)
  if [ -z "$hits" ]; then echo "  (zero constructions, including tests)"; else printf '%s\n' "$hits" | sed 's/^/  /'; fi
  echo ""
  echo "== is the Host-facing EvidenceProvider adapter constructed anywhere? =="
  hits=$(grep -rn 'EvidenceProvider{' --include='*.go' . 2>/dev/null || true)
  if [ -z "$hits" ]; then echo "  (zero constructions, including tests)"; else printf '%s\n' "$hits" | sed 's/^/  /'; fi
  echo ""
  echo "== for comparison: DeploymentExecutor non-test constructions =="
  grep -rn 'DeploymentExecutor{' --include='*.go' . 2>/dev/null | grep -v '_test.go' | sed 's/^/  /'
  echo ""
  echo "== does the pinned wrapper exist and who calls it? =="
  grep -rn 'func (r Registry) InvokeSelectedServiceCarrier\|InvokeSelectedServiceCarrier(' --include='*.go' internal/ 2>/dev/null \
    | grep -v '_test.go' | sed 's/^/  /'
} > "$OUT/anchors.txt"

cat "$OUT/anchors.txt"
