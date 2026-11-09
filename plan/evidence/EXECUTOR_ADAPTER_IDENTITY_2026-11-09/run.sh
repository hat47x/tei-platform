#!/bin/bash
# The DeploymentExecutor adapter used to overwrite the executor's identity
# unconditionally, which made routing.ValidateResult unable to reject a result
# from another executor. This script runs the regression tests and re-proves the
# failure against the old behaviour.
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
  | grep -v 'plan/evidence/EXECUTOR_ADAPTER_IDENTITY_2026-11-09' > "$OUT/pre-status.txt" || true

go test ./internal/plugin/runtime -run 'ExecutorIdentity' -count=1 -v \
  > "$OUT/tests.txt" 2>&1 || true
grep -E '^(=== RUN|--- (PASS|FAIL)|ok|FAIL)' "$OUT/tests.txt" > "$OUT/tests-summary.txt" || true

cp internal/plugin/runtime/adapters.go /tmp/adapters.ea.backup
python3 - > "$OUT/old-behaviour.txt" 2>&1 <<'PY' || true
import io
p = 'internal/plugin/runtime/adapters.go'
t = io.open(p, encoding='utf-8', newline='').read()
old = '\tif out.ExecutorID == "" {\n\t\tout.ExecutorID = string(a.CapabilityID)\n\t}\n'
if old not in t:
    print('conditional not found; cannot re-prove')
    raise SystemExit(0)
t = t.replace(old, '\tout.ExecutorID = string(a.CapabilityID)\n', 1)
io.open(p, 'w', encoding='utf-8', newline='').write(t)
print('reverted adapters.go to the old unconditional overwrite')
PY
go test ./internal/plugin/runtime -run 'ForeignExecutorIdentity' -count=1 \
  >> "$OUT/old-behaviour.txt" 2>&1 || true
cp /tmp/adapters.ea.backup internal/plugin/runtime/adapters.go
echo "restored adapters.go" >> "$OUT/old-behaviour.txt"

cat "$OUT/tests-summary.txt" | sed 's/^/  /'
echo ""
cat "$OUT/old-behaviour.txt" | sed 's/^/  /'
