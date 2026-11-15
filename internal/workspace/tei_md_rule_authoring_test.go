package workspace

import (
	"bytes"
	"errors"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/semantic"
	"tei/internal/semantic/validationeval"
	"tei/internal/sourceindex"
	"tei/internal/syntax"
)

// These tests exercise the product Load, not a reference reader. Expected
// values come from somewhere other than the code under test: the existing
// `validation` grammar for meaning, bytes.Index on the authored document for
// positions, and a recording Source for read demand.

const teiMDRuleDocument = "semantic/sales/order-line-rules.tei.md"

// copyTEIMDRuleWorkspace copies the fixture with LF line endings, whatever the
// checkout did, so each test states its own line-ending variant.
func copyTEIMDRuleWorkspace(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	source := filepath.Join("testdata", "tei-md-rule")
	err := filepath.WalkDir(source, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		target := filepath.Join(root, relative)
		if entry.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, bytes.ReplaceAll(content, []byte("\r\n"), []byte("\n")), 0o644)
	})
	if err != nil {
		t.Fatal(err)
	}
	return root
}

func readTEIMDTestFile(t *testing.T, root, path string) []byte {
	t.Helper()
	content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
	if err != nil {
		t.Fatal(err)
	}
	return content
}

func rewriteTEIMDTestFile(t *testing.T, root, path, old, replacement string) {
	t.Helper()
	content := string(readTEIMDTestFile(t, root, path))
	if strings.Count(content, old) != 1 {
		t.Fatalf("%q must occur exactly once in %s, found %d", old, path, strings.Count(content, old))
	}
	writeTEIMDTestFile(t, root, path, strings.Replace(content, old, replacement, 1))
}

func validationsOf(result Result, namespace core.CanonicalID) []semantic.Validation {
	for _, ns := range result.World.Semantics {
		if ns.ID == namespace {
			out := append([]semantic.Validation(nil), ns.Validations...)
			sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
			return out
		}
	}
	return nil
}

func TestTEIMDRuleSelectedLanguageYieldsTheExistingValidationMeaning(t *testing.T) {
	root := copyTEIMDRuleWorkspace(t)
	loaded, err := Load(root)
	if err != nil {
		t.Fatalf("selected rule language did not load its source: %v", err)
	}

	// Independent expected meaning: the same two rules written in the existing
	// `validation` grammar, loaded by the existing loader, with no language
	// selected and no Markdown source.
	reference := copyTEIMDRuleWorkspace(t)
	if err := os.Remove(filepath.Join(reference, filepath.FromSlash(teiMDRuleDocument))); err != nil {
		t.Fatal(err)
	}
	writeTEIMDTestFile(t, reference, "tei.workspace.yaml", "workspace: tei_md_rule\nformatVersion: 1\nsources:\n  - semantic\n")
	writeTEIMDTestFile(t, reference, "semantic/sales/validation.tei", `namespace sales

validation sales.order_line.quantity_positive positive {
    attribute sales.order_line.quantity
}

validation sales.order.lines.line_number_unique relation-unique {
    relation sales.order.lines
    attribute sales.order_line.line_number
}
`)
	expected, err := Load(reference)
	if err != nil {
		t.Fatal(err)
	}
	got, want := validationsOf(loaded, "sales"), validationsOf(expected, "sales")
	if len(want) != 2 || !reflect.DeepEqual(got, want) {
		t.Fatalf("Markdown rules do not mean what the existing grammar means:\n got=%+v\nwant=%+v", got, want)
	}

	// The quoted example in a ```text fence is documentation.
	for _, validation := range got {
		if strings.Contains(string(validation.ID), "unit_price") {
			t.Fatalf("a quoted example became a definition: %+v", validation)
		}
	}

	wantRead := []AuthoringSourceRead{{
		Path: teiMDRuleDocument,
		Language: AuthoringLanguageBinding{
			LanguageID: "tei.rule-authoring", ContractVersion: "v0",
			ImplementationID: "tei.builtin.rule-authoring-markdown",
		},
		Regions: 2, Definitions: 2,
	}}
	if !reflect.DeepEqual(loaded.AuthoringSources, wantRead) {
		t.Fatalf("authoring read record=%+v, want %+v", loaded.AuthoringSources, wantRead)
	}
	for _, managed := range loaded.ManagedFiles {
		if managed.Path == teiMDRuleDocument && !managed.ReadByLoader {
			t.Fatal("the Markdown source was parsed but is reported as not read")
		}
	}
	for _, finding := range loaded.SourceRootFindings {
		if finding.File == teiMDRuleDocument {
			t.Fatalf("a source that was read is reported as unread: %+v", finding)
		}
	}
}

