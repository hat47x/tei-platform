package main

import (
	"encoding/json"
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

func TestEvidenceArtifactSHA256IdentifiesExactBytes(t *testing.T) {
	got := evidenceArtifactSHA256([]byte("abc"))
	const want = "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"
	if got != want {
		t.Fatalf("SHA-256=%s want=%s", got, want)
	}
	if evidenceArtifactSHA256([]byte("abc\n")) == got {
		t.Fatal("Evidence artifactのexact bytesが変わっているのに同じfingerprintになっています")
	}
}

func TestAcceptanceReceiptDoesNotEmbedPhysicalEvidenceBody(t *testing.T) {
	receipt := acceptanceReceipt{
		EvidenceNature:         acceptanceEvidenceNature,
		RepositoryRevision:     "abc123",
		WorkingTreeClean:       true,
		ReproducibleRevision:   true,
		EvidenceArtifactSHA256: evidenceArtifactSHA256([]byte("evidence")),
		Target:                 "master_data/master_data.customer/find_by_id",
		Provider:               "oracle",
		Accepted:               true,
	}
	raw, err := json.Marshal(receipt)
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	for _, forbidden := range []string{"object", "provenance", "readOnly", "evidencePath", "physicalType"} {
		if strings.Contains(text, `"`+forbidden+`"`) {
			t.Fatalf("acceptance receiptへexternal Evidence本文のfield %qを複製しています: %s", forbidden, text)
		}
	}
}
