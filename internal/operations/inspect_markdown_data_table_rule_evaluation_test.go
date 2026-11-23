package operations

import (
	"bytes"
	"encoding/json"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/codegen/goemit"
	"tei/internal/codegen/goemit/goemittest"
	"tei/internal/core"
	"tei/internal/semantic"
	"tei/internal/semantic/validationeval"
	"tei/internal/semantic/validationgen"
	"tei/internal/sourceindex"
	"tei/internal/validation"
	"tei/internal/workspace"
)

// A single marked Markdown document defines both the Data attribute's type
// and a Rule that reads it. The same typed Rule can be interpreted by the
// existing pure Evaluator or lowered to Go: the second path must not invent
// a new attribute, predicate, Reader, or execution authority.
func TestMarkedDataTableRuleSharesEvaluatorAndGoLoweringMeaning(t *testing.T) {
	root := filepath.Join("..", "workspace", "testdata", "tei-md-data-table")
	const sourcePath = "semantic/organization/transfer-request.tei.md"
	const attributeID = "organization.transfer_request.allocation"
	const ruleID = "organization.transfer_request.allocation_positive"
	loaded, err := workspace.Load(root)
	if err != nil {
		t.Fatalf("product Workspace Load: %v", err)
	}
	if len(loaded.AuthoringSources) != 1 ||
		loaded.AuthoringSources[0].Path != sourcePath ||
		loaded.AuthoringSources[0].Language.LanguageID != "tei.authoring" ||
		loaded.AuthoringSources[0].Language.ContractVersion != "v1" ||
		loaded.AuthoringSources[0].Definitions != 2 {
		t.Fatalf("selected Markdown Data/Rule Reader changed: %+v", loaded.AuthoringSources)
	}
	if diagnostics := validation.Validate(loaded.World); len(diagnostics) != 0 {
		t.Fatalf("marked Data and its Rule are not compatible: %+v", diagnostics)
	}
	index, err := workspace.BuildSourceIndexFromLoad(root, loaded)
	if err != nil {
		t.Fatalf("same-load Source Index: %v", err)
	}
	raw, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(sourcePath)))
	if err != nil {
		t.Fatal(err)
	}

	inspect := func(kind, id, authoredToken string) json.RawMessage {
		t.Helper()
		subject := sourceindex.Subject{Kind: kind, ID: id}
		got, err := InspectionFromLoad(loaded, index, subject)
		if err != nil {
			t.Fatalf("inspect %v: %v", subject, err)
		}
		if err := ValidateInspectionProjection(subject, got); err != nil {
			t.Fatalf("invalid read-only projection for %v: %v", subject, err)
		}
		if got.Effect != "read-only" || len(got.Sources.References) != 1 ||
			len(got.Sources.Artifacts) != 1 || got.Sources.References[0].Role != sourceindex.RoleDefinition ||
			got.Sources.References[0].ArtifactPath != sourcePath ||
			got.Sources.Artifacts[0].Fingerprint != sourceindex.Fingerprint(raw) {
			t.Fatalf("the evaluator's canonical input lacks its exact Source: %+v", got)
		}
		span := got.Sources.References[0].Range
		if err := span.Validate(raw); err != nil {
			t.Fatalf("invalid authored range %v: %v", subject, err)
		}
		if bytes.Count(raw, []byte(authoredToken)) != 1 ||
			span.Start.Offset != bytes.Index(raw, []byte(authoredToken)) {
			t.Fatalf("wrong source cell/Rule for %v: %+v", subject, span)
		}
		return got.Canonical
	}

	var attribute semantic.Attribute
	if err := json.Unmarshal(inspect("semantic-attribute", attributeID, "allocation | std.decimal"), &attribute); err != nil {
		t.Fatal(err)
	}
	var rule semantic.Validation
	if err := json.Unmarshal(inspect("semantic-validation", ruleID, ruleID), &rule); err != nil {
		t.Fatal(err)
	}
	if attribute.ID != core.CanonicalID(attributeID) || attribute.TypeRef != "std.decimal" ||
		rule.ID != core.CanonicalID(ruleID) || rule.Kind != semantic.ValidationPositive ||
		rule.Attribute != attribute.ID || rule.Relation != "" {
		t.Fatalf("Data/Rule typed read binding changed: attribute=%+v rule=%+v", attribute, rule)
	}

	// The interpreter and the Go generator take the same typed Rule. They are
	// distinct realizations, not two sources of business meaning.
	generated, err := validationgen.GeneratePositive(rule, attribute.TypeRef, "main")
	if err != nil {
		t.Fatalf("lower product-selected table Rule: %v", err)
	}
	// The generated Go reads nothing itself; the Trace of the same lowering
	// names the attribute whose value the caller passes to the check
	// function, checked against the function the file declares.
	out := goemit.Emit(goemit.Input{PackageName: "main", Rules: []goemit.Rule{{Validation: rule, AttributeType: attribute.TypeRef}}})
	if !bytes.Equal(generated, out.File("allocation_is_positive.go")) ||
		goemittest.SingleBinding(t, out) != (goemittest.Binding{
			Check: "CheckAllocationIsPositive", Parameter: "allocation", Attribute: attributeID,
		}) ||
		!bytes.Contains(generated, []byte("func CheckAllocationIsPositive(allocation *big.Rat) Result")) ||
		!bytes.Contains(generated, []byte("allocation.Sign() > 0")) ||
		bytes.Contains(generated, []byte("validationeval")) {
		t.Fatalf("generated Go lost the selected attribute or the actual decision:\n%s", generated)
	}

	cases := []struct {
		name      string
		number    *big.Rat
		complete  bool
		want      validationeval.Result
		wantError bool
	}{
		{"positive", big.NewRat(2, 1), true, validationeval.ResultSatisfied, false},
		{"zero", big.NewRat(0, 1), true, validationeval.ResultNotSatisfied, false},
		{"negative", big.NewRat(-1, 1), true, validationeval.ResultNotSatisfied, false},
		{"null", nil, true, "", true},
		{"not-obtained", nil, false, "", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			source := &evaluatorReadProbe{
				attribute: attribute.ID,
				number:    tc.number,
				complete:  tc.complete,
			}
			result, err := validationeval.Evaluate(rule, source)
			if (err != nil) != tc.wantError || result != tc.want {
				t.Fatalf("Evaluator result=%q err=%v, want %q error=%t", result, err, tc.want, tc.wantError)
			}
			if len(source.reads) != 1 || source.reads[0] != "Number:"+attributeID {
				t.Fatalf("Evaluator used an unqualified read instead of table attribute: %v", source.reads)
			}
		})
	}
}

