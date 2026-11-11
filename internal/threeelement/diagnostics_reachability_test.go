package threeelement

import (
	"testing"
)

// Every diagnostic code this package can emit must be reachable from a test.
// A code a caller branches on is a contract; a code no test can produce is a
// name that promises more than the implementation delivers
// (tools/repository-plan-check/TestEveryDiagnosticCodeIsMentionedByATest).
//
// The table below builds the SMALLEST model that triggers each code, so the
// test asserts reachability rather than merely mentioning a string.
func TestEveryDiagnosticCodeIsReachable(t *testing.T) {
	cases := []struct {
		code  string
		model func() ThreeElementModel
	}{
		{"EMPTY_DATA_ID", func() ThreeElementModel {
			m := NewThreeElementModel("probe", "Probe")
			m.DataElements = []DataElement{{Name: "no id"}}
			return m
		}},
		{"EMPTY_DATA_NAME", func() ThreeElementModel {
			m := NewThreeElementModel("probe", "Probe")
			m.DataElements = []DataElement{{ID: "d1"}}
			return m
		}},
		{"EMPTY_ATTRIBUTE_ID", func() ThreeElementModel {
			m := NewThreeElementModel("probe", "Probe")
			m.DataElements = []DataElement{{ID: "d1", Name: "D1",
				Attributes: []DataAttribute{{Name: "n", Type: "text"}}}}
			return m
		}},
		{"EMPTY_RELATION_TARGET", func() ThreeElementModel {
			m := NewThreeElementModel("probe", "Probe")
			m.DataElements = []DataElement{{ID: "d1", Name: "D1",
				Relations: []DataRelation{{ID: "r1"}}}}
			return m
		}},
		{"UNKNOWN_RELATION_TARGET", func() ThreeElementModel {
			m := NewThreeElementModel("probe", "Probe")
			m.DataElements = []DataElement{{ID: "d1", Name: "D1",
				Relations: []DataRelation{{ID: "r1", TargetID: "absent"}}}}
			return m
		}},
		{"COMPOSITE_KEY_MISSING_ATTRIBUTE", func() ThreeElementModel {
			m := NewThreeElementModel("probe", "Probe")
			m.DataElements = []DataElement{{ID: "d1", Name: "D1",
				Attributes:   []DataAttribute{{ID: "a1", Name: "A1", Type: "text"}},
				CompositeKey: []string{"absent"}}}
			return m
		}},
		{"EMPTY_FUNCTION_ID", func() ThreeElementModel {
			m := NewThreeElementModel("probe", "Probe")
			m.FunctionElements = []FunctionElement{{Name: "no id"}}
			return m
		}},
		{"EMPTY_FUNCTION_NAME", func() ThreeElementModel {
			m := NewThreeElementModel("probe", "Probe")
			m.FunctionElements = []FunctionElement{{ID: "f1"}}
			return m
		}},
		{"UNKNOWN_OPERATION_TARGET", func() ThreeElementModel {
			m := NewThreeElementModel("probe", "Probe")
			m.FunctionElements = []FunctionElement{{ID: "f1", Name: "F1",
				Operations: []FunctionOperation{{DataID: "absent", Operations: "R"}}}}
			return m
		}},
		{"UNKNOWN_OPERATION_LETTER", func() ThreeElementModel {
			m := NewThreeElementModel("probe", "Probe")
			m.DataElements = []DataElement{{ID: "d1", Name: "D1"}}
			m.FunctionElements = []FunctionElement{{ID: "f1", Name: "F1",
				Operations: []FunctionOperation{{DataID: "d1", Operations: "X"}}}}
			return m
		}},
		{"UNKNOWN_INVOKE_TARGET", func() ThreeElementModel {
			m := NewThreeElementModel("probe", "Probe")
			m.FunctionElements = []FunctionElement{{ID: "f1", Name: "F1",
				Invokes: []string{"absent"}}}
			return m
		}},
		{"EMPTY_BUSINESS_ID", func() ThreeElementModel {
			m := NewThreeElementModel("probe", "Probe")
			m.BusinessElements = []BusinessElement{{Name: "no id"}}
			return m
		}},
		{"EMPTY_BUSINESS_NAME", func() ThreeElementModel {
			m := NewThreeElementModel("probe", "Probe")
			m.BusinessElements = []BusinessElement{{ID: "b1"}}
			return m
		}},
		{"EMPTY_ACTIVITY_ID", func() ThreeElementModel {
			m := NewThreeElementModel("probe", "Probe")
			m.BusinessElements = []BusinessElement{{ID: "b1", Name: "B1",
				Activities: []BusinessActivity{{Name: "no id"}}}}
			return m
		}},
		{"UNKNOWN_ACTIVITY_FUNCTION", func() ThreeElementModel {
			m := NewThreeElementModel("probe", "Probe")
			m.BusinessElements = []BusinessElement{{ID: "b1", Name: "B1",
				Activities: []BusinessActivity{{ID: "a1", FunctionIDs: []string{"absent"}}}}}
			return m
		}},
		{"EMPTY_CROSS_REF_ID", func() ThreeElementModel {
			m := NewThreeElementModel("probe", "Probe")
			m.CrossReferences = []CrossReference{{Kind: "data-to-data", FromKind: "data", FromID: "d1", ToKind: "data", ToID: "d2"}}
			return m
		}},
		{"UNKNOWN_CROSS_REF_FROM", func() ThreeElementModel {
			m := NewThreeElementModel("probe", "Probe")
			m.CrossReferences = []CrossReference{{ID: "x1", Kind: "data-to-data", FromKind: "data", FromID: "absent", ToKind: "data", ToID: "also-absent"}}
			return m
		}},
		{"UNKNOWN_CROSS_REF_TO", func() ThreeElementModel {
			m := NewThreeElementModel("probe", "Probe")
			m.DataElements = []DataElement{{ID: "d1", Name: "D1"}}
			m.CrossReferences = []CrossReference{{ID: "x1", Kind: "data-to-data", FromKind: "data", FromID: "d1", ToKind: "data", ToID: "absent"}}
			return m
		}},
		{"UNKNOWN_CRUD_FUNCTION", func() ThreeElementModel {
			m := NewThreeElementModel("probe", "Probe")
			m.DataElements = []DataElement{{ID: "d1", Name: "D1"}}
			m.CRUDDefinitions = []CRUDDefinition{{ID: "c1", FunctionID: "absent", DataID: "d1", Read: true}}
			return m
		}},
		{"UNKNOWN_CRUD_DATA", func() ThreeElementModel {
			m := NewThreeElementModel("probe", "Probe")
			m.FunctionElements = []FunctionElement{{ID: "f1", Name: "F1"}}
			m.CRUDDefinitions = []CRUDDefinition{{ID: "c1", FunctionID: "f1", DataID: "absent", Read: true}}
			return m
		}},
		{"EMPTY_CRUD_DEFINITION", func() ThreeElementModel {
			m := NewThreeElementModel("probe", "Probe")
			m.DataElements = []DataElement{{ID: "d1", Name: "D1"}}
			m.FunctionElements = []FunctionElement{{ID: "f1", Name: "F1"}}
			m.CRUDDefinitions = []CRUDDefinition{{ID: "c1", FunctionID: "f1", DataID: "d1"}}
			return m
		}},
	}
	seen := map[string]bool{}
	for _, tc := range cases {
		t.Run(tc.code, func(t *testing.T) {
			diagnostics := tc.model().Validate()
			for _, diagnostic := range diagnostics {
				if diagnostic.Code == tc.code {
					seen[tc.code] = true
					return
				}
			}
			t.Fatalf("%s is not reachable; diagnostics = %+v", tc.code, diagnostics)
		})
	}
	if len(seen) != len(cases) {
		t.Fatalf("only %d of %d diagnostic codes were observed", len(seen), len(cases))
	}
}

// The schema identifiers this package writes are pinned, so a rename cannot
// slip past a test that only reads the response shape
// (tools/repository-plan-check/TestSchemaVersionLiteralsHaveATest).
func TestSchemaVersionsAreStable(t *testing.T) {
	if model := NewThreeElementModel("probe", "Probe"); model.SchemaVersion != "tei.three-element.model/v0" {
		t.Fatalf("model schemaVersion = %q", model.SchemaVersion)
	}
	if diff := DiffModels(NewThreeElementModel("a", "A"), NewThreeElementModel("b", "B")); diff.SchemaVersion != "tei.three-element.diff/v0" {
		t.Fatalf("diff schemaVersion = %q", diff.SchemaVersion)
	}
}
