package driver_test

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/driver"
	"tei/internal/driver/queryresult"
	"tei/internal/query"
	"tei/internal/semantic/derivationref"
)

type d3DriverHostVerifier struct {
	currentCalls int
	deny         bool
}

func (v *d3DriverHostVerifier) VerifyCurrent(context.Context, derivationref.CacheWitness) error {
	v.currentCalls++
	if v.deny {
		return errors.New("current Authority revoked")
	}
	return nil
}

func (v *d3DriverHostVerifier) VerifyAligned(context.Context, derivationref.CacheWitness) error {
	return nil
}

type d3DriverSourceReader struct {
	observation derivationref.SourceObservation
	reads       int
}

func (r *d3DriverSourceReader) ReadCurrentSource(_ context.Context, id string) (derivationref.SourceObservation, error) {
	r.reads++
	if id != r.observation.SourceID {
		return derivationref.SourceObservation{}, errors.New("unknown current source")
	}
	return r.observation, nil
}

func d3DriverEvidence(plan query.DerivedSelect) (derivationref.CacheWitness, time.Time) {
	observed := time.Date(2026, 10, 15, 1, 0, 0, 0, time.UTC)
	return derivationref.CacheWitness{
		RuleID:             plan.Rule.ID,
		RuleRevision:       "rule-v1",
		TargetID:           plan.Binding.TargetID,
		SemanticRevision:   "semantic-v1",
		AcceptanceRevision: "acceptance-v1",
		ProviderRevision:   "provider-v1",
		Inputs: []derivationref.Revision{
			{ID: "inventory.product.reserved", Revision: "input-v1"},
			{ID: "inventory.product.stock", Revision: "input-v1"},
		},
		Sources: []derivationref.SourceObservation{{
			SourceID:   "inventory",
			Revision:   "snapshot-v1",
			SourceAsOf: observed.Add(-time.Minute),
			ObservedAt: observed,
		}},
		Scope: derivationref.CacheScope{
			Environment:     "test",
			Tenant:          "tenant-1",
			Subject:         "user-1",
			PermissionEpoch: "permission-v1",
		},
	}, observed.Add(10 * time.Second)
}

func d3DriverRawRows() [][]any {
	// Deliberately out of Query mapping order. The existing D2 mapping, not
	// positional guessing, must supply inputs to D3.
	return [][]any{
		{"100", "A", "120"}, {"10", "B", "100"}, {"30", "C", "80"},
		{"20", "D", "70"}, {"70", "E", "40"},
	}
}

func TestD3DriverOptInMappingCacheHitAndCurrentAuthorization(t *testing.T) {
	r, q, plan, sqlEvidence := d2DriverFixture()
	witness, now := d3DriverEvidence(plan)
	host := &d3DriverHostVerifier{}
	sources := &d3DriverSourceReader{observation: witness.Sources[0]}
	verifier := derivationref.SourceBoundVerifier{Current: host, Sources: sources}
	policy := derivationref.CachePolicy{MaxAge: time.Minute}
	var memo query.DerivedMemo
	loads := 0
	executor := d2Executor(func(_ context.Context, got query.EmbeddedSQL, ds application.DatabaseRequirement, _ map[string]any, maxRows int) (driver.BrowseQueryResult, bool, error) {
		loads++
		if got.ID != q.ID || got.Source != q.Source || ds.Alias != "store" || maxRows != plan.MaxRows {
			t.Fatalf("native Query/datasource/maxRows changed: %#v %#v bound=%d", got, ds, maxRows)
		}
		return driver.BrowseQueryResult{Raw: queryresult.RawResult{
			Columns: []string{"reserved", "product_id", "stock"},
			Rows:    d3DriverRawRows(),
		}}, true, nil
	})
	denied := core.CanonicalID("")
	readChecks := 0
	authority := d2Authority(func(_ context.Context, id core.CanonicalID) bool {
		readChecks++
		return id != denied
	})
	run := func(at time.Time) (query.DerivedSelectResult, bool, error) {
		return r.ExecuteFreshDerivedBrowseQuery(context.Background(), "inventory", "inventory.browse",
			executor, nil, plan, sqlEvidence, authority, &memo, witness, policy, at, verifier)
	}

	first, hit, err := run(now)
	if err != nil || hit || loads != 1 || first.Count != 3 || len(first.Rows) != 2 {
		t.Fatalf("first D3 source: %#v hit=%v loads=%d err=%v", first, hit, loads, err)
	}
	ids := []string{}
	for _, row := range first.Rows {
		ids = append(ids, row.Values["inventory.product.id"].Decimal)
	}
	if !reflect.DeepEqual(ids, []string{"C", "D"}) {
		t.Fatalf("derived filter/sort/paging or mapping changed: %v", ids)
	}
	beforeChecks := readChecks
	second, hit, err := run(now.Add(time.Second))
	if err != nil || !hit || loads != 1 || second.Count != first.Count ||
		host.currentCalls != 3 || sources.reads != 3 || readChecks <= beforeChecks {
		t.Fatalf("cache hit skipped current Source or FieldRead: hit=%v loads=%d current=%d sourceReads=%d reads=%d err=%v",
			hit, loads, host.currentCalls, sources.reads, readChecks, err)
	}

	// Current FieldRead is required on every cache hit. No Provider dispatch
	// is permitted after a revocation even when the memo entry is still young.
	denied = "inventory.product.available"
	if _, hit, err := run(now.Add(2 * time.Second)); !errors.Is(err, query.ErrDerivedUnauthorized) || hit || loads != 1 {
		t.Fatalf("revoked cache hit accepted: hit=%v loads=%d err=%v", hit, loads, err)
	}
	denied = ""
	host.deny = true
	if _, hit, err := run(now.Add(3 * time.Second)); !errors.Is(err, derivationref.ErrCacheDenied) || hit || loads != 1 {
		t.Fatalf("revoked Host current witness accepted: hit=%v loads=%d err=%v", hit, loads, err)
	}
}

