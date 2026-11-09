#!/bin/bash
# C-2: a Host guard refusal happens BEFORE the carrier is entered. It must be a
# refusal, not an invocation. This script runs the counterfactual tests and the
# CLI cases.
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
  | grep -v 'plan/evidence/CARRIER_GUARD_REFUSAL_2026-11-09' > "$OUT/pre-status.txt" || true

go test ./tools/reference-service-carrier-connect/... \
  -run 'Guard|Route' -v > "$OUT/tests.txt" 2>&1 || true
grep -E '^(=== RUN|--- (PASS|FAIL)|ok|FAIL)' "$OUT/tests.txt" > "$OUT/tests-summary.txt" || true

CMD=./tools/reference-service-carrier-connect/cmd/tei-service-carrier-connect
go run $CMD -application order_entry -environment development \
  > "$OUT/route-present.json" 2> "$OUT/route-present.err.txt"
go run $CMD -application order_entry -environment development -route-unavailable \
  > "$OUT/route-missing.json" 2> "$OUT/route-missing.err.txt"

python3 - "$OUT" <<'PY'
import json
import pathlib
import sys

out = pathlib.Path(sys.argv[1])
lines = ['== counterfactual tests ==']
for line in (out / 'tests-summary.txt').read_text(encoding='utf-8').splitlines():
    if line.startswith('--- ') or line.startswith('ok '):
        lines.append('  ' + line)
lines.append('')
lines.append('== CLI runs ==')
for name in ('route-present', 'route-missing'):
    r = json.load(open(out / (name + '.json'), encoding='utf-8'))
    lines.append('  %-14s ran=%d refused=%d unverified=%d' % (
        name, r['ranCount'], r['refusedCount'], r['unverifiedCount']))
    for o in r['outcomes']:
        lines.append('    carrierInvoked=%s' % o.get('carrierInvoked', False))
(out / 'summary.txt').write_text('\n'.join(lines) + '\n', encoding='utf-8', newline='\n')
print('\n'.join(lines))
PY
