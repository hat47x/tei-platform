#!/bin/bash
# CON-12c names three identity fields: artifact digest, Host/Grant epoch, and
# Source scope. This script shows, for each, WHERE it is enforced (if anywhere),
# by running the scope/revision counterfactuals and listing the code anchors.
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
  | grep -v 'plan/evidence/CON12C_SCOPE_LAYERS_2026-11-09' > "$OUT/pre-status.txt" || true

# 1. The counterfactuals: scope mismatch, stale revision, matching scope.
go test ./tools/reference-service-carrier-connect/... \
  -run 'Scope|Revision' -v > "$OUT/tests.txt" 2>&1 || true
grep -E '^(=== RUN|--- (PASS|FAIL)|ok|FAIL)' "$OUT/tests.txt" > "$OUT/tests-summary.txt" || true

# 2. Where each identity field is compared, and whether a Grant epoch exists.
{
  echo "== where digest is compared =="
  grep -rn 'ArtifactDigest != ' internal/servicehost/ internal/composition/ 2>/dev/null || echo "  (none)"
  echo ""
  echo "== where scope is compared =="
  grep -rn 'Scope != s.Scope\|p.Scope != ' internal/composition/ 2>/dev/null || echo "  (none)"
  echo ""
  echo "== where registry epoch is compared =="
  grep -rn 'Epoch == 0\|epochs\[s.ProviderID\]' internal/composition/ 2>/dev/null || echo "  (none)"
  echo ""
  echo "== a Grant epoch anywhere in Go source =="
  grep -rn 'GrantEpoch' --include='*.go' internal/ cmd/ 2>/dev/null | grep -v '_test.go' || echo "  (none)"
} > "$OUT/anchors.txt"

cat "$OUT/tests-summary.txt" | sed 's/^/  /'
echo ""
cat "$OUT/anchors.txt" | sed 's/^/  /'
