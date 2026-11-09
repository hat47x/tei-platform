#!/bin/bash
# CON-12c on the layer that carries identity: generation.Binding.
# One run answers three questions:
#   1. before the change, which digest does a pinned lease hold?
#   2. after the change, which digest does a NEW acquisition hold?
#   3. after the change, which digest does the OLD lease hold?
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
  | grep -v 'plan/evidence/DIGEST_CHANGE_REUSE_REFERENCE_2026-11-09' > "$OUT/pre-status.txt" || true

go run ./tools/reference-digest-change-reuse/cmd/tei-digest-change-reuse \
  > "$OUT/report.json" 2> "$OUT/report.err.txt"

python3 - "$OUT" <<'PY'
import json
import pathlib
import sys

out = pathlib.Path(sys.argv[1])
r = json.load(open(out / 'report.json', encoding='utf-8'))
lines = []
lines.append('oldDigest: %s' % r['oldDigest'])
lines.append('newDigest: %s' % r['newDigest'])
lines.append('')
for o in r['outcomes']:
    lines.append('%-38s -> %s' % (o['name'], o['matches']))
lines.append('')
lines.append('oldLeaseStillPinned:   %s' % r['oldLeaseStillPinned'])
lines.append('newLeaseUsesNewDigest: %s' % r['newLeaseUsesNewDigest'])
lines.append('ruleFingerprint:       %s' % r['ruleFingerprint'])
(out / 'summary.txt').write_text('\n'.join(lines) + '\n', encoding='utf-8', newline='\n')
print('\n'.join(lines))
PY
