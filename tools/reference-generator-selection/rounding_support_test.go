package generatorselection

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"tei/internal/workspace"
	"testing"
)

func TestFixedGeneratorRejectsUnsupportedRoundingBeforePublication(t *testing.T) {
	root, path, _, _ := paths(t)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	changed := bytes.Replace(raw, []byte(`"half-up"`), []byte(`"half-even"`), 1)
	var contract Contract
	if err := json.Unmarshal(changed, &contract); err != nil {
		t.Fatal(err)
	}
	if err := validateContract(contract); err != nil {
		t.Fatalf("half-even became invalid as a contract: %v", err)
	}
	contractPath := filepath.Join(t.TempDir(), "contract.json")
	if err := os.WriteFile(contractPath, changed, 0o600); err != nil {
		t.Fatal(err)
	}
	loaded, err := workspace.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	for strategy := range generatorTemplates {
		if strategy == "text-template" {
			// text-template is a projection generator: its contract has no
			// rounding block and requires a projection instead. Its acceptance
			// and rejection are covered by text_template_test.go and
			// projection_test.go.
			continue
		}
		t.Run(strategy, func(t *testing.T) {
			parent := t.TempDir()
			out := filepath.Join(parent, "not-created")
			_, err := GenerateVersion(contractPath, strategy, "1", out)
			if err == nil || !strings.Contains(err.Error(), `rounding mode "half-even"`) {
				t.Fatalf("unsupported mode silently generated: %v", err)
			}
			entries, err := os.ReadDir(parent)
			if err != nil || len(entries) != 0 {
				t.Fatal("unsupported generation published files")
			}
			source, manifest, err := RenderForWorld(loaded.World, changed, strategy, "1")
			if err == nil || len(source) != 0 || manifest.ID != "" {
				t.Fatalf("World rendering accepted unsupported mode: %v", err)
			}
			if source, _, err := RenderForWorld(loaded.World, raw, strategy, "1"); err != nil || len(source) == 0 {
				t.Fatalf("half-up regressed: %v", err)
			}
		})
	}
}
