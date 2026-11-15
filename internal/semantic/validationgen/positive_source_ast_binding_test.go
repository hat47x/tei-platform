package validationgen_test

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/semantic"
	"tei/internal/semantic/validationgen"
	"tei/internal/sourceindex"
	"tei/internal/workspace"
)

// This is an I2M-1 test-only witness, not a product Inspector API or a Source
// writer. Authored Source spans and generated Go AST spans refer to distinct
// byte sequences even when different authoring languages generate the same Go.
type ruleASTBindingWitness struct {
	rule              semantic.Validation
	sourcePath        string
	sourceFingerprint string
	modelRevision     string
	sourceStart       int
	sourceEnd         int
	generated         []byte
	observed          observedPositiveGo
}

func testRuleASTBinding(t *testing.T, root, sourcePath string, markdown bool) ruleASTBindingWitness {
	t.Helper()
	const (
		ruleID      = "sales.order_line.quantity_positive"
		attributeID = "sales.order_line.quantity"
	)
	loaded, err := workspace.Load(root)
	if err != nil {
		t.Fatalf("load selected authoring source: %v", err)
	}
	if markdown {
		if len(loaded.AuthoringSources) != 1 ||
			loaded.AuthoringSources[0].Language.LanguageID != "tei.rule-authoring" ||
			loaded.AuthoringSources[0].Language.ContractVersion != "v0" {
			t.Fatalf("Markdown Reader not explicitly selected: %+v", loaded.AuthoringSources)
		}
	} else if len(loaded.AuthoringSources) != 0 {
		t.Fatalf("legacy validation was not read natively: %+v", loaded.AuthoringSources)
	}

	index, err := workspace.BuildSourceIndexFromLoad(root, loaded)
	if err != nil {
		t.Fatalf("build Source Index from this Load: %v", err)
	}
	var rule semantic.Validation
	var attributeType core.CanonicalID
	for _, namespace := range loaded.World.Semantics {
		for _, item := range namespace.Validations {
			if string(item.ID) == ruleID {
				if rule.ID != "" {
					t.Fatal("more than one selected Rule")
				}
				rule = item
			}
		}
		for _, data := range namespace.Data {
			for _, attr := range data.Attributes {
				if string(attr.ID) == attributeID {
					attributeType = attr.TypeRef
				}
			}
		}
	}
	if string(rule.ID) != ruleID || rule.Kind != semantic.ValidationPositive ||
		string(rule.Attribute) != attributeID || rule.Relation != "" ||
		attributeType != "std.decimal" {
		t.Fatalf("incorrect selected Rule or resolved type: %+v, %s", rule, attributeType)
	}
	refs := index.ReferencesFor(sourceindex.Subject{Kind: "semantic-validation", ID: ruleID})
	if len(refs) != 1 || refs[0].Role != sourceindex.RoleDefinition ||
		refs[0].ArtifactPath != sourcePath {
		t.Fatalf("Rule was not bound to its one authored definition: %+v", refs)
	}
	raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(sourcePath)))
	if err != nil {
		t.Fatal(err)
	}
	if err := refs[0].Range.Validate(raw); err != nil {
		t.Fatalf("authored Source span is invalid: %v", err)
	}
	start, end := refs[0].Range.Start.Offset, refs[0].Range.End.Offset
	if start < 0 || end <= start || end > len(raw) ||
		string(raw[start:end]) != ruleID {
		t.Fatalf("the authored span does not identify the Rule: %+v", refs[0].Range)
	}
	fingerprint := sourceindex.Fingerprint(raw)
	artifactMatched := false
	for _, artifact := range index.Artifacts {
		if artifact.Path == sourcePath {
			artifactMatched = artifact.Fingerprint == fingerprint
		}
	}
	if !artifactMatched {
		t.Fatal("Source Index fingerprint does not describe the selected original bytes")
	}

	generated, err := validationgen.GeneratePositive(rule, attributeType, "main")
	if err != nil {
		t.Fatalf("lower the selected, resolved Rule: %v", err)
	}
	observed, err := observeGeneratedPositiveGo(generated)
	if err != nil {
		t.Fatalf("inspect AST of actual generated Go: %v", err)
	}
	if observed.ReadAttribute != string(rule.Attribute) || observed.ReadCount != 1 ||
		string(generated[observed.ConditionStart:observed.ConditionEnd]) != "value.Sign() > 0" ||
		!bytes.Contains(generated[observed.ReadStart:observed.ReadEnd],
			[]byte("source.Number(\""+attributeID+"\")")) {
		t.Fatalf("selected Rule and generated Go AST disagree: %+v", observed)
	}
	return ruleASTBindingWitness{
		rule: rule, sourcePath: sourcePath, sourceFingerprint: fingerprint,
		modelRevision: sourceindex.ModelRevision(index), sourceStart: start, sourceEnd: end,
		generated: generated, observed: observed,
	}
}

