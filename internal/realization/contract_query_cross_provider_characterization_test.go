package realization

import (
	"strings"
	"testing"

	"tei/internal/environment"
)

// This is a structural characterization of the existing Contract Query
// boundary, not an SQL-file selector, SQL execution, or DB conformance test.
// A common SQL body can be supplied as two explicit, provider-bound
// realizations without changing the semantic Query identity.
func TestContractQueryCommonSQLRequiresProviderSpecificRealizations(t *testing.T) {
	base, contract := contractQuerySQLFixture()
	base.Provider = environment.ProviderPostgres
	base.Source = "select customer_id, name from customer_master where customer_id = :customer_id"

	second := base
	second.Provider = environment.ProviderSQLite

	for name, r := range map[string]ContractQuerySQLRealization{
		"postgres": base,
		"sqlite":   second,
	} {
		t.Run(name, func(t *testing.T) {
			if err := r.ValidateContract(contract); err != nil {
				t.Fatalf("explicit provider realization must validate: %v", err)
			}
			got, err := r.MapRuntimeParameters(map[string]any{"id": int64(42)})
			if err != nil {
				t.Fatalf("explicit input mapping failed: %v", err)
			}
			if len(got) != 1 || got["customer_id"] != int64(42) {
				t.Fatalf("semantic input did not map to the expected parameter: %#v", got)
			}
		})
	}

	if base.Target != second.Target {
		t.Fatalf("provider switch changed the canonical target")
	}
	if base.Identity() == second.Identity() {
		t.Fatalf("different providers must have distinct realization identities")
	}
	if base.Source != second.Source {
		t.Fatalf("fixture no longer shares the same SQL text")
	}
}

// Contract validation is performed independently for each provider-bound
// realization. Merely sharing SQL bytes does not confer input/output mapping
// correctness on an unvalidated provider.
func TestContractQuerySharedSQLDoesNotInheritOtherProviderMapping(t *testing.T) {
	base, contract := contractQuerySQLFixture()
	base.Provider = environment.ProviderPostgres
	second := base
	second.Provider = environment.ProviderSQLite
	second.Inputs = nil

	if err := base.ValidateContract(contract); err != nil {
		t.Fatalf("baseline provider realization should validate: %v", err)
	}
	err := second.ValidateContract(contract)
	if err == nil || !strings.Contains(err.Error(), "parameter mapping") {
		t.Fatalf("missing provider-specific input mapping was accepted: %v", err)
	}
}

// A realization identity names the logical target and provider, not the SQL
// bytes. SQL source revisions therefore need separate provenance/invalidation.
func TestContractQuerySQLBodyRevisionDoesNotRenameRealization(t *testing.T) {
	before, contract := contractQuerySQLFixture()
	after := before
	after.Source = "select CUSTOMER_ID, NAME from CUSTOMER_MASTER where CUSTOMER_ID = :customer_id and ACTIVE = 1"

	if err := before.ValidateContract(contract); err != nil {
		t.Fatalf("baseline SQL realization should validate: %v", err)
	}
	if err := after.ValidateContract(contract); err != nil {
		t.Fatalf("revised SQL realization should validate structurally: %v", err)
	}
	if before.Identity() != after.Identity() {
		t.Fatalf("SQL text change unexpectedly renamed the canonical realization")
	}
	if before.Source == after.Source {
		t.Fatalf("fixture did not change the SQL bytes")
	}
}
