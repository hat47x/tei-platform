package operations

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"tei/internal/semantic"
	"tei/internal/sourceindex"
	"tei/internal/workspace"
)

// This is a product Workspace Load of a selected, built-in Markdown language,
// not a reference parser call. One Load supplies the typed validation and its
// original byte span to the existing read-only Inspector and Source Index.
func TestInspectSelectedRuleMarkdownFromSameLoad(t *testing.T) {
	root := filepath.Join("..", "workspace", "testdata", "tei-md-rule")
	const sourcePath = "semantic/sales/order-line-rules.tei.md"
	loaded, err := workspace.Load(root)
	if err != nil {
		t.Fatalf("load selected Rule Markdown Workspace: %v", err)
	}
	if len(loaded.AuthoringSources) != 1 {
		t.Fatalf("selected Markdown reads=%d, want one: %+v", len(loaded.AuthoringSources), loaded.AuthoringSources)
	}
	read := loaded.AuthoringSources[0]
	if read.Path != sourcePath ||
		read.Language.LanguageID != "tei.rule-authoring" ||
		read.Language.ContractVersion != "v0" ||
		read.Language.ImplementationID != "tei.builtin.rule-authoring-markdown" ||
		read.Regions != 2 || read.Definitions != 2 {
		t.Fatalf("unexpected selected language or read coverage: %+v", read)
	}
	index, err := workspace.BuildSourceIndexFromLoad(root, loaded)
	if err != nil {
		t.Fatalf("build Source Index from the same Load bytes: %v", err)
	}
	raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(sourcePath)))
	if err != nil {
		t.Fatal(err)
	}
	expectedFingerprint := sourceindex.Fingerprint(raw)
	// Expected typed values are fixed from the authored declarations, not
	// copied from Inspector's output or generated Go logic.
	cases := []struct {
		id        string
		kind      semantic.ValidationKind
		attribute string
		relation  string
	}{
		{"sales.order_line.quantity_positive", semantic.ValidationPositive,
			"sales.order_line.quantity", ""},
		{"sales.order.lines.line_number_unique", semantic.ValidationRelationUnique,
			"sales.order_line.line_number", "sales.order.lines"},
	}
	for _, tc := range cases {
		t.Run(tc.id, func(t *testing.T) {
			subject := sourceindex.Subject{Kind: "semantic-validation", ID: tc.id}
			inspection, err := InspectionFromLoad(loaded, index, subject)
			if err != nil {
				t.Fatalf("inspect selected Markdown rule: %v", err)
			}
			if inspection.Subject != subject || inspection.Effect != "read-only" ||
				inspection.SourceStatus != "indexed" ||
				!reflect.DeepEqual(inspection.AuthorityLimits, InspectionAuthorityLimits()) {
				t.Fatalf("inspection lost its source/authority boundary: %+v", inspection)
			}
			var value struct {
				ID        string                  `json:"id"`
				Kind      semantic.ValidationKind `json:"kind"`
				Attribute string                  `json:"attribute"`
				Relation  string                  `json:"relation"`
			}
			if err := json.Unmarshal(inspection.Canonical, &value); err != nil {
				t.Fatalf("parse typed canonical validation: %v", err)
			}
			if value.ID != tc.id || value.Kind != tc.kind ||
				value.Attribute != tc.attribute || value.Relation != tc.relation {
				t.Fatalf("canonical validation differs from authored meaning: %+v want %+v", value, tc)
			}

			refs := inspection.Sources.References
			artifacts := inspection.Sources.Artifacts
			if len(refs) != 1 || len(artifacts) != 1 ||
				refs[0].Subject != subject || refs[0].ArtifactPath != sourcePath ||
				refs[0].Role != sourceindex.RoleDefinition ||
				artifacts[0].Path != sourcePath || artifacts[0].Fingerprint != expectedFingerprint {
				t.Fatalf("rule is not tied to one original Markdown source: %+v", inspection.Sources)
			}
			if err := refs[0].Range.Validate(raw); err != nil {
				t.Fatalf("rule byte range is stale: %v", err)
			}
			span := refs[0].Range
			if got := string(raw[span.Start.Offset:span.End.Offset]); got != tc.id {
				t.Fatalf("read rule ID at wrong original byte span: got %q want %q", got, tc.id)
			}
			fromIndex, err := TraceSubjectSourcesFromIndex(index, subject)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(fromIndex, inspection.Sources) {
				t.Fatalf("Inspector and Source Index disagree: inspect=%+v trace=%+v", inspection.Sources, fromIndex)
			}
		})
	}

	// The same Markdown file contains this text inside a fenced text example,
	// not in a selected tei definition region. Finding a string in Source
	// is not evidence of a Canonical Rule or an executable Provider.
	quoted := sourceindex.Subject{Kind: "semantic-validation", ID: "sales.order_line.unit_price_positive"}
	found, err := TraceSubjectSourcesFromIndex(index, quoted)
	if err != nil {
		t.Fatal(err)
	}
	if len(found.References) != 0 || len(found.Artifacts) != 0 {
		t.Fatalf("text-only example entered the Source Index: %+v", found)
	}
	_, err = InspectionFromLoad(loaded, index, quoted)
	inspectionCode(t, err, InspectionNotFound)
}
