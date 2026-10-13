package main

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"testing"

	"tei/internal/environment"
	"tei/internal/ir"
	"tei/internal/workspace"
)

func TestReferenceEnvironmentBindingDoesNotEstablishProviderAuthority(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "workspace")
	loaded, err := workspace.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	assessment, err := InspectWorkspace(root, "order_entry", "application-pattern", "order_entry.customer_reference")
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		environment string
		instance    string
	}{
		{"development", "master_data.dev"},
		{"production", "master_data.prod"},
	} {
		t.Run(tc.environment, func(t *testing.T) {
			binding := ObserveEnvironmentBinding(loaded.World, tc.environment, "order_entry", assessment.ContractQueryUse)
			if binding.Status != "declared-environment-binding" ||
				binding.Instance != tc.instance || binding.Definition != "master_data" ||
				binding.DatabaseProvider != "oracle" {
				t.Fatalf("expected exact environment source declaration: %+v", binding)
			}
			copy := assessment
			copy.EnvironmentBinding = &binding
			projection, err := ProjectForPurpose(copy, "execute-preflight")
			if err != nil {
				t.Fatal(err)
			}
			provider, ok := purposeRequirementByName(projection, "provider-contract")
			if !ok || provider.Status != "not-evaluated" ||
				provider.Reason != "environment-binding-declared-provider-conformance-unverified" {
				t.Fatalf("declared provider label became conformance: %+v", provider)
			}
			if projection.ExecutionAllowed || copy.ExecutionAllowed {
				t.Fatal("declaration was promoted to execution authority")
			}
		})
	}
}

func syntheticEnvironmentBindingWorld() ir.World {
	world := referenceIntentFixture()
	world.Environments = []environment.Environment{{
		ID: "development",
		Instances: []environment.DatabaseInstance{{
			ID: "master_data.dev", Definition: "master_data", Provider: environment.ProviderOracle,
		}},
		Bindings: []environment.ApplicationDatabaseBinding{{
			Application: "example", Alias: "master", Instance: "master_data.dev",
		}},
	}}
	return world
}

func TestEnvironmentBindingMustBeExactAndUnambiguous(t *testing.T) {
	type mutation func(*ir.World)
	for _, tc := range []struct {
		name   string
		mutate mutation
		status string
	}{
		{"valid", func(*ir.World) {}, "declared-environment-binding"},
		{"duplicate-environment", func(w *ir.World) {
			w.Environments = append(w.Environments, w.Environments[0])
		}, "unresolved"},
		{"missing-binding", func(w *ir.World) {
			w.Environments[0].Bindings = nil
		}, "unresolved"},
		{"duplicate-binding", func(w *ir.World) {
			w.Environments[0].Bindings = append(w.Environments[0].Bindings, w.Environments[0].Bindings[0])
		}, "unresolved"},
		{"dangling-instance", func(w *ir.World) {
			w.Environments[0].Bindings[0].Instance = "missing.instance"
		}, "unresolved"},
		{"duplicate-instance", func(w *ir.World) {
			w.Environments[0].Instances = append(w.Environments[0].Instances, w.Environments[0].Instances[0])
		}, "unresolved"},
		{"definition-mismatch", func(w *ir.World) {
			w.Environments[0].Instances[0].Definition = "sales_core"
		}, "incompatible"},
		{"provider-unspecified", func(w *ir.World) {
			w.Environments[0].Instances[0].Provider = ""
		}, "unresolved"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			world := syntheticEnvironmentBindingWorld()
			tc.mutate(&world)
			req := ObserveDeclaredContractQuery(world, "example", "example.customer_reference")
			got := ObserveEnvironmentBinding(world, "development", "example", &req)
			if got.Status != tc.status {
				t.Fatalf("incorrect scoped binding result %+v (want %s)", got, tc.status)
			}
			if got.Status == "declared-environment-binding" && got.DatabaseProvider != "oracle" {
				t.Fatalf("binding selected an unrelated provider: %+v", got)
			}
		})
	}
	world := syntheticEnvironmentBindingWorld()
	req := ObserveDeclaredContractQuery(world, "example", "example.customer_reference")
	for _, tc := range []struct {
		env string
		req *DeclaredContractQuery
	}{
		{"other", &req},
		{"development", nil},
	} {
		got := ObserveEnvironmentBinding(world, tc.env, "example", tc.req)
		if got.Status != "unresolved" {
			t.Fatalf("non-matching environment/contract was accepted: %+v", got)
		}
	}
}

func TestEnvironmentCLIRequiresExplicitOperationPurposeAndPreservesLegacy(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "workspace")
	base := []string{
		"-workspace", root,
		"-application", "order_entry",
		"-subject-kind", "application-pattern",
		"-subject-id", "order_entry.customer_reference",
	}
	for _, args := range [][]string{
		append(append([]string{}, base...), "-environment", "development"),
		append(append([]string{}, base...), "-purpose", "simulate", "-environment", "development"),
		append(append([]string{}, base...), "-purpose", "execute-preflight", "-environment", "bad env"),
	} {
		var output bytes.Buffer
		if err := runCLI(args, &output); err == nil || output.Len() != 0 {
			t.Fatalf("invalid environment scope accepted: args=%v output=%s err=%v", args, output.String(), err)
		}
	}
	var raw bytes.Buffer
	if err := runCLI(base, &raw); err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(raw.Bytes(), []byte("environmentBinding")) {
		t.Fatalf("legacy flat JSON changed: %s", raw.String())
	}

	var result bytes.Buffer
	args := append(append([]string{}, base...), "-purpose", "execute-preflight", "-environment", "development")
	if err := runCLI(args, &result); err != nil {
		t.Fatal(err)
	}
	var projected struct {
		Assessment WorkspaceAssessment `json:"assessment"`
		Readiness  PurposeProjection `json:"readiness"`
	}
	if err := json.Unmarshal(result.Bytes(), &projected); err != nil {
		t.Fatal(err)
	}
	if projected.Assessment.EnvironmentBinding == nil ||
		projected.Assessment.EnvironmentBinding.Status != "declared-environment-binding" ||
		projected.Assessment.EnvironmentBinding.Instance != "master_data.dev" ||
		projected.Readiness.ExecutionAllowed || projected.Assessment.ExecutionAllowed {
		t.Fatalf("environment binding missing or runtime permission granted: %+v", projected)
	}
}
