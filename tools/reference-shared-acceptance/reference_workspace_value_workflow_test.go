package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"tei/internal/realization"
	"tei/internal/workspace"
	"tei/tools/internal/changeanalysis"
)

// The real reference workspace is copied twice and never edited in place.
// The fixture exercises CLI selection, not execution of external providers.
func TestReferenceWorkspaceDefinitionAndPluginWorkflow(t *testing.T) {
	before, after := scratchWorkspace(t), scratchWorkspace(t)
	binDir := t.TempDir()
	ext := ""
	if runtime.GOOS == "windows" {
		ext = ".exe"
	}
	tei, compare := filepath.Join(binDir, "tei"+ext), filepath.Join(binDir, "compare"+ext)
	for _, build := range []struct{ output, pkg string }{{tei, "../../cmd/tei"}, {compare, "."}} {
		cmd := exec.Command("go", "build", "-o", build.output, build.pkg)
		if output, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("build %s: %v\n%s", build.pkg, err, output)
		}
	}
	run := func(bin string, exit int, args ...string) []byte {
		t.Helper()
		cmd := exec.Command(bin, args...)
		var out, stderr bytes.Buffer
		cmd.Stdout, cmd.Stderr = &out, &stderr
		err := cmd.Run()
		code := 0
		if err != nil {
			if e, ok := err.(*exec.ExitError); ok {
				code = e.ExitCode()
			} else {
				t.Fatal(err)
			}
		}
		if code != exit {
			t.Fatalf("%s %v exit=%d want=%d stdout=%s stderr=%s", bin, args, code, exit, out.String(), stderr.String())
		}
		return out.Bytes()
	}
	const rel = "semantic/sales/order.tei"
	original, err := os.ReadFile(filepath.Join(before, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatal(err)
	}
	const anchor = "    attribute externalId type std.text\n"
	if bytes.Count(original, []byte(anchor)) != 1 {
		t.Fatal("reference edit anchor must be unique")
	}
	run(tei, 0, "validate", "-workspace", before)
	var locked struct {
		Result json.RawMessage `json:"result"`
	}
	if err := json.Unmarshal(run(tei, 0, "plugin-lock", "-workspace", before), &locked); err != nil || len(locked.Result) == 0 {
		t.Fatalf("lock: %v", err)
	}
	lockPath := filepath.Join(binDir, "development-lock.json")
	if err := os.WriteFile(lockPath, locked.Result, 0o644); err != nil {
		t.Fatal(err)
	}
	var selected struct {
		Result struct {
			Plugins []json.RawMessage `json:"plugins"`
		} `json:"result"`
	}
	if err := json.Unmarshal(run(tei, 0, "plugin-resolve", "-workspace", before, "-environment", "development", "-plugin-lock", lockPath), &selected); err != nil || len(selected.Result.Plugins) == 0 {
		t.Fatalf("selection: %+v, %v", selected, err)
	}
	changed := bytes.Replace(original, []byte(anchor), []byte(anchor+"    attribute reviewNote type std.text\n"), 1)
	if err := os.WriteFile(filepath.Join(after, filepath.FromSlash(rel)), changed, 0o644); err != nil {
		t.Fatal(err)
	}
	run(tei, 0, "validate", "-workspace", after)
	run(tei, 0, "plugin-resolve", "-workspace", after, "-environment", "development", "-plugin-lock", lockPath)
	var updated struct {
		Result json.RawMessage `json:"result"`
	}
	if err := json.Unmarshal(run(tei, 0, "plugin-lock", "-workspace", after), &updated); err != nil || !bytes.Equal(locked.Result, updated.Result) {
		t.Fatalf("business edit changed plugin lock: %v", err)
	}
	var comparison applicationMembershipComparison
	if err := json.Unmarshal(run(compare, 0, "compare-application-membership", "-before", before, "-after", after), &comparison); err != nil {
		t.Fatal(err)
	}
	if comparison.Membership == nil || len(comparison.StrictChanges) != 0 || len(comparison.Membership.Changes) != 0 || len(comparison.SemanticChanges) != 1 || comparison.SemanticChanges[0].ID != "sales.order.review_note" {
		t.Fatalf("unexpected comparison: %+v", comparison)
	}
	oldWorld, err := workspace.Load(before)
	if err != nil {
		t.Fatal(err)
	}
	newWorld, err := workspace.Load(after)
	if err != nil {
		t.Fatal(err)
	}
	edges, unresolved := changeanalysis.ImpactWithBefore(oldWorld.World, newWorld.World, comparison.SemanticChanges)
	foundOwner := false
	for _, edge := range edges {
		foundOwner = foundOwner || (edge.From == "attribute:sales.order.review_note" && edge.To == "data:sales.order")
	}
	if !foundOwner || len(unresolved) == 0 {
		t.Fatalf("expected owner and unresolved impact scope: edges=%+v unresolved=%+v", edges, unresolved)
	}
	unknown := map[string]bool{}
	for _, finding := range unresolved {
		unknown[finding.Code] = true
	}
	if !unknown["DRIVER_SCENARIO_NOT_RUN"] || !unknown["TARGET_PROJECTION_NOT_EVALUATED"] {
		t.Fatalf("impact coverage limitations missing: %+v", unresolved)
	}
	fileCount := 0
	if err := filepath.WalkDir(before, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		relative, err := filepath.Rel(before, path)
		if err != nil {
			return err
		}
		left, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		right, err := os.ReadFile(filepath.Join(after, relative))
		if err != nil {
			return err
		}
		if !bytes.Equal(left, right) {
			fileCount++
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if fileCount != 1 {
		t.Fatalf("expected one changed file, got %d", fileCount)
	}
	invalid := bytes.Replace(changed, []byte("attribute reviewNote type std.text"), []byte("attribute reviewNote type sales.unknown_type"), 1)
	if err := os.WriteFile(filepath.Join(after, filepath.FromSlash(rel)), invalid, 0o644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"plugin-lock", "-workspace", after}, {"plugin-resolve", "-workspace", after, "-environment", "development", "-plugin-lock", lockPath}, {"plugin-runtime-plan", "-workspace", after, "-environment", "development", "-plugin-lock", lockPath}} {
		if out := run(tei, 1, args...); !bytes.Contains(out, []byte("UNKNOWN_SEMANTIC_ATTRIBUTE_TYPE")) {
			t.Fatalf("missing diagnostic for %v: %s", args, out)
		}
	}
	if out := run(tei, 1, "plugin-resolve", "-workspace", after, "-environment", "development", "-plugin-lock", lockPath, "-diagnostic-candidates"); !bytes.Contains(out, []byte(`"workspaceValidated": false`)) || !bytes.Contains(out, []byte("UNKNOWN_SEMANTIC_ATTRIBUTE_TYPE")) || !bytes.Contains(out, []byte(`"candidates"`)) {
		t.Fatalf("diagnostic: %s", out)
	}
	var invalidReport applicationMembershipComparison
	if err := json.Unmarshal(run(compare, 1, "compare-application-membership", "-before", before, "-after", after), &invalidReport); err != nil || invalidReport.Membership != nil {
		t.Fatalf("invalid comparison: %+v, %v", invalidReport, err)
	}
	validationError := false
	for _, diagnostic := range invalidReport.AfterDiagnostics {
		validationError = validationError || diagnostic.Code == "UNKNOWN_SEMANTIC_ATTRIBUTE_TYPE"
	}
	if !validationError {
		t.Fatalf("comparison hid the invalid model diagnostic: %+v", invalidReport)
	}

	// A distinct Application edit: reorder only namespace declarations. The
	// separate strict comparison remains authoritative for the acceptance path.
	if err := os.WriteFile(filepath.Join(after, filepath.FromSlash(rel)), original, 0o644); err != nil {
		t.Fatal(err)
	}
	const appRel = "applications/order-entry/application.tei"
	appOriginal, err := os.ReadFile(filepath.Join(before, filepath.FromSlash(appRel)))
	if err != nil {
		t.Fatal(err)
	}
	const useOrder = "    uses semantic sales\n    uses semantic customer\n"
	if bytes.Count(appOriginal, []byte(useOrder)) != 1 {
		t.Fatal("application use anchor must be unique")
	}
	appReordered := bytes.Replace(appOriginal, []byte(useOrder), []byte("    uses semantic customer\n    uses semantic sales\n"), 1)
	appPath := filepath.Join(after, filepath.FromSlash(appRel))
	if err := os.WriteFile(appPath, appReordered, 0o644); err != nil {
		t.Fatal(err)
	}
	run(tei, 0, "validate", "-workspace", after)
	run(tei, 0, "plugin-resolve", "-workspace", after, "-environment", "development", "-plugin-lock", lockPath)
	var reordered applicationMembershipComparison
	if err := json.Unmarshal(run(compare, 0, "compare-application-membership", "-before", before, "-after", after), &reordered); err != nil {
		t.Fatal(err)
	}
	if !reordered.WorkspaceValidationPassed || len(reordered.StrictChanges) != 1 || reordered.Membership == nil || len(reordered.Membership.Changes) != 0 || len(reordered.SemanticChanges) != 0 {
		t.Fatalf("reference application reorder: %+v", reordered)
	}
	const validationOrder = "    uses validation sales.order_line.quantity_positive\n    uses validation sales.order.lines.line_number_unique\n"
	if bytes.Count(appReordered, []byte(validationOrder)) != 1 {
		t.Fatal("validation order anchor must be unique")
	}
	changedValidation := bytes.Replace(appReordered, []byte(validationOrder), []byte("    uses validation sales.order.lines.line_number_unique\n    uses validation sales.order_line.quantity_positive\n"), 1)
	if err := os.WriteFile(appPath, changedValidation, 0o644); err != nil {
		t.Fatal(err)
	}
	run(tei, 0, "validate", "-workspace", after)
	var validationChange applicationMembershipComparison
	if err := json.Unmarshal(run(compare, 0, "compare-application-membership", "-before", before, "-after", after), &validationChange); err != nil || validationChange.Membership == nil || len(validationChange.Membership.Changes) != 1 {
		t.Fatalf("ordered validation change: %+v, %v", validationChange, err)
	}
	badApp := bytes.Replace(appReordered, []byte("uses semantic sales"), []byte("uses semantic missing_namespace"), 1)
	if err := os.WriteFile(appPath, badApp, 0o644); err != nil {
		t.Fatal(err)
	}
	var blocked applicationMembershipComparison
	if err := json.Unmarshal(run(compare, 1, "compare-application-membership", "-before", before, "-after", after), &blocked); err != nil || blocked.Membership != nil || blocked.WorkspaceValidationPassed {
		t.Fatalf("default must block invalid model: %+v, %v", blocked, err)
	}
	var diagnosticReport applicationMembershipComparison
	if err := json.Unmarshal(run(compare, 1, "compare-application-membership", "-before", before, "-after", after, "-include-unvalidated-candidates"), &diagnosticReport); err != nil {
		t.Fatal(err)
	}
	if diagnosticReport.WorkspaceValidationPassed || diagnosticReport.Membership == nil || len(diagnosticReport.Membership.Changes) != 1 || !strings.Contains(strings.Join(diagnosticReport.AuthorityLimits, ","), "workspace-validation-errors-candidates-unverified") {
		t.Fatalf("unvalidated candidate: %+v", diagnosticReport)
	}
	foundUnknownNamespace := false
	for _, d := range diagnosticReport.AfterDiagnostics {
		foundUnknownNamespace = foundUnknownNamespace || d.Code == "UNKNOWN_APPLICATION_SEMANTIC_NAMESPACE"
	}
	if !foundUnknownNamespace {
		t.Fatalf("lost namespace diagnostic: %+v", diagnosticReport)
	}
	if out := run(tei, 1, "plugin-resolve", "-workspace", after, "-environment", "development", "-plugin-lock", lockPath); !bytes.Contains(out, []byte("WORKSPACE_VALIDATION_FAILED")) {
		t.Fatalf("invalid application plugin selection: %s", out)
	}
	if out := run(tei, 1, "plugin-resolve", "-workspace", after, "-environment", "development", "-plugin-lock", lockPath, "-diagnostic-candidates"); !bytes.Contains(out, []byte(`"workspaceValidated": false`)) || !bytes.Contains(out, []byte("UNKNOWN_APPLICATION_SEMANTIC_NAMESPACE")) {
		t.Fatalf("invalid application plugin diagnostics: %s", out)
	}
	actualApp, err := os.ReadFile(filepath.Join(repositoryRoot(t), "reference", "workspace", filepath.FromSlash(appRel)))
	if err != nil || !bytes.Equal(actualApp, appOriginal) {
		t.Fatalf("reference application changed during isolated test: %v", err)
	}
	t.Log("reference application: 1 file reordered; strict diff=1 and membership candidate=0; validation order change=1; missing namespace blocks normal paths and opt-in diagnostic remains non-success")

	// A third case changes a shared type without rewriting its consumers.
	if err := os.WriteFile(appPath, appOriginal, 0o644); err != nil {
		t.Fatal(err)
	}
	const oldType = "type OrderId from std.identifier"
	if bytes.Count(original, []byte(oldType)) != 1 {
		t.Fatal("type anchor must be unique")
	}
	typeChanged := bytes.Replace(original, []byte(oldType), []byte("type OrderId from std.text"), 1)
	if err := os.WriteFile(filepath.Join(after, filepath.FromSlash(rel)), typeChanged, 0o644); err != nil {
		t.Fatal(err)
	}
	run(tei, 0, "validate", "-workspace", after)
	run(tei, 0, "plugin-resolve", "-workspace", after, "-environment", "development", "-plugin-lock", lockPath)
	var typeReport applicationMembershipComparison
	if err := json.Unmarshal(run(compare, 0, "compare-application-membership", "-before", before, "-after", after), &typeReport); err != nil {
		t.Fatal(err)
	}
	if len(typeReport.SemanticChanges) != 1 || typeReport.SemanticChanges[0].Kind != "type" || typeReport.SemanticChanges[0].ID != "sales.order_id" {
		t.Fatalf("type change not found: %+v", typeReport.SemanticChanges)
	}
	wantEdges := map[string]bool{"attribute:sales.order.order_id": false, "contract-query:sales_core.order_line/find_by_order": false}
	for _, edge := range typeReport.SemanticImpact {
		if edge.From == "type:sales.order_id" && !strings.HasPrefix(edge.Via, "previous.") {
			if _, ok := wantEdges[edge.To]; ok {
				wantEdges[edge.To] = true
			}
		}
	}
	for target, found := range wantEdges {
		if !found {
			t.Fatalf("missing actual type consumer %s: %+v", target, typeReport.SemanticImpact)
		}
	}
	unknownTypeImpact := false
	for _, finding := range typeReport.SemanticImpactUnresolved {
		unknownTypeImpact = unknownTypeImpact || finding.Code == "SEMANTIC_TYPE_IMPACT_INCOMPLETE"
	}
	if !unknownTypeImpact {
		t.Fatalf("type analysis scope was not marked incomplete: %+v", typeReport.SemanticImpactUnresolved)
	}
	if err := json.Unmarshal(run(tei, 0, "plugin-lock", "-workspace", after), &updated); err != nil || !bytes.Equal(locked.Result, updated.Result) {
		t.Fatalf("type change altered plugin lock: %v", err)
	}
	contractRel := "databases/sales-core/contracts/order-line.tei"
	contractBefore, err := os.ReadFile(filepath.Join(before, filepath.FromSlash(contractRel)))
	if err != nil {
		t.Fatal(err)
	}
	contractAfter, err := os.ReadFile(filepath.Join(after, filepath.FromSlash(contractRel)))
	if err != nil || !bytes.Equal(contractBefore, contractAfter) {
		t.Fatalf("consumer contract was rewritten: %v", err)
	}
	badType := bytes.Replace(original, []byte(oldType), []byte("type OrderId from std.not_present"), 1)
	if err := os.WriteFile(filepath.Join(after, filepath.FromSlash(rel)), badType, 0o644); err != nil {
		t.Fatal(err)
	}
	if out := run(tei, 1, "plugin-resolve", "-workspace", after, "-environment", "development", "-plugin-lock", lockPath); !bytes.Contains(out, []byte("UNKNOWN_SEMANTIC_TYPE_UNDERLYING")) {
		t.Fatalf("unknown underlying type not rejected: %s", out)
	}
	var invalidType applicationMembershipComparison
	if err := json.Unmarshal(run(compare, 1, "compare-application-membership", "-before", before, "-after", after), &invalidType); err != nil || invalidType.Membership != nil || invalidType.WorkspaceValidationPassed {
		t.Fatalf("invalid type comparison: %+v, %v", invalidType, err)
	}
	t.Logf("reference type edit: 1 source changed, existing attribute and Contract Query references retained; %d direct/previous impact edges; incomplete coverage explicit; unchanged plugin lock; unknown base rejected", len(typeReport.SemanticImpact))
	// Change a Database contract independently of Semantic/Application.
	if err := os.WriteFile(filepath.Join(after, filepath.FromSlash(rel)), original, 0o644); err != nil {
		t.Fatal(err)
	}
	const queryInput = "input order_id type sales.order_id"
	if bytes.Count(contractBefore, []byte(queryInput)) != 1 {
		t.Fatal("contract input anchor not unique")
	}
	contractPath := filepath.Join(after, filepath.FromSlash(contractRel))
	if bytes.Count(contractBefore, []byte("contract sales_core.order_line v1.0")) != 1 {
		t.Fatal("contract version anchor not unique")
	}
	contractEdit := bytes.Replace(contractBefore, []byte("contract sales_core.order_line v1.0"), []byte("contract sales_core.order_line v1.1"), 1)
	if err := os.WriteFile(contractPath, contractEdit, 0o644); err != nil {
		t.Fatal(err)
	}
	run(tei, 0, "validate", "-workspace", after)
	run(tei, 0, "plugin-resolve", "-workspace", after, "-environment", "development", "-plugin-lock", lockPath)
	var contractReport applicationMembershipComparison
	if err := json.Unmarshal(run(compare, 0, "compare-application-membership", "-before", before, "-after", after), &contractReport); err != nil {
		t.Fatal(err)
	}
	if len(contractReport.SemanticChanges) != 0 || len(contractReport.StrictChanges) != 0 || contractReport.Membership == nil || len(contractReport.Membership.Changes) != 0 || len(contractReport.ChangedCollections) != 1 || contractReport.ChangedCollections[0] != "databases" {
		t.Fatalf("contract collection change hidden or misclassified: %+v", contractReport)
	}
	if len(contractReport.ContractCandidates) != 1 || contractReport.ContractCandidates[0].Database != "sales_core" || contractReport.ContractCandidates[0].Contract != "sales_core.order_line" || contractReport.ContractCandidates[0].Change != "changed" {
		t.Fatalf("unexpected contract candidates: %+v", contractReport.ContractCandidates)
	}
	observed := map[string]int{}
	for _, consumer := range contractReport.ContractCandidates[0].Consumers {
		observed[consumer.Snapshot+"/"+consumer.Kind]++
		if consumer.Kind != "application-contract-requirement" && consumer.Query != "find_by_order" {
			t.Fatalf("unrelated Query candidate: %+v", consumer)
		}
	}
	for _, snapshot := range []string{"before", "after"} {
		for _, kind := range []string{"application-contract-requirement", "header-detail-binding", "executor-binding", "sql-realization"} {
			if observed[snapshot+"/"+kind] != 1 {
				t.Fatalf("missing or duplicate candidate %s/%s: %+v", snapshot, kind, observed)
			}
		}
	}
	if len(contractReport.ContractCandidates[0].Consumers) != 8 || !strings.Contains(strings.Join(contractReport.AuthorityLimits, ","), "contract-compatibility-releases-dynamic-and-runtime-effects-unverified") {
		t.Fatalf("candidate coverage or limits: %+v", contractReport)
	}
	// A reference Pattern's explicit contract-query invocation is a consumer too.
	patternBefore, patternAfter := scratchWorkspace(t), scratchWorkspace(t)
	const customerRel = "databases/master-data/contracts/customer.tei"
	customerSource, err := os.ReadFile(filepath.Join(patternBefore, filepath.FromSlash(customerRel)))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Count(customerSource, []byte("contract master_data.customer v2.4")) != 1 {
		t.Fatal("customer contract version anchor not unique")
	}
	customerEdit := bytes.Replace(customerSource, []byte("contract master_data.customer v2.4"), []byte("contract master_data.customer v2.5"), 1)
	if err := os.WriteFile(filepath.Join(patternAfter, filepath.FromSlash(customerRel)), customerEdit, 0o644); err != nil {
		t.Fatal(err)
	}
	run(tei, 0, "validate", "-workspace", patternAfter)
	var patternReport applicationMembershipComparison
	if err := json.Unmarshal(run(compare, 0, "compare-application-membership", "-before", patternBefore, "-after", patternAfter), &patternReport); err != nil {
		t.Fatal(err)
	}
	if len(patternReport.ContractCandidates) != 1 || patternReport.ContractCandidates[0].Contract != "master_data.customer" {
		t.Fatalf("unexpected pattern contract candidates: %+v", patternReport.ContractCandidates)
	}
	invocations := map[string]int{}
	for _, consumer := range patternReport.ContractCandidates[0].Consumers {
		if consumer.Kind != "application-pattern-contract-query" {
			continue
		}
		if consumer.Query != "find_by_id" {
			t.Fatalf("unexpected pattern invocation candidate: %+v", consumer)
		}
		invocations[consumer.ID]++
	}
	for _, id := range []string{"order_entry/order_entry.customer_reference", "repair_intake/repair_intake.customer_reference"} {
		if invocations[id] != 2 {
			t.Fatalf("pattern invocation %s candidates=%d, want before/after each: %+v", id, invocations[id], patternReport.ContractCandidates[0].Consumers)
		}
	}
	if len(invocations) != 2 {
		t.Fatalf("unexpected pattern invocation set: %+v", invocations)
	}
	t.Log("reference pattern contract-query: master_data.customer version edit reports both applications' customer_reference/find_by_id in both snapshots")
	contractWorld, err := workspace.Load(after)
	if err != nil {
		t.Fatal(err)
	}
	target := realization.ContractQueryTarget{Database: "sales_core", Contract: "sales_core.order_line", Query: "find_by_order"}
	countReferences := func(target realization.ContractQueryTarget) [4]int {
		var counts [4]int
		for _, app := range contractWorld.World.Applications {
			for _, db := range app.Databases {
				for _, requirement := range db.RequiredContracts {
					if db.TargetDatabase == target.Database && requirement.ContractID == target.Contract {
						counts[0]++
					}
				}
			}
		}
		for _, b := range contractWorld.World.HeaderDetailContractQueryBindings {
			if b.Target == target {
				counts[1]++
			}
		}
		for _, b := range contractWorld.World.ContractQueryExecutorBindings {
			if b.Target == target {
				counts[2]++
			}
		}
		for _, r := range contractWorld.World.ContractQuerySQLRealizations {
			if r.Target == target {
				counts[3]++
			}
		}
		return counts
	}
	if got := countReferences(target); got != [4]int{1, 1, 1, 1} {
		t.Fatalf("expected exact declared consumers: %v", got)
	}
	other := target
	other.Contract = "sales_core.nonexistent"
	if got := countReferences(other); got != [4]int{} {
		t.Fatalf("unrelated contract matched: %v", got)
	}
	other = target
	other.Query = "other_query"
	if got := countReferences(other); got != [4]int{1, 0, 0, 0} {
		t.Fatalf("query identity must distinguish query-level bindings while retaining contract requirement: %v", got)
	}
	for _, path := range []string{
		"applications/order-entry/application.tei",
		"applications/order-entry/realizations/header-detail/order-lines.tei",
		"environments/development/contract-query-executors/order-line-find-by-order.tei",
		"databases/sales-core/query-realizations/order-line-find-by-order.tei",
	} {
		oldBytes, err := os.ReadFile(filepath.Join(before, filepath.FromSlash(path)))
		if err != nil {
			t.Fatal(err)
		}
		newBytes, err := os.ReadFile(filepath.Join(after, filepath.FromSlash(path)))
		if err != nil || !bytes.Equal(oldBytes, newBytes) {
			t.Fatalf("consumer source changed %s: %v", path, err)
		}
	}
	mismatched := bytes.Replace(contractBefore, []byte(queryInput), []byte("input order_id type std.text"), 1)
	if err := os.WriteFile(contractPath, mismatched, 0o644); err != nil {
		t.Fatal(err)
	}
	if out := run(tei, 1, "plugin-resolve", "-workspace", after, "-environment", "development", "-plugin-lock", lockPath); !bytes.Contains(out, []byte("HEADER_DETAIL_BINDING_INPUT_TYPE_MISMATCH")) {
		t.Fatalf("existing binding mismatch not rejected: %s", out)
	}
	renamedInput := bytes.Replace(contractBefore, []byte(queryInput), []byte("input new_order_id type sales.order_id"), 1)
	if err := os.WriteFile(contractPath, renamedInput, 0o644); err != nil {
		t.Fatal(err)
	}
	diagnostics := run(tei, 1, "validate", "-workspace", after, "-json")
	for _, code := range []string{"HEADER_DETAIL_BINDING_INPUT_NOT_IN_SIGNATURE", "INVALID_CONTRACT_QUERY_SQL_REALIZATION"} {
		if !bytes.Contains(diagnostics, []byte(code)) {
			t.Fatalf("input rename hid %s: %s", code, diagnostics)
		}
	}
	badContract := bytes.Replace(contractBefore, []byte(queryInput), []byte("input order_id type sales.not_found"), 1)
	if err := os.WriteFile(contractPath, badContract, 0o644); err != nil {
		t.Fatal(err)
	}
	if out := run(tei, 1, "plugin-resolve", "-workspace", after, "-environment", "development", "-plugin-lock", lockPath); !bytes.Contains(out, []byte("WORKSPACE_VALIDATION_FAILED")) {
		t.Fatalf("unknown query type must fail: %s", out)
	}
	var contractInvalid applicationMembershipComparison
	if err := json.Unmarshal(run(compare, 1, "compare-application-membership", "-before", before, "-after", after), &contractInvalid); err != nil || contractInvalid.Membership != nil || len(contractInvalid.ChangedCollections) != 0 || len(contractInvalid.ContractCandidates) != 0 {
		t.Fatalf("invalid contract comparison: %+v, %v", contractInvalid, err)
	}
	var contractDiagnostic applicationMembershipComparison
	if err := json.Unmarshal(run(compare, 1, "compare-application-membership", "-before", before, "-after", after, "-include-unvalidated-candidates"), &contractDiagnostic); err != nil || contractDiagnostic.WorkspaceValidationPassed || len(contractDiagnostic.AfterDiagnostics) == 0 || len(contractDiagnostic.ChangedCollections) != 1 || contractDiagnostic.ChangedCollections[0] != "databases" {
		t.Fatalf("unvalidated contract comparison: %+v, %v", contractDiagnostic, err)
	}
	if len(contractDiagnostic.ContractCandidates) != 1 || contractDiagnostic.ContractCandidates[0].Contract != "sales_core.order_line" || !strings.Contains(strings.Join(contractDiagnostic.AuthorityLimits, ","), "workspace-validation-errors-candidates-unverified") {
		t.Fatalf("unvalidated contract candidates not explicitly marked: %+v", contractDiagnostic)
	}
	t.Log("reference contract minor-version edit: 1 file; exact consumers: 1 Application requirement, 1 header-detail binding, 1 executor binding, 1 SQL realization; their sources unchanged; mismatch and unknown type rejected")
	actual, err := os.ReadFile(filepath.Join(repositoryRoot(t), "reference", "workspace", filepath.FromSlash(rel)))
	if err != nil || !bytes.Equal(actual, original) {
		t.Fatalf("reference source changed during isolated test: %v", err)
	}
	t.Logf("reference copy: %d edited file; 1 added attribute; %d development plugin descriptors; unchanged lock; %d impact edges / %d unresolved; invalid type blocks lock, resolve, runtime plan and comparison; diagnostic remains unvalidated", fileCount, len(selected.Result.Plugins), len(edges), len(unresolved))
	if !strings.Contains(string(invalid), "sales.unknown_type") {
		t.Fatal("invalid fixture was not written")
	}
}
