package operations

import (
	"encoding/json"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/semantic"
	"tei/internal/semantic/validationeval"
	"tei/internal/sourceindex"
	"tei/internal/workspace"
)

// Editing a rule's referenced attribute must change the typed meaning and the
// actual Evaluator read target even when the rule ID and language stay fixed.
// A Source fingerprint identifies bytes, not whether the rule was accepted.
func TestSelectedMarkdownRuleMutationChangesEvaluatorReadAndInvalidatesSource(t *testing.T) {
	fixture := filepath.Join("..", "workspace", "testdata", "tei-md-rule")
	root := t.TempDir()
	const rulePath = "semantic/sales/order-line-rules.tei.md"
	for _, path := range []string{
		"tei.workspace.yaml",
		"semantic/sales/order.tei",
		rulePath,
	} {
		original, err := os.ReadFile(filepath.Join(fixture, filepath.FromSlash(path)))
		if err != nil {
			t.Fatal(err)
		}
		destination := filepath.Join(root, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(destination), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(destination, original, 0600); err != nil {
			t.Fatal(err)
		}
	}

	const id = "sales.order_line.quantity_positive"
	subject := sourceindex.Subject{Kind: "semantic-validation", ID: id}
	loadRule := func() (semantic.Validation, sourceindex.Index) {
		t.Helper()
		loaded, err := workspace.Load(root)
		if err != nil {
			t.Fatalf("real selected Workspace Load: %v", err)
		}
		if len(loaded.AuthoringSources) != 1 ||
			loaded.AuthoringSources[0].Language.LanguageID != "tei.rule-authoring" ||
			loaded.AuthoringSources[0].Definitions != 2 {
			t.Fatalf("product authoring selection changed: %+v", loaded.AuthoringSources)
		}
		idx, err := workspace.BuildSourceIndexFromLoad(root, loaded)
		if err != nil {
			t.Fatalf("Source Index from actual Load: %v", err)
		}
		inspection, err := InspectionFromLoad(loaded, idx, subject)
		if err != nil {
			t.Fatalf("actual typed rule inspection: %v", err)
		}
		if inspection.Subject != subject || inspection.SourceStatus != "indexed" ||
			len(inspection.Sources.References) != 1 {
			t.Fatalf("selected rule lacks its original Source: %+v", inspection)
		}
		var rule semantic.Validation
		if err := json.Unmarshal(inspection.Canonical, &rule); err != nil {
			t.Fatal(err)
		}
		if string(rule.ID) != id || rule.Kind != semantic.ValidationPositive {
			t.Fatalf("rule changed identity or kind: %+v", rule)
		}
		foundSource := false
		for _, artifact := range idx.Artifacts {
			if artifact.Path == rulePath {
				foundSource = true
			}
		}
		if !foundSource {
			t.Fatal("actual Rule Source missing from index")
		}
		return rule, idx
	}
	evaluate := func(rule semantic.Validation, attribute string, value *big.Rat, want validationeval.Result) {
		t.Helper()
		probe := &evaluatorReadProbe{
			attribute: core.CanonicalID(attribute),
			number:    value,
			complete:  true,
		}
		result, err := validationeval.Evaluate(rule, probe)
		if err != nil || result != want {
			t.Fatalf("rule %s result=%q err=%v, want %q", rule.ID, result, err, want)
		}
		wantRead := "Number:" + attribute
		if len(probe.reads) != 1 || probe.reads[0] != wantRead {
			t.Fatalf("real Evaluator read %v, want exactly %q", probe.reads, wantRead)
		}
	}

	before, indexBefore := loadRule()
	const quantity = "sales.order_line.quantity"
	const lineNumber = "sales.order_line.line_number"
	if string(before.Attribute) != quantity {
		t.Fatalf("original rule has unexpected read target: %+v", before)
	}
	evaluate(before, quantity, big.NewRat(2, 1), validationeval.ResultSatisfied)

	ruleFile := filepath.Join(root, filepath.FromSlash(rulePath))
	bytesBefore, err := os.ReadFile(ruleFile)
	if err != nil {
		t.Fatal(err)
	}
	const originalPredicate = "sales.order_line.quantity is positive"
	const editedPredicate = "sales.order_line.line_number is positive"
	if strings.Count(string(bytesBefore), originalPredicate) != 1 {
		t.Fatal("fixture no longer has one matching rule predicate")
	}
	bytesAfter := []byte(strings.Replace(string(bytesBefore), originalPredicate, editedPredicate, 1))
	if err := os.WriteFile(ruleFile, bytesAfter, 0600); err != nil {
		t.Fatal(err)
	}
	// Old provenance cannot describe the new bytes, even when the Canonical
	// rule identity is unchanged. This is not an Owner acceptance check.
	err = indexBefore.Validate(func(path string) ([]byte, error) {
		return os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
	})
	if err == nil || !strings.Contains(err.Error(), "fingerprint is stale") {
		t.Fatalf("old Source Index was reused after semantic edit: %v", err)
	}

	after, indexAfter := loadRule()
	if before.ID != after.ID || before.Kind != after.Kind ||
		string(after.Attribute) != lineNumber {
		t.Fatalf("rule mutation did not change its actual typed target: before=%+v after=%+v", before, after)
	}
	evaluate(after, lineNumber, big.NewRat(0, 1), validationeval.ResultNotSatisfied)
	if err := indexAfter.Validate(func(path string) ([]byte, error) {
		return os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
	}); err != nil {
		t.Fatalf("new Load Source Index did not validate new bytes: %v", err)
	}
	fingerprint := func(idx sourceindex.Index) string {
		for _, artifact := range idx.Artifacts {
			if artifact.Path == rulePath {
				return artifact.Fingerprint
			}
		}
		return ""
	}
	if fingerprint(indexBefore) == "" || fingerprint(indexAfter) == "" ||
		fingerprint(indexBefore) == fingerprint(indexAfter) {
		t.Fatal("source change did not change actual indexed byte fingerprint")
	}
}
