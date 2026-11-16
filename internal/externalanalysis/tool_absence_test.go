package externalanalysis

import (
	"testing"
	"time"
)

// Tool absence is recorded in Receipt.Tool, not in Receipt.State: the saved bytes
// are still whole, so a missing tool does not downgrade the receipt. That is why
// StateToolAbsent is declared and branched on by callers but never produced by
// Receive. This test pins the decision from both sides, so a future change that
// starts emitting StateToolAbsent has to update the callers deliberately rather
// than silently changing what a receipt means.
func TestReceiveRecordsToolAbsenceWithoutChangingTheState(t *testing.T) {
	raw := fixture(t, "rea-6.8.0/analysis.json")
	absent := Tool{Name: "rea", Present: false, AbsentReason: "declared tool path not found"}

	receipt := Receive(raw, absent, time.Unix(0, 0))

	if receipt.State != StateComplete {
		t.Fatalf("state = %q, want complete: a missing tool does not downgrade whole bytes", receipt.State)
	}
	if receipt.State == StateToolAbsent {
		t.Fatalf("Receive produced %q, but no caller is written to expect it from Receive", StateToolAbsent)
	}
	if receipt.Tool.Present {
		t.Error("Tool.Present = true for an absent tool")
	}
	if receipt.Tool.AbsentReason == "" {
		t.Error("an absent tool must say why it was not usable")
	}
	if receipt.Raw.Fingerprint == "" {
		t.Error("the bytes are still the bytes: the fingerprint must survive an absent tool")
	}

	// A present tool is distinguishable from an absent one only through Tool.
	present := Receive(raw, Tool{Name: "rea", Present: true, ResolvedPath: "/usr/bin/rea"}, time.Unix(0, 0))
	if present.State != receipt.State {
		t.Errorf("present tool state = %q, absent tool state = %q; the state does not depend on the tool",
			present.State, receipt.State)
	}
	if present.Tool.Present == receipt.Tool.Present {
		t.Error("Tool.Present does not distinguish a present tool from an absent one")
	}
}
