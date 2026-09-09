package main

import (
	"fmt"
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	"tei/internal/application"
	"tei/internal/database"
	"tei/internal/ir"
	"tei/internal/workspace"
)

type businessActionOwnershipBoundaryRowProbe struct {
	StepID       string             `json:"step"`
	Data         string             `json:"data"`
	Operation    crudOperationProbe `json:"operation"`
	Database     string             `json:"database"`
	Ownership    database.Ownership `json:"ownership"`
	Access       database.Access    `json:"access"`
	EvidencePath string             `json:"evidencePath"`
}

func TestBusinessActionOwnershipBoundaryProjectionUsesExistingAuthoritiesOnly(t *testing.T) {
	root := repoPath()
	loaded, err := workspace.Load(filepath.Join(root, "reference", "workspace"))
	if err != nil {
		t.Fatal(err)
	}
	f, err := readFixture(filepath.Join(root, "reference", "business-system-design", "order-entry-business-procedure.json"))
	if err != nil {
		t.Fatal(err)
	}

	rows, err := deriveBusinessActionOwnershipBoundaryProjectionProbe(loaded.World, f)
	if err != nil {
		t.Fatal(err)
	}

	want := []businessActionOwnershipBoundaryRowProbe{
		{StepID: "BP-00", Data: "sales.order", Operation: crudReadProbe, Database: "sales_core", Ownership: database.OwnershipManagedHere, Access: database.AccessReadWrite},
		{StepID: "BP-02", Data: "customer.customer", Operation: crudReadProbe, Database: "master_data", Ownership: database.OwnershipExternalObserved, Access: database.AccessRead},
		{StepID: "BP-03", Data: "sales.order_line", Operation: crudReadProbe, Database: "sales_core", Ownership: database.OwnershipManagedHere, Access: database.AccessReadWrite},
	}
	if len(rows) != len(want) {
		t.Fatalf("unexpected ownership boundary rows: got=%+v want=%+v", rows, want)
	}
	for i := range want {
		got := rows[i]
		if got.StepID != want[i].StepID || got.Data != want[i].Data || got.Operation != want[i].Operation || got.Database != want[i].Database || got.Ownership != want[i].Ownership || got.Access != want[i].Access {
			t.Fatalf("ownership boundary row mismatch at %d: got=%+v want=%+v", i, got, want[i])
		}
		if got.EvidencePath == "" {
			t.Fatalf("ownership boundary rowにevidence pathがありません: %+v", got)
		}
	}
}

func TestBusinessActionOwnershipBoundaryProjectionExposesExternalObservedReadWithoutInventingSubsystem(t *testing.T) {
	root := repoPath()
	loaded, err := workspace.Load(filepath.Join(root, "reference", "workspace"))
	if err != nil {
		t.Fatal(err)
	}
	f, err := readFixture(filepath.Join(root, "reference", "business-system-design", "order-entry-business-procedure.json"))
	if err != nil {
		t.Fatal(err)
	}

	rows, err := deriveBusinessActionOwnershipBoundaryProjectionProbe(loaded.World, f)
	if err != nil {
		t.Fatal(err)
	}
	var bp02 *businessActionOwnershipBoundaryRowProbe
	for i := range rows {
		if rows[i].StepID == "BP-02" {
			bp02 = &rows[i]
			break
		}
	}
	if bp02 == nil {
		t.Fatal("BP-02 ownership boundary rowがありません")
	}
	if bp02.Ownership != database.OwnershipExternalObserved || bp02.Access != database.AccessRead || bp02.Operation != crudReadProbe {
		t.Fatalf("BP-02 external-observed read boundaryを失いました: %+v", *bp02)
	}

	typ := reflect.TypeOf(businessActionOwnershipBoundaryRowProbe{})
	for _, forbidden := range []string{"Subsystem", "SubjectArea", "OwnerSubsystem", "BusinessArea", "CanonicalSubsystem"} {
		if _, found := typ.FieldByName(forbidden); found {
			t.Fatalf("derived ownership viewへ未証明のCanonical boundary field %qを追加しました", forbidden)
		}
	}
}

func TestBusinessActionOwnershipBoundaryProjectionKeepsNoCRUDTraceStepsUnknown(t *testing.T) {
	root := repoPath()
	loaded, err := workspace.Load(filepath.Join(root, "reference", "workspace"))
	if err != nil {
		t.Fatal(err)
	}
	f, err := readFixture(filepath.Join(root, "reference", "business-system-design", "order-entry-business-procedure.json"))
	if err != nil {
		t.Fatal(err)
	}

	rows, err := deriveBusinessActionOwnershipBoundaryProjectionProbe(loaded.World, f)
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, row := range rows {
		seen[row.StepID] = true
	}
	for _, stepID := range []string{"BP-01", "BP-04", "BP-05", "BP-06", "BP-07", "BP-08", "BP-09"} {
		if seen[stepID] {
			t.Fatalf("%sへCRUD authorityなしでownership operation rowを生成しました", stepID)
		}
	}
}

