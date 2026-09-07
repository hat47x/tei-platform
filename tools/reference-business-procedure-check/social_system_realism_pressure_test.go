package main

import (
	"path/filepath"
	"reflect"
	"testing"

	"tei/internal/application"
	"tei/internal/database"
	"tei/internal/ir"
	"tei/internal/workspace"
)

func TestOrderEntryReferenceExercisesSocialSystemBoundaryWithoutSyntheticSecondApplication(t *testing.T) {
	root := repoPath()
	loaded, err := workspace.Load(filepath.Join(root, "reference", "workspace"))
	if err != nil {
		t.Fatal(err)
	}
	f, err := readFixture(filepath.Join(root, "reference", "business-system-design", "order-entry-business-procedure.json"))
	if err != nil {
		t.Fatal(err)
	}

	result := evaluate(loaded.World, f)
	if err := checkExpectedDiagnostics(result.Diagnostics, ""); err != nil {
		t.Fatalf("current order_entry Business Procedureにblocking cross-view diagnosticがあります: %v\n%+v", err, result.Diagnostics)
	}

	app, ok := findApplication(loaded.World, "order_entry")
	if !ok {
		t.Fatal("Reference Workspaceにorder_entry Applicationがありません")
	}
	if _, ok := findApplication(loaded.World, "fulfillment_console"); ok {
		t.Fatal("social-system realism Pressureがsynthetic second Applicationに依存しています")
	}

	salesReq, ok := databaseRequirementByAlias(app.Databases, "sales")
	if !ok {
		t.Fatal("order_entryのmanaged-here sales datasourceを解決できません")
	}
	if salesReq.Access != database.AccessReadWrite || string(salesReq.TargetDatabase) != "sales_core" {
		t.Fatalf("sales datasourceのaccepted ownership/access boundaryが変化しました: %+v", salesReq)
	}
	salesDB, ok := databaseByID(loaded.World, "sales_core")
	if !ok || salesDB.Ownership != database.OwnershipManagedHere {
		t.Fatalf("sales_coreをmanaged-here databaseとして確認できません: %+v found=%v", salesDB, ok)
	}

	masterReq, ok := databaseRequirementByAlias(app.Databases, "master")
	if !ok {
		t.Fatal("order_entryのexternal master datasourceを解決できません")
	}
	if masterReq.Access != database.AccessRead || string(masterReq.TargetDatabase) != "master_data" {
		t.Fatalf("master datasourceのaccepted read-only boundaryが変化しました: %+v", masterReq)
	}
	masterDB, ok := databaseByID(loaded.World, "master_data")
	if !ok || masterDB.Ownership != database.OwnershipExternalObserved {
		t.Fatalf("master_dataをexternal-observed databaseとして確認できません: %+v found=%v", masterDB, ok)
	}

	owner, customerPattern, ok := findPatternOwner(loaded.World, "order_entry.customer_reference")
	if !ok {
		t.Fatal("customer reference Patternを一意に解決できません")
	}
	if string(owner.ID) != "order_entry" || string(customerPattern.Datasource) != "master" || customerPattern.ContractQuery == nil {
		t.Fatalf("customer referenceがexternal-observed masterへのexplicit traceを失いました: owner=%s pattern=%+v", owner.ID, customerPattern)
	}

	_, submitPattern, ok := findPatternOwner(loaded.World, "order_entry.submit_order")
	if !ok || string(submitPattern.Capability) != "sales.order.submit" {
		t.Fatalf("submit Business interactionからaccepted command Capabilityをtraceできません: %+v found=%v", submitPattern, ok)
	}

	steps := stepsByID(f)
	assertResidualBoundaryStep(t, steps, "BP-04", "manual-unresolved")
	assertExplicitCapabilityBoundaryStep(t, steps, "BP-06", "security-boundary", "security.authorize")
	assertResidualBoundaryStep(t, steps, "BP-08", "observation-boundary")
	assertResidualBoundaryStep(t, steps, "BP-09", "manual-external-unresolved")
	assertApplicationUseStep(t, steps, "BP-07")

	if !stepOccursBefore(f, "BP-06", "BP-07") || !stepOccursBefore(f, "BP-07", "BP-08") || !stepOccursBefore(f, "BP-08", "BP-09") {
		t.Fatal("Authorization -> Command -> separate Observation -> manual/external continuationのBusiness順序をReferenceから確認できません")
	}
}

func databaseRequirementByAlias(requirements []application.DatabaseRequirement, alias string) (application.DatabaseRequirement, bool) {
	for _, item := range requirements {
		if string(item.Alias) == alias {
			return item, true
		}
	}
	return application.DatabaseRequirement{}, false
}

func databaseByID(world ir.World, id string) (database.Database, bool) {
	for _, item := range world.Databases {
		if string(item.ID) == id {
			return item, true
		}
	}
	return database.Database{}, false
}

func stepsByID(f fixture) map[string]step {
	result := make(map[string]step, len(f.Steps))
	for _, item := range f.Steps {
		result[item.ID] = item
	}
	return result
}

func assertResidualBoundaryStep(t *testing.T, steps map[string]step, id, realization string) {
	t.Helper()
	item, ok := steps[id]
	if !ok {
		t.Fatalf("Business Procedure step %sがありません", id)
	}
	if item.Realization != realization {
		t.Fatalf("Business Procedure step %s realization=%q want=%q", id, item.Realization, realization)
	}
	if len(item.ApplicationPatterns) != 0 || len(item.Capabilities) != 0 {
		t.Fatalf("%s residual stepへfake Application Pattern / Capabilityを要求しました: %+v", realization, item)
	}
}

func assertExplicitCapabilityBoundaryStep(t *testing.T, steps map[string]step, id, realization, capability string) {
	t.Helper()
	item, ok := steps[id]
	if !ok {
		t.Fatalf("Business Procedure step %sがありません", id)
	}
	if item.Realization != realization {
		t.Fatalf("Business Procedure step %s realization=%q want=%q", id, item.Realization, realization)
	}
	if len(item.ApplicationPatterns) != 0 {
		t.Fatalf("%s stepへApplication Patternを捏造しました: %+v", realization, item)
	}
	if !reflect.DeepEqual(item.Capabilities, []string{capability}) {
		t.Fatalf("%s stepからexisting semantic Capabilityへのexplicit refが失われました: got=%v want=%s", realization, item.Capabilities, capability)
	}
}

func assertApplicationUseStep(t *testing.T, steps map[string]step, id string) {
	t.Helper()
	item, ok := steps[id]
	if !ok {
		t.Fatalf("Business Procedure step %sがありません", id)
	}
	if item.Realization != "application-use" || len(item.ApplicationPatterns) == 0 {
		t.Fatalf("application-use step %sからexplicit Application Pattern refが失われました: %+v", id, item)
	}
}

func stepOccursBefore(f fixture, before, after string) bool {
	beforeIndex, afterIndex := -1, -1
	for i, item := range f.Steps {
		switch item.ID {
		case before:
			beforeIndex = i
		case after:
			afterIndex = i
		}
	}
	return beforeIndex >= 0 && afterIndex >= 0 && beforeIndex < afterIndex
}
