package hostqualification

import (
	"os"
	"path/filepath"
	"sync"
	"testing"

	"tei/internal/operations"
)

// This package tests the pre-Host read-only baseline. These tests are NOT
// evidence that Wails Web sessions or Desktop windows are isolated.
func fixtureWorkspace(t *testing.T, namespace, source string) string {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "tei.workspace.yaml"),
		[]byte("workspace: studio_"+namespace+"\nsources:\n  - semantic\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(root, "semantic", namespace, "model.tei")
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestIndependentReadOnlyCatalogsDoNotExchangeSubjects(t *testing.T) {
	alpha := fixtureWorkspace(t, "studio_alpha", "namespace studio_alpha\n")
	beta := fixtureWorkspace(t, "studio_beta", "namespace studio_beta\n")

	type observation struct {
		namespace string
		catalog   operations.ModelCatalog
		err       error
	}
	got := make(chan observation, 16)
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		for _, input := range []struct{ namespace, root string }{
			{"studio_alpha", alpha}, {"studio_beta", beta},
		} {
			wg.Add(1)
			go func(namespace, root string) {
				defer wg.Done()
				catalog, err := operations.CatalogWorkspace(root, "")
				got <- observation{namespace: namespace, catalog: catalog, err: err}
			}(input.namespace, input.root)
		}
	}
	wg.Wait()
	close(got)

	count := 0
	for observation := range got {
		count++
		if observation.err != nil {
			t.Fatalf("%s catalog: %v", observation.namespace, observation.err)
		}
		if observation.catalog.Effect != "read-only" {
			t.Fatalf("%s catalog unexpectedly has effect %q", observation.namespace, observation.catalog.Effect)
		}
		found := false
		for _, entry := range observation.catalog.Entries {
			if entry.Subject.Kind != "semantic" {
				continue
			}
			if entry.Subject.ID == observation.namespace {
				found = true
			}
			if entry.Subject.ID == "studio_alpha" && observation.namespace != "studio_alpha" ||
				entry.Subject.ID == "studio_beta" && observation.namespace != "studio_beta" {
				t.Fatalf("catalog for %s contained the other workspace's subject: %+v",
					observation.namespace, entry.Subject)
			}
		}
		if !found {
			t.Fatalf("catalog for %s is missing its own namespace", observation.namespace)
		}
	}
	if count != 16 {
		t.Fatalf("expected 16 independent observations, got %d", count)
	}
}

func TestSyntaxFindingRemainsScopedToItsWorkspace(t *testing.T) {
	healthy := fixtureWorkspace(t, "studio_ok", "namespace studio_ok\n")
	broken := fixtureWorkspace(t, "studio_broken",
		"namespace studio_broken\n\ndata Broken {\n    attribute invalid type }\n}\n")

	type observation struct {
		root   string
		result operations.WorkspaceValidation
		err    error
	}
	got := make(chan observation, 2)
	var wg sync.WaitGroup
	for _, root := range []string{healthy, broken} {
		wg.Add(1)
		go func(root string) {
			defer wg.Done()
			result, err := operations.ValidateWorkspace(root)
			got <- observation{root: root, result: result, err: err}
		}(root)
	}
	wg.Wait()
	close(got)

	for observation := range got {
		if observation.err != nil {
			t.Fatalf("validation failed before returning diagnostics: %v", observation.err)
		}
		if observation.root == healthy {
			if observation.result.Direct != nil || observation.result.HasErrors() {
				t.Fatalf("healthy workspace received errors: %+v", observation.result)
			}
		} else {
			if observation.result.Direct == nil ||
				observation.result.Direct.Code != "PARSE_ERROR" ||
				!observation.result.HasErrors() {
				t.Fatalf("broken workspace lost its direct syntax finding: %+v", observation.result)
			}
		}
	}
}