func TestTEIMDRuleSourceIndexPointsAtTheAuthoredBytes(t *testing.T) {
	root := copyTEIMDRuleWorkspace(t)
	loaded, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	index, err := BuildSourceIndexFromLoad(root, loaded)
	if err != nil {
		t.Fatalf("strict Source Index from Load-time bytes: %v", err)
	}
	document := readTEIMDTestFile(t, root, teiMDRuleDocument)

	for _, id := range []string{"sales.order_line.quantity_positive", "sales.order.lines.line_number_unique"} {
		if bytes.Count(document, []byte(id)) != 1 {
			t.Fatalf("fixture must write %s exactly once", id)
		}
		wantStart := bytes.Index(document, []byte(id))
		refs := index.ReferencesFor(sourceindex.Subject{Kind: "semantic-validation", ID: id})
		if len(refs) != 1 {
			t.Fatalf("%s has %d source references, want 1", id, len(refs))
		}
		ref := refs[0]
		if ref.ArtifactPath != teiMDRuleDocument || ref.Role != sourceindex.RoleDefinition {
			t.Fatalf("%s reference=%+v", id, ref)
		}
		if ref.Range.Start.Offset != wantStart || ref.Range.End.Offset != wantStart+len(id) {
			t.Fatalf("%s range=[%d,%d), want [%d,%d)", id,
				ref.Range.Start.Offset, ref.Range.End.Offset, wantStart, wantStart+len(id))
		}
		wantLine := 1 + bytes.Count(document[:wantStart], []byte("\n"))
		if ref.Range.Start.Line != wantLine {
			t.Fatalf("%s line=%d, want %d", id, ref.Range.Start.Line, wantLine)
		}
	}

	var artifact *sourceindex.Artifact
	for i := range index.Artifacts {
		if index.Artifacts[i].Path == teiMDRuleDocument {
			artifact = &index.Artifacts[i]
		}
	}
	if artifact == nil || artifact.Fingerprint != sourceindex.Fingerprint(document) {
		t.Fatalf("artifact does not identify the authored bytes: %+v", artifact)
	}

	// A later edit makes the old index stale. Prose counts: it is part of the
	// document's identity even though it defines nothing.
	writeTEIMDTestFile(t, root, teiMDRuleDocument, string(document)+"\n追記した説明。\n")
	readDisk := func(path string) ([]byte, error) {
		return os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
	}
	if err := index.Validate(readDisk); err == nil || !strings.Contains(err.Error(), "fingerprint is stale") {
		t.Fatalf("an index of the old bytes validated against the edited document: %v", err)
	}
	// The strict builder keeps answering from the bytes Load read.
	again, err := BuildSourceIndexFromLoad(root, loaded)
	if err != nil || !reflect.DeepEqual(again, index) {
		t.Fatalf("strict index re-read the file after Load: %v", err)
	}
}

// recordingRuleSource records what the evaluator asks for. Strict mode refuses
// a request for anything but the declared attribute.
type recordingRuleSource struct {
	strict   bool
	declared core.CanonicalID
	value    *big.Rat
	calls    []string
}

func (s *recordingRuleSource) Number(attribute core.CanonicalID) (*big.Rat, error) {
	s.calls = append(s.calls, "Number("+string(attribute)+")")
	if s.strict && attribute != s.declared {
		return nil, fmt.Errorf("unexpected Number request for %s", attribute)
	}
	return s.value, nil
}

func (s *recordingRuleSource) RelationKeys(relation, attribute core.CanonicalID) ([]string, error) {
	s.calls = append(s.calls, "RelationKeys("+string(relation)+","+string(attribute)+")")
	return nil, fmt.Errorf("unexpected RelationKeys request")
}

