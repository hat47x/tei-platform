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
	"tei/internal/query"
	"tei/internal/realization"
	"tei/internal/workspace"
)

type crudOperationProbe string

const (
	crudCreateProbe crudOperationProbe = "C"
	crudReadProbe   crudOperationProbe = "R"
	crudUpdateProbe crudOperationProbe = "U"
	crudDeleteProbe crudOperationProbe = "D"
)

type businessActionCRUDCellProbe struct {
	Data          string             `json:"data"`
	Operation     crudOperationProbe `json:"operation"`
	EvidencePaths []string           `json:"evidencePaths"`
}

type businessActionCRUDProjectionRowProbe struct {
	StepID                       string                        `json:"step"`
	Cells                        []businessActionCRUDCellProbe `json:"cells,omitempty"`
	NoCRUDTraceFromCurrentRefs   bool                          `json:"noCRUDTraceFromCurrentRefs"`
}

func TestBusinessActionCRUDProjectionDerivesOnlyEvidenceBackedReads(t *testing.T) {
	root := repoPath()
	loaded, err := workspace.Load(filepath.Join(root, "reference", "workspace"))
	if err != nil {
		t.Fatal(err)
	}
	f, err := readFixture(filepath.Join(root, "reference", "business-system-design", "order-entry-business-procedure.json"))
	if err != nil {
		t.Fatal(err)
	}

	rows, err := deriveBusinessActionCRUDProjectionProbe(loaded.World, f)
	if err != nil {
		t.Fatal(err)
	}
	byStep := businessActionCRUDRowsByStep(rows)

	assertCRUDCells(t, byStep["BP-00"], []businessActionCRUDCellProbe{{Data: "sales.order", Operation: crudReadProbe}})
	assertCRUDCells(t, byStep["BP-02"], []businessActionCRUDCellProbe{{Data: "customer.customer", Operation: crudReadProbe}})
	assertCRUDCells(t, byStep["BP-03"], []businessActionCRUDCellProbe{{Data: "sales.order_line", Operation: crudReadProbe}})

	for _, stepID := range []string{"BP-01", "BP-04", "BP-05", "BP-06", "BP-07", "BP-08", "BP-09"} {
		row := byStep[stepID]
		if !row.NoCRUDTraceFromCurrentRefs || len(row.Cells) != 0 {
			t.Fatalf("%sへcurrent authorityのないCRUD operationを推測しました: %+v", stepID, row)
		}
	}
}

func TestBusinessActionCRUDProjectionDoesNotTreatRecordHeaderOrCommandKindAsGenericCRUD(t *testing.T) {
	root := repoPath()
	loaded, err := workspace.Load(filepath.Join(root, "reference", "workspace"))
	if err != nil {
		t.Fatal(err)
	}
	f, err := readFixture(filepath.Join(root, "reference", "business-system-design", "order-entry-business-procedure.json"))
	if err != nil {
		t.Fatal(err)
	}

	rows, err := deriveBusinessActionCRUDProjectionProbe(loaded.World, f)
	if err != nil {
		t.Fatal(err)
	}
	byStep := businessActionCRUDRowsByStep(rows)

	// record is only a one-record Application intent; it does not say read/update.
	if len(byStep["BP-01"].Cells) != 0 {
		t.Fatalf("record PatternだけからCRUDを推測しました: %+v", byStep["BP-01"])
	}
	// header-detail becomes R only because an exact Contract Query read realization exists.
	assertCRUDCells(t, byStep["BP-03"], []businessActionCRUDCellProbe{{Data: "sales.order_line", Operation: crudReadProbe}})
	// command invokes a Capability; Capability identity alone does not say C/U/D.
	if len(byStep["BP-07"].Cells) != 0 {
		t.Fatalf("command PatternだけからC/U/Dを推測しました: %+v", byStep["BP-07"])
	}
}

