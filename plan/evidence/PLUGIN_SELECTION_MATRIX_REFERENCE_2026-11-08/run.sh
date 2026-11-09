#!/bin/bash
set -eu
export PATH=/usr/local/go/bin:/usr/bin:/bin
export GIT_DIR=/mnt/d/GIT/hat47x/tei-platform/.git/worktrees/tei-wt-matrix-20261108
export GIT_WORK_TREE=/mnt/d/GIT/hat47x/tei-wt-matrix-20261108
cd "$GIT_WORK_TREE"
OUT=plan/evidence/PLUGIN_SELECTION_MATRIX_REFERENCE_2026-11-08/raw
rm -rf "$OUT"; mkdir -p "$OUT"
git rev-parse HEAD > "$OUT/head.txt"; go version >> "$OUT/head.txt"
git status --porcelain=v1 --untracked-files=all | grep -v 'plan/evidence/PLUGIN_SELECTION_MATRIX_REFERENCE' > "$OUT/pre-status.txt" || true
go run ./tools/reference-plugin-selection-matrix/cmd/tei-plugin-selection-matrix -workspace reference/workspace > "$OUT/matrix.json" 2> "$OUT/matrix.err.txt"
python3 - "$OUT" <<'PY'
import json,pathlib,sys
out=pathlib.Path(sys.argv[1]); lines=[]
r=json.load(open(out/'matrix.json',encoding='utf-8'))
lines.append('environments: %s' % r['environments'])
lines.append('selections: %d missing: %d ambiguous: %d' % (len(r['selections']), r['missingCount'], r['ambiguousCount']))
for s in r['selections']:
    caps=','.join('%s/%s'%(c['plugin'],c['id']) for c in s['capabilities']) or '-'
    lines.append('  %-9s %-14s %-22s %-24s %s' % (s['verdict'], s['environment'], s['form'], s['binding'], caps))
lines.append('')
lines.append('ruleFingerprint: %s' % r['ruleFingerprint'])
(out/'summary.txt').write_text('\n'.join(lines)+'\n',encoding='utf-8',newline='\n')
print('\n'.join(lines))
PY