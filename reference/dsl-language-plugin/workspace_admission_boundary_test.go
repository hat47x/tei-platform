package languagereference_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/workspace"
	languagereference "tei/reference/dsl-language-plugin"
	teimdreader "tei/reference/tei-md-graph-reader"
)

// A reference-local parser selection does not bind that parser to the product
// Workspace. This test intentionally demonstrates the gap, not a selected TEI
// Language or accepted Graph Meaning. Replace the rejection expectation only
// after the product's explicit Language/implementation selection is exercised.
func TestSelectedReferenceGraphDoesNotAdmitWorkspaceMarkdown(t *testing.T) {
	original, err := os.ReadFile(filepath.Join("..", "..", "testdata", "tei-md-graph", "baseline.tei.md"))
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "semantic"), 0700); err != nil {
		t.Fatal(err)
	}
	const selectedPath = "semantic/graph.tei.md"
	if err := os.WriteFile(filepath.Join(root, "tei.workspace.yaml"), []byte("workspace: graph_language_reference_boundary\nformatVersion: 1\nsources:\n  - semantic\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(selectedPath)), original, 0600); err != nil {
		t.Fatal(err)
	}
	assertUnbound := func() {
		t.Helper()
		_, err := workspace.Load(root)
		if err == nil || !strings.Contains(err.Error(), "TEI_MD_PARSER_UNBOUND") || !strings.Contains(err.Error(), selectedPath) {
			t.Fatalf("reference selection must not admit the managed Markdown Source: %v", err)
		}
	}
	assertUnbound()

	// This pin is synthetic and only qualifies the E1 reference selector; the
	// real product Manifest/Accepted Contract/Lock supplied no such selection.
	pin := languagereference.Selection{
		LanguageID: "reference.graph.markdown", ContractVersion: "fixture-v1",
		ContractFingerprint: "fixed-graph-rule", ImplementationID: "reference.graph.reader",
		ImplementationFingerprint: "fixture-parser-v1",
	}
	calls := 0
	parser := languagereference.Provider[teimdreader.RuleCandidate]{
		Selection: pin,
		Parse: func(s string) (teimdreader.RuleCandidate, error) {
			calls++
			return teimdreader.ParseRuleCandidate([]byte(s))
		},
	}
	candidate, err := languagereference.ParseSelected(pin, []languagereference.Provider[teimdreader.RuleCandidate]{parser}, string(original))
	if err != nil {
		t.Fatalf("reference-only selected graph was not parsed: %v", err)
	}
	if calls != 1 || candidate.Rule.ID != "node.edge_ids" || len(candidate.Graph.Nodes) != 2 || len(candidate.Graph.Edges) != 2 ||
		candidate.Graph.Edges[0].ID == candidate.Graph.Edges[1].ID {
		t.Fatalf("reference graph identity or parser invocation lost: calls=%d candidate=%+v", calls, candidate)
	}
	span := candidate.Rule.ConditionSource
	if span.Start < 0 || span.End > len(original) || span.Start >= span.End || string(original[span.Start:span.End]) != "起点 = 対象ノード" {
		t.Fatalf("reference candidate lost its original byte span: %+v", span)
	}

	for _, tc := range []struct {
		name      string
		selection languagereference.Selection
		providers []languagereference.Provider[teimdreader.RuleCandidate]
		want      error
	}{
		{"no accepted selection", languagereference.Selection{}, []languagereference.Provider[teimdreader.RuleCandidate]{parser}, languagereference.ErrInvalidSelection},
		{"different parser fingerprint", languagereference.Selection{
			LanguageID: pin.LanguageID, ContractVersion: pin.ContractVersion,
			ContractFingerprint: pin.ContractFingerprint, ImplementationID: pin.ImplementationID,
			ImplementationFingerprint: "other",
		}, []languagereference.Provider[teimdreader.RuleCandidate]{parser}, languagereference.ErrProviderMissing},
		{"ambiguous provider", pin, []languagereference.Provider[teimdreader.RuleCandidate]{parser, parser}, languagereference.ErrProviderAmbiguous},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := languagereference.ParseSelected(tc.selection, tc.providers, string(original))
			if !errors.Is(err, tc.want) {
				t.Fatalf("reference selector returned %v, want %v", err, tc.want)
			}
			if calls != 1 {
				t.Fatalf("an unresolved/ambiguous selection called the parser %d times", calls)
			}
		})
	}
	assertUnbound()
}
