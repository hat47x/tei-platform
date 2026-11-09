#!/usr/bin/env bash
set -euo pipefail

# Manual conformance verification for the E1 plan and E2 session reference.
# Full repository health is a separate optional diagnosis under the current
# AGENTS.md integration rules; this script only checks the Web reference.
# This script never invokes GitHub Actions and does not modify the repository.

usage() {
  cat <<'HELP'
Usage: reference/web-composition/verify.sh [--race]

Checks the Web E1 Node composition and route-table tests plus the E2 Go
reference packages. This is a scoped check, not repository-wide health.
Record exact-head evidence and any unrun checks in the corresponding PR.
HELP
}

race=false
if (($#)); then
  case "$1" in
    --race) race=true ;;
    --help|-h) usage; exit 0 ;;
    *) echo "invalid argument: $1" >&2; usage >&2; exit 2 ;;
  esac
  if (($# != 1)); then
    usage >&2
    exit 2
  fi
fi

root="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$root"
if [[ ! -f go.mod || ! -f reference/web-composition/plan-check.test.mjs ||
      ! -f reference/web-composition/route-table.test.mjs ]]; then
  echo "error: complete TEI repository checkout required" >&2
  exit 2
fi
for cmd in go node; do
  if ! command -v "$cmd" >/dev/null 2>&1; then
    echo "error: $cmd is not installed" >&2
    exit 2
  fi
done

export GOENV=off GOWORK=off GOFLAGS=
packages=(
  ./reference/web-composition/go-host
  ./tools/reference-first-web-target
  ./internal/security/legacyauth
)

echo "==> Web E1 Node composition tests" >&2
node --test \
  reference/web-composition/plan-check.test.mjs \
  reference/web-composition/route-table.test.mjs
echo "==> Web E2 Go vet" >&2
go vet "${packages[@]}"
echo "==> Web E2 Go tests" >&2
go test -count=1 "${packages[@]}"
if [[ "$race" == true ]]; then
  echo "==> Web E2 Go race tests" >&2
  go test -race -count=1 "${packages[@]}"
fi
echo "PASS: scoped Web reference checks (not repository-wide health)" >&2
