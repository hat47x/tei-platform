#!/bin/bash
# Re-run the composition co-location and servicehost callers survey at the
# current main. The earlier survey (COMPOSITION_NEXT_EDGE_STATIC_SURVEY_2026-11-07)
# is kept as history; this one asks whether its headline numbers still hold.
#
# Two different counts are reported and never mixed:
#   files   - a .go file whose text contains an import line
#   packages - a Go package whose dependency closure contains the package
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
  | grep -v 'plan/evidence/COMPOSITION_COLOCATION_RESURVEY_2026-11-08' > "$OUT/pre-status.txt" || true

# ---- file level ----
list_go() { find internal cmd tools reference -name '*.go' -type f 2>/dev/null; }
TOTAL_FILES=$(list_go | wc -l)

: > "$OUT/co-located-files.txt"
: > "$OUT/servicehost-importers.txt"
: > "$OUT/operations-importers.txt"
while IFS= read -r f; do
  if grep -q '"tei/internal/operations"' "$f" 2>/dev/null; then
    echo "$f" >> "$OUT/operations-importers.txt"
    if grep -q '"tei/internal/composition"' "$f" 2>/dev/null; then
      echo "$f" >> "$OUT/co-located-files.txt"
    fi
  fi
done < <(list_go)
while IFS= read -r f; do
  case "$f" in *_test.go) continue ;; esac
  grep -q '"tei/internal/servicehost"' "$f" 2>/dev/null && echo "$f" >> "$OUT/servicehost-importers.txt"
done < <(list_go)

# ---- package level ----
# The raw dependency dump is about 1 MB. Keep it outside the evidence directory
# and publish only the compact lists derived from it.
RAW=$(mktemp -d)
go list -f '{{.ImportPath}} {{join .Deps " "}}' ./... > "$RAW/pkg-deps.txt" 2> "$OUT/pkg-deps.err.txt" || true
go list -f '{{if eq .Name "main"}}{{.ImportPath}}{{end}}' ./... > "$OUT/main-packages.txt" 2>/dev/null || true
grep -c . "$OUT/main-packages.txt" > "$OUT/main-count.txt" || echo 0 > "$OUT/main-count.txt"

grep 'tei/internal/composition' "$RAW/pkg-deps.txt" | awk '{print $1}' | sort > "$OUT/pkgs-composition.txt" || true
grep 'tei/internal/operations' "$RAW/pkg-deps.txt" | awk '{print $1}' | sort > "$OUT/pkgs-operations.txt" || true
grep 'tei/internal/servicehost' "$RAW/pkg-deps.txt" | awk '{print $1}' | sort > "$OUT/pkgs-servicehost.txt" || true
sort -u "$OUT/main-packages.txt" | grep -v '^$' > "$OUT/main-packages-sorted.txt"

# Intersections use join on sorted files so blank lines can never shift a match.
join -j 1 "$OUT/pkgs-composition.txt" "$OUT/pkgs-operations.txt" | awk '{print $1}' > "$OUT/pkgs-both.txt"
join -j 1 "$OUT/pkgs-both.txt" "$OUT/main-packages-sorted.txt" | awk '{print $1}' > "$OUT/main-pkgs-both.txt"
join -j 1 "$OUT/pkgs-composition.txt" "$OUT/main-packages-sorted.txt" | awk '{print $1}' > "$OUT/main-pkgs-composition.txt"
join -j 1 "$OUT/pkgs-operations.txt" "$OUT/main-packages-sorted.txt" | awk '{print $1}' > "$OUT/main-pkgs-operations.txt"

{
  echo "== file level =="
  echo "go files scanned: $TOTAL_FILES"
  echo "files importing operations: $(wc -l < "$OUT/operations-importers.txt")"
  echo "files importing both operations and composition: $(wc -l < "$OUT/co-located-files.txt")"
  echo "non-test files importing servicehost: $(wc -l < "$OUT/servicehost-importers.txt")"
  echo
  echo "== package level =="
  echo "main packages: $(cat "$OUT/main-count.txt")"
  echo "packages whose closure includes composition: $(wc -l < "$OUT/pkgs-composition.txt")"
  echo "packages whose closure includes operations: $(wc -l < "$OUT/pkgs-operations.txt")"
  echo "packages whose closure includes servicehost: $(wc -l < "$OUT/pkgs-servicehost.txt")"
  echo "packages whose closure includes BOTH composition and operations: $(wc -l < "$OUT/pkgs-both.txt")"
  echo "main packages among those: $(wc -l < "$OUT/main-pkgs-both.txt")"
  echo "main packages whose closure includes composition: $(wc -l < "$OUT/main-pkgs-composition.txt")"
  echo "main packages whose closure includes operations: $(wc -l < "$OUT/main-pkgs-operations.txt")"
} > "$OUT/summary.txt"

rm -rf "$RAW"
cat "$OUT/summary.txt"
