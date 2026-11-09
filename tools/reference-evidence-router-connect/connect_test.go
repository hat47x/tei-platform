package evidencerouterconnect

import (
	"context"
	"testing"

	"tei/internal/core"
	"tei/internal/deployment"
	"tei/internal/deployment/durable"
	"tei/internal/deployment/evidence"
	"tei/internal/environment"
)

// recordingProvider counts entries into Collect and answers with a fixed
// outcome. It proves the path was entered, not that the evidence is true.
type recordingProvider struct {
	id      core.CanonicalID
	kind    deployment.EvidenceKind
	answer  durable.EvidenceOutcome
	fail    bool
	calls   int
	wrongID bool
}

func (p *recordingProvider) ID() string { return string(p.id) }

func (p *recordingProvider) Supports(k deployment.EvidenceKind) bool { return k == p.kind }

func (p *recordingProvider) Collect(_ context.Context, req evidence.Request) (durable.Evidence, error) {
	p.calls++
	if p.fail {
		return durable.Evidence{}, errProvider
	}
	actionID := req.ActionID
	if p.wrongID {
		actionID = "someone.else"
	}
	return durable.Evidence{
		ID: "evidence-1", ActionID: actionID, Kind: req.Kind, Outcome: p.answer,
	}, nil
}

type providerFailure struct{}

func (providerFailure) Error() string { return "the provider failed after it was entered" }

var errProvider = providerFailure{}

func envWith(kind deployment.EvidenceKind, id string) environment.Environment {
	return environment.Environment{
		ID: "development",
		EvidenceProviders: []environment.EvidenceProviderBinding{{
			Kind: kind, Provider: core.CanonicalID(id),
		}},
	}
}

func record(id string, kind deployment.EvidenceKind) durable.ActionRecord {
	return durable.ActionRecord{
		Action: durable.Action{
			ID: id, Kind: durable.ActionRolloutHealthGate, Mode: durable.ModeEvidence,
			EvidenceKind: kind,
		},
	}
}

// The binding selects the provider, and the existing Router enters it. This is
// the same four-stage shape as the observation reference.
func TestADeclaredProviderIsResolvedAndEntered(t *testing.T) {
	provider := &recordingProvider{
		id: "health.reference", kind: deployment.EvidenceRolloutHealth,
		answer: durable.EvidencePass,
	}
	report := Run(context.Background(), Input{
		Environment: envWith(deployment.EvidenceRolloutHealth, "health.reference"),
		Providers:   map[core.CanonicalID]evidence.Provider{"health.reference": provider},
		Records:     []durable.ActionRecord{record("gate-1", deployment.EvidenceRolloutHealth)},
	})
	if report.Collected != 1 || report.Refused != 0 || report.Invoked != 1 {
		t.Fatalf("report = %+v", report)
	}
	if provider.calls != 1 {
		t.Fatalf("the provider was entered %d times", provider.calls)
	}
	out := report.Outcomes[0]
	if !out.Resolved || !out.Selected || !out.Invoked || out.Status != StatusCollected {
		t.Fatalf("outcome = %+v", out)
	}
	if out.Outcome != string(durable.EvidencePass) {
		t.Fatalf("the provider's outcome was not carried through: %+v", out)
	}
}

// A record that is not an evidence gate is refused with its own status, so the
// reason is visible instead of hidden behind a generic router error.
func TestANonEvidenceRecordIsRefusedWithItsOwnStatus(t *testing.T) {
	report := Run(context.Background(), Input{
		Environment: envWith(deployment.EvidenceRolloutHealth, "health.reference"),
		Providers:   map[core.CanonicalID]evidence.Provider{},
		Records: []durable.ActionRecord{{
			Action: durable.Action{ID: "db-1", Kind: durable.ActionDatabaseRelease, Mode: durable.ModeCommand},
		}},
	})
	if report.Refused != 1 || report.Invoked != 0 {
		t.Fatalf("report = %+v", report)
	}
	if report.Outcomes[0].Status != StatusNotEvidence {
		t.Fatalf("status = %q", report.Outcomes[0].Status)
	}
	if report.Outcomes[0].Resolved {
		t.Fatal("a non-evidence record must not report a resolved binding")
	}
}

