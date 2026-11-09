#!/bin/bash
# EvidenceProviders: does the same four-stage shape hold as Observation?
#   1. provider registered   -> COLLECTED, pass carried through
#   2. nothing registered    -> NOT_REGISTERED, resolved stays true
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
  | grep -v 'plan/evidence/EVIDENCE_ROUTER_CONNECT_REFERENCE_2026-11-09' > "$OUT/pre-status.txt" || true

CMD=./tools/reference-evidence-router-connect/cmd/tei-evidence-router-connect

go run $CMD -workspace reference/workspace -environment development \
  > "$OUT/registered.json" 2> "$OUT/registered.err.txt"

go run $CMD -workspace reference/workspace -environment development -register-none \
  > "$OUT/unregistered.json" 2> "$OUT/unregistered.err.txt"

python3 - "$OUT" <<'PY'
import json
import pathlib
import sys

out = pathlib.Path(sys.argv[1])
lines = []
for name in ('registered', 'unregistered'):
    r = json.load(open(out / (name + '.json'), encoding='utf-8'))
    lines.append('== %s ==' % name)
    lines.append('  collected=%d refused=%d invoked=%d' % (
        r['collectedCount'], r['refusedCount'], r['invokedCount']))
    for o in r['outcomes']:
        lines.append('    action=%-16s provider=%-18s status=%-16s resolved=%s selected=%s invoked=%s outcome=%s' % (
            o['actionId'], o.get('providerId') or '-', o['status'],
            o['resolved'], o['selected'], o['invoked'], o.get('evidenceOutcome') or '-'))
    lines.append('')
present = json.load(open(out / 'registered.json', encoding='utf-8'))
lines.append('ruleFingerprint: %s' % present['ruleFingerprint'])
(out / 'summary.txt').write_text('\n'.join(lines) + '\n', encoding='utf-8', newline='\n')
print('\n'.join(lines))
PY
