#!/bin/bash
# CON-12d: a partial run must be distinguishable from a complete one.
# Three cases, both references:
#   1. every declared provider registered -> complete
#   2. only one of two registered         -> complete=false (observation)
#   3. nothing registered                 -> complete=false
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
  | grep -v 'plan/evidence/PARTIAL_RUN_VERDICT_2026-11-09' > "$OUT/pre-status.txt" || true

OBS=./tools/reference-observation-router-connect/cmd/tei-observation-router-connect
EV=./tools/reference-evidence-router-connect/cmd/tei-evidence-router-connect

go run $OBS -workspace reference/workspace -environment development \
  > "$OUT/obs-complete.json" 2> "$OUT/obs-complete.err.txt"
go run $OBS -workspace reference/workspace -environment development -register-none \
  > "$OUT/obs-none.json" 2> "$OUT/obs-none.err.txt"
go run $EV -workspace reference/workspace -environment development \
  > "$OUT/ev-complete.json" 2> "$OUT/ev-complete.err.txt"
go run $EV -workspace reference/workspace -environment development -register-none \
  > "$OUT/ev-none.json" 2> "$OUT/ev-none.err.txt"

python3 - "$OUT" <<'PY'
import json
import pathlib
import sys

out = pathlib.Path(sys.argv[1])
lines = []
for name, ok_key in (('obs-complete', 'observedCount'), ('obs-none', 'observedCount'),
                     ('ev-complete', 'collectedCount'), ('ev-none', 'collectedCount')):
    r = json.load(open(out / (name + '.json'), encoding='utf-8'))
    lines.append('== %s ==' % name)
    lines.append('  attempted=%d %s=%d refused=%d invoked=%d complete=%s' % (
        r['attemptedCount'], ok_key, r[ok_key], r['refusedCount'],
        r['invokedCount'], r['complete']))
    lines.append('')
present = json.load(open(out / 'obs-complete.json', encoding='utf-8'))
lines.append('ruleFingerprint: %s' % present['ruleFingerprint'])
(out / 'summary.txt').write_text('\n'.join(lines) + '\n', encoding='utf-8', newline='\n')
print('\n'.join(lines))
PY
