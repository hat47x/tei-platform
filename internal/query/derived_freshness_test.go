package query_test

import (
	"context"
	"errors"
	"reflect"
	"strconv"
	"testing"
	"time"

	"tei/internal/core"
	"tei/internal/query"
	"tei/internal/semantic/derivationref"
)

type d3QueryVerifier struct {
	currentCalls int
	alignedCalls int
	deny         bool
}

func (v *d3QueryVerifier) VerifyCurrent(_ context.Context, _ derivationref.CacheWitness) error {
	v.currentCalls++
	if v.deny {
		return errors.New("current epoch revoked")
	}
	return nil
}
func (v *d3QueryVerifier) VerifyAligned(_ context.Context, _ derivationref.CacheWitness) error {
	v.alignedCalls++
	return nil
}

func d3QueryWitness() derivationref.CacheWitness {
	observed := time.Date(2026, 10, 15, 1, 0, 0, 0, time.UTC)
	return derivationref.CacheWitness{
		RuleID: "available-rule",
		RuleRevision: "v1",
		TargetID: "product.available",
		SemanticRevision: "semantic-1",
		AcceptanceRevision: "accepted-1",
		ProviderRevision: "provider-1",
		Inputs: []derivationref.Revision{
			{ID: "product.reserved", Revision: "revision-1"},
			{ID: "product.stock", Revision: "revision-1"},
		},
		Sources: []derivationref.SourceObservation{{
			SourceID: "inventory",
			Revision: "source-1",
			SourceAsOf: observed.Add(-time.Minute),
			ObservedAt: observed,
		}},
		Scope: derivationref.CacheScope{
			Environment: "test",
			Tenant: "tenant-1",
			Subject: "user-1",
			PermissionEpoch: "policy-1",
		},
	}
}

func TestD3MemoReusesD2OnlyUnderSameCurrentWitness(t *testing.T) {
	q, plan, sqlEvidence, rows := d2Fixture()
	witness := d3QueryWitness()
	verifier := &d3QueryVerifier{}
	allowed := func(core.CanonicalID) bool { return true }
	policy := derivationref.CachePolicy{MaxAge: time.Minute}
	now := time.Date(2026, 10, 15, 1, 0, 10, 0, time.UTC)
	var memo query.DerivedMemo
	loaded := 0
	load := func(context.Context, int) ([]query.DerivedRow, bool, error) {
		loaded++
		return rows, true, nil
	}
	first, hit, err := memo.Resolve(context.Background(), q, plan, sqlEvidence, witness, policy, now, verifier, allowed, load)
	if err != nil || hit || loaded != 1 ||
		!reflect.DeepEqual(d2IDs(first.Rows), []string{"B", "C", "D", "A", "E"}) {
		t.Fatalf("initial D2 evaluate: result=%#v hit=%v calls=%d err=%v", first, hit, loaded, err)
	}
	// A caller cannot mutate the stored cache entry through a result map.
	first.Rows[0].Values["product.id"] = derivationref.Input{State: derivationref.Present, Decimal: "FORGED"}
	again, hit, err := memo.Resolve(context.Background(), q, plan, sqlEvidence, witness, policy, now.Add(time.Second), verifier, allowed, nil)
	if err != nil || !hit || loaded != 1 || again.Rows[0].Values["product.id"].Decimal != "B" {
		t.Fatalf("reuse returned mutable or incorrect result: %#v hit=%v err=%v", again, hit, err)
	}
	// The corresponding source/field revision must change when reserved
	// changes. Without evidence of change, no backend can infer freshness.
	rows[0].Values["product.reserved"] = derivationref.Input{State: derivationref.Present, Decimal: "10"}
	changed := d3QueryWitness()
	changed.Inputs[0].Revision = "revision-2"
	changed.Sources[0].Revision = "source-2"
	updated, hit, err := memo.Resolve(context.Background(), q, plan, sqlEvidence, changed, policy, now.Add(2*time.Second), verifier, allowed, load)
	if err != nil || hit || loaded != 2 ||
		!reflect.DeepEqual(d2IDs(updated.Rows), []string{"A", "B", "C", "D", "E"}) ||
		updated.Rows[0].Values["product.available"].Decimal != "110" {
		t.Fatalf("stale 20 reused after allocation update: %#v hit=%v err=%v", updated, hit, err)
	}
	changed.RuleRevision = "v2"
	if _, hit, err := memo.Resolve(context.Background(), q, plan, sqlEvidence, changed, policy, now.Add(3*time.Second), verifier, allowed, load); err != nil || hit || loaded != 3 {
		t.Fatalf("rule revision reused cached value: hit=%v calls=%d err=%v", hit, loaded, err)
	}
	changed.Scope.Tenant = "tenant-2"
	if _, hit, err := memo.Resolve(context.Background(), q, plan, sqlEvidence, changed, policy, now.Add(4*time.Second), verifier, allowed, load); err != nil || hit || loaded != 4 {
		t.Fatalf("tenant-scoped result leaked: hit=%v calls=%d err=%v", hit, loaded, err)
	}
	changed.Scope.PermissionEpoch = "policy-2"
	if _, hit, err := memo.Resolve(context.Background(), q, plan, sqlEvidence, changed, policy, now.Add(5*time.Second), verifier, allowed, load); err != nil || hit || loaded != 5 {
		t.Fatalf("authorization epoch reused: hit=%v calls=%d err=%v", hit, loaded, err)
	}
	if verifier.currentCalls < 7 {
		t.Fatalf("current verifier was not checked on hits and misses: %d", verifier.currentCalls)
	}
}

