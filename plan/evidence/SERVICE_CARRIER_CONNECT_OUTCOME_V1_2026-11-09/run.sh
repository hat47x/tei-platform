#!/bin/bash
# C-2 report v1: separate pre-invocation refusal from post-invocation
# uncertainty. This reproduces the counterfactual tests and the CLI runs.
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
  | grep -v 'plan/evidence/SERVICE_CARRIER_CONNECT_OUTCOME_V1_2026-11-09' > "$OUT/pre-status.txt" || true

# 1. The counterfactual tests: pre-invocation refusal, post-entry failure,
#    artifact refusal inside the carrier, and a route for another Binding.
go test ./tools/reference-service-carrier-connect/... -count=1 \
  -run 'Effect|Unverified|ArtifactRefusal|Route' -v > "$OUT/counterfactuals.txt" 2>&1 || true
grep -E '^(=== RUN|--- (PASS|FAIL)|ok|FAIL)' "$OUT/counterfactuals.txt" > "$OUT/counterfactuals-summary.txt" || true

# 2. The CLI runs, showing the counters for the three Host conditions.
CMD=./tools/reference-service-carrier-connect/cmd/tei-service-carrier-connect
go run $CMD -application order_entry -environment development \
  > "$OUT/route-present.json" 2> "$OUT/route-present.err.txt"
set +e
go run $CMD -application order_entry -environment development -route-unavailable \
  > "$OUT/route-missing.json" 2> "$OUT/route-missing.err.txt"
echo "route-missing exit=$?" > "$OUT/route-missing.exit.txt"
set -e
go run $CMD -unconfigured > "$OUT/inputs-missing.json" 2> "$OUT/inputs-missing.err.txt"

python3 - "$OUT" <<'PY'
import json
import pathlib
import sys

out = pathlib.Path(sys.argv[1])
lines = []
for name in ('route-present', 'route-missing', 'inputs-missing'):
    report = json.load(open(out / (name + '.json'), encoding='utf-8'))
    lines.append('== %s ==' % name)
    lines.append('  ran=%d refused=%d unverified=%d' % (
        report['ranCount'], report['refusedCount'], report['unverifiedCount']))
    for o in report['outcomes']:
        lines.append('    request=%s generation=%s pinned=%s carrierInvoked=%s%s' % (
            o['requestId'], o['generation'], o['pinnedGeneration'],
            o.get('carrierInvoked', False),
            (' error=' + o['error'][:60]) if o.get('error') else ''))
    lines.append('')
present = json.load(open(out / 'route-present.json', encoding='utf-8'))
lines.append('ruleFingerprint: %s' % present['ruleFingerprint'])
(out / 'summary.txt').write_text('\n'.join(lines) + '\n', encoding='utf-8', newline='\n')
print('\n'.join(lines))
PY
