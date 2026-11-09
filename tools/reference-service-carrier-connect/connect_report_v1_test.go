package servicecarrierconnect

import (
	"encoding/json"
	"strings"
	"testing"

	pruntime "tei/internal/plugin/runtime"
)

// The original C-2 evidence was recorded as v0. Its raw SHA and policy
// fingerprint must not be silently reassigned to the new report semantics.
func TestC2ReportUsesDistinctV1SchemaAndRuleIdentity(t *testing.T) {
	report := Run(inputs(t, &recordingProcess{payload: `{}`}, nil), nil)
	if report.Schema != "tei.reference.service-carrier-connect/v1" ||
		report.RuleVersion != "v1" ||
		report.RuleFingerprint == "sha256:b0b27eb27b498aece0fcfdb39df3228422c47643dd242ff405107fb51b9533f9" ||
		!strings.HasPrefix(report.RuleFingerprint, "sha256:") ||
		len(report.RuleFingerprint) != len("sha256:")+64 ||
		report.Ran != 0 || report.Refused != 0 || report.Unverified != 0 {
		t.Fatalf("C-2 report reused v0 or fabricated an invocation: %+v", report)
	}
}

// The code in an Outcome must not say 'refused' after the carrier was entered.
// The result is a local invocation witness, not proof the Process was called
// or any effect committed.
func TestC2V1CodesDoNotEquatePreflightRefusalAndPostEntryUnverified(t *testing.T) {
	t.Run("C2V1-PRE-route-unavailable", func(t *testing.T) {
		impl := &recordingProcess{payload: `{}`}
		got := Run(inputs(t, impl, errNoRoute), []Request{
			{RequestID: "r1", Payload: json.RawMessage(`{}`)},
		})
		if got.Ran != 0 || got.Refused != 1 || got.Unverified != 0 ||
			len(got.Outcomes) != 1 || got.Outcomes[0].CarrierInvoked ||
			!strings.HasPrefix(got.Outcomes[0].Error, RefusedRouteUnavailable+":") ||
			impl.calls != 0 {
			t.Fatalf("preflight rejection was incorrectly labeled: %+v; calls=%d", got, impl.calls)
		}
	})
	t.Run("C2V1-POST-process-error", func(t *testing.T) {
		impl := &failingProcess{}
		in := inputs(t, &recordingProcess{payload: `{}`}, nil)
		in.Runtime.Processes = impl
		got := Run(in, []Request{{RequestID: "r1", Payload: json.RawMessage(`{}`)}})
		if got.Ran != 1 || got.Refused != 0 || got.Unverified != 1 ||
			len(got.Outcomes) != 1 || !got.Outcomes[0].CarrierInvoked ||
			!strings.HasPrefix(got.Outcomes[0].Error, UnverifiedInvocation+":") ||
			impl.calls != 1 {
			t.Fatalf("post-entry error was mislabeled preflight: %+v; calls=%d", got, impl.calls)
		}
	})
	t.Run("C2V1-POST-artifact-deny-no-process", func(t *testing.T) {
		impl := &recordingProcess{payload: `{}`}
		in := inputs(t, impl, nil)
		in.Runtime.Verifier = gatedVerifier{refuse: true}
		got := Run(in, []Request{{RequestID: "r1", Payload: json.RawMessage(`{}`)}})
		if got.Ran != 1 || got.Refused != 0 || got.Unverified != 1 ||
			len(got.Outcomes) != 1 || !got.Outcomes[0].CarrierInvoked ||
			!strings.HasPrefix(got.Outcomes[0].Error, UnverifiedInvocation+":") ||
			impl.calls != 0 {
			t.Fatalf("carrier entry or provider process reach was conflated: %+v; calls=%d", got, impl.calls)
		}
	})
}

var _ pruntime.ProcessLauncher = (*failingProcess)(nil)
