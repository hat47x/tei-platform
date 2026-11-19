package workspace

import (
	"bytes"
	"math/big"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/semantic"
	"tei/internal/semantic/validationeval"
	"tei/internal/sourceindex"
)

// A named input is currently metadata for the single input already supplied
// by the rule subject. This product test does not admit extra argument values,
// a multi-input evaluator, or an implicit choice of Plugin implementation.
func TestNamedUnaryOperatorPreservesSubjectMeaningAndSourceRevision(t *testing.T) {
	const operatorPath = "semantic/sales/zz-operators.tei.md"
	const operatorID = "sales.non_negative"
	const ruleID = "sales.order_line.discount_non_negative"

	root := teiMDOperatorWorkspace(t)
	rewriteTEIMDTestFile(t, root, operatorPath,
		"operator sales.non_negative: takes number",
		"operator sales.non_negative: takes amount: number")

	load := func() (Result, sourceindex.Index) {
		t.Helper()
		loaded, err := Load(root)
		if err != nil {
			t.Fatalf("selected product Reader: %v", err)
		}
		if codes := diagnosticCodes(loaded); len(codes) != 0 {
			t.Fatalf("named unary contract failed semantic validation: %v", codes)
		}
		index, err := BuildSourceIndexFromLoad(root, loaded)
		if err != nil {
			t.Fatalf("Load-time Source Index: %v", err)
		}
		return loaded, index
	}
	loaded, index := load()
	operators := operatorsOf(loaded)
	want := semantic.Operator{
		ID: operatorID, Symbol: "non_negative",
		Input: semantic.OperatorInputNumber, InputName: "amount",
	}
	if len(operators) != 1 || !reflect.DeepEqual(operators[0], want) {
		t.Fatalf("named input lost at semantic boundary: got %+v, want %+v", operators, want)
	}
	rule := loadedValidation(t, loaded, ruleID)
	if rule.Kind != semantic.ValidationOperator || rule.Attribute != "sales.order_line.discount" ||
		rule.Operator != operatorID {
		t.Fatalf("the named declaration changed the rule's subject/read: %+v", rule)
	}

	raw := readTEIMDTestFile(t, root, operatorPath)
	refs := index.ReferencesFor(sourceindex.Subject{Kind: "semantic-operator", ID: operatorID})
	if len(refs) != 1 || refs[0].ArtifactPath != operatorPath ||
		refs[0].Role != sourceindex.RoleDefinition ||
		refs[0].Range.Start.Offset != bytes.Index(raw, []byte(operatorID)) ||
		refs[0].Range.End.Offset != refs[0].Range.Start.Offset+len(operatorID) {
		t.Fatalf("named declaration lost its original Source anchor: %+v", refs)
	}
	if err := refs[0].Range.Validate(raw); err != nil {
		t.Fatalf("invalid authored byte range: %v", err)
	}

	// The same typed rule still invokes the selected one-number implementation.
	// The contract's input name does not become an extra Source read.
	calls := 0
	binding := validationeval.Operators{
		Declared: operators,
		Implementations: []validationeval.OperatorImplementation{{
			Operator: operatorID, Input: semantic.OperatorInputNumber,
			Number: func(value *big.Rat) (bool, error) {
				calls++
				return value.Sign() >= 0, nil
			},
		}},
	}
	for _, tc := range []struct {
		value int64
		want  validationeval.Result
	}{
		{value: 3, want: validationeval.ResultSatisfied},
		{value: 0, want: validationeval.ResultSatisfied},
		{value: -1, want: validationeval.ResultNotSatisfied},
	} {
		reader := &fixedNumberSource{value: big.NewRat(tc.value, 1)}
		result, err := validationeval.EvaluateWithOperators(rule, reader, binding)
		if err != nil || result != tc.want ||
			!reflect.DeepEqual(reader.calls, []string{"Number(sales.order_line.discount)"}) {
			t.Fatalf("%d: result=%q err=%v logical reads=%v", tc.value, result, err, reader.calls)
		}
	}
	if calls != 3 {
		t.Fatalf("selected implementation invoked %d times, want three", calls)
	}

	readCurrent := func(rel string) ([]byte, error) {
		return os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	}
	if err := index.Validate(readCurrent); err != nil {
		t.Fatalf("original Source Index invalid: %v", err)
	}
	// Explanatory prose is not an Operator declaration. It may preserve
	// Canonical meaning but must invalidate the previous raw Source revision.
	revised := append(append([]byte(nil), raw...), []byte("\n補足：名前は入力の説明であり、二つ目の値を読みません。\n")...)
	writeTEIMDTestFile(t, root, operatorPath, string(revised))
	if err := index.Validate(readCurrent); err == nil {
		t.Fatal("stale Source Index was accepted after a prose-only edit")
	}
	reloaded, revisedIndex := load()
	if !reflect.DeepEqual(loaded.World.Semantics, reloaded.World.Semantics) {
		t.Fatal("prose-only edit unexpectedly changed typed Operator/Rule meaning")
	}
	revisedRefs := revisedIndex.ReferencesFor(sourceindex.Subject{Kind: "semantic-operator", ID: operatorID})
	if !reflect.DeepEqual(refs, revisedRefs) {
		t.Fatalf("trailing prose moved the Operator definition: before=%+v after=%+v", refs, revisedRefs)
	}
	if err := revisedIndex.Validate(readCurrent); err != nil {
		t.Fatalf("fresh Source Index invalid: %v", err)
	}
	findFingerprint := func(idx sourceindex.Index) string {
		for _, artifact := range idx.Artifacts {
			if artifact.Path == operatorPath {
				return artifact.Fingerprint
			}
		}
		return ""
	}
	if findFingerprint(index) != sourceindex.Fingerprint(raw) ||
		findFingerprint(revisedIndex) != sourceindex.Fingerprint(revised) ||
		findFingerprint(index) == findFingerprint(revisedIndex) {
		t.Fatal("a prose-only edit was not represented by a distinct Source fingerprint")
	}
	// The declaration's label is meaningful, unlike descriptive prose.
	// The change must be visible in the existing typed Operator contract.
	rewriteTEIMDTestFile(t, root, operatorPath,
		"takes amount: number", "takes quantity: number")
	renamed, _ := load()
	declared := operatorsOf(renamed)
	if len(declared) != 1 || declared[0].InputName != "quantity" ||
		reflect.DeepEqual(declared, operators) ||
		!reflect.DeepEqual(validationsOf(loaded, core.CanonicalID("sales")),
			validationsOf(renamed, core.CanonicalID("sales"))) {
		t.Fatalf("input-name edit was lost or incorrectly changed the rule: old=%+v new=%+v", operators, declared)
	}
}

