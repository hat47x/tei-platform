package driver_test

import (
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/driver"
	"tei/internal/environment"
	"tei/internal/ir"
)

// The reference Workspace stores sales.order.lines in public.order_lines,
// with the order's sales.order.order_id in column order_id, and its binding
// hands sales.order.order_id to input order_id, which the realization passes
// as parameter order_id. The cases below change one of those facts at a time
// and say by hand whether the Query still reads every line of the order.

const referenceOrderLinesSQL = `SELECT
    order_line_id,
    line_number,
    item_code,
    quantity,
    unit_price
FROM public.order_lines
WHERE order_id = :order_id
ORDER BY line_number, order_line_id;
`

type coverageEdit func(world *ir.World)

func declareCovers(relation string) coverageEdit {
	return func(world *ir.World) {
		for i := range world.HeaderDetailContractQueryBindings {
			if world.HeaderDetailContractQueryBindings[i].Pattern == "order_entry.order_lines" {
				world.HeaderDetailContractQueryBindings[i].CoversRelation = core.CanonicalID(relation)
			}
		}
	}
}

func useSQL(source string) coverageEdit {
	return func(world *ir.World) {
		for i := range world.ContractQuerySQLRealizations {
			if world.ContractQuerySQLRealizations[i].Target.Identity() == "sales_core/sales_core.order_line/find_by_order" {
				world.ContractQuerySQLRealizations[i].Source = source
			}
		}
	}
}

func qualifyOrderLines(t *testing.T, provider environment.DatabaseProvider, edits ...coverageEdit) driver.HeaderDetailCoverage {
	t.Helper()
	world := loadHeaderDetailReferenceWorld(t)
	for _, edit := range edits {
		edit(&world)
	}
	resolver := driver.NewResolver(world)
	resolution, err := resolver.ResolveHeaderDetailContractQuery("order_entry", "order_entry.order_lines")
	if err != nil {
		t.Fatal(err)
	}
	return resolver.QualifyHeaderDetailCoverage(resolution, provider)
}

func TestTheReferenceQueryReadsEveryLineOfTheOrder(t *testing.T) {
	world := loadHeaderDetailReferenceWorld(t)
	found := false
	for _, r := range world.ContractQuerySQLRealizations {
		if r.Target.Identity() == "sales_core/sales_core.order_line/find_by_order" {
			found = strings.Join(strings.Fields(r.Source), " ") == strings.Join(strings.Fields(referenceOrderLinesSQL), " ")
		}
	}
	if !found {
		t.Fatal("the reference SQL changed; the cases below are written from it")
	}

	got := qualifyOrderLines(t, environment.ProviderPostgres, declareCovers("sales.order.lines"))
	if !got.Complete || got.Declared != "sales.order.lines" || got.Reason != "" ||
		!strings.Contains(got.Basis, "order_id = sales.order.order_id") || !strings.Contains(got.Basis, "public.order_lines") {
		t.Fatalf("%+v", got)
	}
}

func TestADeclarationAloneIsNotCoverage(t *testing.T) {
	for _, tc := range []struct {
		name     string
		edits    []coverageEdit
		declared string
		reason   string
	}{
		{"nothing declared", nil, "", "does not state"},
		{"some declared", []coverageEdit{func(w *ir.World) {
			for i := range w.HeaderDetailContractQueryBindings {
				w.HeaderDetailContractQueryBindings[i].CoversSome = true
			}
		}}, "some", "only some rows"},
		{"another relation declared", []coverageEdit{declareCovers("sales.order.customer")}, "sales.order.customer", "but the Pattern reads"},
	} {
		got := qualifyOrderLines(t, environment.ProviderPostgres, tc.edits...)
		if got.Complete || got.Declared != tc.declared || !strings.Contains(got.Reason, tc.reason) || got.Basis != "" {
			t.Errorf("%s: %+v", tc.name, got)
		}
	}
}

