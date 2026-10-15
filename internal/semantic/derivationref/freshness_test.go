package derivationref_test

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"tei/internal/semantic/derivationref"
)

type d3WitnessVerifier struct {
	currentCalls int
	alignedCalls int
	denyCurrent  bool
	denyAligned  bool
}

func (v *d3WitnessVerifier) VerifyCurrent(_ context.Context, _ derivationref.CacheWitness) error {
	v.currentCalls++
	if v.denyCurrent {
		return errors.New("current authority or revisions changed")
	}
	return nil
}

func (v *d3WitnessVerifier) VerifyAligned(_ context.Context, _ derivationref.CacheWitness) error {
	v.alignedCalls++
	if v.denyAligned {
		return errors.New("providers cannot attest shared snapshot")
	}
	return nil
}

func d3Witness() derivationref.CacheWitness {
	seen := time.Date(2026, 10, 15, 1, 0, 0, 0, time.UTC)
	return derivationref.CacheWitness{
		RuleID:             "available",
		RuleRevision:       "v1",
		TargetID:           "product.available",
		SemanticRevision:   "semantic-1",
		AcceptanceRevision: "accepted-1",
		ProviderRevision:   "provider-1",
		Inputs: []derivationref.Revision{
			{ID: "product.reserved", Revision: "1"},
			{ID: "product.stock", Revision: "1"},
		},
		Sources: []derivationref.SourceObservation{{
			SourceID: "inventory",
			Revision: "source-1",
			SourceAsOf: seen.Add(-time.Minute),
			ObservedAt: seen,
		}},
		Scope: derivationref.CacheScope{
			Environment: "dev",
			Tenant: "tenant-a",
			Subject: "subject-a",
			PermissionEpoch: "policy-1",
		},
	}
}

func TestD3PureD1RequiresNoCacheDeclaration(t *testing.T) {
	value, err := derivationref.Multiply(
		derivationref.Input{State: derivationref.Present, Decimal: "2"},
		derivationref.Input{State: derivationref.Present, Decimal: "300"},
	)
	if err != nil || value.State != derivationref.Present || value.Decimal != "600" {
		t.Fatalf("pure result without cache metadata: %#v, %v", value, err)
	}
}

func TestD3CacheWitnessVersionsScopeAndSourceTime(t *testing.T) {
	witness := d3Witness()
	v := &d3WitnessVerifier{}
	policy := derivationref.CachePolicy{MaxAge: time.Minute}
	key, err := derivationref.QualifyCache(context.Background(), witness, policy, v)
	if err != nil || key == "" || v.currentCalls != 1 {
		t.Fatalf("qualified witness: key=%q verifier=%#v err=%v", key, v, err)
	}
	modifications := []struct {
		name string
		edit func(*derivationref.CacheWitness)
	}{
		{"input revision", func(w *derivationref.CacheWitness) { w.Inputs[0].Revision = "2" }},
		{"rule revision", func(w *derivationref.CacheWitness) { w.RuleRevision = "v2" }},
		{"semantic revision", func(w *derivationref.CacheWitness) { w.SemanticRevision = "semantic-2" }},
		{"acceptance revision", func(w *derivationref.CacheWitness) { w.AcceptanceRevision = "accepted-2" }},
		{"provider revision", func(w *derivationref.CacheWitness) { w.ProviderRevision = "provider-2" }},
		{"tenant", func(w *derivationref.CacheWitness) { w.Scope.Tenant = "tenant-b" }},
		{"subject", func(w *derivationref.CacheWitness) { w.Scope.Subject = "subject-b" }},
		{"permission epoch", func(w *derivationref.CacheWitness) { w.Scope.PermissionEpoch = "policy-2" }},
		{"source revision", func(w *derivationref.CacheWitness) { w.Sources[0].Revision = "source-2" }},
		{"source observed time", func(w *derivationref.CacheWitness) { w.Sources[0].ObservedAt = w.Sources[0].ObservedAt.Add(time.Minute) }},
		{"source as-of time", func(w *derivationref.CacheWitness) { w.Sources[0].SourceAsOf = w.Sources[0].SourceAsOf.Add(time.Minute) }},
	}
	for _, tc := range modifications {
		t.Run(tc.name, func(t *testing.T) {
			next := d3Witness()
			tc.edit(&next)
			got, err := derivationref.QualifyCache(context.Background(), next, policy, v)
			if err != nil || got == key {
				t.Fatalf("different source or context reused same cache key: %q %v", got, err)
			}
		})
	}
	if derivationref.ReusableAt(time.Time{}, time.Now(), policy) ||
		derivationref.ReusableAt(time.Now(), time.Now().Add(2*time.Minute), policy) ||
		derivationref.ReusableAt(time.Now().Add(time.Minute), time.Now(), policy) {
		t.Fatal("unbounded/future/expired cache accepted")
	}
	observed := time.Date(2026, 10, 15, 1, 0, 0, 0, time.UTC)
	if !derivationref.ReusableAt(observed, observed.Add(30*time.Second), policy) {
		t.Fatal("fresh bounded result incorrectly rejected")
	}
}

