#!/bin/bash
set -eu
export PATH=/usr/local/go/bin:/usr/bin:/bin
export GIT_DIR=/mnt/d/GIT/hat47x/tei-platform/.git/worktrees/tei-wt-cover-20261108
export GIT_WORK_TREE=/mnt/d/GIT/hat47x/tei-wt-cover-20261108
cd "$GIT_WORK_TREE"
OUT=plan/evidence/PLUGIN_CAPABILITY_COVERAGE_REFERENCE_2026-11-08/raw
rm -rf "$OUT"; mkdir -p "$OUT"
git rev-parse HEAD > "$OUT/head.txt"; go version >> "$OUT/head.txt"
git status --porcelain=v1 --untracked-files=all | grep -v 'plan/evidence/PLUGIN_CAPABILITY_COVERAGE_REFERENCE' > "$OUT/pre-status.txt" || true
go run ./tools/reference-plugin-capability-coverage/cmd/tei-plugin-capability-coverage -workspace reference/workspace > "$OUT/reference-coverage.json" 2> "$OUT/reference-coverage.err.txt"
go run ./cmd/tei plugin-lock -workspace reference/workspace > "$OUT/reference-plugin-lock.json" 2> "$OUT/reference-plugin-lock.err.txt"
python3 - "$OUT" <<'PY'
import json,pathlib,sys
out=pathlib.Path(sys.argv[1]); lines=[]
r=json.load(open(out/'reference-coverage.json',encoding='utf-8'))
lines.append('environments: %s' % r['environments'])
lines.append('declared capabilities: %d' % len(r['capabilities']))
lines.append('unselected: %d' % r['unselectedCount'])
for c in r['capabilities']:
    lines.append('  %-9s %-30s %-22s %-22s %s' % (
        c['status'], c['plugin'], c['kind'], c['id'], ','.join(c.get('environments') or []) or '-'))
lock=json.load(open(out/'reference-plugin-lock.json',encoding='utf-8'))
envs=lock['result']['environments']
lines.append('')
lines.append('plugin-lock environments: %s' % [e['environment'] for e in envs])
for e in envs:
    lines.append('  %s plugins=%d' % (e['environment'], len(e['plugins'])))
(out/'summary.txt').write_text('\n'.join(lines)+'\n',encoding='utf-8',newline='\n')
print('\n'.join(lines))
PY