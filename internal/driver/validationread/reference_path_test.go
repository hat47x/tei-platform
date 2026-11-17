package validationread_test

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/driver"
	"tei/internal/driver/queryresult"
	sqlquery "tei/internal/driver/queryruntime/sql"
	"tei/internal/driver/validationread"
	"tei/internal/environment"
	"tei/internal/ir"
	"tei/internal/semantic"
	"tei/internal/semantic/validationeval"
	"tei/internal/workspace"
)

// These tests read the order lines of the reference Workspace the way the
// product reads them. Nothing about the path is built here: the Pattern, its
// Contract Query binding, the Environment's Database instance, the executor
// binding, the SQL realization and its SQL file, and the rules the
// Application uses are all taken from reference/workspace as loaded.
//
// The only stand-in is the database: a Runner that returns rows for the SQL
// it is handed. The same path against a PostgreSQL server is in
// postgres_path_test.go.
//
// The expected answers are written by hand from the rows each case returns.

func repositoryRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("the test source path is not available")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
}

func referenceWorld(t *testing.T) ir.World {
	t.Helper()
	loaded, err := workspace.Load(filepath.Join(repositoryRoot(t), "reference", "workspace"))
	if err != nil {
		t.Fatalf("the reference Workspace did not load: %v", err)
	}
	return loaded.World
}

// orderHeader is the header record the detail is read for.
func orderHeader(orderID int64) queryresult.MappedResult {
	return queryresult.MappedResult{
		Columns: []queryresult.Column{{Raw: "order_id", Attribute: "sales.order.order_id"}},
		Rows:    [][]any{{orderID}},
	}
}

// rowsRunner stands in for the database. It keeps what it was handed.
type rowsRunner struct {
	columns    []string
	rows       [][]any
	source     string
	parameters map[string]any
}

func (r *rowsRunner) ExecuteSQL(_ context.Context, source string, parameters map[string]any) (sqlquery.RawExecution, error) {
	r.source, r.parameters = source, parameters
	return sqlquery.RawExecution{Result: queryresult.RawResult{Columns: r.columns, Rows: r.rows}}, nil
}

var orderLineColumns = []string{"order_line_id", "line_number", "item_code", "quantity", "unit_price"}

// readOrderLines runs the reference header-detail Pattern in the development
// Environment and reads its rows as the lines of the order.
func readOrderLines(t *testing.T, world ir.World, runner sqlquery.Runner, orderID int64, coverage validationeval.Coverage) *validationread.Detail {
	t.Helper()
	execution := executeOrderLines(t, world, runner, orderID)
	detail, err := validationread.FromHeaderDetail(world.Semantics, execution.Execution, coverage)
	if err != nil {
		t.Fatal(err)
	}
	return detail
}

func executeOrderLines(t *testing.T, world ir.World, runner sqlquery.Runner, orderID int64) driver.HeaderDetailContractQueryEnvironmentExecution {
	t.Helper()
	executor := sqlquery.Executor{
		Provider:     environment.ProviderPostgres,
		Realizations: world.ContractQuerySQLRealizations,
		Runner:       runner,
	}
	execution, err := driver.NewResolver(world).ExecuteHeaderDetailContractQueryInEnvironment(
		context.Background(),
		"development", "order_entry", "order_entry.order_lines",
		orderHeader(orderID), 0,
		driver.ContractQueryExecutorMap{
			{Environment: "development", Instance: "sales_core.dev", Capability: "database.query.postgres"}: executor,
		},
	)
	if err != nil {
		t.Fatalf("the order lines were not read: %v", err)
	}
	return execution
}

func orderEntryRules(t *testing.T, world ir.World) []semantic.Validation {
	t.Helper()
	rules, err := driver.NewResolver(world).ResolveApplicationValidations("order_entry")
	if err != nil {
		t.Fatal(err)
	}
	return rules
}

func told(decision validationeval.Decision) string {
	if decision.Undecided != nil {
		return "undecided: " + string(decision.Undecided.Reason) + " " + string(decision.Undecided.Attribute)
	}
	return string(decision.Result)
}

