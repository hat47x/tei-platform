#!/bin/bash
# Does the workspace validation check every enabled feature, or only some?
#
# Three copies of the accepted Workspace differ only in which editor feature is
# enabled, with no plugin capability that supports it in either case.
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
  | grep -v 'plan/evidence/FEATURE_SELECTION_CHECK_ASYMMETRY_2026-11-08' > "$OUT/pre-status.txt" || true

W=$(mktemp -d)
trap 'rm -rf "$W"' EXIT
for name in sql-only workbench-only accepted; do
  cp -r reference/workspace "$W/$name"
done

python3 - "$W" <<'PY'
import io
import pathlib
import sys

w = pathlib.Path(sys.argv[1])

def rewrite(rel, keep_feature, drop_plugin):
    p = w / rel
    t = io.open(p, encoding='utf-8', newline='').read()
    if drop_plugin:
        t = t.replace('    plugin tei.editor.intelligence.sql ^1\n', '')
    for feature in ('editor.sql_analysis', 'editor.database_workbench'):
        if keep_feature and feature != keep_feature:
            t = t.replace('    feature %s\n' % feature, '')
    io.open(p, 'w', encoding='utf-8', newline='').write(t)

for env in ('development', 'production'):
    rel = 'sql-only/environments/%s/environment.tei' % env
    rewrite(rel, 'editor.sql_analysis', True)
    rel = 'workbench-only/environments/%s/environment.tei' % env
    rewrite(rel, 'editor.database_workbench', True)
    rel = 'accepted/environments/%s/environment.tei' % env
    rewrite(rel, None, False)
print('fixtures ready')
PY

for name in sql-only workbench-only accepted; do
  set +e
  go run ./cmd/tei validate -workspace "$W/$name" > "$OUT/$name.out.txt" 2> "$OUT/$name.err.txt"
  echo "$name exit=$?" >> "$OUT/exit-codes.txt"
  set -e
done

python3 - "$OUT" <<'PY'
import pathlib
import sys

out = pathlib.Path(sys.argv[1])
lines = []
for name in ('sql-only', 'workbench-only', 'accepted'):
    result = (out / (name + '.out.txt')).read_text(encoding='utf-8')
    code = [l for l in (out / 'exit-codes.txt').read_text(encoding='utf-8').splitlines() if l.startswith(name)][0]
    count = sum(1 for l in result.splitlines() if 'ENVIRONMENT_FEATURE_PLUGIN' in l)
    lines.append('== %s ==' % name)
    lines.append('  %s' % code)
    lines.append('  ENVIRONMENT_FEATURE_PLUGIN diagnostics: %d' % count)
    for l in result.splitlines():
        if 'ENVIRONMENT_FEATURE_PLUGIN' in l:
            lines.append('    %s' % l.strip())
    lines.append('')
(out / 'summary.txt').write_text('\n'.join(lines) + '\n', encoding='utf-8', newline='\n')
print('\n'.join(lines))
PY
