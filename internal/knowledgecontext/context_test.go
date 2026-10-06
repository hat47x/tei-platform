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
			if len(ctx.DataDesign.BusinessIdentityCandidates) != 0 {
				t.Fatalf("invalid relation-unique produced a business identity candidate: %+v", ctx.DataDesign.BusinessIdentityCandidates)
			}
			if len(ctx.DataDesign.FunctionalDependencyCandidates) != 0 {
				t.Fatalf("invalid relation-unique produced an FD candidate: %+v", ctx.DataDesign.FunctionalDependencyCandidates)
			}
			if ctx.DataDesign.CandidateKeys.Nature != "unresolved" ||
				ctx.DataDesign.BusinessOccurrenceIdentifier.Nature != "unresolved" {
				t.Fatalf("invalid relation-unique created business identity: %+v", ctx.DataDesign)
			}
		})
	}
}

func TestDataDesignContextDoesNotResolveContradictoryValidationScope(t *testing.T) {
	w := dataDesignRelationUniqueWorld()
	w.Semantics[0].Data[0].Relations[0].Target = "sales.other"

	ctx := Build(
		w,
		sourceindex.Index{},
		subject.Subject{Kind: "semantic-validation", ID: "sales.order.lines.line_number_unique"},
	)
	if ctx.SemanticElement == nil {
		t.Fatal("validation semantic element is missing")
	}
	if ctx.DataDesign != nil {
		t.Fatalf("contradictory validation scope was guessed: %+v", ctx.DataDesign)
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

func TestDataDesignContextKeepsMultiplePhysicalKeysOutOfLogicalIdentity(t *testing.T) {
	w := dataDesignRelationUniqueWorld()
	w.Semantics[0].Data[1].Attributes = append(
		w.Semantics[0].Data[1].Attributes,
		semantic.Attribute{
			ID:      "sales.order_line.order_line_id",
			Symbol:  "orderLineId",
			TypeRef: "std.identifier",
		},
	)
	w.Persistence = []persistence.Mapping{
		{
			ID:           "sales_core.order_line",
			Database:     "sales_core",
			SemanticData: "sales.order_line",
			Table:        "public.order_lines",
			Columns: []persistence.ColumnMapping{
				{
					Attribute:   "sales.order_line.order_line_id",
					Column:      "order_line_id",
					Storage:     persistence.StorageInt64,
					Nullability: persistence.Required,
				},
				{
					Attribute:   "sales.order_line.line_number",
					Column:      "line_number",
					Storage:     persistence.StorageInt64,
					Nullability: persistence.Required,
				},
			},
			PrimaryKey: []core.CanonicalID{"sales.order_line.order_line_id"},
		},
		{
			ID:           "archive.order_line",
			Database:     "archive",
			SemanticData: "sales.order_line",
			Table:        "order_line_archive",
			Columns: []persistence.ColumnMapping{
				{
					Attribute:   "sales.order_line.line_number",
					Column:      "line_no",
					Storage:     persistence.StorageInt64,
					Nullability: persistence.Required,
				},
			},
			PrimaryKey: []core.CanonicalID{"sales.order_line.line_number"},
		},
	}

	ctx := Build(
		w,
		sourceindex.Index{},
		subject.Subject{Kind: "semantic-data", ID: "sales.order_line"},
	)
	if ctx.DataDesign == nil || len(ctx.DataDesign.PersistenceRealizations) != 2 {
		t.Fatalf("multiple persistence realizations are missing: %+v", ctx.DataDesign)
	}
	if ctx.DataDesign.PersistenceRealizations[0].Mapping != "archive.order_line" ||
		ctx.DataDesign.PersistenceRealizations[1].Mapping != "sales_core.order_line" {
		t.Fatalf("persistence realizations are not deterministic: %+v", ctx.DataDesign.PersistenceRealizations)
	}
	if len(ctx.DataDesign.PersistenceRealizations[0].Uniqueness) != 1 ||
		len(ctx.DataDesign.PersistenceRealizations[0].Uniqueness[0].Attributes) != 1 ||
		len(ctx.DataDesign.PersistenceRealizations[1].Uniqueness) != 1 ||
		len(ctx.DataDesign.PersistenceRealizations[1].Uniqueness[0].Attributes) != 1 ||
		ctx.DataDesign.PersistenceRealizations[0].Uniqueness[0].Attributes[0] != "sales.order_line.line_number" ||
		ctx.DataDesign.PersistenceRealizations[1].Uniqueness[0].Attributes[0] != "sales.order_line.order_line_id" {
		t.Fatalf("different physical primary keys were not preserved independently: %+v", ctx.DataDesign.PersistenceRealizations)
	}
	if ctx.DataDesign.BusinessOccurrenceIdentifier.Nature != "unresolved" ||
		ctx.DataDesign.CandidateKeys.Nature != "unresolved" ||
		ctx.DataDesign.FunctionalDependencies.Nature != "unresolved" {
		t.Fatalf("multiple physical keys changed logical identity authority: %+v", ctx.DataDesign)
	}
	if len(ctx.DataDesign.BusinessIdentityCandidates) != 1 ||
		ctx.DataDesign.BusinessIdentityCandidates[0].BasisConstraint != "sales.order.lines.line_number_unique" {
		t.Fatalf("physical keys replaced semantic identity candidate: %+v", ctx.DataDesign.BusinessIdentityCandidates)
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
	if len(ctx.DataDesign.BusinessIdentityCandidates) != 1 {
		t.Fatalf("valid business identity candidate is missing: %+v", ctx.DataDesign.BusinessIdentityCandidates)
	}
	identityCandidate := ctx.DataDesign.BusinessIdentityCandidates[0]
	if len(identityCandidate.Components) != 2 ||
		identityCandidate.Components[0].Kind != "relation-source-occurrence" ||
		identityCandidate.Components[0].Data != "sales.order" ||
		identityCandidate.Components[1].Kind != "attribute" ||
		identityCandidate.Components[1].Attribute != "sales.order_line.line_number" ||
		ctx.DataDesign.BusinessOccurrenceIdentifier.Nature != "unresolved" ||
		ctx.DataDesign.CandidateKeys.Nature != "unresolved" {
		t.Fatalf("business identity candidate boundary drifted: candidate=%+v state=%+v/%+v", identityCandidate, ctx.DataDesign.BusinessOccurrenceIdentifier, ctx.DataDesign.CandidateKeys)
	}
	if len(ctx.DataDesign.FunctionalDependencyCandidates) != 1 ||
		ctx.DataDesign.FunctionalDependencyCandidates[0].Nature != "candidate-from-accepted-scoped-uniqueness" ||
		ctx.DataDesign.FunctionalDependencies.Nature != "unresolved" {
		t.Fatalf("FD candidate boundary drifted: candidates=%+v state=%+v", ctx.DataDesign.FunctionalDependencyCandidates, ctx.DataDesign.FunctionalDependencies)
	}
	fdCandidate := ctx.DataDesign.FunctionalDependencyCandidates[0]
	if len(fdCandidate.Determinants) != 2 ||
		fdCandidate.Determinants[0].Kind != "relation-source-occurrence" ||
		fdCandidate.Determinants[0].Data != "sales.order" ||
		fdCandidate.Determinants[1].Kind != "attribute" ||
		fdCandidate.Determinants[1].Attribute != "sales.order_line.line_number" {
		t.Fatalf("FD candidate lost scoped determinant terms: %+v", fdCandidate)
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
