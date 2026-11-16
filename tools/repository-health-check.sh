#!/usr/bin/env bash
set -uo pipefail

# リポジトリ全体を読み取り専用で検証する。
#
# GitHub ActionsのPR・main検証と、ローカルの任意診断の双方から使う。
# 統合の判断は変更範囲に応じた検証で行い、この全体検証の全項目合格を
# 一律の統合条件とはしない（AGENTS.mdとCONTRIBUTING.mdを参照）。
#
# Usage:
#   tools/repository-health-check.sh          # gofmt, go vet, go test, nested modules, workspace validate
#   tools/repository-health-check.sh --fast   # skip go test, including in nested modules
#   tools/repository-health-check.sh --fix    # gofmt -w on drifted files first, then check
#   (--fast and --fix can be combined)
#
# It runs every step even if an earlier one fails, prints a summary, and exits 1 if
# any step failed. Without --fix it never modifies files. The Go environment is fixed
# the same way the Gate runners fix it (GOENV=off GOWORK=off GOFLAGS=).
#
# A nested go.mod is a separate build unit: `go vet ./...` and `go test ./...` from
# the repository root stop at it, so a module under e.g. plugins/ or cmd/ leaves
# those steps green without running its own tests. The module-scope step below
# enumerates nested modules, refuses an undeclared one, and verifies the declared
# ones. An excluded module must give a reason, which is printed so the exclusion is
# visible instead of silent (#4612). --fast keeps the declaration comparison, which
# is what stops a new module from escaping, and drops only the module test runs.

fast=false
fix=false
for arg in "$@"; do
  case "$arg" in
    --fast) fast=true ;;
    --fix) fix=true ;;
    -h | --help)
      sed -n '3,27p' "$0"
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
# The root Go steps above stop at a nested go.mod, so this is the only place a
# nested module is verified in this script. The declaration comparison runs in
# both modes because it is what keeps a new module from leaving the verification
# silently; --fast drops only the module test runs, like the root go test step.
if [[ "$fast" == true ]]; then
  run_step "nested Go modules are declared" \
    go run ./tools/repository-module-scope -check-only
else
  run_step "nested Go modules are declared and verified" \
    go run ./tools/repository-module-scope
fi
run_step "tei validate -workspace reference/workspace" workspace_valid
# Requalification is a synthetic decision-table fixture, not a live Source/
# Consumer closure probe. Run it directly so its negative controls cannot
# silently disappear behind the Go-only repository checks.
run_step "claim requalification synthetic fixture (R01-R17)" \
  python3 -B plan/fixtures/business-completion-claims/verify_claim_requalification.py
run_step "ADR index matches plan/adr/0*.md" go run ./tools/reference-adr-index-check .

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
