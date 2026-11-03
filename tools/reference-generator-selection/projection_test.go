package generatorselection

import (
	"encoding/json"
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
