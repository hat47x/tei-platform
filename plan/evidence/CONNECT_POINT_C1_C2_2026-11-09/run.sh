#!/bin/bash
# Compare the two connect points named in the decision material:
#   C-1 routing.Router (deployment execution)
#   C-2 servicehost.SelectedServiceCarrierInvoker (service carrier)
#
# The two are not alternatives for the same command: one executes a deployment
# action, the other carries a Service request. This script records what each
# connect point requires, so a reader can see whether they compete.
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
  | grep -v 'plan/evidence/CONNECT_POINT_C1_C2_2026-11-09' > "$OUT/pre-status.txt" || true

# 1. What each entry point takes, read from its signature and doc comment.
{
  echo '== C-1: routing.Router.Dispatch =='
  grep -n 'func (r Router) Dispatch' -A 6 internal/deployment/routing/executor.go || true
  echo
  echo '== C-1: what selects the executor =='
  grep -n 'func RouteCommand' -A 3 internal/deployment/routing/executor.go || true
  echo
  echo '== C-2: servicehost.SelectedServiceCarrierInvoker.Invoke =='
  grep -n 'func (i *SelectedServiceCarrierInvoker) Invoke' -A 6 internal/servicehost/service_carrier.go || true
  echo
  echo '== C-2: what the invoker requires =='
  grep -n 'func NewSelectedServiceCarrierInvoker' -A 8 internal/servicehost/service_carrier.go || true
} > "$OUT/signatures.txt"

# 2. Which Go type each connect point hands to the runtime layer.
{
  echo '== C-1 hands to the executor SPI =='
  grep -n 'type Executor interface' -A 6 internal/deployment/executor/model.go || true
  echo
  echo '== C-2 hands to the Runtime carrier =='
  grep -n 'func (r Registry) InvokeSelectedServiceCarrier' -A 6 internal/plugin/runtime/service_carrier.go || true
} > "$OUT/hands-to.txt"

# 3. The inputs each connect point needs beyond the command itself.
{
  echo '== C-1 inputs =='
  grep -n 'type selfBuild' /dev/null 2>/dev/null || true
  echo '  command: executor.Command (action id, kind, database/application)'
  echo '  registry: map[core.CanonicalID]executor.Executor, supplied by the Host'
  echo '  environment: read from the Workspace by routing.RouteCommand'
  echo
  echo '== C-2 inputs =='
  echo '  coordinator: *composition.Coordinator (Host generation pinning)'
  echo '  runtime: pruntime.Registry (processes, verifier, service acceptance)'
  echo '  routes: ServiceCarrierRouteSource (Host-owned, per pinned Binding)'
  echo '  plan: pruntime.EnvironmentPlan (from the plugin lock + environment)'
  echo '  selection: pruntime.ServiceProviderSelection (contract/scope/artifact)'
} > "$OUT/inputs.txt"

# 4. Count non-test construction sites of each connect point, per the earlier
#    surveys, to show which one is already reachable from product code.
grep -rl --include='*.go' 'Router{' internal cmd tools reference 2>/dev/null | grep -v '_test.go' > "$OUT/c1-sites.txt" || true
grep -rl --include='*.go' 'NewSelectedServiceCarrierInvoker' internal cmd tools reference 2>/dev/null | grep -v '_test.go' > "$OUT/c2-sites.txt" || true

{
  echo '== non-test sites naming each connect point =='
  echo "C-1 routing.Router: $(wc -l < "$OUT/c1-sites.txt")"
  echo "C-2 NewSelectedServiceCarrierInvoker: $(wc -l < "$OUT/c2-sites.txt")"
} > "$OUT/summary.txt"

cat "$OUT/summary.txt"
