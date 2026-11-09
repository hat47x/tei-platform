#!/bin/bash
set -eu
export PATH=/usr/local/go/bin:/usr/bin:/bin
export GIT_DIR=/mnt/d/GIT/hat47x/tei-platform/.git/worktrees/tei-wt-driftref-20261107
export GIT_WORK_TREE=/mnt/d/GIT/hat47x/tei-wt-driftref-20261107
cd "$GIT_WORK_TREE"
OUT=plan/evidence/OPERATION_DRIFT_CHECK_REFERENCE_2026-11-07/raw
rm -rf "$OUT"; mkdir -p "$OUT"
git rev-parse HEAD > "$OUT/head.txt"; go version >> "$OUT/head.txt"
git status --porcelain=v1 --untracked-files=all | grep -v 'plan/evidence/OPERATION_DRIFT_CHECK_REFERENCE' > "$OUT/pre-status.txt" || true
W=$(mktemp -d); trap 'rm -rf "$W"' EXIT
cp -r reference/standard-crud/workspace "$W/before"
cp -r reference/standard-crud/workspace "$W/after"
python3 - "$W/after/semantic/sales/order.tei" <<'PY'
import io,sys
p=sys.argv[1]; t=io.open(p,encoding='utf-8',newline='').read()
if 'attribute note type std.text' not in t:
    lines=t.split('\n')
    for i,l in enumerate(lines):
        if 'attribute status' in l:
            lines.insert(i+1,'    attribute note type std.text'); break
    io.open(p,'w',encoding='utf-8',newline='').write('\n'.join(lines))
print('attribute added')
PY
# Capture the fingerprints of the BEFORE workspace to use as the caller's stored expectation.
go run ./cmd/tei operations -workspace "$W/before" -application ledger > "$OUT/before-operations.json" 2>"$OUT/before-operations.err.txt"
python3 - "$OUT" "$W" <<'PY'
import json,pathlib,subprocess,sys
out=pathlib.Path(sys.argv[1]); work=pathlib.Path(sys.argv[2])
before=json.load(open(out/'before-operations.json',encoding='utf-8'))
open(out/'stored-expectation.txt','w',encoding='utf-8',newline='\n').write(
    'worldFingerprint=%s\nmodelRevision=%s\n' % (before['worldFingerprint'], before['modelRevision']))
PY
CMD=./tools/reference-operation-drift-check/cmd/tei-operation-drift-check
# Same workspace, same expectation -> current (exit 0)
go run $CMD -workspace "$W/before" -application ledger \
  -expected-world-fingerprint "$(grep worldFingerprint "$OUT/stored-expectation.txt" | cut -d= -f2)" \
  -expected-model-revision "$(grep modelRevision "$OUT/stored-expectation.txt" | cut -d= -f2)" \
  > "$OUT/same-workspace.json" 2>"$OUT/same-workspace.err.txt"; echo "exit=$?" >> "$OUT/same-workspace.err.txt"
# Changed workspace, same expectation -> stale (exit 1)
set +e
go run $CMD -workspace "$W/after" -application ledger \
  -expected-world-fingerprint "$(grep worldFingerprint "$OUT/stored-expectation.txt" | cut -d= -f2)" \
  -expected-model-revision "$(grep modelRevision "$OUT/stored-expectation.txt" | cut -d= -f2)" \
  > "$OUT/changed-workspace.json" 2>"$OUT/changed-workspace.err.txt"
echo "exit=$?" > "$OUT/changed-workspace.exit.txt"
set -e
# No expectation -> unchecked (exit 0)
go run $CMD -workspace "$W/before" -application ledger > "$OUT/no-expectation.json" 2>"$OUT/no-expectation.err.txt"
python3 - "$OUT" <<'PY'
import json,pathlib,sys
out=pathlib.Path(sys.argv[1]); lines=[]
for name in ('same-workspace','changed-workspace','no-expectation'):
    r=json.load(open(out/(name+'.json'),encoding='utf-8'))
    lines.append('== %s ==' % name)
    lines.append('  status=%s diagnostic=%s' % (r['status'], r.get('diagnostic','')))
    for f in r['findings']:
        lines.append('    %s match=%s' % (f['value'], f['match']))
    lines.append('')
lines.append('changed-workspace exit code: %s' % (out/'changed-workspace.exit.txt').read_text().strip())
(out/'summary.txt').write_text('\n'.join(lines)+'\n',encoding='utf-8',newline='\n')
print('\n'.join(lines))
PY