func TestBusinessActionCRUDProjectionDoesNotAttributeDatabaseWriteContractToCommandWithoutActionLink(t *testing.T) {
	root := repoPath()
	loaded, err := workspace.Load(filepath.Join(root, "reference", "workspace"))
	if err != nil {
		t.Fatal(err)
	}
	f, err := readFixture(filepath.Join(root, "reference", "business-system-design", "order-entry-business-procedure.json"))
	if err != nil {
		t.Fatal(err)
	}

	if !databaseContractWriteExistsCRUDProbe(loaded.World, "sales_core", "sales_core.order", "sales.order.status") {
		t.Fatal("test setup: sales_core.order write sales.order.status がありません")
	}
	rows, err := deriveBusinessActionCRUDProjectionProbe(loaded.World, f)
	if err != nil {
		t.Fatal(err)
	}
	row := businessActionCRUDRowsByStep(rows)["BP-07"]
	for _, cell := range row.Cells {
		if cell.Operation == crudCreateProbe || cell.Operation == crudUpdateProbe || cell.Operation == crudDeleteProbe {
			t.Fatalf("Database Contract writeをBP-07 commandへ暗黙帰属しました: %+v", row)
		}
	}
	if !row.NoCRUDTraceFromCurrentRefs {
		t.Fatalf("BP-07をoperation解決済みと誤表示しました: %+v", row)
	}
}

func TestBusinessActionCRUDProjectionKeepsSemanticTraceAndOperationTraceSeparate(t *testing.T) {
	root := repoPath()
	loaded, err := workspace.Load(filepath.Join(root, "reference", "workspace"))
	if err != nil {
		t.Fatal(err)
	}
	f, err := readFixture(filepath.Join(root, "reference", "business-system-design", "order-entry-business-procedure.json"))
	if err != nil {
		t.Fatal(err)
	}

	dataRows, err := deriveBusinessActionDataProjectionProbe(loaded.World, f)
	if err != nil {
		t.Fatal(err)
	}
	crudRows, err := deriveBusinessActionCRUDProjectionProbe(loaded.World, f)
	if err != nil {
		t.Fatal(err)
	}
	dataByStep := businessActionDataRowsByStep(dataRows)
	crudByStep := businessActionCRUDRowsByStep(crudRows)

	if !reflect.DeepEqual(dataByStep["BP-02"].Data, []string{"customer.customer", "sales.order"}) {
		t.Fatalf("BP-02 Data trace setupが変化しました: %+v", dataByStep["BP-02"])
	}
	assertCRUDCells(t, crudByStep["BP-02"], []businessActionCRUDCellProbe{{Data: "customer.customer", Operation: crudReadProbe}})

	if !reflect.DeepEqual(dataByStep["BP-03"].Data, []string{"sales.order", "sales.order_line"}) {
		t.Fatalf("BP-03 Data trace setupが変化しました: %+v", dataByStep["BP-03"])
	}
	assertCRUDCells(t, crudByStep["BP-03"], []businessActionCRUDCellProbe{{Data: "sales.order_line", Operation: crudReadProbe}})
}

func TestBusinessActionCRUDProjectionFailsClosedWhenReadContractNoLongerSupportsReference(t *testing.T) {
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
			if string(contract.ID) != "master_data.customer" {
				continue
			}
			contract.Reads = nil
			removed = true
		}
	}
	if !removed {
		t.Fatal("test setup: master_data.customer contractがありません")
	}

	if _, err := deriveBusinessActionCRUDProjectionProbe(changed, f); err == nil {
		t.Fatal("reference Contract Queryのread authority喪失をRとして黙って保持しました")
	}
}

func TestBusinessActionCRUDProjectionIsDerivedAndDoesNotClaimCompleteness(t *testing.T) {
	projectionType := reflect.TypeOf(businessActionCRUDProjectionRowProbe{})
	want := []string{"StepID", "Cells", "NoCRUDTraceFromCurrentRefs"}
	if got := fieldNames(projectionType); !reflect.DeepEqual(got, want) {
		t.Fatalf("CRUD projection boundaryが変化しました: got=%v want=%v", got, want)
	}
	for _, forbidden := range []string{"Complete", "Authoritative", "Ownership", "WriteAuthority", "BusinessFlow", "Workflow", "SourceOfTruth"} {
		if _, found := projectionType.FieldByName(forbidden); found {
			t.Fatalf("derived CRUD projectionへauthority/completeness field %qを追加しました", forbidden)
		}
	}
}

