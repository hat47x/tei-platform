#!/bin/bash
# Section 12 says Instances and Bindings are "read, indexed, validated" but asks
# whether a resolver is needed. This script checks whether one already exists and
# whether product code calls it to reach a live executor.
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
  | grep -v 'plan/evidence/INSTANCES_BINDINGS_RESOLVER_2026-11-09' > "$OUT/pre-status.txt" || true

{
  echo "== does a resolver over Instances+Bindings already exist? =="
  grep -rn 'func ResolveContractQueryExecutor' --include='*.go' internal/ || echo "  (none)"
  echo ""
  echo "== the resolved value it returns =="
  grep -n -A6 'type ResolvedContractQueryExecutor struct' internal/realization/contract_query_binding.go || echo "  (none)"
  echo ""
  echo "== non-test product call sites =="
  hits=$(grep -rn 'realization.ResolveContractQueryExecutor(' --include='*.go' internal/ cmd/ 2>/dev/null | grep -v '_test.go' || true)
  if [ -z "$hits" ]; then echo "  (none)"; else printf '%s\n' "$hits" | sed 's/^/  /'; fi
  echo ""
  echo "== test call sites =="
  grep -rc 'ResolveContractQueryExecutor' --include='*_test.go' internal/ reference/ 2>/dev/null | grep -v ':0$' | sed 's/^/  /' || echo "  (none)"
  echo ""
  echo "== does it check the instance exists and matches the target? =="
  grep -n 'InstanceByID\|instance.Definition != target.Database\|BINDING\|Binding.*蜈・焚' internal/realization/contract_query_binding.go 2>/dev/null | head -6 | sed 's/^/  /' || echo "  (none)"
} > "$OUT/anchors.txt"

cat "$OUT/anchors.txt"