func TestBusinessActionOwnershipBoundaryProjectionFailsClosedWithCRUDReadAuthority(t *testing.T) {
	root := repoPath()
	loaded, err := workspace.Load(filepath.Join(root, "reference", "workspace"))
	if err != nil {
		t.Fatal(err)
	}
	f, err := readFixture(filepath.Join(root, "reference", "business-system-design", "order-entry-business-procedure.json"))
	if err != nil {
		t.Fatal(err)
	}

	changed := cloneWorldDatabaseContractsCRUDProbe(loaded.World)
	removed := false
	for databaseIndex := range changed.Databases {
		if string(changed.Databases[databaseIndex].ID) != "master_data" {
			continue
		}
		for contractIndex := range changed.Databases[databaseIndex].Contracts {
			contract := &changed.Databases[databaseIndex].Contracts[contractIndex]
			if string(contract.ID) == "master_data.customer" {
				contract.Reads = nil
				removed = true
			}
		}
	}
	if !removed {
		t.Fatal("test setup: master_data.customer contractがありません")
	}
	if _, err := deriveBusinessActionOwnershipBoundaryProjectionProbe(changed, f); err == nil {
		t.Fatal("CRUD read authorityが失われたのにownership boundary rowを保持しました")
	}
}

func TestBusinessActionOwnershipBoundaryProjectionDerivesOwnershipInsteadOfCopyingItIntoBusinessSource(t *testing.T) {
	root := repoPath()
	loaded, err := workspace.Load(filepath.Join(root, "reference", "workspace"))
	if err != nil {
		t.Fatal(err)
	}
	f, err := readFixture(filepath.Join(root, "reference", "business-system-design", "order-entry-business-procedure.json"))
	if err != nil {
		t.Fatal(err)
	}

	changed := loaded.World
	changed.Databases = append([]database.Database(nil), loaded.World.Databases...)
	changedOwnership := false
	for i := range changed.Databases {
		if string(changed.Databases[i].ID) == "master_data" {
			changed.Databases[i].Ownership = database.OwnershipManagedHere
			changedOwnership = true
		}
	}
	if !changedOwnership {
		t.Fatal("test setup: master_data databaseがありません")
	}

	rows, err := deriveBusinessActionOwnershipBoundaryProjectionProbe(changed, f)
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range rows {
		if row.StepID == "BP-02" {
			if row.Ownership != database.OwnershipManagedHere {
				t.Fatalf("Database authority変更がderived ownership viewへ反映されません: %+v", row)
			}
			return
		}
	}
	t.Fatal("BP-02 ownership boundary rowがありません")
}

func deriveBusinessActionOwnershipBoundaryProjectionProbe(world ir.World, f fixture) ([]businessActionOwnershipBoundaryRowProbe, error) {
	crudRows, err := deriveBusinessActionCRUDProjectionProbe(world, f)
	if err != nil {
		return nil, err
	}
	crudByStep := businessActionCRUDRowsByStep(crudRows)

	rowByKey := map[string]businessActionOwnershipBoundaryRowProbe{}
	for _, step := range f.Steps {
		accepted := crudByStep[step.ID]
		for _, patternID := range step.ApplicationPatterns {
			owner, pattern, ok := findPatternOwner(world, patternID)
			if !ok {
				return nil, fmt.Errorf("step %s Application Pattern %sをexactに解決できません", step.ID, patternID)
			}
			candidate, ok, err := ownershipBoundaryCandidateForPatternProbe(world, step.ID, owner, pattern)
			if err != nil {
				return nil, err
			}
			if !ok {
				continue
			}
			if !crudProjectionContainsCellProbe(accepted, candidate.Data, candidate.Operation) {
				return nil, fmt.Errorf("step %s ownership candidate %s/%s lacks accepted CRUD authority", step.ID, candidate.Data, candidate.Operation)
			}
			key := candidate.StepID + "\x00" + candidate.Data + "\x00" + string(candidate.Operation) + "\x00" + candidate.Database
			rowByKey[key] = candidate
		}
	}

	rows := make([]businessActionOwnershipBoundaryRowProbe, 0, len(rowByKey))
	for _, row := range rowByKey {
		rows = append(rows, row)
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].StepID != rows[j].StepID {
			return rows[i].StepID < rows[j].StepID
		}
		if rows[i].Data != rows[j].Data {
			return rows[i].Data < rows[j].Data
		}
		return rows[i].Database < rows[j].Database
	})
	return rows, nil
}

