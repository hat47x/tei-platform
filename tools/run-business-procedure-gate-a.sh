#!/usr/bin/env bash
set -uo pipefail

# Lane C (Business Procedure) Gate A actual-execution runner.
#
# Executes Steps A0-A7 of
#   plan/BUSINESS_PROCEDURE_GATE_A_EXECUTION_RUNBOOK_2026-09.md
# against ONE revision R (the current HEAD of a clean checkout) and writes the
# Evidence record (steps, exit codes, logs, summary JSON) outside the repository.
#
# Usage (from the repository root, clean full checkout):
#   TEI_GATE_A_ARTIFACT_DIR=/path/outside/repo/run-R tools/run-business-procedure-gate-a.sh
#
# Every step runs even if an earlier one fails (the Runbook: do not fail fast
# unless a later step is meaningless). Exit status is 0 only if Gate A PASS.
# Nothing in the repository is modified.

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

if [[ -n "$(git status --porcelain --untracked-files=all --ignored=matching)" ]]; then
  echo "error: the checkout is not clean; Gate A binds Evidence to one exact revision R" >&2
  git status --short --untracked-files=all --ignored=matching | head -20 >&2
  exit 2
fi

if [[ -n "${TEI_GATE_A_ARTIFACT_DIR:-}" ]]; then
  artifact_dir="$TEI_GATE_A_ARTIFACT_DIR"
  if [[ -e "$artifact_dir" ]]; then
    echo "error: artifact directory already exists: $artifact_dir" >&2
    exit 2
  fi
  mkdir -m 700 -p "$artifact_dir"
else
  artifact_dir="$(mktemp -d "${TMPDIR:-/tmp}/tei-gate-a.XXXXXX")"
