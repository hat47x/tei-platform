package threeelement

import "testing"

// Each of these codes is a contract: a caller branches on it. A code no test
// mentions cannot be shown to be reachable, so the run-time value would mean less
// than its name suggests (#4032). The fixture below builds the smallest model
// that emits each one.

// codesFrom runs Validate and returns the set of codes it produced.
func codesFrom(model ThreeElementModel) map[string]bool {
	codes := map[string]bool{}
	for _, diagnostic := range model.Validate() {
		codes[diagnostic.Code] = true
	}
	return codes
}

func TestValidateReportsEmptyIdentities(t *testing.T) {
	model := ThreeElementModel{
		ModelID: "empty_ids",
		DataElements: []DataElement{{
			ID: "", Name: "",
			Attributes: []DataAttribute{{ID: ""}},
		}},
		FunctionElements: []FunctionElement{{ID: "", Name: ""}},
		BusinessElements: []BusinessElement{{
			ID: "", Name: "",
			Activities: []BusinessActivity{{ID: ""}},
		}},
		CrossReferences: []CrossReference{{ID: "", FromKind: "data", FromID: "", ToKind: "data", ToID: ""}},
	}
	codes := codesFrom(model)
	for _, want := range []string{
		"EMPTY_DATA_ID", "EMPTY_DATA_NAME", "EMPTY_ATTRIBUTE_ID",
		"EMPTY_FUNCTION_ID", "EMPTY_FUNCTION_NAME",
		"EMPTY_BUSINESS_ID", "EMPTY_BUSINESS_NAME", "EMPTY_ACTIVITY_ID",
		"EMPTY_CROSS_REF_ID",
	} {
		if !codes[want] {
			t.Errorf("Validate did not report %s; codes=%v", want, codes)
		}
	}
}

func TestValidateReportsDanglingReferences(t *testing.T) {
	model := ThreeElementModel{
		ModelID:      "dangling",
		DataElements: []DataElement{{ID: "order", Name: "Order", Relations: []DataRelation{{ID: "r1", TargetID: ""}}}},
		FunctionElements: []FunctionElement{{
			ID: "f1", Name: "F",
			Operations: []FunctionOperation{{DataID: "missing", Operations: "X"}},
			Invokes:    []string{"missing_fn"},
		}},
		BusinessElements: []BusinessElement{{
			ID: "b1", Name: "B",
			Activities: []BusinessActivity{{ID: "a1", FunctionIDs: []string{"missing_fn"}}},
		}},
		CrossReferences: []CrossReference{{
			ID: "x1", FromKind: "nothing", FromID: "missing", ToKind: "nothing", ToID: "missing",
		}},
	}
	codes := codesFrom(model)
	for _, want := range []string{
		"EMPTY_RELATION_TARGET",
		"UNKNOWN_OPERATION_TARGET", "UNKNOWN_OPERATION_LETTER", "UNKNOWN_INVOKE_TARGET",
		"UNKNOWN_ACTIVITY_FUNCTION", "UNKNOWN_CROSS_REF_FROM", "UNKNOWN_CROSS_REF_TO",
	} {
		if !codes[want] {
			t.Errorf("Validate did not report %s; codes=%v", want, codes)
		}
	}
}

func TestValidateReportsUnknownRelationTargetAndCompositeKey(t *testing.T) {
	model := ThreeElementModel{
		ModelID: "targets",
		DataElements: []DataElement{{
			ID: "order", Name: "Order",
			Relations:    []DataRelation{{ID: "r1", TargetID: "elsewhere"}},
			CompositeKey: []string{"not_an_attribute"},
		}},
	}
	codes := codesFrom(model)
	for _, want := range []string{"UNKNOWN_RELATION_TARGET", "COMPOSITE_KEY_MISSING_ATTRIBUTE"} {
		if !codes[want] {
			t.Errorf("Validate did not report %s; codes=%v", want, codes)
		}
	}
}

func TestValidateReportsCRUDProblems(t *testing.T) {
	model := ThreeElementModel{
		ModelID:          "crud",
		DataElements:     []DataElement{{ID: "order", Name: "Order"}},
		FunctionElements: []FunctionElement{{ID: "f1", Name: "F"}},
		CRUDDefinitions: []CRUDDefinition{
			{ID: "c1", FunctionID: "missing", DataID: "missing"},
			{ID: "c2", FunctionID: "f1", DataID: "order"},
		},
	}
	codes := codesFrom(model)
	for _, want := range []string{"UNKNOWN_CRUD_FUNCTION", "UNKNOWN_CRUD_DATA", "EMPTY_CRUD_DEFINITION"} {
		if !codes[want] {
			t.Errorf("Validate did not report %s; codes=%v", want, codes)
		}
	}
}

// The point of the coverage check is reachability, not merely the string: a
// fixture that produces nothing must not be able to satisfy it.
func TestHealthyModelEmitsNoneOfTheseCodes(t *testing.T) {
	model := ThreeElementModel{
		ModelID:          "healthy",
		DataElements:     []DataElement{{ID: "order", Name: "Order", Attributes: []DataAttribute{{ID: "id", Name: "ID"}}}},
		FunctionElements: []FunctionElement{{ID: "f1", Name: "F"}},
		BusinessElements: []BusinessElement{{ID: "b1", Name: "B"}},
		CRUDDefinitions:  []CRUDDefinition{{ID: "c1", FunctionID: "f1", DataID: "order", Read: true}},
	}
	codes := codesFrom(model)
	for _, code := range []string{
		"EMPTY_DATA_ID", "EMPTY_CRUD_DEFINITION", "UNKNOWN_CRUD_FUNCTION",
		"UNKNOWN_RELATION_TARGET", "COMPOSITE_KEY_MISSING_ATTRIBUTE",
	} {
		if codes[code] {
			t.Errorf("a healthy model reported %s", code)
		}
	}
}
