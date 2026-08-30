package evidence_test

import (
	"context"
	"testing"

	"tei/internal/core"
	"tei/internal/deployment"
	"tei/internal/deployment/durable"
	devidence "tei/internal/deployment/evidence"
	"tei/internal/environment"
)

type fakeEvidence struct{ id string }

func (f fakeEvidence) ID() string { return f.id }
func (f fakeEvidence) Supports(k deployment.EvidenceKind) bool {
	return k == deployment.EvidenceRolloutHealth
}
func (f fakeEvidence) Collect(_ context.Context, r devidence.Request) (durable.Evidence, error) {
	return durable.Evidence{ID: "health-1", Outcome: durable.EvidencePass}, nil
}

func TestEvidenceBinding(t *testing.T) {
	env := environment.Environment{ID: "production", EvidenceProviders: []environment.EvidenceProviderBinding{{Kind: deployment.EvidenceRolloutHealth, Provider: "health.reference"}}}
	r := devidence.Router{Providers: map[core.CanonicalID]devidence.Provider{"health.reference": fakeEvidence{id: "health.reference"}}}
	rec := durable.ActionRecord{Action: durable.Action{ID: "app/health-1000", Mode: durable.ModeEvidence, EvidenceKind: durable.EvidenceRolloutHealth, Application: "order_entry", Version: "3.8.0", TrafficBasisPoints: 1000}}
	e, err := r.Collect(context.Background(), env, rec)
	if err != nil {
		t.Fatal(err)
	}
	if e.ActionID != rec.ID || e.Kind != durable.EvidenceRolloutHealth || e.SourceRef == "" {
		t.Fatalf("unexpected evidence: %+v", e)
	}
}
