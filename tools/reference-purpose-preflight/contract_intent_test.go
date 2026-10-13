package main

import (
	"bytes"
	"path/filepath"
	"testing"

	"tei/internal/application"
	"tei/internal/database"
	"tei/internal/ir"
)

func TestRealReferenceContractRequirementDoesNotProveProvider(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "workspace")
	assessment, err := InspectWorkspace(root, "order_entry", "application-pattern", "order_entry.customer_reference")
	if err != nil {
		t.Fatal(err)
	}
	c := assessment.ContractQueryUse
	if c == nil || c.Status != "declared-requirement" || c.DatasourceAlias != "master" ||
		c.TargetDatabase != "master_data" || c.Contract != "master_data.customer" ||
		c.Query != "find_by_id" || c.CompatibleMajor != 2 {
		t.Fatalf("actual explicit Contract Query requirement was not recovered: %+v", assessment)
	}
	if assessment.RuleMeaning != "not-established" || assessment.ExecutionAllowed {
		t.Fatalf("an authored requirement became business acceptance or execution: %+v", assessment)
	}
	projection, err := ProjectForPurpose(assessment, "execute-preflight")
	if err != nil {
		t.Fatal(err)
	}
	ob, ok := purposeRequirementByName(projection, "provider-contract")
	if !ok || ob.Status != "conditional" ||
		ob.Reason != "declared-query-dependency-provider-qualification-unverified" ||
		projection.ExecutionAllowed {
		t.Fatalf("the Contract Query declaration wrongly established Provider authority: %+v", projection)
	}
}

func referenceIntentFixture() ir.World {
	return ir.World{Applications: []application.Application{{
		ID: "example",
		Databases: []application.DatabaseRequirement{{
			Alias: "master", TargetDatabase: "master_data", Access: database.AccessRead,
			RequiredContracts: []database.ContractRequirement{{
				ContractID: "master_data.customer", CompatibleMajor: 2,
			}},
		}},
		Patterns: []application.Pattern{{
			ID: "example.customer_reference", Kind: application.PatternReference,
			Datasource: "master",
			ContractQuery: &application.ContractQueryInvocation{
				Contract: "master_data.customer", Query: "find_by_id",
			},
		}},
	}}}
}

func TestContractRequirementMustBeExactAndUnambiguous(t *testing.T) {
	world := referenceIntentFixture()
	assertStatus := func(name string, expected string) {
		t.Helper()
		got := ObserveDeclaredContractQuery(world, "example", "example.customer_reference")
		if got.Status != expected {
			t.Fatalf("%s: got %+v, expected %s", name, got, expected)
		}
	}
	assertStatus("matching requirement", "declared-requirement")
	world.Applications[0].Databases = append(world.Applications[0].Databases, world.Applications[0].Databases[0])
	assertStatus("duplicate datasource", "unresolved")
	world = referenceIntentFixture()
	world.Applications[0].Databases[0].RequiredContracts = nil
	assertStatus("no required contract", "unresolved")
	world = referenceIntentFixture()
	world.Applications[0].Patterns[0].ContractQuery.Query = ""
	assertStatus("incomplete query", "unresolved")
	world = referenceIntentFixture()
	world.Applications[0].Patterns[0].ContractQuery = nil
	assertStatus("no query declared", "not-declared")
	world = referenceIntentFixture()
	world.Applications[0].Patterns[0].Kind = application.PatternCommand
	assertStatus("not a reference", "not-reference")
	world = referenceIntentFixture()
	world.Applications = append(world.Applications, world.Applications[0])
	assertStatus("duplicate Application", "unresolved")
}

func TestContractQueryIntentRemainsOptInForCLI(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "workspace")
	args := []string{
		"-workspace", root, "-application", "order_entry",
		"-subject-kind", "application-pattern", "-subject-id", "order_entry.customer_reference",
	}
	// Existing flat consumers must not receive the optional authoring hint.
	var legacy bytes.Buffer
	if err := runCLI(args, &legacy); err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(legacy.Bytes(), []byte("contractQueryUse")) {
		t.Fatalf("legacy CLI changed: %s", legacy.String())
	}
	var detailed bytes.Buffer
	withPurpose := append(append([]string{}, args...), "-purpose", "execute-preflight")
	if err := runCLI(withPurpose, &detailed); err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(detailed.Bytes(), []byte(`"status": "declared-requirement"`)) ||
		!bytes.Contains(detailed.Bytes(), []byte(`"executionAllowed": false`)) {
		t.Fatalf("opt-in qualified declaration or no-execution guard missing: %s", detailed.String())
	}
}
