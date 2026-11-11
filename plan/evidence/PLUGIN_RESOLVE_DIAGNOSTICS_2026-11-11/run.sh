#!/bin/bash
# The resolver refused with plain fmt.Errorf text, so a caller could only branch
# on the message. This script shows the codes now attached, then proves that the
# message text did not change by running the pre-existing tests that match it.
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
  | grep -v 'plan/evidence/PLUGIN_RESOLVE_DIAGNOSTICS_2026-11-11' > "$OUT/pre-status.txt" || true

{
  echo "== the codes the resolver can now report =="
  grep -n 'Diagnostic[A-Za-z]* = "' internal/plugin/diagnostics.go | sed 's/^/  /'
  echo ""
  echo "== remaining plain fmt.Errorf in the resolver (should be none in resolve.go) =="
  count=$(grep -c 'fmt.Errorf' internal/plugin/resolve.go || true)
  echo "  resolve.go: $count"
  echo ""
  echo "== how many existing tests branch on the message text =="
  grep -rc 'strings.Contains(err.Error()' internal/plugin/*_test.go | grep -v ':0' | sed 's/^/  /'
  echo ""
} > "$OUT/anchors.txt"

go test ./internal/plugin/... -count=1 > "$OUT/tests.txt" 2>&1 || true
grep -E '^(ok|FAIL|---)' "$OUT/tests.txt" > "$OUT/tests-summary.txt" || true
go test ./internal/plugin -run 'ResolveDiagnostics|Cycle|Ordinary|Duplicate|Invalid' -count=1 -v \
  > "$OUT/new-tests.txt" 2>&1 || true
grep -E '^(=== RUN|--- |ok|FAIL)' "$OUT/new-tests.txt" > "$OUT/new-tests-summary.txt" || true

cat "$OUT/anchors.txt" | sed 's/^/  /'
echo ""
cat "$OUT/new-tests-summary.txt" | sed 's/^/  /'
echo ""
cat "$OUT/tests-summary.txt" | sed 's/^/  /'
