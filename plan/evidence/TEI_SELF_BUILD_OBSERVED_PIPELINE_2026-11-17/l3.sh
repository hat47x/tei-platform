#!/usr/bin/env bash
# stage0 (seed) -> stage1 -> stage2 for three subjects: tei, its builder, and
# the catalog authoring tool. Both stages run under the process observer;
# stage 2 also records opened paths.
# usage: l3.sh <checkout on a linux filesystem> <work dir> <go toolchain root>
set -euo pipefail
REPO=$(cd "$1" && pwd)
L=$2
GOROOT_DIR=$3
SP=$(cd "$(dirname "$0")" && pwd)
export PATH=$GOROOT_DIR/bin:$PATH
export GOTOOLCHAIN=local
GOEXE=$GOROOT_DIR/bin/go
cd "$REPO"
chmod -R u+w "$L" 2>/dev/null || true
rm -rf "$L"
mkdir -p "$L/stage0" "$L/decl" "$L/probe"
SUPPLY=$(go env GOMODCACHE)/cache/download

# --- stage 0: the external seed builds the three programs and the observer.
go build -trimpath -o "$L/stage0/tei" ./cmd/tei
go build -trimpath -o "$L/stage0/builder" ./tools/reference-go-build-from-plan
go build -trimpath -o "$L/stage0/author" ./tools/reference-build-catalog-authoring
go build -trimpath -o "$L/observe" ./tools/reference-build-process-observation

# --- declarations: written before any stage and never regenerated.
printf 'workspace: self_build_outer\nformatVersion: 1\nsources:\n  - semantic\n' > "$L/decl/tei.workspace.yaml"
cat > "$L/decl/capability.json" <<'JSON'
{
  "providers": [
    {"plugin": "tei.builder.go", "version": "0.0.0", "implementation": "go.build-from-plan",
     "contract": "tei.plugin-build", "contractMajor": 1, "artifact": "builder",
     "recipeKinds": ["go-command"], "targets": ["host/linux/amd64"], "profiles": ["release"]}
  ],
  "targets": [{"target": "host/linux/amd64", "profiles": ["release"]}]
}
JSON
declare -A ENTRY=( [tei]=./cmd/tei [builder]=./tools/reference-go-build-from-plan [author]=./tools/reference-build-catalog-authoring )
declare -A IDS=( [tei]=tei.command.tei [builder]=tei.tool.go_build_from_plan [author]=tei.tool.build_catalog_authoring )
declare_subjects() { # $1: toolchain closure digest, or empty to leave it unnamed
  for SUBJECT in tei builder author; do
    echo "${IDS[$SUBJECT]}" > "$L/decl/$SUBJECT.id"
    go list -deps -f '{{if not .Standard}}{{.Dir}}{{end}}' "${ENTRY[$SUBJECT]}" > "$L/decl/$SUBJECT.golist"
    python3 "$SP/write_intent.py" "$REPO" "$L/decl" "$SUBJECT" "${ENTRY[$SUBJECT]}" "${IDS[$SUBJECT]}" "$1"
  done
}

# The toolchain is named by digest. The digest is measured once: the seed-built
# tei builds the smallest subject with the toolchain left unnamed, and the
# builder's report of what it ran is then fixed in every declaration.
declare_subjects ""
bash "$SP/stage.sh" "$L/probe" "$L/stage0/tei" "$L/stage0/builder" "$L/stage0/author" "$L/decl" "$REPO" "$GOEXE" "$SUPPLY" author > /dev/null
TOOLCHAIN=$(python3 -c 'import json,sys; print(json.load(open(sys.argv[1]))["toolchainObservation"]["reported"]["closureDigest"])' "$L/probe/author/report.json")
echo "toolchain closure: $TOOLCHAIN"
declare_subjects "$TOOLCHAIN"
sha256sum "$L"/decl/* > "$L/decl.sha256"

# --- stage 1: stage-0 programs build stage-1 programs.
t=$(date +%s)
"$L/observe" observe -out "$L/stage1.observation.json" -- \
  bash "$SP/stage.sh" "$L/stage1" "$L/stage0/tei" "$L/stage0/builder" "$L/stage0/author" "$L/decl" "$REPO" "$GOEXE" "$SUPPLY"
echo "stage1: $(( $(date +%s) - t ))s"

# --- stage 2: only stage-1 programs are named.
t=$(date +%s)
"$L/observe" observe -files -out "$L/stage2.observation.json" -- \
  bash "$SP/stage.sh" "$L/stage2" "$L/stage1/tei/artifact" "$L/stage1/builder/artifact" "$L/stage1/author/artifact" "$L/decl" "$REPO" "$GOEXE" "$SUPPLY"
echo "stage2: $(( $(date +%s) - t ))s"
sha256sum -c --quiet "$L/decl.sha256" && echo "declarations unchanged across stages"
git -C "$REPO" rev-parse HEAD
git -C "$REPO" status --short | head -5
