#!/bin/bash
# Reproduce the operation/pattern crosswalk report for two Workspaces.
#
# Usage: run.sh <outdir>
# The repository root is derived from the script location, so it works from a
# worktree as well as from a plain checkout.
#
# The accepted reference Workspace is a historical baseline and is never edited.
# It is copied into a temporary directory and the candidate files are overlaid
# on the copy only, exactly as reference/web-composition/go-host does.
set -eu

[ "$#" -ge 1 ] || { echo "usage: run.sh <outdir>" >&2; exit 2; }
OUT=$(mkdir -p "$1" && cd "$1" && pwd)
REPO=$(cd "$(dirname "$0")/../../.." && pwd)

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
git -C "$REPO" status --porcelain=v1 --untracked-files=all \
  | grep -v 'plan/evidence/OPERATION_PATTERN_CROSSWALK_REFERENCE_2026-11-07' > "$OUT/pre-status.txt" || true

cp -r "$REPO/reference/workspace" "$WORK/accepted"
cp -r "$REPO/reference/workspace" "$WORK/overlay"
cp -r "$REPO/reference/candidates/order-quote/applications/order-entry/." "$WORK/overlay/applications/order-entry/"
cp -r "$REPO/reference/candidates/order-quote/semantic/sales/." "$WORK/overlay/semantic/sales/"

cd "$REPO"
for name in accepted overlay; do
  go run ./tools/reference-operation-pattern-crosswalk/cmd/tei-operation-pattern-crosswalk \
    -workspace "$WORK/$name" > "$OUT/$name-crosswalk.json" 2> "$OUT/$name-crosswalk.err.txt"
done

python3 - "$OUT" <<'PY'
import json
import pathlib
import sys

out = pathlib.Path(sys.argv[1])
lines = []
for name in ("accepted", "overlay"):
    report = json.load(open(out / (name + "-crosswalk.json"), encoding="utf-8"))
    lines.append("== %s ==" % name)
    lines.append("rule=%s/%s %s" % (report["ruleId"], report["ruleVersion"], report["ruleFingerprint"]))
    for app in report["applications"]:
        lines.append("application %s: operations=%d patterns=%d"
                     % (app["id"], len(app["operations"]), len(app["patterns"])))
        lines.append("  links (%d):" % len(app.get("links") or []))
        for link in (app.get("links") or []):
            lines.append("    %s <- %s [%s] via=%s subject=%s"
                         % (link["operation"], link["pattern"], link["patternKind"],
                            link["via"], link["subject"]))
        lines.append("  unlinked (%d):" % len(app.get("unlinked") or []))
        for item in (app.get("unlinked") or []):
            lines.append("    %s (%s) %s: %s"
                         % (item["pattern"], item["kind"], item["code"], item["detail"]))
        lines.append("  conflicts=%d ambiguous=%s"
                     % (len(app["conflicts"]), app.get("ambiguous")))
    lines.append("")
(out / "summary.txt").write_text("\n".join(lines) + "\n", encoding="utf-8", newline="\n")
print("\n".join(lines))
PY
