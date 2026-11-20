package workspace

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"tei/internal/sourceindex"
)

func TestExplicitMathPreviewUsesSelectedLoadBytesAndNeverDefinesRule(t *testing.T) {
	baselineRoot := copyTEIMDRuleWorkspace(t)
	baseline, err := Load(baselineRoot)
	if err != nil {
		t.Fatal(err)
	}
	wantRules := validationsOf(baseline, "sales")

	root := copyTEIMDRuleWorkspace(t)
	original := readTEIMDTestFile(t, root, teiMDRuleDocument)
	const inline = "\\(\\frac{1}{2}\\)"
	const display = "\\[\r\n\\sqrt{x^2}=\\lvert x\\rvert\r\n\\]"
	preface := "\xef\xbb\xbf# 数式のプレビュー\r\n\r\n金額は$100。説明用の " +
		inline + " と " + inline + " は同じTeXでも別の出現です。\r\n\r\n" +
		display + "\r\n\r\n~~~text\r\n" + inline + "\r\n~~~\r\n\r\n"
	authored := append([]byte(preface), bytes.ReplaceAll(original, []byte("\n"), []byte("\r\n"))...)
	writeTEIMDTestFile(t, root, teiMDRuleDocument, string(authored))

	loaded, err := Load(root)
	if err != nil {
		t.Fatalf("selected Markdown Reader rejected prose math: %v", err)
	}
	if got := validationsOf(loaded, "sales"); !reflect.DeepEqual(got, wantRules) {
		t.Fatalf("display prose changed accepted meaning: got=%+v want=%+v", got, wantRules)
	}
	if len(loaded.AuthoringSources) != 1 || loaded.AuthoringSources[0].Definitions != 2 {
		t.Fatalf("display prose changed authoring definitions: %+v", loaded.AuthoringSources)
	}
	fingerprint := sourceindex.Fingerprint(authored)
	selectAt := func(start int, literal string) (ExplicitMathPreview, error) {
		t.Helper()
		return SelectExplicitMathPreview(root, loaded, ExplicitMathPreviewRequest{
			Path: teiMDRuleDocument, SourceFingerprint: fingerprint,
			StartOffset: start, EndOffset: start + len(literal),
		})
	}
	first := bytes.Index(authored, []byte(inline))
	second := bytes.Index(authored[first+len(inline):], []byte(inline)) + first + len(inline)
	if first < 0 || second < first+len(inline) {
		t.Fatal("two explicit occurrences were not found")
	}
	one, err := selectAt(first, inline)
	if err != nil {
		t.Fatal(err)
	}
	two, err := selectAt(second, inline)
	if err != nil {
		t.Fatal(err)
	}
	if one.Display || one.Delimiter != "tex-inline-parens" ||
		one.TeX != "\\frac{1}{2}" || one.Occurrence == two.Occurrence ||
		one.SourceRange.Start.Offset != first || one.TeXRange.Start.Offset != first+2 ||
		one.SourceFingerprint != fingerprint {
		t.Fatalf("explicit occurrence was not tied to its exact source bytes: %+v / %+v", one, two)
	}

	displayAt := bytes.Index(authored, []byte(display))
	shown, err := selectAt(displayAt, display)
	if err != nil || !shown.Display || shown.Delimiter != "tex-display-brackets" ||
		shown.TeX != "\r\n\\sqrt{x^2}=\\lvert x\\rvert\r\n" {
		t.Fatalf("multiline CRLF display math lost raw TeX: %+v / %v", shown, err)
	}
	if got := string(authored[shown.TeXRange.Start.Offset:shown.TeXRange.End.Offset]); got != shown.TeX {
		t.Fatalf("TeX range does not identify the authored bytes: %q / %q", got, shown.TeX)
	}
	if shown.SourceRange.Start.Line < 1 || shown.TeXRange.End.Offset > len(authored) {
		t.Fatalf("source position was not reconstructed: %+v", shown)
	}
	// An explicit selection is a preview, not a new semantic Source reference.
	index, err := BuildSourceIndexFromLoad(root, loaded)
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{
		"sales.order_line.quantity_positive",
		"sales.order.lines.line_number_unique",
	} {
		refs := index.ReferencesFor(sourceindex.Subject{Kind: "semantic-validation", ID: id})
		if len(refs) != 1 || refs[0].Role != sourceindex.RoleDefinition ||
			refs[0].Range.Start.Offset != bytes.Index(authored, []byte(id)) {
			t.Fatalf("meaning reference was affected by display preview: %+v", refs)
		}
	}

	// The old Load is an immutable historical source; an on-disk change
	// cannot silently alter its preview or qualify it for current editing.
	writeTEIMDTestFile(t, root, teiMDRuleDocument, string(authored)+"\r\n更新された説明。\r\n")
	historical, err := selectAt(first, inline)
	if err != nil || historical != one {
		t.Fatalf("historical Load preview was re-read from disk: %+v / %v", historical, err)
	}
	readDisk := func(path string) ([]byte, error) {
		return os.ReadFile(filepath.Join(root, filepath.FromSlash(path)))
	}
	if err := index.Validate(readDisk); err == nil ||
		!strings.Contains(err.Error(), "fingerprint is stale") {
		t.Fatalf("old Source Index remained current after edit: %v", err)
	}
	updated, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := SelectExplicitMathPreview(root, updated, ExplicitMathPreviewRequest{
		Path: teiMDRuleDocument, SourceFingerprint: fingerprint,
		StartOffset: first, EndOffset: first + len(inline),
	}); err == nil || !strings.Contains(err.Error(), "fingerprint") {
		t.Fatalf("old request fingerprint qualified a new source revision: %v", err)
	}
	revised, err := SelectExplicitMathPreview(root, updated, ExplicitMathPreviewRequest{
		Path: teiMDRuleDocument, SourceFingerprint: sourceindex.Fingerprint(
			append(bytes.Clone(authored), []byte("\r\n更新された説明。\r\n")...)),
		StartOffset: first, EndOffset: first + len(inline),
	})
	if err != nil || revised.Occurrence == one.Occurrence || revised.TeX != one.TeX {
		t.Fatalf("different revisions shared a preview occurrence identity: %+v / %v", revised, err)
	}
	if got := validationsOf(updated, "sales"); !reflect.DeepEqual(got, wantRules) {
		t.Fatalf("display edit changed accepted Rules: %+v", got)
	}
}