// Changing only the Data table's Attribute type must keep the Rule's Source
// identity but invalidate the typed connection. A text Attribute must not
// reach either the numeric Go generator or an accepted evaluator attempt.
func TestMarkedDataTableTypeChangeRefusesPositiveRule(t *testing.T) {
	fixture := filepath.Join("..", "workspace", "testdata", "tei-md-data-table")
	root := t.TempDir()
	const sourcePath = "semantic/organization/transfer-request.tei.md"
	copyFixture := func(path string) []byte {
		t.Helper()
		raw, err := os.ReadFile(filepath.Join(fixture, filepath.FromSlash(path)))
		if err != nil {
			t.Fatal(err)
		}
		destination := filepath.Join(root, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(destination, raw, 0o644); err != nil {
			t.Fatal(err)
		}
		return raw
	}
	copyFixture("tei.workspace.yaml")
	raw := copyFixture(sourcePath)

	const original = "| allocation | std.decimal |"
	const changed = "| allocation | std.text |"
	if bytes.Count(raw, []byte(original)) != 1 {
		t.Fatal("the baseline must contain exactly one editable type cell")
	}
	before, err := ReadWorkspaceSubject(root, sourceindex.Subject{
		Kind: "semantic-validation", ID: "organization.transfer_request.allocation_positive",
	})
	if err != nil {
		t.Fatal(err)
	}
	revised := bytes.Replace(raw, []byte(original), []byte(changed), 1)
	if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(sourcePath)), revised, 0o644); err != nil {
		t.Fatal(err)
	}
	loaded, err := workspace.Load(root)
	if err != nil {
		t.Fatalf("type edit must remain parseable, not silently ignored: %v", err)
	}
	diagnostics := validation.Validate(loaded.World)
	if len(diagnostics) != 1 ||
		diagnostics[0].Code != "SEMANTIC_VALIDATION_POSITIVE_REQUIRES_NUMERIC_ATTRIBUTE" {
		t.Fatalf("non-numeric marked table input was accepted or misclassified: %+v", diagnostics)
	}
	index, err := workspace.BuildSourceIndexFromLoad(root, loaded)
	if err != nil {
		t.Fatal(err)
	}
	attributeSubject := sourceindex.Subject{
		Kind: "semantic-attribute", ID: "organization.transfer_request.allocation",
	}
	attributeView, err := InspectionFromLoad(loaded, index, attributeSubject)
	if err != nil {
		t.Fatal(err)
	}
	var attribute semantic.Attribute
	if err := json.Unmarshal(attributeView.Canonical, &attribute); err != nil || attribute.TypeRef != "std.text" {
		t.Fatalf("Data table type mutation was not read: %+v err=%v", attribute, err)
	}
	ruleSubject := sourceindex.Subject{
		Kind: "semantic-validation", ID: "organization.transfer_request.allocation_positive",
	}
	ruleView, err := InspectionFromLoad(loaded, index, ruleSubject)
	if err != nil {
		t.Fatal(err)
	}
	var rule semantic.Validation
	if err := json.Unmarshal(ruleView.Canonical, &rule); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before.Inspection.Canonical, ruleView.Canonical) ||
		before.Inspection.WorldFingerprint == ruleView.WorldFingerprint ||
		before.Inspection.SourceIndexFingerprint == ruleView.SourceIndexFingerprint {
		t.Fatalf("same authored Rule must keep its meaning but lose stale Data/Source binding: before=%+v after=%+v", before.Inspection, ruleView)
	}
	generated, err := validationgen.GeneratePositive(rule, attribute.TypeRef, "main")
	if err == nil || !strings.Contains(err.Error(), "requires resolved std.decimal") || len(generated) != 0 {
		t.Fatalf("unsupported text input produced generated Go: generated=%q err=%v", generated, err)
	}
}
