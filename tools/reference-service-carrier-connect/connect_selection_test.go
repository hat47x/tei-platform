package servicecarrierconnect

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"tei/internal/composition"
	"tei/internal/core"
	"tei/internal/servicehost"
)

// CON-12c on the ServiceProviderSelection layer: servicehost compares the
// route's selection against the pinned Binding, including ArtifactDigest. A
// route whose selection carries a different digest must be refused BEFORE the
// carrier runs, not invoked with the wrong artifact.
//
// This is the layer the identity survey found carries digest/scope fields
// (runtime.ServiceProviderSelection), unlike the observation/evidence
// declarations which carry neither.
func TestARouteSelectionWithAnotherDigestStopsBeforeTheProvider(t *testing.T) {
	process := &recordingProcess{payload: `{}`}
	in := inputs(t, process, nil)

	original := in.Current
	in.Current = func(
		ctx context.Context, app, envID core.CanonicalID, b composition.Binding,
	) (servicehost.SelectedServiceCarrierRoute, error) {
		r, err := original(ctx, app, envID, b)
		if err != nil {
			return r, err
		}
		// Change only the selection's digest. The Binding stays pinned, so the
		// only way to notice is the selection-vs-Binding comparison.
		r.Selection.ArtifactDigest = "sha256:" + strings.Repeat("c", 64)
		return r, nil
	}

	report := Run(in, []Request{{RequestID: "r1", Payload: json.RawMessage(`{}`)}})

	// The Provider is never called: servicehost compares the selection against
	// the pinned Binding before handing the request to the runtime.
	if process.calls != 0 {
		t.Fatalf("the process was entered %d times despite a selection digest mismatch", process.calls)
	}
	// The CARRIER was entered (the Host guard passed, the lease was acquired), so
	// this is not a refusal. It is ran + unverified: entered, then stopped before
	// the Provider answered. The README states this distinction explicitly.
	if report.Ran != 1 {
		t.Fatalf("ran = %d, want 1 (the carrier was entered)", report.Ran)
	}
	if report.Unverified != 1 {
		t.Fatalf("unverified = %d, want 1", report.Unverified)
	}
	if report.Refused != 0 {
		t.Fatalf("a post-entry mismatch was counted as a refusal: refused=%d", report.Refused)
	}
	if !report.Outcomes[0].CarrierInvoked {
		t.Fatal("the carrier was entered, so the outcome must record it")
	}
}

// The same mismatch for the contract version must also be refused.
func TestARouteSelectionWithAnotherContractVersionStopsBeforeTheProvider(t *testing.T) {
	process := &recordingProcess{payload: `{}`}
	in := inputs(t, process, nil)

	original := in.Current
	in.Current = func(
		ctx context.Context, app, envID core.CanonicalID, b composition.Binding,
	) (servicehost.SelectedServiceCarrierRoute, error) {
		r, err := original(ctx, app, envID, b)
		if err != nil {
			return r, err
		}
		r.Selection.ContractVersion = "9.9.9"
		return r, nil
	}

	report := Run(in, []Request{{RequestID: "r1", Payload: json.RawMessage(`{}`)}})
	if process.calls != 0 {
		t.Fatalf("the process was entered %d times", process.calls)
	}
	if report.Ran != 1 || report.Unverified != 1 || report.Refused != 0 {
		t.Fatalf("contract mismatch: ran=%d unverified=%d refused=%d, want 1/1/0",
			report.Ran, report.Unverified, report.Refused)
	}
}

// A route whose selection matches the Binding must still run. This is the
// control: the check above is not simply refusing everything.
func TestAMatchingSelectionStillRuns(t *testing.T) {
	process := &recordingProcess{payload: `{}`}
	report := Run(inputs(t, process, nil),
		[]Request{{RequestID: "r1", Payload: json.RawMessage(`{}`)}})
	if report.Ran != 1 || report.Refused != 0 {
		t.Fatalf("ran=%d refused=%d, want 1 and 0", report.Ran, report.Refused)
	}
	if process.calls != 1 {
		t.Fatalf("the process was entered %d times, want 1", process.calls)
	}
}