func loadedValidation(t *testing.T, result Result, id core.CanonicalID) semantic.Validation {
	t.Helper()
	for _, validation := range validationsOf(result, "sales") {
		if validation.ID == id {
			return validation
		}
	}
	t.Fatalf("validation %s was not loaded", id)
	return semantic.Validation{}
}

// A rule that reads another attribute can return the same result when the two
// attributes happen to hold the same value. Equal results therefore do not
// show that the right rule was loaded; the read demand does.
func TestTEIMDRuleEqualResultDoesNotHideAWrongAttribute(t *testing.T) {
	const id = "sales.order_line.quantity_positive"
	authored := copyTEIMDRuleWorkspace(t)
	loaded, err := Load(authored)
	if err != nil {
		t.Fatal(err)
	}
	wrong := copyTEIMDRuleWorkspace(t)
	rewriteTEIMDTestFile(t, wrong, teiMDRuleDocument,
		"sales.order_line.quantity is positive", "sales.order_line.line_number is positive")
	loadedWrong, err := Load(wrong)
	if err != nil {
		t.Fatal(err)
	}

	two := big.NewRat(2, 1)
	right := &recordingRuleSource{value: two}
	other := &recordingRuleSource{value: two}
	gotRight, errRight := validationeval.Evaluate(loadedValidation(t, loaded, id), right)
	gotOther, errOther := validationeval.Evaluate(loadedValidation(t, loadedWrong, id), other)
	if errRight != nil || errOther != nil || gotRight != validationeval.ResultSatisfied || gotOther != gotRight {
		t.Fatalf("both rules must be satisfied by the value 2: %q/%v %q/%v", gotRight, errRight, gotOther, errOther)
	}
	if got := strings.Join(right.calls, ","); got != "Number(sales.order_line.quantity)" {
		t.Fatalf("authored rule read %q", got)
	}
	if got := strings.Join(other.calls, ","); got != "Number(sales.order_line.line_number)" {
		t.Fatalf("changed rule read %q", got)
	}
	strict := &recordingRuleSource{strict: true, declared: "sales.order_line.quantity", value: two}
	if got, err := validationeval.Evaluate(loadedValidation(t, loadedWrong, id), strict); err == nil {
		t.Fatalf("a rule reading another attribute passed the declared read demand: %q", got)
	}

	// Zero is a value and fails; it is not the same as having no value.
	zero := &recordingRuleSource{strict: true, declared: "sales.order_line.quantity", value: new(big.Rat)}
	if got, err := validationeval.Evaluate(loadedValidation(t, loaded, id), zero); err != nil || got != validationeval.ResultNotSatisfied {
		t.Fatalf("quantity 0: result=%q err=%v, want not satisfied", got, err)
	}
}

