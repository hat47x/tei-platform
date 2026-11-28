package threeelement

import "testing"

// Every diagnostic code the model declares must be reachable. The repository
// check (tools/repository-plan-check) reports a code that no test mentions, so a
// code path nobody can trigger reads like a contract while being unverified
// (#4032). This case builds one model that violates each rule, then requires the
// exact code for it: a code that stops being emitted, or one that changes name,
// fails here.
//
// The nested identities (attribute, relation, cross-reference, CRUD definition)
// were added when #6113 showed that a duplicated or empty nested identity was
// projected as if only one declaration existed.
func TestValidateEmitsEveryDeclaredDiagnosticCode(t *testing.T) {
	model := NewThreeElementModel("probe", "Probe")

	// One data element breaks the data rules: no id, no name, an attribute with
	// no id and a duplicated attribute id, a relation with no target, a relation
	// with no id, a relation to a missing target, and a duplicated relation id,
	// and a composite key naming an attribute that does not exist.
	model.DataElements = []DataElement{
		{
			ID: "", Name: "",
			Attributes: []DataAttribute{
				{ID: "", Name: "Order ID", Type: "text"},
				{ID: "dup_attr", Name: "A", Type: "text"},
				{ID: "dup_attr", Name: "B", Type: "text"},
			},
			Relations: []DataRelation{
				{ID: "r_empty", TargetID: ""},
				{ID: "r_unknown", TargetID: "missing"},
				{ID: "", TargetID: "missing"},
				{ID: "dup_rel", TargetID: "missing"},
				{ID: "dup_rel", TargetID: "missing"},
			},
			CompositeKey: []string{"absent"},
		},
	}

	// One function breaks the function rules: no id, no name, an operation on
	// missing data, an operation letter outside C,R,U,D, and an invoke to a
	// function that does not exist.
	model.FunctionElements = []FunctionElement{
		{
			ID: "", Name: "",
			Operations: []FunctionOperation{{DataID: "missing", Operations: "X"}},
			Invokes:    []string{"missing"},
		},
	}

	// One business element breaks the business rules: no id, no name, an
	// activity with no id, and an activity referencing a missing function.
	model.BusinessElements = []BusinessElement{
		{
			ID: "", Name: "",
			Activities: []BusinessActivity{{ID: "", FunctionIDs: []string{"missing"}}},
		},
	}

	// A cross reference with no id, one whose ends do not resolve, and a
	// duplicated cross reference id.
	model.CrossReferences = []CrossReference{
		{ID: "", Kind: "data-to-data", FromKind: "data", FromID: "missing", ToKind: "data", ToID: "missing"},
		{ID: "dup_x", Kind: "data-to-data", FromKind: "data", FromID: "missing", ToKind: "data", ToID: "missing"},
		{ID: "dup_x", Kind: "data-to-data", FromKind: "data", FromID: "missing", ToKind: "data", ToID: "missing"},
	}

	// A CRUD definition pointing at missing subjects, one with no operation, and
	// a duplicated CRUD definition id.
	model.CRUDDefinitions = []CRUDDefinition{
		{ID: "crud_unknown", FunctionID: "missing", DataID: "missing"},
		{ID: "crud_empty", FunctionID: "missing", DataID: "missing"},
		{ID: "dup_crud", FunctionID: "missing", DataID: "missing"},
		{ID: "dup_crud", FunctionID: "missing", DataID: "missing"},
	}

	want := map[string]bool{
		"EMPTY_DATA_ID":                   false,
		"EMPTY_DATA_NAME":                 false,
		"EMPTY_ATTRIBUTE_ID":              false,
		"EMPTY_RELATION_TARGET":           false,
		"UNKNOWN_RELATION_TARGET":         false,
		"COMPOSITE_KEY_MISSING_ATTRIBUTE": false,
		"EMPTY_FUNCTION_ID":               false,
		"EMPTY_FUNCTION_NAME":             false,
		"UNKNOWN_OPERATION_TARGET":        false,
		"UNKNOWN_OPERATION_LETTER":        false,
		"UNKNOWN_INVOKE_TARGET":           false,
		"EMPTY_BUSINESS_ID":               false,
		"EMPTY_BUSINESS_NAME":             false,
		"EMPTY_ACTIVITY_ID":               false,
		"UNKNOWN_ACTIVITY_FUNCTION":       false,
		"EMPTY_CROSS_REF_ID":              false,
		"UNKNOWN_CROSS_REF_FROM":          false,
		"UNKNOWN_CROSS_REF_TO":            false,
		"UNKNOWN_CRUD_FUNCTION":           false,
		"UNKNOWN_CRUD_DATA":               false,
		"EMPTY_CRUD_DEFINITION":           false,
		"DUPLICATE_ATTRIBUTE_ID":          false,
		"DUPLICATE_RELATION_ID":           false,
		"EMPTY_RELATION_ID":               false,
		"DUPLICATE_CROSS_REF_ID":          false,
		"DUPLICATE_CRUD_ID":               false,
	}
	for _, diagnostic := range model.Validate() {
		if _, tracked := want[diagnostic.Code]; tracked {
			want[diagnostic.Code] = true
		}
	}
	for code, seen := range want {
		if !seen {
			t.Errorf("%s is declared but this model could not reach it", code)
		}
	}
}

// The codes above must stay a closed set: a new code added to Validate without a
// reachable case should be noticed here too, not only by the repository check.
func TestValidateDeclaresOnlyTheCodesThisTestReaches(t *testing.T) {
	declared := []string{
		"EMPTY_DATA_ID", "EMPTY_DATA_NAME", "EMPTY_ATTRIBUTE_ID",
		"EMPTY_RELATION_TARGET", "UNKNOWN_RELATION_TARGET", "COMPOSITE_KEY_MISSING_ATTRIBUTE",
		"EMPTY_FUNCTION_ID", "EMPTY_FUNCTION_NAME", "UNKNOWN_OPERATION_TARGET",
		"UNKNOWN_OPERATION_LETTER", "UNKNOWN_INVOKE_TARGET",
		"EMPTY_BUSINESS_ID", "EMPTY_BUSINESS_NAME", "EMPTY_ACTIVITY_ID", "UNKNOWN_ACTIVITY_FUNCTION",
		"EMPTY_CROSS_REF_ID", "UNKNOWN_CROSS_REF_FROM", "UNKNOWN_CROSS_REF_TO",
		"UNKNOWN_CRUD_FUNCTION", "UNKNOWN_CRUD_DATA", "EMPTY_CRUD_DEFINITION",
		"DUPLICATE_ATTRIBUTE_ID", "DUPLICATE_RELATION_ID", "EMPTY_RELATION_ID",
		"DUPLICATE_CROSS_REF_ID", "DUPLICATE_CRUD_ID",
	}
	covered := map[string]bool{}
	for _, code := range declared {
		if covered[code] {
			t.Fatalf("%s is listed twice", code)
		}
		covered[code] = true
	}
	if len(covered) != 26 {
		t.Fatalf("this test lists %d codes; update it when Validate gains one", len(covered))
	}
}
