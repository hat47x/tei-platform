#!/bin/bash
# Compare the declared Patterns and the derived Operations of one Application.
#
# Usage: run.sh <outdir>
# Run from anywhere; the repository root is derived from this script's path.
#
# The accepted reference Workspace is a historical baseline and is never edited.
# It is copied into a temporary directory; the candidate files are overlaid on
# the copy only, exactly as reference/web-composition/go-host does in its
# authorization test.
set -eu

[ "$#" -ge 1 ] || { echo "usage: run.sh <outdir>" >&2; exit 2; }
OUT=$(mkdir -p "$1" && cd "$1" && pwd)
REPO=$(cd "$(dirname "$0")/../../.." && pwd)
APP=${APP:-order_entry}

# A worktree created on Windows records an absolute Windows path in .git, which
# a Linux git cannot resolve. Derive GIT_DIR from the worktree location instead
# of hardcoding a machine-specific path. On a native checkout this is a no-op.
if [ -f "$REPO/.git" ]; then
  pointer=$(head -n 1 "$REPO/.git")
  case "$pointer" in
    gitdir:*[A-Za-z]:/*)
      win=${pointer#gitdir: }
      drive=$(printf '%s' "$win" | cut -c1 | tr '[:upper:]' '[:lower:]')
      rest=${win#*:}
      GIT_DIR="/mnt/$drive$rest"
      export GIT_DIR
      export GIT_WORK_TREE="$REPO"
      ;;
  esac
fi

export PATH=/usr/local/go/bin:/usr/bin:/bin:$PATH

WORK=$(mktemp -d)
trap 'rm -rf "$WORK"' EXIT

git -C "$REPO" rev-parse HEAD > "$OUT/head.txt"
go version >> "$OUT/head.txt"
# This evidence directory is untracked while the run happens; exclude it so the
# record shows only changes that this reproduction did not create.
git -C "$REPO" status --porcelain=v1 --untracked-files=all \
  | grep -v 'plan/evidence/PATTERN_KIND_OPERATION_MAPPING_2026-11-07' > "$OUT/pre-status.txt" || true

cp -r "$REPO/reference/workspace" "$WORK/accepted"
cp -r "$REPO/reference/workspace" "$WORK/overlay"
cp -r "$REPO/reference/candidates/order-quote/applications/order-entry/." "$WORK/overlay/applications/order-entry/"
cp -r "$REPO/reference/candidates/order-quote/semantic/sales/." "$WORK/overlay/semantic/sales/"

cd "$REPO"
for name in accepted overlay; do
  go run ./cmd/tei operations -workspace "$WORK/$name" -application "$APP" \
    > "$OUT/$name-operations.json" 2> "$OUT/$name-operations.err.txt"
  go run ./cmd/tei artifacts -workspace "$WORK/$name" -application "$APP" \
    > "$OUT/$name-artifacts.json" 2> "$OUT/$name-artifacts.err.txt"
done

python3 - "$OUT" <<'PY'
import json, sys, pathlib

out = pathlib.Path(sys.argv[1])


def patterns(path):
    doc = json.load(open(path, encoding="utf-8"))
    for art in doc["artifacts"]:
        if art["kind"] == "function-definition":
            return [l for l in art["content"].splitlines() if l.startswith("pattern ")]
    return []


lines = []
for name in ("accepted", "overlay"):
    ops = json.load(open(out / (name + "-operations.json"), encoding="utf-8"))
    lines.append("== %s ==" % name)
    lines.append("worldFingerprint=%s" % ops["worldFingerprint"])
    lines.append("modelRevision=%s" % ops["modelRevision"])
    lines.append("derived operations (%d):" % len(ops["operations"]))
    for o in ops["operations"]:
        lines.append("  %s | kind=%s | target=%s | binding=%s"
                     % (o["id"], o["kind"], o["target"], o.get("binding")))
    pats = patterns(out / (name + "-artifacts.json"))
    lines.append("declared patterns (%d):" % len(pats))
    for p in pats:
        lines.append("  " + p)
    lines.append("")
(out / "summary.txt").write_text("\n".join(lines) + "\n", encoding="utf-8", newline="\n")
print("\n".join(lines))
PY
