#!/bin/bash
# Re-derive the two child-identity observations.
#
#   run.sh proxy    a wrapper keeps its own bytes and changes which builder it runs
#   run.sh direct   the -child file itself is replaced between the two builds
#
# Usage from the repository root. Both modes use the committed fixture next to this
# script. No emulator is needed: both observations are about the entry's bookkeeping,
# which does not depend on the child's architecture.
set -uo pipefail

MODE="${1:-proxy}"
case "$MODE" in
  proxy|direct) ;;
  *) echo "usage: $0 [proxy|direct]" >&2; exit 2 ;;
esac

HERE="$(cd "$(dirname "$0")" && pwd)"
REPO_ROOT="$(cd "$HERE/../../.." && pwd)"
cd "$REPO_ROOT" || exit 1

OUT="$(mktemp -d)"
trap 'rm -rf "$OUT"' EXIT
mkdir -p "$OUT/bin" "$OUT/root/workspace" "$OUT/root/src"

echo "=== build the tools ==="
for target in ./cmd/tei ./tools/reference-build-catalog-authoring \
              ./tools/reference-build-authoring-intent \
              ./tools/reference-plugin-package-build-child; do
  GOOS=linux go build -o "$OUT/bin/$(basename "$target")" "$target" || exit 1
done
# Two builders from the same source with different bytes and the same behaviour.
GOOS=linux go build -trimpath -o "$OUT/bin/builder-a" ./tools/reference-plugin-package-build-child || exit 1
GOOS=linux go build -o "$OUT/bin/builder-b" ./tools/reference-plugin-package-build-child || exit 1
if cmp -s "$OUT/bin/builder-a" "$OUT/bin/builder-b"; then
  echo "the two builders are byte-identical; the experiment proves nothing" >&2
  exit 1
fi
echo "builder-a $(sha256sum "$OUT/bin/builder-a" | cut -c1-16)…"
echo "builder-b $(sha256sum "$OUT/bin/builder-b" | cut -c1-16)…"

R="$OUT/root"
cp "$HERE/fixture/tei.workspace.yaml" "$R/workspace/tei.workspace.yaml"
cp "$HERE/fixture/plugin.tei" "$R/src/plugin.tei"
cp "$HERE/fixture/implementation.go.txt" "$R/src/implementation.go.txt"
cp "$HERE/fixture/intent.json" "$R/intent.json"
cp "$OUT/bin/builder-a" "$R/builder"

if [ "$MODE" = "proxy" ]; then
  # Same bytes on both runs; the launched builder changes.
  cat > "$R/child-run" <<EOF
#!/bin/sh
COUNT="$OUT/count"
n=\$(cat "\$COUNT" 2>/dev/null || echo 0)
n=\$((n + 1))
echo "\$n" > "\$COUNT"
echo "wrapper: invocation \$n -> builder-\$( [ "\$n" -eq 1 ] && echo a || echo b )" >> "$OUT/wrapper.log"
if [ "\$n" -eq 1 ]; then exec "$OUT/bin/builder-a" "\$@"; else exec "$OUT/bin/builder-b" "\$@"; fi
EOF
else
  # The named file itself changes on the first invocation.
  cat > "$R/child-run.second" <<EOF
#!/bin/sh
exec "$OUT/bin/builder-b" "\$@"
EOF
  cat > "$R/child-run" <<EOF
#!/bin/sh
cp "$R/child-run.second" "$R/child-run"
chmod +x "$R/child-run"
exec "$OUT/bin/builder-a" "\$@"
EOF
fi
chmod +x "$R/child-run" "$R/child-run.second" 2>/dev/null || chmod +x "$R/child-run"
echo "child-run digest before: $(sha256sum "$R/child-run" | cut -c1-16)…"

cat > "$R/declaration.json" <<'JSON'
{
  "providers": [
    {"plugin": "tei.builder.go", "version": "1.2.0", "implementation": "go.builder",
     "contract": "tei.plugin-build", "contractMajor": 1,
     "artifact": "builder", "recipeKinds": ["go-package"],
     "targets": ["host/linux/amd64"], "profiles": ["release"]}
  ],
  "targets": [{"target": "host/linux/amd64", "profiles": ["release"]}]
}
JSON

echo ""
echo "=== author the catalog ==="
"$OUT/bin/reference-build-catalog-authoring" -declaration "$R/declaration.json" -root "$R" \
  -output "$R/catalog.json" > "$OUT/authoring.json" 2>&1
DIGEST=$(python3 -c "
import json
try: print(json.load(open('$OUT/authoring.json'))['catalogDigest'])
except Exception: print('')
")
[ -z "$DIGEST" ] && { echo "no catalog digest:"; head -5 "$OUT/authoring.json" >&2; exit 1; }
echo "catalogDigest=$DIGEST"

echo "=== resolve a plan for the child ==="
"$OUT/bin/reference-build-authoring-intent" -intent "$R/intent.json" -catalog "$R/catalog.json" \
  -source-root "$R/src" > "$OUT/receipt.json" 2>"$OUT/r.err" || { echo "resolver failed:"; head -3 "$OUT/r.err" >&2; exit 1; }
python3 - "$OUT/receipt.json" "$OUT/plan.json" <<'PY'
import json, sys
json.dump(json.load(open(sys.argv[1]))["plan"], open(sys.argv[2], "w"), ensure_ascii=False)
PY

# -intent makes the child report the declaration bytes, which condition 2 needs.
CHILDARG=$(python3 -c "
import json
print(json.dumps(['-plan','$OUT/plan.json','-plugin','$R/src/plugin.tei','-intent','$R/intent.json']))
")

echo "=== tei self-build (mode: $MODE) ==="
"$OUT/bin/tei" self-build \
  -workspace "$R/workspace" -intent "tei.plugin.l2b_e2e" -intent-file "$R/intent.json" \
  -build-catalog "$R/catalog.json" -build-catalog-digest "$DIGEST" -build-source-root "$R/src" \
  -child "$R/child-run" -child-arg "$CHILDARG" \
  -state "$OUT/state" -output "$OUT/package.json" > "$OUT/report.json" 2>"$OUT/tei.err"
echo "tei exit=$?"
head -3 "$OUT/tei.err"
echo "child-run digest after : $(sha256sum "$R/child-run" | cut -c1-16)…"
[ -f "$OUT/wrapper.log" ] && { echo "--- which builder ran ---"; cat "$OUT/wrapper.log"; }

echo ""
echo "=== report ==="
python3 - "$OUT/report.json" "$MODE" <<'PY'
import json, sys

try:
    d = json.load(open(sys.argv[1]))
except Exception as e:
    print("cannot decode report:", e)
    raise SystemExit(1)
l2b = d.get("l2b") or {}
rep = d.get("artifactReproducibility") or {}
print("mode                :", sys.argv[2])
print("l2b.claimed         :", l2b.get("claimed"))
print("l2b.unmet           :", l2b.get("unmet"))
print("reproducibility     :", rep.get("state"))
print("mismatchReason      :", rep.get("mismatchReason"))
print("firstChildDigest    :", rep.get("firstChildDigest"))
print("secondChildDigest   :", rep.get("secondChildDigest"))
print("firstArtifactDigest :", rep.get("firstArtifactDigest"))
print("secondArtifactDigest:", rep.get("secondArtifactDigest"))
PY
