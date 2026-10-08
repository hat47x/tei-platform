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

func TestDataDesignContextDoesNotDeriveCandidatesFromDuplicateSemanticIdentity(t *testing.T) {
	for _, tc := range []struct {
		name          string
		ambiguityKind string
		ambiguityID   core.CanonicalID
		mutate        func(*ir.World)
	}{
		{
			name:          "duplicate data id",
			ambiguityKind: "duplicate-semantic-data",
			ambiguityID:   "sales.order_line",
			mutate: func(w *ir.World) {
				duplicate := w.Semantics[0].Data[1]
				w.Semantics[0].Data = append(w.Semantics[0].Data, duplicate)
			},
		},
		{
			name:          "duplicate attribute id",
			ambiguityKind: "duplicate-semantic-attribute",
			ambiguityID:   "sales.order_line.line_number",
			mutate: func(w *ir.World) {
				w.Semantics[0].Data[0].Attributes = append(
					w.Semantics[0].Data[0].Attributes,
					semantic.Attribute{
						ID:      "sales.order_line.line_number",
						Symbol:  "conflictingLineNumber",
						TypeRef: "std.integer",
					},
				)
			},
		},
		{
			name:          "duplicate relation id",
			ambiguityKind: "duplicate-semantic-relation",
			ambiguityID:   "sales.order.lines",
			mutate: func(w *ir.World) {
				w.Semantics[0].Data[1].Relations = append(
					w.Semantics[0].Data[1].Relations,
					semantic.Relation{
						ID:          "sales.order.lines",
						Symbol:      "conflictingLines",
						Target:      "sales.order_line",
						Cardinality: semantic.CardinalityMany,
					},
				)
			},
		},
		{
			name:          "duplicate validation id",
			ambiguityKind: "duplicate-semantic-validation",
			ambiguityID:   "sales.order.lines.line_number_unique",
			mutate: func(w *ir.World) {
				duplicate := w.Semantics[0].Validations[0]
				w.Semantics[0].Validations = append(w.Semantics[0].Validations, duplicate)
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := dataDesignRelationUniqueWorld()
			tc.mutate(&w)

			dataContext := Build(
				w,
				sourceindex.Index{},
				subject.Subject{Kind: "semantic-data", ID: "sales.order_line"},
			)
			if dataContext.DataDesign == nil {
				t.Fatal("data design context is missing")
			}
			if len(dataContext.DataDesign.ScopedUniqueness) != 0 ||
				len(dataContext.DataDesign.BusinessIdentityCandidates) != 0 ||
				len(dataContext.DataDesign.FunctionalDependencyCandidates) != 0 {
				t.Fatalf("ambiguous semantic identity produced design candidates: %+v", dataContext.DataDesign)
			}
			if dataContext.DataDesign.ResolutionStatus != "ambiguous" {
				t.Fatalf("structurally ambiguous context was marked reusable: %+v", dataContext.DataDesign)
			}
			if !containsDataDesignAmbiguity(
				dataContext.DataDesign.Ambiguities,
				tc.ambiguityKind,
				tc.ambiguityID,
			) {
				t.Fatalf("semantic ambiguity was not exposed: want=%s:%s got=%+v", tc.ambiguityKind, tc.ambiguityID, dataContext.DataDesign.Ambiguities)
			}
			if !containsDataDesignString(
				dataContext.DataDesign.AuthorityLimits,
				"ambiguous-semantic-identity-does-not-produce-design-candidates",
			) {
				t.Fatalf("ambiguous identity authority limit is missing: %+v", dataContext.DataDesign.AuthorityLimits)
			}

			validationContext := Build(
				w,
				sourceindex.Index{},
				subject.Subject{Kind: "semantic-validation", ID: "sales.order.lines.line_number_unique"},
			)
			if validationContext.DataDesign != nil {
				t.Fatalf("ambiguous validation scope was resolved: %+v", validationContext.DataDesign)
			}
			if !containsDataDesignString(
				validationContext.Unresolved,
				"data-design-validation-scope-not-uniquely-resolvable",
			) {
				t.Fatalf("ambiguous validation scope reason is missing: %+v", validationContext.Unresolved)
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

func TestDataDesignContextKeepsCompositeDomainConstraintUnresolvedWithoutFirstClassEvidence(t *testing.T) {
	ctx := Build(
		dataDesignRelationUniqueWorld(),
		sourceindex.Index{},
		subject.Subject{Kind: "semantic-data", ID: "sales.order_line"},
	)
	if ctx.DataDesign == nil {
		t.Fatal("data design context is missing")
	}
	if ctx.DataDesign.CompositeDomainConstraints.Nature != "unresolved" ||
		ctx.DataDesign.CompositeDomainConstraints.Reason != "composite-domain-constraint-not-first-class-in-current-canonical-model" {
		t.Fatalf("composite domain constraint was invented or left unexplained: %+v", ctx.DataDesign.CompositeDomainConstraints)
	}
	for _, question := range []string{
		"which-value-combinations-are-allowed-or-forbidden",
		"is-this-constraint-about-identity-functional-dependency-or-domain-membership",
		"which-source-or-contract-establishes-the-allowed-combinations",
	} {
		if !containsDataDesignString(ctx.DataDesign.CompositeDomainConstraintEvidenceQuestions, question) {
			t.Fatalf("composite domain constraint Evidence question %q is missing: %+v", question, ctx.DataDesign.CompositeDomainConstraintEvidenceQuestions)
		}
	}
	for _, limit := range []string{
		"absence-of-authored-composite-domain-constraint-is-not-no-composite-domain-constraint",
		"relation-shape-is-not-composite-domain-constraint-evidence",
		"scoped-uniqueness-is-not-composite-domain-constraint",
		"persistence-unique-constraint-is-not-composite-domain-constraint",
	} {
		if !containsDataDesignString(ctx.DataDesign.AuthorityLimits, limit) {
			t.Fatalf("composite domain constraint authority limit %q is missing: %+v", limit, ctx.DataDesign.AuthorityLimits)
		}
	}
}

func TestDataDesignContextKeepsTemporalHistoryUnresolvedFromDatetimeAttribute(t *testing.T) {
	w := dataDesignRelationUniqueWorld()
	w.Semantics[0].Data[1].Attributes = append(
		w.Semantics[0].Data[1].Attributes,
		semantic.Attribute{
			ID:      "sales.order_line.observed_at",
			Symbol:  "observedAt",
			TypeRef: "std.datetime",
		},
	)
	ctx := Build(
		w,
		sourceindex.Index{},
		subject.Subject{Kind: "semantic-data", ID: "sales.order_line"},
	)
	if ctx.DataDesign == nil {
		t.Fatal("data design context is missing")
	}
	if ctx.DataDesign.TemporalHistoryConditions.Nature != "unresolved" ||
		ctx.DataDesign.TemporalHistoryConditions.Reason != "temporal-history-condition-not-first-class-in-current-canonical-model" {
		t.Fatalf("datetime attribute was promoted to temporal semantics: %+v", ctx.DataDesign.TemporalHistoryConditions)
	}
	for _, question := range []string{
		"when-is-the-business-occurrence-or-identity-valid",
		"can-the-business-identifier-be-reused-across-time",
		"does-the-history-record-represent-a-version-of-the-same-occurrence-or-a-new-occurrence",
		"may-validity-periods-overlap",
		"is-current-state-derived-from-validity-or-marked-by-a-latest-flag",
		"which-state-transitions-change-identity-membership-or-validity",
		"which-history-or-validity-period-must-be-preserved",
		"which-source-or-contract-establishes-the-temporal-semantics",
	} {
		if !containsDataDesignString(ctx.DataDesign.TemporalHistoryEvidenceQuestions, question) {
			t.Fatalf("temporal/history Evidence question %q is missing: %+v", question, ctx.DataDesign.TemporalHistoryEvidenceQuestions)
		}
	}
	for _, limit := range []string{
		"absence-of-authored-temporal-history-condition-is-not-timeless-semantics",
		"datetime-or-timestamp-attribute-is-not-temporal-semantics",
		"persistence-history-column-is-not-business-temporal-semantics",
		"effective-date-or-history-number-is-not-business-occurrence-identity-by-default",
		"latest-flag-is-not-business-identity-evidence",
		"history-row-identity-is-not-business-occurrence-identity-by-default",
		"temporal-history-model-is-opt-in-not-universal",
	} {
		if !containsDataDesignString(ctx.DataDesign.AuthorityLimits, limit) {
			t.Fatalf("temporal/history authority limit %q is missing: %+v", limit, ctx.DataDesign.AuthorityLimits)
		}
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

func TestDataDesignContextDoesNotPromoteCompositePhysicalUniqueToBusinessMeaning(t *testing.T) {
	w := dataDesignRelationUniqueWorld()
	w.Semantics[0].Data[1].Attributes = append(
		w.Semantics[0].Data[1].Attributes,
		semantic.Attribute{
			ID:      "sales.order_line.item_code",
			Symbol:  "itemCode",
			TypeRef: "std.code",
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
					Attribute:   "sales.order_line.line_number",
					Column:      "line_number",
					Storage:     persistence.StorageInt64,
					Nullability: persistence.Required,
				},
				{
					Attribute:   "sales.order_line.item_code",
					Column:      "item_code",
					Storage:     persistence.StorageText,
					Nullability: persistence.Required,
				},
			},
			UniqueConstraints: []persistence.UniqueConstraintMapping{
				{
					Name: "order_lines_line_item_uq",
					Attributes: []core.CanonicalID{
						"sales.order_line.line_number",
						"sales.order_line.item_code",
					},
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
		t.Fatalf("composite physical uniqueness realization is missing: %+v", ctx.DataDesign)
	}
	realization := ctx.DataDesign.PersistenceRealizations[0]
	if len(realization.Uniqueness) != 1 ||
		realization.Uniqueness[0].Kind != "unique-constraint" ||
		realization.Uniqueness[0].Name != "order_lines_line_item_uq" ||
		len(realization.Uniqueness[0].Attributes) != 2 ||
		realization.Uniqueness[0].Attributes[0] != "sales.order_line.line_number" ||
		realization.Uniqueness[0].Attributes[1] != "sales.order_line.item_code" ||
		realization.Uniqueness[0].Nature != "accepted-realization" {
		t.Fatalf("composite physical unique fact drifted: %+v", realization.Uniqueness)
	}
	if ctx.DataDesign.BusinessOccurrenceIdentifier.Nature != "unresolved" ||
		ctx.DataDesign.CandidateKeys.Nature != "unresolved" ||
		ctx.DataDesign.FunctionalDependencies.Nature != "unresolved" ||
		ctx.DataDesign.CompositeDomainConstraints.Nature != "unresolved" {
		t.Fatalf("composite physical unique was promoted to Semantic meaning: %+v", ctx.DataDesign)
	}
	if len(ctx.DataDesign.BusinessIdentityCandidates) != 1 ||
		ctx.DataDesign.BusinessIdentityCandidates[0].BasisConstraint != "sales.order.lines.line_number_unique" {
		t.Fatalf("physical composite unique replaced Semantic identity candidate: %+v", ctx.DataDesign.BusinessIdentityCandidates)
	}
}

func TestDataDesignContextSuppressesDuplicatePersistenceMappingIdentity(t *testing.T) {
	w := dataDesignRelationUniqueWorld()
	w.Persistence = []persistence.Mapping{
		{
			ID:           "sales_core.order_line",
			Database:     "sales_core",
			SemanticData: "sales.order_line",
			Table:        "public.order_lines",
			PrimaryKey:   []core.CanonicalID{"sales.order_line.line_number"},
		},
		{
			ID:           "sales_core.order_line",
			Database:     "archive",
			SemanticData: "sales.order_line",
			Table:        "archive.order_lines",
			PrimaryKey:   []core.CanonicalID{"sales.order_line.line_number"},
		},
	}

	dataContext := Build(
		w,
		sourceindex.Index{},
		subject.Subject{Kind: "semantic-data", ID: "sales.order_line"},
	)
	if dataContext.DataDesign == nil {
		t.Fatal("data design context is missing")
	}
	if len(dataContext.DataDesign.PersistenceRealizations) != 0 {
		t.Fatalf("ambiguous persistence identity produced realization facts: %+v", dataContext.DataDesign.PersistenceRealizations)
	}
	if dataContext.DataDesign.ResolutionStatus != "ambiguous" {
		t.Fatalf("ambiguous persistence context was marked resolved: %+v", dataContext.DataDesign)
	}
	if !containsDataDesignAmbiguityEffect(
		dataContext.DataDesign.Ambiguities,
		"duplicate-persistence-mapping",
		"sales_core.order_line",
		"realization-suppressed",
	) {
		t.Fatalf("persistence ambiguity was not exposed: %+v", dataContext.DataDesign.Ambiguities)
	}
	if !containsDataDesignString(
		dataContext.DataDesign.AuthorityLimits,
		"ambiguous-persistence-identity-does-not-produce-realization-facts",
	) {
		t.Fatalf("persistence ambiguity authority limit is missing: %+v", dataContext.DataDesign.AuthorityLimits)
	}

	persistenceContext := Build(
		w,
		sourceindex.Index{},
		subject.Subject{Kind: "persistence", ID: "sales_core.order_line"},
	)
	if persistenceContext.DataDesign != nil {
		t.Fatalf("duplicate persistence subject was resolved by read order: %+v", persistenceContext.DataDesign)
	}
	if !containsDataDesignString(
		persistenceContext.Unresolved,
		"data-design-persistence-subject-not-uniquely-resolvable",
	) {
		t.Fatalf("duplicate persistence reason is missing: %+v", persistenceContext.Unresolved)
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

func TestDataDesignContextFingerprintTracksRelevantEvidenceOnly(t *testing.T) {
	w := dataDesignRelationUniqueWorld()
	relevantPath := "semantic/sales/order.tei"
	relevantSubject := subject.Subject{Kind: "semantic-data", ID: "sales.order_line"}
	baseSources := sourceindex.Index{
		Artifacts: []sourceindex.Artifact{
			{
				Path:        relevantPath,
				Fingerprint: sourceindex.Fingerprint([]byte("relevant-v1")),
			},
		},
		References: []sourceindex.Reference{
			{
				Subject:      relevantSubject,
				ArtifactPath: relevantPath,
				Role:         sourceindex.RoleDefinition,
			},
		},
	}

	first := Build(w, baseSources, relevantSubject)
	if first.DataDesign == nil ||
		first.DataDesign.ProjectionRevision != dataDesignProjectionRevision ||
		first.DataDesign.Fingerprint == "" ||
		first.DataDesign.EvidenceCoverageStatus != "incomplete" {
		t.Fatalf("partial data design fingerprint boundary is missing: %+v", first.DataDesign)
	}

	withUnrelated := baseSources
	withUnrelated.Artifacts = append(
		append([]sourceindex.Artifact(nil), baseSources.Artifacts...),
		sourceindex.Artifact{
			Path:        "semantic/other/example.tei",
			Fingerprint: sourceindex.Fingerprint([]byte("unrelated")),
		},
	)
	withUnrelated.References = append(
		append([]sourceindex.Reference(nil), baseSources.References...),
		sourceindex.Reference{
			Subject:      subject.Subject{Kind: "semantic-data", ID: "other.example"},
			ArtifactPath: "semantic/other/example.tei",
			Role:         sourceindex.RoleDefinition,
		},
	)
	second := Build(w, withUnrelated, relevantSubject)
	if second.DataDesign == nil || second.DataDesign.Fingerprint != first.DataDesign.Fingerprint {
		t.Fatalf("unrelated source changed Data Design fingerprint: first=%q second=%q", first.DataDesign.Fingerprint, second.DataDesign.Fingerprint)
	}

	changedRelevant := baseSources
	changedRelevant.Artifacts = []sourceindex.Artifact{
		{
			Path:        relevantPath,
			Fingerprint: sourceindex.Fingerprint([]byte("relevant-v2")),
		},
	}
	third := Build(w, changedRelevant, relevantSubject)
	if third.DataDesign == nil || third.DataDesign.Fingerprint == first.DataDesign.Fingerprint {
		t.Fatalf("relevant source change did not invalidate Data Design fingerprint: first=%q third=%q", first.DataDesign.Fingerprint, third.DataDesign.Fingerprint)
	}

	changedProjection := dataDesignRelationUniqueWorld()
	changedProjection.Semantics[0].Data[0].Relations[0].Cardinality = semantic.CardinalityOne
	fourth := Build(changedProjection, baseSources, relevantSubject)
	if fourth.DataDesign == nil ||
		fourth.DataDesign.Fingerprint == first.DataDesign.Fingerprint {
		t.Fatalf(
			"derived projection change did not invalidate Data Design fingerprint: first=%q fourth=%q",
			first.DataDesign.Fingerprint,
			fourth.DataDesign.Fingerprint,
		)
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
	if ctx.DataDesign.EvidenceCoverageStatus != "incomplete" ||
		ctx.DataDesign.Fingerprint == "" {
		t.Fatalf("missing Evidence must remain explicitly incomplete: %+v", ctx.DataDesign)
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
	for _, needed := range []string{
		"history-version-identity",
		"identifier-reuse-policy",
	} {
		if !containsDataDesignString(identityCandidate.EvidenceNeeded, needed) {
			t.Fatalf("business identity temporal Evidence need %q is missing: %+v", needed, identityCandidate.EvidenceNeeded)
		}
	}
	for _, needed := range []string{
		"history-version-identity",
		"temporal-dependency-scope",
	} {
		if !containsDataDesignString(fdCandidate.EvidenceNeeded, needed) {
			t.Fatalf("FD temporal Evidence need %q is missing: %+v", needed, fdCandidate.EvidenceNeeded)
		}
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

func TestDerivedDataDesignCandidateFingerprintSeparatesMeaningFromEvidenceRequirements(t *testing.T) {
	ctx := Build(
		dataDesignRelationUniqueWorld(),
		sourceindex.Index{},
		subject.Subject{Kind: "semantic-data", ID: "sales.order_line"},
	)
	if ctx.DataDesign == nil ||
		len(ctx.DataDesign.BusinessIdentityCandidates) != 1 ||
		len(ctx.DataDesign.FunctionalDependencyCandidates) != 1 {
		t.Fatalf("derived candidates are missing: %+v", ctx.DataDesign)
	}

	identity := ctx.DataDesign.BusinessIdentityCandidates[0]
	fd := ctx.DataDesign.FunctionalDependencyCandidates[0]
	if identity.Fingerprint == "" ||
		identity.EvidenceRequirementFingerprint == "" ||
		fd.Fingerprint == "" ||
		fd.EvidenceRequirementFingerprint == "" {
		t.Fatalf("candidate fingerprints are missing: identity=%+v fd=%+v", identity, fd)
	}
	if identity.Fingerprint == fd.Fingerprint {
		t.Fatalf("typed candidate identities collided: %s", identity.Fingerprint)
	}

	reorderedIdentity := identity
	reorderedIdentity.Components = append([]DataDesignDeterminantTerm(nil), identity.Components...)
	reorderedIdentity.Components[0], reorderedIdentity.Components[1] =
		reorderedIdentity.Components[1], reorderedIdentity.Components[0]
	reorderedIdentity.EvidenceNeeded = append([]string(nil), identity.EvidenceNeeded...)
	for i, j := 0, len(reorderedIdentity.EvidenceNeeded)-1; i < j; i, j = i+1, j-1 {
		reorderedIdentity.EvidenceNeeded[i], reorderedIdentity.EvidenceNeeded[j] =
			reorderedIdentity.EvidenceNeeded[j], reorderedIdentity.EvidenceNeeded[i]
	}
	if got := businessIdentityCandidateFingerprint(reorderedIdentity); got != identity.Fingerprint {
		t.Fatalf("business identity candidate ordering changed identity: have=%s got=%s", identity.Fingerprint, got)
	}
	if got := dataDesignCandidateEvidenceRequirementFingerprint(
		"tei.knowledge.data-design.business-identity-candidate-evidence/v1",
		identity.Fingerprint,
		reorderedIdentity.EvidenceNeeded,
	); got != identity.EvidenceRequirementFingerprint {
		t.Fatalf("business identity Evidence ordering changed review contract: have=%s got=%s", identity.EvidenceRequirementFingerprint, got)
	}

	changedIdentityEvidence := identity
	changedIdentityEvidence.EvidenceNeeded = append(
		append([]string(nil), identity.EvidenceNeeded...),
		"additional-review-requirement",
	)
	if got := businessIdentityCandidateFingerprint(changedIdentityEvidence); got != identity.Fingerprint {
		t.Fatalf("Evidence requirement changed business identity candidate identity: have=%s got=%s", identity.Fingerprint, got)
	}
	if got := dataDesignCandidateEvidenceRequirementFingerprint(
		"tei.knowledge.data-design.business-identity-candidate-evidence/v1",
		identity.Fingerprint,
		changedIdentityEvidence.EvidenceNeeded,
	); got == identity.EvidenceRequirementFingerprint {
		t.Fatalf("changed business identity Evidence requirements kept review fingerprint: %s", got)
	}

	changedIdentity := identity
	changedIdentity.Scope = "global"
	changedIdentityFingerprint := businessIdentityCandidateFingerprint(changedIdentity)
	if changedIdentityFingerprint == identity.Fingerprint {
		t.Fatalf("business identity candidate semantic change kept fingerprint: %s", changedIdentityFingerprint)
	}
	if got := dataDesignCandidateEvidenceRequirementFingerprint(
		"tei.knowledge.data-design.business-identity-candidate-evidence/v1",
		changedIdentityFingerprint,
		changedIdentity.EvidenceNeeded,
	); got == identity.EvidenceRequirementFingerprint {
		t.Fatalf("business identity semantic change kept review fingerprint: %s", got)
	}

	reorderedFD := fd
	reorderedFD.Determinants = append([]DataDesignDeterminantTerm(nil), fd.Determinants...)
	reorderedFD.Determinants[0], reorderedFD.Determinants[1] =
		reorderedFD.Determinants[1], reorderedFD.Determinants[0]
	reorderedFD.EvidenceNeeded = append([]string(nil), fd.EvidenceNeeded...)
	for i, j := 0, len(reorderedFD.EvidenceNeeded)-1; i < j; i, j = i+1, j-1 {
		reorderedFD.EvidenceNeeded[i], reorderedFD.EvidenceNeeded[j] =
			reorderedFD.EvidenceNeeded[j], reorderedFD.EvidenceNeeded[i]
	}
	if got := functionalDependencyCandidateFingerprint(reorderedFD); got != fd.Fingerprint {
		t.Fatalf("FD candidate ordering changed identity: have=%s got=%s", fd.Fingerprint, got)
	}
	if got := dataDesignCandidateEvidenceRequirementFingerprint(
		"tei.knowledge.data-design.functional-dependency-candidate-evidence/v1",
		fd.Fingerprint,
		reorderedFD.EvidenceNeeded,
	); got != fd.EvidenceRequirementFingerprint {
		t.Fatalf("FD Evidence ordering changed review contract: have=%s got=%s", fd.EvidenceRequirementFingerprint, got)
	}

	changedFDEvidence := fd
	changedFDEvidence.EvidenceNeeded = append(
		append([]string(nil), fd.EvidenceNeeded...),
		"additional-review-requirement",
	)
	if got := functionalDependencyCandidateFingerprint(changedFDEvidence); got != fd.Fingerprint {
		t.Fatalf("Evidence requirement changed FD candidate identity: have=%s got=%s", fd.Fingerprint, got)
	}
	if got := dataDesignCandidateEvidenceRequirementFingerprint(
		"tei.knowledge.data-design.functional-dependency-candidate-evidence/v1",
		fd.Fingerprint,
		changedFDEvidence.EvidenceNeeded,
	); got == fd.EvidenceRequirementFingerprint {
		t.Fatalf("changed FD Evidence requirements kept review fingerprint: %s", got)
	}

	changedFD := fd
	changedFD.DependentOccurrence = "sales.other"
	changedFDFingerprint := functionalDependencyCandidateFingerprint(changedFD)
	if changedFDFingerprint == fd.Fingerprint {
		t.Fatalf("FD candidate semantic change kept fingerprint: %s", changedFDFingerprint)
	}
	if got := dataDesignCandidateEvidenceRequirementFingerprint(
		"tei.knowledge.data-design.functional-dependency-candidate-evidence/v1",
		changedFDFingerprint,
		changedFD.EvidenceNeeded,
	); got == fd.EvidenceRequirementFingerprint {
		t.Fatalf("FD semantic change kept review fingerprint: %s", got)
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

func containsDataDesignAmbiguity(values []DataDesignAmbiguity, kind string, id core.CanonicalID) bool {
	return containsDataDesignAmbiguityEffect(values, kind, id, "design-candidates-suppressed")
}

func containsDataDesignAmbiguityEffect(
	values []DataDesignAmbiguity,
	kind string,
	id core.CanonicalID,
	effect string,
) bool {
	for _, value := range values {
		if value.Kind == kind && value.ID == id && value.Count > 1 && value.Effect == effect {
			return true
		}
	}
	return false
}

func containsDataDesignString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func containsDataDesignSubject(values []subject.Subject, want subject.Subject) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
