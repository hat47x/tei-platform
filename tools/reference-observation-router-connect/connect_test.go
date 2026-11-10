package observationrouterconnect

import (
	"context"
	"errors"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/deployment"
	"tei/internal/deployment/observation"
	"tei/internal/environment"
)

// recordingProvider counts entries into Observe and returns a fixed fragment.
// It proves the path was entered, not that a Provider works.
type recordingProvider struct {
	id    core.CanonicalID
	kind  deployment.ObservationKind
	calls int
	// answer controls what Observe returns.
	answer string
}

func (p *recordingProvider) ID() string { return string(p.id) }

func (p *recordingProvider) Supports(k deployment.ObservationKind) bool { return k == p.kind }

func (p *recordingProvider) Observe(_ context.Context, req observation.Request) (observation.Fragment, error) {
	p.calls++
	switch p.answer {
	case "error":
		return observation.Fragment{}, errProvider
	case "empty":
		return observation.Fragment{
			ProviderID: p.id, Environment: req.Environment, Kind: req.Kind,
		}, nil
	case "wrong-id":
		return observation.Fragment{
			ProviderID: core.MustCanonicalID("someone.else"), Environment: req.Environment, Kind: req.Kind,
		}, nil
	default:
		return observation.Fragment{
			ProviderID: p.id, Environment: req.Environment, Kind: req.Kind,
			ObservedAt: "2026-11-09T00:00:00Z",
		}, nil
	}
}

type providerFailure struct{}

func (providerFailure) Error() string { return "the provider failed after it was entered" }

var errProvider = providerFailure{}

func envWith(pairs ...struct {
	kind deployment.ObservationKind
	id   string
}) environment.Environment {
	env := environment.Environment{ID: "development"}
	for _, p := range pairs {
		env.ObservationProviders = append(env.ObservationProviders,
			environment.ObservationProviderBinding{Kind: p.kind, Provider: core.CanonicalID(p.id)})
	}
	return env
}

func pair(kind deployment.ObservationKind, id string) struct {
	kind deployment.ObservationKind
	id   string
} {
	return struct {
		kind deployment.ObservationKind
		id   string
	}{kind, id}
}

// The binding selects the provider, and the existing Router enters it. This is
// the gap the survey found: resolution existed, entry did not run outside tests.
func TestADeclaredProviderIsResolvedAndEntered(t *testing.T) {
	env := envWith(pair(deployment.ObservationDatabaseState, "db.observer"))
	provider := &recordingProvider{id: "db.observer", kind: deployment.ObservationDatabaseState}
	report := Run(context.Background(), Input{
		Environment: env,
		Providers:   map[core.CanonicalID]observation.Provider{"db.observer": provider},
	})
	if report.Observed != 1 || report.Refused != 0 || report.Invoked != 1 {
		t.Fatalf("report = %+v", report)
	}
	if provider.calls != 1 {
		t.Fatalf("the provider was entered %d times", provider.calls)
	}
	out := report.Outcomes[0]
	if !out.Resolved || !out.Selected || !out.Invoked || out.Status != StatusObserved {
		t.Fatalf("outcome = %+v", out)
	}
}

// A provider the declaration names but the Host did not register is refused
// before any entry, and the two facts stay distinguishable.
func TestAnUnregisteredProviderIsRefusedBeforeEntry(t *testing.T) {
	env := envWith(pair(deployment.ObservationDatabaseState, "db.observer"))
	report := Run(context.Background(), Input{
		Environment: env,
		Providers:   map[core.CanonicalID]observation.Provider{},
	})
	if report.Observed != 0 || report.Refused != 1 || report.Invoked != 0 {
		t.Fatalf("report = %+v", report)
	}
	out := report.Outcomes[0]
	if !out.Resolved || out.Selected {
		t.Fatalf("resolution and selection were conflated: %+v", out)
	}
	if out.Status != StatusNotRegistered {
		t.Fatalf("status = %q", out.Status)
	}
}

