package functioncontext_test

import (
	"testing"

	"tei/internal/driver"
	"tei/internal/functioncontext"
)

// declaredWithColumns is declaredOrders plus the Persistence Mapping's physical columns.
func declaredWithColumns() driver.DeclaredBrowseRelations {
	d := declaredOrders
	d.Columns = []driver.DeclaredColumn{
		{Attribute: "sales.order.order_id", Column: "order_id"},
		{Attribute: "sales.order.total", Column: "total_amount"},
	}
	return d
}

func baseColumn(column, attname string) functioncontext.ColumnProvenance {
	return functioncontext.ColumnProvenance{Column: column, Relation: "public.orders", RelKind: "r", Attname: attname}
}

func withProvenance(t *testing.T, declared driver.DeclaredBrowseRelations, obs *functioncontext.AcquisitionObservation,
	prov []functioncontext.ColumnProvenance, rows [][]any) functioncontext.Context {
	t.Helper()
	c, err := functioncontext.From(functioncontext.Input{Execution: execution(rows, nil), Declared: declared, Acquisition: obs, Provenance: prov})
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func provenanceOf(c functioncontext.Context) []string {
	var out []string
	for _, column := range c.Columns {
		out = append(out, column.Provenance)
	}
	return out
}

func TestANullInTheDeclaredColumnOfASingleScanIsStored(t *testing.T) {
	prov := []functioncontext.ColumnProvenance{baseColumn("order_id", "order_id"), baseColumn("total_amount", "total_amount")}
	c := withProvenance(t, declaredWithColumns(), cleanObservation(), prov, [][]any{{"O-1", nil}, {"O-2", "2.00"}})
	if got := provenanceOf(c); got[0] != functioncontext.ProvDeclared || got[1] != functioncontext.ProvDeclared {
		t.Fatalf("provenance=%v", got)
	}
	if c.NullOrigin != functioncontext.NullOriginStored {
		t.Fatalf("nullOrigin=%s", c.NullOrigin)
	}
	if none := withProvenance(t, declaredWithColumns(), cleanObservation(), prov, [][]any{{"O-1", "1.00"}}); none.NullOrigin != functioncontext.NullOriginNone {
		t.Fatalf("no NULL in the rows must say so: %s", none.NullOrigin)
	}
}

func TestANullThatMayNotBeStoredStaysUnestablished(t *testing.T) {
	rows := [][]any{{"O-1", nil}}
	cases := map[string]struct {
		prov   []functioncontext.ColumnProvenance
		want   string
		mutate func(*functioncontext.AcquisitionObservation)
	}{
		"an expression (COALESCE / NULL::numeric)": {
			prov: []functioncontext.ColumnProvenance{baseColumn("order_id", "order_id"), {Column: "total_amount"}},
			want: functioncontext.ProvComputed,
		},
		"a view under the declared name": {
			prov: []functioncontext.ColumnProvenance{baseColumn("order_id", "order_id"),
				{Column: "total_amount", Relation: "public.orders", RelKind: "v", Attname: "total_amount"}},
			want: functioncontext.ProvNonTable,
		},
		"another column aliased to the declared name": {
			prov: []functioncontext.ColumnProvenance{baseColumn("order_id", "order_id"), baseColumn("total_amount", "customer_id")},
			want: functioncontext.ProvOtherColumn,
		},
		"another relation": {
			prov: []functioncontext.ColumnProvenance{baseColumn("order_id", "order_id"),
				{Column: "total_amount", Relation: "public.order_lines", RelKind: "r", Attname: "total_amount"}},
			want: functioncontext.ProvOtherColumn,
		},
		// An outer join returns NULL for "no match" in a column that IS the declared column.
		"the same declared column behind a join": {
			prov:   []functioncontext.ColumnProvenance{baseColumn("order_id", "order_id"), baseColumn("total_amount", "total_amount")},
			want:   functioncontext.ProvDeclared,
			mutate: func(o *functioncontext.AcquisitionObservation) { o.PlanNodes = append(o.PlanNodes, "Hash Join") },
		},
	}
	for name, tc := range cases {
		obs := cleanObservation()
		if tc.mutate != nil {
			tc.mutate(obs)
		}
		c := withProvenance(t, declaredWithColumns(), obs, tc.prov, rows)
		if got := c.Columns[1].Provenance; got != tc.want {
			t.Errorf("%s: provenance=%s want %s", name, got, tc.want)
		}
		if c.NullOrigin != functioncontext.NullOriginUnestablished {
			t.Errorf("%s: nullOrigin=%s", name, c.NullOrigin)
		}
	}
}

func TestProvenanceThatIsMissingOrMisalignedIsUnobservedAndClaimsNothing(t *testing.T) {
	rows := [][]any{{"O-1", nil}}
	for name, prov := range map[string][]functioncontext.ColumnProvenance{
		"none":        nil,
		"too few":     {baseColumn("order_id", "order_id")},
		"other names": {baseColumn("x", "order_id"), baseColumn("y", "total_amount")},
	} {
		c := withProvenance(t, declaredWithColumns(), cleanObservation(), prov, rows)
		for _, status := range provenanceOf(c) {
			if status != functioncontext.ProvUnobserved {
				t.Errorf("%s: %v", name, provenanceOf(c))
			}
		}
		if c.NullOrigin != functioncontext.NullOriginUnestablished || c.Grain.Level != functioncontext.GrainUnestablished {
			t.Errorf("%s: nullOrigin=%s level=%s", name, c.NullOrigin, c.Grain.Level)
		}
	}
	// Nothing declared to compare against: a plain table column is only "unchecked".
	none := driver.DeclaredBrowseRelations{Status: driver.DeclaredRelationsNone}
	prov := []functioncontext.ColumnProvenance{baseColumn("order_id", "order_id"), baseColumn("total_amount", "total_amount")}
	c := withProvenance(t, none, cleanObservation(), prov, rows)
	if c.Columns[1].Provenance != functioncontext.ProvUnchecked || c.NullOrigin != functioncontext.NullOriginUnestablished {
		t.Fatalf("%v %s", provenanceOf(c), c.NullOrigin)
	}
}

func TestEntityGrainNeedsASingleScanDeclaredKeyColumnsAndUniqueValues(t *testing.T) {
	prov := []functioncontext.ColumnProvenance{baseColumn("order_id", "order_id"), baseColumn("total_amount", "total_amount")}
	rows := [][]any{{"O-1", "1.00"}, {"O-2", "2.00"}}
	if c := withProvenance(t, declaredWithColumns(), cleanObservation(), prov, rows); c.Grain.Level != functioncontext.GrainEntity {
		t.Fatalf("level=%s", c.Grain.Level)
	}
	// Duplicated key values: not entity rows.
	if c := withProvenance(t, declaredWithColumns(), cleanObservation(), prov, [][]any{{"O-1", "1.00"}, {"O-1", "9.00"}}); c.Grain.Level != functioncontext.GrainUnestablished {
		t.Fatalf("duplicate key: level=%s", c.Grain.Level)
	}
	// A joined plan: the same rows are no longer rows of the declared relation.
	joined := cleanObservation()
	joined.PlanNodes = append(joined.PlanNodes, "Nested Loop")
	if c := withProvenance(t, declaredWithColumns(), joined, prov, rows); c.Grain.Level != functioncontext.GrainUnestablished {
		t.Fatalf("joined plan: level=%s", c.Grain.Level)
	}
	// The key column is another column aliased to the declared name.
	swapped := []functioncontext.ColumnProvenance{baseColumn("order_id", "customer_id"), baseColumn("total_amount", "total_amount")}
	if c := withProvenance(t, declaredWithColumns(), cleanObservation(), swapped, rows); c.Grain.Level != functioncontext.GrainUnestablished {
		t.Fatalf("aliased key: level=%s", c.Grain.Level)
	}
	// Unobserved plan: no claim.
	blind := cleanObservation()
	blind.PlanRelations, blind.PlanNodes, blind.PlanNote = nil, nil, "cannot explain"
	if c := withProvenance(t, declaredWithColumns(), blind, prov, rows); c.Grain.Level != functioncontext.GrainUnestablished {
		t.Fatalf("unobserved plan: level=%s", c.Grain.Level)
	}
}
