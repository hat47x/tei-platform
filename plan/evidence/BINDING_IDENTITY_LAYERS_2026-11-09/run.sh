#!/bin/bash
# CON-12c asks: if artifact digest, Host/Grant epoch or Source scope changes, does
# the path reuse the old binding? That question presupposes the binding CARRIES
# those fields. This script checks which layers do, and which do not.
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
  | grep -v 'plan/evidence/BINDING_IDENTITY_LAYERS_2026-11-09' > "$OUT/pre-status.txt" || true

# Print the field list of a struct declaration, so the presence or absence of
# digest / epoch / scope is visible rather than asserted.
fields() {
  label="$1"
  file="$2"
  pattern="$3"
  out="$OUT/$4.txt"
  {
    echo "== $label =="
    echo "   file: $file"
    awk -v pat="$pattern" '
      $0 ~ pat { inm=1 }
      inm { print "   " $0 }
      inm && /^}/ { exit }
    ' "$file"
  } > "$out"
  cat "$out"
  echo ""
}

fields "generation.Binding (used by C-1/C-2)" "core/generation/router.go" "type Binding struct" "generation-binding"
fields "environment.ObservationProviderBinding (used by the observation reference)" "internal/environment/model.go" "type ObservationProviderBinding struct" "observation-binding"
fields "environment.EvidenceProviderBinding (used by the evidence reference)" "internal/environment/model.go" "type EvidenceProviderBinding struct" "evidence-binding"
fields "runtime.ServiceProviderSelection (used by the service carrier)" "internal/plugin/runtime/service_carrier.go" "type ServiceProviderSelection struct" "service-selection"

# Count the identity fields each layer carries. Only counting, no interpretation.
{
  echo "== identity fields per binding =="
  for f in generation-binding observation-binding evidence-binding service-selection; do
    n=$(grep -cE 'ArtifactDigest|Generation|Scope|Epoch|Revision' "$OUT/$f.txt" || true)
    echo "   $f: $n"
  done
} > "$OUT/identity-count.txt"
cat "$OUT/identity-count.txt"
