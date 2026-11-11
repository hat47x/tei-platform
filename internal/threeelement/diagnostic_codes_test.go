package threeelement

import "testing"

// Every diagnostic code this package writes describes a specific defect in a
// three-element model. A code that no test names is a rule nothing proves can
// fire, so each one is produced here by breaking one field of an otherwise empty
// model (#4032).
func TestDiagnosticCodesAreProduced(t *testing.T) {
	cases := []struct {
		name  string
		model ThreeElementModel
		want  string
	}{
		{
			name:  "data element without an id",
			model: ThreeElementModel{DataElements: []DataElement{{}}},
			want:  "EMPTY_DATA_ID",
		},
		{
			name:  "data element without a name",
			model: ThreeElementModel{DataElements: []DataElement{{ID: "order"}}},
			want:  "EMPTY_DATA_NAME",
		},
		{
			name: "attribute without an id",
			model: ThreeElementModel{DataElements: []DataElement{
				{ID: "order", Name: "Order", Attributes: []DataAttribute{{}}},
			}},
			want: "EMPTY_ATTRIBUTE_ID",
		},
		{
			name: "relation without a target",
			model: ThreeElementModel{DataElements: []DataElement{
				{ID: "order", Name: "Order", Relations: []DataRelation{{ID: "r1"}}},
			}},
			want: "EMPTY_RELATION_TARGET",
		},
		{
			name: "relation to unknown data",
			model: ThreeElementModel{DataElements: []DataElement{
				{ID: "order", Name: "Order", Relations: []DataRelation{{ID: "r1", TargetID: "missing"}}},
			}},
			want: "UNKNOWN_RELATION_TARGET",
		},
		{
			name: "composite key without a matching attribute",
			model: ThreeElementModel{DataElements: []DataElement{
				{ID: "order", Name: "Order", CompositeKey: []string{"orderId"},
					Attributes: []DataAttribute{{ID: "other"}}},
			}},
			want: "COMPOSITE_KEY_MISSING_ATTRIBUTE",
		},
		{
			name:  "function element without an id",
			model: ThreeElementModel{FunctionElements: []FunctionElement{{}}},
			want:  "EMPTY_FUNCTION_ID",
		},
		{
			name:  "function element without a name",
			model: ThreeElementModel{FunctionElements: []FunctionElement{{ID: "submit"}}},
			want:  "EMPTY_FUNCTION_NAME",
		},
		{
			name: "operation on unknown data",
			model: ThreeElementModel{FunctionElements: []FunctionElement{
				{ID: "submit", Name: "Submit", Operations: []FunctionOperation{{DataID: "missing", Operations: "R"}}},
			}},
			want: "UNKNOWN_OPERATION_TARGET",
		},
		{
			name: "operation letter outside CRUD",
			model: ThreeElementModel{FunctionElements: []FunctionElement{
				{ID: "submit", Name: "Submit", Operations: []FunctionOperation{{DataID: "missing", Operations: "X"}}},
			}},
			want: "UNKNOWN_OPERATION_LETTER",
		},
		{
			name: "invoke of an unknown function",
			model: ThreeElementModel{FunctionElements: []FunctionElement{
				{ID: "submit", Name: "Submit", Invokes: []string{"missing"}},
			}},
			want: "UNKNOWN_INVOKE_TARGET",
		},
		{
			name:  "business element without an id",
			model: ThreeElementModel{BusinessElements: []BusinessElement{{}}},
			want:  "EMPTY_BUSINESS_ID",
		},
		{
			name:  "business element without a name",
			model: ThreeElementModel{BusinessElements: []BusinessElement{{ID: "sales"}}},
			want:  "EMPTY_BUSINESS_NAME",
		},
		{
			name: "activity without an id",
			model: ThreeElementModel{BusinessElements: []BusinessElement{
				{ID: "sales", Name: "Sales", Activities: []BusinessActivity{{}}},
			}},
			want: "EMPTY_ACTIVITY_ID",
		},
		{
			name: "activity referencing an unknown function",
			model: ThreeElementModel{BusinessElements: []BusinessElement{
				{ID: "sales", Name: "Sales", Activities: []BusinessActivity{{ID: "a1", FunctionIDs: []string{"missing"}}}},
			}},
			want: "UNKNOWN_ACTIVITY_FUNCTION",
		},
		{
			name:  "cross-reference without an id",
			model: ThreeElementModel{CrossReferences: []CrossReference{{FromKind: "data", FromID: "order", ToKind: "data", ToID: "order"}}},
			want:  "EMPTY_CROSS_REF_ID",
		},
		{
			name:  "cross-reference from an unknown subject",
			model: ThreeElementModel{CrossReferences: []CrossReference{{ID: "x1", FromKind: "data", FromID: "missing", ToKind: "data", ToID: "order"}}},
			want:  "UNKNOWN_CROSS_REF_FROM",
		},
		{
			name:  "cross-reference to an unknown subject",
			model: ThreeElementModel{CrossReferences: []CrossReference{{ID: "x1", FromKind: "data", FromID: "order", ToKind: "data", ToID: "missing"}}},
			want:  "UNKNOWN_CROSS_REF_TO",
		},
		{
			name:  "CRUD referencing an unknown function",
			model: ThreeElementModel{CRUDDefinitions: []CRUDDefinition{{ID: "c1", FunctionID: "missing", DataID: "order", Read: true}}},
			want:  "UNKNOWN_CRUD_FUNCTION",
		},
		{
			name:  "CRUD referencing unknown data",
			model: ThreeElementModel{CRUDDefinitions: []CRUDDefinition{{ID: "c1", FunctionID: "submit", DataID: "missing", Read: true}}},
			want:  "UNKNOWN_CRUD_DATA",
		},
		{
			name:  "CRUD without any operation",
			model: ThreeElementModel{CRUDDefinitions: []CRUDDefinition{{ID: "c1", FunctionID: "submit", DataID: "order"}}},
			want:  "EMPTY_CRUD_DEFINITION",
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			model := testCase.model
			model.ModelID, model.Name = "probe", "Probe"
			for _, diagnostic := range model.Validate() {
				if diagnostic.Code == testCase.want {
					return
				}
			}
			var codes []string
			for _, diagnostic := range model.Validate() {
				codes = append(codes, diagnostic.Code)
			}
			t.Fatalf("expected %s, got %v", testCase.want, codes)
		})
	}
}
