#!/bin/bash
# Build the C-2 connect point outside a test and carry requests through it.
#
# The Host-owned inputs are supplied by the CLI below with a reference-only
# process, verifier and acceptance reader: no external effect is performed.
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
  | grep -v 'plan/evidence/SERVICE_CARRIER_CONNECT_REFERENCE_2026-11-09' > "$OUT/pre-status.txt" || true

CMD=./tools/reference-service-carrier-connect/cmd/tei-service-carrier-connect

# 1. A Host that has a route for the pinned generation.
go run $CMD -application order_entry -environment development \
  > "$OUT/route-present.json" 2> "$OUT/route-present.err.txt"

# 2. A Host whose route source is unavailable.
set +e
go run $CMD -application order_entry -environment development -route-unavailable \
  > "$OUT/route-missing.json" 2> "$OUT/route-missing.err.txt"
echo "route-missing exit=$?" > "$OUT/route-missing.exit.txt"
set -e

# 3. No pinned Binding at all (the Host supplied nothing).
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
    lines.append('  ran: %d  refused: %d' % (report['ranCount'], report['refusedCount']))
    for o in report['outcomes']:
        lines.append('    request=%s generation=%s pinned=%s%s' % (
            o['requestId'], o['generation'], o['pinnedGeneration'],
            (' error=' + o['error'][:70]) if o.get('error') else ''))
    lines.append('')
present = json.load(open(out / 'route-present.json', encoding='utf-8'))
lines.append('ruleFingerprint: %s' % present['ruleFingerprint'])
(out / 'summary.txt').write_text('\n'.join(lines) + '\n', encoding='utf-8', newline='\n')
print('\n'.join(lines))
PY
