package realization

import (
	"strings"
	"testing"

	"tei/internal/database"
	"tei/internal/environment"
	"tei/internal/query"
)

func contractQuerySQLFixture() (ContractQuerySQLRealization, database.QueryContract) {
	realization := ContractQuerySQLRealization{
		Target: ContractQueryTarget{
			Database: "master_data",
			Contract: "master_data.customer",
			Query:    "find_by_id",
		},
		Provider: environment.ProviderOracle,
		Source:   "select CUSTOMER_ID, NAME from CUSTOMER_MASTER where CUSTOMER_ID = :customer_id",
		Inputs: []ContractQueryInputParameterMapping{
			{Input: "id", Parameter: "customer_id"},
		},
		ResultColumns: []query.ResultColumnMapping{
			{Column: "CUSTOMER_ID", Attribute: "customer.customer.customer_id"},
			{Column: "NAME", Attribute: "customer.customer.name"},
		},
	}
	contract := database.QueryContract{
		Name: "find_by_id",
		Inputs: []database.QueryInput{
			{Name: "id", Type: "customer.customer_id"},
		},
		Result:            "customer.customer",
		ResultCardinality: database.QueryResultZeroOrOne,
	}
	return realization, contract
}

func TestContractQuerySQLRealizationKeepsPhysicalNamesOutsideContractSignature(t *testing.T) {
	r, contract := contractQuerySQLFixture()
	if err := r.ValidateContract(contract); err != nil {
		t.Fatalf("明示したSQL realizationをContract Query signatureへ接続できません: %v", err)
	}
	if r.Identity() != "master_data/master_data.customer/find_by_id/sql/oracle" {
		t.Fatalf("SQL realization identityが期待値と異なります: %s", r.Identity())
	}
	if r.Inputs[0].Input != "id" || r.Inputs[0].Parameter != "customer_id" {
		t.Fatalf("Contract inputと物理parameterを別identityとして保持できていません: %#v", r.Inputs)
	}
}

func TestContractQuerySQLRealizationDoesNotInferParameterFromSameName(t *testing.T) {
	r, contract := contractQuerySQLFixture()
	r.Inputs = nil

	err := r.ValidateContract(contract)
	if err == nil || !strings.Contains(err.Error(), "input idのparameter mappingがありません") {
		t.Fatalf("Contract input名から物理parameterを暗黙に補完しています: %v", err)
	}
}

func TestContractQuerySQLRealizationRejectsUnknownInput(t *testing.T) {
	r, contract := contractQuerySQLFixture()
	r.Inputs[0].Input = "customer_id"

	err := r.ValidateContract(contract)
	if err == nil || !strings.Contains(err.Error(), "signatureにないinput customer_id") {
		t.Fatalf("Contractに存在しないinput mappingを受け入れています: %v", err)
	}
}

func TestContractQuerySQLRealizationRejectsDuplicatePhysicalParameter(t *testing.T) {
	r, contract := contractQuerySQLFixture()
	contract.Inputs = append(contract.Inputs, database.QueryInput{Name: "tenant", Type: "customer.tenant_id"})
	r.Inputs = append(r.Inputs, ContractQueryInputParameterMapping{Input: "tenant", Parameter: "customer_id"})

	err := r.ValidateContract(contract)
	if err == nil || !strings.Contains(err.Error(), "物理parameter \"customer_id\"が重複") {
		t.Fatalf("複数inputを同じ物理parameterへ曖昧に対応付けています: %v", err)
	}
}

func TestContractQuerySQLRealizationRejectsDuplicateResultAttribute(t *testing.T) {
	r, contract := contractQuerySQLFixture()
	r.ResultColumns[1].Attribute = "customer.customer.customer_id"

	err := r.ValidateContract(contract)
	if err == nil || !strings.Contains(err.Error(), "Semantic attribute customer.customer.customer_idのmappingが重複") {
		t.Fatalf("複数raw列を同じSemantic attributeへ曖昧に対応付けています: %v", err)
	}
}

func TestContractQuerySQLRealizationRejectsMissingResultMapping(t *testing.T) {
	r, contract := contractQuerySQLFixture()
	r.ResultColumns = nil

	err := r.ValidateContract(contract)
	if err == nil || !strings.Contains(err.Error(), "result column mappingがありません") {
		t.Fatalf("result mappingなしで物理列からSemantic rowを推測しようとしています: %v", err)
	}
}

func TestContractQuerySQLRealizationRejectsUnknownProvider(t *testing.T) {
	r, contract := contractQuerySQLFixture()
	r.Provider = environment.DatabaseProvider("unknown-db")

	err := r.ValidateContract(contract)
	if err == nil || !strings.Contains(err.Error(), "Database provider \"unknown-db\"は未定義") {
		t.Fatalf("未定義providerのSQL realizationを受け入れています: %v", err)
	}
}

func TestContractQuerySQLRealizationMapsRuntimeInputToExplicitPhysicalParameter(t *testing.T) {
	r, _ := contractQuerySQLFixture()

	mapped, err := r.MapRuntimeParameters(map[string]any{"id": int64(42)})
	if err != nil {
		t.Fatalf("Contract input値を物理parameterへ対応付けられません: %v", err)
	}
	if len(mapped) != 1 || mapped["customer_id"] != int64(42) {
		t.Fatalf("物理parameter名またはruntime値が変わっています: %#v", mapped)
	}
	if _, exists := mapped["id"]; exists {
		t.Fatalf("Contract input名をそのまま物理parameter名として利用しています: %#v", mapped)
	}
}

func TestContractQuerySQLRealizationRejectsMissingRuntimeInput(t *testing.T) {
	r, _ := contractQuerySQLFixture()

	_, err := r.MapRuntimeParameters(map[string]any{})
	if err == nil || !strings.Contains(err.Error(), "input idのruntime値がありません") {
		t.Fatalf("不足したruntime inputをSQL parameter名から補完しています: %v", err)
	}
}

func TestContractQuerySQLRealizationRejectsUnexpectedRuntimeInput(t *testing.T) {
	r, _ := contractQuerySQLFixture()

	_, err := r.MapRuntimeParameters(map[string]any{"id": int64(42), "tenant": "north"})
	if err == nil || !strings.Contains(err.Error(), "runtime input tenantのparameter mappingがありません") {
		t.Fatalf("未定義runtime inputを黙ってSQLへ渡しています: %v", err)
	}
}