// answers asks every rule the Application uses and writes the answers as
// "<rule>: relation <answer>" or "<rule>: rows <answer>, <answer>".
func answers(t *testing.T, world ir.World, detail *validationread.Detail) []string {
	t.Helper()
	var out []string
	for _, rule := range orderEntryRules(t, world) {
		decisions, err := detail.Decide(rule, validationeval.Operators{})
		if err != nil {
			t.Fatalf("%s: %v", rule.ID, err)
		}
		if decisions.Rule != rule.ID || (decisions.Relation != nil) == (decisions.Rows != nil) {
			t.Fatalf("%s: exactly one of the two answers must be set: %+v", rule.ID, decisions)
		}
		if decisions.Relation != nil {
			out = append(out, string(rule.ID)+": relation "+told(*decisions.Relation))
			continue
		}
		rows := make([]string, 0, len(decisions.Rows))
		for _, decision := range decisions.Rows {
			rows = append(rows, told(decision))
		}
		out = append(out, string(rule.ID)+": rows "+strings.Join(rows, ", "))
	}
	return out
}

func TestOrderLinesOfTheReferenceWorkspaceAreReadThroughTheProductPath(t *testing.T) {
	world := referenceWorld(t)
	// Three lines of order 1001. The third has quantity 0. Numbers arrive as
	// a PostgreSQL driver hands them over: bigint as int64, numeric as text.
	runner := &rowsRunner{columns: orderLineColumns, rows: [][]any{
		{int64(11), int64(1), "ITEM-A", "2.500", "1000.00"},
		{int64(12), int64(2), "ITEM-B", "1.000", "500.50"},
		{int64(13), int64(3), "ITEM-C", "0.000", "10.00"},
	}}
	detail := readOrderLines(t, world, runner, 1001, validationeval.CoverageComplete)

	// The SQL the Runner was handed is the reference Workspace's own file,
	// and the parameter is the order of the header record.
	file, err := os.ReadFile(filepath.Join(repositoryRoot(t), "reference", "workspace", "databases", "sales-core", "query-realizations", "order-line-find-by-order.sql"))
	if err != nil {
		t.Fatal(err)
	}
	normalize := func(text string) string { return strings.Join(strings.Fields(text), " ") }
	if normalize(runner.source) != normalize(string(file)) {
		t.Fatalf("the SQL that ran is not the reference file:\n%s", runner.source)
	}
	if len(runner.parameters) != 1 || runner.parameters["order_id"] != int64(1001) {
		t.Fatalf("parameters=%#v", runner.parameters)
	}

	sameAnswers(t, "order 1001", answers(t, world, detail), []string{
		"sales.order_line.quantity_positive: rows satisfied, satisfied, not-satisfied",
		"sales.order.lines.line_number_unique: relation satisfied",
		"sales.order_line.line_number_positive: rows satisfied, satisfied, satisfied",
	})
}

func TestOrderLinesWithARepeatedLineNumber(t *testing.T) {
	world := referenceWorld(t)
	rows := [][]any{
		{int64(21), int64(1), "ITEM-A", "1", "100"},
		{int64(22), int64(1), "ITEM-B", "2", "100"},
		{int64(23), int64(0), "ITEM-C", "3", "100"},
	}
	want := []string{
		"sales.order_line.quantity_positive: rows satisfied, satisfied, satisfied",
		"sales.order.lines.line_number_unique: relation not-satisfied",
		"sales.order_line.line_number_positive: rows satisfied, satisfied, not-satisfied",
	}
	// A repeated line number that was seen stays a violation when the rows
	// are only part of the order.
	for _, coverage := range []validationeval.Coverage{validationeval.CoverageComplete, validationeval.CoveragePartial} {
		detail := readOrderLines(t, world, &rowsRunner{columns: orderLineColumns, rows: rows}, 1002, coverage)
		sameAnswers(t, "order 1002, "+string(coverage), answers(t, world, detail), want)
	}
}

func TestOrderLinesReadInPart(t *testing.T) {
	world := referenceWorld(t)
	rows := [][]any{
		{int64(31), int64(1), "ITEM-A", "1", "100"},
		{int64(32), int64(2), "ITEM-B", "2", "100"},
	}
	// The rules about one line answer for the lines that were read. The rule
	// about all the lines cannot: a line that was not read may repeat a
	// number.
	part := readOrderLines(t, world, &rowsRunner{columns: orderLineColumns, rows: rows}, 1003, validationeval.CoveragePartial)
	sameAnswers(t, "a page of order 1003", answers(t, world, part), []string{
		"sales.order_line.quantity_positive: rows satisfied, satisfied",
		"sales.order.lines.line_number_unique: relation undecided: relation-partially-read sales.order_line.line_number",
		"sales.order_line.line_number_positive: rows satisfied, satisfied",
	})

	// When the caller states nothing, the rule about all the lines is
	// refused, and the rules about one line are still answered.
	unstated := readOrderLines(t, world, &rowsRunner{columns: orderLineColumns, rows: rows}, 1003, "")
	for _, rule := range orderEntryRules(t, world) {
		decisions, err := unstated.Decide(rule, validationeval.Operators{})
		if rule.Kind == semantic.ValidationRelationUnique {
			if err == nil || !strings.Contains(err.Error(), "unknown coverage") {
				t.Fatalf("%s: err=%v decisions=%+v", rule.ID, err, decisions)
			}
			continue
		}
		if err != nil || len(decisions.Rows) != 2 || told(decisions.Rows[0]) != "satisfied" || told(decisions.Rows[1]) != "satisfied" {
			t.Fatalf("%s: err=%v decisions=%+v", rule.ID, err, decisions)
		}
	}
}

