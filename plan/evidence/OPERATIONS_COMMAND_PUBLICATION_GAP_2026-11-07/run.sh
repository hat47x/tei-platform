#!/bin/bash
# Observe what the CLI advertises versus what it accepts, and what `operations`
# returns on success and on failure.
#
# Usage: run.sh <outdir>
# Reads only. The accepted Workspace is never modified.
set -eu

[ "$#" -ge 1 ] || { echo "usage: run.sh <outdir>" >&2; exit 2; }
OUT=$(mkdir -p "$1" && cd "$1" && pwd)
REPO=$(cd "$(dirname "$0")/../../.." && pwd)

# A worktree created on Windows records an absolute Windows path in .git, which
# a Linux git cannot resolve. Derive GIT_DIR from the worktree location; on a
# native checkout this is a no-op.
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
  | grep -v 'plan/evidence/OPERATIONS_COMMAND_PUBLICATION_GAP_2026-11-07' > "$OUT/pre-status.txt" || true

python3 plan/evidence/OPERATIONS_COMMAND_PUBLICATION_GAP_2026-11-07/compare_commands.py . "$OUT/command-coverage.txt"

go run ./cmd/tei commands > "$OUT/commands-catalog.json" 2> "$OUT/commands-catalog.err.txt"

go run ./cmd/tei operations -workspace reference/standard-crud/workspace -application ledger \
  > "$OUT/operations-success.json" 2> "$OUT/operations-success.err.txt"

set +e
go run ./cmd/tei operations -workspace "$OUT/does-not-exist" -application ledger \
  > "$OUT/operations-failure.stdout.txt" 2> "$OUT/operations-failure.stderr.txt"
echo "exit=$?" >> "$OUT/operations-failure.stderr.txt"
set -e

python3 - "$OUT" <<'PY'
import json
import pathlib
import sys

out = pathlib.Path(sys.argv[1])
lines = []

catalog = json.load(open(out / "commands-catalog.json", encoding="utf-8"))
lines.append("catalog scope: %s" % catalog["scope"])
lines.append("catalog entries: %d" % len(catalog["commands"]))
names = [c["name"] for c in catalog["commands"]]
for name in ("operations", "artifacts", "periods", "view"):
    lines.append("  %-12s in catalog: %s" % (name, name in names))
lines.append("")

success = json.load(open(out / "operations-success.json", encoding="utf-8"))
lines.append("operations success: top-level keys = %s" % sorted(success.keys()))
lines.append("  has ok: %s" % ("ok" in success))
lines.append("  has result: %s" % ("result" in success))
lines.append("  schemaVersion: %s" % success.get("schemaVersion"))
lines.append("")

stdout = (out / "operations-failure.stdout.txt").read_text(encoding="utf-8")
stderr = (out / "operations-failure.stderr.txt").read_text(encoding="utf-8")
lines.append("operations failure: stdout bytes = %d" % len(stdout))
lines.append("operations failure: stdout parses as JSON = %s"
             % ("yes" if stdout.strip().startswith("{") else "no"))
lines.append("operations failure: first stderr line = %s" % stderr.splitlines()[0])
lines.append("operations failure: stderr lines = %d" % len(stderr.splitlines()))

(out / "summary.txt").write_text("\n".join(lines) + "\n", encoding="utf-8", newline="\n")
print("\n".join(lines))
PY
