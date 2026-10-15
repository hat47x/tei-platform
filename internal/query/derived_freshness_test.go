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
	denyAligned  bool
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
	if v.denyAligned {
		return errors.New("snapshot alignment not attested")
	}
	return nil
}

func d3QueryWitness() derivationref.CacheWitness {
	observed := time.Date(2026, 10, 15, 1, 0, 0, 0, time.UTC)
	return derivationref.CacheWitness{
		RuleID:             "available-rule",
		RuleRevision:       "v1",
		TargetID:           "product.available",
		SemanticRevision:   "semantic-1",
		AcceptanceRevision: "accepted-1",
		ProviderRevision:   "provider-1",
		Inputs: []derivationref.Revision{
			{ID: "product.reserved", Revision: "revision-1"},
			{ID: "product.stock", Revision: "revision-1"},
		},
		Sources: []derivationref.SourceObservation{{
			SourceID:   "inventory",
			Revision:   "source-1",
			SourceAsOf: observed.Add(-time.Minute),
			ObservedAt: observed,
		}},
		Scope: derivationref.CacheScope{
			Environment:     "test",
			Tenant:          "tenant-1",
			Subject:         "user-1",
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

func TestD3MemoRejectsStaleSourceEvenInsideComputedTTL(t *testing.T) {
	q, plan, proof, rows := d2Fixture()
	w := d3QueryWitness()
	policy := derivationref.CachePolicy{
		MaxAge: 10 * time.Minute, MaxSourceAge: 2 * time.Minute,
	}
	now := time.Date(2026, 10, 15, 1, 0, 10, 0, time.UTC)
	v := &d3QueryVerifier{}
	allow := func(core.CanonicalID) bool { return true }
	var memo query.DerivedMemo
	loads := 0
	load := func(context.Context, int) ([]query.DerivedRow, bool, error) {
		loads++
		return rows, true, nil
	}
	// SourceAsOf=00:59, but the calculation is freshly evaluated at 01:00:10.
	// Both source and memo are within their independent age limits.
	if _, hit, err := memo.Resolve(context.Background(), q, plan, proof, w, policy,
		now, v, allow, load); err != nil || hit || loads != 1 {
		t.Fatalf("fresh source miss failed: hit=%v loads=%d err=%v", hit, loads, err)
	}
	if _, hit, err := memo.Resolve(context.Background(), q, plan, proof, w, policy,
		now.Add(20*time.Second), v, allow, nil); err != nil || !hit || loads != 1 {
		t.Fatalf("fresh source hit failed: hit=%v loads=%d err=%v", hit, loads, err)
	}
	// At 01:01:01 source is 2m01s old, though the computed result is only
	// 51s old (within its ten-minute TTL). Neither a hit nor a re-evaluation
	// over the exact same stale SourceAsOf can satisfy the consumer.
	if _, hit, err := memo.Resolve(context.Background(), q, plan, proof, w, policy,
		now.Add(51*time.Second), v, allow, load); !errors.Is(err, derivationref.ErrCacheStale) ||
		hit || loads != 1 {
		t.Fatalf("stale provider data passed through TTL: hit=%v loads=%d err=%v", hit, loads, err)
	}
	// A new source observation, backed by a current verifier and a new
	// revision, must miss the old key and compute over the revised source.
	fresh := d3QueryWitness()
	fresh.Sources[0].Revision = "source-2"
	fresh.Sources[0].SourceAsOf = now.Add(20 * time.Second)
	fresh.Sources[0].ObservedAt = now.Add(30 * time.Second)
	fresh.Inputs[0].Revision = "revision-2"
	if _, hit, err := memo.Resolve(context.Background(), q, plan, proof, fresh, policy,
		now.Add(51*time.Second), v, allow, load); err != nil || hit || loads != 2 {
		t.Fatalf("newer observed source failed to recompute: hit=%v loads=%d err=%v", hit, loads, err)
	}
	// Without the optional source-age policy, the old source is not
	// automatically declared current, but the existing contract permits it
	// with its still-attested revisions.
	if _, hit, err := memo.Resolve(context.Background(), q, plan, proof, w,
		derivationref.CachePolicy{MaxAge: 10 * time.Minute},
		now.Add(51*time.Second), v, allow, nil); err != nil || !hit || loads != 2 {
		t.Fatalf("source-age opt-in incorrectly changed default: hit=%v loads=%d err=%v", hit, loads, err)
	}
}

type d3MemoSourceReader struct {
	sources map[string]derivationref.SourceObservation
	reads   int
	err     error
}

func (r *d3MemoSourceReader) ReadCurrentSource(_ context.Context, id string) (derivationref.SourceObservation, error) {
	r.reads++
	if r.err != nil {
		return derivationref.SourceObservation{}, r.err
	}
	if obs, ok := r.sources[id]; ok {
		return obs, nil
	}
	return derivationref.SourceObservation{}, errors.New("unknown Provider source")
}

func TestD3MemoProviderProvenancePreventsForgedCacheHitAndRecomputes(t *testing.T) {
	q, plan, proof, rows := d2Fixture()
	w := d3QueryWitness()
	now := time.Date(2026, 10, 15, 1, 0, 10, 0, time.UTC)
	provider := &d3MemoSourceReader{sources: map[string]derivationref.SourceObservation{
		"inventory": w.Sources[0],
	}}
	v := derivationref.SourceBoundVerifier{Current: &d3QueryVerifier{}, Sources: provider}
	allow := func(core.CanonicalID) bool { return true }
	policy := derivationref.CachePolicy{MaxAge: time.Minute, MaxSourceAge: 2 * time.Minute}
	var memo query.DerivedMemo
	loads := 0
	load := func(context.Context, int) ([]query.DerivedRow, bool, error) {
		loads++
		return rows, true, nil
	}
	if _, hit, err := memo.Resolve(context.Background(), q, plan, proof, w,
		policy, now, v, allow, load); err != nil || hit || loads != 1 || provider.reads != 2 {
		t.Fatalf("verified D2 cache fill failed: hit=%v loads=%d reads=%d err=%v", hit, loads, provider.reads, err)
	}
	// A forged as-of that looks newer with the same revision must not enter
	// a fresh memo key or return a value based on user-supplied evidence.
	forged := d3QueryWitness()
	forged.Sources[0].SourceAsOf = forged.Sources[0].SourceAsOf.Add(30 * time.Second)
	if _, hit, err := memo.Resolve(context.Background(), q, plan, proof, forged,
		policy, now, v, allow, nil); !errors.Is(err, derivationref.ErrCacheDenied) || hit {
		t.Fatalf("fabricated source timestamp bypassed verifier: hit=%v err=%v", hit, err)
	}
	updated := d3QueryWitness()
	updated.Sources[0].Revision = "source-2"
	updated.Sources[0].SourceAsOf = now.Add(-5 * time.Second)
	updated.Sources[0].ObservedAt = now.Add(-2 * time.Second)
	updated.Inputs[0].Revision = "revision-2"
	provider.sources["inventory"] = updated.Sources[0]
	if _, hit, err := memo.Resolve(context.Background(), q, plan, proof, w,
		policy, now, v, allow, nil); !errors.Is(err, derivationref.ErrCacheDenied) || hit {
		t.Fatalf("changed provider revision bypassed current verification: hit=%v err=%v", hit, err)
	}
	rows[0].Values["product.reserved"] = derivationref.Input{State: derivationref.Present, Decimal: "10"}
	result, hit, err := memo.Resolve(context.Background(), q, plan, proof, updated,
		policy, now, v, allow, load)
	if err != nil || hit || loads != 2 || result.Rows[0].Values["product.available"].Decimal != "110" {
		t.Fatalf("trusted revised source did not recompute D2 result: hit=%v loads=%d result=%#v err=%v", hit, loads, result, err)
	}
}

func TestD3MemoProviderChangeDuringLoadDoesNotStoreStaleResult(t *testing.T) {
	q, plan, proof, rows := d2Fixture()
	w := d3QueryWitness()
	now := time.Date(2026, 10, 15, 1, 0, 10, 0, time.UTC)
	provider := &d3MemoSourceReader{sources: map[string]derivationref.SourceObservation{
		"inventory": w.Sources[0],
	}}
	v := derivationref.SourceBoundVerifier{Current: &d3QueryVerifier{}, Sources: provider}
	allow := func(core.CanonicalID) bool { return true }
	var memo query.DerivedMemo
	policy := derivationref.CachePolicy{MaxAge: time.Minute}
	// The source changes between pre-read qualification and D2's post-load
	// qualification. No stale computed result may become a memo entry.
	load := func(context.Context, int) ([]query.DerivedRow, bool, error) {
		changed := w.Sources[0]
		changed.Revision = "source-2"
		provider.sources["inventory"] = changed
		return rows, true, nil
	}
	if _, hit, err := memo.Resolve(context.Background(), q, plan, proof, w,
		policy, now, v, allow, load); !errors.Is(err, derivationref.ErrCacheDenied) || hit {
		t.Fatalf("source changed during read but value returned: hit=%v err=%v", hit, err)
	}
	provider.sources["inventory"] = w.Sources[0]
	if _, hit, err := memo.Resolve(context.Background(), q, plan, proof, w,
		policy, now, v, allow, nil); !errors.Is(err, query.ErrDerivedIncomplete) || hit {
		t.Fatalf("result from invalidated load was stored: hit=%v err=%v", hit, err)
	}
}

func TestD3MemoDisabledDoesNotRequireCacheWitness(t *testing.T) {
	q, plan, proof, rows := d2Fixture()
	// A calculation with no cache, source-age or alignment requirement must
	// not require any trusted clock or temporal witness.
	now := time.Time{}
	// No cache/freshness/snapshot request: neither a source timestamp nor
	// a cache witness or cache verifier should be necessary to evaluate D2.
	policy := derivationref.CachePolicy{}
	allow := func(core.CanonicalID) bool { return true }
	var memo query.DerivedMemo
	calls := 0
	load := func(context.Context, int) ([]query.DerivedRow, bool, error) {
		calls++
		return rows, true, nil
	}
	for i := 0; i < 2; i++ {
		result, hit, err := memo.Resolve(context.Background(), q, plan, proof,
			derivationref.CacheWitness{}, policy, now, nil, allow, load)
		if err != nil || hit || result.Count != 5 ||
			!reflect.DeepEqual(d2IDs(result.Rows), []string{"B", "C", "D", "A", "E"}) {
			t.Fatalf("uncached D2 evaluation %d: hit=%v result=%#v err=%v", i, hit, result, err)
		}
	}
	if calls != 2 {
		t.Fatalf("uncached evaluation incorrectly reused a result: calls=%d", calls)
	}
	// Neither cache storage nor even the memo object is needed for pure
	// evaluation, while complete retrieval and D2 authorization still apply.
	var nilMemo *query.DerivedMemo
	if _, hit, err := nilMemo.Resolve(context.Background(), q, plan, proof,
		derivationref.CacheWitness{}, policy, now, nil, allow, load); err != nil || hit || calls != 3 {
		t.Fatalf("nil memo should be optional for pure evaluation: hit=%v calls=%d err=%v", hit, calls, err)
	}
}

func TestD3MemoDisabledRetainsD2AuthorizationAndCompleteness(t *testing.T) {
	q, plan, proof, rows := d2Fixture()
	now := time.Date(2026, 10, 15, 1, 0, 10, 0, time.UTC)
	var memo query.DerivedMemo
	calls := 0
	load := func(context.Context, int) ([]query.DerivedRow, bool, error) {
		calls++
		return rows, true, nil
	}
	// The absence of a cache witness is never a bypass of D2's raw column
	// read authorization. Refuse before asking the Provider for data.
	denyStock := func(id core.CanonicalID) bool { return id != "product.stock" }
	if _, hit, err := memo.Resolve(context.Background(), q, plan, proof,
		derivationref.CacheWitness{}, derivationref.CachePolicy{}, now, nil,
		denyStock, load); !errors.Is(err, query.ErrDerivedUnauthorized) || hit || calls != 0 {
		t.Fatalf("unprivileged raw read dispatched: hit=%v calls=%d err=%v", hit, calls, err)
	}

	// The derived output target is independently protected; allowing every
	// raw source field does not authorize disclosure of the computed value.
	denyTarget := func(id core.CanonicalID) bool { return id != "product.available" }
	if _, hit, err := memo.Resolve(context.Background(), q, plan, proof,
		derivationref.CacheWitness{}, derivationref.CachePolicy{}, time.Time{}, nil,
		denyTarget, load); !errors.Is(err, query.ErrDerivedUnauthorized) || hit || calls != 0 {
		t.Fatalf("unprivileged derived target dispatched: hit=%v calls=%d err=%v", hit, calls, err)
	}

	// Recheck FieldRead after loading: an authorization revocation observed
	// during Provider access must prevent result disclosure even without memo.
	allowed := true
	canRead := func(core.CanonicalID) bool { return allowed }
	revokeDuringLoad := func(context.Context, int) ([]query.DerivedRow, bool, error) {
		calls++
		allowed = false
		return rows, true, nil
	}
	if _, hit, err := memo.Resolve(context.Background(), q, plan, proof,
		derivationref.CacheWitness{}, derivationref.CachePolicy{}, now, nil,
		canRead, revokeDuringLoad); !errors.Is(err, query.ErrDerivedUnauthorized) || hit || calls != 1 {
		t.Fatalf("read revocation at disclosure was bypassed: hit=%v calls=%d err=%v", hit, calls, err)
	}

	// The full unpaginated source requirement remains intact in cache-off
	// mode; returning a physical page is not equivalent to D2 completeness.
	incomplete := func(context.Context, int) ([]query.DerivedRow, bool, error) {
		calls++
		return rows[:1], false, nil
	}
	allow := func(core.CanonicalID) bool { return true }
	if _, hit, err := memo.Resolve(context.Background(), q, plan, proof,
		derivationref.CacheWitness{}, derivationref.CachePolicy{}, now, nil,
		allow, incomplete); !errors.Is(err, query.ErrDerivedIncomplete) || hit || calls != 2 {
		t.Fatalf("partial physical row source was accepted: hit=%v calls=%d err=%v", hit, calls, err)
	}
	q.Source += " LIMIT 1"
	if _, hit, err := memo.Resolve(context.Background(), q, plan, proof,
		derivationref.CacheWitness{}, derivationref.CachePolicy{}, now, nil,
		allow, load); !errors.Is(err, query.ErrDerivedUnsupported) || hit || calls != 2 {
		t.Fatalf("changed native SQL bypassed D2 validation: hit=%v calls=%d err=%v", hit, calls, err)
	}
}

func TestD3MemoExplicitFreshnessAndCacheStillRequireWitness(t *testing.T) {
	q, plan, proof, rows := d2Fixture()
	now := time.Date(2026, 10, 15, 1, 0, 10, 0, time.UTC)
	var memo query.DerivedMemo
	allow := func(core.CanonicalID) bool { return true }
	calls := 0
	load := func(context.Context, int) ([]query.DerivedRow, bool, error) {
		calls++
		return rows, true, nil
	}
	cases := []struct {
		name   string
		policy derivationref.CachePolicy
	}{
		{"enabled memo", derivationref.CachePolicy{MaxAge: time.Minute}},
		{"source age despite disabled memo", derivationref.CachePolicy{MaxSourceAge: 2 * time.Minute}},
		{"snapshot alignment despite disabled memo", derivationref.CachePolicy{RequireAligned: true}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Missing cache evidence must be rejected even when memo writes
			// are disabled but a source-age/alignment guarantee was requested.
			if _, hit, err := memo.Resolve(context.Background(), q, plan, proof,
				derivationref.CacheWitness{}, tc.policy, now, nil, allow, load); !errors.Is(err, derivationref.ErrCacheEvidence) || hit {
				t.Fatalf("missing witness accepted: hit=%v err=%v", hit, err)
			}
			if _, hit, err := memo.Resolve(context.Background(), q, plan, proof,
				d3QueryWitness(), tc.policy, now, nil, allow, load); !errors.Is(err, derivationref.ErrCacheDenied) || hit {
				t.Fatalf("missing verifier accepted: hit=%v err=%v", hit, err)
			}
		})
	}
	if calls != 0 {
		t.Fatalf("unverified cache/freshness path dispatched Provider: calls=%d", calls)
	}
}

// A source-age promise or a TTL measurement needs an explicit trusted time,
// even when all other witness fields have otherwise valid values. Rejection
// must happen before any Provider dispatch.
func TestD3MemoTemporalPoliciesRequireEvaluationTime(t *testing.T) {
	q, plan, proof, rows := d2Fixture()
	w := d3QueryWitness()
	v := &d3QueryVerifier{}
	var memo query.DerivedMemo
	allow := func(core.CanonicalID) bool { return true }
	calls := 0
	load := func(context.Context, int) ([]query.DerivedRow, bool, error) {
		calls++
		return rows, true, nil
	}
	tests := []struct {
		name   string
		policy derivationref.CachePolicy
	}{
		{"memo TTL", derivationref.CachePolicy{MaxAge: time.Minute}},
		{"source age without cache", derivationref.CachePolicy{MaxSourceAge: time.Minute}},
		{"memo TTL and source age", derivationref.CachePolicy{MaxAge: time.Minute, MaxSourceAge: time.Minute}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, hit, err := memo.Resolve(context.Background(), q, plan, proof,
				w, tc.policy, time.Time{}, v, allow, load)
			if !errors.Is(err, derivationref.ErrCacheEvidence) || hit {
				t.Fatalf("zero trusted evaluation time accepted: hit=%v err=%v", hit, err)
			}
		})
	}
	if calls != 0 || v.currentCalls != 0 || v.alignedCalls != 0 {
		t.Fatalf("untrusted evaluation time reached Provider/verifier: loads=%d current=%d aligned=%d",
			calls, v.currentCalls, v.alignedCalls)
	}
}

func TestD3MemoNoCacheExplicitSourceAgeAndAlignmentAreVerified(t *testing.T) {
	q, plan, proof, rows := d2Fixture()
	w := d3QueryWitness()
	now := time.Date(2026, 10, 15, 1, 0, 10, 0, time.UTC)
	v := &d3QueryVerifier{}
	allow := func(core.CanonicalID) bool { return true }
	calls := 0
	load := func(context.Context, int) ([]query.DerivedRow, bool, error) {
		calls++
		return rows, true, nil
	}
	var memo query.DerivedMemo
	// Disabling memo storage must not disable a consumer-requested source
	// freshness guarantee or current witness revalidation.
	policy := derivationref.CachePolicy{MaxSourceAge: 2 * time.Minute}
	for i := 0; i < 2; i++ {
		result, hit, err := memo.Resolve(context.Background(), q, plan, proof, w,
			policy, now, v, allow, load)
		if err != nil || hit || result.Count != 5 {
			t.Fatalf("source freshness with memo OFF %d: hit=%v count=%d err=%v", i, hit, result.Count, err)
		}
	}
	if calls != 2 || v.currentCalls != 4 {
		t.Fatalf("memo off skipped current witness pre/post: loads=%d verifies=%d", calls, v.currentCalls)
	}
	// Strong alignment is independent of whether any result is cached.
	second := w.Sources[0]
	second.SourceID = "pricing"
	w.Sources = append(w.Sources, second)
	policy = derivationref.CachePolicy{RequireAligned: true}
	if _, hit, err := memo.Resolve(context.Background(), q, plan, proof, w,
		policy, now, v, allow, load); !errors.Is(err, derivationref.ErrNotCoherent) || hit || calls != 2 {
		t.Fatalf("unattested cross-source snapshot accepted: hit=%v loads=%d err=%v", hit, calls, err)
	}
	for i := range w.Sources {
		w.Sources[i].SnapshotRef = "provider-attested-batch"
	}
	if _, hit, err := memo.Resolve(context.Background(), q, plan, proof, w,
		policy, now, v, allow, load); err != nil || hit || calls != 3 || v.alignedCalls != 2 {
		t.Fatalf("attested no-cache snapshot not evaluated: hit=%v loads=%d aligned=%d err=%v", hit, calls, v.alignedCalls, err)
	}
}

func TestD3MemoAlignmentOnlyZeroClockStillChecksTrustedTwoSourceSnapshot(t *testing.T) {
	q, plan, proof, rows := d2Fixture()
	w := d3QueryWitness()
	second := w.Sources[0]
	second.SourceID = "pricing"
	second.Revision = "price-v1"
	w.Sources[0].SnapshotRef = "provider-attested-batch"
	second.SnapshotRef = "provider-attested-batch"
	w.Sources = append(w.Sources, second)

	// A matching claimed snapshot is not sufficient: both the current Host
	// and a distinct Provider read must attest the two sources.
	provider := &d3MemoSourceReader{sources: map[string]derivationref.SourceObservation{
		"inventory": w.Sources[0],
		"pricing":   w.Sources[1],
	}}
	host := &d3QueryVerifier{}
	verifier := derivationref.SourceBoundVerifier{Current: host, Sources: provider}
	policy := derivationref.CachePolicy{RequireAligned: true}
	allow := func(core.CanonicalID) bool { return true }
	loads := 0
	load := func(context.Context, int) ([]query.DerivedRow, bool, error) {
		loads++
		return rows, true, nil
	}
	var memo query.DerivedMemo

	result, hit, err := memo.Resolve(context.Background(), q, plan, proof, w,
		policy, time.Time{}, verifier, allow, load)
	if err != nil || hit || result.Count != 5 || loads != 1 ||
		provider.reads != 4 || host.currentCalls != 2 || host.alignedCalls != 2 {
		t.Fatalf("zero-clock aligned source rejected or skipped checks: result=%#v hit=%v err=%v loads=%d sourceReads=%d current=%d aligned=%d",
			result, hit, err, loads, provider.reads, host.currentCalls, host.alignedCalls)
	}

	// Matching snapshot refs cannot override an explicit Host rejection.
	host.denyAligned = true
	_, hit, err = memo.Resolve(context.Background(), q, plan, proof, w,
		policy, time.Time{}, verifier, allow, load)
	if !errors.Is(err, derivationref.ErrNotCoherent) || hit || loads != 1 ||
		provider.reads != 6 || host.currentCalls != 3 || host.alignedCalls != 3 {
		t.Fatalf("unattested zero-clock snapshot reached Provider load: hit=%v err=%v loads=%d sourceReads=%d current=%d aligned=%d",
			hit, err, loads, provider.reads, host.currentCalls, host.alignedCalls)
	}

	// Disabling clock checks never makes an invalid policy acceptable.
	_, hit, err = memo.Resolve(context.Background(), q, plan, proof, w,
		derivationref.CachePolicy{MaxSourceAge: -time.Second, RequireAligned: true},
		time.Time{}, verifier, allow, load)
	if !errors.Is(err, derivationref.ErrCacheEvidence) || hit || loads != 1 {
		t.Fatalf("invalid age policy reached Provider load: hit=%v err=%v loads=%d", hit, err, loads)
	}
}
