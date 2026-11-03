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
	actual, err := os.ReadFile(filepath.Join(repositoryRoot(t), "reference", "workspace", filepath.FromSlash(rel)))
	if err != nil || !bytes.Equal(actual, original) {
		t.Fatalf("reference source changed during isolated test: %v", err)
	}
	t.Logf("reference copy: %d edited file; 1 added attribute; %d development plugin descriptors; unchanged lock; %d impact edges / %d unresolved; invalid type blocks lock, resolve, runtime plan and comparison; diagnostic remains unvalidated", fileCount, len(selected.Result.Plugins), len(edges), len(unresolved))
	if !strings.Contains(string(invalid), "sales.unknown_type") {
		t.Fatal("invalid fixture was not written")
	}
}
