package main

import (
	"os"
	"path/filepath"
	"testing"

	"tei/internal/database"
	"tei/internal/workspace"
)

func TestCrossApplicationAccessProbeUsesPatternOwningApplication(t *testing.T) {
	root := repoPath()
	probeRoot := prepareProcedureAuthoringWorkspace(t, root)
	addSyntheticSecondApplicationWithWritableMaster(t, probeRoot)

	loaded, err := workspace.Load(probeRoot)
	if err != nil {
		t.Fatal(err)
	}
	f, err := readFixture(filepath.Join(root, "reference", "business-system-design", "order-entry-business-procedure.json"))
	if err != nil {
		t.Fatal(err)
	}
	f.Steps = append(f.Steps, step{
		ID:             "BP-XACCESS",
		BusinessAction: "update customer information from the fulfillment application context",
		Realization:    "application-use",
		ApplicationPatterns: []string{
			"fulfillment_console.customer_reference",
		},
	})

	result := evaluate(loaded.World, f)
	applyAccessProbe(&result, loaded.World, f, accessProbe{
		SchemaVersion:      probeSchemaVersion,
		Procedure:          f.Procedure.ID,
		Step:               "BP-XACCESS",
		ApplicationPattern: "fulfillment_console.customer_reference",
		RequestedAccess:    accessIntent{Required: database.AccessWrite},
		Reason:             "the owning fulfillment application explicitly accepts read-write access to master",
	})

	for _, item := range result.Diagnostics {
		if item.Step == "BP-XACCESS" {
			t.Fatalf("cross-Application access probeをprimary fixture Applicationのpolicyで判定しました: %+v", item)
		}
	}
}

func TestCrossApplicationAccessProbeStillEnforcesOwningApplicationPolicy(t *testing.T) {
	root := repoPath()
	probeRoot := prepareProcedureAuthoringWorkspace(t, root)
	addSyntheticSecondApplicationWithWritableMaster(t, probeRoot)

	loaded, err := workspace.Load(probeRoot)
	if err != nil {
		t.Fatal(err)
	}
	f, err := readFixture(filepath.Join(root, "reference", "business-system-design", "order-entry-business-procedure.json"))
	if err != nil {
		t.Fatal(err)
	}
	f.Steps = append(f.Steps, step{
		ID:             "BP-XACCESS",
		BusinessAction: "update customer information from the fulfillment application context",
		Realization:    "application-use",
		ApplicationPatterns: []string{
			"fulfillment_console.customer_reference",
		},
	})

	// Tighten only the owning second Application to read. If the checker truly
	// resolves policy from the Pattern owner, the same write probe must now
	// produce the expected access conflict.
	writeSyntheticSecondApplication(t, probeRoot, "read")
	loaded, err = workspace.Load(probeRoot)
	if err != nil {
		t.Fatal(err)
	}
	result := evaluate(loaded.World, f)
	applyAccessProbe(&result, loaded.World, f, accessProbe{
		SchemaVersion:      probeSchemaVersion,
		Procedure:          f.Procedure.ID,
		Step:               "BP-XACCESS",
		ApplicationPattern: "fulfillment_console.customer_reference",
		RequestedAccess:    accessIntent{Required: database.AccessWrite},
		Reason:             "write must be checked against the Pattern owning Application",
	})

	found := false
	for _, item := range result.Diagnostics {
		if item.Step == "BP-XACCESS" && item.Code == "BUSINESS_FUNCTION_DATASOURCE_ACCESS_CONFLICT" {
			found = true
		}
	}
	if !found {
		t.Fatalf("Pattern owning Applicationのread-only policyをcross-Application access probeへ適用できません: %+v", result.Diagnostics)
	}
}

func addSyntheticSecondApplicationWithWritableMaster(t *testing.T, workspaceRoot string) {
	t.Helper()
	writeSyntheticSecondApplication(t, workspaceRoot, "read-write")
}

func writeSyntheticSecondApplication(t *testing.T, workspaceRoot, access string) {
	t.Helper()
	unit := filepath.Join(workspaceRoot, "applications", "fulfillment-console")
	patterns := filepath.Join(unit, "patterns")
	if err := os.MkdirAll(patterns, 0o755); err != nil {
		t.Fatal(err)
	}
	applicationSource := `application fulfillment_console {
    uses semantic sales
    uses semantic customer

    database master {
        target master_data
        access ` + access + `
        requires {
            contract master_data.customer ^2
        }
    }
}
`
	if err := os.WriteFile(filepath.Join(unit, "application.tei"), []byte(applicationSource), 0o600); err != nil {
		t.Fatal(err)
	}
	patternSource := `pattern fulfillment_console.customer_reference reference {
    relation sales.order.customer
    datasource master
    contract-query {
        contract master_data.customer
        query find_by_id
        input customer.customer.customer_id -> id
    }
}
`
	if err := os.WriteFile(filepath.Join(patterns, "customer-reference.tei"), []byte(patternSource), 0o600); err != nil {
		t.Fatal(err)
	}
}
