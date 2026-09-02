package validation

import (
	"testing"

	"tei/internal/application"
	"tei/internal/database"
	"tei/internal/ir"
	"tei/internal/semantic"
)

func TestValidateReferenceContractQueryCrossLayerAcceptsExplicitThreeLayerContract(t *testing.T) {
	world := referenceContractQueryCrossWorld()

	if got := ValidateReferenceContractQueryCrossLayer(world); len(got) != 0 {
		t.Fatalf("三層が明示的に整合したreferenceを拒否しています: %#v", got)
	}
}

func TestValidateReferenceContractQueryCrossLayerRequiresRelationKey(t *testing.T) {
	world := referenceContractQueryCrossWorld()
	world.Semantics[0].Data[0].Relations[0].Keys = nil

	diagnostics := ValidateReferenceContractQueryCrossLayer(world)
	if !hasReferenceCrossDiagnostic(diagnostics, "REFERENCE_CONTRACT_QUERY_RELATION_KEY_REQUIRED") {
		t.Fatalf("Relation keyなしでContract Query traversalを許可しています: %#v", diagnostics)
	}
}

func TestValidateReferenceContractQueryCrossLayerRequiresTypedSignature(t *testing.T) {
	world := referenceContractQueryCrossWorld()
	world.Databases[0].Contracts[0].Queries[0] = database.QueryContract{Name: "find_by_id"}

	diagnostics := ValidateReferenceContractQueryCrossLayer(world)
	if !hasReferenceCrossDiagnostic(diagnostics, "REFERENCE_CONTRACT_QUERY_SIGNATURE_REQUIRED") {
		t.Fatalf("name-only Contract Queryをreference traversalへ利用できています: %#v", diagnostics)
	}
}

func TestValidateReferenceContractQueryCrossLayerRejectsNonKeyTargetBinding(t *testing.T) {
	world := referenceContractQueryCrossWorld()
	world.Applications[0].Patterns[0].ContractQuery.Inputs[0].TargetAttribute = "customer.customer.name"

	diagnostics := ValidateReferenceContractQueryCrossLayer(world)
	if !hasReferenceCrossDiagnostic(diagnostics, "REFERENCE_CONTRACT_QUERY_BINDING_TARGET_NOT_RELATION_KEY") {
		t.Fatalf("Relation target keyではないattributeからQuery inputへのbindingを許可しています: %#v", diagnostics)
	}
	if !hasReferenceCrossDiagnostic(diagnostics, "REFERENCE_CONTRACT_QUERY_RELATION_KEY_NOT_BOUND") {
		t.Fatalf("本来必要なRelation target keyのbinding欠如を検出していません: %#v", diagnostics)
	}
}

func TestValidateReferenceContractQueryCrossLayerRequiresAllQueryInputsToBeBound(t *testing.T) {
	world := referenceContractQueryCrossWorld()
	world.Databases[0].Contracts[0].Queries[0].Inputs = append(
		world.Databases[0].Contracts[0].Queries[0].Inputs,
		database.QueryInput{Name: "tenant", Type: "customer.customer_id"},
	)

	diagnostics := ValidateReferenceContractQueryCrossLayer(world)
	if !hasReferenceCrossDiagnostic(diagnostics, "REFERENCE_CONTRACT_QUERY_INPUT_NOT_BOUND") {
		t.Fatalf("値の供給元がないQuery inputを許可しています: %#v", diagnostics)
	}
}

func TestValidateReferenceContractQueryCrossLayerRejectsUnknownBoundInput(t *testing.T) {
	world := referenceContractQueryCrossWorld()
	world.Applications[0].Patterns[0].ContractQuery.Inputs[0].Input = "customer"

	diagnostics := ValidateReferenceContractQueryCrossLayer(world)
	if !hasReferenceCrossDiagnostic(diagnostics, "REFERENCE_CONTRACT_QUERY_INPUT_NOT_IN_SIGNATURE") {
		t.Fatalf("signatureにないQuery inputへのbindingを許可しています: %#v", diagnostics)
	}
}

func TestValidateReferenceContractQueryCrossLayerRejectsInputTypeMismatch(t *testing.T) {
	world := referenceContractQueryCrossWorld()
	world.Databases[0].Contracts[0].Queries[0].Inputs[0].Type = "customer.other_id"

	diagnostics := ValidateReferenceContractQueryCrossLayer(world)
	if !hasReferenceCrossDiagnostic(diagnostics, "REFERENCE_CONTRACT_QUERY_INPUT_TYPE_MISMATCH") {
		t.Fatalf("Relation target keyとQuery inputのSemantic型不一致を許可しています: %#v", diagnostics)
	}
}

