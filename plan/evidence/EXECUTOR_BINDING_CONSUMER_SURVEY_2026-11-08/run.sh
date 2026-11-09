#!/bin/bash
# Where does the Environment's deployment-executor binding actually go?
#
# The previous record showed that the reference implementations build
# pruntime.DeploymentExecutor with a hard-coded CapabilityID, not from a binding.
# This script follows the binding from its reader to the point where it is used.
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
  | grep -v 'plan/evidence/EXECUTOR_BINDING_CONSUMER_SURVEY_2026-11-08' > "$OUT/pre-status.txt" || true

# 1. Readers of env.DeploymentExecutors.
{
  echo "== non-test readers of Environment.DeploymentExecutors =="
  grep -rn --include='*.go' 'DeploymentExecutors' internal cmd tools reference 2>/dev/null \
    | grep -v '_test.go' || echo '  (none)'
} > "$OUT/readers.txt"

# 2. Who calls routing.RouteCommand / RouteCommands, and what do they do with it?
{
  echo "== non-test callers of routing.RouteCommand(s) =="
  grep -rn --include='*.go' 'RouteCommand' internal cmd tools reference 2>/dev/null \
    | grep -v '_test.go' || echo '  (none)'
} > "$OUT/routing-callers.txt"

# 3. Does the routed result ever reach an executor.Execute? Show the call sites
#    of executor.Execute and the surrounding 3 lines.
{
  echo "== non-test call sites of executor Execute =="
  grep -rn --include='*.go' '\.Execute(' internal cmd tools reference 2>/dev/null \
    | grep -v '_test.go' | grep -v 'ExecuteContractQuery' | head -20 || echo '  (none)'
} > "$OUT/execute-sites.txt"

# 4. Does the CLI actually run an executor, or only record/validate attempts?
{
  echo "== CLI deployment subcommands that mention routing or executor =="
  grep -n 'deploymentrouting\|executor\.' cmd/tei/main.go || true
} > "$OUT/cli.txt"

cat "$OUT/cli.txt"
