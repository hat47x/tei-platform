#!/usr/bin/env bash
# One observed and reconciled self-build of ./cmd/tei, driven by the stage-1 tei.
# usage: judge.sh honest|copy
#   honest: the declared builder is the stage-1 builder
#   copy:   the declared builder is a wrapper that
# copies the first run's state into the second and then runs the real builder.
set -euo pipefail
GOROOT_DIR=$HOME/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.26.11.linux-amd64
export PATH=$GOROOT_DIR/bin:$PATH
export GOTOOLCHAIN=local
SP=$(dirname "$0")
W=$HOME/fable3
L=${JUDGE_L3:-$W/l3}
REPO=${JUDGE_REPO:-$W/tei}
MODE=${1:-honest}
D=$W/judge-$MODE
chmod -R u+w "$D" 2>/dev/null || true
rm -rf "$D"
mkdir -p "$D/catalog" "$D/workspace" "$D/wrap"
cp "$SP/copywrap/main.go" "$D/wrap/main.go"
if [ "$MODE" = copy ]; then
  (cd "$D/wrap" && printf 'module copywrap

go 1.26
' > go.mod && go build -trimpath -o "$D/catalog/builder" .)
else
  cp "$L/stage1/builder/artifact" "$D/catalog/builder"
fi
cp "$L/decl/capability.json" "$D/catalog/capability.json"
cp "$L/decl/tei.workspace.yaml" "$D/workspace/tei.workspace.yaml"
"$L/stage1/author/artifact" -declaration "$D/catalog/capability.json" -root "$D/catalog" -output "$D/catalog/catalog.json" > /dev/null
read -r CATSUM _ < <(sha256sum "$D/catalog/catalog.json")
export COPYWRAP_REAL_BUILDER=$L/stage1/builder/artifact
export COPYWRAP_SIDE_FILE=$D/side
set +e
"$L/observe" observe -files -out "$D/observation.json" -- \
  "$L/stage1/tei/artifact" self-build \
  -workspace "$D/workspace" -intent tei.command.tei -intent-file "$L/decl/tei.intent.json" \
  -build-catalog "$D/catalog/catalog.json" -build-catalog-digest "sha256:$CATSUM" -build-source-root "$REPO" \
  -child "$D/catalog/builder" \
  -child-arg "[\"-intent\",\"$L/decl/tei.intent.json\",\"-repo\",\"$REPO\",\"-go\",\"$GOROOT_DIR/bin/go\"]" \
  -hand-over-plan -supply "$(go env GOMODCACHE)/cache/download" \
  -state "$D/state" -output "$D/artifact" > "$D/report.json" 2> "$D/stderr.txt"
echo "self-build rc=$?"
"$L/observe" scope -self-build-report "$D/report.json" -read source-root="$REPO" -read catalog="$D/catalog/catalog.json" \
  -read intent="$L/decl/tei.intent.json" -out "$D/scope.json"
"$L/observe" reconcile -observation "$D/observation.json" -scope "$D/scope.json" -out "$D/ledger.json"
echo "reconcile rc=$?"
set -e
python3 - "$D" <<'PY'
import json, sys
d = sys.argv[1]
r = json.load(open(d + "/report.json"))
a = r["artifactReproducibility"]
print("ENTRY SAYS: ok", r["ok"], "| l2b.claimed", r["l2b"]["claimed"], "| reproducibility", a["state"],
      "| stateSeparation.basis", a["stateSeparation"]["basis"], "| stateObservation.kind", a["stateObservation"]["kind"],
      "| builderBinding", r["builderBinding"]["state"])
l = json.load(open(d + "/ledger.json"))
print("OBSERVER SAYS:", l["boundedAbsence"]["state"])
for reason in l["boundedAbsence"]["reasons"]:
    print("  -", reason)
print("  execs by role:", l["execs"]["byRole"])
print("  undeclared execs (first 2):", [(e["step"], e["image"].split("/fable3/")[-1], e["imageDigest"][:19]) for e in l["execs"]["undeclared"][:2]])
print("  crossStep total:", l["files"]["crossStepTotal"], "sample:",
      [(f["step"], f["role"], f["image"].split("/")[-1], "r%d w%d" % (f["read"], f["write"])) for f in l["files"]["crossStep"][:2]])
PY
