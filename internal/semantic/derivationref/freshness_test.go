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

func TestD3ConsumerSourceAgeIsIndependentOfMemoTTL(t *testing.T) {
	now := time.Date(2026, 10, 15, 1, 0, 0, 0, time.UTC)
	w := d3Witness()
	w.Sources[0].ObservedAt = now.Add(-10 * time.Second)
	w.Sources[0].SourceAsOf = now.Add(-30 * time.Second)
	policy := derivationref.CachePolicy{MaxAge: time.Hour, MaxSourceAge: time.Minute}
	if err := derivationref.CheckSourceFreshAt(w, policy, now); err != nil {
		t.Fatalf("30-second-old source should satisfy one-minute bound: %v", err)
	}
	w.Sources[0].SourceAsOf = now.Add(-time.Minute)
	if err := derivationref.CheckSourceFreshAt(w, policy, now); err != nil {
		t.Fatalf("inclusive source-age boundary rejected: %v", err)
	}
	w.Sources[0].SourceAsOf = now.Add(-2 * time.Minute)
	if err := derivationref.CheckSourceFreshAt(w, policy, now); !errors.Is(err, derivationref.ErrCacheStale) {
		t.Fatalf("two-minute-old source was incorrectly fresh: %v", err)
	}
	// The default has no source age bound; this is a conscious consumer
	// choice, not proof that a stale source is new.
	if err := derivationref.CheckSourceFreshAt(w, derivationref.CachePolicy{MaxAge: time.Hour}, now); err != nil {
		t.Fatalf("optional bound became mandatory: %v", err)
	}
	// With multiple sources, the freshness of one cannot cover the other.
	other := w.Sources[0]
	other.SourceID = "pricing"
	other.SourceAsOf = now.Add(-10 * time.Second)
	w.Sources = append(w.Sources, other)
	if err := derivationref.CheckSourceFreshAt(w, policy, now); !errors.Is(err, derivationref.ErrCacheStale) {
		t.Fatalf("one stale dependency was masked by newer provider: %v", err)
	}
	w.Sources[0].SourceAsOf = now.Add(-30 * time.Second)
	if err := derivationref.CheckSourceFreshAt(w, policy, now); err != nil {
		t.Fatalf("both providers fresh should be permitted: %v", err)
	}
	// Attestation of an aligned cross-provider snapshot is a separate
	// requirement from the ages checked here.
	if _, err := derivationref.QualifyCache(context.Background(), w,
		derivationref.CachePolicy{MaxAge: time.Hour, RequireAligned: true},
		&d3WitnessVerifier{}); !errors.Is(err, derivationref.ErrNotCoherent) {
		t.Fatalf("source freshness incorrectly established atomicity: %v", err)
	}
}

func TestD3ConsumerSourceAgeRejectsFutureOrInvalidEvidence(t *testing.T) {
	now := time.Date(2026, 10, 15, 1, 0, 0, 0, time.UTC)
	for _, tc := range []struct {
		name string
		edit func(*derivationref.CacheWitness)
	}{
		{"future observation", func(w *derivationref.CacheWitness) { w.Sources[0].ObservedAt = now.Add(time.Second); w.Sources[0].SourceAsOf = now.Add(-time.Second) }},
		{"future source as-of", func(w *derivationref.CacheWitness) { w.Sources[0].ObservedAt = now.Add(time.Second); w.Sources[0].SourceAsOf = now.Add(time.Second) }},
		{"missing source as-of", func(w *derivationref.CacheWitness) { w.Sources[0].SourceAsOf = time.Time{} }},
		{"source as-of after observation", func(w *derivationref.CacheWitness) { w.Sources[0].ObservedAt = now.Add(-time.Minute); w.Sources[0].SourceAsOf = now }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := d3Witness()
			tc.edit(&w)
			if err := derivationref.CheckSourceFreshAt(w, derivationref.CachePolicy{}, now); !errors.Is(err, derivationref.ErrCacheEvidence) {
				t.Fatalf("chronology was accepted: %v", err)
			}
		})
	}
	if err := derivationref.CheckSourceFreshAt(d3Witness(),
		derivationref.CachePolicy{MaxSourceAge: -time.Second}, now); !errors.Is(err, derivationref.ErrCacheEvidence) {
		t.Fatalf("negative source age accepted: %v", err)
	}
	if err := derivationref.CheckSourceFreshAt(d3Witness(), derivationref.CachePolicy{}, time.Time{}); !errors.Is(err, derivationref.ErrCacheEvidence) {
		t.Fatalf("untrusted zero evaluation time accepted: %v", err)
	}
}

type d3CurrentSourceReader struct {
	observations map[string]derivationref.SourceObservation
	fail         bool
	reads        int
}

func (r *d3CurrentSourceReader) ReadCurrentSource(_ context.Context, id string) (derivationref.SourceObservation, error) {
	r.reads++
	if r.fail {
		return derivationref.SourceObservation{}, errors.New("provider metadata read failed")
	}
	observed, ok := r.observations[id]
	if !ok {
		return derivationref.SourceObservation{}, errors.New("unknown source")
	}
	return observed, nil
}

