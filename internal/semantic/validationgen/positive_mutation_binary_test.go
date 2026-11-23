package validationgen_test

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"tei/internal/codegen/goemit"
	"tei/internal/codegen/goemit/goemittest"
	"tei/internal/core"
	"tei/internal/semantic"
	"tei/internal/sourceindex"
	"tei/internal/workspace"
)

// A source edit must reach executable generated code, not only the semantic
// evaluator or the text returned by GeneratePositive. This is a direct Go
// toolchain test, NOT the selected TEI Builder/Plugin/CLI product route.
func TestSelectedRuleAttributeEditChangesCompiledGeneratedGo(t *testing.T) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("test source file location unavailable")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", "..", ".."))
	fixture := filepath.Join(repoRoot, "internal", "workspace", "testdata", "tei-md-rule")
	workspaceRoot := t.TempDir()
	const ruleFile = "semantic/sales/order-line-rules.tei.md"
	const orderFile = "semantic/sales/order.tei"
	for _, name := range []string{"tei.workspace.yaml", orderFile, ruleFile} {
		src, err := os.ReadFile(filepath.Join(fixture, filepath.FromSlash(name)))
		if err != nil {
			t.Fatal(err)
		}
		dst := filepath.Join(workspaceRoot, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(dst), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(dst, src, 0600); err != nil {
			t.Fatal(err)
		}
	}

	// Both phases have exactly the same declared Data model. Introducing
	// another decimal Attribute here avoids faking type resolution for a
	// non-decimal target; only the rule's reference changes between phases.
	orderPath := filepath.Join(workspaceRoot, filepath.FromSlash(orderFile))
	orderBytes, err := os.ReadFile(orderPath)
	if err != nil {
		t.Fatal(err)
	}
	const quantityDeclaration = "    attribute quantity type std.decimal"
	if bytes.Count(orderBytes, []byte(quantityDeclaration)) != 1 {
		t.Fatal("fixture has no unique decimal quantity declaration")
	}
	orderWithSecondDecimal := bytes.Replace(orderBytes, []byte(quantityDeclaration),
		[]byte(quantityDeclaration+"\n    attribute secondaryQuantity type std.decimal"), 1)
	if err := os.WriteFile(orderPath, orderWithSecondDecimal, 0600); err != nil {
		t.Fatal(err)
	}

	const ruleID = "sales.order_line.quantity_positive"
	ruleSourcePath := filepath.Join(workspaceRoot, filepath.FromSlash(ruleFile))
	selectAndGenerate := func() (goemit.Output, sourceindex.Index) {
		t.Helper()
		loaded, err := workspace.Load(workspaceRoot)
		if err != nil {
			t.Fatalf("product Workspace.Load failed: %v", err)
		}
		if len(loaded.AuthoringSources) != 1 ||
			loaded.AuthoringSources[0].Language.LanguageID != "tei.rule-authoring" ||
			loaded.AuthoringSources[0].Language.ContractVersion != "v0" ||
			loaded.AuthoringSources[0].Definitions != 2 {
			t.Fatalf("actual selected reader changed: %+v", loaded.AuthoringSources)
		}
		var rule semantic.Validation
		var attributeType core.CanonicalID
		for _, ns := range loaded.World.Semantics {
			for _, candidate := range ns.Validations {
				if string(candidate.ID) == ruleID {
					if rule.ID != "" {
						t.Fatal("multiple selected rules have the same identity")
					}
					rule = candidate
				}
			}
		}
		for _, ns := range loaded.World.Semantics {
			for _, data := range ns.Data {
				for _, attribute := range data.Attributes {
					if attribute.ID == rule.Attribute {
						attributeType = attribute.TypeRef
					}
				}
			}
		}
		if rule.ID != ruleID || rule.Kind != semantic.ValidationPositive ||
			attributeType != "std.decimal" {
			t.Fatalf("selected meaning or product attribute type is not qualified: %+v type=%q", rule, attributeType)
		}
		index, err := workspace.BuildSourceIndexFromLoad(workspaceRoot, loaded)
		if err != nil {
			t.Fatal(err)
		}
		refs := index.ReferencesFor(sourceindex.Subject{Kind: "semantic-validation", ID: ruleID})
		source, err := os.ReadFile(ruleSourcePath)
		if err != nil {
			t.Fatal(err)
		}
		if len(refs) != 1 || refs[0].ArtifactPath != ruleFile ||
			refs[0].Role != sourceindex.RoleDefinition {
			t.Fatalf("generated rule has no exact product source: %+v", refs)
		}
		at := refs[0].Range
		if err := at.Validate(source); err != nil {
			t.Fatal(err)
		}
		if string(source[at.Start.Offset:at.End.Offset]) != ruleID {
			t.Fatalf("source range does not identify selected rule: %+v", at)
		}
		out := goemit.Emit(goemit.Input{PackageName: "main", Rules: []goemit.Rule{{Validation: rule, AttributeType: attributeType}}})
		if err := out.Err(); err != nil {
			t.Fatal(err)
		}
		generated := out.File(out.Trace[0].File)
		if !bytes.Contains(generated, []byte(".Sign() > 0")) ||
			bytes.Contains(generated, []byte("validationeval")) {
			t.Fatalf("generated code contains no owned positive decision:\n%s", generated)
		}
		return out, index
	}

	// These are *independent* expected results for the two generated binaries.
	// The record is identical in both: 2 for quantity, 0 for secondary
	// quantity. main.go and cases.go are identical too; binding.go names the
	// attribute and check function from Emit's Trace, so the harness reports
	// the attribute the Trace binds. The generated Go reads no attribute
	// itself; which value reaches it is decided by that Trace-driven harness.
	record := []goemittest.Case{{Name: "fixed", Row: map[string]goemittest.Cell{
		"sales.order_line.quantity":           {Text: "2"},
		"sales.order_line.secondary_quantity": {Text: "0"},
	}}}
	buildAndExecute := func(name string, out goemit.Output, want string) {
		t.Helper()
		// A module of its own, outside the repository: the generated Go and
		// the harness need nothing but the standard library.
		buildRoot := goemittest.WriteModule(t, goemittest.Program(t, out, record))
		binaryPath := filepath.Join(buildRoot, name)
		if runtime.GOOS == "windows" {
			binaryPath += ".exe"
		}
		goemittest.RunGo(t, buildRoot, "build", "-trimpath", "-o", binaryPath, ".")
		run := exec.Command(binaryPath)
		run.Dir = buildRoot
		output, err := run.CombinedOutput()
		if err != nil {
			t.Fatalf("generated %s binary failed: %v\n%s", name, err, output)
		}
		if string(output) != want {
			t.Fatalf("generated %s consumer output=%q, want %q", name, output, want)
		}
	}

	beforeOut, beforeIndex := selectAndGenerate()
	buildAndExecute("before", beforeOut, "fixed:satisfied:1:sales.order_line.quantity\n")

	ruleBytes, err := os.ReadFile(ruleSourcePath)
	if err != nil {
		t.Fatal(err)
	}
	const oldPredicate = "sales.order_line.quantity is positive"
	const newPredicate = "sales.order_line.secondary_quantity is positive"
	if bytes.Count(ruleBytes, []byte(oldPredicate)) != 1 {
		t.Fatal("fixture no longer contains the original predicate exactly once")
	}
	edited := bytes.Replace(ruleBytes, []byte(oldPredicate), []byte(newPredicate), 1)
	if err := os.WriteFile(ruleSourcePath, edited, 0600); err != nil {
		t.Fatal(err)
	}
	// The old load's Source Index cannot certify even a same-ID new rule.
	readCurrent := func(path string) ([]byte, error) {
		return os.ReadFile(filepath.Join(workspaceRoot, filepath.FromSlash(path)))
	}
	if err := beforeIndex.Validate(readCurrent); err == nil ||
		!strings.Contains(err.Error(), "fingerprint is stale") {
		t.Fatalf("old Source Index survived the one-factor rule change: %v", err)
	}

	afterOut, afterIndex := selectAndGenerate()
	// The generated package itself must change, not only the harness: compare
	// the files Emit returned, which the harness never writes. Comparing the
	// compiled binaries would pass on binding.go alone, because the harness
	// writes the bound attribute name into it.
	if samePackage(goemittest.Files(beforeOut), goemittest.Files(afterOut)) {
		t.Fatal("changed product meaning did not change the generated package bytes")
	}
	afterGo := afterOut.File(afterOut.Trace[0].File)
	if afterOut.Trace[0].Attribute != "sales.order_line.secondary_quantity" ||
		afterOut.Trace[0].Check != "CheckSecondaryQuantityIsPositive" ||
		!bytes.Contains(afterGo, []byte("func CheckSecondaryQuantityIsPositive(")) {
		t.Fatalf("generated Go is not bound to the edited attribute: %+v", afterOut.Trace[0])
	}
	// Same record, same main.go and cases.go. The different result comes from
	// the binding, not from different compiled logic: binding.go, written from
	// the Trace, makes the harness read secondary_quantity (0) instead of
	// quantity (2), and both check functions decide "> 0" the same way. What
	// the generated package adds is the renamed check function: binding.go
	// calls CheckSecondaryQuantityIsPositive, so the binary builds only when
	// the regenerated package declares the function the Trace names.
	buildAndExecute("after", afterOut, "fixed:not-satisfied:1:sales.order_line.secondary_quantity\n")
	if err := afterIndex.Validate(readCurrent); err != nil {
		t.Fatalf("new Source Index rejected actual edited bytes: %v", err)
	}
}

// samePackage reports whether two emitted packages have the same paths and
// bytes.
func samePackage(a, b map[string][]byte) bool {
	if len(a) != len(b) {
		return false
	}
	for path, content := range a {
		if other, ok := b[path]; !ok || !bytes.Equal(content, other) {
			return false
		}
	}
	return true
}
