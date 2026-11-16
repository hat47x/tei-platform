package xeadsource

import (
	"reflect"
	"testing"
)

// A composite design key declares its member order only through TableKeyField
// SortKey. When that order is not uniquely determined, the key's own structure
// is unresolved regardless of whether any Relationship references the key.
// These expectations are written from the synthetic source declaration alone;
// they are not transcribed from the current diagnostic output.
func TestInspectDiagnosesCompositeKeyOrderWithoutRelationshipReference(t *testing.T) {
	source := []byte(`<System>
		<Table ID="T1">
			<TableField ID="A" DataTypeID="DT1"/>
			<TableField ID="B" DataTypeID="DT2"/>
			<TableKey ID="KDUPLICATE" Type="PK">
				<TableKeyField FieldID="A" SortKey="10"/>
				<TableKeyField FieldID="B" SortKey="10"/>
			</TableKey>
			<TableKey ID="KMISSING" Type="PK">
				<TableKeyField FieldID="A" SortKey="10"/>
				<TableKeyField FieldID="B"/>
			</TableKey>
			<TableKey ID="KORDERED" Type="SK">
				<TableKeyField FieldID="A" SortKey="10"/>
				<TableKeyField FieldID="B" SortKey="20"/>
			</TableKey>
		</Table>
	</System>`)
	inspection, err := Inspect(FormatXead, source, nil)
	if err != nil {
		t.Fatal(err)
	}

	diagnosticsByKey := map[string][]string{}
	for _, diagnostic := range inspection.RelationalDiagnostics {
		if diagnostic.Nature != RelationalDiagnosticNature {
			t.Fatalf("key order diagnostic gained a non-unresolved nature: %#v", diagnostic)
		}
		diagnosticsByKey[diagnostic.KeyLegacyID] = append(
			diagnosticsByKey[diagnostic.KeyLegacyID],
			diagnostic.Code,
		)
	}
	if want := []string{"key-order-unresolved"}; !reflect.DeepEqual(diagnosticsByKey["KDUPLICATE"], want) {
		t.Fatalf("duplicate SortKey was not preserved as unresolved: %#v", diagnosticsByKey)
	}
	if want := []string{"key-order-unresolved"}; !reflect.DeepEqual(diagnosticsByKey["KMISSING"], want) {
		t.Fatalf("missing SortKey was not preserved as unresolved: %#v", diagnosticsByKey)
	}
	if _, reported := diagnosticsByKey["KORDERED"]; reported {
		t.Fatalf("uniquely ordered key was reported as unresolved: %#v", diagnosticsByKey)
	}

	resolutionByKey := map[string]RelationalFactResolution{}
	for _, resolution := range inspection.RelationalFactResolutions {
		if resolution.Kind != "legacy-key" {
			continue
		}
		for _, key := range inspection.RelationalFacts.LegacyKeys {
			if key.Locator == resolution.Locator {
				resolutionByKey[key.KeyLegacyID] = resolution
			}
		}
	}
	for _, unresolved := range []string{"KDUPLICATE", "KMISSING"} {
		resolution, ok := resolutionByKey[unresolved]
		if !ok || resolution.Status != RelationalFactResolutionUnresolved {
			t.Fatalf("key %q was reported as locally resolved: %#v", unresolved, resolutionByKey)
		}
		if !reflect.DeepEqual(resolution.DiagnosticCodes, []string{"key-order-unresolved"}) {
			t.Fatalf("key %q lost its local order diagnostic: %#v", unresolved, resolution)
		}
	}
	if ordered := resolutionByKey["KORDERED"]; ordered.Status != RelationalFactResolutionResolved {
		t.Fatalf("ordered key was blocked by unrelated keys: %#v", ordered)
	}
	if inspection.RelationalResolutionStatus != RelationalFactResolutionUnresolved {
		t.Fatalf("full-set status hid the unresolved composite key: %#v", inspection)
	}
}

// A single-member key needs no declared order; the existing resolved reading of
// ordinary known structures must not change.
func TestInspectKeepsSingleMemberKeyResolvedWithoutSortKey(t *testing.T) {
	inspection, err := Inspect(
		FormatXead,
		[]byte(`<System><Table ID="T1"><TableField ID="A"/><TableKey ID="K1" Type="PK"><TableKeyField FieldID="A"/></TableKey></Table></System>`),
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(inspection.RelationalDiagnostics) != 0 {
		t.Fatalf("single-member key produced order diagnostics: %#v", inspection.RelationalDiagnostics)
	}
	if inspection.RelationalResolutionStatus != RelationalFactResolutionResolved {
		t.Fatalf("single-member key was not resolved: %#v", inspection)
	}
}

// The key's own order defect and the relationship's dependency on that key are
// separate observations; neither may be used to fill the other's meaning.
func TestInspectKeepsKeyOrderDiagnosticSeparateFromRelationshipDiagnostic(t *testing.T) {
	source := []byte(`<System>
		<Table ID="T1">
			<TableField ID="A" DataTypeID="DT1"/>
			<TableField ID="B" DataTypeID="DT2"/>
			<TableKey ID="K1" Type="PK">
				<TableKeyField FieldID="A" SortKey="10"/>
				<TableKeyField FieldID="B" SortKey="10"/>
			</TableKey>
		</Table>
		<Table ID="T2">
			<TableField ID="C" DataTypeID="DT1"/>
			<TableField ID="D" DataTypeID="DT2"/>
			<TableKey ID="K2" Type="PK">
				<TableKeyField FieldID="C" SortKey="10"/>
				<TableKeyField FieldID="D" SortKey="20"/>
			</TableKey>
		</Table>
		<Relationship ID="R1" Table1ID="T1" TableKey1ID="K1" Table2ID="T2" TableKey2ID="K2"/>
	</System>`)
	inspection, err := Inspect(FormatXead, source, nil)
	if err != nil {
		t.Fatal(err)
	}

	var sawKeyOrder, sawRelationshipOrder bool
	for _, diagnostic := range inspection.RelationalDiagnostics {
		switch diagnostic.Code {
		case "key-order-unresolved":
			sawKeyOrder = diagnostic.KeyLegacyID == "K1" && diagnostic.TableLegacyID == "T1" &&
				diagnostic.RelationshipID == "" && diagnostic.Side == ""
		case "relationship-key-order-unresolved":
			sawRelationshipOrder = diagnostic.RelationshipID == "R1" &&
				diagnostic.Side == "table1" && diagnostic.KeyLegacyID == "K1"
		}
	}
	if !sawKeyOrder || !sawRelationshipOrder || len(inspection.RelationalDiagnostics) != 2 {
		t.Fatalf("key order and relationship order were merged or lost: %#v", inspection.RelationalDiagnostics)
	}
}