func TestTEIMDRuleSelectionNegatives(t *testing.T) {
	const selected = "workspace: tei_md_rule\nformatVersion: 1\nsources:\n  - semantic\n  - plugins\nlanguages:\n  - tei.rule-authoring/v0\n"
	cases := []struct {
		name    string
		arrange func(t *testing.T, root string)
		want    string
	}{
		{"no language selected", func(t *testing.T, root string) {
			writeTEIMDTestFile(t, root, "tei.workspace.yaml", "workspace: tei_md_rule\nformatVersion: 1\nsources:\n  - semantic\n")
		}, "TEI_MD_PARSER_UNBOUND"},
		{"empty languages section", func(t *testing.T, root string) {
			writeTEIMDTestFile(t, root, "tei.workspace.yaml", "workspace: tei_md_rule\nformatVersion: 1\nsources:\n  - semantic\nlanguages:\n")
		}, "TEI_MD_PARSER_UNBOUND"},
		{"version this tei does not carry", func(t *testing.T, root string) {
			rewriteTEIMDTestFile(t, root, "tei.workspace.yaml", "tei.rule-authoring/v0", "tei.rule-authoring/v1")
		}, "TEI_LANGUAGE_UNAVAILABLE"},
		{"language without a version", func(t *testing.T, root string) {
			rewriteTEIMDTestFile(t, root, "tei.workspace.yaml", "tei.rule-authoring/v0", "tei.rule-authoring")
		}, "<language-id>/<contract-version>"},
		{"language selected twice", func(t *testing.T, root string) {
			rewriteTEIMDTestFile(t, root, "tei.workspace.yaml", "  - tei.rule-authoring/v0\n", "  - tei.rule-authoring/v0\n  - tei.rule-authoring/v0\n")
		}, "selected more than once"},
		{"source root the language does not read", func(t *testing.T, root string) {
			writeTEIMDTestFile(t, root, "tei.workspace.yaml", selected)
			writeTEIMDTestFile(t, root, "plugins/plugin.tei.md", "```tei\nnamespace sales\nrule sales.r: sales.order_line.quantity is positive\n```\n")
		}, "TEI_MD_PARSER_UNBOUND"},
		{"suffix in another case", func(t *testing.T, root string) {
			writeTEIMDTestFile(t, root, "semantic/sales/more.tei.MD", "```tei\nnamespace sales\nrule sales.r: sales.order_line.quantity is positive\n```\n")
		}, "TEI_MD_PARSER_UNBOUND"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := copyTEIMDRuleWorkspace(t)
			tc.arrange(t, root)
			loaded, err := Load(root)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("want an error containing %q, got %v (validations=%+v)", tc.want, err, validationsOf(loaded, "sales"))
			}
		})
	}
}

func countingAuthoringProvider(language, version, implementation string, calls *int, load func(string, *Result, string, []byte) (int, int, error)) authoringLanguageProvider {
	return authoringLanguageProvider{
		Binding:    AuthoringLanguageBinding{LanguageID: language, ContractVersion: version, ImplementationID: implementation},
		SourceRoot: "semantic",
		Suffix:     teiMarkdownSuffix,
		Load: func(root string, r *Result, filename string, src []byte) (int, int, error) {
			*calls++
			if load == nil {
				return 0, 0, nil
			}
			return load(root, r, filename, src)
		},
	}
}

func loadWithAuthoringProviders(root string, providers ...authoringLanguageProvider) (Result, error) {
	return loadWithStackHooks(root, map[string]bool{}, loadHooks{authoringProviders: providers})
}

