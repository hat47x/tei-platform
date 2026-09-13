package main

import (
	"fmt"
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	"tei/internal/ir"
	"tei/internal/workspace"
)

type crossViewProofOutcomeProbe string

const (
	proofSupportedProbe     crossViewProofOutcomeProbe = "supported"
	proofContradictionProbe crossViewProofOutcomeProbe = "contradiction"
	proofUnresolvedProbe    crossViewProofOutcomeProbe = "unresolved"
)

type crossViewProofResultProbe struct {
	ID                 string                     `json:"id"`
	Family             string                     `json:"family"`
	Question           string                     `json:"question"`
	Result             crossViewProofOutcomeProbe `json:"result"`
	EvidencePaths      []string                   `json:"evidencePaths,omitempty"`
	MissingAuthorities []string                   `json:"missingAuthorities,omitempty"`
	Reason             string                     `json:"reason"`
}

func TestDesignSynthesisCrossViewProofReportR1ReproducesSelectedOutcomes(t *testing.T) {
	root := repoPath()
	loaded, err := workspace.Load(filepath.Join(root, "reference", "workspace"))
	if err != nil {
		t.Fatal(err)
	}
	f, err := readFixture(filepath.Join(root, "reference", "business-system-design", "order-entry-business-procedure.json"))
	if err != nil {
		t.Fatal(err)
	}
	policy, err := readLegacyAuthorizationFixtureProbe(filepath.Join(root, "reference", "legacy-auth", "authorization.json"))
	if err != nil {
		t.Fatal(err)
	}

	results, err := deriveOrderEntryCrossViewProofReportProbe(loaded.World, f, policy)
	if err != nil {
		t.Fatal(err)
	}
	byID := crossViewProofResultsByIDProbe(results)

	want := map[string]crossViewProofOutcomeProbe{
		"P1-00":  proofSupportedProbe,
		"P2-04":  proofUnresolvedProbe,
		"P3-02":  proofContradictionProbe,
		"P3-07":  proofUnresolvedProbe,
		"P4-06a": proofSupportedProbe,
		"P4-06b": proofUnresolvedProbe,
	}
	if len(byID) != len(want) {
		t.Fatalf("proof result count=%d want=%d: %+v", len(byID), len(want), results)
	}
	for id, outcome := range want {
		result, ok := byID[id]
		if !ok {
			t.Fatalf("proof result %sがありません: %+v", id, results)
		}
		if result.Result != outcome {
			t.Fatalf("proof result %s=%q want=%q: %+v", id, result.Result, outcome, result)
		}
		if result.Reason == "" {
			t.Fatalf("proof result %sに理由がありません", id)
		}
		if outcome == proofUnresolvedProbe && len(result.MissingAuthorities) == 0 {
			t.Fatalf("unresolved proof %sにmissing authorityがありません: %+v", id, result)
		}
		if outcome != proofUnresolvedProbe && len(result.EvidencePaths) == 0 {
			t.Fatalf("resolved proof %sにEvidence pathがありません: %+v", id, result)
		}
	}
}

func TestDesignSynthesisCrossViewProofReportKeepsDatabaseWriteAndCommandAttributionSeparate(t *testing.T) {
	root := repoPath()
	loaded, err := workspace.Load(filepath.Join(root, "reference", "workspace"))
	if err != nil {
		t.Fatal(err)
	}
	f, err := readFixture(filepath.Join(root, "reference", "business-system-design", "order-entry-business-procedure.json"))
	if err != nil {
		t.Fatal(err)
	}
	policy, err := readLegacyAuthorizationFixtureProbe(filepath.Join(root, "reference", "legacy-auth", "authorization.json"))
	if err != nil {
		t.Fatal(err)
	}

	results, err := deriveOrderEntryCrossViewProofReportProbe(loaded.World, f, policy)
	if err != nil {
		t.Fatal(err)
	}
	result := crossViewProofResultsByIDProbe(results)["P3-07"]
	if result.Result != proofUnresolvedProbe {
		t.Fatalf("DB writeの存在だけでsubmit mutationを解決しました: %+v", result)
	}
	if !containsStringProofProbe(result.EvidencePaths, "database-contract:sales_core/sales_core.order/write:sales.order.status") {
		t.Fatalf("既知のDatabase write Evidenceを失いました: %+v", result)
	}
	if !containsStringProofProbe(result.MissingAuthorities, "capability-to-database-write-attribution") {
		t.Fatalf("不足authorityを明示できません: %+v", result)
	}
}

