#!/bin/bash
# The DeploymentExecutor adapter calls the runtime with a fixed CapabilityID. Can
# it verify that the Environment's binding actually selects that id? This script
# shows what the adapter has available and whether it can route.
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
  | grep -v 'plan/evidence/ADAPTER_BINDING_BLINDSPOT_2026-11-09' > "$OUT/pre-status.txt" || true

{
  echo "== DeploymentExecutor adapter struct =="
  sed -n '/^type DeploymentExecutor struct/,/^}/p' internal/plugin/runtime/adapters.go | sed 's/^/  /'
  echo ""
  echo "== EnvironmentPlan fields (does it carry bindings?) =="
  sed -n '/^type EnvironmentPlan struct/,/^}/p' internal/plugin/runtime/model.go | sed 's/^/  /'
  echo ""
  echo "== does the adapter ever consult the binding? =="
  hits=$(grep -n 'RouteCommand\|routing\.' internal/plugin/runtime/adapters.go | grep -v 'reading\|// ' || true)
  if [ -z "$hits" ]; then echo "  (no; routing is mentioned only in a comment)"; else printf '%s\n' "$hits" | sed 's/^/  /'; fi
  echo ""
  echo "== what the adapter does check =="
  sed -n '/^func (a DeploymentExecutor) Supports/,/^}/p' internal/plugin/runtime/adapters.go | sed 's/^/  /'
  echo ""
  echo "== do the non-test construction sites route before Execute? =="
  for f in tools/reference-deployment-attempt/main.go tools/reference-plugin-build-supply/stage4.go tools/reference-plugin-self-build/main.go examples/process-plugin/main.go; do
    if [ -f "$f" ]; then
      routed=$(grep -c 'RouteCommand' "$f" || true)
      validated=$(grep -c 'ValidateResult' "$f" || true)
      dispatched=$(grep -c '\.Dispatch(' "$f" || true)
      echo "  $f: RouteCommand=$routed ValidateResult=$validated Dispatch=$dispatched"
    fi
  done
} > "$OUT/anchors.txt"

cat "$OUT/anchors.txt"
