package knowledgecontext

import (
	"testing"

	"tei/internal/core"
	"tei/internal/database/persistence"
	"tei/internal/ir"
	"tei/internal/semantic"
	"tei/internal/sourceindex"
	"tei/internal/subject"
)

func TestDataDesignContextDoesNotPromoteInvalidRelationUnique(t *testing.T) {
	for _, tc := range []struct {
		name   string
		mutate func(*ir.World)
	}{
		{
			name: "relation is not many",
			mutate: func(w *ir.World) {
				w.Semantics[0].Data[0].Relations[0].Cardinality = semantic.CardinalityOne
			},
		},
		{
			name: "attribute is outside relation target",
			mutate: func(w *ir.World) {
				w.Semantics[0].Validations[0].Attribute = "sales.order.total"
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := dataDesignRelationUniqueWorld()
			tc.mutate(&w)
			ctx := Build(
				w,
				sourceindex.Index{},
				subject.Subject{Kind: "semantic-data", ID: "sales.order_line"},
			)
			if ctx.DataDesign == nil {
				t.Fatal("data design context is missing")
			}
			if len(ctx.DataDesign.ScopedUniqueness) != 0 {
				t.Fatalf("invalid relation-unique was promoted: %+v", ctx.DataDesign.ScopedUniqueness)
			}
			if ctx.DataDesign.CandidateKeys.Nature != "unresolved" ||
				ctx.DataDesign.BusinessOccurrenceIdentifier.Nature != "unresolved" {
				t.Fatalf("invalid relation-unique created business identity: %+v", ctx.DataDesign)
			}
		})
	}
}

func TestDataDesignContextKeepsPersistenceKeysAsRealizationFacts(t *testing.T) {
	w := dataDesignRelationUniqueWorld()
	w.Persistence = []persistence.Mapping{
		{
			ID:           "sales_core.order_line",
			Database:     "sales_core",
			SemanticData: "sales.order_line",
			Table:        "public.order_lines",
			Columns: []persistence.ColumnMapping{
				{
					Attribute:   "sales.order_line.line_number",
					Column:      "line_number",
					Storage:     persistence.StorageInt64,
					Nullability: persistence.Required,
				},
			},
			PrimaryKey: []core.CanonicalID{"sales.order_line.line_number"},
			UniqueConstraints: []persistence.UniqueConstraintMapping{
				{
					Name:       "order_lines_line_number_uq",
					Attributes: []core.CanonicalID{"sales.order_line.line_number"},
				},
			},
		},
	}

	ctx := Build(
		w,
		sourceindex.Index{},
		subject.Subject{Kind: "semantic-data", ID: "sales.order_line"},
	)
	if ctx.DataDesign == nil || len(ctx.DataDesign.PersistenceRealizations) != 1 {
		t.Fatalf("persistence realization is missing: %+v", ctx.DataDesign)
	}
	realization := ctx.DataDesign.PersistenceRealizations[0]
	if realization.Authority != "database-owned-realization" ||
		len(realization.Uniqueness) != 2 {
		t.Fatalf("persistence uniqueness facts are incomplete: %+v", realization)
	}
	for _, uniqueness := range realization.Uniqueness {
		if uniqueness.Nature != "accepted-realization" ||
			len(uniqueness.Attributes) != 1 ||
			uniqueness.Attributes[0] != "sales.order_line.line_number" {
			t.Fatalf("persistence uniqueness fact drifted: %+v", uniqueness)
		}
	}
	if ctx.DataDesign.BusinessOccurrenceIdentifier.Nature != "unresolved" ||
		ctx.DataDesign.CandidateKeys.Nature != "unresolved" ||
		ctx.DataDesign.FunctionalDependencies.Nature != "unresolved" {
		t.Fatalf("persistence uniqueness was promoted to logical identity: %+v", ctx.DataDesign)
	}
	if !containsDataDesignSubject(
		ctx.DataDesign.EvidenceMissing,
		subject.Subject{Kind: "persistence", ID: "sales_core.order_line"},
	) {
		t.Fatalf("missing persistence source Evidence was hidden: %+v", ctx.DataDesign.EvidenceMissing)
	}
}

func TestDataDesignContextReportsMissingAuthoritativeEvidence(t *testing.T) {
	ctx := Build(
		dataDesignRelationUniqueWorld(),
		sourceindex.Index{},
		subject.Subject{Kind: "semantic-data", ID: "sales.order_line"},
	)
	if ctx.DataDesign == nil {
		t.Fatal("data design context is missing")
	}
	if len(ctx.DataDesign.ScopedUniqueness) != 1 {
		t.Fatalf("valid scoped uniqueness is missing: %+v", ctx.DataDesign.ScopedUniqueness)
	}
	for _, want := range []subject.Subject{
		{Kind: "semantic-data", ID: "sales.order"},
		{Kind: "semantic-data", ID: "sales.order_line"},
		{Kind: "semantic-attribute", ID: "sales.order_line.line_number"},
		{Kind: "semantic-relation", ID: "sales.order.lines"},
		{Kind: "semantic-validation", ID: "sales.order.lines.line_number_unique"},
	} {
		if !containsDataDesignSubject(ctx.DataDesign.EvidenceMissing, want) {
			t.Fatalf("missing Evidence subject %v was not reported: %+v", want, ctx.DataDesign.EvidenceMissing)
		}
	}
}

func dataDesignRelationUniqueWorld() ir.World {
	return ir.World{
		Semantics: []semantic.Namespace{
			{
				ID: "sales",
				Data: []semantic.Data{
					{
						ID:     "sales.order",
						Symbol: "Order",
						Attributes: []semantic.Attribute{
							{ID: "sales.order.total", Symbol: "total", TypeRef: "std.money"},
						},
						Relations: []semantic.Relation{
							{
								ID:          "sales.order.lines",
								Symbol:      "lines",
								Target:      "sales.order_line",
								Cardinality: semantic.CardinalityMany,
							},
						},
					},
					{
						ID:     "sales.order_line",
						Symbol: "OrderLine",
						Attributes: []semantic.Attribute{
							{ID: "sales.order_line.line_number", Symbol: "lineNumber", TypeRef: "std.integer"},
						},
					},
				},
				Validations: []semantic.Validation{
					{
						ID:        "sales.order.lines.line_number_unique",
						Symbol:    "line_number_unique",
						Kind:      semantic.ValidationRelationUnique,
						Relation:  "sales.order.lines",
						Attribute: "sales.order_line.line_number",
					},
				},
			},
		},
	}
}

func containsDataDesignSubject(values []subject.Subject, want subject.Subject) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
