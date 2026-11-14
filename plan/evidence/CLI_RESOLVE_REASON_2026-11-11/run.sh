#!/bin/bash
# The CLI reported every resolver failure as PLUGIN_RESOLVE_FAILED plus prose. The
# resolver now classifies itself (plugin.ResolveCode); this script shows that the
# CLI carries that reason alongside, without changing its own code or message.
set -eu

[ "$#" -ge 1 ] || { echo "usage: run.sh <outdir>" >&2; exit 2; }
OUT=$(mkdir -p "$1" && cd "$1" && pwd)
REPO=$(cd "$(dirname "$0")/../../.." && pwd)

export PATH=/usr/local/go/bin:/usr/bin:/bin:$PATH
cd "$REPO"

git rev-parse HEAD > "$OUT/head.txt"
go version >> "$OUT/head.txt"
git status --porcelain=v1 --untracked-files=all \
  | grep -v 'plan/evidence/CLI_RESOLVE_REASON_2026-11-11' > "$OUT/pre-status.txt" || true

{
  echo "== the new field on the CLI failure =="
  sed -n '/^type cliFailure struct/,/^}/p' cmd/tei/cli_result.go | sed 's/^/  /'
  echo ""
  echo "== the one helper both resolve call sites now use =="
  sed -n '/^func pluginResolveFailure/,/^}/p' cmd/tei/cli_result.go | sed 's/^/  /'
  echo ""
  echo "== the CLI codes are unchanged =="
  grep -n 'resultCodePluginResolveFailed = \|resultCodeInvalidPluginLock = ' cmd/tei/cli_result.go | sed 's/^/  /'
} > "$OUT/anchors.txt"

go test ./cmd/tei -run 'ResolveReason|PluginResolveFailure|TheReasonIsOmitted' -count=1 -v \
  > "$OUT/tests.txt" 2>&1 || true
grep -E '^(=== RUN|--- (PASS|FAIL)|ok|FAIL)' "$OUT/tests.txt" > "$OUT/tests-summary.txt" || true

cat "$OUT/anchors.txt" | sed 's/^/  /'
echo ""
cat "$OUT/tests-summary.txt" | sed 's/^/  /'
