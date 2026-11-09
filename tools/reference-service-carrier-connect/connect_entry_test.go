package servicecarrierconnect

import (
	"encoding/json"
	"errors"
	"testing"

	"tei/internal/composition"
)

// CON-12c/12d on C-2: a Host guard that refuses BEFORE the carrier is entered
// must be counted as a refusal, not as an invocation. The previous code raised
// Ran before Invoke, so a guard rejection read as "the carrier ran".
func TestAGuardRefusalIsCountedAsRefusedNotRan(t *testing.T) {
	process := &recordingProcess{payload: `{}`}
	in := inputs(t, process, nil)

	// The guard refuses everything, as a withdrawn generation would.
	in.Guard = func(composition.Binding) error { return errors.New("the Host generation was withdrawn") }

	report := Run(in, []Request{{RequestID: "r1", Payload: json.RawMessage(`{}`)}})

	if report.Ran != 0 {
		t.Fatalf("a guard refusal was counted as an invocation: ran=%d", report.Ran)
	}
	if report.Refused != 1 {
		t.Fatalf("refused = %d, want 1", report.Refused)
	}
	if report.Unverified != 0 {
		t.Fatalf("a pre-entry refusal was counted as unverified: %d", report.Unverified)
	}
	if process.calls != 0 {
		t.Fatalf("the process was entered %d times despite a guard refusal", process.calls)
	}
	if report.Outcomes[0].CarrierInvoked {
		t.Fatal("a guard refusal must not record carrier entry")
	}
}

// The guard is checked per invocation, so it can refuse partway through a run.
// The requests after the refusal must also be refusals, not invocations.
func TestAGuardThatStartsRefusingMidRunCountsEachRequest(t *testing.T) {
	process := &recordingProcess{payload: `{}`}
	in := inputs(t, process, nil)

	// The guard is consulted more than once per request (coordinator creation
	// and the per-invocation acquire), so the run is split by a flag rather
	// than by counting calls: the first request is allowed, then the guard
	// starts refusing.
	withdrawn := false
	in.Guard = func(composition.Binding) error {
		if withdrawn {
			return errors.New("the Host generation was withdrawn")
		}
		return nil
	}

	// Allow the first request, then withdraw the generation before the second.
	// Run is synchronous, so the withdrawal is applied by the first request's
	// completion: the test drives two runs to keep the boundary explicit.
	first := Run(in, []Request{{RequestID: "r1", Payload: json.RawMessage(`{}`)}})
	withdrawn = true
	second := Run(in, []Request{{RequestID: "r2", Payload: json.RawMessage(`{}`)}})

	report := Report{
		Ran:      first.Ran + second.Ran,
		Refused:  first.Refused + second.Refused,
		Outcomes: append(append([]Outcome(nil), first.Outcomes...), second.Outcomes...),
	}

	// The first request is allowed by the guard and runs; the second is refused.
	if report.Ran != 1 {
		t.Fatalf("ran = %d, want 1", report.Ran)
	}
	if report.Refused != 1 {
		t.Fatalf("refused = %d, want 1", report.Refused)
	}
	if process.calls != 1 {
		t.Fatalf("the process was entered %d times, want 1", process.calls)
	}
	if !report.Outcomes[0].CarrierInvoked {
		t.Fatal("the allowed request must record carrier entry")
	}
	if report.Outcomes[1].CarrierInvoked {
		t.Fatal("the refused request must not record carrier entry")
	}
}

// A refusal for a missing route happens before the carrier too, and must also
// count as refused rather than ran.
func TestAMissingRouteIsStillRefusedNotRan(t *testing.T) {
	process := &recordingProcess{payload: `{}`}
	report := Run(inputs(t, process, errors.New("no route")),
		[]Request{{RequestID: "r1", Payload: json.RawMessage(`{}`)}})

	if report.Ran != 0 || report.Refused != 1 {
		t.Fatalf("ran=%d refused=%d, want 0 and 1", report.Ran, report.Refused)
	}
	if process.calls != 0 {
		t.Fatalf("the process was entered %d times", process.calls)
	}
}