func TestTEIMDRuleOnlyTheSelectedImplementationReads(t *testing.T) {
	baseline, err := Load(copyTEIMDRuleWorkspace(t))
	if err != nil {
		t.Fatal(err)
	}

	t.Run("an unselected implementation changes nothing", func(t *testing.T) {
		var ruleCalls, otherCalls int
		rule := countingAuthoringProvider("tei.rule-authoring", "v0", "tei.builtin.rule-authoring-markdown", &ruleCalls, loadRuleAuthoringMarkdown)
		other := countingAuthoringProvider("example.other", "v1", "example.other-reader", &otherCalls, nil)
		for _, providers := range [][]authoringLanguageProvider{{rule, other}, {other, rule}} {
			ruleCalls, otherCalls = 0, 0
			loaded, err := loadWithAuthoringProviders(copyTEIMDRuleWorkspace(t), providers...)
			if err != nil {
				t.Fatal(err)
			}
			if ruleCalls != 1 || otherCalls != 0 {
				t.Fatalf("calls: selected=%d unselected=%d, want 1 and 0", ruleCalls, otherCalls)
			}
			if !reflect.DeepEqual(validationsOf(loaded, "sales"), validationsOf(baseline, "sales")) {
				t.Fatal("adding an unselected implementation changed the loaded meaning")
			}
		}
	})

	t.Run("selecting B reads with B and never with A", func(t *testing.T) {
		var ruleCalls, otherCalls int
		rule := countingAuthoringProvider("tei.rule-authoring", "v0", "tei.builtin.rule-authoring-markdown", &ruleCalls, loadRuleAuthoringMarkdown)
		other := countingAuthoringProvider("example.other", "v1", "example.other-reader", &otherCalls, nil)
		root := copyTEIMDRuleWorkspace(t)
		rewriteTEIMDTestFile(t, root, "tei.workspace.yaml", "tei.rule-authoring/v0", "example.other/v1")
		loaded, err := loadWithAuthoringProviders(root, rule, other)
		if err != nil {
			t.Fatal(err)
		}
		if ruleCalls != 0 || otherCalls != 1 {
			t.Fatalf("calls: A=%d B=%d, want 0 and 1", ruleCalls, otherCalls)
		}
		if len(validationsOf(loaded, "sales")) != 0 {
			t.Fatal("the unselected rule implementation contributed meaning")
		}
		if len(loaded.AuthoringSources) != 1 || loaded.AuthoringSources[0].Language.ImplementationID != "example.other-reader" {
			t.Fatalf("read record names the wrong implementation: %+v", loaded.AuthoringSources)
		}
	})

	t.Run("two selected languages claiming one source are refused", func(t *testing.T) {
		var ruleCalls, otherCalls int
		rule := countingAuthoringProvider("tei.rule-authoring", "v0", "tei.builtin.rule-authoring-markdown", &ruleCalls, loadRuleAuthoringMarkdown)
		other := countingAuthoringProvider("example.other", "v1", "example.other-reader", &otherCalls, nil)
		root := copyTEIMDRuleWorkspace(t)
		rewriteTEIMDTestFile(t, root, "tei.workspace.yaml", "  - tei.rule-authoring/v0\n", "  - tei.rule-authoring/v0\n  - example.other/v1\n")
		if _, err := loadWithAuthoringProviders(root, rule, other); err == nil || !strings.Contains(err.Error(), "TEI_MD_PARSER_AMBIGUOUS") {
			t.Fatalf("ambiguous claim was resolved silently: %v", err)
		}
		if ruleCalls != 0 || otherCalls != 0 {
			t.Fatalf("an implementation read an ambiguous source: %d %d", ruleCalls, otherCalls)
		}
	})

	t.Run("two implementations of one contract are refused in either order", func(t *testing.T) {
		var firstCalls, secondCalls int
		first := countingAuthoringProvider("tei.rule-authoring", "v0", "implementation-a", &firstCalls, loadRuleAuthoringMarkdown)
		second := countingAuthoringProvider("tei.rule-authoring", "v0", "implementation-b", &secondCalls, nil)
		for _, providers := range [][]authoringLanguageProvider{{first, second}, {second, first}} {
			if _, err := loadWithAuthoringProviders(copyTEIMDRuleWorkspace(t), providers...); err == nil ||
				!strings.Contains(err.Error(), "TEI_LANGUAGE_AMBIGUOUS") {
				t.Fatalf("registration order chose an implementation: %v", err)
			}
		}
		if firstCalls != 0 || secondCalls != 0 {
			t.Fatalf("an ambiguous implementation read a source: %d %d", firstCalls, secondCalls)
		}
	})
}

