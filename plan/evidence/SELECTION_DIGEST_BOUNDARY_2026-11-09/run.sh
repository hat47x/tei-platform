#!/bin/bash
# CON-12c on the ServiceProviderSelection layer.
#
# servicehost compares route.Selection against the pinned Binding before the
# Provider is called. Three cases, each a distinct counter pattern:
#   1. selection digest differs   -> ran=1 unverified=1 refused=0, Provider 0
#   2. selection contract differs -> ran=1 unverified=1 refused=0, Provider 0
#   3. selection matches          -> ran=1, Provider 1
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
  | grep -v 'plan/evidence/SELECTION_DIGEST_BOUNDARY_2026-11-09' > "$OUT/pre-status.txt" || true

go test ./tools/reference-service-carrier-connect/... \
  -run 'Selection' -v > "$OUT/tests.txt" 2>&1 || true
grep -E '^(=== RUN|--- (PASS|FAIL)|ok|FAIL)' "$OUT/tests.txt" > "$OUT/tests-summary.txt" || true

cat "$OUT/tests-summary.txt" | sed 's/^/  /'
