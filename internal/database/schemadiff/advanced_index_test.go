package schemadiff_test

import (
	"testing"

	"tei/internal/core"
	"tei/internal/database/physical"
	"tei/internal/database/schemadiff"
)

func TestAdvancedIndexStructuralEqualityIgnoresUsableActualState(t *testing.T) {
	desiredIndex := advancedIndex()
	actualIndex := advancedIndex()
	actualIndex.State = &physical.IndexState{Valid: true, Ready: true, Live: true}

	diff := compareIndexes(t, desiredIndex, actualIndex)
	if len(diff.Changes) != 0 {
		t.Fatalf("unexpected diff: %#v", diff)
	}
}

func TestAdvancedIndexStructuralDriftRequiresReplacement(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*physical.Index)
	}{
		{name: "predicate", mutate: func(i *physical.Index) { i.Predicate.Source = "score >= 10" }},
		{name: "expression", mutate: func(i *physical.Index) { i.Keys[1].Expression.Source = "upper(name)" }},
		{name: "expression-provider", mutate: func(i *physical.Index) { i.Keys[1].Expression.Provider = "other-provider" }},
		{name: "include", mutate: func(i *physical.Index) { i.IncludedColumns = []string{"payload", "tag"} }},
		{name: "include-order", mutate: func(i *physical.Index) {
			i.IncludedColumns = []string{"payload", "tag"}
			i.IncludedColumns[0], i.IncludedColumns[1] = i.IncludedColumns[1], i.IncludedColumns[0]
		}},
		{name: "order", mutate: func(i *physical.Index) { i.Keys[0].Order = physical.IndexOrderAsc }},
		{name: "nulls-order", mutate: func(i *physical.Index) { i.Keys[0].NullsOrder = physical.IndexNullsFirst }},
		{name: "collation", mutate: func(i *physical.Index) { i.Keys[1].Collation = "pg_catalog.C" }},
		{name: "operator-class", mutate: func(i *physical.Index) { i.Keys[0].OperatorClass = "pg_catalog.int4_ops" }},
		{name: "unique-null-treatment", mutate: func(i *physical.Index) { i.UniqueNullTreatment = physical.UniqueNullsDistinct }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			desiredIndex := advancedIndex()
			actualIndex := advancedIndex()
			actualIndex.State = &physical.IndexState{Valid: true, Ready: true, Live: true}
			tt.mutate(&actualIndex)

			diff := compareIndexes(t, desiredIndex, actualIndex)
			if len(diff.Changes) != 2 || diff.Changes[0].Kind != schemadiff.DropIndex || diff.Changes[1].Kind != schemadiff.AddIndex {
				t.Fatalf("structural drift must be drop+add, got %#v", diff)
			}
			for _, change := range diff.Changes {
				if change.Kind == schemadiff.IndexStateDrift {
					t.Fatalf("structural drift was mislabeled as lifecycle drift: %#v", diff)
				}
			}
		})
	}
}

func TestAdvancedIndexLifecycleStateDriftDoesNotRequestReplacement(t *testing.T) {
	tests := []struct {
		name  string
		state physical.IndexState
	}{
		{name: "invalid", state: physical.IndexState{Valid: false, Ready: true, Live: true}},
		{name: "not-ready", state: physical.IndexState{Valid: true, Ready: false, Live: true}},
		{name: "not-live", state: physical.IndexState{Valid: true, Ready: true, Live: false}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			desiredIndex := advancedIndex()
			actualIndex := advancedIndex()
			actualIndex.State = &tt.state

			diff := compareIndexes(t, desiredIndex, actualIndex)
			if len(diff.Changes) != 1 || diff.Changes[0].Kind != schemadiff.IndexStateDrift {
				t.Fatalf("lifecycle drift must be one index-state-drift, got %#v", diff)
			}
			if diff.Changes[0].Index == nil || diff.Changes[0].Index.State == nil || diff.Changes[0].Index.State.Usable() {
				t.Fatalf("lifecycle observation was not preserved: %#v", diff.Changes[0])
			}
		})
	}
}

func advancedIndex() physical.Index {
	return physical.Index{
		Name: "orders_search_idx",
		Keys: []physical.IndexKey{
			{Column: "score", Order: physical.IndexOrderDesc, NullsOrder: physical.IndexNullsLast},
			{Expression: &physical.ProviderExpression{Provider: "postgres", Source: "lower(name)"}, Collation: "public.case_insensitive", OperatorClass: "public.text_pattern_ops", Order: physical.IndexOrderAsc, NullsOrder: physical.IndexNullsLast},
		},
		IncludedColumns:     []string{"tag", "payload"},
		Predicate:           &physical.ProviderExpression{Provider: "postgres", Source: "score > 0"},
		Unique:              true,
		UniqueNullTreatment: physical.UniqueNullsNotDistinct,
		Method:              "btree",
	}
}

func compareIndexes(t *testing.T, desiredIndex, actualIndex physical.Index) schemadiff.Diff {
	t.Helper()
	database := core.MustCanonicalID("advanced_index_test")
	desired := physical.SchemaSnapshot{Database: database, Tables: []physical.Table{{Schema: "public", Name: "orders", Kind: physical.TableKindTable, Indexes: []physical.Index{desiredIndex}}}}
	actual := physical.SchemaSnapshot{Database: database, Tables: []physical.Table{{Schema: "public", Name: "orders", Kind: physical.TableKindTable, Indexes: []physical.Index{actualIndex}}}}
	diff, err := schemadiff.Compare(desired.Normalize(), actual.Normalize())
	if err != nil {
		t.Fatal(err)
	}
	return diff
}
