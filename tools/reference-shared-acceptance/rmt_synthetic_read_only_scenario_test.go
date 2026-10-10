package main

import (
	"context"
	"fmt"
	"math/big"
	"reflect"
	"testing"
	"time"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/driver"
	"tei/internal/driver/queryresult"
	"tei/internal/query"
	"tei/internal/semantic/validationeval"
	"tei/internal/workspace"
)

// rmtSyntheticReadOnlyExecutor is a test-only provider. Its result is NOT
// evidence of an actual database, original Driver, or production authorization.
type rmtSyntheticReadOnlyExecutor func(context.Context, query.EmbeddedSQL, application.DatabaseRequirement, map[string]any) (driver.BrowseQueryResult, error)

func (e rmtSyntheticReadOnlyExecutor) ExecuteBrowseQuery(
	ctx context.Context,
	q query.EmbeddedSQL,
	db application.DatabaseRequirement,
	parameters map[string]any,
) (driver.BrowseQueryResult, error) {
	return e(ctx, q, db, parameters)
}

// This is a bounded TEI-only vertical slice:
// synthetic RMT candidate -> explicit Review/Decision -> workspace reload ->
// Driver query resolution -> synthetic read-only provider -> mapped result.
// It cannot prove the original runtime behavior or a real DB transaction.
func TestRMTSyntheticReadOnlySliceRequiresAcceptedMeaningAndRecordsNoStoreMutation(t *testing.T) {
	dir := scratchWorkspace(t)
	originalTree := treeDigest(t, dir)
	beforeAccept, err := workspace.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	previousRules, err := driver.NewResolver(beforeAccept.World).ResolveApplicationValidations("order_entry")
	if err != nil {
		t.Fatal(err)
	}
	if _, exists := findRule(previousRules, "sales.order_line.unit_price_positive"); exists {
		t.Fatal("fixture unexpectedly contains the rule before Acceptance")
	}
	candidate := unitPriceWithSelectionCandidate(revisionOfDir(t, dir))
	grounded, err := attachRMTTranscriptionEvidence(
		mustJSON(t, candidate),
		rmtEvidenceJSON(t, true),
	)
	if err != nil {
		t.Fatal(err)
	}

	// RMT source observations are candidate evidence, not authoring authority.
	ev, err := evaluate(dir, mustJSON(t, grounded))
	if err != nil {
		t.Fatal(err)
	}
	if !ev.preview.Acceptable {
		t.Fatalf("synthetic candidate was not acceptable: %+v", allDiagnostics(ev.preview))
	}
	assertTreeUnchanged(t, originalTree, treeDigest(t, dir))
	review := buildReviewProjection(ev)
	if !review.PreviewAcceptable || review.ReviewFingerprint == "" {
		t.Fatalf("no exact Review for explicit Decision: %+v", review)
	}
	decision, err := accept(dir, mustJSON(t, grounded), acceptOptions{
		ExpectedReviewFingerprint: review.ReviewFingerprint,
	})
	if err != nil || decision.Decision != "accepted" || !decision.SourceChanged {
		t.Fatalf("scratch acceptance failed: decision=%+v err=%v", decision, err)
	}
	if decision.ReviewBinding == nil || decision.ReviewBinding.Fingerprint != review.ReviewFingerprint {
		t.Fatal("accepted meaning was not bound to reviewed projection")
	}

	loaded, err := workspace.Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	resolver := driver.NewResolver(loaded.World)
	acceptedRules, err := resolver.ResolveApplicationValidations(core.MustCanonicalID("order_entry"))
	if err != nil {
		t.Fatal(err)
	}
	unitPriceRule, found := findRule(acceptedRules, core.MustCanonicalID("sales.order_line.unit_price_positive"))
	if !found {
		t.Fatalf("reload did not preserve accepted rule: %+v", acceptedRules)
	}

	// In-memory state is deliberately independent of the source model. This
	// allows a before/after check without claiming actual database observations.
	store := map[int64]string{101: "OPEN"}
	before := map[int64]string{101: "OPEN"}
	from := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	calls := 0
	executor := rmtSyntheticReadOnlyExecutor(func(
		_ context.Context,
		q query.EmbeddedSQL,
		datasource application.DatabaseRequirement,
		params map[string]any,
	) (driver.BrowseQueryResult, error) {
		calls++
		if q.ID != "order_entry.recent_orders" || datasource.Alias != "sales" ||
			datasource.TargetDatabase != "sales_core" {
			return driver.BrowseQueryResult{}, fmt.Errorf("unexpected query/datasource: %s / %+v", q.ID, datasource)
		}
		if len(params) != 1 || params["from"] != from {
			return driver.BrowseQueryResult{}, fmt.Errorf("unexpected query parameters: %+v", params)
		}
		return driver.BrowseQueryResult{Raw: queryresult.RawResult{
			Columns: []string{"total_amount", "status", "ordered_at", "customer_id", "order_id"},
			Rows: [][]any{{
				3500.0,
				store[101],
				time.Date(2026, 9, 2, 12, 0, 0, 0, time.UTC),
				int64(7),
				int64(101),
			}},
		}}, nil
	})

	execution, err := resolver.ExecuteBrowseQuery(
		context.Background(),
		core.MustCanonicalID("order_entry"),
		core.MustCanonicalID("order_entry.recent_orders"),
		executor,
		map[string]any{"from": from},
	)
	if err != nil {
		t.Fatal(err)
	}
	if calls != 1 || len(execution.Result.Mapped.Rows) != 1 {
		t.Fatalf("query execution calls=%d result=%+v", calls, execution.Result.Mapped)
	}
	var orderID any
	for i, column := range execution.Result.Mapped.Columns {
		if column.Attribute == "sales.order.order_id" {
			orderID = execution.Result.Mapped.Rows[0][i]
		}
	}
	if orderID != int64(101) {
		t.Fatalf("explicit result mapping was lost: orderID=%#v", orderID)
	}

	// The accepted rule must actually change what this scenario can evaluate.
	// Associate the explicitly mapped order ID with independently held synthetic
	// line values. Merely resolving the accepted rule without evaluating it
	// would not establish a causal link between the Decision and this scenario.
	lineUnitPrices := map[int64]*big.Rat{101: big.NewRat(35, 1)}
	selectedOrderID, ok := orderID.(int64)
	if !ok {
		t.Fatalf("mapped order ID has unexpected type: %T", orderID)
	}
	readUnitPrice := func() numberSource {
		return numberSource{
			"sales.order_line.unit_price": new(big.Rat).Set(lineUnitPrices[selectedOrderID]),
		}
	}
	result, err := validationeval.Evaluate(unitPriceRule, readUnitPrice())
	if err != nil || result != validationeval.ResultSatisfied {
		t.Fatalf("accepted Validation did not pass synthetic positive line: result=%s err=%v", result, err)
	}
	lineUnitPrices[selectedOrderID] = big.NewRat(0, 1)
	result, err = validationeval.Evaluate(unitPriceRule, readUnitPrice())
	if err != nil || result != validationeval.ResultNotSatisfied {
		t.Fatalf("accepted Validation did not reject synthetic nonpositive line: result=%s err=%v", result, err)
	}
	if !reflect.DeepEqual(store, before) {
		t.Fatalf("synthetic read-only store changed: before=%+v after=%+v", before, store)
	}

	// Missing required result mappings are not silently interpreted as
	// an empty or successful query. No content is inferred from legacy names.
	_, err = resolver.ExecuteBrowseQuery(
		context.Background(), "order_entry", "order_entry.recent_orders",
		rmtSyntheticReadOnlyExecutor(func(
			context.Context, query.EmbeddedSQL, application.DatabaseRequirement, map[string]any,
		) (driver.BrowseQueryResult, error) {
			return driver.BrowseQueryResult{Raw: queryresult.RawResult{
				Columns: []string{"order_id"},
				Rows: [][]any{{int64(101)}},
			}}, nil
		}),
		map[string]any{"from": from},
	)
	if err == nil {
		t.Fatal("incomplete physical result was accepted as an authoritative Semantic result")
	}
	if !reflect.DeepEqual(store, before) {
		t.Fatal("negative query mutated the synthetic store")
	}
}