// Every SQL below keeps the declaration. Each one drops, or may drop, a line
// of the order, or reads something the matcher does not read; none of them is
// complete.
func TestAQueryThatMayDropALineIsPartial(t *testing.T) {
	const head = "SELECT order_line_id, line_number, item_code, quantity, unit_price FROM public.order_lines WHERE order_id = :order_id"
	for name, source := range map[string]string{
		"another condition":               head + " AND order_line_id <> 3",
		"a condition that is always true": head + " AND 1 = 1",
		"a limit":                         head + " ORDER BY line_number LIMIT 2",
		"an offset":                       head + " OFFSET 1",
		"a fetch":                         head + " FETCH FIRST 2 ROWS ONLY",
		"distinct":                        "SELECT DISTINCT line_number FROM public.order_lines WHERE order_id = :order_id",
		"a group":                         head + " GROUP BY order_line_id",
		"an or":                           head + " OR order_id = :order_id",
		"a comment":                       head + " -- every line",
		"a block comment":                 "SELECT /* all */ order_line_id FROM public.order_lines WHERE order_id = :order_id",
		"a second statement":              head + "; DELETE FROM public.order_lines",
		"a union":                         head + " UNION SELECT order_line_id, line_number, item_code, quantity, unit_price FROM public.order_lines",
		"a join":                          "SELECT order_line_id FROM public.order_lines JOIN public.orders ON order_id = order_id WHERE order_id = :order_id",
		"another table":                   "SELECT order_line_id FROM public.order_lines_archive WHERE order_id = :order_id",
		"a table without its schema":      "SELECT order_line_id FROM order_lines WHERE order_id = :order_id",
		"another column":                  "SELECT order_line_id FROM public.order_lines WHERE line_number = :order_id",
		"a literal":                       "SELECT order_line_id FROM public.order_lines WHERE order_id = 1001",
		"a cast":                          "SELECT order_line_id FROM public.order_lines WHERE order_id = :order_id::bigint",
		"no condition":                    "SELECT order_line_id FROM public.order_lines",
		"a quoted name":                   `SELECT "order_line_id" FROM public.order_lines WHERE order_id = :order_id`,
		"every column":                    "SELECT * FROM public.order_lines WHERE order_id = :order_id",
		"a sub-query":                     "SELECT order_line_id FROM (SELECT * FROM public.order_lines) WHERE order_id = :order_id",
		"a with":                          "WITH x AS (SELECT 1) SELECT order_line_id FROM public.order_lines WHERE order_id = :order_id",
		"the same column twice":           head + " AND order_id = :order_id",
		"an unknown parameter":            "SELECT order_line_id FROM public.order_lines WHERE order_id = :customer_id",
		"empty":                           "",
		"a column compared with itself":   "SELECT order_line_id FROM public.order_lines WHERE order_id = order_id",
		"a pattern match":                 "SELECT order_line_id FROM public.order_lines WHERE order_id LIKE :order_id",
	} {
		got := qualifyOrderLines(t, environment.ProviderPostgres, declareCovers("sales.order.lines"), useSQL(source))
		if got.Complete || got.Reason == "" || got.Declared != "sales.order.lines" {
			t.Errorf("%s: %+v", name, got)
		}
	}
}

// Writing the same read differently does not change what it reads.
func TestTheSameReadWrittenDifferentlyIsComplete(t *testing.T) {
	for name, source := range map[string]string{
		"one line, lower case": "select order_line_id from public.order_lines where order_id = :order_id",
		"no order":             "SELECT order_line_id, line_number FROM public.order_lines WHERE order_id = :order_id",
		"descending":           "SELECT order_line_id FROM public.order_lines WHERE order_id = :order_id ORDER BY line_number DESC, order_line_id ASC",
		"no semicolon, tabs":   "SELECT\torder_line_id\nFROM\tpublic.order_lines\nWHERE\torder_id = :order_id",
		"upper-case table":     "SELECT order_line_id FROM PUBLIC.ORDER_LINES WHERE ORDER_ID = :order_id",
	} {
		got := qualifyOrderLines(t, environment.ProviderPostgres, declareCovers("sales.order.lines"), useSQL(source))
		if !got.Complete {
			t.Errorf("%s: %+v", name, got)
		}
	}
}

