#!/bin/bash
# TestParallelClaim expects exactly one winner and one ErrAlreadyClaimed. Before
# this change the loser saw SQLITE_BUSY instead, so the test failed on every run
# on a clean main. This script reproduces the old behaviour, then shows the fix.
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
  | grep -v 'plan/evidence/DISPATCHER_LEASE_BUSY_2026-11-11' > "$OUT/pre-status.txt" || true

# After the fix: five consecutive runs of the whole tool.
for i in 1 2 3 4 5; do
  go test ./tools/reference-external-role-dispatcher/internal -run 'TestParallelClaim' -count=1 \
    > "$OUT/after-$i.txt" 2>&1 || true
  echo "run $i: $(grep -e 'ok' -e 'FAIL' "$OUT/after-$i.txt" | head -1)" >> "$OUT/after.txt"
done

# Re-prove: remove the two store properties and watch the test fail again.
cp tools/reference-external-role-dispatcher/internal/lease.go /tmp/lease.backup
python3 - > "$OUT/before.txt" 2>&1 <<'PY' || true
import io
p = 'tools/reference-external-role-dispatcher/internal/lease.go'
t = io.open(p, encoding='utf-8', newline='').read()
for old in ('\tdb.SetMaxOpenConns(1)\n', '\tif _, err := db.Exec(`PRAGMA busy_timeout=5000;`); err != nil {\n\t\treturn nil, fmt.Errorf("set busy timeout: %w", err)\n\t}\n'):
    if old not in t:
        print('anchor not found; cannot re-prove')
        raise SystemExit(0)
    t = t.replace(old, '', 1)
io.open(p, 'w', encoding='utf-8', newline='').write(t)
print('removed the single connection and the busy timeout')
PY
go test ./tools/reference-external-role-dispatcher/internal -run 'TestParallelClaim' -count=1 \
  >> "$OUT/before.txt" 2>&1 || true
cp /tmp/lease.backup tools/reference-external-role-dispatcher/internal/lease.go
echo "restored lease.go" >> "$OUT/before.txt"

cat "$OUT/after.txt" | sed 's/^/  /'
echo ""
cat "$OUT/before.txt" | sed 's/^/  /'
