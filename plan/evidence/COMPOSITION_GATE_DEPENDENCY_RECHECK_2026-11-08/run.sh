#!/bin/bash
# Recheck the three dependency-shape claims the composition kernel gate records,
# plus the "no non-test caller of servicehost" claim, at the current main.
#
# The gate text says:
#   - A's cmd/tei does not include internal/composition
#   - B's execution unit does not include internal/operations
#   - internal/servicehost has no non-test importer
#
# Each claim is checked at package (closure) level, and the counterpart
# direction is reported too, so a reader can see both sides.
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
  | grep -v 'plan/evidence/COMPOSITION_GATE_DEPENDENCY_RECHECK_2026-11-08' > "$OUT/pre-status.txt" || true

RAW=$(mktemp -d)
go list -f '{{.ImportPath}} {{join .Deps " "}}' ./... > "$RAW/deps.txt" 2> "$OUT/go-list.err.txt" || true

# A: the CLI package. The gate's "A" is cmd/tei.
grep '^tei/cmd/tei ' "$RAW/deps.txt" > "$RAW/cmd-deps.txt" || true
{
  echo "== tei/cmd/tei =="
  echo "includes internal/composition: $(grep -c 'tei/internal/composition' "$RAW/cmd-deps.txt" || true)"
  echo "includes internal/operations: $(grep -c 'tei/internal/operations' "$RAW/cmd-deps.txt" || true)"
  echo "includes internal/servicehost: $(grep -c 'tei/internal/servicehost' "$RAW/cmd-deps.txt" || true)"
} > "$OUT/claim-a.txt"

# B: the host-owned execution unit. The gate's B is reference/web-composition/go-host.
grep '^tei/reference/web-composition/go-host ' "$RAW/deps.txt" > "$RAW/host-deps.txt" || true
{
  echo "== reference/web-composition/go-host =="
  echo "includes internal/operations: $(grep -c 'tei/internal/operations' "$RAW/host-deps.txt" || true)"
  echo "includes internal/composition: $(grep -c 'tei/internal/composition' "$RAW/host-deps.txt" || true)"
  echo "includes internal/servicehost: $(grep -c 'tei/internal/servicehost' "$RAW/host-deps.txt" || true)"
} > "$OUT/claim-b.txt"

# C: servicehost callers. Count packages whose closure includes it, excluding
# the package itself, and split test-only importers from non-test ones.
grep 'tei/internal/servicehost' "$RAW/deps.txt" | awk '{print $1}' | sort > "$RAW/servicehost-closure.txt" || true
grep -v '^tei/internal/servicehost$' "$RAW/servicehost-closure.txt" > "$OUT/servicehost-closure-callers.txt" || true
{
  echo "== packages whose closure includes internal/servicehost =="
  cat "$OUT/servicehost-closure-callers.txt"
  echo "count (excluding servicehost itself): $(wc -l < "$OUT/servicehost-closure-callers.txt")"
} > "$OUT/claim-c.txt"

# File-level check, matching how the gate stated it.
find internal cmd tools reference -name '*.go' -type f > "$RAW/files.txt" 2>/dev/null || true
: > "$OUT/servicehost-file-importers.txt"
: > "$OUT/servicehost-file-importers-nontest.txt"
while IFS= read -r f; do
  if grep -q '"tei/internal/servicehost"' "$f" 2>/dev/null; then
    echo "$f" >> "$OUT/servicehost-file-importers.txt"
    case "$f" in
      *_test.go) ;;
      *) echo "$f" >> "$OUT/servicehost-file-importers-nontest.txt" ;;
    esac
  fi
done < <(cat "$RAW/files.txt")
echo "file importers (all): $(wc -l < "$OUT/servicehost-file-importers.txt")" >> "$OUT/claim-c.txt"
echo "file importers (non-test): $(wc -l < "$OUT/servicehost-file-importers-nontest.txt")" >> "$OUT/claim-c.txt"

rm -rf "$RAW"
cat "$OUT/claim-a.txt" "$OUT/claim-b.txt" "$OUT/claim-c.txt"