// A rule in a different authoring document has no local declaration input
// index. It must fail closed on any assigned input value rather than promise
// the more specific subject-rebinding diagnostic that a co-located
// declaration can provide.
func TestNamedUnaryOperatorRefusesCrossDocumentValueAtProductLoad(t *testing.T) {
	const operatorPath = "semantic/sales/zz-operators.tei.md"
	const rulePath = "semantic/sales/discount-rules.tei.md"
	root := teiMDOperatorWorkspace(t)
	rewriteTEIMDTestFile(t, root, operatorPath,
		"operator sales.non_negative: takes number",
		"operator sales.non_negative: takes amount: number")
	rewriteTEIMDTestFile(t, root, rulePath,
		"sales.order_line.discount is sales.non_negative",
		"sales.order_line.discount is sales.non_negative amount: 0")
	_, err := Load(root)
	if err == nil || !strings.Contains(err.Error(), "the meaning has no place for an operator input value yet") ||
		!strings.Contains(err.Error(), "sales.order_line.discount_non_negative") ||
		!strings.Contains(err.Error(), "discount-rules.tei.md:") {
		t.Fatalf("a cross-document input value escaped product fail-closed diagnostics: %v", err)
	}
}

// Moving a selected Operator document without changing its bytes must retain
// the typed Rule and Operator meaning, but cannot retain the old physical
// Source locator or ModelRevision. In particular, the byte fingerprint alone
// is not a sufficient identity for a selected Source.
func TestNamedUnaryOperatorMoveRetainsMeaningButChangesSourceRevision(t *testing.T) {
	const oldPath = "semantic/sales/zz-operators.tei.md"
	const newPath = "semantic/sales/zy-operators.tei.md"
	const operatorID = "sales.non_negative"

	root := teiMDOperatorWorkspace(t)
	rewriteTEIMDTestFile(t, root, oldPath,
		"operator sales.non_negative: takes number",
		"operator sales.non_negative: takes amount: number")

	load := func() (Result, sourceindex.Index) {
		t.Helper()
		loaded, err := Load(root)
		if err != nil {
			t.Fatal(err)
		}
		if codes := diagnosticCodes(loaded); len(codes) != 0 {
			t.Fatalf("selected Operator/Rule is invalid: %v", codes)
		}
		index, err := BuildSourceIndexFromLoad(root, loaded)
		if err != nil {
			t.Fatal(err)
		}
		return loaded, index
	}
	readCurrent := func(rel string) ([]byte, error) {
		return os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	}
	before, oldIndex := load()
	raw := readTEIMDTestFile(t, root, oldPath)
	if err := oldIndex.Validate(readCurrent); err != nil {
		t.Fatalf("original Source Index: %v", err)
	}
	subject := sourceindex.Subject{Kind: "semantic-operator", ID: operatorID}
	oldRefs := oldIndex.ReferencesFor(subject)
	if len(oldRefs) != 1 || oldRefs[0].ArtifactPath != oldPath ||
		oldRefs[0].Range.Start.Offset != bytes.Index(raw, []byte(operatorID)) {
		t.Fatalf("original Operator source is not independently located: %+v", oldRefs)
	}

	if err := os.Rename(filepath.Join(root, filepath.FromSlash(oldPath)),
		filepath.Join(root, filepath.FromSlash(newPath))); err != nil {
		t.Fatal(err)
	}
	after, newIndex := load()
	if !reflect.DeepEqual(before.World.Semantics, after.World.Semantics) {
		t.Fatal("a Source-only move changed the typed Rule or Operator meaning")
	}
	moved := readTEIMDTestFile(t, root, newPath)
	if !bytes.Equal(raw, moved) {
		t.Fatal("moving the document changed its authored bytes")
	}
	newRefs := newIndex.ReferencesFor(subject)
	if len(newRefs) != 1 || newRefs[0].ArtifactPath != newPath ||
		newRefs[0].Range != oldRefs[0].Range ||
		newRefs[0].Role != sourceindex.RoleDefinition {
		t.Fatalf("Operator Source range was not retained at the new location: old=%+v new=%+v", oldRefs, newRefs)
	}
	if err := newRefs[0].Range.Validate(moved); err != nil {
		t.Fatalf("relocated range is not valid: %v", err)
	}
	if sourceindex.ModelRevision(oldIndex) == sourceindex.ModelRevision(newIndex) {
		t.Fatal("the selected Source path changed but ModelRevision did not")
	}
	if err := oldIndex.Validate(readCurrent); err == nil {
		t.Fatal("a stale Source Index was accepted after its selected document moved")
	}
	if err := newIndex.Validate(readCurrent); err != nil {
		t.Fatalf("the freshly loaded Source Index is invalid: %v", err)
	}
}
