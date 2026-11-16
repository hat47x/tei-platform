package command_test

import (
	"errors"
	"testing"

	"tei/internal/command"
	"tei/internal/failure"
)

// TestReportedFailureDropsAPartialError checks what the Command boundary reports for
// a bare PartialError.
//
// X-TEA's bulk update (F-14) could leave external calls applied after the database
// was rolled back, and the knowledge inventory records that it had no way to say
// "partly done". TEI has PartialError for exactly that meaning, but this boundary
// does not report it: the error is not an ExecutionError and does not carry the
// effect-unknown flag, so no code reaches the caller at all.
func TestReportedFailureDropsAPartialError(t *testing.T) {
	err := &failure.PartialError{
		Code:    "COMPENSATION_PARTIAL",
		Message: "2 of 3 compensations were applied",
	}
	code, _, ok := command.ReportedFailure(err)
	if ok {
		t.Fatalf("a partial success now reports %q; the note in plan/evidence/XTEA_LOWCODE_PARTIAL_SUCCESS_CONNECTION_2026-11.md says it reports nothing", code)
	}
	if code != "" {
		t.Fatalf("code=%q", code)
	}
	if !errors.Is(err, err) {
		t.Fatal("sanity")
	}
}
