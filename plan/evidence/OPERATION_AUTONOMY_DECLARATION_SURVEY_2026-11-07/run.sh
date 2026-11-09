#!/bin/bash
# Observe how the autonomy class is derived today.
#
# Usage: run.sh <outdir>
#
# Questions:
#   1. Which operations does the surface mark unattended, and on what evidence?
#   2. Does the Environment model carry a "service subject" declaration that the
#      design's unattended condition asks for?
#   3. Can an additional declaration change the autonomy class of an operation?
#
# Reads only. The accepted Workspaces are never modified.
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
  | grep -v 'plan/evidence/OPERATION_AUTONOMY_DECLARATION_SURVEY_2026-11-07' > "$OUT/pre-status.txt" || true

# Q2: where an autonomy-related declaration could live.
{
  echo "== non-test occurrences of autonomy-related declarations =="
  for needle in "ServiceSubject" "service subject" "human-gated" "Autonomy"; do
    echo "-- $needle --"
    grep -rn --include='*.go' -F "$needle" internal cmd 2>/dev/null \
      | grep -v '_test.go' | head -20 || true
  done
  echo
  echo "== Environments present in the fixtures used =="
  for ws in reference/standard-crud/workspace reference/workspace; do
    echo "-- $ws --"
    find "$ws" -path '*/environments/*' -type f 2>/dev/null | sed "s|$ws/||" || true
  done
} > "$OUT/declaration-scan.txt"

# Q1 and Q3: the surface for the standard-crud fixture (has writes), and for the
# accepted reference fixture (has an environment file). A copy that ADDS a
# declaration is not attempted, because no grammar or model field exists for a
# service subject; the scan above records that absence.
WORK=$(mktemp -d)
trap 'rm -rf "$WORK"' EXIT
cp -r reference/standard-crud/workspace "$WORK/standard-crud"
cp -r reference/workspace "$WORK/reference"

for name in standard-crud reference; do
  go run ./cmd/tei operations -workspace "$WORK/$name" -application "$([ "$name" = reference ] && echo order_entry || echo ledger)" \
    > "$OUT/$name-operations.json" 2> "$OUT/$name-operations.err.txt"
done

python3 - "$OUT" <<'PY'
import json
import pathlib
import sys

out = pathlib.Path(sys.argv[1])
lines = []
for name in ("standard-crud", "reference"):
    surface = json.load(open(out / (name + "-operations.json"), encoding="utf-8"))
    lines.append("== %s ==" % name)
    lines.append("  application: %s" % surface.get("application"))
    lines.append("  worldFingerprint: %s" % surface["worldFingerprint"])
    counts = {}
    for op in surface["operations"]:
        counts[op["autonomy"]] = counts.get(op["autonomy"], 0) + 1
        lines.append("  %s | kind=%s | autonomy=%s" % (op["id"], op["kind"], op["autonomy"]))
    lines.append("  autonomy counts: %s" % counts)
    lines.append("")
(out / "summary.txt").write_text("\n".join(lines) + "\n", encoding="utf-8", newline="\n")
print("\n".join(lines))
PY