func deriveBusinessActionCRUDProjectionProbe(world ir.World, f fixture) ([]businessActionCRUDProjectionRowProbe, error) {
	rows := make([]businessActionCRUDProjectionRowProbe, 0, len(f.Steps))
	for _, step := range f.Steps {
		cellEvidence := map[string]map[string]struct{}{}

		for _, patternID := range step.ApplicationPatterns {
			owner, pattern, ok := findPatternOwner(world, patternID)
			if !ok {
				return nil, fmt.Errorf("step %s Application Pattern %sをexactに解決できません", step.ID, patternID)
			}
			switch pattern.Kind {
			case application.PatternBrowse:
				if err := addBrowseReadCRUDProbe(world, owner, pattern, cellEvidence); err != nil {
					return nil, err
				}
			case application.PatternReference:
				if err := addReferenceReadCRUDProbe(world, owner, pattern, cellEvidence); err != nil {
					return nil, err
				}
			case application.PatternHeaderDetail:
				if err := addHeaderDetailReadCRUDProbe(world, owner, pattern, cellEvidence); err != nil {
					return nil, err
				}
			case application.PatternRecord, application.PatternCommand:
				// Current Pattern contract does not provide a persistence CRUD operation.
			default:
				return nil, fmt.Errorf("step %s Pattern %s has unsupported kind %q", step.ID, pattern.ID, pattern.Kind)
			}
		}

		cells := make([]businessActionCRUDCellProbe, 0, len(cellEvidence))
		for key, evidenceSet := range cellEvidence {
			dataID, operation, err := splitCRUDCellKeyProbe(key)
			if err != nil {
				return nil, err
			}
			cells = append(cells, businessActionCRUDCellProbe{
				Data:          dataID,
				Operation:     operation,
				EvidencePaths: sortedProjectionKeys(evidenceSet),
			})
		}
		sort.Slice(cells, func(i, j int) bool {
			if cells[i].Data != cells[j].Data {
				return cells[i].Data < cells[j].Data
			}
			return cells[i].Operation < cells[j].Operation
		})
		rows = append(rows, businessActionCRUDProjectionRowProbe{
			StepID:                     step.ID,
			Cells:                      cells,
			NoCRUDTraceFromCurrentRefs: len(cells) == 0,
		})
	}
	return rows, nil
}

func addBrowseReadCRUDProbe(world ir.World, owner application.Application, pattern application.Pattern, cellEvidence map[string]map[string]struct{}) error {
	if pattern.Data == "" || pattern.Query == "" {
		return fmt.Errorf("browse Pattern %s lacks Data/Query authority", pattern.ID)
	}
	if !semanticDataExistsExactlyOnce(world, string(pattern.Data)) {
		return fmt.Errorf("browse Pattern %s Data %sをexactに解決できません", pattern.ID, pattern.Data)
	}
	q, ok := findEmbeddedQueryExactlyOnceCRUDProbe(world, string(pattern.Query))
	if !ok {
		return fmt.Errorf("browse Pattern %s Query %sをexactに解決できません", pattern.ID, pattern.Query)
	}
	if q.Application != owner.ID {
		return fmt.Errorf("browse Pattern %s Query owner mismatch: pattern app=%s query app=%s", pattern.ID, owner.ID, q.Application)
	}
	if q.Datasource == "" {
		return fmt.Errorf("browse Pattern %s Query %s lacks datasource", pattern.ID, pattern.Query)
	}
	datasource, ok := owner.DatabaseByAliasExact(q.Datasource)
	if !ok || datasource.Access == database.AccessWrite {
		return fmt.Errorf("browse Pattern %s Query datasource %s is not readable", pattern.ID, q.Datasource)
	}
	addCRUDCellEvidenceProbe(cellEvidence, string(pattern.Data), crudReadProbe,
		"application-pattern:"+string(pattern.ID)+"/browse-query:"+string(pattern.Query)+"/data:"+string(pattern.Data))
	return nil
}

func addReferenceReadCRUDProbe(world ir.World, owner application.Application, pattern application.Pattern, cellEvidence map[string]map[string]struct{}) error {
	if pattern.Relation == "" || pattern.Datasource == "" || pattern.ContractQuery == nil {
		return fmt.Errorf("reference Pattern %s lacks Relation/Datasource/ContractQuery authority", pattern.ID)
	}
	_, relation, ok := findSemanticRelationExactlyOnce(world, string(pattern.Relation))
	if !ok {
		return fmt.Errorf("reference Pattern %s Relation %sをexactに解決できません", pattern.ID, pattern.Relation)
	}
	datasource, ok := owner.DatabaseByAliasExact(pattern.Datasource)
	if !ok || datasource.Access == database.AccessWrite {
		return fmt.Errorf("reference Pattern %s datasource %s is not readable", pattern.ID, pattern.Datasource)
	}
	contract, queryContract, err := resolveContractQueryReadCRUDProbe(world, string(datasource.TargetDatabase), string(pattern.ContractQuery.Contract), string(pattern.ContractQuery.Query))
	if err != nil {
		return fmt.Errorf("reference Pattern %s: %w", pattern.ID, err)
	}
	targetData := string(relation.Target)
	if string(queryContract.Result) != targetData {
		return fmt.Errorf("reference Pattern %s Contract Query result %s != Relation target %s", pattern.ID, queryContract.Result, targetData)
	}
	if !contractReadsCRUDProbe(contract, targetData) {
		return fmt.Errorf("reference Pattern %s Contract %s does not declare read %s", pattern.ID, contract.ID, targetData)
	}
	addCRUDCellEvidenceProbe(cellEvidence, targetData, crudReadProbe,
		"application-pattern:"+string(pattern.ID)+"/contract-query:"+string(datasource.TargetDatabase)+"/"+string(contract.ID)+"/"+queryContract.Name+"/read:"+targetData)
	return nil
}

