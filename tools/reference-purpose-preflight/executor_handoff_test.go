package main

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"testing"

	"tei/internal/environment"
	"tei/internal/ir"
	"tei/internal/plugin"
	"tei/internal/realization"
)

func TestRealContractQueryExecutorHandoffDoesNotAuthorizeRuntime(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "workspace")
	for _, tc := range []struct {
		env, wantStatus, wantPlugin string
	}{
		{"development", "resolved-declaration", "tei.database.oracle"},
		{"production", "not-declared", ""},
	} {
		t.Run(tc.env, func(t *testing.T) {
			report, err := InspectWorkspaceForEnvironment(root, "order_entry", "application-pattern", "order_entry.customer_reference", tc.env)
			if err != nil {
				t.Fatal(err)
			}
			handoff := report.ExecutorHandoff
			if handoff == nil || handoff.Status != tc.wantStatus ||
				handoff.Plugin != tc.wantPlugin || report.ExecutionAllowed {
				t.Fatalf("executor declaration misrepresented: %+v", report)
			}
			if tc.env == "development" &&
				(handoff.Executor != "database.query.oracle" || handoff.PluginVersion != "1.0.0") {
				t.Fatalf("wrong reference executor/Plugin identity: %+v", handoff)
			}
			readiness, err := ProjectForPurpose(report, "execute-preflight")
			if err != nil {
				t.Fatal(err)
			}
			ob, ok := purposeRequirementByName(readiness, "query-executor")
			expected := "conditional"
			if tc.env == "development" {
				expected = "not-evaluated"
			}
			if !ok || ob.Status != expected || readiness.ExecutionAllowed {
				t.Fatalf("static declaration became Provider Conformance or execution: %+v", readiness)
			}
		})
	}
}

func sampleQueryExecutorWorld() ir.World {
	world := syntheticEnvironmentBindingWorld()
	world.Plugins = []plugin.Definition{{
		ID: "tei.database.oracle", Version: "1.0.0",
		Runtime: plugin.Runtime{Kind: plugin.RuntimeBuiltin},
		Provides: []plugin.Capability{{
			Kind:              plugin.CapabilityContractQueryExecutor,
			ID:                "database.query.oracle",
			DatabaseProviders: []environment.DatabaseProvider{environment.ProviderOracle},
		}},
	}}
	world.Environments[0].Plugins = []environment.ExtensionRequirement{{ID: "tei.database.oracle", CompatibleMajor: 1}}
	world.ContractQueryExecutorBindings = []realization.ContractQueryExecutorBinding{{
		Environment: "development",
		Instance:    "master_data.dev",
		Target: realization.ContractQueryTarget{
			Database: "master_data", Contract: "master_data.customer", Query: "find_by_id",
		},
		Executor: "database.query.oracle",
	}}
	return world
}

func TestExistingExecutorResolverRejectsAmbiguityAndUnsupportedProvider(t *testing.T) {
	cases := []struct {
		name   string
		change func(*ir.World)
		want   string
	}{
		{"valid", func(*ir.World) {}, "resolved-declaration"},
		{"no-binding", func(w *ir.World) {
			w.ContractQueryExecutorBindings = nil
		}, "not-declared"},
		{"duplicate-binding", func(w *ir.World) {
			w.ContractQueryExecutorBindings = append(w.ContractQueryExecutorBindings, w.ContractQueryExecutorBindings[0])
		}, "unresolved"},
		{"missing-executor", func(w *ir.World) {
			w.ContractQueryExecutorBindings[0].Executor = "database.query.missing"
		}, "unresolved"},
		{"unsupported-provider", func(w *ir.World) {
			w.Plugins[0].Provides[0].DatabaseProviders = []environment.DatabaseProvider{environment.ProviderPostgres}
		}, "unresolved"},
		{"missing-plugin", func(w *ir.World) {
			w.Plugins = nil
		}, "unresolved"},
		{"binding-different-target", func(w *ir.World) {
			w.ContractQueryExecutorBindings[0].Target.Query = "different_query"
		}, "not-declared"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			world := sampleQueryExecutorWorld()
			tc.change(&world)
			req := ObserveDeclaredContractQuery(world, "example", "example.customer_reference")
			binding := ObserveEnvironmentBinding(world, "development", "example", &req)
			got := ObserveQueryExecutorHandoff(world, &req, &binding)
			if got.Status != tc.want {
				t.Fatalf("%s: expected %s, got %+v", tc.name, tc.want, got)
			}
			if got.Status == "resolved-declaration" &&
				(got.Plugin != "tei.database.oracle" || got.Executor != "database.query.oracle") {
				t.Fatalf("existing resolver selected wrong exact Plugin: %+v", got)
			}
		})
	}
}

func TestExecutorHandoffCLIOptInAndNoRuntimePermission(t *testing.T) {
	base := []string{
		"-workspace", filepath.Join("..", "..", "reference", "workspace"),
		"-application", "order_entry",
		"-subject-kind", "application-pattern",
		"-subject-id", "order_entry.customer_reference",
	}
	var old bytes.Buffer
	if err := runCLI(base, &old); err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(old.Bytes(), []byte("executorHandoff")) {
		t.Fatalf("new field leaked into legacy Workspace JSON: %s", old.String())
	}
	var buf bytes.Buffer
	args := append(append([]string{}, base...), "-purpose", "execute-preflight", "-environment", "development")
	if err := runCLI(args, &buf); err != nil {
		t.Fatal(err)
	}
	var got struct {
		Assessment WorkspaceAssessment `json:"assessment"`
		Readiness  PurposeProjection   `json:"readiness"`
	}
	if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.Assessment.ExecutorHandoff == nil ||
		got.Assessment.ExecutorHandoff.Status != "resolved-declaration" ||
		got.Assessment.ExecutionAllowed || got.Readiness.ExecutionAllowed {
		t.Fatalf("opt-in executor resolution was missing or executable: %+v", got)
	}
}
