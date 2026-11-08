package servicecarrierconnect

import (
	"context"
	"encoding/json"
	"testing"

	"tei/internal/plugin"
	pruntime "tei/internal/plugin/runtime"
)

// failingProcess records that it was entered and then fails. The carrier is
// already in flight, so an external effect may exist even though the answer is
// unusable. This is the counterfactual the counters must not hide.
type failingProcess struct{ calls int }

func (p *failingProcess) Invoke(
	_ context.Context, _ pruntime.ProcessCommand, _ pruntime.Request,
) (pruntime.Response, error) {
	p.calls++
	return pruntime.Response{}, errProcessFailed
}

type processFailure struct{}

func (processFailure) Error() string { return "the process failed after it was entered" }

var errProcessFailed = processFailure{}

// gatedVerifier refuses the artifact, so the carrier stops before the process.
type gatedVerifier struct{ refuse bool }

func (g gatedVerifier) Verify(context.Context, plugin.Identity, plugin.Artifact) (pruntime.ArtifactVerification, error) {
	if g.refuse {
		return pruntime.ArtifactVerification{}, errArtifactRefused
	}
	return pruntime.ArtifactVerification{VerifierID: "reference-only", DigestVerified: true}, nil
}

type artifactFailure struct{}

func (artifactFailure) Error() string { return "the artifact was refused" }

var errArtifactRefused = artifactFailure{}

// errNoRoute stands for "the Host has no route for this generation".
type routeFailure struct{}

func (routeFailure) Error() string { return "no current route" }

var errNoRoute = routeFailure{}

func TestAnArtifactRefusalHappensInsideTheCarrier(t *testing.T) {
	// The verifier refuses, but the refusal happens INSIDE the carrier, after
	// the reference has already entered it. The process is never reached, yet
	// the carrier was entered, so this is Ran+Unverified, not Refused.
	//
	// This is the honest reading: "the reference refused" and "the carrier
	// stopped before doing anything" are different statements, and only the
	// reference's own pre-checks can claim the latter.
	in := inputs(t, &recordingProcess{payload: `{}`}, nil)
	in.Runtime = pruntime.Registry{
		Processes:         &recordingProcess{payload: `{}`},
		Verifier:          gatedVerifier{refuse: true},
		ServiceAcceptance: acceptingService{},
	}
	report := Run(in, []Request{{RequestID: "r1", Payload: json.RawMessage(`{}`)}})
	if report.Ran != 1 || report.Refused != 0 || report.Unverified != 1 {
		t.Fatalf("an artifact refusal was counted as not-run: %+v", report)
	}
	if !report.Outcomes[0].CarrierInvoked {
		t.Fatal("entry into the carrier must be recorded even when it refuses inside")
	}
}

func TestAFailureAfterEntryIsUnverifiedNotRefused(t *testing.T) {
	// The process is entered and then fails. v0 would have counted this as a
	// refusal, which reads as "nothing happened" and hides a possible effect.
	in := inputs(t, &recordingProcess{payload: `{}`}, nil)
	failing := &failingProcess{}
	in.Runtime = pruntime.Registry{
		Processes:         failing,
		Verifier:          acceptingVerifier{},
		ServiceAcceptance: acceptingService{},
	}
	report := Run(in, []Request{{RequestID: "r1", Payload: json.RawMessage(`{}`)}})
	if failing.calls != 1 {
		t.Fatalf("the process was entered %d times", failing.calls)
	}
	if report.Ran != 1 || report.Refused != 0 || report.Unverified != 1 {
		t.Fatalf("a post-entry failure was counted as not-run: %+v", report)
	}
	if !report.Outcomes[0].CarrierInvoked {
		t.Fatal("a post-entry failure must record carrier entry")
	}
}

func TestAnUnavailableRouteIsTheOnlyPreInvocationRefusal(t *testing.T) {
	// Two requests, no route. The reference refuses both before entering the
	// carrier, so Refused counts both and Ran stays zero.
	in := inputs(t, &recordingProcess{payload: `{}`}, errNoRoute)
	report := Run(in, []Request{
		{RequestID: "r1", Payload: json.RawMessage(`{}`)},
		{RequestID: "r2", Payload: json.RawMessage(`{}`)},
	})
	if report.Refused != 2 || report.Ran != 0 || report.Unverified != 0 {
		t.Fatalf("report = %+v", report)
	}
	for _, o := range report.Outcomes {
		if o.CarrierInvoked {
			t.Fatal("a pre-invocation refusal must not record carrier entry")
		}
	}
}
