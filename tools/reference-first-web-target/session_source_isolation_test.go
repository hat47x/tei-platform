//go:build !tei_order_only && !tei_repair_only

package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"testing"
)

// A Go test of the default target does not prove that a tagged target omits
// sensitive source files. Check the actual go list selection for all profiles,
// including the selected tests, so a future untagged session file is caught.
func TestSessionCommandSourceAndTestIsolationAcrossApplicationTags(t *testing.T) {
	orderOnlySource := []string{
		"session_command_route.go",
		"session_command_scenario.go",
	}
	orderOnlyTests := []string{
		"session_command_route_test.go",
		"session_command_scenario_test.go",
		"session_effect_authorization_test.go",
		"session_effect_grant_contract_test.go",
		"session_result_contract_test.go",
	}
	for _, tc := range []struct {
		name      string
		tags      string
		hasOrder  bool
		hasRepair bool
	}{
		{name: "combined", hasOrder: true},
		{name: "order-only", tags: "tei_order_only", hasOrder: true},
		{name: "repair-only", tags: "tei_repair_only", hasRepair: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			args := []string{"list", "-json"}
			if tc.tags != "" {
				args = append(args, "-tags="+tc.tags)
			}
			args = append(args, ".")
			cmd := exec.Command("go", args...)
			cmd.Env = append(os.Environ(), "GOENV=off", "GOWORK=off", "GOFLAGS=")
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("go %v: %v\n%s", args, err, output)
			}
			var pkg struct {
				GoFiles      []string
				TestGoFiles  []string
				XTestGoFiles []string
				Incomplete   bool
				Error        *struct{ Err string }
			}
			if err := json.Unmarshal(output, &pkg); err != nil {
				t.Fatal(err)
			}
			if pkg.Incomplete || pkg.Error != nil {
				t.Fatalf("incomplete Go package under %q: %+v", tc.tags, pkg.Error)
			}
			source := make(map[string]bool)
			tests := make(map[string]bool)
			for _, file := range pkg.GoFiles {
				source[file] = true
			}
			for _, file := range append(pkg.TestGoFiles, pkg.XTestGoFiles...) {
				tests[file] = true
			}
			for _, file := range orderOnlySource {
				if source[file] != tc.hasOrder {
					t.Errorf("compiled order-only source %s selected=%t; want=%t", file, source[file], tc.hasOrder)
				}
			}
			for _, file := range orderOnlyTests {
				if tests[file] != tc.hasOrder {
					t.Errorf("compiled order-only test %s selected=%t; want=%t", file, tests[file], tc.hasOrder)
				}
			}
			const stub = "session_command_repair_stub.go"
			if source[stub] != tc.hasRepair {
				t.Errorf("repair-only fail-closed stub %s selected=%t; want=%t", stub, source[stub], tc.hasRepair)
			}
		})
	}
}
