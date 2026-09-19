#!/usr/bin/env bash
set -euo pipefail

files=(
  tools/reference-business-procedure-check/knowledge_portability_r0_pressure_test.go
  tools/reference-business-procedure-check/second_application_reference_pre_gate_pressure_test.go
  tools/reference-business-procedure-check/second_application_schema_sufficiency_pressure_test.go
  tools/reference-business-procedure-check/gate_c_second_application_entry_preflight_pressure_test.go
  tools/reference-business-procedure-check/second_application_gate_c_actual_entry_pressure_test.go
  tools/reference-business-procedure-check/second_application_s1_s2_actual_pressure_test.go
  tools/reference-business-procedure-check/second_application_s5_freshness_capture_pressure_test.go
)

unformatted="$(gofmt -l "${files[@]}")"
if [[ -n "$unformatted" ]]; then
  printf 'Unformatted Go files:\n%s\n' "$unformatted"
  exit 1
fi

run_tests() {
  go test -count=1 -v ./tools/reference-business-procedure-check \
    -run '^(TestKnowledgePortabilityR0|TestSecondApplication|TestGateCSecondApplication)'
}

if [[ -n "${TEI_REFERENCE_EVIDENCE_OUT:-}" ]]; then
  mkdir -p "$(dirname "$TEI_REFERENCE_EVIDENCE_OUT")"
  run_tests | tee "$TEI_REFERENCE_EVIDENCE_OUT"
else
  run_tests
fi
