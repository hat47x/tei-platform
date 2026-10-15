//go:build studiohost_parity

package hostqualification

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"tei/internal/operations"
	"tei/internal/sourceindex"
	"tei/internal/studiohost"
)

// Only run in a non-merged qualification snapshot that includes the exact
// internal/studiohost implementation from PoC A PR #1832. This calls the REAL
// Go Host bridge; it does not exercise Wails IPC, JS or browser sessions.
func TestRealStudioHostCLIAndGoOperationParity(t *testing.T) {
	binary := testCLIBinary(t)
	reference := filepath.Join("..", "..", "reference", "workspace")
	normal := fixtureWorkspace(t, "hostnormal", "namespace hostnormal\n")
	syntax := fixtureWorkspace(t, "hostsyntax", "namespace hostsyntax\n\ndata Broken {\n    attribute invalid type }\n}\n")
	semantic := t.TempDir()
	if err := os.WriteFile(filepath.Join(semantic, "tei.workspace.yaml"),
		[]byte("workspace: host_semantic\nsources:\n  - applications\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	appFile := filepath.Join(semantic, "applications", "order-entry", "application.tei")
	if err := os.MkdirAll(filepath.Dir(appFile), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(appFile, []byte("application order_entry {\n    database sales {\n        target sales_core\n        access invalid\n    }\n}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	missingManifest := t.TempDir() // allowed root; workspace.Load rejects missing manifest
	roots := []string{normal, syntax, semantic, missingManifest, reference}
	host, err := studiohost.NewReadOnly(roots)
	if err != nil {
		t.Fatal(err)
	}
	if listed := host.ListWorkspaces(); !listed.OK || len(listed.Value) != len(roots) {
		t.Fatalf("Host allowlist does not match test inputs: %+v", listed)
	}

	for _, tc := range []struct {
		name, workspaceID, root string
		exit                    int
	}{
		{name: "valid", workspaceID: "workspace-1", root: normal, exit: 0},
		{name: "syntax-diagnostic", workspaceID: "workspace-2", root: syntax, exit: 1},
		{name: "semantic-diagnostic", workspaceID: "workspace-3", root: semantic, exit: 1},
		{name: "load-failure", workspaceID: "workspace-4", root: missingManifest, exit: 2},
	} {
		t.Run("validate-"+tc.name, func(t *testing.T) {
			cli := observeCLI(t, binary, tc.exit, "validate", "-workspace", tc.root, "-json")
			goValue, goErr := operations.ValidateWorkspace(tc.root)
			hostResponse := host.ValidateWorkspace(tc.workspaceID)
			if tc.exit == 2 {
				var stage *operations.ValidationFailure
				if !errors.As(goErr, &stage) || stage.Stage != operations.StageLoad ||
					cli.Error == nil || cli.Error.Code != "WORKSPACE_LOAD_FAILED" ||
					hostResponse.OK || hostResponse.Value != nil || hostResponse.Failure == nil ||
					hostResponse.Failure.Code != cli.Error.Code || hostResponse.Failure.Stage != string(stage.Stage) {
					t.Fatalf("load classification differs: CLI=%+v Go=%v Host=%+v", cli, goErr, hostResponse)
				}
				return
			}
			if goErr != nil || hostResponse.Failure != nil || !hostResponse.OK || hostResponse.Value == nil ||
				cli.Error != nil || cli.OK != !goValue.HasErrors() {
				t.Fatalf("validation status differs: CLI=%+v Go=%+v error=%v Host=%+v", cli, goValue, goErr, hostResponse)
			}
			cliDiagnostics := cliFindings(cli)
			goDiagnostics := typedFindings(t, goValue)
			hostDiagnostics := typedFindings(t, *hostResponse.Value)
			// A diagnostic fixture must not silently turn into an empty baseline.
			if tc.exit == 1 && (len(cliDiagnostics) == 0 || len(goDiagnostics) == 0 || len(hostDiagnostics) == 0) {
				t.Fatalf("diagnostic fixture %q produced no findings: CLI=%v Go=%v Host=%v",
					tc.name, cliDiagnostics, goDiagnostics, hostDiagnostics)
			}
			if !reflect.DeepEqual(normalizedJSON(t, *hostResponse.Value), normalizedJSON(t, goValue)) ||
				!reflect.DeepEqual(cliDiagnostics, goDiagnostics) ||
				!reflect.DeepEqual(cliDiagnostics, hostDiagnostics) {
				t.Fatalf("validation semantics differ: CLI=%v Go=%v Host=%v",
					cliDiagnostics, goDiagnostics, hostDiagnostics)
			}
		})
	}

	t.Run("catalog-reference", func(t *testing.T) {
		cli := observeCLI(t, binary, 0, "model", "catalog", "-workspace", reference, "-namespace", "sales")
		goValue, goErr := operations.CatalogWorkspace(reference, "sales")
		hostResponse := host.CatalogWorkspace("workspace-5", "sales")
		if goErr != nil || !cli.OK || !hostResponse.OK || hostResponse.Value == nil ||
			hostResponse.Failure != nil || cli.Error != nil ||
			!reflect.DeepEqual(normalizedJSON(t, goValue), normalizedJSON(t, *hostResponse.Value)) ||
			!reflect.DeepEqual(normalizedRawJSON(t, cli.Result), normalizedJSON(t, *hostResponse.Value)) {
			t.Fatalf("catalog parity failed: CLI=%+v Go=%+v err=%v Host=%+v", cli, goValue, goErr, hostResponse)
		}
	})
	t.Run("catalog-load-failure", func(t *testing.T) {
		cli := observeCLI(t, binary, 2, "model", "catalog", "-workspace", missingManifest)
		_, goErr := operations.CatalogWorkspace(missingManifest, "")
		hostResponse := host.CatalogWorkspace("workspace-4", "")
		var stage *operations.ValidationFailure
		if !errors.As(goErr, &stage) || stage.Stage != operations.StageLoad ||
			cli.Error == nil || cli.Error.Code != "WORKSPACE_LOAD_FAILED" ||
			hostResponse.OK || hostResponse.Value != nil || hostResponse.Failure == nil ||
			hostResponse.Failure.Code != cli.Error.Code || hostResponse.Failure.Stage != string(stage.Stage) {
			t.Fatalf("catalog load classification differs: CLI=%+v Go=%v Host=%+v", cli, goErr, hostResponse)
		}
	})

	for _, tc := range []struct {
		name, kind, id, failureCode string
		exit                        int
	}{
		{name: "success", kind: "semantic-data", id: "sales.order", exit: 0},
		{name: "not-found", kind: "semantic-data", id: "sales.missing", exit: 1, failureCode: operations.InspectionNotFound},
		{name: "unsupported", kind: "not-a-kind", id: "sales.order", exit: 2, failureCode: operations.InspectionUnsupportedKind},
		{name: "invalid", kind: "semantic-data", id: " sales.order", exit: 2, failureCode: operations.InspectionInvalidSubject},
	} {
		t.Run("inspect-"+tc.name, func(t *testing.T) {
			cli := observeCLI(t, binary, tc.exit,
				"inspect", "-workspace", reference, "-subject-kind", tc.kind, "-subject-id", tc.id)
			goValue, goErr := operations.InspectSubject(reference, sourceindex.Subject{Kind: tc.kind, ID: tc.id})
			hostResponse := host.InspectSubject("workspace-5", tc.kind, tc.id)
			if tc.failureCode != "" {
				var typed *operations.InspectionError
				if !errors.As(goErr, &typed) || typed.Code != tc.failureCode ||
					cli.Error == nil || cli.Error.Code != typed.Code ||
					hostResponse.OK || hostResponse.Value != nil || hostResponse.Failure == nil ||
					hostResponse.Failure.Code != typed.Code {
					t.Fatalf("subject rejection differs: CLI=%+v Go=%v Host=%+v", cli, goErr, hostResponse)
				}
				return
			}
			if goErr != nil || cli.Error != nil || !cli.OK || !hostResponse.OK ||
				hostResponse.Value == nil || hostResponse.Failure != nil ||
				!reflect.DeepEqual(normalizedRawJSON(t, cli.Result), normalizedJSON(t, *hostResponse.Value)) ||
				!reflect.DeepEqual(normalizedJSON(t, goValue), normalizedJSON(t, *hostResponse.Value)) {
				t.Fatalf("subject result differs: CLI=%+v Go=%+v err=%v Host=%+v", cli, goValue, goErr, hostResponse)
			}
		})
	}
}