func TestD3MemoAuthorityExpiryQueryShapeAndCompleteness(t *testing.T) {
	q, plan, sqlEvidence, rows := d2Fixture()
	w := d3QueryWitness()
	policy := derivationref.CachePolicy{MaxAge: time.Minute}
	now := time.Date(2026, 10, 15, 1, 0, 10, 0, time.UTC)
	v := &d3QueryVerifier{}
	var memo query.DerivedMemo
	loadCount := 0
	load := func(context.Context, int) ([]query.DerivedRow, bool, error) {
		loadCount++
		return rows, true, nil
	}
	allow := func(core.CanonicalID) bool { return true }
	if _, _, err := memo.Resolve(context.Background(), q, plan, sqlEvidence, w, policy, now, v, allow, load); err != nil {
		t.Fatal(err)
	}
	denyRaw := func(id core.CanonicalID) bool { return id != "product.stock" }
	if _, _, err := memo.Resolve(context.Background(), q, plan, sqlEvidence, w, policy, now, v, denyRaw, nil); !errors.Is(err, query.ErrDerivedUnauthorized) {
		t.Fatalf("protected input cached: %v", err)
	}
	v.deny = true
	if _, _, err := memo.Resolve(context.Background(), q, plan, sqlEvidence, w, policy, now, v, allow, nil); !errors.Is(err, derivationref.ErrCacheDenied) {
		t.Fatalf("revoked Host witness cached: %v", err)
	}
	v.deny = false
	if _, hit, err := memo.Resolve(context.Background(), q, plan, sqlEvidence, w, policy, now.Add(2*time.Minute), v, allow, load); err != nil || hit || loadCount != 2 {
		t.Fatalf("expired TTL reused: hit=%v calls=%d err=%v", hit, loadCount, err)
	}
	// Different D2 window or filter must not reuse an earlier result.
	plan.Where = &query.DerivedWhere{Op: ">=", Decimal: "50"}
	if got, hit, err := memo.Resolve(context.Background(), q, plan, sqlEvidence, w, policy, now.Add(2*time.Minute), v, allow, load); err != nil || hit || got.Count != 3 {
		t.Fatalf("predicate reused cache: %#v hit=%v err=%v", got, hit, err)
	}
	// A false "complete" flag is a failure, never an empty cache entry.
	var invalid query.DerivedMemo
	incomplete := func(context.Context, int) ([]query.DerivedRow, bool, error) {
		return rows[:1], false, nil
	}
	if _, _, err := invalid.Resolve(context.Background(), q, plan, sqlEvidence, w, policy, now, v, allow, incomplete); !errors.Is(err, query.ErrDerivedIncomplete) {
		t.Fatalf("partial provider source cached: %v", err)
	}
	if _, _, err := invalid.Resolve(context.Background(), q, plan, sqlEvidence, w, policy, now, v, allow, nil); !errors.Is(err, query.ErrDerivedIncomplete) {
		t.Fatalf("incomplete result left cached entry: %v", err)
	}
	// SQL source changes must be rejected before cache read or execution.
	q.Source += " LIMIT 1"
	if _, _, err := memo.Resolve(context.Background(), q, plan, sqlEvidence, w, policy, now, v, allow, load); !errors.Is(err, query.ErrDerivedUnsupported) {
		t.Fatalf("changed native SQL reused prior result: %v", err)
	}
}