fi
artifact_dir="$(cd "$artifact_dir" && pwd -P)"
case "$artifact_dir" in
  "$repo_root" | "$repo_root"/*)
    echo "error: artifact directory must be outside the repository: $artifact_dir" >&2
    exit 2
    ;;
esac

manifest="tools/business-procedure-gate-a-critical-tests.txt"
revision="$(git rev-parse HEAD)"
started_at="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
echo "Gate A artifact directory: $artifact_dir" >&2
echo "R = $revision" >&2

steps_file="$artifact_dir/steps.tsv"
: >"$steps_file"
failed=()

# run_step <id> <name> <command...>  (stdout+stderr -> <id>.log)
run_step() {
  local id="$1" name="$2"
  shift 2
  echo "==> $id $name" >&2
  local t0 t1 code
  t0="$(date +%s)"
  "$@" >"$artifact_dir/$id.log" 2>&1
  code=$?
  t1="$(date +%s)"
  printf '%s\t%s\t%s\t%s\t%s\n' "$id" "$name" "$code" "$((t1 - t0))" "$*" >>"$steps_file"
  if [[ "$code" -ne 0 ]]; then
    failed+=("$id")
    echo "    FAIL ($code), see $artifact_dir/$id.log" >&2
  fi
  return 0
}

# ---- A0 environment capture (secrets are never recorded) ----
{
  echo "repositoryRevision=$revision"
  echo "branch=$(git branch --show-current)"
  echo "mergeBaseWithOriginMain=$(git merge-base HEAD origin/main 2>/dev/null || echo unknown)"
  echo "leftRightCountOriginMainVsHead=$(git rev-list --left-right --count origin/main...HEAD 2>/dev/null || echo unknown)"
  echo "os_arch=$(go env GOOS)/$(go env GOARCH)"
  go version
  git --version
  go env GOOS GOARCH GOVERSION CGO_ENABLED GOMOD GOWORK
} >"$artifact_dir/A0-environment.log" 2>&1
printf 'A0\tenvironment capture\t0\t0\t(see A0-environment.log)\n' >>"$steps_file"

# ---- A1 format check (read-only) ----
format_check() {
  local dirs=() d drift
  for d in cmd internal examples tools; do
    [[ -d "$d" ]] && dirs+=("$d")
  done
  drift="$(gofmt -l "${dirs[@]}")"
  if [[ -n "$drift" ]]; then
    printf 'gofmt drift:\n%s\n' "$drift"
    return 1
  fi
}
run_step A1 "format check (gofmt -l cmd internal examples tools)" format_check

# ---- A2 static analysis ----
run_step A2 "go vet ./..." go vet ./...

# ---- A3 full test suite ----
run_step A3 "go test -count=1 ./..." go test -count=1 ./...

# ---- A4 Lane C focused package ----
run_step A4 "go test -count=1 ./tools/reference-business-procedure-check" \
  go test -count=1 ./tools/reference-business-procedure-check

# ---- A5 reference checker baseline ----
run_step A5 "reference checker baseline" \
  go run ./tools/reference-business-procedure-check \
  -workspace reference/workspace \
  -fixture reference/business-system-design/order-entry-business-procedure.json

# ---- A6 intentional defect checker (detected defect = PASS) ----
run_step A6 "intentional defect checker (expects BUSINESS_FUNCTION_DATASOURCE_ACCESS_CONFLICT)" \
  go run ./tools/reference-business-procedure-check \
  -workspace reference/workspace \
  -fixture reference/business-system-design/order-entry-business-procedure.json \
  -probe reference/business-system-design/order-entry-customer-edit-conflict.json \
  -expect-diagnostic BUSINESS_FUNCTION_DATASOURCE_ACCESS_CONFLICT

# ---- A7 critical Pressure preflight + focused triage ----
# The exact test names must all exist at R (a -run regex that matches nothing still exits 0),
# and every one of them must actually run and PASS in the focused run.
critical_preflight_and_triage() {
  local listed missing=0 name count expected regex
  listed="$(go test ./tools/reference-business-procedure-check -list 'Test.*')" || return 1
  expected=0
  while IFS= read -r name; do
    [[ -z "$name" ]] && continue
    expected=$((expected + 1))
    if ! printf '%s\n' "$listed" | grep -Fxq "$name"; then
      echo "missing expected Lane C test at revision R: $name"
      missing=$((missing + 1))
    fi
  done <"$manifest"
  if [[ "$missing" -ne 0 ]]; then
    return 1
  fi
  regex="^($(paste -sd'|' "$manifest"))\$"
  go test -count=1 -v ./tools/reference-business-procedure-check -run "$regex" >"$artifact_dir/A7-focused-run.out" 2>&1 || {
    cat "$artifact_dir/A7-focused-run.out"
    return 1
  }
  count="$(grep -c '^--- PASS: ' "$artifact_dir/A7-focused-run.out" || true)"
  echo "manifest tests: $expected; focused run PASS lines: $count"
  if [[ "$count" -ne "$expected" ]]; then
    echo "focused run did not execute every manifest test"
    return 1
  fi
}
run_step A7 "critical Pressure preflight (exact names) + focused run" critical_preflight_and_triage

finished_at="$(date -u +%Y-%m-%dT%H:%M:%SZ)"

# ---- Evidence summary (Runbook section 12) ----
gate_a_pass=true
failed_json=""
for f in "${failed[@]:-}"; do
  [[ -n "$f" ]] && failed_json+="${failed_json:+, }\"$f\""
done
if [[ "${#failed[@]}" -gt 0 ]]; then
  gate_a_pass=false
fi
summary="$artifact_dir/gate-a-summary.json"
{
  printf '{\n'
  printf '  "evidenceNature": "business-procedure-lane-c-gate-a-actual-execution",\n'
  printf '  "repositoryRevision": "%s",\n' "$revision"
  printf '  "repositoryClean": true,\n'
  printf '  "runnerKind": "local-full-checkout",\n'
  printf '  "osArch": "%s/%s",\n' "$(go env GOOS)" "$(go env GOARCH)"
  printf '  "goVersion": "%s",\n' "$(go version | sed 's/"/\\"/g')"
  printf '  "startedAt": "%s",\n' "$started_at"
  printf '  "finishedAt": "%s",\n' "$finished_at"
  printf '  "steps": [\n'
  first=true
  while IFS=$'\t' read -r id name code secs cmd; do
    [[ "$first" == true ]] || printf ',\n'
    first=false
    printf '    {"id": "%s", "name": "%s", "exit": %s, "seconds": %s, "log": "%s"}' \
      "$id" "$(printf '%s' "$name" | sed 's/"/\\"/g')" "$code" "$secs" "$id.log"
  done <"$steps_file"
  printf '\n  ],\n'
  printf '  "failedSteps": [%s],\n' "$failed_json"
  printf '  "gateAPassed": %s,\n' "$gate_a_pass"
  printf '  "authorityLimits": [\n'
  printf '    "gate-a-is-mechanical-execution-evidence-not-contract-adr-readiness",\n'
  printf '    "go-test-pass-does-not-mean-pas-07-accepted"\n'
  printf '  ]\n'
  printf '}\n'
} >"$summary"

post_revision="$(git rev-parse HEAD)"
if [[ "$post_revision" != "$revision" || -n "$(git status --porcelain --untracked-files=all --ignored=matching)" ]]; then
  echo "error: the repository changed during the run; discard this Evidence" >&2
  exit 3
fi

cat "$summary"
if [[ "$gate_a_pass" == true ]]; then
  echo "Gate A PASS at R=$revision" >&2
  exit 0
fi
echo "Gate A NOT PASSED at R=$revision (failed: ${failed[*]})" >&2
exit 1