func TestExplicitMathPreviewFailClosedWithoutSelectionAuthority(t *testing.T) {
	root := copyTEIMDRuleWorkspace(t)
	preface := "# 日本語と通貨 $100。\n\n\\(x^2\\)\n\n$$y^2$$\n\n"
	original := readTEIMDTestFile(t, root, teiMDRuleDocument)
	authored := append([]byte(preface), original...)
	writeTEIMDTestFile(t, root, teiMDRuleDocument, string(authored))
	loaded, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	digest := sourceindex.Fingerprint(authored)
	fragment := "\\(x^2\\)"
	start := bytes.Index(authored, []byte(fragment))
	valid := ExplicitMathPreviewRequest{
		Path: teiMDRuleDocument, SourceFingerprint: digest,
		StartOffset: start, EndOffset: start + len(fragment),
	}
	if _, err := SelectExplicitMathPreview(root, loaded, valid); err != nil {
		t.Fatalf("positive control failed: %v", err)
	}
	dollars := bytes.Index(authored, []byte("$100"))
	cases := []struct {
		name string
		edit func(*ExplicitMathPreviewRequest)
	}{
		{"missing revision", func(r *ExplicitMathPreviewRequest) { r.SourceFingerprint = "" }},
		{"wrong revision", func(r *ExplicitMathPreviewRequest) { r.SourceFingerprint = "sha256:other" }},
		{"unselected path", func(r *ExplicitMathPreviewRequest) { r.Path = "drafts/unselected.tei.md" }},
		{"path traversal", func(r *ExplicitMathPreviewRequest) { r.Path = "../semantic/model.tei.md" }},
		{"ordinary Markdown", func(r *ExplicitMathPreviewRequest) { r.Path = "semantic/README.md" }},
		{"mid-token selection", func(r *ExplicitMathPreviewRequest) { r.StartOffset++ }},
		{"empty span", func(r *ExplicitMathPreviewRequest) { r.EndOffset = r.StartOffset }},
		{"outside bytes", func(r *ExplicitMathPreviewRequest) { r.EndOffset = len(authored) + 1 }},
		{"currency", func(r *ExplicitMathPreviewRequest) {
			r.StartOffset, r.EndOffset = dollars, dollars+len("$100")
		}},
		{"single dollar math not admitted", func(r *ExplicitMathPreviewRequest) {
			r.StartOffset, r.EndOffset = dollars, dollars+1
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			request := valid
			tc.edit(&request)
			if result, err := SelectExplicitMathPreview(root, loaded, request); err == nil {
				t.Fatalf("unqualified selection succeeded: %+v", result)
			}
		})
	}
	_, unsupported := SelectExplicitMathPreview(root, loaded, ExplicitMathPreviewRequest{
		Path: teiMDRuleDocument, SourceFingerprint: digest,
		StartOffset: dollars, EndOffset: dollars + len("$100"),
	})
	if !errors.Is(unsupported, ErrMathPreviewUnsupported) {
		t.Fatalf("single-dollar currency must have a typed unsupported outcome: %v", unsupported)
	}
	// Dollars enclosing an explicitly selected display expression are valid
	// only as a whole selection, not as automatic currency detection.
	display := "$$y^2$$"
	displayAt := bytes.Index(authored, []byte(display))
	shown, err := SelectExplicitMathPreview(root, loaded, ExplicitMathPreviewRequest{
		Path: teiMDRuleDocument, SourceFingerprint: digest,
		StartOffset: displayAt, EndOffset: displayAt + len(display),
	})
	if err != nil || !shown.Display || shown.TeX != "y^2" || shown.Delimiter != "tex-display-dollars" {
		t.Fatalf("explicit dollar-delimited display selection failed: %+v / %v", shown, err)
	}
	other := copyTEIMDRuleWorkspace(t)
	if _, err := SelectExplicitMathPreview(other, loaded, valid); err == nil {
		t.Fatal("a Load snapshot from another Workspace was accepted")
	}
}