func addHeaderDetailReadCRUDProbe(world ir.World, owner application.Application, pattern application.Pattern, cellEvidence map[string]map[string]struct{}) error {
	binding, ok := findHeaderDetailBindingExactlyOnceCRUDProbe(world, string(owner.ID), string(pattern.ID))
	if !ok {
		return fmt.Errorf("header-detail Pattern %s realization bindingをexactに解決できません", pattern.ID)
	}
	datasource, ok := owner.DatabaseByAliasExact(binding.Datasource)
	if !ok || datasource.Access == database.AccessWrite {
		return fmt.Errorf("header-detail Pattern %s datasource %s is not readable", pattern.ID, binding.Datasource)
	}
	if datasource.TargetDatabase != binding.Target.Database {
		return fmt.Errorf("header-detail Pattern %s datasource database %s != binding database %s", pattern.ID, datasource.TargetDatabase, binding.Target.Database)
	}
	contract, queryContract, err := resolveContractQueryReadCRUDProbe(world, string(binding.Target.Database), string(binding.Target.Contract), string(binding.Target.Query))
	if err != nil {
		return fmt.Errorf("header-detail Pattern %s: %w", pattern.ID, err)
	}
	resultData := string(queryContract.Result)
	if resultData == "" || !semanticDataExistsExactlyOnce(world, resultData) {
		return fmt.Errorf("header-detail Pattern %s Contract Query result %sをexact Dataとして解決できません", pattern.ID, queryContract.Result)
	}
	if !contractReadsCRUDProbe(contract, resultData) {
		return fmt.Errorf("header-detail Pattern %s Contract %s does not declare read %s", pattern.ID, contract.ID, resultData)
	}
	addCRUDCellEvidenceProbe(cellEvidence, resultData, crudReadProbe,
		"header-detail-binding:"+binding.Target.Identity()+"/read:"+resultData)
	return nil
}

func resolveContractQueryReadCRUDProbe(world ir.World, databaseID, contractID, queryName string) (database.Contract, database.QueryContract, error) {
	db, ok := findDatabaseExactlyOnceCRUDProbe(world, databaseID)
	if !ok {
		return database.Contract{}, database.QueryContract{}, fmt.Errorf("Database %sをexactに解決できません", databaseID)
	}
	contract, ok := findDatabaseContractExactlyOnceCRUDProbe(db, contractID)
	if !ok {
		return database.Contract{}, database.QueryContract{}, fmt.Errorf("Contract %sをexactに解決できません", contractID)
	}
	queryContract, ok := findQueryContractExactlyOnceCRUDProbe(contract, queryName)
	if !ok {
		return database.Contract{}, database.QueryContract{}, fmt.Errorf("Contract Query %s/%sをexactに解決できません", contractID, queryName)
	}
	return contract, queryContract, nil
}

func findEmbeddedQueryExactlyOnceCRUDProbe(world ir.World, id string) (query.EmbeddedSQL, bool) {
	var found query.EmbeddedSQL
	count := 0
	for _, item := range world.Queries {
		if string(item.ID) == id {
			found = item
			count++
		}
	}
	return found, count == 1
}

func findHeaderDetailBindingExactlyOnceCRUDProbe(world ir.World, applicationID, patternID string) (realization.HeaderDetailContractQueryBinding, bool) {
	var found realization.HeaderDetailContractQueryBinding
	count := 0
	for _, item := range world.HeaderDetailContractQueryBindings {
		if string(item.Application) == applicationID && string(item.Pattern) == patternID {
			found = item
			count++
		}
	}
	return found, count == 1
}

