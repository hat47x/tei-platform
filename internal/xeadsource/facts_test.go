package xeadsource

import (
	"reflect"
	"testing"
)

func TestExtractRelationalSourceFactsPreservesLegacyKeyAndRelationshipReferences(t *testing.T) {
	doc, err := Parse(FormatXead, []byte(`<System><Table ID="T1"><TableField ID="F1" AttributeType="DERIVABLE" DataTypeID="DT1"/><TableField ID="F2" AttributeType="NATIVE" DataTypeID="DT2"/><TableKey ID="K1" Type="PK"><TableKeyField FieldID="F1" SortKey="10"/><TableKeyField FieldID="F2" SortKey="20"/></TableKey></Table><Relationship ID="R1" Table1ID="T1" TableKey1ID="K1" Table2ID="T2" TableKey2ID="K2"/></System>`))
	if err != nil {
		t.Fatal(err)
	}
	got := ExtractRelationalSourceFacts(doc)
	if len(got.LegacyKeys) != 1 {
		t.Fatalf("legacy key count = %d: %#v", len(got.LegacyKeys), got)
	}
	key := got.LegacyKeys[0]
	if key.TableLegacyID != "T1" || key.KeyLegacyID != "K1" || key.KeyType != "PK" || key.Nature != SourceFactNature {
		t.Fatalf("legacy key source fact drifted: %#v", key)
	}
	if !reflect.DeepEqual(key.Members, []LegacyKeyMemberFact{
		{Locator: "/System[1]/Table[1]/TableKey[3]/TableKeyField[1]", FieldLegacyID: "F1", AttributeType: "DERIVABLE", DataTypeLegacyID: "DT1", SortKey: "10", SourceOrdinal: 0},
		{Locator: "/System[1]/Table[1]/TableKey[3]/TableKeyField[2]", FieldLegacyID: "F2", AttributeType: "NATIVE", DataTypeLegacyID: "DT2", SortKey: "20", SourceOrdinal: 1},
	}) {
		t.Fatalf("legacy composite key membership drifted: %#v", key.Members)
	}
	if !reflect.DeepEqual(key.AuthorityLimits, legacyKeyAuthorityLimits) {
		t.Fatalf("legacy key authority limits drifted: %#v", key.AuthorityLimits)
	}

	if len(got.RelationshipKeys) != 1 {
		t.Fatalf("relationship key count = %d: %#v", len(got.RelationshipKeys), got)
	}
	relation := got.RelationshipKeys[0]
	if relation.RelationshipID != "R1" ||
		relation.Table1LegacyID != "T1" ||
		relation.Key1LegacyID != "K1" ||
		relation.Table2LegacyID != "T2" ||
		relation.Key2LegacyID != "K2" ||
		relation.Nature != SourceFactNature {
		t.Fatalf("relationship key source fact drifted: %#v", relation)
	}
	if !reflect.DeepEqual(relation.AuthorityLimits, relationshipKeyAuthorityLimits) {
		t.Fatalf("relationship authority limits drifted: %#v", relation.AuthorityLimits)
	}
}

func TestExtractRelationalSourceFactsKeepsDeclaredKeyFieldsSeparateFromDataIdentity(t *testing.T) {
	doc, err := Parse(FormatXeaf, []byte(`<System><Function ID="F1" KeyFields="A; B;"><Detail ID="D1" KeyFields="H;L;"/></Function></System>`))
	if err != nil {
		t.Fatal(err)
	}
	got := ExtractRelationalSourceFacts(doc)
	if len(got.DeclaredKeyFields) != 2 {
		t.Fatalf("declared key field count = %d: %#v", len(got.DeclaredKeyFields), got)
	}
	want := []DeclaredKeyFieldsFact{
		{
			Locator:         "/System[1]/Function[1]",
			SourceName:      "Function",
			OwnerLegacyID:   "F1",
			Fields:          []string{"A", "B"},
			Nature:          SourceFactNature,
			AuthorityLimits: declaredKeyFieldsAuthorityLimits,
		},
		{
			Locator:         "/System[1]/Function[1]/Detail[1]",
			SourceName:      "Detail",
			OwnerLegacyID:   "D1",
			Fields:          []string{"H", "L"},
			Nature:          SourceFactNature,
			AuthorityLimits: declaredKeyFieldsAuthorityLimits,
		},
	}
	if !reflect.DeepEqual(got.DeclaredKeyFields, want) {
		t.Fatalf("declared key fields drifted: got=%#v want=%#v", got.DeclaredKeyFields, want)
	}
}

func TestExtractRelationalSourceFactsPreservesXeafRuntimeKeyAndReferMappings(t *testing.T) {
	doc, err := Parse(FormatXeaf, []byte(`<System><Table ID="T1"><Key ID="K1" Type="PK" Fields="A;B;"/><Refer ID="R1" ToTable="T2" TableAlias="REF" ToKeyFields="C;D;" WithKeyFields="A;B;"/></Table></System>`))
	if err != nil {
		t.Fatal(err)
	}
	got := ExtractRelationalSourceFacts(doc)
	if len(got.RuntimeKeys) != 1 {
		t.Fatalf("runtime key count = %d: %#v", len(got.RuntimeKeys), got)
	}
	key := got.RuntimeKeys[0]
	if key.TableLegacyID != "T1" ||
		key.KeyLegacyID != "K1" ||
		key.KeyType != "PK" ||
		!reflect.DeepEqual(key.Fields, []string{"A", "B"}) ||
		key.Nature != SourceFactNature ||
		!reflect.DeepEqual(key.AuthorityLimits, runtimeKeyAuthorityLimits) {
		t.Fatalf("runtime key fact drifted: %#v", key)
	}

	if len(got.RuntimeRefers) != 1 {
		t.Fatalf("runtime refer count = %d: %#v", len(got.RuntimeRefers), got)
	}
	refer := got.RuntimeRefers[0]
	if refer.ReferLegacyID != "R1" ||
		refer.SourceTableLegacyID != "T1" ||
		refer.ToTableLegacyID != "T2" ||
		refer.TableAlias != "REF" ||
		!reflect.DeepEqual(refer.ToKeyFields, []string{"C", "D"}) ||
		!reflect.DeepEqual(refer.WithKeyFields, []string{"A", "B"}) ||
		refer.Nature != SourceFactNature ||
		!reflect.DeepEqual(refer.AuthorityLimits, runtimeReferAuthorityLimits) {
		t.Fatalf("runtime refer fact drifted: %#v", refer)
	}
}

func TestExtractRelationalSourceFactsDoesNotInventFactsFromNamesOrTypes(t *testing.T) {
	doc, err := Parse(FormatXead, []byte(`<System><Table ID="ORDER"><TableField ID="ORDER_ID" Name="Primary Key"/></Table></System>`))
	if err != nil {
		t.Fatal(err)
	}
	got := ExtractRelationalSourceFacts(doc)
	if len(got.LegacyKeys) != 0 || len(got.RelationshipKeys) != 0 || len(got.DeclaredKeyFields) != 0 {
		t.Fatalf("key facts were inferred from names: %#v", got)
	}
}