func TestD3CacheFailClosedOnMissingOrForgedEvidence(t *testing.T) {
	policy := derivationref.CachePolicy{MaxAge: time.Minute}
	witness := d3Witness()
	badCases := []struct {
		name string
		edit func(*derivationref.CacheWitness)
	}{
		{"missing rule revision", func(w *derivationref.CacheWitness) { w.RuleRevision = "" }},
		{"missing source date", func(w *derivationref.CacheWitness) { w.Sources[0].SourceAsOf = time.Time{} }},
		{"source later than observation", func(w *derivationref.CacheWitness) { w.Sources[0].SourceAsOf = w.Sources[0].ObservedAt.Add(time.Hour) }},
		{"missing tenant", func(w *derivationref.CacheWitness) { w.Scope.Tenant = "" }},
		{"missing input revision", func(w *derivationref.CacheWitness) { w.Inputs[0].Revision = "" }},
		{"input duplicate", func(w *derivationref.CacheWitness) { w.Inputs = append(w.Inputs, w.Inputs[0]) }},
		{"input noncanonical ordering", func(w *derivationref.CacheWitness) { w.Inputs[0], w.Inputs[1] = w.Inputs[1], w.Inputs[0] }},
		{"duplicate source", func(w *derivationref.CacheWitness) { w.Sources = append(w.Sources, w.Sources[0]) }},
	}
	for _, tc := range badCases {
		t.Run(tc.name, func(t *testing.T) {
			test := d3Witness()
			tc.edit(&test)
			if _, err := derivationref.QualifyCache(context.Background(), test, policy, &d3WitnessVerifier{}); !errors.Is(err, derivationref.ErrCacheEvidence) {
				t.Fatalf("got %v, expected incomplete evidence", err)
			}
		})
	}
	if _, err := derivationref.QualifyCache(context.Background(), witness, policy, nil); !errors.Is(err, derivationref.ErrCacheDenied) {
		t.Fatalf("self-report without Host verifier accepted: %v", err)
	}
	v := &d3WitnessVerifier{denyCurrent: true}
	if _, err := derivationref.QualifyCache(context.Background(), witness, policy, v); !errors.Is(err, derivationref.ErrCacheDenied) {
		t.Fatalf("revoked current evidence accepted: %v", err)
	}
}

