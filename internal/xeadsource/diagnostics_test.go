package xeadsource

import "testing"

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
	codes := map[string]bool{}
	for _, diagnostic := range got {
		if diagnostic.Nature != RelationalDiagnosticNature {
			t.Fatalf("diagnostic gained non-unresolved nature: %#v", diagnostic)
		}
		codes[diagnostic.Code] = true
	}
	for _, want := range []string{
		"key-field-not-found",
		"relationship-key-not-found",
		"relationship-table-not-found",
	} {
		if !codes[want] {
			t.Fatalf("diagnostic %q is missing: %#v", want, got)
		}
	}
	if len(got) != 3 {
		t.Fatalf("unexpected extra diagnostics: %#v", got)
	}
}

func TestDiagnoseRelationalSourceFactsChecksExplicitXeafReferMappings(t *testing.T) {
	doc, err := Parse(FormatXeaf, []byte(`<System><Table ID="T1"><Field ID="A"/><Field ID="B"/><Key ID="K1" Type="PK" Fields="A;B;"/><Refer ID="R1" ToTable="T2" ToKeyFields="C;D;" WithKeyFields="A;B;"/></Table><Table ID="T2"><Field ID="C"/><Field ID="D"/><Key ID="K2" Type="PK" Fields="C;D;"/></Table></System>`))
	if err != nil {
		t.Fatal(err)
	}
	if got := DiagnoseRelationalSourceFacts(doc, ExtractRelationalSourceFacts(doc)); len(got) != 0 {
		t.Fatalf("explicit runtime mapping produced diagnostics: %#v", got)
	}
}

func TestDiagnoseRelationalSourceFactsFailsClosedOnBrokenXeafReferMapping(t *testing.T) {
	doc, err := Parse(FormatXeaf, []byte(`<System><Table ID="T1"><Field ID="A"/><Key Type="PK" Fields="MISSING;"/><Refer ID="R1" ToTable="T2" ToKeyFields="C;D;" WithKeyFields="A;"/></Table><Table ID="T2"><Field ID="C"/></Table></System>`))
	if err != nil {
		t.Fatal(err)
	}
	got := DiagnoseRelationalSourceFacts(doc, ExtractRelationalSourceFacts(doc))
	codes := map[string]bool{}
	for _, diagnostic := range got {
		codes[diagnostic.Code] = true
	}
	for _, want := range []string{
		"runtime-key-field-not-found",
		"runtime-refer-target-field-not-found",
		"runtime-refer-key-arity-mismatch",
	} {
		if !codes[want] {
			t.Fatalf("runtime diagnostic %q is missing: %#v", want, got)
		}
	}
}

func TestDiagnoseRelationalSourceFactsUsesSortKeyOrderForCompositeRelationship(t *testing.T) {
	doc, err := Parse(FormatXead, []byte(`<System><Table ID="T1"><TableField ID="A" DataTypeID="D2"/><TableField ID="B" DataTypeID="D1"/><TableKey ID="K1" Type="PK"><TableKeyField FieldID="A" SortKey="20"/><TableKeyField FieldID="B" SortKey="10"/></TableKey></Table><Table ID="T2"><TableField ID="C" DataTypeID="D1"/><TableField ID="D" DataTypeID="D2"/><TableKey ID="K2" Type="PK"><TableKeyField FieldID="C" SortKey="10"/><TableKeyField FieldID="D" SortKey="20"/></TableKey></Table><Relationship ID="R1" Table1ID="T1" TableKey1ID="K1" Table2ID="T2" TableKey2ID="K2"/></System>`))
	if err != nil {
		t.Fatal(err)
	}
	if got := DiagnoseRelationalSourceFacts(doc, ExtractRelationalSourceFacts(doc)); len(got) != 0 {
		t.Fatalf("SortKey-aligned relationship produced diagnostics: %#v", got)
	}
}

