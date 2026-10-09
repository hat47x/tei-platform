#!/usr/bin/env bash
set -uo pipefail

# Read-only repository health check.
#
# GitHub Actions is currently used for manual diagnostics only (see the section on
# GitHub Actions in CONTRIBUTING.md), so nothing checks a change before it reaches
# main. This script is the local replacement for the checks CI would run.
#
# Usage:
#   tools/repository-health-check.sh          # gofmt, go vet, go test, workspace validate
#   tools/repository-health-check.sh --fast   # skip go test
#   tools/repository-health-check.sh --fix    # gofmt -w on drifted files first, then check
#   (--fast and --fix can be combined)
#
# It runs every step even if an earlier one fails, prints a summary, and exits 1 if
# any step failed. Without --fix it never modifies files. The Go environment is fixed
# the same way the Gate runners fix it (GOENV=off GOWORK=off GOFLAGS=).

fast=false
fix=false
for arg in "$@"; do
  case "$arg" in
    --fast) fast=true ;;
    --fix) fix=true ;;
    -h | --help)
      sed -n '3,19p' "$0"
      exit 0
      ;;
    *)
      echo "error: unknown argument '$arg' (see --help)" >&2
      exit 2
      ;;
  esac
done

repo_root="$(git rev-parse --show-toplevel 2>/dev/null || true)"
if [[ -z "$repo_root" || "$PWD" != "$repo_root" ]]; then
  echo "error: run from the TEI repository root" >&2
  exit 2
fi
if ! command -v go >/dev/null 2>&1; then
  echo "error: go is not on PATH" >&2
  exit 2
fi

export GOENV=off GOWORK=off GOFLAGS=

failed=()
passed=()

run_step() {
  local name="$1"
  shift
  echo "==> ${name}" >&2
  if "$@"; then
    passed+=("$name")
  else
    failed+=("$name")
  fi
}

gofmt_clean() {
  local drift
  drift="$(gofmt -l . 2>&1)"
  if [[ -n "$drift" ]]; then
    printf 'gofmt drift in:\n%s\n' "$drift" >&2
    return 1
  fi
}

workspace_valid() {
  local out
  out="$(go run ./cmd/tei validate -workspace reference/workspace 2>&1)" || {
    printf '%s\n' "$out" >&2
    return 1
  }
  if [[ "$(printf '%s\n' "$out" | tail -n 1)" != "ok" ]]; then
    printf '%s\n' "$out" >&2
    return 1
  fi
}

if [[ "$fix" == true ]]; then
  fixed="$(gofmt -l . 2>/dev/null)"
  if [[ -n "$fixed" ]]; then
    printf '%s\n' "$fixed" | xargs gofmt -w
    printf 'gofmt -w applied to:\n%s\n' "$fixed" >&2
  fi
fi

run_step "gofmt -l . is empty" gofmt_clean
run_step "go vet ./..." go vet ./...
if [[ "$fast" == false ]]; then
  run_step "go test -count=1 -timeout 60m ./..." go test -count=1 -timeout 60m ./...
fi
run_step "tei validate -workspace reference/workspace" workspace_valid

echo >&2
for name in "${passed[@]}"; do
  echo "PASS  ${name}" >&2
done
for name in "${failed[@]}"; do
  echo "FAIL  ${name}" >&2
done

if [[ "${#failed[@]}" -gt 0 ]]; then
  exit 1
fi