// A failure after entry is not a refusal: the provider ran, so it is counted as
// invoked even though the answer is unusable.
func TestAFailureAfterEntryIsCountedAsInvoked(t *testing.T) {
	env := envWith(pair(deployment.ObservationDatabaseState, "db.observer"))
	provider := &recordingProvider{
		id: "db.observer", kind: deployment.ObservationDatabaseState, answer: "error",
	}
	report := Run(context.Background(), Input{
		Environment: env,
		Providers:   map[core.CanonicalID]observation.Provider{"db.observer": provider},
	})
	if report.Invoked != 1 {
		t.Fatalf("entry was not counted: %+v", report)
	}
	if report.Observed != 0 {
		t.Fatalf("a failed observation was counted as observed: %+v", report)
	}
	if report.Outcomes[0].Status != StatusObserveFailed {
		t.Fatalf("status = %q", report.Outcomes[0].Status)
	}
}

// A fragment naming another provider is not an observation of this one.
func TestAFragmentForAnotherProviderIsRefused(t *testing.T) {
	env := envWith(pair(deployment.ObservationDatabaseState, "db.observer"))
	provider := &recordingProvider{
		id: "db.observer", kind: deployment.ObservationDatabaseState, answer: "wrong-id",
	}
	report := Run(context.Background(), Input{
		Environment: env,
		Providers:   map[core.CanonicalID]observation.Provider{"db.observer": provider},
	})
	if report.Observed != 0 {
		t.Fatalf("a mismatched fragment was accepted: %+v", report)
	}
	// The existing Router rejects the mismatch itself, so the outcome is a
	// post-entry failure rather than a refusal.
	if report.Outcomes[0].Status != StatusObserveFailed {
		t.Fatalf("status = %q", report.Outcomes[0].Status)
	}
	if !report.Outcomes[0].Invoked {
		t.Fatal("entry into the provider must still be recorded")
	}
}

// An environment with no declaration is refused, not guessed.
func TestNoDeclarationIsRefused(t *testing.T) {
	report := Run(context.Background(), Input{Environment: environment.Environment{ID: "development"}})
	if len(report.Outcomes) != 0 {
		t.Fatalf("an empty declaration produced outcomes: %+v", report.Outcomes)
	}
	// With no kinds requested and none declared, there is nothing to attempt.
	if report.Observed != 0 || report.Refused != 0 {
		t.Fatalf("report = %+v", report)
	}
}

// An empty answer is still an entry: the reference distinguishes "entered and
// answered with nothing" from "entered and answered".
func TestAnEmptyFragmentIsStillAnObservation(t *testing.T) {
	env := envWith(pair(deployment.ObservationDatabaseState, "db.observer"))
	provider := &recordingProvider{
		id: "db.observer", kind: deployment.ObservationDatabaseState, answer: "empty",
	}
	report := Run(context.Background(), Input{
		Environment: env,
		Providers:   map[core.CanonicalID]observation.Provider{"db.observer": provider},
	})
	if report.Observed != 1 || report.Invoked != 1 {
		t.Fatalf("report = %+v", report)
	}
	if report.Outcomes[0].FragmentFields != 0 {
		t.Fatalf("an empty fragment reported fields: %+v", report.Outcomes[0])
	}
}

func TestReportStatesAuthorityAndItsLimits(t *testing.T) {
	report := Run(context.Background(), Input{Environment: environment.Environment{ID: "development"}})
	if report.Authority != Authority || report.Schema != Schema || report.RuleFingerprint == "" {
		t.Fatalf("bad header: %+v", report)
	}
	if len(report.Notes) == 0 {
		t.Fatal("the report must state what it does not establish")
	}
	for _, note := range report.Notes {
		if strings.Contains(note, "no authentication") {
			return
		}
	}
	t.Fatal("the notes do not state that no authority is issued")
}

var _ = errors.New
