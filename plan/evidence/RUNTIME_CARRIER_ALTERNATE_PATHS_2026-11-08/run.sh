#!/bin/bash
# internal/servicehost is not the only way to reach the Runtime. Enumerate the
# non-test entry points that reach plugin/runtime, and whether each goes through
# servicehost's Host-generation pinning or not.
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
  | grep -v 'plan/evidence/RUNTIME_CARRIER_ALTERNATE_PATHS_2026-11-08' > "$OUT/pre-status.txt" || true

scan_importers() {
  pkg="$1"
  label="$2"
  echo "== $label ==" >> "$OUT/importers.txt"
  found=0
  while IFS= read -r f; do
    case "$f" in *_test.go) continue ;; esac
    if grep -q "\"$pkg\"" "$f" 2>/dev/null; then
      echo "  NON-TEST  $f" >> "$OUT/importers.txt"
      found=$((found + 1))
    fi
  done < <(find internal cmd tools reference examples -name '*.go' -type f 2>/dev/null)
  echo "  count: $found" >> "$OUT/importers.txt"
  echo "" >> "$OUT/importers.txt"
}

: > "$OUT/importers.txt"
scan_importers 'tei/internal/plugin/runtime' 'non-test importers of plugin/runtime'
scan_importers 'tei/internal/servicehost' 'non-test importers of servicehost'

# For each non-test importer of plugin/runtime, does it also reference servicehost?
{
  echo "== per importer: does it reach the Runtime via servicehost? =="
  while IFS= read -r line; do
    f=$(printf '%s' "$line" | sed 's/^  NON-TEST  //')
    [ -f "$f" ] || continue
    if grep -q '"tei/internal/servicehost"' "$f" 2>/dev/null; then
      via=servicehost
    else
      via=direct
    fi
    printf '  %-8s %s\n' "$via" "$f"
  done < <(grep '  NON-TEST  ' "$OUT/importers.txt" || true)
} > "$OUT/paths.txt"

# How many call InvokeSelectedServiceCarrier (the Host-pinned carrier) outside tests?
{
  echo "== non-test references to InvokeSelectedServiceCarrier =="
  grep -rn --include='*.go' 'InvokeSelectedServiceCarrier' internal cmd tools reference 2>/dev/null \
    | grep -v '_test.go' || echo '  (none)'
  echo
  echo "== non-test references to InvokeWithProvenance =="
  grep -rn --include='*.go' 'InvokeWithProvenance' internal cmd tools reference 2>/dev/null \
    | grep -v '_test.go' || echo '  (none)'
} > "$OUT/entrypoints.txt"

# Counts.
{
  echo "== counts =="
  echo "non-test importers of plugin/runtime: $(find internal cmd tools reference examples -name '*.go' -type f 2>/dev/null | grep -v '_test.go' | xargs grep -l '"tei/internal/plugin/runtime"' 2>/dev/null | wc -l)"
  echo "among those, also importing servicehost: $(find internal cmd tools reference examples -name '*.go' -type f 2>/dev/null | grep -v '_test.go' | xargs grep -l '"tei/internal/plugin/runtime"' 2>/dev/null | xargs grep -l '"tei/internal/servicehost"' 2>/dev/null | wc -l)"
} > "$OUT/summary.txt"

cat "$OUT/summary.txt"
