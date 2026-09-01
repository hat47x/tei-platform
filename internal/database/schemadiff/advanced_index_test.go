package schemadiff_test

import (
	"testing"

	"tei/internal/core"
	"tei/internal/database/physical"
	"tei/internal/database/schemadiff"
)

func indexSchema(indexes ...physical.Index) physical.SchemaSnapshot {
	return physical.SchemaSnapshot{
		Database: core.MustCanonicalID("sales_core"),
		Tables:   []physical.Table{{Schema: "public", Name: "customers", Kind: physical.TableKindTable, Indexes: indexes}},
	}.Normalize()
}

func simpleIndex(name string) physical.Index {
	return physical.Index{
		Name:   name,
		Method: "btree",
		Keys:   []physical.IndexKey{physical.ColumnIndexKey("email")},
	}
}

func postgresExpr(source string) *physical.ProviderExpression {
	return &physical.ProviderExpression{Provider: "postgres", Dialect: "postgresql", Source: source}
}

func TestCompareIndexIgnoresUsableActualState(t *testing.T) {
	desired := simpleIndex("customers_email_idx")
	actual := desired
	actual.State = &physical.IndexState{Valid: true, Ready: true, Live: true}

	diff, err := schemadiff.Compare(indexSchema(desired), indexSchema(actual))
	if err != nil {
		t.Fatal(err)
	}
	if len(diff.Changes) != 0 {
		t.Fatalf("usable actual state produced drift: %#v", diff.Changes)
	}
}

func TestCompareIndexReportsLifecycleDriftSeparately(t *testing.T) {
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
			desired := simpleIndex("customers_email_idx")
			actual := desired
			actual.State = &tt.state

			diff, err := schemadiff.Compare(indexSchema(desired), indexSchema(actual))
			if err != nil {
				t.Fatal(err)
			}
			if len(diff.Changes) != 1 || diff.Changes[0].Kind != schemadiff.IndexStateDrift || diff.Changes[0].Index == nil {
				t.Fatalf("changes=%#v", diff.Changes)
			}
			if diff.Changes[0].Index.State == nil || diff.Changes[0].Index.State.Usable() {
				t.Fatalf("state finding=%#v", diff.Changes[0].Index)
			}
		})
	}
}

func TestCompareIndexTreatsAdvancedStructureAsMeaningful(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*physical.Index)
	}{
		{name: "expression", mutate: func(i *physical.Index) {
			i.Keys[0] = physical.IndexKey{Expression: postgresExpr("lower(email)"), Order: physical.IndexOrderDefault, NullsOrder: physical.IndexNullsOrderDefault}
		}},
		{name: "include", mutate: func(i *physical.Index) { i.IncludedColumns = []string{"display_name"} }},
		{name: "predicate", mutate: func(i *physical.Index) { i.Predicate = postgresExpr("deleted_at IS NULL") }},
		{name: "sort", mutate: func(i *physical.Index) { i.Keys[0].Order = physical.IndexOrderDesc }},
		{name: "null order", mutate: func(i *physical.Index) { i.Keys[0].NullsOrder = physical.IndexNullsOrderFirst }},
		{name: "collation", mutate: func(i *physical.Index) { i.Keys[0].Collation = "pg_catalog.C" }},
		{name: "operator class", mutate: func(i *physical.Index) { i.Keys[0].OperatorClass = "pg_catalog.text_pattern_ops" }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			desired := simpleIndex("customers_email_idx")
			actual := desired
			actual.Keys = append([]physical.IndexKey(nil), desired.Keys...)
			tt.mutate(&actual)
			diff, err := schemadiff.Compare(indexSchema(desired), indexSchema(actual))
			if err != nil {
				t.Fatal(err)
			}
			if len(diff.Changes) != 2 || diff.Changes[0].Kind != schemadiff.DropIndex || diff.Changes[1].Kind != schemadiff.AddIndex {
				t.Fatalf("changes=%#v", diff.Changes)
			}
		})
	}
}

func TestCompareIndexPreservesKeyOrder(t *testing.T) {
	desired := simpleIndex("customers_identity_idx")
	desired.Keys = []physical.IndexKey{physical.ColumnIndexKey("tenant_id"), physical.ColumnIndexKey("external_id")}
	actual := desired
	actual.Keys = []physical.IndexKey{physical.ColumnIndexKey("external_id"), physical.ColumnIndexKey("tenant_id")}

	diff, err := schemadiff.Compare(indexSchema(desired), indexSchema(actual))
	if err != nil {
		t.Fatal(err)
	}
	if len(diff.Changes) != 2 || diff.Changes[0].Kind != schemadiff.DropIndex || diff.Changes[1].Kind != schemadiff.AddIndex {
		t.Fatalf("changes=%#v", diff.Changes)
	}
}

func TestCompareIndexTreatsUniqueNullTreatmentAsStructural(t *testing.T) {
	desired := simpleIndex("customers_email_uidx")
	desired.Unique = true
	desired.UniqueNullTreatment = physical.UniqueNullTreatmentDistinct
	actual := desired
	actual.UniqueNullTreatment = physical.UniqueNullTreatmentNotDistinct

	diff, err := schemadiff.Compare(indexSchema(desired), indexSchema(actual))
	if err != nil {
		t.Fatal(err)
	}
	if len(diff.Changes) != 2 || diff.Changes[0].Kind != schemadiff.DropIndex || diff.Changes[1].Kind != schemadiff.AddIndex {
		t.Fatalf("changes=%#v", diff.Changes)
	}
}
