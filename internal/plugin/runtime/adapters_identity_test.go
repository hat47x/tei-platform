package runtime

import (
	"testing"

	"tei/internal/core"
	"tei/internal/deployment"
	"tei/internal/deployment/durable"
	"tei/internal/deployment/evidence"
	"tei/internal/deployment/observation"
)

// A provider that leaves the identity empty gets it filled. This is the only
// case the adapter is allowed to write, and the Router relies on it.
func TestAnEmptyObservedIdentityIsFilled(t *testing.T) {
	req := observation.Request{
		Environment: core.MustCanonicalID("development"),
		Kind:        deployment.ObservationDatabaseState,
	}
	got := completeObservedFragment(observation.Fragment{}, req, "db.observer")
	if got.ProviderID != "db.observer" {
		t.Fatalf("provider id = %q", got.ProviderID)
	}
	if got.Environment != req.Environment {
		t.Fatalf("environment = %q", got.Environment)
	}
	if got.Kind != req.Kind {
		t.Fatalf("kind = %q", got.Kind)
	}
}

// A provider that names ANOTHER identity keeps it. The adapter must not
// overwrite it, because the Router compares this field and would otherwise
// never see the mismatch. This is the regression the fix closes.
func TestAForeignObservedIdentityIsNotOverwritten(t *testing.T) {
	req := observation.Request{
		Environment: core.MustCanonicalID("development"),
		Kind:        deployment.ObservationDatabaseState,
	}
	in := observation.Fragment{
		ProviderID:  "someone.else",
		Environment: req.Environment,
		Kind:        req.Kind,
	}
	got := completeObservedFragment(in, req, "db.observer")
	if got.ProviderID != "someone.else" {
		t.Fatalf("a foreign provider id was overwritten with %q", got.ProviderID)
	}
}

// A provider that names another environment keeps it too: the Router checks
// that field as well.
func TestAForeignObservedEnvironmentIsNotOverwritten(t *testing.T) {
	req := observation.Request{
		Environment: core.MustCanonicalID("development"),
		Kind:        deployment.ObservationDatabaseState,
	}
	in := observation.Fragment{
		ProviderID:  "db.observer",
		Environment: core.MustCanonicalID("production"),
		Kind:        req.Kind,
	}
	got := completeObservedFragment(in, req, "db.observer")
	if got.Environment != "production" {
		t.Fatalf("a foreign environment was overwritten with %q", got.Environment)
	}
}

// The evidence side has the same rule for the action id.
func TestAForeignEvidenceActionIsNotOverwritten(t *testing.T) {
	req := evidence.Request{ActionID: "gate-1", Kind: deployment.EvidenceRolloutHealth}
	in := durable.Evidence{ActionID: "someone.else", Kind: req.Kind}
	got := completeCollectedEvidence(in, req)
	if got.ActionID != "someone.else" {
		t.Fatalf("a foreign action id was overwritten with %q", got.ActionID)
	}
}

func TestAnEmptyEvidenceActionIsFilled(t *testing.T) {
	req := evidence.Request{ActionID: "gate-1", Kind: deployment.EvidenceRolloutHealth}
	got := completeCollectedEvidence(durable.Evidence{}, req)
	if got.ActionID != "gate-1" {
		t.Fatalf("action id = %q", got.ActionID)
	}
	if got.Kind != req.Kind {
		t.Fatalf("kind = %q", got.Kind)
	}
}
