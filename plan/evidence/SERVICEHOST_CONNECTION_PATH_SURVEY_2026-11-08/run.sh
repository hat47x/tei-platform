#!/bin/bash
# Why does internal/servicehost have no non-test caller? Enumerate the pieces
# it needs and show where each of those pieces is built outside tests.
#
# The package composes three Host-owned inputs:
#   1 a Coordinator (composition.NewSelectedCoordinator)
#   2 a Runtime registry (plugin/runtime)
#   3 a ServiceCarrierRouteSource (BindHostApplicationEnvironmentRoute)
# Each is counted by constructor call site, split test / non-test.
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
  | grep -v 'plan/evidence/SERVICEHOST_CONNECTION_PATH_SURVEY_2026-11-08' > "$OUT/pre-status.txt" || true

# Call sites, split by test / non-test. A call site is the file that names the
# constructor, excluding the file that defines it.
scan() {
  needle="$1"
  label="$2"
  files=$(grep -rl --include='*.go' "$needle" internal cmd tools reference 2>/dev/null || true)
  printf '== %s ==\n' "$label" >> "$OUT/constructors.txt"
  if [ -z "$files" ]; then
    printf '  (no call sites)\n' >> "$OUT/constructors.txt"
    return
  fi
  for f in $files; do
    case "$f" in
      *_test.go) tag=TEST ;;
      *) tag="NON-TEST" ;;
    esac
    printf '  %-9s %s\n' "$tag" "$f" >> "$OUT/constructors.txt"
  done
}

: > "$OUT/constructors.txt"
scan 'NewSelectedServiceCarrierInvoker(' 'NewSelectedServiceCarrierInvoker'
scan 'BindHostApplicationEnvironmentRoute(' 'BindHostApplicationEnvironmentRoute'
scan 'NewSelectedCoordinator(' 'composition.NewSelectedCoordinator'
scan 'composition.NewRegistry()' 'composition.NewRegistry'
scan 'NewSelectedServiceCarrierInvoker' 'servicehost package usage (any reference)'

# The three inputs the invoker needs, so the reader can see which are missing.
{
  echo
  echo '== what the invoker needs (doc comments) =='
  grep -n 'func NewSelectedServiceCarrierInvoker' -A 6 internal/servicehost/service_carrier.go || true
} >> "$OUT/constructors.txt"

# Counts only.
{
  echo "== counts =="
  echo "non-test call sites of NewSelectedServiceCarrierInvoker: $(grep -rl --include='*.go' 'NewSelectedServiceCarrierInvoker' internal cmd tools reference 2>/dev/null | grep -v '_test.go' | grep -v 'internal/servicehost/service_carrier.go' | wc -l)"
  echo "test call sites of NewSelectedServiceCarrierInvoker: $(grep -rl --include='*.go' 'NewSelectedServiceCarrierInvoker' internal cmd tools reference 2>/dev/null | grep '_test.go' | wc -l)"
  echo "non-test call sites of NewSelectedCoordinator: $(grep -rl --include='*.go' 'NewSelectedCoordinator' internal cmd tools reference 2>/dev/null | grep -v '_test.go' | grep -vc 'internal/composition/bootstrap.go' || echo 0)"
  echo "non-test call sites of composition.NewRegistry: $(grep -rl --include='*.go' 'composition.NewRegistry()' internal cmd tools reference 2>/dev/null | grep -v '_test.go' | wc -l)"
} > "$OUT/summary.txt"

cat "$OUT/summary.txt"