func TestDiagnoseRelationalSourceFactsReportsCompositeRelationshipDataTypeMismatch(t *testing.T) {
	doc, err := Parse(FormatXead, []byte(`<System><Table ID="T1"><TableField ID="A" DataTypeID="D1"/><TableField ID="B" DataTypeID="D2"/><TableKey ID="K1" Type="PK"><TableKeyField FieldID="A" SortKey="10"/><TableKeyField FieldID="B" SortKey="20"/></TableKey></Table><Table ID="T2"><TableField ID="C" DataTypeID="D1"/><TableField ID="D" DataTypeID="OTHER"/><TableKey ID="K2" Type="PK"><TableKeyField FieldID="C" SortKey="10"/><TableKeyField FieldID="D" SortKey="20"/></TableKey></Table><Relationship ID="R1" Table1ID="T1" TableKey1ID="K1" Table2ID="T2" TableKey2ID="K2"/></System>`))
	if err != nil {
		t.Fatal(err)
	}
	got := DiagnoseRelationalSourceFacts(doc, ExtractRelationalSourceFacts(doc))
	if len(got) != 1 ||
		got[0].Code != "relationship-key-data-type-mismatch" ||
		got[0].RelationshipID != "R1" ||
		got[0].LeftFieldLegacyID != "B" ||
		got[0].RightFieldLegacyID != "D" ||
		got[0].LeftDataTypeLegacyID != "D2" ||
		got[0].RightDataTypeLegacyID != "OTHER" {
		t.Fatalf("data type mismatch was not preserved as unresolved: %#v", got)
	}
}

func TestDiagnoseRelationalSourceFactsDoesNotGuessCompositeOrderWhenSortKeyIsAmbiguous(t *testing.T) {
	doc, err := Parse(FormatXead, []byte(`<System><Table ID="T1"><TableField ID="A" DataTypeID="D1"/><TableField ID="B" DataTypeID="D2"/><TableKey ID="K1" Type="PK"><TableKeyField FieldID="A" SortKey="10"/><TableKeyField FieldID="B" SortKey="10"/></TableKey></Table><Table ID="T2"><TableField ID="C" DataTypeID="D1"/><TableField ID="D" DataTypeID="D2"/><TableKey ID="K2" Type="PK"><TableKeyField FieldID="C" SortKey="10"/><TableKeyField FieldID="D" SortKey="20"/></TableKey></Table><Relationship ID="R1" Table1ID="T1" TableKey1ID="K1" Table2ID="T2" TableKey2ID="K2"/></System>`))
	if err != nil {
		t.Fatal(err)
	}
	got := DiagnoseRelationalSourceFacts(doc, ExtractRelationalSourceFacts(doc))
	if len(got) != 1 ||
		got[0].Code != "relationship-key-order-unresolved" ||
		got[0].Side != "table1" ||
		got[0].RelationshipID != "R1" ||
		got[0].TableLegacyID != "T1" ||
		got[0].KeyLegacyID != "K1" {
		t.Fatalf("ambiguous SortKey order was guessed instead of preserved: %#v", got)
	}
}

func TestDiagnoseRelationalSourceFactsRejectsDuplicateLegacyKeyIdentity(t *testing.T) {
	doc, err := Parse(FormatXead, []byte(`<System><Table ID="T1"><TableField ID="A"/><TableKey ID="K1" Type="PK"><TableKeyField FieldID="A"/></TableKey><TableKey ID="K1" Type="SK"><TableKeyField FieldID="A"/></TableKey></Table><Table ID="T2"><TableField ID="B"/><TableKey ID="K2" Type="PK"><TableKeyField FieldID="B"/></TableKey></Table><Relationship ID="R1" Table1ID="T1" TableKey1ID="K1" Table2ID="T2" TableKey2ID="K2"/></System>`))
	if err != nil {
		t.Fatal(err)
	}
	got := DiagnoseRelationalSourceFacts(doc, ExtractRelationalSourceFacts(doc))
	codes := map[string]RelationalDiagnostic{}
	for _, diagnostic := range got {
		codes[diagnostic.Code] = diagnostic
	}
	duplicate, ok := codes["key-id-duplicate"]
	if !ok ||
		duplicate.TableLegacyID != "T1" ||
		duplicate.KeyLegacyID != "K1" ||
		duplicate.Count != 2 {
		t.Fatalf("duplicate key identity was not diagnosed: %#v", got)
	}
	ambiguous, ok := codes["relationship-key-ambiguous"]
	if !ok ||
		ambiguous.RelationshipID != "R1" ||
		ambiguous.Side != "table1" ||
		ambiguous.TableLegacyID != "T1" ||
		ambiguous.KeyLegacyID != "K1" ||
		ambiguous.Count != 2 {
		t.Fatalf("relationship resolved an ambiguous key identity: %#v", got)
	}
	if len(got) != 2 {
		t.Fatalf("unexpected duplicate-key diagnostics: %#v", got)
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