func TestAnOrderWithoutLines(t *testing.T) {
	world := referenceWorld(t)
	none := readOrderLines(t, world, &rowsRunner{columns: orderLineColumns}, 1004, validationeval.CoverageComplete)
	// No line means no answer about a line. It does not mean a satisfied
	// rule: Rows is empty, not one "satisfied".
	sameAnswers(t, "order 1004", answers(t, world, none), []string{
		"sales.order_line.quantity_positive: rows ",
		"sales.order.lines.line_number_unique: relation satisfied",
		"sales.order_line.line_number_positive: rows ",
	})
}

// The quantity of the reference Workspace is declared std.decimal. A driver
// that hands it over as a binary floating point value is refused, and the
// error names the line.
func TestAQuantityHandedOverAsFloatingPointIsRefused(t *testing.T) {
	world := referenceWorld(t)
	detail := readOrderLines(t, world, &rowsRunner{columns: orderLineColumns, rows: [][]any{
		{int64(41), int64(1), "ITEM-A", "1", "100"},
		{int64(42), int64(2), "ITEM-B", float64(0.5), "100"},
	}}, 1005, validationeval.CoverageComplete)
	rules := orderEntryRules(t, world)
	decisions, err := detail.Decide(rules[0], validationeval.Operators{})
	if err == nil || !strings.Contains(err.Error(), "row 1") || !strings.Contains(err.Error(), "binary floating point") || decisions.Rows != nil {
		t.Fatalf("err=%v decisions=%+v", err, decisions)
	}
	// The line numbers are not affected.
	unique, err := detail.Decide(rules[1], validationeval.Operators{})
	if err != nil || told(*unique.Relation) != "satisfied" {
		t.Fatalf("err=%v", err)
	}
}

// A realization that does not return the quantity leaves it unread. The
// reference realization returns it, so the mapping is narrowed here; the
// rest of the path is unchanged.
func TestARealizationThatDoesNotReturnTheQuantity(t *testing.T) {
	world := referenceWorld(t)
	narrowed := false
	for i := range world.ContractQuerySQLRealizations {
		realization := &world.ContractQuerySQLRealizations[i]
		if realization.Target.Identity() != "sales_core/sales_core.order_line/find_by_order" {
			continue
		}
		kept := realization.ResultColumns[:0:0]
		for _, column := range realization.ResultColumns {
			if column.Attribute != "sales.order_line.quantity" {
				kept = append(kept, column)
			}
		}
		narrowed = len(kept) == len(realization.ResultColumns)-1
		realization.ResultColumns = kept
	}
	if !narrowed {
		t.Fatal("the reference realization no longer maps the quantity")
	}
	detail := readOrderLines(t, world, &rowsRunner{
		columns: []string{"order_line_id", "line_number", "item_code", "unit_price"},
		rows:    [][]any{{int64(51), int64(1), "ITEM-A", "100"}},
	}, 1006, validationeval.CoverageComplete)
	sameAnswers(t, "order 1006 without the quantity", answers(t, world, detail), []string{
		"sales.order_line.quantity_positive: rows undecided: value-not-fetched sales.order_line.quantity",
		"sales.order.lines.line_number_unique: relation satisfied",
		"sales.order_line.line_number_positive: rows satisfied",
	})
}

