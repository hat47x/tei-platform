#!/bin/bash
# Resolve an environment's deployment-executor binding and dispatch through it.
#
# The reference registers a recorder executor, so this observes the path from
# binding to execution without touching an external system.
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
  | grep -v 'plan/evidence/BINDING_TO_EXECUTION_REFERENCE_2026-11-09' > "$OUT/pre-status.txt" || true

CMD=./tools/reference-binding-to-execution/cmd/tei-binding-to-execution

# 1. A bound action: the environment names migration.atlas for database-release.
go run $CMD -workspace reference/workspace -environment development \
  -action-id db/apply -kind database-release -database sales_core \
  > "$OUT/bound.json" 2> "$OUT/bound.err.txt"

# 2. An action the environment binds to an executor this Host did not register.
set +e
go run $CMD -workspace reference/workspace -environment development \
  -action-id app/start -kind application-start -application order_entry \
  > "$OUT/unregistered.json" 2> "$OUT/unregistered.err.txt"
echo "unregistered exit=$?" > "$OUT/unregistered.exit.txt"
set -e

# 3. An environment that does not exist.
go run $CMD -workspace reference/workspace -environment staging \
  -action-id db/apply -kind database-release -database sales_core \
  > "$OUT/unknown-env.json" 2> "$OUT/unknown-env.err.txt"

python3 - "$OUT" <<'PY'
import json
import pathlib
import sys

out = pathlib.Path(sys.argv[1])
lines = []
for name in ('bound', 'unregistered', 'unknown-env'):
    report = json.load(open(out / (name + '.json'), encoding='utf-8'))
    lines.append('== %s ==' % name)
    lines.append('  environment: %s' % report['environment'])
    lines.append('  ran: %d  refused: %d' % (report['ranCount'], report['refusedCount']))
    for o in report['outcomes']:
        lines.append('    action=%s executor=%s bindingResolved=%s%s' % (
            o['actionId'], o.get('executorId') or '-', o['bindingResolved'],
            (' error=' + o['error'][:60]) if o.get('error') else ''))
    lines.append('')
bound = json.load(open(out / 'bound.json', encoding='utf-8'))
lines.append('ruleFingerprint: %s' % bound['ruleFingerprint'])
(out / 'summary.txt').write_text('\n'.join(lines) + '\n', encoding='utf-8', newline='\n')
print('\n'.join(lines))
PY
