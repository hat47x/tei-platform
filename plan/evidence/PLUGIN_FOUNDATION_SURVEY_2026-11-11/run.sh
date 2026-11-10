#!/bin/bash
# Survey the four themes of the plugin-foundation goal against the current code:
#   minimum core, extension points, dependency resolution, load-time diagnostics.
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
  | grep -v 'plan/evidence/PLUGIN_FOUNDATION_SURVEY_2026-11-11' > "$OUT/pre-status.txt" || true

{
  echo "== 1. dependency resolution: does it exist? =="
  grep -n 'func (c Catalog) ResolveEnvironment\|func (c Catalog) resolvePlugin\|func (c Catalog) resolveBundle' internal/plugin/resolve.go | sed 's/^/  /'
  echo ""
  echo "   cycle detection:"
  grep -n 'dependency cycle' internal/plugin/resolve.go | sed 's/^/    /'
  echo ""
  echo "   transitive requirement check:"
  grep -n 'already resolved to\|incompatible with required major' internal/plugin/resolve.go | sed 's/^/    /'
  echo ""
  echo "== 2. what resolution returns: a value or structured diagnostics? =="
  grep -n 'ResolvedSet{}, fmt.Errorf' internal/plugin/resolve.go | wc -l | sed 's/^/  plain fmt.Errorf returns: /'
  echo "  a Diagnostic type in internal/plugin:"
  hits=$(grep -rn 'type Diagnostic struct' internal/plugin/ 2>/dev/null || true)
  if [ -z "$hits" ]; then echo "    (none)"; else printf '%s\n' "$hits" | sed 's/^/    /'; fi
  echo ""
  echo "  the nearest existing Diagnostic type lives elsewhere:"
  grep -n 'type Diagnostic struct' internal/formatversion/formatversion.go | sed 's/^/    /'
  sed -n '/type Diagnostic struct/,/^}/p' internal/formatversion/formatversion.go | sed 's/^/    /'
  echo ""
  echo "== 3. do callers depend on the error text? =="
  grep -rn 'strings.Contains(err.Error()' internal/plugin/*_test.go | wc -l | sed 's/^/  test assertions matching substrings: /'
  grep -rn 'err.Error() != firstError' internal/plugin/*_test.go | sed 's/^/  /'
  echo ""
  echo "== 4. the core's capability vocabulary: closed or open? =="
  grep -n 'type CapabilityKind string' internal/plugin/model.go | sed 's/^/  /'
  grep -rhn 'Capability[A-Za-z]* CapabilityKind = "' internal/plugin/model.go | sed 's/^/  /'
  echo ""
  echo "== 5. extension points beyond the capability kind =="
  ls internal/plugin/implementation/ internal/plugin/inputprofile/ 2>/dev/null | sed 's/^/  /'
} > "$OUT/anchors.txt"

cat "$OUT/anchors.txt"