func TestTEIMDRuleIdentityCannotBeDeclaredTwice(t *testing.T) {
	t.Run("native validation and Markdown rule", func(t *testing.T) {
		root := copyTEIMDRuleWorkspace(t)
		writeTEIMDTestFile(t, root, "semantic/sales/validation.tei",
			"namespace sales\n\nvalidation sales.order_line.quantity_positive positive {\n    attribute sales.order_line.unit_price\n}\n")
		_, err := Load(root)
		var sourceError *SourceError
		if !errors.As(err, &sourceError) {
			t.Fatalf("two declarations of one identity loaded: %v", err)
		}
		document := readTEIMDTestFile(t, root, teiMDRuleDocument)
		if want := bytes.Index(document, []byte("sales.order_line.quantity_positive")); sourceError.Range.Start.Offset != want {
			t.Fatalf("error offset=%d, want the Markdown declaration at %d", sourceError.Range.Start.Offset, want)
		}
		if !strings.Contains(sourceError.Message, "semantic/sales/validation.tei") {
			t.Fatalf("message does not name the other declaration: %s", sourceError.Message)
		}
	})

	t.Run("two Markdown sources", func(t *testing.T) {
		root := copyTEIMDRuleWorkspace(t)
		writeTEIMDTestFile(t, root, "semantic/sales/z-copy.tei.md",
			"```tei\nnamespace sales\nrule sales.order_line.quantity_positive:\n  sales.order_line.unit_price is positive\n```\n")
		_, err := Load(root)
		if err == nil || !strings.Contains(err.Error(), "already declared at "+teiMDRuleDocument) {
			t.Fatalf("the later source silently replaced the earlier one: %v", err)
		}
	})

	t.Run("same symbol under another namespace is another identity", func(t *testing.T) {
		root := copyTEIMDRuleWorkspace(t)
		const other = "semantic/purchase/rules.tei.md"
		writeTEIMDTestFile(t, root, other,
			"# 発注明細\n\n```tei\nnamespace purchase\nrule purchase.order_line.quantity_positive:\n  purchase.order_line.quantity is positive\n```\n")
		loaded, err := Load(root)
		if err != nil {
			t.Fatal(err)
		}
		index, err := BuildSourceIndexFromLoad(root, loaded)
		if err != nil {
			t.Fatal(err)
		}
		for id, path := range map[string]string{
			"sales.order_line.quantity_positive":    teiMDRuleDocument,
			"purchase.order_line.quantity_positive": other,
		} {
			refs := index.ReferencesFor(sourceindex.Subject{Kind: "semantic-validation", ID: id})
			if len(refs) != 1 || refs[0].ArtifactPath != path {
				t.Fatalf("%s resolves to %+v, want one reference in %s", id, refs, path)
			}
			document := readTEIMDTestFile(t, root, path)
			if want := bytes.Index(document, []byte(id)); refs[0].Range.Start.Offset != want {
				t.Fatalf("%s offset=%d, want %d", id, refs[0].Range.Start.Offset, want)
			}
		}
		// The namespace the Markdown source introduced is anchored there too.
		refs := index.ReferencesFor(sourceindex.Subject{Kind: "semantic", ID: "purchase"})
		if len(refs) != 1 || refs[0].ArtifactPath != other {
			t.Fatalf("namespace purchase resolves to %+v", refs)
		}
	})
}

func TestTEIMDRuleSourceContributesAllOrNothing(t *testing.T) {
	root := copyTEIMDRuleWorkspace(t)
	rewriteTEIMDTestFile(t, root, teiMDRuleDocument,
		"sales.order_line.line_number is unique per sales.order.lines", "line_number is unique per lines")

	if _, err := Load(root); err == nil {
		t.Fatal("a source with an unresolved reference loaded")
	}

	// A validation load reports the problem at its own authored bytes and
	// returns no partial World.
	loaded, err := LoadForValidation(root)
	var sourceErrors *SourceErrors
	if !errors.As(err, &sourceErrors) {
		t.Fatalf("want collected source errors, got %v", err)
	}
	if len(validationsOf(loaded, "sales")) != 0 {
		t.Fatal("the readable first region contributed meaning although the source was refused")
	}
	document := readTEIMDTestFile(t, root, teiMDRuleDocument)
	var offsets []int
	for _, sourceError := range sourceErrors.Errors {
		offsets = append(offsets, sourceError.Range.Start.Offset)
	}
	want := []int{bytes.Index(document, []byte("line_number is unique"))}
	if !reflect.DeepEqual(offsets, want) {
		t.Fatalf("error offsets=%v, want %v", offsets, want)
	}
}