func TestSelectedRuleAuthoringSourceAndGeneratedGoASTRemainSeparatelyTraceable(t *testing.T) {
	const (
		markdownPath = "semantic/sales/order-line-rules.tei.md"
		legacyPath   = "semantic/sales/validation.tei"
	)
	fixture := filepath.Join("..", "..", "workspace", "testdata", "tei-md-rule")
	legacyFixture := filepath.Join("..", "..", "..", "reference", "workspace", legacyPath)
	temp := t.TempDir()
	markdownRoot := filepath.Join(temp, "markdown")
	legacyRoot := filepath.Join(temp, "legacy")

	copySource := func(dst, src string) {
		t.Helper()
		data, err := os.ReadFile(src)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(dst, data, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range []string{"tei.workspace.yaml", "semantic/sales/order.tei", markdownPath} {
		copySource(filepath.Join(markdownRoot, filepath.FromSlash(name)),
			filepath.Join(fixture, filepath.FromSlash(name)))
	}
	if err := os.MkdirAll(legacyRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(legacyRoot, "tei.workspace.yaml"),
		[]byte("workspace: i2m_legacy_source\nformatVersion: 1\nsources:\n  - semantic\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	copySource(filepath.Join(legacyRoot, "semantic", "sales", "order.tei"),
		filepath.Join(fixture, "semantic", "sales", "order.tei"))
	copySource(filepath.Join(legacyRoot, filepath.FromSlash(legacyPath)), legacyFixture)

	markdown := testRuleASTBinding(t, markdownRoot, markdownPath, true)
	legacy := testRuleASTBinding(t, legacyRoot, legacyPath, false)
	if !reflect.DeepEqual(markdown.rule, legacy.rule) ||
		!bytes.Equal(markdown.generated, legacy.generated) ||
		!reflect.DeepEqual(markdown.observed, legacy.observed) {
		t.Fatal("different authored languages did not produce the same typed Rule and generated AST")
	}
	if markdown.sourcePath == legacy.sourcePath ||
		markdown.sourceFingerprint == legacy.sourceFingerprint ||
		markdown.modelRevision == legacy.modelRevision {
		t.Fatal("different authored Sources were wrongly coalesced by identical generated Go")
	}
	// Equal AST offsets identify equal *generated Go*, not interchangeable
	// authored Source positions. Keep the latter rooted in their own Index.
	if markdown.observed.ReadStart >= markdown.observed.ReadEnd ||
		markdown.sourceStart >= markdown.sourceEnd {
		t.Fatal("either source span is unavailable")
	}

	markdownFile := filepath.Join(markdownRoot, filepath.FromSlash(markdownPath))
	raw, err := os.ReadFile(markdownFile)
	if err != nil {
		t.Fatal(err)
	}
	loadedBefore, err := workspace.Load(markdownRoot)
	if err != nil {
		t.Fatal(err)
	}
	oldIndex, err := workspace.BuildSourceIndexFromLoad(markdownRoot, loadedBefore)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(markdownFile, append(raw, []byte("\n注記のみ更新。\n")...), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := oldIndex.Validate(func(path string) ([]byte, error) {
		return os.ReadFile(filepath.Join(markdownRoot, filepath.FromSlash(path)))
	}); err == nil || !strings.Contains(err.Error(), "fingerprint is stale") {
		t.Fatalf("old authored Source evidence remained current after prose edit: %v", err)
	}
	updated := testRuleASTBinding(t, markdownRoot, markdownPath, true)
	if !reflect.DeepEqual(markdown.rule, updated.rule) ||
		!bytes.Equal(markdown.generated, updated.generated) ||
		!reflect.DeepEqual(markdown.observed, updated.observed) ||
		markdown.sourceFingerprint == updated.sourceFingerprint ||
		markdown.modelRevision == updated.modelRevision {
		t.Fatal("prose edit did not separate authored Source revision from generated Go behavior")
	}

	// A generated-code mutant can still parse and pass a read-shape observer
	// while reading a different Attribute. The *binding* must reject it.
	const original = "source.Number(\"sales.order_line.quantity\")"
	const wrong = "source.Number(\"sales.order_line.discount\")"
	mutated := bytes.Replace(markdown.generated, []byte(original), []byte(wrong), 1)
	if bytes.Equal(markdown.generated, mutated) {
		t.Fatal("read-target mutant was not applied")
	}
	mutantAST, err := observeGeneratedPositiveGo(mutated)
	if err != nil {
		t.Fatalf("mutant should remain syntactically observable: %v", err)
	}
	if mutantAST.ReadAttribute == string(markdown.rule.Attribute) {
		t.Fatal("mutant Go read was incorrectly bound to the authored Rule")
	}
	// Structural observation also rejects a semantically wrong threshold.
	threshold := bytes.Replace(markdown.generated, []byte("value.Sign() > 0"),
		[]byte("value.Sign() >= 0"), 1)
	if bytes.Equal(threshold, markdown.generated) {
		t.Fatal("threshold mutant was not applied")
	}
	if _, err := observeGeneratedPositiveGo(threshold); err == nil {
		t.Fatal("mutated threshold remained a qualified strict-positive decision")
	}
}