func TestD3SourceBoundVerifierRejectsForgedOrChangedSourceProvenance(t *testing.T) {
	ctx := context.Background()
	w := d3Witness()
	host := &d3WitnessVerifier{}
	source := &d3CurrentSourceReader{observations: map[string]derivationref.SourceObservation{
		"inventory": w.Sources[0],
	}}
	v := derivationref.SourceBoundVerifier{Current: host, Sources: source}
	policy := derivationref.CachePolicy{MaxAge: time.Minute, MaxSourceAge: 2 * time.Minute}
	key, err := derivationref.QualifyCache(ctx, w, policy, v)
	if err != nil || key == "" || host.currentCalls != 1 || source.reads != 1 {
		t.Fatalf("trusted observation did not qualify: key=%q err=%v host=%d provider=%d", key, err, host.currentCalls, source.reads)
	}
	for _, tc := range []struct {
		name string
		edit func(*derivationref.CacheWitness)
	}{
		{"forged newer as-of with unchanged source revision", func(w *derivationref.CacheWitness) {
			w.Sources[0].SourceAsOf = w.Sources[0].SourceAsOf.Add(30 * time.Second)
		}},
		{"forged observed-at with unchanged source revision", func(w *derivationref.CacheWitness) {
			w.Sources[0].ObservedAt = w.Sources[0].ObservedAt.Add(30 * time.Second)
		}},
		{"invented source revision", func(w *derivationref.CacheWitness) {
			w.Sources[0].Revision = "source-unobserved"
		}},
		{"invented snapshot reference", func(w *derivationref.CacheWitness) {
			w.Sources[0].SnapshotRef = "snapshot-forged"
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			forged := d3Witness()
			tc.edit(&forged)
			if _, err := derivationref.QualifyCache(ctx, forged, policy, v); !errors.Is(err, derivationref.ErrCacheDenied) {
				t.Fatalf("self-reported provenance accepted: %v", err)
			}
		})
	}
	updated := w.Sources[0]
	updated.Revision = "source-2"
	source.observations["inventory"] = updated
	if _, err := derivationref.QualifyCache(ctx, w, policy, v); !errors.Is(err, derivationref.ErrCacheDenied) {
		t.Fatalf("source revision advance reused cache evidence: %v", err)
	}
	source.fail = true
	if _, err := derivationref.QualifyCache(ctx, w, policy, v); !errors.Is(err, derivationref.ErrCacheDenied) {
		t.Fatalf("unavailable provider metadata treated as valid: %v", err)
	}
	source.fail = false
	// The public qualification boundary intentionally reports only
	// ErrCacheDenied for verifier failures, to avoid leaking Provider details.
	// The internal adapter retains the precise stale/evidence reason.
	if err := v.VerifyCurrent(ctx, w); !errors.Is(err, derivationref.ErrCacheStale) {
		t.Fatalf("internal verifier lost source mismatch diagnosis: %v", err)
	}
	if _, err := derivationref.QualifyCache(ctx, w, policy,
		derivationref.SourceBoundVerifier{Current: host}); !errors.Is(err, derivationref.ErrCacheDenied) {
		t.Fatalf("no source provider accepted: %v", err)
	}
	if _, err := derivationref.QualifyCache(ctx, w, policy,
		derivationref.SourceBoundVerifier{Sources: source}); !errors.Is(err, derivationref.ErrCacheDenied) {
		t.Fatalf("no host verification accepted: %v", err)
	}
}

func TestD3SourceBoundVerifierEveryProviderAndAlignmentAreSeparate(t *testing.T) {
	w := d3Witness()
	second := w.Sources[0]
	second.SourceID = "pricing"
	second.Revision = "price-v1"
	w.Sources = append(w.Sources, second)
	source := &d3CurrentSourceReader{observations: map[string]derivationref.SourceObservation{
		"inventory": w.Sources[0], "pricing": second,
	}}
	host := &d3WitnessVerifier{}
	v := derivationref.SourceBoundVerifier{Current: host, Sources: source}
	policy := derivationref.CachePolicy{MaxAge: time.Minute}
	if _, err := derivationref.QualifyCache(context.Background(), w, policy, v); err != nil || source.reads != 2 {
		t.Fatalf("all source metadata must be checked: reads=%d err=%v", source.reads, err)
	}
	second.Revision = "price-v2"
	source.observations["pricing"] = second
	if _, err := derivationref.QualifyCache(context.Background(), w, policy, v); !errors.Is(err, derivationref.ErrCacheDenied) {
		t.Fatalf("one stale dependency was incorrectly accepted: %v", err)
	}
	// Even matching snapshots do not prove cross-provider atomicity; the
	// separate existing alignment verifier must attest it explicitly.
	source.observations["pricing"] = w.Sources[1]
	for i := range w.Sources {
		w.Sources[i].SnapshotRef = "batch-1"
		actual := source.observations[w.Sources[i].SourceID]
		actual.SnapshotRef = "batch-1"
		source.observations[w.Sources[i].SourceID] = actual
	}
	host.denyAligned = true
	policy.RequireAligned = true
	if _, err := derivationref.QualifyCache(context.Background(), w, policy, v); !errors.Is(err, derivationref.ErrNotCoherent) {
		t.Fatalf("aligned snapshot falsely inferred from matching ref: %v", err)
	}
	host.denyAligned = false
	if _, err := derivationref.QualifyCache(context.Background(), w, policy, v); err != nil {
		t.Fatalf("provider metadata and independent snapshot approval: %v", err)
	}
}