// One typed change ("this rule constrains that attribute") becomes one
// byte-range replacement of the authored document, and the saved document is
// read again by the product Load.
func TestTEIMDRuleTypedEditChangesOnlyItsBytesAndReloads(t *testing.T) {
	const ruleID = "sales.order_line.quantity_positive"
	for _, variant := range []string{"lf", "crlf", "bom"} {
		t.Run(variant, func(t *testing.T) {
			root := copyTEIMDRuleWorkspace(t)
			original := string(readTEIMDTestFile(t, root, teiMDRuleDocument))
			// The old attribute also appears in prose and in a quoted example, so
			// a text search for it would have three places to choose from.
			original = strings.Replace(original, "数量が0や負の数の明細は",
				"数量（sales.order_line.quantity）が0や負の数の明細は", 1)
			original = strings.Replace(original, "```text\n", "```text\nrule sales.example: sales.order_line.quantity is positive\n", 1)
			switch variant {
			case "crlf":
				original = strings.ReplaceAll(original, "\n", "\r\n")
			case "bom":
				original = "\xef\xbb\xbf" + original
			}
			if strings.Count(original, "sales.order_line.quantity is positive") != 2 ||
				strings.Count(original, "sales.order_line.quantity）") != 1 {
				t.Fatal("fixture must repeat the old attribute text outside the rule")
			}
			writeTEIMDTestFile(t, root, teiMDRuleDocument, original)
			before, err := Load(root)
			if err != nil {
				t.Fatal(err)
			}

			edit, err := syntax.PlanRuleSubjectEdit(original, ruleID, "sales.order_line.unit_price")
			if err != nil {
				t.Fatal(err)
			}
			index, err := BuildSourceIndexFromLoad(root, before)
			if err != nil {
				t.Fatal(err)
			}
			for _, artifact := range index.Artifacts {
				if artifact.Path == teiMDRuleDocument && artifact.Fingerprint != edit.BaseFingerprint {
					t.Fatalf("the plan and the Source Index name different documents: %s vs %s", edit.BaseFingerprint, artifact.Fingerprint)
				}
			}
			updated, err := syntax.ApplyTextEdit(original, edit)
			if err != nil {
				t.Fatal(err)
			}

			// Independent check of locality: the document is the old one with
			// exactly the planned range replaced, located here by the rule header.
			header := strings.Index(original, "rule "+ruleID+":")
			wantStart := header + strings.Index(original[header:], "sales.order_line.quantity is")
			if edit.Start != wantStart || edit.Old != "sales.order_line.quantity" {
				t.Fatalf("edit=[%d,%d) %q, want the rule's own subject at %d", edit.Start, edit.End, edit.Old, wantStart)
			}
			if updated[:edit.Start] != original[:edit.Start] || updated[edit.Start+len(edit.New):] != original[edit.End:] {
				t.Fatal("bytes outside the planned range changed")
			}

			writeTEIMDTestFile(t, root, teiMDRuleDocument, updated)
			after, err := Load(root)
			if err != nil {
				t.Fatalf("the saved document does not load: %v", err)
			}
			want := validationsOf(before, "sales")
			for i := range want {
				if want[i].ID == ruleID {
					want[i].Attribute = "sales.order_line.unit_price"
				}
			}
			if got := validationsOf(after, "sales"); !reflect.DeepEqual(got, want) {
				t.Fatalf("reloaded meaning=%+v, want only the edited rule to change: %+v", got, want)
			}

			// The same plan cannot be applied again: the document it names is gone.
			if _, err := syntax.ApplyTextEdit(updated, edit); err == nil || !strings.Contains(err.Error(), "stale edit") {
				t.Fatalf("a stale plan was applied to the saved document: %v", err)
			}
			// Nor to a document someone else changed elsewhere, although the
			// planned range still holds the same text.
			elsewhere := original + "\n別の人が追記した説明。\n"
			if _, err := syntax.ApplyTextEdit(elsewhere, edit); err == nil || !strings.Contains(err.Error(), "stale edit") {
				t.Fatalf("a plan for another document version was applied: %v", err)
			}
		})
	}

	t.Run("refusals", func(t *testing.T) {
		original := string(readTEIMDTestFile(t, copyTEIMDRuleWorkspace(t), teiMDRuleDocument))
		for name, plan := range map[string]func() (syntax.TextEdit, error){
			"unknown rule": func() (syntax.TextEdit, error) {
				return syntax.PlanRuleSubjectEdit(original, "sales.order_line.unit_price_positive", "sales.order_line.unit_price")
			},
			"unqualified attribute": func() (syntax.TextEdit, error) {
				return syntax.PlanRuleSubjectEdit(original, ruleID, "quantity")
			},
			"no change": func() (syntax.TextEdit, error) {
				return syntax.PlanRuleSubjectEdit(original, ruleID, "sales.order_line.quantity")
			},
		} {
			if edit, err := plan(); err == nil {
				t.Errorf("%s: planned %+v", name, edit)
			}
		}
	})
}
