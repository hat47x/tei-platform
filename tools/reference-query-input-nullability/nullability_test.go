package queryinputnullability

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func testdata(t *testing.T, name string) string {
	t.Helper()
	return filepath.Join("testdata", name)
}

func bindFixture(t *testing.T, requestName string) Receipt {
	t.Helper()
	receipt, err := BindFiles(testdata(t, "contract.json"), testdata(t, requestName))
	if err != nil {
		t.Fatal(err)
	}
	return receipt
}

func boundByName(receipt Receipt) map[string]BoundInput {
	out := map[string]BoundInput{}
	for _, input := range receipt.Inputs {
		out[input.Name] = input
	}
	return out
}

func TestDistinguishesMissingNullAndValue(t *testing.T) {
	receipt := bindFixture(t, "request.json")
	if !receipt.Passed || receipt.QueryNotExecuted {
		t.Fatalf("request did not pass: %+v", receipt)
	}
	bound := boundByName(receipt)
	if bound["customer_id"].State != "present" || !bound["customer_id"].Bound {
		t.Fatalf("customer_id was not present: %+v", bound["customer_id"])
	}
	if bound["nickname"].State != "null" || !bound["nickname"].Bound {
		t.Fatalf("nickname was not explicit null: %+v", bound["nickname"])
	}
	if bound["region"].State != "missing" || bound["region"].Bound {
		t.Fatalf("region was not omitted: %+v", bound["region"])
	}
	if value, ok := receipt.SQLParameters["nickname"]; !ok || value != nil {
		t.Fatalf("allowed null was not bound as SQL NULL: %+v", receipt.SQLParameters)
	}
	if _, ok := receipt.SQLParameters["region"]; ok {
		t.Fatalf("missing optional input created a parameter: %+v", receipt.SQLParameters)
	}
}

func TestMissingRequiredInputIsRejectedBeforeSQL(t *testing.T) {
	receipt := bindFixture(t, "request-missing-required.json")
	if receipt.Passed || !receipt.QueryNotExecuted {
		t.Fatalf("missing required input was not rejected: %+v", receipt)
	}
	if len(receipt.SQLParameters) != 0 {
		t.Fatalf("rejected request produced SQL parameters: %+v", receipt.SQLParameters)
	}
	if len(receipt.Rejected) != 1 || receipt.Rejected[0].Code != "MISSING_REQUIRED_INPUT" {
		t.Fatalf("unexpected diagnostics: %+v", receipt.Rejected)
	}
}

func TestForbiddenExplicitNullIsRejectedBeforeSQL(t *testing.T) {
	receipt := bindFixture(t, "request-null-forbidden.json")
	if receipt.Passed || !receipt.QueryNotExecuted {
		t.Fatalf("forbidden null was not rejected: %+v", receipt)
	}
	if len(receipt.SQLParameters) != 0 {
		t.Fatalf("rejected request produced SQL parameters: %+v", receipt.SQLParameters)
	}
	if len(receipt.Rejected) != 1 || receipt.Rejected[0].Code != "NULL_NOT_ALLOWED" {
		t.Fatalf("unexpected diagnostics: %+v", receipt.Rejected)
	}
}

func TestEmptyStringIsAValueNotNull(t *testing.T) {
	receipt := bindFixture(t, "request-empty-string.json")
	if !receipt.Passed {
		t.Fatalf("empty string request was rejected: %+v", receipt)
	}
	nickname := boundByName(receipt)["nickname"]
	if nickname.State != "present" || nickname.Value == nil || *nickname.Value != "" {
		t.Fatalf("empty string was not preserved as present: %+v", nickname)
	}
	if value, ok := receipt.SQLParameters["nickname"]; !ok || value != "" {
		t.Fatalf("empty string parameter was not preserved: %+v", receipt.SQLParameters)
	}
}

func TestWrongJSONTypeIsNotCoerced(t *testing.T) {
	receipt := bindFixture(t, "request-wrong-type.json")
	if receipt.Passed || !receipt.QueryNotExecuted {
		t.Fatalf("non-string value was not rejected: %+v", receipt)
	}
	if len(receipt.SQLParameters) != 0 {
		t.Fatalf("rejected request produced SQL parameters: %+v", receipt.SQLParameters)
	}
	if len(receipt.Rejected) != 1 || receipt.Rejected[0].Code != "UNSUPPORTED_VALUE" {
		t.Fatalf("unexpected diagnostics: %+v", receipt.Rejected)
	}
}

func TestUnknownInputIsRejected(t *testing.T) {
	receipt := bindFixture(t, "request-unknown-input.json")
	if receipt.Passed || !receipt.QueryNotExecuted {
		t.Fatalf("unknown input was not rejected: %+v", receipt)
	}
	if len(receipt.Rejected) != 1 || receipt.Rejected[0].Code != "UNKNOWN_INPUT" || receipt.Rejected[0].Input != "unexpected" {
		t.Fatalf("unexpected diagnostics: %+v", receipt.Rejected)
	}
}

func TestContractRequiresExplicitPresenceAndNullPolicy(t *testing.T) {
	contract := Contract{
		Schema: ContractSchema, ID: "q", Version: "1",
		Inputs: []Input{{Name: "id", Type: "customer.customer_id"}},
	}
	if err := ValidateContract(contract); err == nil {
		t.Fatal("contract without presence/null policy was accepted")
	}
	contract.Inputs[0].Presence = PresenceOptional
	contract.Inputs[0].Null = NullForbidden
	if err := ValidateContract(contract); err != nil {
		t.Fatalf("explicit optional/forbidden contract was rejected: %v", err)
	}
}

func TestRequiredInputMayAllowExplicitNull(t *testing.T) {
	contract := Contract{
		Schema: ContractSchema, ID: "q", Version: "1",
		Inputs: []Input{{Name: "id", Type: "customer.customer_id", Presence: PresenceRequired, Null: NullAllowed}},
	}
	request := Request{
		Schema: RequestSchema, ID: "r",
		Inputs: map[string]json.RawMessage{"id": json.RawMessage("null")},
	}
	receipt := Bind(contract, request)
	if !receipt.Passed {
		t.Fatalf("required+null-allowed input was rejected: %+v", receipt)
	}
	value, ok := receipt.SQLParameters["id"]
	if !ok || value != nil {
		t.Fatalf("explicit null was not bound as SQL NULL: %+v", receipt.SQLParameters)
	}
}

func TestLoadRejectsTrailingJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), "request.json")
	raw := []byte(`{"schema":"tei.reference.query-input-nullability-request/v0","id":"r","inputs":{}} {}`)
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadRequest(path); err == nil {
		t.Fatal("trailing JSON value was accepted")
	}
}
