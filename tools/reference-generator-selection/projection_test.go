package generatorselection

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func projectionContract(kind string, rounding Rounding) Contract {
	return Contract{
		Schema: ContractSchema, ID: "sui.card.text", Version: "1", Status: "reference-local",
		Inputs:     []InputSpec{{Attribute: "sui.card.text", Primitive: "std.text"}},
		Output:     TypeSpec{Primitive: "std.text"},
		Rounding:   rounding,
		Projection: &ProjectionSpec{Kind: kind},
	}
}

func TestValidateProjectionContractAcceptsNoRounding(t *testing.T) {
	if err := validateContract(projectionContract("identity", Rounding{})); err != nil {
		t.Fatal(err)
	}
	if err := validateContract(projectionContract("normalize", Rounding{})); err != nil {
		t.Fatal(err)
	}
}

func TestValidateProjectionContractRejectsRoundingAndUnknownKind(t *testing.T) {
	if err := validateContract(projectionContract("identity", Rounding{Precision: 8, Scale: 2, Mode: "half-up"})); err == nil {
		t.Fatal("a projection contract with rounding was accepted")
	}
	if err := validateContract(projectionContract("uppercase", Rounding{})); err == nil {
		t.Fatal("an unknown projection kind was accepted")
	}
}

func TestProjectionResultKeepsTextAndState(t *testing.T) {
	var got Result
	if err := json.Unmarshal([]byte(`{"state":"present","text":"hello"}`), &got); err != nil {
		t.Fatal(err)
	}
	want := Result{State: "present", Text: "hello"}
	if got != want {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

func recordProjectionContract() Contract {
	return Contract{
		Schema: ContractSchema, ID: "sui.card.record", Version: "1", Status: "reference-local",
		Inputs: []InputSpec{
			{Attribute: "sui.card.text", Primitive: "std.text"},
			{Attribute: "sui.card.document_id", Primitive: "std.identifier"},
		},
		Output: TypeSpec{Primitive: "std.text"},
		Projection: &ProjectionSpec{Kind: "record", Fields: []ProjectionField{
			{Name: "text", Input: "text", Attribute: "sui.card.text", Primitive: "std.text"},
			{Name: "documentId", Input: "documentId", Attribute: "sui.card.document_id", Primitive: "std.identifier"},
		}},
	}
}

// #2585 PFC-08: a record field must reference a declared contract input and
// agree with its primitive. The runtime input key stays a free name.
func TestRecordProjectionFieldsMustReferenceDeclaredInputs(t *testing.T) {
	if err := validateContract(recordProjectionContract()); err != nil {
		t.Fatalf("the reference record contract was rejected: %v", err)
	}

	undeclared := recordProjectionContract()
	undeclared.Projection.Fields[0].Attribute = "sui.card.unknown"
	if err := validateContract(undeclared); err == nil {
		t.Fatal("a field referencing an undeclared contract input was accepted")
	}

	mismatched := recordProjectionContract()
	mismatched.Projection.Fields[1].Primitive = "std.text"
	if err := validateContract(mismatched); err == nil {
		t.Fatal("a field primitive that disagrees with the contract input was accepted")
	}
}

func TestReferenceRecordProjectionContractIsValid(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("testdata", "sui-record", "contract.json"))
	if err != nil {
		t.Fatal(err)
	}
	var contract Contract
	if err := json.Unmarshal(raw, &contract); err != nil {
		t.Fatal(err)
	}
	if err := validateContract(contract); err != nil {
		t.Fatalf("the checked-in record contract is invalid: %v", err)
	}
}
