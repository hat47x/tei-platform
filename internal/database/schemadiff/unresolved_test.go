package schemadiff_test

import (
	"testing"

	"tei/internal/core"
	"tei/internal/database/physical"
	"tei/internal/database/schemadiff"
)

func TestProviderExpressionTextDifferenceIsUnresolved(t *testing.T) {
	desired := indexSnapshot(expressionIndex("lower(name)"))
	actualIndex := expressionIndex("lower((name)::text)")
	actualIndex.State = &physical.IndexState{Valid: true, Ready: true, Live: true}
	actual := indexSnapshot(actualIndex)

	diff, err := schemadiff.Compare(desired, actual)
	if err != nil {
		t.Fatal(err)
	}
	if len(diff.Changes) != 0 {
		t.Fatalf("text-only expression difference became structural change: %#v", diff.Changes)
	}
	if len(diff.Unresolved) != 1 {
		t.Fatalf("unresolved=%#v", diff.Unresolved)
	}
	u := diff.Unresolved[0]
	if u.Kind != schemadiff.UnresolvedProviderExpression || u.Reason != schemadiff.UnresolvedNormalizationRequired {
		t.Fatalf("unresolved kind/reason=%#v", u)
	}
	if u.Table != "public.customers" || u.ObjectKind != "index" || u.ObjectName != "customers_lower_name_idx" || u.FieldPath != "index.keys[0].expression" {
		t.Fatalf("unresolved locator=%#v", u)
	}
	if u.DesiredExpression == nil || u.DesiredExpression.Source != "lower(name)" || u.ActualExpression == nil || u.ActualExpression.Source != "lower((name)::text)" {
		t.Fatalf("unresolved expressions=%#v", u)
	}
}

func TestPredicateTextDifferenceIsUnresolved(t *testing.T) {
	desiredIndex := columnIndex()
	desiredIndex.Predicate = expression("profile IS NOT NULL")
	actualIndex := columnIndex()
	actualIndex.Predicate = expression("(profile IS NOT NULL)")
	actualIndex.State = &physical.IndexState{Valid: true, Ready: true, Live: true}

	diff, err := schemadiff.Compare(indexSnapshot(desiredIndex), indexSnapshot(actualIndex))
	if err != nil {
		t.Fatal(err)
	}
	if len(diff.Changes) != 0 || len(diff.Unresolved) != 1 || diff.Unresolved[0].FieldPath != "index.predicate" {
		t.Fatalf("diff=%#v", diff)
	}
}

func TestProviderOrDialectDifferenceRemainsStructural(t *testing.T) {
	desired := expressionIndex("lower(name)")
	actual := expressionIndex("lower(name)")
	actual.Keys[0].Expression.Provider = "oracle"
	actual.Keys[0].Expression.Dialect = "oracle"
	actual.State = &physical.IndexState{Valid: true, Ready: true, Live: true}

	diff, err := schemadiff.Compare(indexSnapshot(desired), indexSnapshot(actual))
	if err != nil {
		t.Fatal(err)
	}
	if len(diff.Unresolved) != 0 || len(diff.Changes) != 2 || diff.Changes[0].Kind != schemadiff.DropIndex || diff.Changes[1].Kind != schemadiff.AddIndex {
		t.Fatalf("provider difference=%#v", diff)
	}
}

func TestKnownStructuralDifferenceTakesPriorityOverExpressionUnresolved(t *testing.T) {
	desired := expressionIndex("lower(name)")
	desired.IncludedColumns = []string{"external_id"}
	actual := expressionIndex("lower((name)::text)")
	actual.State = &physical.IndexState{Valid: true, Ready: true, Live: true}

	diff, err := schemadiff.Compare(indexSnapshot(desired), indexSnapshot(actual))
	if err != nil {
		t.Fatal(err)
	}
	if len(diff.Unresolved) != 0 || len(diff.Changes) != 2 || diff.Changes[0].Kind != schemadiff.DropIndex || diff.Changes[1].Kind != schemadiff.AddIndex {
		t.Fatalf("known structural difference=%#v", diff)
	}
}

func TestIndexStateDriftSurvivesUnresolvedStructureComparison(t *testing.T) {
	desired := expressionIndex("lower(name)")
	actual := expressionIndex("lower((name)::text)")
	actual.State = &physical.IndexState{Valid: false, Ready: false, Live: true}

	diff, err := schemadiff.Compare(indexSnapshot(desired), indexSnapshot(actual))
	if err != nil {
		t.Fatal(err)
	}
	if len(diff.Unresolved) != 1 || len(diff.Changes) != 1 || diff.Changes[0].Kind != schemadiff.IndexStateDrift {
		t.Fatalf("state/unresolved=%#v", diff)
	}
}

func expressionIndex(source string) physical.Index {
	return physical.Index{
		Name:   "customers_lower_name_idx",
		Method: "btree",
		Keys: []physical.IndexKey{{
			Expression: expression(source),
			Order:      physical.IndexOrderDefault,
			NullsOrder: physical.IndexNullsOrderDefault,
		}},
	}
}

func columnIndex() physical.Index {
	return physical.Index{
		Name:   "customers_lower_name_idx",
		Method: "btree",
		Keys:   []physical.IndexKey{physical.ColumnIndexKey("name")},
	}
}

func expression(source string) *physical.ProviderExpression {
	return &physical.ProviderExpression{Provider: "postgres", Dialect: "postgresql", Source: source}
}

func indexSnapshot(index physical.Index) physical.SchemaSnapshot {
	return physical.SchemaSnapshot{
		Database: core.MustCanonicalID("sales_core"),
		Tables: []physical.Table{{
			Schema:  "public",
			Name:    "customers",
			Kind:    physical.TableKindTable,
			Indexes: []physical.Index{index},
		}},
	}
}
