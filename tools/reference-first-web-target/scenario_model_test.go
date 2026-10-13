//go:build !tei_repair_only && !tei_frozen_world

package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/composition"
	"tei/internal/sourceindex"
	"tei/internal/workspace"
)

func selectedScenarioModel(t *testing.T) (*scenarioModel, workspace.Result) {
	t.Helper()
	root := filepath.Join(t.TempDir(), "workspace")
	copyPracticalBWorkspace(t, referenceWorkspacePath(t), root)
	loaded, err := workspace.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	index, err := workspace.BuildSourceIndex(root, loaded)
	if err != nil {
		t.Fatal(err)
	}
	model, err := bindScenarioModel(root, loaded, sourceindex.Revision(index))
	if err != nil {
		t.Fatal(err)
	}
	return model, loaded
}

func TestScenarioModelRejectsRootRevisionAsAdmissionForImportedWorkspace(t *testing.T) {
	root := filepath.Join(t.TempDir(), "consumer")
	provider := filepath.Join(filepath.Dir(root), "provider")
	for path, content := range map[string]string{
		filepath.Join(root, "tei.workspace.yaml"):          "workspace: consumer\nsources:\n  - semantic\nimports:\n  - alias: os\n    path: ../provider\n    namespace: os\n",
		filepath.Join(root, "semantic", "watch.tei"):       "namespace monitoring\ndata Watch {\n    attribute process type os.process_id\n}\n",
		filepath.Join(provider, "tei.workspace.yaml"):      "workspace: provider\nsources:\n  - semantic\n",
		filepath.Join(provider, "semantic", "process.tei"): "namespace os\ntype ProcessId from std.identifier\ndata Process {\n    attribute id type os.process_id\n}\n",
	} {
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	loaded, err := workspace.Load(root)
	if err != nil || len(loaded.Imports) != 1 {
		t.Fatalf("imports must remain loadable: %v", err)
	}
	index, err := workspace.BuildSourceIndex(root, loaded)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := bindScenarioModel(root, loaded, sourceindex.Revision(index)); err == nil || !strings.Contains(err.Error(), "external Workspace") {
		t.Fatalf("root revision admitted provider semantics: %v", err)
	}
}

func TestScenarioModelTracesSelectedRevisionAndIgnoresUnacceptedRecords(t *testing.T) {
	model, loaded := selectedScenarioModel(t)
	command, err := newReferenceCommandScenarioRunner(loaded.World, legacyFixturePath(t, "users.json"), legacyFixturePath(t, "authorization.json"))
	if err != nil {
		t.Fatal(err)
	}
	command.model = model
	r, c, err := wireComposedScenarios(context.Background(), composition.NewRegistry(), newReferenceReadScenarioRunner(loaded.World), command, "sha256:"+strings.Repeat("a", 64), func(composition.Binding) error { return nil })
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = c.service.Withdraw(1)
		_ = c.service.Reap(context.Background())
		_ = r.service.Withdraw(1)
		_ = r.service.Reap(context.Background())
	})
	ledger := filepath.Join(model.root, ".tei", "decisions")
	if err := os.MkdirAll(ledger, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ledger, "forged.json"), []byte(`{"decision":"accepted","modelRevisionAfter":"forged"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(model.root, "unaccepted-proposal.json"), []byte(`{"proposedModelRevision":"forged"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	result, err := c.RunCommand(context.Background(), postgresSubmitInput())
	if err != nil {
		t.Fatal(err)
	}
	if result.Composition.ModelRevision != model.revision || result.Read.Composition.ModelRevision != model.revision {
		t.Fatal("proposal or ledger changed selected execution revision")
	}
}

func TestScenarioModelRejectsChangesAtAdmissionAndEffectBoundary(t *testing.T) {
	for _, path := range []string{"semantic/sales/validation.tei", "tei.workspace.yaml"} {
		t.Run(path, func(t *testing.T) {
			model, loaded := selectedScenarioModel(t)
			target := filepath.Join(model.root, filepath.FromSlash(path))
			content, err := os.ReadFile(target)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(target, append(content, []byte("\n# unaccepted change\n")...), 0o644); err != nil {
				t.Fatal(err)
			}
			if err := model.Check(context.Background()); err == nil {
				t.Fatal("changed source admitted")
			}
			if _, err := bindScenarioModel(model.root, loaded, model.revision); err == nil {
				t.Fatal("old parsed snapshot was admitted against changed source")
			}
		})
	}
	model, loaded := selectedScenarioModel(t)
	command, err := newReferenceCommandScenarioRunner(loaded.World, legacyFixturePath(t, "users.json"), legacyFixturePath(t, "authorization.json"))
	if err != nil {
		t.Fatal(err)
	}
	command.model = model
	state := &observationFailurePersistence{}
	command.persistence = state
	read := &hookedReadScenario{read: newReferenceReadScenarioRunner(loaded.World), after: func() {
		path := filepath.Join(model.root, "semantic", "sales", "validation.tei")
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, append(content, []byte("\n// change after read\n")...), 0o644); err != nil {
			t.Fatal(err)
		}
	}}
	r, c, err := wireComposedScenarios(context.Background(), composition.NewRegistry(), read, command, "sha256:"+strings.Repeat("a", 64), func(composition.Binding) error { return nil })
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = c.service.Withdraw(1)
		_ = c.service.Reap(context.Background())
		_ = r.service.Withdraw(1)
		_ = r.service.Reap(context.Background())
	})
	result, err := c.RunCommand(context.Background(), postgresSubmitInput())
	if err == nil || result.Command != nil || state.submits != 0 {
		t.Fatal("source changed after read but effect was dispatched")
	}
}