// A provider the declaration names but the Host did not register is refused
// before entry, and resolution stays visible.
func TestAnUnregisteredProviderIsRefusedBeforeEntry(t *testing.T) {
	report := Run(context.Background(), Input{
		Environment: envWith(deployment.EvidenceRolloutHealth, "health.reference"),
		Providers:   map[core.CanonicalID]evidence.Provider{},
		Records:     []durable.ActionRecord{record("gate-1", deployment.EvidenceRolloutHealth)},
	})
	if report.Collected != 0 || report.Refused != 1 || report.Invoked != 0 {
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

// A failure after entry is counted as invoked: the provider ran, so the
// reference must not present it as never having happened.
func TestAFailureAfterEntryIsCountedAsInvoked(t *testing.T) {
	provider := &recordingProvider{
		id: "health.reference", kind: deployment.EvidenceRolloutHealth, fail: true,
	}
	report := Run(context.Background(), Input{
		Environment: envWith(deployment.EvidenceRolloutHealth, "health.reference"),
		Providers:   map[core.CanonicalID]evidence.Provider{"health.reference": provider},
		Records:     []durable.ActionRecord{record("gate-1", deployment.EvidenceRolloutHealth)},
	})
	if report.Invoked != 1 {
		t.Fatalf("entry was not counted: %+v", report)
	}
	if report.Collected != 0 {
		t.Fatalf("a failed collection was counted as collected: %+v", report)
	}
	if report.Outcomes[0].Status != StatusCollectFailed {
		t.Fatalf("status = %q", report.Outcomes[0].Status)
	}
}

// Evidence naming another action is rejected by the existing Router, so the
// reference reports a post-entry failure rather than a refusal.
func TestEvidenceForAnotherActionIsRejectedAfterEntry(t *testing.T) {
	provider := &recordingProvider{
		id: "health.reference", kind: deployment.EvidenceRolloutHealth,
		answer: durable.EvidencePass, wrongID: true,
	}
	report := Run(context.Background(), Input{
		Environment: envWith(deployment.EvidenceRolloutHealth, "health.reference"),
		Providers:   map[core.CanonicalID]evidence.Provider{"health.reference": provider},
		Records:     []durable.ActionRecord{record("gate-1", deployment.EvidenceRolloutHealth)},
	})
	if report.Collected != 0 {
		t.Fatalf("mismatched evidence was accepted: %+v", report)
	}
	if report.Outcomes[0].Status != StatusCollectFailed {
		t.Fatalf("status = %q", report.Outcomes[0].Status)
	}
	if !report.Outcomes[0].Invoked {
		t.Fatal("entry into the provider must still be recorded")
	}
}

// An environment with no declaration is refused, not guessed.
func TestNoDeclarationIsRefused(t *testing.T) {
	report := Run(context.Background(), Input{
		Environment: environment.Environment{ID: "development"},
		Records:     []durable.ActionRecord{record("gate-1", deployment.EvidenceRolloutHealth)},
	})
	if report.Refused != 1 {
		t.Fatalf("report = %+v", report)
	}
	if report.Outcomes[0].Status != StatusBindingMissig {
		t.Fatalf("status = %q", report.Outcomes[0].Status)
	}
	if report.Outcomes[0].ProviderID != "" {
		t.Fatalf("a missing declaration produced an id: %q", report.Outcomes[0].ProviderID)
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
	found := false
	for _, note := range report.Notes {
		if note == "a pass outcome is the provider's claim, not proof that external state matches it" {
			found = true
		}
	}
	if !found {
		t.Fatal("the notes do not state that a pass outcome is only a claim")
	}
}
