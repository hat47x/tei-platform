//go:build !tei_order_only && !tei_repair_only

package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"testing"
)

func sourceFilesForBuildTags(t *testing.T, tags string) map[string]bool {
	t.Helper()
	args := []string{"list", "-json"}
	if tags != "" {
		args = append(args, "-tags="+tags)
	}
	args = append(args, ".")
	command := exec.Command("go", args...)
	command.Env = append(os.Environ(), "GOENV=off", "GOWORK=off", "GOFLAGS=")
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("go %v failed: %v\n%s", args, err, output)
	}
	var pkg struct {
		Name       string
		GoFiles    []string
		CgoFiles   []string
		Incomplete bool
		Error      *struct{ Err string }
	}
	if err := json.Unmarshal(output, &pkg); err != nil {
		t.Fatalf("decode go list %q: %v", tags, err)
	}
	if pkg.Name != "main" || pkg.Incomplete || pkg.Error != nil {
		t.Fatalf("invalid %q build package: %+v", tags, pkg)
	}
	result := make(map[string]bool)
	for _, name := range append(pkg.GoFiles, pkg.CgoFiles...) {
		if result[name] {
			t.Fatalf("%q lists compiled source %s more than once", tags, name)
		}
		result[name] = true
	}
	return result
}

func TestFirstWebTargetApplicationBuildTagsSelectDisjointBusinessSources(t *testing.T) {
	common := []string{
		"server.go", "scenario_contracts.go", "projection.go",
		"shared_semantic_projection.go", "reference_contract_query_fixture.go",
	}
	orderSources := []string{
		"read_scenario.go", "command_scenario.go", "custom_ui.go",
		"order_scenario_http.go", "main_order_only.go", "order_only_wiring.go",
		"reference_order_contract_query_fixture.go",
	}
	repairSources := []string{
		"customer_lookup_scenario.go", "repair_scenario_http.go",
		"main_repair_only.go", "repair_only_wiring.go",
	}
	legacySources := []string{
		"main.go", "scoped_routes.go",
		"combined_scenario_routes.go", "combined_scenario_templates.go",
	}
	for _, tc := range []struct {
		name, tags       string
		include, exclude []string
	}{
		{
			name: "combined Gate C default",
			include: append(append(append([]string(nil), common...), legacySources...),
				"read_scenario.go", "command_scenario.go", "custom_ui.go",
				"order_scenario_http.go", "customer_lookup_scenario.go", "repair_scenario_http.go",
				"reference_order_contract_query_fixture.go"),
			exclude: []string{"main_order_only.go", "main_repair_only.go", "order_only_wiring.go", "repair_only_wiring.go",
				"single_application_startup.go"},
		},
		{
			name: "order_entry only", tags: "tei_order_only",
			include: append(append([]string(nil), common...), append(orderSources, "single_application_startup.go")...),
			exclude: append(append([]string(nil), repairSources...), legacySources...),
		},
		{
			name: "repair_intake only", tags: "tei_repair_only",
			include: append(append([]string(nil), common...), append(repairSources, "single_application_startup.go")...),
			exclude: append(append([]string(nil), orderSources...), legacySources...),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			files := sourceFilesForBuildTags(t, tc.tags)
			for _, name := range tc.include {
				if !files[name] {
					t.Errorf("compiled build %q omitted required file %s", tc.tags, name)
				}
			}
			for _, name := range tc.exclude {
				if files[name] {
					t.Errorf("compiled build %q included forbidden file %s", tc.tags, name)
				}
			}
		})
	}
}

// Even when no successful package main is selected, "go list" alone can
// still return a syntactically valid package. Check that mutually exclusive
// flags never select any executable main or Application-specific routes.
func TestConflictingApplicationTagsSelectNoEntrypoint(t *testing.T) {
	files := sourceFilesForBuildTags(t, "tei_order_only,tei_repair_only")
	for _, forbidden := range []string{
		"main.go", "main_order_only.go", "main_repair_only.go",
		"single_application_startup.go",
		"combined_scenario_routes.go", "combined_scenario_templates.go",
		"order_only_wiring.go", "repair_only_wiring.go",
		"read_scenario.go", "command_scenario.go",
		"customer_lookup_scenario.go", "order_scenario_http.go",
		"repair_scenario_http.go",
	} {
		if files[forbidden] {
			t.Errorf("conflicting Application tags unexpectedly selected %s", forbidden)
		}
	}
}