func findDatabaseExactlyOnceCRUDProbe(world ir.World, id string) (database.Database, bool) {
	var found database.Database
	count := 0
	for _, item := range world.Databases {
		if string(item.ID) == id {
			found = item
			count++
		}
	}
	return found, count == 1
}

func findDatabaseContractExactlyOnceCRUDProbe(db database.Database, id string) (database.Contract, bool) {
	var found database.Contract
	count := 0
	for _, item := range db.Contracts {
		if string(item.ID) == id {
			found = item
			count++
		}
	}
	return found, count == 1
}

func findQueryContractExactlyOnceCRUDProbe(contract database.Contract, name string) (database.QueryContract, bool) {
	var found database.QueryContract
	count := 0
	for _, item := range contract.Queries {
		if item.Name == name {
			found = item
			count++
		}
	}
	return found, count == 1
}

func contractReadsCRUDProbe(contract database.Contract, target string) bool {
	for _, item := range contract.Reads {
		if string(item) == target {
			return true
		}
	}
	return false
}

func databaseContractWriteExistsCRUDProbe(world ir.World, databaseID, contractID, target string) bool {
	db, ok := findDatabaseExactlyOnceCRUDProbe(world, databaseID)
	if !ok {
		return false
	}
	contract, ok := findDatabaseContractExactlyOnceCRUDProbe(db, contractID)
	if !ok {
		return false
	}
	for _, item := range contract.Writes {
		if string(item) == target {
			return true
		}
	}
	return false
}

func addCRUDCellEvidenceProbe(cellEvidence map[string]map[string]struct{}, dataID string, operation crudOperationProbe, evidence string) {
	key := dataID + "\x00" + string(operation)
	if cellEvidence[key] == nil {
		cellEvidence[key] = map[string]struct{}{}
	}
	cellEvidence[key][evidence] = struct{}{}
}

func splitCRUDCellKeyProbe(key string) (string, crudOperationProbe, error) {
	for _, operation := range []crudOperationProbe{crudCreateProbe, crudReadProbe, crudUpdateProbe, crudDeleteProbe} {
		suffix := "\x00" + string(operation)
		if len(key) > len(suffix) && key[len(key)-len(suffix):] == suffix {
			return key[:len(key)-len(suffix)], operation, nil
		}
	}
	return "", "", fmt.Errorf("invalid CRUD cell key %q", key)
}

func businessActionCRUDRowsByStep(rows []businessActionCRUDProjectionRowProbe) map[string]businessActionCRUDProjectionRowProbe {
	result := make(map[string]businessActionCRUDProjectionRowProbe, len(rows))
	for _, row := range rows {
		result[row.StepID] = row
	}
	return result
}

func assertCRUDCells(t *testing.T, row businessActionCRUDProjectionRowProbe, want []businessActionCRUDCellProbe) {
	t.Helper()
	got := make([]businessActionCRUDCellProbe, len(row.Cells))
	copy(got, row.Cells)
	for i := range got {
		got[i].EvidencePaths = nil
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("step %s CRUD cells=%+v want=%+v", row.StepID, got, want)
	}
	if len(want) == 0 && !row.NoCRUDTraceFromCurrentRefs {
		t.Fatalf("step %s no-trace flag=false", row.StepID)
	}
	if len(want) > 0 && row.NoCRUDTraceFromCurrentRefs {
		t.Fatalf("step %s has cells but no-trace flag=true", row.StepID)
	}
	for _, cell := range row.Cells {
		if len(cell.EvidencePaths) == 0 {
			t.Fatalf("step %s CRUD cell lacks evidence path: %+v", row.StepID, cell)
		}
	}
}

func cloneWorldDatabaseContractsCRUDProbe(world ir.World) ir.World {
	result := world
	result.Databases = append([]database.Database(nil), world.Databases...)
	for databaseIndex := range result.Databases {
		result.Databases[databaseIndex].Contracts = append([]database.Contract(nil), world.Databases[databaseIndex].Contracts...)
		for contractIndex := range result.Databases[databaseIndex].Contracts {
			original := world.Databases[databaseIndex].Contracts[contractIndex]
			contract := &result.Databases[databaseIndex].Contracts[contractIndex]
			contract.Reads = append([]core.CanonicalID(nil), original.Reads...)
			contract.Writes = append([]core.CanonicalID(nil), original.Writes...)
			contract.Queries = append([]database.QueryContract(nil), original.Queries...)
		}
	}
	return result
}
