#!/bin/bash
# The previous record counted three non-test paths that call
# InvokeWithProvenance directly. This script reads what each path says it is for,
# so a reader can judge whether the lack of Host-generation pinning is stated.
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
  | grep -v 'plan/evidence/RUNTIME_UNPINNED_PATH_PURPOSES_2026-11-08' > "$OUT/pre-status.txt" || true

# The three non-test call sites reported by the previous record.
SITES=(
  "internal/plugin/runtime/adapters.go"
  "tools/reference-order-rollout-host-continuity/host.go"
  "tools/reference-plugin-capability-contract/runtime_authority_bridge.go"
)

: > "$OUT/sites.txt"
for f in "${SITES[@]}"; do
  printf '=== %s ===\n' "$f" >> "$OUT/sites.txt"
  if [ ! -f "$f" ]; then
    printf '  (file not found)\n\n' >> "$OUT/sites.txt"
    continue
  fi
  printf '  package: %s\n' "$(head -n 1 "$f")" >> "$OUT/sites.txt"
  printf '  -- call site --\n' >> "$OUT/sites.txt"
  grep -n 'InvokeWithProvenance' "$f" >> "$OUT/sites.txt" || true
  printf '  -- nearest preceding comment block (up to 8 lines) --\n' >> "$OUT/sites.txt"
  line=$(grep -n 'InvokeWithProvenance' "$f" | head -n 1 | cut -d: -f1)
  if [ -n "$line" ]; then
    start=$((line - 8))
    [ "$start" -lt 1 ] && start=1
    sed -n "${start},$((line - 1))p" "$f" >> "$OUT/sites.txt"
  fi
  printf '\n' >> "$OUT/sites.txt"
done

# Does any of the three mention pinning, generation, lease or coordinator?
{
  echo "== does each site mention Host pinning vocabulary? =="
  for f in "${SITES[@]}"; do
    printf '%-72s' "$f"
    for needle in 'Host' 'generation' 'lease' 'Coordinator' 'pin'; do
      if grep -q "$needle" "$f" 2>/dev/null; then
        printf '%s=Y ' "$needle"
      else
        printf '%s=- ' "$needle"
      fi
    done
    printf '\n'
  done
} > "$OUT/vocabulary.txt"

cat "$OUT/vocabulary.txt"