func TestARuleTheRowsCannotAnswerIsRefused(t *testing.T) {
	world := referenceWorld(t)
	detail := readOrderLines(t, world, &rowsRunner{columns: orderLineColumns, rows: [][]any{
		{int64(61), int64(1), "ITEM-A", "1", "100"},
	}}, 1007, validationeval.CoverageComplete)

	for name, tc := range map[string]struct {
		rule semantic.Validation
		want string
	}{
		// The total belongs to the order, not to its lines.
		"an item of another record": {
			semantic.Validation{ID: "sales.order.total_positive", Kind: semantic.ValidationPositive, Attribute: "sales.order.total"},
			"is not an item of sales.order_line",
		},
		"another relation": {
			semantic.Validation{ID: "x", Kind: semantic.ValidationRelationUnique, Relation: "sales.order.customer", Attribute: "sales.order_line.line_number"},
			"not of sales.order.customer",
		},
		"a kind the reader does not know": {
			semantic.Validation{ID: "x", Kind: "some-new-kind", Attribute: "sales.order_line.quantity"},
			"does not know whether",
		},
	} {
		decisions, err := detail.Decide(tc.rule, validationeval.Operators{})
		if err == nil || !strings.Contains(err.Error(), tc.want) || decisions.Relation != nil || decisions.Rows != nil {
			t.Fatalf("%s: err=%v decisions=%+v", name, err, decisions)
		}
	}
}

// Every kind the model lists is asked either of one row or of all the rows.
// A kind added to the model without a decision here fails this test instead
// of being refused only when a Workspace first uses it.
func TestEveryKindOfRuleIsAskedOneWayOrTheOther(t *testing.T) {
	detail, err := validationread.NewDetail(testSemantics, membersRelation, membershipData, nil, validationeval.CoverageComplete)
	if err != nil {
		t.Fatal(err)
	}
	relationKinds := map[semantic.ValidationKind]bool{semantic.ValidationRelationUnique: true, semantic.ValidationNoOverlap: true}
	for _, kind := range semantic.ValidationKinds {
		rule := semantic.Validation{ID: core.CanonicalID("rule." + string(kind)), Kind: kind,
			Attribute: startsOn, Compared: endsOn, Relation: membersID}
		if kind == semantic.ValidationRelationUnique {
			rule.Attribute = personID
		}
		decisions, err := detail.Decide(rule, validationeval.Operators{})
		if err != nil {
			// With no rows, a rule about one row asks nothing, so only the
			// reader's own refusal can appear here.
			t.Fatalf("%s: %v", kind, err)
		}
		if (decisions.Relation != nil) != relationKinds[kind] {
			t.Fatalf("%s: asked of all the rows=%v, want %v", kind, decisions.Relation != nil, relationKinds[kind])
		}
	}
}

// A rule about presence is decided from whether the row holds a value, not
// from the value. An empty text is a value; only SQL NULL is none; a column
// the query did not return is not known to be either.
func TestPresenceIsReadFromTheRow(t *testing.T) {
	endRequiresPerson := semantic.Validation{ID: "organization.membership.person_required_with_end",
		Kind: semantic.ValidationRequiredIf, Attribute: personID, Compared: endsOn}
	detail, err := validationread.NewDetail(testSemantics, membersRelation, membershipData, []driver.ContractQueryRow{
		row(map[core.CanonicalID]any{personID: "P001", endsOn: "2026-09-30"}),
		row(map[core.CanonicalID]any{personID: nil, endsOn: "2026-09-30"}),
		row(map[core.CanonicalID]any{personID: nil, endsOn: nil}),
		row(map[core.CanonicalID]any{personID: "", endsOn: "2026-09-30"}),
		row(map[core.CanonicalID]any{personID: nil}),
		row(map[core.CanonicalID]any{endsOn: "2026-09-30"}),
		// The end is not a date. Presence does not parse it.
		row(map[core.CanonicalID]any{personID: nil, endsOn: "soon"}),
	}, validationeval.CoverageComplete)
	if err != nil {
		t.Fatal(err)
	}
	decisions, err := detail.Decide(endRequiresPerson, validationeval.Operators{})
	if err != nil {
		t.Fatal(err)
	}
	got := make([]string, 0, len(decisions.Rows))
	for _, decision := range decisions.Rows {
		got = append(got, told(decision))
	}
	sameAnswers(t, "a person is required when there is an end", got, []string{
		"satisfied",
		"not-satisfied",
		"satisfied",
		"satisfied",
		"undecided: value-not-fetched " + string(endsOn),
		"undecided: value-not-fetched " + string(personID),
		"not-satisfied",
	})

	one, _ := detail.Row(0)
	if _, err := one.ReadPresence("sales.order.total"); err == nil || !strings.Contains(err.Error(), "is not an item of") {
		t.Fatalf("err=%v", err)
	}
}