func TestDesignSynthesisCrossViewProofReportDoesNotBackfillBusinessTargetFromSecurityPolicy(t *testing.T) {
	root := repoPath()
	loaded, err := workspace.Load(filepath.Join(root, "reference", "workspace"))
	if err != nil {
		t.Fatal(err)
	}
	f, err := readFixture(filepath.Join(root, "reference", "business-system-design", "order-entry-business-procedure.json"))
	if err != nil {
		t.Fatal(err)
	}
	policy, err := readLegacyAuthorizationFixtureProbe(filepath.Join(root, "reference", "legacy-auth", "authorization.json"))
	if err != nil {
		t.Fatal(err)
	}

	results, err := deriveOrderEntryCrossViewProofReportProbe(loaded.World, f, policy)
	if err != nil {
		t.Fatal(err)
	}
	result := crossViewProofResultsByIDProbe(results)["P4-06b"]
	if result.Result != proofUnresolvedProbe {
		t.Fatalf("Security policyだけでBusiness protected targetを解決しました: %+v", result)
	}
	if !containsStringProofProbe(result.EvidencePaths, "legacy-authorization:execute/allow:order_entry.submit_order") {
		t.Fatalf("corroborating Security Evidenceを失いました: %+v", result)
	}
	if !containsStringProofProbe(result.MissingAuthorities, "business-protected-target-authority") {
		t.Fatalf("Business-side target authority不足を明示できません: %+v", result)
	}
}

func TestDesignSynthesisCrossViewProofReportIsDerivedAndNotASecondAuthority(t *testing.T) {
	resultType := reflect.TypeOf(crossViewProofResultProbe{})
	want := []string{"ID", "Family", "Question", "Result", "EvidencePaths", "MissingAuthorities", "Reason"}
	if got := fieldNames(resultType); !reflect.DeepEqual(got, want) {
		t.Fatalf("proof report shapeが変化しました: got=%v want=%v", got, want)
	}
	for _, forbidden := range []string{"Canonical", "Authoritative", "Accepted", "Complete", "Confidence", "Score", "WriteBack", "SourceOfTruth"} {
		if _, found := resultType.FieldByName(forbidden); found {
			t.Fatalf("derived proof reportへauthority/completeness field %qを追加しました", forbidden)
		}
	}
}