func TestValidateReferenceContractQueryCrossLayerRejectsResultDataMismatch(t *testing.T) {
	world := referenceContractQueryCrossWorld()
	world.Databases[0].Contracts[0].Queries[0].Result = "customer.other"

	diagnostics := ValidateReferenceContractQueryCrossLayer(world)
	if !hasReferenceCrossDiagnostic(diagnostics, "REFERENCE_CONTRACT_QUERY_RESULT_DATA_MISMATCH") {
		t.Fatalf("Relation targetと異なるQuery result Dataを許可しています: %#v", diagnostics)
	}
}

func TestValidateReferenceContractQueryCrossLayerAcceptsZeroOrOneForOneRelation(t *testing.T) {
	world := referenceContractQueryCrossWorld()
	world.Databases[0].Contracts[0].Queries[0].ResultCardinality = database.QueryResultZeroOrOne

	if got := ValidateReferenceContractQueryCrossLayer(world); len(got) != 0 {
		t.Fatalf("external-observed readで許容するzero-or-oneをone Relationと矛盾扱いしています: %#v", got)
	}
}

func TestValidateReferenceContractQueryCrossLayerRejectsManyForOneRelation(t *testing.T) {
	world := referenceContractQueryCrossWorld()
	world.Databases[0].Contracts[0].Queries[0].ResultCardinality = database.QueryResultMany

	diagnostics := ValidateReferenceContractQueryCrossLayer(world)
	if !hasReferenceCrossDiagnostic(diagnostics, "REFERENCE_CONTRACT_QUERY_RESULT_CARDINALITY_MISMATCH") {
		t.Fatalf("one Relationにmany resultを返すQueryを許可しています: %#v", diagnostics)
	}
}

func referenceContractQueryCrossWorld() ir.World {
	customerIDType := semantic.Type{ID: "customer.customer_id", Symbol: "CustomerId", Kind: semantic.TypeAlias, Underlying: "std.identifier"}
	otherIDType := semantic.Type{ID: "customer.other_id", Symbol: "OtherId", Kind: semantic.TypeAlias, Underlying: "std.identifier"}

	order := semantic.Data{
		ID:     "sales.order",
		Symbol: "Order",
		Attributes: []semantic.Attribute{
			{ID: "sales.order.customer_id", Symbol: "customerId", TypeRef: "customer.customer_id"},
		},
		Relations: []semantic.Relation{
			{
				ID:          "sales.order.customer",
				Symbol:      "customer",
				Target:      "customer.customer",
				Cardinality: semantic.CardinalityOne,
				Keys: []semantic.RelationKey{
					{SourceAttribute: "sales.order.customer_id", TargetAttribute: "customer.customer.customer_id"},
				},
			},
		},
	}
	customer := semantic.Data{
		ID:     "customer.customer",
		Symbol: "Customer",
		Attributes: []semantic.Attribute{
			{ID: "customer.customer.customer_id", Symbol: "customerId", TypeRef: "customer.customer_id"},
			{ID: "customer.customer.name", Symbol: "name", TypeRef: "customer.other_id"},
		},
	}
	other := semantic.Data{ID: "customer.other", Symbol: "Other"}

	return ir.World{
		Semantics: []semantic.Namespace{
			{
				ID:    "customer",
				Types: []semantic.Type{customerIDType, otherIDType},
				Data:  []semantic.Data{order, customer, other},
			},
		},
		Applications: []application.Application{
			{
				ID: "order_entry",
				Databases: []application.DatabaseRequirement{
					{
						Alias:          "master",
						TargetDatabase: "master_data",
						Access:         database.AccessRead,
						RequiredContracts: []database.ContractRequirement{
							{ContractID: "master_data.customer", CompatibleMajor: 2},
						},
					},
				},
				Patterns: []application.Pattern{
					{
						ID:         "order_entry.customer_reference",
						Kind:       application.PatternReference,
						Relation:   "sales.order.customer",
						Datasource: "master",
						ContractQuery: &application.ContractQueryInvocation{
							Contract: "master_data.customer",
							Query:    "find_by_id",
							Inputs: []application.ContractQueryInputBinding{
								{TargetAttribute: "customer.customer.customer_id", Input: "id"},
							},
						},
					},
				},
			},
		},
		Databases: []database.Database{
			{
				ID: "master_data",
				Contracts: []database.Contract{
					{
						ID:      "master_data.customer",
						Version: database.ContractVersion{Major: 2, Minor: 4},
						Queries: []database.QueryContract{
							{
								Name: "find_by_id",
								Inputs: []database.QueryInput{
									{Name: "id", Type: "customer.customer_id"},
								},
								Result:            "customer.customer",
								ResultCardinality: database.QueryResultZeroOrOne,
							},
						},
					},
				},
			},
		},
	}
}

func hasReferenceCrossDiagnostic(diagnostics []Diagnostic, code string) bool {
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == code {
			return true
		}
	}
	return false
}
