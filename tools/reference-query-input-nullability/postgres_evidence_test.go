package queryinputnullability

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestPostgresEvidenceAgainstEphemeralDatabase(t *testing.T) {
	dsn := os.Getenv("TEI_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("TEI_TEST_POSTGRES_DSN is not set")
	}
	contractPath := filepath.Join("testdata", "contract.json")
	receipt, err := RunPostgresEvidence(context.Background(), dsn, contractPath)
	if err != nil {
		t.Fatal(err)
	}
	if !receipt.Passed {
		t.Fatalf("postgres evidence failed: %+v", receipt)
	}
	if receipt.EvidenceNature != "real-postgres-ephemeral-fixture" || receipt.DatabaseProvider != "postgres" {
		t.Fatalf("unexpected evidence nature: %+v", receipt)
	}
	byName := map[string]PostgresCase{}
	for _, caseReceipt := range receipt.Cases {
		byName[caseReceipt.Name] = caseReceipt
	}
	if len(byName) != 5 {
		t.Fatalf("unexpected case count: %d", len(byName))
	}
	for _, name := range []string{"explicit-null-allowed", "omitted-optional"} {
		caseReceipt := byName[name]
		if caseReceipt.ObservedNull == nil || !*caseReceipt.ObservedNull {
			t.Fatalf("%s did not observe SQL NULL: %+v", name, caseReceipt)
		}
	}
	empty := byName["empty-string-is-value"]
	if empty.ObservedNull == nil || *empty.ObservedNull || empty.ObservedValue == nil || *empty.ObservedValue != "" {
		t.Fatalf("empty string was not preserved as a value: %+v", empty)
	}
	for _, name := range []string{"forbidden-null", "missing-required"} {
		caseReceipt := byName[name]
		if !caseReceipt.QueryNotExecuted || len(caseReceipt.Rejected) == 0 {
			t.Fatalf("%s was not rejected before SQL: %+v", name, caseReceipt)
		}
	}
}