func TestTheDefinitionsTheQueryIsMatchedAgainst(t *testing.T) {
	covered := declareCovers("sales.order.lines")
	for _, tc := range []struct {
		name     string
		provider environment.DatabaseProvider
		edit     coverageEdit
		reason   string
	}{
		{"another provider has no realization", environment.ProviderOracle, nil, "no SQL realization"},
		{"the parameter may be null", environment.ProviderPostgres, func(w *ir.World) {
			for i := range w.ContractQuerySQLRealizations {
				for j := range w.ContractQuerySQLRealizations[i].Inputs {
					w.ContractQuerySQLRealizations[i].Inputs[j].Nullable = true
				}
			}
		}, "may be null"},
		{"the input is bound to another attribute", environment.ProviderPostgres, func(w *ir.World) {
			for i := range w.HeaderDetailContractQueryBindings {
				for j := range w.HeaderDetailContractQueryBindings[i].Inputs {
					w.HeaderDetailContractQueryBindings[i].Inputs[j].SourceAttribute = "sales.order.customer_id"
				}
			}
		}, "is not bound to it"},
		{"the relation is stored in another column", environment.ProviderPostgres, func(w *ir.World) {
			for i := range w.Persistence {
				for j := range w.Persistence[i].RelationStorage {
					for k := range w.Persistence[i].RelationStorage[j].Components {
						w.Persistence[i].RelationStorage[j].Components[k].Column = "parent_order_id"
					}
				}
			}
		}, "does not compare column parent_order_id"},
		{"the relation is stored with two columns", environment.ProviderPostgres, func(w *ir.World) {
			for i := range w.Persistence {
				for j := range w.Persistence[i].RelationStorage {
					storage := &w.Persistence[i].RelationStorage[j]
					extra := storage.Components[0]
					extra.Column, extra.SourceAttribute = "tenant_id", "sales.order.customer_id"
					storage.Components = append(storage.Components, extra)
				}
			}
		}, "compares 1 column(s)"},
		{"the relation storage is missing", environment.ProviderPostgres, func(w *ir.World) {
			for i := range w.Persistence {
				w.Persistence[i].RelationStorage = nil
			}
		}, "does not say how"},
		{"the lines are stored in another table", environment.ProviderPostgres, func(w *ir.World) {
			for i := range w.Persistence {
				if w.Persistence[i].SemanticData == "sales.order_line" {
					w.Persistence[i].Table = "public.lines"
				}
			}
		}, "stored in public.lines"},
		{"the lines have no persistence mapping", environment.ProviderPostgres, func(w *ir.World) {
			kept := w.Persistence[:0]
			for _, m := range w.Persistence {
				if m.SemanticData != "sales.order_line" {
					kept = append(kept, m)
				}
			}
			w.Persistence = kept
		}, "no persistence mapping"},
	} {
		edits := []coverageEdit{covered}
		if tc.edit != nil {
			edits = append(edits, tc.edit)
		}
		got := qualifyOrderLines(t, tc.provider, edits...)
		if got.Complete || !strings.Contains(got.Reason, tc.reason) {
			t.Errorf("%s: %+v", tc.name, got)
		}
	}
}

