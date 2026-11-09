#!/bin/bash
# Compare where an operation/capability kind vocabulary is enforced.
#
# Usage: run.sh <outdir>
#
# Runs a tiny untracked program (created and then removed) that asks two
# questions against the imported packages:
#   1. Does plugin.ValidateCapabilityStructure accept an unknown CapabilityKind?
#   2. Does operations.DeriveOperationSurface tolerate an unknown operation kind
#      when a Pattern/derived operation carries one?
# Neither question writes to the repository or to a Workspace.
set -eu

[ "$#" -ge 1 ] || { echo "usage: run.sh <outdir>" >&2; exit 2; }
OUT=$(mkdir -p "$1" && cd "$1" && pwd)
REPO=$(cd "$(dirname "$0")/../../.." && pwd)

# A worktree created on Windows records an absolute Windows path in .git, which
# a Linux git cannot resolve. Derive GIT_DIR from the worktree location.
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
  | grep -v 'plan/evidence/OPERATION_VOCABULARY_PLACEMENT_2026-11-07' > "$OUT/pre-status.txt" || true

PROBE=$(mktemp -d "$REPO/.vocab-probe.XXXXXX")
cleanup() { rm -rf "$PROBE"; }
trap cleanup EXIT

cat > "$PROBE/main.go" <<'EOF'
package main

import (
	"fmt"

	"tei/internal/application"
	"tei/internal/ir"
	"tei/internal/operations"
	"tei/internal/plugin"
	"tei/internal/semantic"
)

func main() {
	// 1. An unknown plugin CapabilityKind.
	unknown := plugin.Capability{
		Kind: plugin.CapabilityKind("billing-executor"),
		ID:   "billing.executor",
	}
	errUnknown := plugin.ValidateCapabilityStructure([]plugin.Capability{unknown})
	fmt.Printf("unknown plugin CapabilityKind accepted: %t\n", errUnknown == nil)
	if errUnknown != nil {
		fmt.Printf("  rejected with: %v\n", errUnknown)
	}
	known := plugin.Capability{Kind: plugin.CapabilityDatabaseProvider, ID: "database.postgres"}
	errKnown := plugin.ValidateCapabilityStructure([]plugin.Capability{known})
	fmt.Printf("known plugin CapabilityKind accepted: %t\n", errKnown == nil)

	// 2. An unknown derived operation kind. The surface derives its own kinds,
	//    so the question is whether the rule's vocabulary is a closed set that
	//    the derivation would reject, or an open set that it ranks.
	world := ir.World{
		Semantics: []semantic.Namespace{{ID: "s",
			Data: []semantic.Data{{ID: "s.thing", Symbol: "Thing"}},
		}},
		Applications: []application.Application{{ID: "app"}},
	}
	surface := operations.DeriveOperationSurface(world, "app")
	fmt.Printf("derived operations for a World with no persistence: %d\n", len(surface.Operations))
	fmt.Printf("  gaps: %d\n", len(surface.Gaps))
	fmt.Printf("  rule: %s/%s %s\n", surface.Rule.ID, surface.Rule.Version, surface.Rule.Fingerprint)
}
EOF

cat > "$PROBE/store.json" <<'EOF'
{"schema":"unused"}
EOF

go run "$PROBE/main.go" > "$OUT/vocabulary-probe.txt" 2> "$OUT/vocabulary-probe.err.txt"

python3 - "$OUT" <<'PY'
import pathlib
import sys

out = pathlib.Path(sys.argv[1])
probe = (out / "vocabulary-probe.txt").read_text(encoding="utf-8")
err = (out / "vocabulary-probe.err.txt").read_text(encoding="utf-8")
lines = ["== vocabulary probe =="]
lines += ["  " + line for line in probe.splitlines()]
lines += ["  (stderr bytes: %d)" % len(err)]
(out / "summary.txt").write_text("\n".join(lines) + "\n", encoding="utf-8", newline="\n")
print("\n".join(lines))
PY
