package driver_test

import (
	"context"
	"errors"
	"testing"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/driver"
	"tei/internal/driver/queryresult"
	"tei/internal/query"
)

// D2AF-01/02: a Host field-read grant that existed at preflight but is
// revoked during an actual Provider callback must prevent both the raw
// Browse result and the derived value from being returned to the caller.
//
// This is a synchronous synthetic Provider and Host authority. It does not
// prove that a real Host's grant API is atomic, or test OS/Network disclosure.
func TestD2DriverRechecksAuthorityAfterCompleteSourceRead(t *testing.T) {
	for _, denied := range []core.CanonicalID{
		"inventory.product.stock",
		"inventory.product.available",
	} {
		t.Run(string(denied), func(t *testing.T) {
			resolver, _, plan, proof := d2DriverFixture()
			revoked := false
			checks := 0
			calls := 0
			auth := d2Authority(func(_ context.Context, id core.CanonicalID) bool {
				checks++
				return !revoked || id != denied
			})
			executor := d2Executor(func(_ context.Context, _ query.EmbeddedSQL,
				_ application.DatabaseRequirement, _ map[string]any, _ int) (driver.BrowseQueryResult, bool, error) {
				calls++
				revoked = true // authority changes after D2 preflight
				return driver.BrowseQueryResult{Raw: queryresult.RawResult{
					Columns: []string{"product_id", "stock", "reserved"},
					Rows:    [][]any{{"A", "100", "10"}},
				}}, true, nil
			})
			got, err := resolver.ExecuteDerivedBrowseQuery(
				context.Background(), "inventory", "inventory.browse",
				executor, nil, plan, proof, auth,
			)
			if !errors.Is(err, query.ErrDerivedUnauthorized) {
				t.Fatalf("revocation during Provider callback returned result=%#v, error=%v", got, err)
			}
			if calls != 1 {
				t.Fatalf("expected exactly one bounded complete Provider read; got %d", calls)
			}
			if checks < 5 {
				t.Fatalf("post-read Host authorization was not rechecked: %d checks", checks)
			}
		})
	}
}

// D2AF-03: the unchanged current Host grant remains sufficient for the
// existing supported, bounded complete raw-source D2 path.
func TestD2DriverCurrentGrantRemainsValidAfterCompleteSourceRead(t *testing.T) {
	resolver, _, plan, proof := d2DriverFixture()
	checks := 0
	auth := d2Authority(func(_ context.Context, _ core.CanonicalID) bool {
		checks++
		return true
	})
	executor := d2Executor(func(_ context.Context, _ query.EmbeddedSQL,
		_ application.DatabaseRequirement, _ map[string]any, _ int) (driver.BrowseQueryResult, bool, error) {
		return driver.BrowseQueryResult{Raw: queryresult.RawResult{
			Columns: []string{"product_id", "stock", "reserved"},
			Rows:    [][]any{{"A", "100", "10"}},
		}}, true, nil
	})
	got, err := resolver.ExecuteDerivedBrowseQuery(
		context.Background(), "inventory", "inventory.browse",
		executor, nil, plan, proof, auth,
	)
	if err != nil {
		t.Fatalf("unchanged authorized source was rejected: %v", err)
	}
	if got.Derived.Count != 1 || checks < 8 {
		t.Fatalf("pre+post current Host grant verification missing: count=%d checks=%d", got.Derived.Count, checks)
	}
}
