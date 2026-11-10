#!/bin/bash
# Re-derive the arm64 L2-b observation.
#
# Usage, from the repository root:
#
#   plan/evidence/TEI_GO_SELF_BUILD_ARM64_L2B_CHAIN_2026-11-10/run.sh amd64
#   plan/evidence/TEI_GO_SELF_BUILD_ARM64_L2B_CHAIN_2026-11-10/run.sh arm64
#
# The arm64 run needs an aarch64 emulator. Set QEMU to its path, or set
# QEMU_DOWNLOAD=1 to fetch the static binary from the multiarch release into
# $HOME/qemuprobe (no root needed). Without either, the arm64 run stops with a
# stated reason instead of claiming a result.
#
# Why a wrapper: an emulated arm64 process cannot execve another arm64 binary
# without binfmt_misc, which needs root, and self-build execs the child. The /bin/sh
# wrapper runs natively and starts the emulator explicitly, so `tei` and every tool
# in the chain are arm64 while only the exec is brokered.
set -uo pipefail

ARCH="${1:-amd64}"
case "$ARCH" in
  amd64|arm64) ;;
  *) echo "usage: $0 [amd64|arm64]" >&2; exit 2 ;;
esac

REPO_ROOT="$(cd "$(dirname "$0")/../../.." && pwd)"
FIXTURE="$(cd "$(dirname "$0")/fixture" && pwd)"
cd "$REPO_ROOT" || exit 1

QEMU="${QEMU:-$HOME/qemuprobe/qemu-aarch64-static}"
if [ "$ARCH" = "arm64" ] && [ ! -x "$QEMU" ]; then
  if [ "${QEMU_DOWNLOAD:-0}" = "1" ]; then
    mkdir -p "$(dirname "$QEMU")"
    curl -sSL -o "$QEMU" \
      "https://github.com/multiarch/qemu-user-static/releases/download/v7.2.0-1/qemu-aarch64-static" \
      || { echo "download failed" >&2; exit 1; }
    chmod +x "$QEMU"
  else
    echo "no aarch64 emulator at $QEMU; set QEMU=<path> or QEMU_DOWNLOAD=1" >&2
    exit 2
  fi
fi

WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT
mkdir -p "$WORK/bin" "$WORK/root/work" "$WORK/root/workspace" "$WORK/root/src"

echo "=== build tools for $ARCH ==="
for target in ./cmd/tei ./tools/reference-build-catalog-authoring \
              ./tools/reference-plugin-package-build-child \
              ./tools/reference-build-authoring-intent; do
  name=$(basename "$target")
  GOOS=linux GOARCH="$ARCH" go build -o "$WORK/bin/$name" "$target" || {
    echo "build failed: $target" >&2; exit 1; }
done

R="$WORK/root"
cp "$FIXTURE/tei.workspace.yaml" "$R/workspace/tei.workspace.yaml"
cp "$FIXTURE/plugin.tei" "$R/src/plugin.tei"
cp "$FIXTURE/implementation.go.txt" "$R/src/implementation.go.txt"
cp "$FIXTURE/intent.json" "$R/intent.json"
cp "$FIXTURE/capability-declaration.json" "$R/declaration.json"
cp "$WORK/bin/reference-plugin-package-build-child" "$R/builder"
chmod +x "$R/builder"

if [ "$ARCH" = "arm64" ]; then
  printf '#!/bin/sh\nexec %s %s/bin/reference-plugin-package-build-child "$@"\n' \
    "$QEMU" "$WORK" > "$R/child-run"
else
  printf '#!/bin/sh\nexec %s/bin/reference-plugin-package-build-child "$@"\n' \
    "$WORK" > "$R/child-run"
fi
chmod +x "$R/child-run"

run() {
  local bin="$1"; shift
  if [ "$ARCH" = "arm64" ]; then "$QEMU" "$bin" "$@"; else "$bin" "$@"; fi
}

echo "=== author the catalog ==="
run "$WORK/bin/reference-build-catalog-authoring" \
    -declaration "$R/declaration.json" -root "$R" -output "$R/catalog.json" \
  > "$WORK/authoring.json" 2>&1
DIGEST=$(python3 -c "
import json
try: print(json.load(open('$WORK/authoring.json'))['catalogDigest'])
except Exception: print('')
")
[ -z "$DIGEST" ] && { echo "no catalog digest; authoring output:" >&2; head -5 "$WORK/authoring.json" >&2; exit 1; }
echo "catalogDigest=$DIGEST"

echo "=== resolve, to give the child a plan file ==="
run "$WORK/bin/reference-build-authoring-intent" \
    -intent "$R/intent.json" -catalog "$R/catalog.json" -source-root "$R/src" \
  > "$WORK/receipt.json" 2>"$WORK/resolver.err"
python3 - "$WORK/receipt.json" "$WORK/plan.json" <<'PY'
import json, sys
json.dump(json.load(open(sys.argv[1]))["plan"], open(sys.argv[2], "w"), ensure_ascii=False)
PY

echo "=== tei self-build ==="
CHILDARG=$(python3 -c "
import json
print(json.dumps(['-plan','$WORK/plan.json','-plugin','$R/src/plugin.tei','-intent','$R/intent.json']))
")
run "$WORK/bin/tei" self-build \
  -workspace "$R/workspace" \
  -intent "tei.plugin.l2b_e2e" \
  -intent-file "$R/intent.json" \
  -build-catalog "$R/catalog.json" \
  -build-catalog-digest "$DIGEST" \
  -build-source-root "$R/src" \
  -child "$R/child-run" \
  -child-arg "$CHILDARG" \
  -state "$WORK/state" \
  -output "$WORK/package.json" > "$WORK/report.json" 2>"$WORK/tei.err"
echo "tei exit=$?"
head -3 "$WORK/tei.err"

echo "=== result ==="
python3 - "$WORK/report.json" "$ARCH" <<'PY'
import json, sys

try:
    d = json.load(open(sys.argv[1]))
except Exception as e:
    print("cannot decode report:", e)
    raise SystemExit(1)
r = d.get("planReceipt") or {}
b = d.get("planArtifactBinding") or {}
rep = d.get("artifactReproducibility") or {}
l2b = d.get("l2b") or {}
print("arch             :", sys.argv[2])
print("ok               :", d.get("ok"))
print("qualified        :", d.get("qualified"))
print("l2b.claimed      :", l2b.get("claimed"))
print("l2b.unmet        :", l2b.get("unmet"))
print("reproducibility  :", rep.get("state"))
print("intentFingerprint:", r.get("intentFingerprint"))
print("sourceClosure    :", r.get("sourceClosureFingerprint"))
print("catalogDigest    :", r.get("catalogDigest"))
print("planFingerprint  :", r.get("planFingerprint"))
print("artifactDigest   :", b.get("artifactDigest"))
PY