func TestD3DriverOptInRejectsSQLSourceAndFieldReadChanges(t *testing.T) {
	r, _, plan, proof := d2DriverFixture()
	witness, now := d3DriverEvidence(plan)
	host := &d3DriverHostVerifier{}
	sources := &d3DriverSourceReader{observation: witness.Sources[0]}
	verifier := derivationref.SourceBoundVerifier{Current: host, Sources: sources}
	policy := derivationref.CachePolicy{MaxAge: time.Minute}
	var memo query.DerivedMemo
	loads := 0
	complete := true
	rows := d3DriverRawRows()
	denyAfterLoad := false
	denyField := false
	executor := d2Executor(func(context.Context, query.EmbeddedSQL, application.DatabaseRequirement, map[string]any, int) (driver.BrowseQueryResult, bool, error) {
		loads++
		denyField = denyAfterLoad
		return driver.BrowseQueryResult{Raw: queryresult.RawResult{
			Columns: []string{"reserved", "product_id", "stock"},
			Rows:    rows,
		}}, complete, nil
	})
	authority := d2Authority(func(_ context.Context, id core.CanonicalID) bool {
		return !(denyField && id == "inventory.product.available")
	})
	run := func(evidence query.CompleteSourceEvidence) (query.DerivedSelectResult, bool, error) {
		return r.ExecuteFreshDerivedBrowseQuery(context.Background(), "inventory", "inventory.browse",
			executor, nil, plan, evidence, authority, &memo, witness, policy, now, verifier)
	}

	invalid := proof
	invalid.SQLSHA256 = "stale-or-self-reported-sql-evidence"
	if _, hit, err := run(invalid); !errors.Is(err, query.ErrDerivedUnsupported) || hit || loads != 0 {
		t.Fatalf("SQL preflight bypassed: hit=%v loads=%d err=%v", hit, loads, err)
	}
	complete = false
	if _, hit, err := run(proof); !errors.Is(err, query.ErrDerivedIncomplete) || hit || loads != 1 {
		t.Fatalf("incomplete Provider source accepted: hit=%v loads=%d err=%v", hit, loads, err)
	}
	complete = true
	rows = append(rows, make([]any, 3))
	for len(rows) <= plan.MaxRows {
		rows = append(rows, []any{"1", "overflow", "2"})
	}
	if _, hit, err := run(proof); !errors.Is(err, query.ErrDerivedIncomplete) || hit || loads != 2 {
		t.Fatalf("overbound Provider source accepted: hit=%v loads=%d err=%v", hit, loads, err)
	}
	rows = d3DriverRawRows()
	denyAfterLoad = true
	if _, hit, err := run(proof); !errors.Is(err, query.ErrDerivedUnauthorized) || hit || loads != 3 {
		t.Fatalf("post-load FieldRead revocation accepted: hit=%v loads=%d err=%v", hit, loads, err)
	}
	// Reauthorization after the failed load must not make that result a memo hit.
	denyAfterLoad, denyField = false, false
	result, hit, err := run(proof)
	if err != nil || hit || result.Count != 3 || loads != 4 {
		t.Fatalf("failed authorization was cached: hit=%v loads=%d err=%v", hit, loads, err)
	}
}
