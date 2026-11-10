package servicecarrierconnect

import (
	"encoding/json"
	"testing"

	"tei/internal/composition"
)

// CON-12c names three identity fields: artifact digest, Host/Grant epoch, and
// Source scope. The previous test covered the digest at the servicehost
// comparison. This one covers scope, which is enforced one layer down: the
// composition Coordinator revalidates the selection on every Acquire, and
// validateSelectionLocked compares a.Scope against the registered provider.
//
// A Selection whose Scope does not match the registered provider must be
// refused when the coordinator is built, before any carrier entry.
func TestASelectionWhoseScopeDoesNotMatchIsRefused(t *testing.T) {
	process := &recordingProcess{payload: `{}`}
	in := inputs(t, process, nil)

	// Keep the ProviderID and offer, but claim another scope. The registry
	// registered this provider under "reference/quote", so this selection no
	// longer describes the provider it names.
	in.Selected.Scope = "another/scope"

	report := Run(in, []Request{{RequestID: "r1", Payload: json.RawMessage(`{}`)}})

	if process.calls != 0 {
		t.Fatalf("the process was entered %d times despite a scope mismatch", process.calls)
	}
	if report.Ran != 0 {
		t.Fatalf("a scope mismatch was counted as an invocation: ran=%d", report.Ran)
	}
	if report.Refused != 1 {
		t.Fatalf("refused = %d, want 1", report.Refused)
	}
	if report.Outcomes[0].CarrierInvoked {
		t.Fatal("a pre-entry refusal must not record carrier entry")
	}
}

// The matching selection still runs, so the check above is not refusing
// everything.
func TestTheMatchingScopeStillRuns(t *testing.T) {
	process := &recordingProcess{payload: `{}`}
	in := inputs(t, process, nil)
	if in.Selected.Scope != "reference/quote" {
		t.Fatalf("fixture scope = %q", in.Selected.Scope)
	}
	report := Run(in, []Request{{RequestID: "r1", Payload: json.RawMessage(`{}`)}})
	if report.Ran != 1 || report.Refused != 0 {
		t.Fatalf("ran=%d refused=%d, want 1 and 0", report.Ran, report.Refused)
	}
	if process.calls != 1 {
		t.Fatalf("the process was entered %d times, want 1", process.calls)
	}
}

// A Selection with the right scope but a stale registry revision must also be
// refused: revision and epoch are the other half of "this selection still
// describes the provider".
func TestASelectionWithAStaleRevisionIsRefused(t *testing.T) {
	process := &recordingProcess{payload: `{}`}
	in := inputs(t, process, nil)
	in.Selected.Revision = 99

	report := Run(in, []Request{{RequestID: "r1", Payload: json.RawMessage(`{}`)}})
	if process.calls != 0 || report.Ran != 0 || report.Refused != 1 {
		t.Fatalf("a stale revision was not refused: calls=%d ran=%d refused=%d",
			process.calls, report.Ran, report.Refused)
	}
}

// keep the composition import meaningful in this file.
var _ = composition.Binding{}
