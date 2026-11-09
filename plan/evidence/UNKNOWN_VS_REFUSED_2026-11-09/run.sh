#!/bin/bash
# CON-12d, the unknown half: a post-entry failure must not read as "not run".
#
# The CLI registers fixed providers. To produce an entering-then-failing
# provider from the shell, this script uses `go test` with a run filter that
# exercises the counterfactual, plus the normal CLI runs for the counts.
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
  | grep -v 'plan/evidence/UNKNOWN_VS_REFUSED_2026-11-09' > "$OUT/pre-status.txt" || true

# The counterfactuals: post-entry failure, pre-entry refusal, and both together.
go test ./tools/reference-observation-router-connect/... \
  -run 'PostEntry|PreEntry|UnknownAndRefused|UnknownIsInside' -v \
  > "$OUT/observation-tests.txt" 2>&1 || true
grep -E '^(=== RUN|--- (PASS|FAIL)|ok|FAIL)' "$OUT/observation-tests.txt" \
  > "$OUT/observation-summary.txt" || true

go test ./tools/reference-evidence-router-connect/... \
  -run 'PostEntry|PreEntry|UnknownAndRefused|UnknownIsInside' -v \
  > "$OUT/evidence-tests.txt" 2>&1 || true
grep -E '^(=== RUN|--- (PASS|FAIL)|ok|FAIL)' "$OUT/evidence-tests.txt" \
  > "$OUT/evidence-summary.txt" || true

# The CLI runs still show the pre-entry refusal counts.
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
lines.append('== CLI runs (pre-entry refusals only) ==')
for name, ok in (('obs-complete', 'observedCount'), ('obs-none', 'observedCount'),
                 ('ev-complete', 'collectedCount'), ('ev-none', 'collectedCount')):
    r = json.load(open(out / (name + '.json'), encoding='utf-8'))
    lines.append('  %-14s attempted=%d %s=%d refused=%d unknown=%d invoked=%d complete=%s' % (
        name, r['attemptedCount'], ok, r[ok], r['refusedCount'],
        r['unknownCount'], r['invokedCount'], r['complete']))
lines.append('')
lines.append('== counterfactual tests ==')
for name in ('observation', 'evidence'):
    path = out / (name + '-summary.txt')
    if path.exists():
        for line in path.read_text(encoding='utf-8').splitlines():
            if line.startswith('--- ') or line.startswith('ok '):
                lines.append('  %-10s %s' % (name, line))
(out / 'summary.txt').write_text('\n'.join(lines) + '\n', encoding='utf-8', newline='\n')
print('\n'.join(lines))
PY