func TestD3MemoNoCacheOptInAndMultiSourceCoherence(t *testing.T) {
	q, plan, sqlEvidence, rows := d2Fixture()
	w := d3QueryWitness()
	policy := derivationref.CachePolicy{} // explicit cache-disabled mode
	if policy.MaxAge != 0 {
		t.Fatal("pure Go evaluation should not require caching")
	}
	now := time.Date(2026, 10, 15, 1, 0, 10, 0, time.UTC)
	v := &d3QueryVerifier{}
	var memo query.DerivedMemo
	calls := 0
	load := func(context.Context, int) ([]query.DerivedRow, bool, error) {
		calls++
		return rows, true, nil
	}
	for i := 0; i < 2; i++ {
		_, hit, err := memo.Resolve(context.Background(), q, plan, sqlEvidence, w, policy, now, v, func(core.CanonicalID) bool { return true }, load)
		if err != nil || hit {
			t.Fatalf("cache disabled but hit=%v err=%v", hit, err)
		}
	}
	if calls != 2 {
		t.Fatalf("cache disabled but provider skipped: %d", calls)
	}
	second := w.Sources[0]
	second.SourceID = "pricing"
	w.Sources = append(w.Sources, second)
	policy = derivationref.CachePolicy{MaxAge: time.Minute, RequireAligned: true}
	if _, _, err := memo.Resolve(context.Background(), q, plan, sqlEvidence, w, policy, now, v, func(core.CanonicalID) bool { return true }, load); !errors.Is(err, derivationref.ErrNotCoherent) {
		t.Fatalf("unknown cross-source snapshot claimed coherent: %v", err)
	}
	if calls != 2 {
		t.Fatal("unverified cross-source query reached provider")
	}
}

func TestD3MemoHasBoundedMemoryAndEvictsOldest(t *testing.T) {
	q, plan, proof, rows := d2Fixture()
	var memo query.DerivedMemo
	v := &d3QueryVerifier{}
	allow := func(core.CanonicalID) bool { return true }
	at := time.Date(2026, 10, 15, 1, 0, 10, 0, time.UTC)
	policy := derivationref.CachePolicy{MaxAge: time.Hour}
	loads := 0
	load := func(context.Context, int) ([]query.DerivedRow, bool, error) {
		loads++
		return rows, true, nil
	}
	for i := 0; i < 65; i++ {
		w := d3QueryWitness()
		w.Inputs[0].Revision = strconv.Itoa(i)
		w.Sources[0].Revision = strconv.Itoa(i)
		if _, hit, err := memo.Resolve(context.Background(), q, plan, proof, w, policy, at.Add(time.Duration(i)*time.Second), v, allow, load); err != nil || hit {
			t.Fatalf("memo entry %d unexpectedly hit or failed: %v %v", i, hit, err)
		}
	}
	if loads != 65 {
		t.Fatalf("expected distinct bounded entries: %d", loads)
	}
	first := d3QueryWitness()
	first.Inputs[0].Revision = "0"
	first.Sources[0].Revision = "0"
	if _, hit, err := memo.Resolve(context.Background(), q, plan, proof, first, policy, at.Add(65*time.Second), v, allow, load); err != nil || hit || loads != 66 {
		t.Fatalf("oldest entry survived bounded cache: hit=%v loads=%d err=%v", hit, loads, err)
	}
}
