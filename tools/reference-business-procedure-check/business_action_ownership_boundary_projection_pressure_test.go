package main

import (
	"fmt"
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	"tei/internal/application"
	"tei/internal/core"
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

func deriveBusinessActionOwnershipBoundaryProjectionProbe(world ir.World, f fixture) ([]businessActionOwnershipBoundaryRowProbe, error) {
	rows := []businessActionOwnershipBoundaryRowProbe{}
	for _, step := range f.Steps {
		for _, patternID := range step.ApplicationPatterns {
			owner, pattern, ok := findPatternOwner(world, patternID)
			if !ok {
				return nil, fmt.Errorf("step %s Application Pattern %sをexactに解決できません", step.ID, patternID)
			}
			row, ok, err := ownershipBoundaryRowForPatternProbe(world, step.ID, owner, pattern)
			if err != nil {
				return nil, err
			}
			if ok {
				rows = append(rows, row)
			}
		}
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

func ownershipBoundaryRowForPatternProbe(world ir.World, stepID string, owner application.Application, pattern application.Pattern) (businessActionOwnershipBoundaryRowProbe, bool, error) {
	resolveDatabase := func(alias core.LocalName) (application.DatabaseRequirement, database.Database, error) {
		requirement, ok := owner.DatabaseByAlias(alias)
		if !ok {
			return application.DatabaseRequirement{}, database.Database{}, fmt.Errorf("Pattern %s datasource %sをApplication %sで解決できません", pattern.ID, alias, owner.ID)
		}
		db, ok := findDatabaseExactlyOnceOwnershipProbe(world, requirement.TargetDatabase)
		if !ok {
			return application.DatabaseRequirement{}, database.Database{}, fmt.Errorf("Database %sをexactに解決できません", requirement.TargetDatabase)
		}
		return requirement, db, nil
	}

	switch pattern.Kind {
	case application.PatternBrowse:
		q, ok := findEmbeddedQueryExactlyOnceCRUDProbe(world, string(pattern.Query))
		if !ok {
			return businessActionOwnershipBoundaryRowProbe{}, false, fmt.Errorf("browse Pattern %s Query %sをexactに解決できません", pattern.ID, pattern.Query)
		}
		requirement, db, err := resolveDatabase(q.Datasource)
		if err != nil {
			return businessActionOwnershipBoundaryRowProbe{}, false, err
		}
		if requirement.Access == database.AccessWrite {
			return businessActionOwnershipBoundaryRowProbe{}, false, fmt.Errorf("browse Pattern %s datasource is not readable", pattern.ID)
		}
		return businessActionOwnershipBoundaryRowProbe{
			StepID:       stepID,
			Data:         string(pattern.Data),
			Operation:    crudReadProbe,
			Database:     string(db.ID),
			Ownership:    db.Ownership,
			Access:       requirement.Access,
			EvidencePath: "application-pattern:" + string(pattern.ID) + "/query:" + string(q.ID) + "/database:" + string(db.ID),
		}, true, nil

	case application.PatternReference:
		requirement, db, err := resolveDatabase(pattern.Datasource)
		if err != nil {
			return businessActionOwnershipBoundaryRowProbe{}, false, err
		}
		_, relation, ok := findSemanticRelationExactlyOnce(world, string(pattern.Relation))
		if !ok {
			return businessActionOwnershipBoundaryRowProbe{}, false, fmt.Errorf("reference Pattern %s Relation %sをexactに解決できません", pattern.ID, pattern.Relation)
		}
		if pattern.ContractQuery == nil {
			return businessActionOwnershipBoundaryRowProbe{}, false, fmt.Errorf("reference Pattern %s lacks ContractQuery", pattern.ID)
		}
		_, queryContract, err := resolveContractQueryReadCRUDProbe(world, string(db.ID), string(pattern.ContractQuery.Contract), string(pattern.ContractQuery.Query))
		if err != nil {
			return businessActionOwnershipBoundaryRowProbe{}, false, err
		}
		if string(queryContract.Result) != string(relation.Target) {
			return businessActionOwnershipBoundaryRowProbe{}, false, fmt.Errorf("reference Pattern %s result/target mismatch", pattern.ID)
		}
		return businessActionOwnershipBoundaryRowProbe{
			StepID:       stepID,
			Data:         string(queryContract.Result),
			Operation:    crudReadProbe,
			Database:     string(db.ID),
			Ownership:    db.Ownership,
			Access:       requirement.Access,
			EvidencePath: "application-pattern:" + string(pattern.ID) + "/contract-query:" + string(pattern.ContractQuery.Contract) + "/database:" + string(db.ID),
		}, true, nil

	case application.PatternHeaderDetail:
		binding, ok := findHeaderDetailBindingExactlyOnceCRUDProbe(world, string(owner.ID), string(pattern.ID))
		if !ok {
			return businessActionOwnershipBoundaryRowProbe{}, false, fmt.Errorf("header-detail Pattern %s realization bindingをexactに解決できません", pattern.ID)
		}
		requirement, db, err := resolveDatabase(binding.Datasource)
		if err != nil {
			return businessActionOwnershipBoundaryRowProbe{}, false, err
		}
		_, queryContract, err := resolveContractQueryReadCRUDProbe(world, string(db.ID), string(binding.Target.Contract), string(binding.Target.Query))
		if err != nil {
			return businessActionOwnershipBoundaryRowProbe{}, false, err
		}
		return businessActionOwnershipBoundaryRowProbe{
			StepID:       stepID,
			Data:         string(queryContract.Result),
			Operation:    crudReadProbe,
			Database:     string(db.ID),
			Ownership:    db.Ownership,
			Access:       requirement.Access,
			EvidencePath: "application-pattern:" + string(pattern.ID) + "/header-detail-binding/database:" + string(db.ID),
		}, true, nil

	case application.PatternRecord, application.PatternCommand:
		return businessActionOwnershipBoundaryRowProbe{}, false, nil
	default:
		return businessActionOwnershipBoundaryRowProbe{}, false, fmt.Errorf("Pattern %s unsupported kind %q", pattern.ID, pattern.Kind)
	}
}

func findDatabaseExactlyOnceOwnershipProbe(world ir.World, id core.CanonicalID) (database.Database, bool) {
	var found database.Database
	count := 0
	for _, db := range world.Databases {
		if db.ID == id {
			found = db
			count++
		}
	}
	return found, count == 1
}
