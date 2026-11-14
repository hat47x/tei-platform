package driver_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"tei/internal/application"
	"tei/internal/driver"
	"tei/internal/driver/queryresult"
	"tei/internal/query"
)

func TestBrowseQueryRejectsBadInputsBeforeAnyExecutorCall(t *testing.T) {
	from := time.Date(2026, 11, 7, 10, 5, 0, 0, time.UTC)
	var nilTime *time.Time
	cases := []struct {
		name    string
		params  map[string]any
		wantErr string
	}{
		{"required from missing", nil, "requires input from"},
		{"text timestamp is not typed", map[string]any{"from": "2026-11-07T10:05:00Z"}, "requires std.datetime"},
		{"integer timestamp is not typed", map[string]any{"from": int64(42)}, "requires std.datetime"},
		{"explicit null is not missing", map[string]any{"from": nil}, "NULL"},
		{"typed nil pointer is not time.Time", map[string]any{"from": nilTime}, "requires std.datetime"},
		{"extra status cannot pass through", map[string]any{"from": from, "status": "accepted"}, "undeclared input status"},
		{"other query name is not from", map[string]any{"after": from}, "undeclared input after"},
		{"deterministic unknown input error", map[string]any{"zeta": 1, "alpha": 1, "from": from}, "undeclared input alpha"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			executor := browseQueryExecutorFunc(func(context.Context, query.EmbeddedSQL, application.DatabaseRequirement, map[string]any) (driver.BrowseQueryResult, error) {
				calls++
				return driver.BrowseQueryResult{}, nil
			})
			_, err := driver.NewResolver(referenceWorld(t)).ExecuteBrowseQuery(
				context.Background(), "order_entry", "order_entry.recent_orders", executor, tc.params,
			)
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("want refusal %q, got %v", tc.wantErr, err)
			}
			if calls != 0 {
				t.Fatalf("executor was called %d times after refused input", calls)
			}
			if strings.Contains(err.Error(), "2026-11-07T10:05:00Z") {
				t.Fatalf("a parameter value was included in the diagnostic: %s", err)
			}
		})
	}
}

func TestBrowseQueryPassesOnlyTheDeclaredDatetimeUnchanged(t *testing.T) {
	from := time.Date(2026, 11, 7, 19, 5, 0, 123000000, time.FixedZone("JST", 9*3600))
	calls := 0
	executor := browseQueryExecutorFunc(func(_ context.Context, q query.EmbeddedSQL, _ application.DatabaseRequirement, got map[string]any) (driver.BrowseQueryResult, error) {
		calls++
		if len(q.Inputs) != 1 || q.Inputs[0].Name != "from" || q.Inputs[0].Type != "std.datetime" {
			t.Fatalf("declared Query input changed: %+v", q.Inputs)
		}
		v, ok := got["from"].(time.Time)
		if len(got) != 1 || !ok || !v.Equal(from) || v.Location() != from.Location() {
			t.Fatalf("runtime timestamp was coerced: %+v", got)
		}
		return driver.BrowseQueryResult{
			Raw: queryresult.RawResult{
				Columns: []string{"order_id", "customer_id", "ordered_at", "status", "total_amount"},
				Rows:    [][]any{},
			},
		}, nil
	})
	_, err := driver.NewResolver(referenceWorld(t)).ExecuteBrowseQuery(
		context.Background(), "order_entry", "order_entry.recent_orders", executor, map[string]any{"from": from},
	)
	if err != nil || calls != 1 {
		t.Fatalf("declared input failed to reach executor exactly once: err=%v calls=%d", err, calls)
	}
}

func TestBrowseQueryRefusesDuplicateAndUnqualifiedInputDeclarations(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func(*query.EmbeddedSQL)
		wantErr string
	}{
		{"duplicate declaration", func(q *query.EmbeddedSQL) {
			q.Inputs = append(q.Inputs, query.Input{Name: "from", Type: "std.datetime"})
		}, "more than once"},
		{"undeclared runtime type contract", func(q *query.EmbeddedSQL) {
			q.Inputs[0].Type = "std.money"
		}, "unsupported runtime binding"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			w := referenceWorld(t)
			changed := false
			for i := range w.Queries {
				if w.Queries[i].ID == "order_entry.recent_orders" {
					tc.mutate(&w.Queries[i])
					changed = true
				}
			}
			if !changed {
				t.Fatal("test fixture Query not found")
			}
			calls := 0
			executor := browseQueryExecutorFunc(func(context.Context, query.EmbeddedSQL, application.DatabaseRequirement, map[string]any) (driver.BrowseQueryResult, error) {
				calls++
				return driver.BrowseQueryResult{}, nil
			})
			_, err := driver.NewResolver(w).ExecuteBrowseQuery(
				context.Background(), "order_entry", "order_entry.recent_orders", executor,
				map[string]any{"from": time.Date(2026, 11, 7, 10, 0, 0, 0, time.UTC)},
			)
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) || calls != 0 {
				t.Fatalf("expected %q before executor; got err=%v calls=%d", tc.wantErr, err, calls)
			}
		})
	}
}

// An old synthetic Query can still be replayed in the generic reference
// executor to verify historical behavior. This must never be counted as a
// qualified typed Product Read; the Environment path refuses the same input.
func TestBrowseQueryHistoricalNoInputsIsNotTypedQualification(t *testing.T) {
	w := referenceWorld(t)
	for i := range w.Queries {
		if w.Queries[i].ID == "order_entry.recent_orders" {
			w.Queries[i].Inputs = nil
		}
	}
	calls := 0
	executor := browseQueryExecutorFunc(func(_ context.Context, q query.EmbeddedSQL, _ application.DatabaseRequirement, _ map[string]any) (driver.BrowseQueryResult, error) {
		calls++
		if len(q.Inputs) != 0 {
			t.Fatalf("historical fixture unexpectedly gained typed input: %+v", q.Inputs)
		}
		return driver.BrowseQueryResult{
			Raw: queryresult.RawResult{
				Columns: []string{"order_id", "customer_id", "ordered_at", "status", "total_amount"},
				Rows:    [][]any{},
			},
		}, nil
	})
	out, err := driver.NewResolver(w).ExecuteBrowseQuery(context.Background(), "order_entry", "order_entry.recent_orders", executor,
		map[string]any{"from": time.Date(2026, 11, 7, 10, 5, 0, 0, time.UTC)})
	if err != nil || calls != 1 || out.Resolution.Query == nil || len(out.Resolution.Query.Inputs) != 0 {
		t.Fatalf("legacy fixture should replay without claiming typed input: err=%v calls=%d", err, calls)
	}
}
