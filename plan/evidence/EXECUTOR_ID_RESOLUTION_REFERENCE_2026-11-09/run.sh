#!/bin/bash
# C-3: resolve the deployment executor's CapabilityID from the Environment
# binding instead of naming it in code, then check the plugin lock admits it.
#
# Four cases:
#   1. binding + real lock            -> RESOLVED
#   2. binding, no lock               -> LOCK_REJECTED
#   3. no binding for the action      -> BINDING_MISSING
#   4. an action the binding does not
#      cover                          -> BINDING_MISSING
# The hard-coded id each reference implementation embeds is passed in so the
# report can show whether the two agree.
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
  | grep -v 'plan/evidence/EXECUTOR_ID_RESOLUTION_REFERENCE_2026-11-09' > "$OUT/pre-status.txt" || true

CMD=./tools/reference-executor-id-resolution/cmd/tei-executor-id-resolution
LOCK=reference/workspace/.tei/plugin-lock.json

# 1. Binding and the real lock: the resolved id is admitted and supports the action.
go run $CMD -workspace reference/workspace -environment development \
  -action database-release -database sales_core -plugin-lock "$LOCK" \
  -hard-coded-id migration.atlas > "$OUT/resolved.json" 2> "$OUT/resolved.err.txt"

# 2. Binding but no lock: admission cannot be established.
go run $CMD -workspace reference/workspace -environment development \
  -action database-release -database sales_core \
  -hard-coded-id migration.atlas > "$OUT/no-lock.json" 2> "$OUT/no-lock.err.txt"

# 3. A bound action whose executor differs from the hard-coded one: the
#    reference must follow the binding, not the code.
go run $CMD -workspace reference/workspace -environment development \
  -action application-retire -application order_entry -plugin-lock "$LOCK" \
  -hard-coded-id migration.atlas > "$OUT/other-bound-executor.json" 2> "$OUT/other-bound-executor.err.txt"

# 4. A database the binding does not name.
go run $CMD -workspace reference/workspace -environment development \
  -action database-release -database master_data -plugin-lock "$LOCK" \
  -hard-coded-id migration.atlas > "$OUT/wrong-database.json" 2> "$OUT/wrong-database.err.txt"

python3 - "$OUT" <<'PY'
import json
import pathlib
import sys

out = pathlib.Path(sys.argv[1])
lines = []
for name in ('resolved', 'no-lock', 'other-bound-executor', 'wrong-database'):
    r = json.load(open(out / (name + '.json'), encoding='utf-8'))
    lines.append('== %s ==' % name)
    lines.append('  status: %s  admitted: %s' % (r['status'], r['admitted']))
    lines.append('  resolvedId: %s' % (r.get('resolvedId') or '-'))
    lines.append('  hardCodedId: %s' % (r.get('hardCodedId') or '-'))
    if r.get('detail'):
        lines.append('  detail: %s' % r['detail'][:80])
    lines.append('')
r = json.load(open(out / 'resolved.json', encoding='utf-8'))
lines.append('ruleFingerprint: %s' % r['ruleFingerprint'])
(out / 'summary.txt').write_text('\n'.join(lines) + '\n', encoding='utf-8', newline='\n')
print('\n'.join(lines))
PY