func ownershipBoundaryCandidateForPatternProbe(world ir.World, stepID string, owner application.Application, pattern application.Pattern) (businessActionOwnershipBoundaryRowProbe, bool, error) {
	resolve := func(requirement application.DatabaseRequirement, dataID, evidence string) (businessActionOwnershipBoundaryRowProbe, bool, error) {
		db, ok := findDatabaseExactlyOnceCRUDProbe(world, string(requirement.TargetDatabase))
		if !ok {
			return businessActionOwnershipBoundaryRowProbe{}, false, fmt.Errorf("Database %sをexactに解決できません", requirement.TargetDatabase)
		}
		return businessActionOwnershipBoundaryRowProbe{
			StepID:       stepID,
			Data:         dataID,
			Operation:    crudReadProbe,
			Database:     string(db.ID),
			Ownership:    db.Ownership,
			Access:       requirement.Access,
			EvidencePath: evidence + "/database:" + string(db.ID),
		}, true, nil
	}

	switch pattern.Kind {
	case application.PatternBrowse:
		q, ok := findEmbeddedQueryExactlyOnceCRUDProbe(world, string(pattern.Query))
		if !ok {
			return businessActionOwnershipBoundaryRowProbe{}, false, fmt.Errorf("browse Pattern %s Query %sをexactに解決できません", pattern.ID, pattern.Query)
		}
		requirement, ok := owner.DatabaseByAlias(q.Datasource)
		if !ok {
			return businessActionOwnershipBoundaryRowProbe{}, false, fmt.Errorf("browse Pattern %s datasource %sをApplication %sで解決できません", pattern.ID, q.Datasource, owner.ID)
		}
		return resolve(requirement, string(pattern.Data), "application-pattern:"+string(pattern.ID)+"/query:"+string(q.ID))

	case application.PatternReference:
		_, relation, ok := findSemanticRelationExactlyOnce(world, string(pattern.Relation))
		if !ok {
			return businessActionOwnershipBoundaryRowProbe{}, false, fmt.Errorf("reference Pattern %s Relation %sをexactに解決できません", pattern.ID, pattern.Relation)
		}
		requirement, ok := owner.DatabaseByAlias(pattern.Datasource)
		if !ok {
			return businessActionOwnershipBoundaryRowProbe{}, false, fmt.Errorf("reference Pattern %s datasource %sをApplication %sで解決できません", pattern.ID, pattern.Datasource, owner.ID)
		}
		return resolve(requirement, string(relation.Target), "application-pattern:"+string(pattern.ID)+"/relation:"+string(pattern.Relation))

	case application.PatternHeaderDetail:
		binding, ok := findHeaderDetailBindingExactlyOnceCRUDProbe(world, string(owner.ID), string(pattern.ID))
		if !ok {
			return businessActionOwnershipBoundaryRowProbe{}, false, fmt.Errorf("header-detail Pattern %s realization bindingをexactに解決できません", pattern.ID)
		}
		requirement, ok := owner.DatabaseByAlias(binding.Datasource)
		if !ok {
			return businessActionOwnershipBoundaryRowProbe{}, false, fmt.Errorf("header-detail Pattern %s datasource %sをApplication %sで解決できません", pattern.ID, binding.Datasource, owner.ID)
		}
		if requirement.TargetDatabase != binding.Target.Database {
			return businessActionOwnershipBoundaryRowProbe{}, false, fmt.Errorf("header-detail Pattern %s datasource database %s != binding database %s", pattern.ID, requirement.TargetDatabase, binding.Target.Database)
		}
		_, queryContract, err := resolveContractQueryReadCRUDProbe(world, string(binding.Target.Database), string(binding.Target.Contract), string(binding.Target.Query))
		if err != nil {
			return businessActionOwnershipBoundaryRowProbe{}, false, err
		}
		return resolve(requirement, string(queryContract.Result), "application-pattern:"+string(pattern.ID)+"/header-detail-binding:"+binding.Target.Identity())

	case application.PatternRecord, application.PatternCommand:
		return businessActionOwnershipBoundaryRowProbe{}, false, nil
	default:
		return businessActionOwnershipBoundaryRowProbe{}, false, fmt.Errorf("Pattern %s unsupported kind %q", pattern.ID, pattern.Kind)
	}
}

func crudProjectionContainsCellProbe(row businessActionCRUDProjectionRowProbe, dataID string, operation crudOperationProbe) bool {
	for _, cell := range row.Cells {
		if cell.Data == dataID && cell.Operation == operation {
			return true
		}
	}
	return false
}
