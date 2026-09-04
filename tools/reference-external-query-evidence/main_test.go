package main

import (
	"strings"
	"testing"

	"tei/internal/database/externalqueryevidence"
	"tei/internal/environment"
	"tei/internal/realization"
)

func TestFindRealizationDoesNotInferMissingDefinitionFromEvidence(t *testing.T) {
	evidence := externalqueryevidence.Evidence{
		Target: realization.ContractQueryTarget{
			Database: "master_data",
			Contract: "master_data.customer",
			Query:    "find_by_id",
		},
		Provider: environment.ProviderOracle,
	}

	_, err := findRealization(nil, evidence)
	if err == nil || !strings.Contains(err.Error(), "Evidenceからrealizationを生成・補完しません") {
		t.Fatalf("SQL realizationがないのにEvidenceから物理実装を補完しています: %v", err)
	}
}

func TestFindRealizationRequiresExactTargetAndProvider(t *testing.T) {
	evidence := externalqueryevidence.Evidence{
		Target: realization.ContractQueryTarget{
			Database: "master_data",
			Contract: "master_data.customer",
			Query:    "find_by_id",
		},
		Provider: environment.ProviderOracle,
	}
	definitions := []realization.ContractQuerySQLRealization{
		{Target: evidence.Target, Provider: environment.ProviderPostgres},
		{
			Target: realization.ContractQueryTarget{
				Database: "master_data",
				Contract: "master_data.customer",
				Query:    "other_query",
			},
			Provider: environment.ProviderOracle,
		},
		{Target: evidence.Target, Provider: environment.ProviderOracle},
	}

	found, err := findRealization(definitions, evidence)
	if err != nil {
		t.Fatalf("完全一致するSQL realizationを選べません: %v", err)
	}
	if found.Target != evidence.Target || found.Provider != evidence.Provider {
		t.Fatalf("target/provider以外からSQL realizationを選んでいます: %#v", found)
	}
}
