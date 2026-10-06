package xeadsource

import (
	"reflect"
	"testing"
)

func TestDiagnoseRelationalSourceFactsAcceptsExplicitResolvableKeyReferences(t *testing.T) {
	doc, err := Parse(FormatXead, []byte(`<System><Table ID="T1"><TableField ID="A"/><TableKey ID="K1" Type="PK"><TableKeyField FieldID="A"/></TableKey></Table><Table ID="T2"><TableField ID="B"/><TableKey ID="K2" Type="PK"><TableKeyField FieldID="B"/></TableKey></Table><Relationship ID="R1" Table1ID="T1" TableKey1ID="K1" Table2ID="T2" TableKey2ID="K2"/></System>`))
	if err != nil {
		t.Fatal(err)
	}
	facts := ExtractRelationalSourceFacts(doc)
	if got := DiagnoseRelationalSourceFacts(doc, facts); len(got) != 0 {
		t.Fatalf("resolvable relation produced diagnostics: %#v", got)
	}
}

func TestDiagnoseRelationalSourceFactsFailsClosedOnMissingReferences(t *testing.T) {
	doc, err := Parse(FormatXead, []byte(`<System><Table ID="T1"><TableField ID="A"/><TableKey ID="K1" Type="PK"><TableKeyField FieldID="MISSING"/></TableKey></Table><Relationship ID="R1" Table1ID="T1" TableKey1ID="UNKNOWN" Table2ID="T2" TableKey2ID="K2"/></System>`))
	if err != nil {
		t.Fatal(err)
	}
	got := DiagnoseRelationalSourceFacts(doc, ExtractRelationalSourceFacts(doc))
	codes := make([]string, 0, len(got))
	for _, diagnostic := range got {
		if diagnostic.Nature != RelationalDiagnosticNature {
			t.Fatalf("diagnostic gained non-unresolved nature: %#v", diagnostic)
		}
		codes = append(codes, diagnostic.Code)
	}
	if !reflect.DeepEqual(codes, []string{
		"key-field-not-found",
		"relationship-key-not-found",
		"relationship-table-not-found",
	}) {
		t.Fatalf("unexpected diagnostics: %#v", got)
	}
}

func TestDiagnoseRelationalSourceFactsReportsCompositeKeyArityMismatchWithoutRepair(t *testing.T) {
	doc, err := Parse(FormatXead, []byte(`<System><Table ID="T1"><TableField ID="A"/><TableField ID="B"/><TableKey ID="K1" Type="PK"><TableKeyField FieldID="A"/><TableKeyField FieldID="B"/></TableKey></Table><Table ID="T2"><TableField ID="C"/><TableKey ID="K2" Type="PK"><TableKeyField FieldID="C"/></TableKey></Table><Relationship ID="R1" Table1ID="T1" TableKey1ID="K1" Table2ID="T2" TableKey2ID="K2"/></System>`))
	if err != nil {
		t.Fatal(err)
	}
	got := DiagnoseRelationalSourceFacts(doc, ExtractRelationalSourceFacts(doc))
	if len(got) != 1 ||
		got[0].Code != "relationship-key-arity-mismatch" ||
		got[0].RelationshipID != "R1" ||
		got[0].LeftCount != 2 ||
		got[0].RightCount != 1 {
		t.Fatalf("arity mismatch was not preserved as unresolved: %#v", got)
	}
}
