#!/bin/bash
# The rule text says "list and read are unattended because they have no effect".
# Nothing enforced that. This script runs the invariant test, then proves the
# test catches a violation by marking an effectful kind unattended.
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
  | grep -v 'plan/evidence/UNATTENDED_AUTONOMY_INVARIANT_2026-11-10' > "$OUT/pre-status.txt" || true

go test ./internal/operations -run 'Unattended' -count=1 -v \
  > "$OUT/tests.txt" 2>&1 || true
grep -E '^(=== RUN|--- (PASS|FAIL)|ok|FAIL)' "$OUT/tests.txt" > "$OUT/tests-summary.txt" || true

# Re-prove: mark an effectful kind unattended and watch the invariant fail.
cp internal/operations/surface.go /tmp/surface.unat.backup
python3 - > "$OUT/violation.txt" 2>&1 <<'PY' || true
import io, re
p = 'internal/operations/surface.go'
t = io.open(p, encoding='utf-8', newline='').read()
m = re.search(r'(Kind:\s+"create",\n(?:\s+\w+:.*\n)*?\s+Autonomy:\s+)"attended"', t)
if not m:
    print('create/attended anchor not found; cannot re-prove')
    raise SystemExit(0)
t = t[:m.start(1)] + m.group(1) + '"unattended"' + t[m.end():]
io.open(p, 'w', encoding='utf-8', newline='').write(t)
print('marked an effectful kind (create) unattended')
PY
go test ./internal/operations -run 'OnlyEffectFreeKinds' -count=1 \
  >> "$OUT/violation.txt" 2>&1 || true
cp /tmp/surface.unat.backup internal/operations/surface.go
echo "restored surface.go" >> "$OUT/violation.txt"

cat "$OUT/tests-summary.txt" | sed 's/^/  /'
echo ""
cat "$OUT/violation.txt" | sed 's/^/  /'
