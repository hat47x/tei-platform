// E26 measures only locally observable source declarations and the isolated
// Native candidate overlay. It does not estimate developer labor, AI prompts,
// real SUI integration work, authorization or production adoption.
package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/driver"
	"tei/internal/workspace"
)

type e26ObservedFile struct {
	Path          string `json:"path"`
	NonblankLines int    `json:"nonblankLines"`
}

type e26ObservedBaseline struct {
	Consumer              string            `json:"consumer"`
	Status                string            `json:"status"`
	ExistingFiles         []e26ObservedFile `json:"existingFiles,omitempty"`
	AddedTEISourceFiles   []e26ObservedFile `json:"addedTeiSourceFiles,omitempty"`
	ManifestSourceRoots   int               `json:"manifestSourceRoots,omitempty"`
	NativeInputAttributes int               `json:"nativeInputAttributes,omitempty"`
	AuthorityVerified     bool              `json:"authorityVerified"`
	DeployedToRealTarget  bool              `json:"deployedToRealTarget"`
}

func e26NonblankLines(src string) int {
	count := 0
	for _, line := range strings.Split(src, "\n") {
		if strings.TrimSpace(line) != "" {
			count++
		}
	}
	return count
}

func e26ObservedSource(t *testing.T, root, path string) e26ObservedFile {
	t.Helper()
	source, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
	if err != nil {
		t.Fatal(err)
	}
	return e26ObservedFile{Path: path, NonblankLines: e26NonblankLines(string(source))}
}

func e26LogBaseline(t *testing.T, report e26ObservedBaseline) {
	t.Helper()
	b, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("E26_OBSERVED_BASELINE %s", b)
}

func TestE26ExistingOrderWebSourceBaseline(t *testing.T) {
	// This counts existing sources, not new work required to adopt TEI.
	root := filepath.Join("..", "..", "reference", "workspace")
	paths := []string{
		"applications/order-entry/application.tei",
		"applications/order-entry/patterns/submit-order.tei",
		"applications/order-entry/realizations/submit-order.tei",
		"environments/development/environment.tei",
	}
	report := e26ObservedBaseline{Consumer: "order_entry_web", Status: "existing_source_only"}
	observed := []int{23, 3, 1, 32}
	for i, path := range paths {
		item := e26ObservedSource(t, root, path)
		if item.NonblankLines != observed[i] {
			t.Fatalf("source authoring baseline drift %s: got %d, was %d", path, item.NonblankLines, observed[i])
		}
		report.ExistingFiles = append(report.ExistingFiles, item)
	}
	ui := e26ObservedSource(t, filepath.Join("..", "..", "reference"), "ui-customization/web-react/project-extension.mjs")
	if ui.NonblankLines != 8 {
		t.Fatalf("project-owned UI baseline changed: %+v", ui)
	}
	report.ExistingFiles = append(report.ExistingFiles, ui)
	world, err := workspace.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := driver.NewResolver(world.World).ResolvePattern("order_entry", "order_entry.submit_order"); err != nil {
		t.Fatalf("recorded order command no longer resolves: %v", err)
	}
	e26LogBaseline(t, report)
}

func TestE26NativeOverlayAuthoringBaseline(t *testing.T) {
	stage0Root := filepath.Join("..", "..", "reference", "sui-adoption", "workspace")
	stage0, err := workspace.Load(stage0Root)
	if err != nil {
		t.Fatal(err)
	}
	if len(stage0.World.Applications) != 0 {
		t.Fatal("Stage 0 cannot be counted as an accepted SUI Application")
	}
	baseManifest := e26ObservedSource(t, stage0Root, "tei.workspace.yaml")
	if baseManifest.NonblankLines != 3 {
		t.Fatalf("SUI Stage 0 manifest baseline drift: %+v", baseManifest)
	}
	overlay := e22Workspace(t, true, false)
	candidate, err := workspace.Load(overlay)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := driver.NewResolver(candidate.World).ResolvePattern("sui", "sui.move"); err != nil {
		t.Fatalf("isolated Native Command candidate does not resolve: %v", err)
	}
	report := e26ObservedBaseline{
		Consumer:            "sui_native_candidate",
		Status:              "temporary_test_overlay_only",
		ExistingFiles:       []e26ObservedFile{baseManifest},
		ManifestSourceRoots: 2,
	}
	paths := []string{
		"semantic/sui/native-action.tei",
		"applications/sui/application.tei",
		"applications/sui/patterns/move.tei",
	}
	for _, path := range paths {
		report.AddedTEISourceFiles = append(report.AddedTEISourceFiles, e26ObservedSource(t, overlay, path))
	}
	for i, want := range []int{3, 3, 3} {
		if report.AddedTEISourceFiles[i].NonblankLines != want {
			t.Fatalf("E22 temporary overlay authoring changed: %+v", report.AddedTEISourceFiles[i])
		}
	}
	manifest := e26ObservedSource(t, overlay, "tei.workspace.yaml")
	if manifest.NonblankLines != 4 || !strings.Contains(mustReadE26(t, filepath.Join(overlay, "tei.workspace.yaml")), "  - applications\n") {
		t.Fatalf("E22 candidate manifest does not add applications source root: %+v", manifest)
	}
	// Reading Stage 0 island source into an isolated test Workspace is not
	// creating or accepting a second SUI-owned domain implementation.
	originalIsland := mustReadE26(t, filepath.Join(stage0Root, "semantic", "sui", "island.tei"))
	copiedIsland := mustReadE26(t, filepath.Join(overlay, "semantic", "sui", "island.tei"))
	if originalIsland != copiedIsland {
		t.Fatal("test fixture changed the SUI Stage 0 semantic source")
	}
	e26LogBaseline(t, report)
}

func mustReadE26(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestE26TypedNativeOverlayAddsStructureWithoutReplacingReceiver(t *testing.T) {
	overlay := e22Workspace(t, true, false)
	path := filepath.Join(overlay, "semantic", "sui", "native-action.tei")
	if err := os.WriteFile(path, []byte(e24TypedNativeSource), 0600); err != nil {
		t.Fatal(err)
	}
	loaded, err := workspace.Load(overlay)
	if err != nil {
		t.Fatal(err)
	}
	r, err := driver.NewResolver(loaded.World).ResolvePattern("sui", "sui.move")
	if err != nil || r.Capability == nil || len(r.Capability.Inputs) != 1 {
		t.Fatalf("typed Native candidate not resolved: %v", err)
	}
	if len(r.Capability.Inputs) != 1 || r.Capability.Output != "" || r.Capability.Effect != nil {
		t.Fatal("typed Source does not constitute persistence or a completed SUI domain")
	}
	report := e26ObservedBaseline{
		Consumer:              "sui_native_typed_candidate",
		Status:                "temporary_test_overlay_only",
		ManifestSourceRoots:   2,
		NativeInputAttributes: 3,
	}
	for _, path := range []string{
		"semantic/sui/native-action.tei",
		"applications/sui/application.tei",
		"applications/sui/patterns/move.tei",
	} {
		report.AddedTEISourceFiles = append(report.AddedTEISourceFiles, e26ObservedSource(t, overlay, path))
	}
	for i, want := range []int{9, 3, 3} {
		if report.AddedTEISourceFiles[i].NonblankLines != want {
			t.Fatalf("E24 typed candidate authoring baseline changed: %+v", report.AddedTEISourceFiles[i])
		}
	}
	e26LogBaseline(t, report)
}
