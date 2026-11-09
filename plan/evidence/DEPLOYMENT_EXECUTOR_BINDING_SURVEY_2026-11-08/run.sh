#!/bin/bash
# The product package internal/plugin/runtime exposes a DeploymentExecutor that
# calls InvokeWithProvenance directly (no Host-generation pinning). Who builds it
# outside tests, and with what Plan/Registry?
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
  | grep -v 'plan/evidence/DEPLOYMENT_EXECUTOR_BINDING_SURVEY_2026-11-08' > "$OUT/pre-status.txt" || true

# Every site naming the type, split test / non-test.
: > "$OUT/sites.txt"
while IFS= read -r f; do
  case "$f" in
    *_test.go) tag=TEST ;;
    *) tag="NON-TEST" ;;
  esac
  printf '%s  %s\n' "$tag" "$f" >> "$OUT/sites.txt"
done < <(grep -rl --include='*.go' 'pruntime.DeploymentExecutor{\|runtime.DeploymentExecutor{' internal cmd tools reference examples 2>/dev/null | sort)

# For each non-test construction, show the CapabilityID and where the Plan comes from.
: > "$OUT/construction.txt"
while IFS= read -r line; do
  case "$line" in
    "NON-TEST"*) ;;
    *) continue ;;
  esac
  f=${line#NON-TEST  }
  printf '=== %s ===\n' "$f" >> "$OUT/construction.txt"
  grep -n 'DeploymentExecutor{' -A 4 "$f" >> "$OUT/construction.txt" 2>/dev/null || true
  printf '  -- CapabilityID occurrences --\n' >> "$OUT/construction.txt"
  grep -n 'CapabilityID' "$f" >> "$OUT/construction.txt" 2>/dev/null || true
  printf '\n' >> "$OUT/construction.txt"
done < "$OUT/sites.txt"

# Does any construction site also build a composition.Coordinator?
{
  echo "== construction sites that also use composition.Coordinator =="
  while IFS= read -r line; do
    case "$line" in
      "NON-TEST"*) ;;
      *) continue ;;
    esac
    f=${line#NON-TEST  }
    if grep -q 'Coordinator' "$f" 2>/dev/null; then
      printf '  YES  %s\n' "$f"
    else
      printf '  no   %s\n' "$f"
    fi
  done < "$OUT/sites.txt"
} > "$OUT/coordinator.txt"

{
  echo "== counts =="
  echo "sites naming DeploymentExecutor: $(wc -l < "$OUT/sites.txt")"
  echo "  non-test: $(grep -c '^NON-TEST' "$OUT/sites.txt" || true)"
  echo "  test: $(grep -c '^TEST' "$OUT/sites.txt" || true)"
  echo "non-test sites also using Coordinator: $(grep -c 'YES' "$OUT/coordinator.txt" || true)"
} > "$OUT/summary.txt"

cat "$OUT/summary.txt"
