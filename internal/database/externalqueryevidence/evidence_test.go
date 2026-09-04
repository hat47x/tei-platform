package externalqueryevidence

import (
	"strings"
	"testing"

	"tei/internal/database"
	"tei/internal/environment"
	"tei/internal/ir"
	"tei/internal/query"
	"tei/internal/realization"
	"tei/internal/sourceindex"
)

func TestValidateAgainstAcceptsExplicitExternalObservedEvidence(t *testing.T) {
	world := testWorld(database.OwnershipExternalObserved)
	realizationDef := testRealization()
	evidence := testEvidence(realizationDef)

	if err := ValidateAgainst(world, realizationDef, evidence); err != nil {
		t.Fatalf("明示されたexternal-observed Evidenceを受け入れられません: %v", err)
	}
}

func TestValidateAgainstRejectsManagedHereDatabase(t *testing.T) {
	world := testWorld(database.OwnershipManagedHere)
	realizationDef := testRealization()
	err := ValidateAgainst(world, realizationDef, testEvidence(realizationDef))
	if err == nil || !strings.Contains(err.Error(), "external-observedではない") {
		t.Fatalf("managed-here Databaseをexternal Evidence対象として受け入れています: %v", err)
	}
}

func TestValidateAgainstRejectsPhysicalParameterMismatchWithoutInference(t *testing.T) {
	world := testWorld(database.OwnershipExternalObserved)
	realizationDef := testRealization()
	evidence := testEvidence(realizationDef)
	evidence.Inputs[0].Parameter = "id"

	err := ValidateAgainst(world, realizationDef, evidence)
	if err == nil || !strings.Contains(err.Error(), "physical parameter") {
		t.Fatalf("観測parameterとauthoringの不一致を名前一致等で補っています: %v", err)
	}
}

func TestValidateAgainstRejectsRawResultMappingMismatchWithoutInference(t *testing.T) {
	world := testWorld(database.OwnershipExternalObserved)
	realizationDef := testRealization()
	evidence := testEvidence(realizationDef)
	evidence.Results[0].Attribute = "customer.customer.name"

	err := ValidateAgainst(world, realizationDef, evidence)
	if err == nil || !strings.Contains(err.Error(), "Semantic attribute") {
		t.Fatalf("観測raw result mappingとauthoringの不一致を補っています: %v", err)
	}
}

func TestValidateAgainstRejectsStaleNativeSQLFingerprint(t *testing.T) {
	world := testWorld(database.OwnershipExternalObserved)
	realizationDef := testRealization()
	evidence := testEvidence(realizationDef)
	evidence.NativeSource.Fingerprint = sourceindex.Fingerprint([]byte("different SQL"))

	err := ValidateAgainst(world, realizationDef, evidence)
	if err == nil || !strings.Contains(err.Error(), "fingerprintがstale") {
		t.Fatalf("別native SQLを確認したEvidenceを現在のrealizationへ流用しています: %v", err)
	}
}

func TestValidateAgainstRequiresPhysicalProvenance(t *testing.T) {
	world := testWorld(database.OwnershipExternalObserved)
	realizationDef := testRealization()
	evidence := testEvidence(realizationDef)
	evidence.Provenance.SourceRef = ""

	err := ValidateAgainst(world, realizationDef, evidence)
	if err == nil || !strings.Contains(err.Error(), "source provenance") {
		t.Fatalf("provenanceのない物理情報を受け入れています: %v", err)
	}
}

func TestValidateAgainstRequiresEffectiveReadOnlyEvidence(t *testing.T) {
	world := testWorld(database.OwnershipExternalObserved)
	realizationDef := testRealization()
	evidence := testEvidence(realizationDef)
	evidence.ReadOnly = ReadOnlyProof{}

	err := ValidateAgainst(world, realizationDef, evidence)
	if err == nil || !strings.Contains(err.Error(), "read-only保証Evidence") {
		t.Fatalf("実効的read-only根拠のない外部Queryを受け入れています: %v", err)
	}
}

func TestValidateAgainstRejectsProviderMismatch(t *testing.T) {
	world := testWorld(database.OwnershipExternalObserved)
	realizationDef := testRealization()
	evidence := testEvidence(realizationDef)
	evidence.Provider = environment.ProviderPostgres

	err := ValidateAgainst(world, realizationDef, evidence)
	if err == nil || !strings.Contains(err.Error(), "provider") {
		t.Fatalf("観測providerとrealization providerの不一致を受け入れています: %v", err)
	}
}

func testWorld(ownership database.Ownership) ir.World {
	return ir.World{Databases: []database.Database{{
		ID:        "master_data",
		Ownership: ownership,
		Contracts: []database.Contract{{
			ID: "master_data.customer",
			Queries: []database.QueryContract{{
				Name: "find_by_id",
				Inputs: []database.QueryInput{{
					Name: "id",
					Type: "std.int64",
				}},
				Result:            "customer.customer",
				ResultCardinality: database.QueryResultZeroOrOne,
			}},
		}},
	}}}
}

func testRealization() realization.ContractQuerySQLRealization {
	return realization.ContractQuerySQLRealization{
		Target: realization.ContractQueryTarget{
			Database: "master_data",
			Contract: "master_data.customer",
			Query:    "find_by_id",
		},
		Provider: environment.ProviderOracle,
		Source:   "select observed_customer_id from observed_customer_view where observed_customer_id = :observed_id",
		Inputs: []realization.ContractQueryInputParameterMapping{{
			Input:     "id",
			Parameter: "observed_id",
		}},
		ResultColumns: []query.ResultColumnMapping{{
			Column:    "observed_customer_id",
			Attribute: "customer.customer.customer_id",
		}},
	}
}

func testEvidence(realizationDef realization.ContractQuerySQLRealization) Evidence {
	return Evidence{
		Target:   realizationDef.Target,
		Provider: environment.ProviderOracle,
		Object: PhysicalObject{
			Kind:     "view",
			Identity: "observed_customer_view",
		},
		Inputs: []InputObservation{{
			Input:        "id",
			Parameter:    "observed_id",
			PhysicalType: "NUMBER",
		}},
		Results: []ResultObservation{{
			Column:       "observed_customer_id",
			Attribute:    "customer.customer.customer_id",
			PhysicalType: "NUMBER",
		}},
		NativeSource: NativeSourceProof{
			EvidenceRef: "test-evidence://native-query-source",
			Fingerprint: sourceindex.Fingerprint([]byte(realizationDef.Source)),
		},
		Provenance: Provenance{
			SourceRef:  "test-evidence://schema-observation",
			ObservedAt: "2026-09-04T00:00:00Z",
			Processing: "synthetic-test-only",
		},
		ReadOnly: ReadOnlyProof{
			Mechanism:   "database-read-only-principal",
			EvidenceRef: "test-evidence://read-only-principal",
		},
	}
}