func TestD3MultipleSourcesNeverInferAtomicAlignment(t *testing.T) {
	w := d3Witness()
	other := w.Sources[0]
	other.SourceID = "pricing"
	w.Sources = append(w.Sources, other) // no snapshot attestation
	policy := derivationref.CachePolicy{MaxAge: time.Minute, RequireAligned: true}
	v := &d3WitnessVerifier{}
	if _, err := derivationref.QualifyCache(context.Background(), w, policy, v); !errors.Is(err, derivationref.ErrNotCoherent) {
		t.Fatalf("unproven multi-provider snapshot accepted: %v", err)
	}
	for i := range w.Sources {
		w.Sources[i].SnapshotRef = "source-attested-batch"
	}
	v.denyAligned = true
	if _, err := derivationref.QualifyCache(context.Background(), w, policy, v); !errors.Is(err, derivationref.ErrNotCoherent) {
		t.Fatalf("provider denial ignored: %v", err)
	}
	v.denyAligned = false
	if _, err := derivationref.QualifyCache(context.Background(), w, policy, v); err != nil || v.alignedCalls != 2 {
		t.Fatalf("external alignment approval not applied: %v calls=%d", err, v.alignedCalls)
	}
	// Non-atomic snapshots may be used only if the consumer has explicitly
	// not requested coherence; the result does not claim aligned consistency.
	for i := range w.Sources {
		w.Sources[i].SnapshotRef = ""
	}
	if _, err := derivationref.QualifyCache(context.Background(), w, derivationref.CachePolicy{MaxAge: time.Minute}, v); err != nil {
		t.Fatalf("unknown-consistency opt-in should remain expressible: %v", err)
	}
}

type d3FactReader struct {
	fact derivationref.ConfirmedFact
	calls int
}
func (r *d3FactReader) ReadConfirmed(_ context.Context, _ string) (derivationref.ConfirmedFact, error) {
	r.calls++
	return r.fact, nil
}
type d3FactAuthority struct { deny bool }
func (a d3FactAuthority) CanReadConfirmed(_ context.Context, _ derivationref.ConfirmedFact) error {
	if a.deny {
		return errors.New("denied")
	}
	return nil
}

func TestD3ConfirmedValueIsNotRecomputedOrRewritten(t *testing.T) {
	now := time.Date(2026, 10, 15, 1, 0, 0, 0, time.UTC)
	stored := derivationref.ConfirmedFact{
		FactID: "order-123.amount",
		TargetID: "order.amount",
		Value: derivationref.Input{State: derivationref.Present, Decimal: "600"},
		SourceRuleID: "amount", SourceRuleRevision: "v1",
		ConfirmedAt: now.Add(-24*time.Hour),
		CommandID: "order-123.confirm",
		PersistedRevision: "order-123-v1",
	}
	reader := &d3FactReader{fact: stored}
	fact, err := derivationref.ReadConfirmed(context.Background(), stored.FactID, reader, d3FactAuthority{})
	if err != nil || !reflect.DeepEqual(fact, stored) {
		t.Fatalf("confirmed read: %#v, %v", fact, err)
	}
	latest, err := derivationref.Multiply(
		derivationref.Input{State: derivationref.Present, Decimal: "2"},
		derivationref.Input{State: derivationref.Present, Decimal: "350"},
	)
	if err != nil || latest.Decimal != "700" || fact.Value.Decimal != "600" ||
		reader.fact.Value.Decimal != "600" || reader.calls != 1 {
		t.Fatal("current master silently rewrote confirmed business fact")
	}
	if _, err := derivationref.ReadConfirmed(context.Background(), stored.FactID, reader, d3FactAuthority{deny: true}); !errors.Is(err, derivationref.ErrCacheDenied) {
		t.Fatalf("revoked fact disclosure: %v", err)
	}
	if _, err := derivationref.ReadConfirmed(context.Background(), stored.FactID, reader, nil); !errors.Is(err, derivationref.ErrConfirmed) {
		t.Fatalf("missing authority accepted: %v", err)
	}
	reader.fact.CommandID = ""
	if _, err := derivationref.ReadConfirmed(context.Background(), stored.FactID, reader, d3FactAuthority{}); !errors.Is(err, derivationref.ErrConfirmed) {
		t.Fatalf("unproven persisted fact accepted: %v", err)
	}
}
