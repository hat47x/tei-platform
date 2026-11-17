#!/usr/bin/env bash
# One stage of the three-subject self-build. Everything a stage does is in this
# file, so observing this script observes the whole stage.
# usage: stage.sh <stage dir> <tei> <builder> <catalog-author> <decl dir> <repo> <go> <supply> [subject...]
set -euo pipefail
OUT=$1; TEI=$2; BUILDER=$3; AUTHOR=$4; DECL=$5; REPO=$6; GOEXE=$7; SUPPLY=$8
shift 8
SUBJECTS=("$@")
if [ ${#SUBJECTS[@]} -eq 0 ]; then SUBJECTS=(tei builder author); fi
mkdir -p "$OUT/catalog" "$OUT/workspace"
cp "$BUILDER" "$OUT/catalog/builder"
cp "$DECL/capability.json" "$OUT/catalog/capability.json"
cp "$DECL/tei.workspace.yaml" "$OUT/workspace/tei.workspace.yaml"
"$AUTHOR" -declaration "$OUT/catalog/capability.json" -root "$OUT/catalog" -output "$OUT/catalog/catalog.json" > "$OUT/catalog/author.out"
read -r CATSUM _ < <(sha256sum "$OUT/catalog/catalog.json")
for SUBJECT in "${SUBJECTS[@]}"; do
  mkdir -p "$OUT/$SUBJECT"
  read -r ID < "$DECL/$SUBJECT.id"
  set +e
  "$TEI" self-build \
    -workspace "$OUT/workspace" \
    -intent "$ID" \
    -intent-file "$DECL/$SUBJECT.intent.json" \
    -build-catalog "$OUT/catalog/catalog.json" \
    -build-catalog-digest "sha256:$CATSUM" \
    -build-source-root "$REPO" \
    -child "$OUT/catalog/builder" \
    -child-arg "[\"-intent\",\"$DECL/$SUBJECT.intent.json\",\"-repo\",\"$REPO\",\"-go\",\"$GOEXE\"]" \
    -hand-over-plan \
    -supply "$SUPPLY" \
    -state "$OUT/$SUBJECT/state" \
    -output "$OUT/$SUBJECT/artifact" > "$OUT/$SUBJECT/report.json" 2> "$OUT/$SUBJECT/stderr.txt"
  echo "$SUBJECT rc=$?"
  set -e
done