func deriveOrderEntryCrossViewProofReportProbe(world ir.World, f fixture, policy legacyAuthorizationFixtureProbe) ([]crossViewProofResultProbe, error) {
	dataRows, err := deriveBusinessActionDataProjectionProbe(world, f)
	if err != nil {
		return nil, err
	}
	dataByStep := businessActionDataRowsByStep(dataRows)
	crudRows, err := deriveBusinessActionCRUDProjectionProbe(world, f)
	if err != nil {
		return nil, err
	}
	crudByStep := businessActionCRUDRowsByStep(crudRows)

	bp00, ok := findStepProofProbe(f, "BP-00")
	if !ok || !containsStringProofProbe(bp00.ApplicationPatterns, "order_entry.recent_orders") {
		return nil, fmt.Errorf("P1-00 setup: BP-00 -> order_entry.recent_orders exact refがありません")
	}
	if _, _, ok := findPatternOwner(world, "order_entry.recent_orders"); !ok {
		return nil, fmt.Errorf("P1-00 setup: order_entry.recent_ordersをexactに解決できません")
	}

	bp04Data, ok := dataByStep["BP-04"]
	if !ok || !bp04Data.NoDataTraceFromCurrentRefs {
		return nil, fmt.Errorf("P2-04 setup: BP-04 no-current-data-traceが成立しません: %+v", bp04Data)
	}

	owner, customerPattern, ok := findPatternOwner(world, "order_entry.customer_reference")
	if !ok {
		return nil, fmt.Errorf("P3-02 setup: order_entry.customer_referenceをexactに解決できません")
	}
	masterDatasource, ok := owner.DatabaseByAliasExact(customerPattern.Datasource)
	if !ok {
		return nil, fmt.Errorf("P3-02 setup: datasource %sをApplication %sでexactに解決できません", customerPattern.Datasource, owner.ID)
	}
	if string(masterDatasource.Access) != "read" {
		return nil, fmt.Errorf("P3-02 setup: master datasource access=%s want=read", masterDatasource.Access)
	}

	if !databaseContractWriteExistsCRUDProbe(world, "sales_core", "sales_core.order", "sales.order.status") {
		return nil, fmt.Errorf("P3-07 setup: sales_core.order write sales.order.statusがありません")
	}
	bp07CRUD, ok := crudByStep["BP-07"]
	if !ok || !bp07CRUD.NoCRUDTraceFromCurrentRefs || len(bp07CRUD.Cells) != 0 {
		return nil, fmt.Errorf("P3-07 setup: BP-07 operation attributionが既に解決されています: %+v", bp07CRUD)
	}

	bp06, ok := findStepProofProbe(f, "BP-06")
	if !ok || !containsStringProofProbe(bp06.Capabilities, "security.authorize") {
		return nil, fmt.Errorf("P4-06a setup: BP-06 -> security.authorize exact refがありません")
	}
	_, _, _, capabilityCounts := semanticCounts(world)
	if capabilityCounts["security.authorize"] != 1 {
		return nil, fmt.Errorf("P4-06a setup: security.authorize count=%d want=1", capabilityCounts["security.authorize"])
	}

	candidate, err := deriveApplicationPatternTargetAwareStepProbe(f, "BP-06", nil)
	if err != nil {
		return nil, err
	}
	if len(candidate.ApplicationPatternTargets) != 0 {
		return nil, fmt.Errorf("P4-06b setup: Business protected targetが既に存在します: %v", candidate.ApplicationPatternTargets)
	}
	if !legacyAuthorizationAllowsTarget(policy, "execute", "order_entry.submit_order") {
		return nil, fmt.Errorf("P4-06b setup: Legacy Authorization submit target ruleがありません")
	}

	results := []crossViewProofResultProbe{
		{
			ID:            "P1-00",
			Family:        "business-realization",
			Question:      "BP-00 is supported by order_entry.recent_orders",
			Result:        proofSupportedProbe,
			EvidencePaths: []string{"business-step:BP-00/application-pattern:order_entry.recent_orders"},
			Reason:        "Business sourceのexplicit Pattern refがnormalized Worldでexactに1件解決できる",
		},
		{
			ID:                 "P2-04",
			Family:             "business-data-trace",
			Question:           "BP-04 has no Data dependency",
			Result:             proofUnresolvedProbe,
			MissingAuthorities: []string{"accepted-data-or-relation-trace-for:BP-04"},
			Reason:             "current refsからData traceが無いことはData dependency不存在のauthorityにならない",
		},
		{
			ID:            "P3-02",
			Family:        "mutation-authority",
			Question:      "order_entry.customer_reference may mutate the master datasource",
			Result:        proofContradictionProbe,
			EvidencePaths: []string{"application:order_entry/datasource:master/access:read"},
			Reason:        "explicit datasource access=readとmaster mutation候補が両立しない",
		},
		{
			ID:                 "P3-07",
			Family:             "mutation-authority",
			Question:           "sales.order.submit mutates sales.order.status",
			Result:             proofUnresolvedProbe,
			EvidencePaths:      []string{"database-contract:sales_core/sales_core.order/write:sales.order.status"},
			MissingAuthorities: []string{"capability-to-database-write-attribution"},
			Reason:             "DB write authorityは存在するがBP-07 / sales.order.submitへ帰属するaccepted edgeがない",
		},
		{
			ID:            "P4-06a",
			Family:        "business-security-boundary",
			Question:      "BP-06 uses security.authorize",
			Result:        proofSupportedProbe,
			EvidencePaths: []string{"business-step:BP-06/capability:security.authorize"},
			Reason:        "Business sourceのexplicit Capability refがnormalized Worldでexactに1件解決できる",
		},
		{
			ID:                 "P4-06b",
			Family:             "business-security-boundary",
			Question:           "BP-06 protected target is order_entry.submit_order",
			Result:             proofUnresolvedProbe,
			EvidencePaths:      []string{"legacy-authorization:execute/allow:order_entry.submit_order"},
			MissingAuthorities: []string{"business-protected-target-authority"},
			Reason:             "Security policyはtarget候補をcorroborateできるがBusiness targetをauthor/backfillしない",
		},
	}
	for i := range results {
		sort.Strings(results[i].EvidencePaths)
		sort.Strings(results[i].MissingAuthorities)
	}
	return results, nil
}

func findStepProofProbe(f fixture, id string) (step, bool) {
	var found step
	count := 0
	for _, item := range f.Steps {
		if item.ID == id {
			found = item
			count++
		}
	}
	return found, count == 1
}

func crossViewProofResultsByIDProbe(results []crossViewProofResultProbe) map[string]crossViewProofResultProbe {
	byID := make(map[string]crossViewProofResultProbe, len(results))
	for _, result := range results {
		byID[result.ID] = result
	}
	return byID
}

func containsStringProofProbe(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}
