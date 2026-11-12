package functioncontext_test

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/driver"
	"tei/internal/driver/queryresult"
	"tei/internal/environment"
	"tei/internal/functioncontext"
	"tei/internal/query"
	"tei/internal/semantic"
)

func execution(rows [][]any, keys []semantic.CandidateKey) driver.BrowseQueryEnvironmentExecution {
	cols := []queryresult.Column{
		{Raw: "order_id", Attribute: "sales.order.order_id"},
		{Raw: "total_amount", Attribute: "sales.order.total"},
	}
	raw := make([][]any, len(rows))
	copy(raw, rows)
	return driver.BrowseQueryEnvironmentExecution{
		Environment:       "development",
		Instance:          environment.DatabaseInstance{ID: "sales_core.dev", Definition: "sales_core", Provider: environment.ProviderPostgres},
		QuerySourceSHA256: "abc",
		Execution: driver.BrowseQueryExecution{
			Resolution: driver.PatternResolution{
				Application: application.Application{ID: "order_entry"},
				Pattern:     application.Pattern{ID: "order_entry.recent_orders"},
				Query:       &query.EmbeddedSQL{ID: "order_entry.recent_orders", Dialect: environment.ProviderPostgres},
				Datasource:  &application.DatabaseRequirement{Alias: "sales", TargetDatabase: "sales_core"},
				Data:        &semantic.Data{ID: "sales.order", CandidateKeys: keys},
			},
			Result: driver.BrowseQueryResult{
				Raw:    queryresult.RawResult{Columns: []string{"order_id", "total_amount"}, Rows: raw},
				Mapped: queryresult.MappedResult{Columns: cols, Rows: rows},
			},
		},
	}
}

func build(t *testing.T, e driver.BrowseQueryEnvironmentExecution) functioncontext.Context {
	t.Helper()
	c, err := functioncontext.From(functioncontext.Input{Execution: e})
	if err != nil {
		t.Fatal(err)
	}
	return c
}

// The ceiling is the same for every input: no row count, including zero, may raise it.
func TestCoverageCeilingDoesNotMoveWithTheRowset(t *testing.T) {
	for name, rows := range map[string][][]any{
		"zero rows": {},
		"one row":   {{"O-1", "1.00"}},
		"many rows": {{"O-1", "1.00"}, {"O-2", "2.00"}, {"O-3", "3.00"}},
	} {
		c := build(t, execution(rows, nil))
		if c.Coverage.Status != semantic.CoverageUnestablished || c.Coverage.EmptyJustified {
			t.Errorf("%s: coverage=%+v", name, c.Coverage)
		}
		if c.Coverage.Visibility != functioncontext.NotObserved || c.Currency != functioncontext.NotEvaluated {
			t.Errorf("%s: visibility/currency raised: %+v %s", name, c.Coverage, c.Currency)
		}
		if c.Source.SourceRevision != functioncontext.SourceUnknown || c.Binding.Assertion != functioncontext.BindingHostAsserted {
			t.Errorf("%s: source/binding raised", name)
		}
	}
}

func TestNullIsNullAndItsOriginIsNotClaimed(t *testing.T) {
	c := build(t, execution([][]any{{"O-1", nil}, {"O-2", "0.00"}}, nil))
	if c.Rows[0][1].State != functioncontext.StateNull || c.Rows[1][1].State != functioncontext.StateValue {
		t.Fatalf("cells=%+v", c.Rows)
	}
	if c.Rows[1][1].Value != "0.00" {
		t.Fatalf("a zero amount must stay a value: %+v", c.Rows[1][1])
	}
	if c.NullOrigin != "unestablished" {
		t.Fatalf("nullOrigin=%s", c.NullOrigin)
	}
}

func TestIdentityIsNeverInferredAndDuplicatesAreCounted(t *testing.T) {
	rows := [][]any{{"O-1", "1.00"}, {"O-1", "1.00"}, {"O-2", "2.00"}}
	c := build(t, execution(rows, nil))
	if c.Grain.Identity != functioncontext.IdentityUndeclared || len(c.Grain.Key) != 0 {
		t.Fatalf("an undeclared key was inferred: %+v", c.Grain)
	}
	if c.Grain.ExactDuplicateRow != 1 {
		t.Fatalf("duplicates=%d", c.Grain.ExactDuplicateRow)
	}
	keyed := build(t, execution(rows, []semantic.CandidateKey{{ID: "sales.order.pk", Attributes: []core.CanonicalID{"sales.order.order_id"}}}))
	if keyed.Grain.Identity != functioncontext.IdentityDeclared || len(keyed.Grain.Key) != 1 {
		t.Fatalf("a declared key was dropped: %+v", keyed.Grain)
	}
}

func TestTimeIsRenderedInUTCAndDisclosureEqualsAcquisition(t *testing.T) {
	jst := time.FixedZone("JST", 9*3600)
	c := build(t, execution([][]any{{time.Date(2026, 11, 7, 19, 10, 0, 0, jst), "5.25"}}, nil))
	if got := c.Rows[0][0].Value; got != "2026-11-07T10:10:00Z" {
		t.Fatalf("time=%v", got)
	}
	if c.TimeFormat != functioncontext.TimeRendering || c.Coverage.Pagination != functioncontext.NotDeclared {
		t.Fatalf("rendering/pagination words: %s %s", c.TimeFormat, c.Coverage.Pagination)
	}
	if c.Disclosure.Rows != 1 || c.Disclosure.Columns != 2 || !strings.HasPrefix(c.Disclosure.Omitted, "none") {
		t.Fatalf("disclosure=%+v", c.Disclosure)
	}
}

func TestRefusesAnExecutionWithoutADigestOrWithRaggedRows(t *testing.T) {
	e := execution([][]any{{"O-1", "1.00"}}, nil)
	e.QuerySourceSHA256 = ""
	if _, err := functioncontext.From(functioncontext.Input{Execution: e}); err == nil {
		t.Fatal("an execution without a Source digest was projected")
	}
	e = execution([][]any{{"O-1"}}, nil)
	if _, err := functioncontext.From(functioncontext.Input{Execution: e}); err == nil {
		t.Fatal("a ragged mapped row was projected")
	}
}

func TestParameterValuesAreNeverPartOfTheContext(t *testing.T) {
	c, err := functioncontext.From(functioncontext.Input{
		Execution:  execution(nil, nil),
		Parameters: []functioncontext.ParameterShape{{Name: "from", GoType: "time.Time"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(c)
	if strings.Contains(string(raw), "2026-11-07T09") {
		t.Fatalf("a parameter value leaked: %s", raw)
	}
}
