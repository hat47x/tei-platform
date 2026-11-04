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
	actual, err := os.ReadFile(filepath.Join(repositoryRoot(t), "reference", "workspace", filepath.FromSlash(rel)))
	if err != nil || !bytes.Equal(actual, original) {
		t.Fatalf("reference source changed during isolated test: %v", err)
	}
	t.Logf("reference copy: %d edited file; 1 added attribute; %d development plugin descriptors; unchanged lock; %d impact edges / %d unresolved; invalid type blocks lock, resolve, runtime plan and comparison; diagnostic remains unvalidated", fileCount, len(selected.Result.Plugins), len(edges), len(unresolved))
	if !strings.Contains(string(invalid), "sales.unknown_type") {
		t.Fatal("invalid fixture was not written")
	}
}