// A Relation stored with two columns is read by comparing both, joined by
// AND. Comparing them with OR reads the rows of other headers too.
func TestARelationStoredWithTwoColumns(t *testing.T) {
	twoColumns := func(w *ir.World) {
		for i := range w.Persistence {
			for j := range w.Persistence[i].RelationStorage {
				storage := &w.Persistence[i].RelationStorage[j]
				extra := storage.Components[0]
				extra.Column, extra.SourceAttribute = "customer_id", "sales.order.customer_id"
				storage.Components = append(storage.Components, extra)
			}
		}
		for i := range w.ContractQuerySQLRealizations {
			r := &w.ContractQuerySQLRealizations[i]
			if r.Target.Identity() == "sales_core/sales_core.order_line/find_by_order" {
				extra := r.Inputs[0]
				extra.Input, extra.Parameter = "customer_id", "customer_id"
				r.Inputs = append(r.Inputs, extra)
			}
		}
		for i := range w.HeaderDetailContractQueryBindings {
			b := &w.HeaderDetailContractQueryBindings[i]
			extra := b.Inputs[0]
			extra.SourceAttribute, extra.Input = "sales.order.customer_id", "customer_id"
			b.Inputs = append(b.Inputs, extra)
		}
	}
	const head = "SELECT order_line_id FROM public.order_lines WHERE "
	for _, tc := range []struct {
		where    string
		complete bool
	}{
		{"order_id = :order_id AND customer_id = :customer_id", true},
		{"customer_id = :customer_id AND order_id = :order_id", true},
		{"order_id = :order_id OR customer_id = :customer_id", false},
		{"order_id = :order_id", false},
		{"order_id = :customer_id AND customer_id = :order_id", false},
	} {
		got := qualifyOrderLines(t, environment.ProviderPostgres, declareCovers("sales.order.lines"), twoColumns, useSQL(head+tc.where))
		if got.Complete != tc.complete {
			t.Errorf("%s: %+v", tc.where, got)
		}
	}
}

// The relation storage of another Relation in the same mapping says nothing
// about this one.
func TestTheRelationStorageOfAnotherRelationIsNotRead(t *testing.T) {
	otherFirst := func(w *ir.World) {
		for i := range w.Persistence {
			m := &w.Persistence[i]
			if len(m.RelationStorage) == 0 {
				continue
			}
			other := m.RelationStorage[0]
			other.Relation = "sales.order.returned_lines"
			other.Components = append(other.Components[:0:0], other.Components...)
			other.Components[0].Column = "returned_order_id"
			m.RelationStorage = append(m.RelationStorage, other)
		}
	}
	got := qualifyOrderLines(t, environment.ProviderPostgres, declareCovers("sales.order.lines"), otherFirst)
	if !got.Complete {
		t.Fatalf("%+v", got)
	}
}

// Two SQL realizations of one Contract Query for one provider leave it
// unclear which one runs; neither is matched.
func TestTwoRealizationsOfOneQueryArePartial(t *testing.T) {
	duplicate := func(w *ir.World) {
		for _, r := range w.ContractQuerySQLRealizations {
			if r.Target.Identity() == "sales_core/sales_core.order_line/find_by_order" {
				r.Source = "SELECT order_line_id FROM public.order_lines WHERE order_id = :order_id AND line_number = :order_id"
				w.ContractQuerySQLRealizations = append(w.ContractQuerySQLRealizations, r)
				return
			}
		}
	}
	got := qualifyOrderLines(t, environment.ProviderPostgres, declareCovers("sales.order.lines"), duplicate)
	if got.Complete || !strings.Contains(got.Reason, "more than one SQL realization") {
		t.Fatalf("%+v", got)
	}
}

// Repeating a Relation storage declaration is ambiguous even if both copies
// currently happen to say the same thing. A later edit must not silently
// change which definition qualifies the query as complete.
func TestTwoStorageDefinitionsOfOneRelationArePartial(t *testing.T) {
	duplicate := func(w *ir.World) {
		for i := range w.Persistence {
			mapping := &w.Persistence[i]
			if mapping.SemanticData != "sales.order_line" {
				continue
			}
			for _, storage := range mapping.RelationStorage {
				if storage.Relation == "sales.order.lines" {
					mapping.RelationStorage = append(mapping.RelationStorage, storage)
					return
				}
			}
		}
	}
	got := qualifyOrderLines(t, environment.ProviderPostgres, declareCovers("sales.order.lines"), duplicate)
	if got.Complete || !strings.Contains(got.Reason, "more than once") {
		t.Fatalf("ambiguous Relation storage qualified as complete: %+v", got)
	}
}
