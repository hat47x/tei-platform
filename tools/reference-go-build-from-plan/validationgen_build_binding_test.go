package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"tei/internal/buildintent"
	"tei/internal/codegen/goemit"
	"tei/internal/codegen/goemit/goemittest"
	"tei/internal/core"
	"tei/internal/plugin"
	"tei/internal/semantic"
	"tei/internal/sourceindex"
	"tei/internal/workspace"
)

// This tests the pre-existing reference builder, not a second Go Builder.
// The source is generated from a real product-selected Rule, then independently
// re-read, staged and compiled by reference-go-build-from-plan. A matching
// digest and BuildArtifactLock are evidence for this local build attempt only.
func TestSelectedRuleGoIsActuallyConsumedByBuildPlanBuilder(t *testing.T) {
	if _, err := exec.LookPath("go"); err != nil {
		t.Skipf("Go toolchain unavailable: %v", err)
	}
	root := repoRoot(t)
	fixture := filepath.Join(root, "internal", "workspace", "testdata", "tei-md-rule")
	loaded, err := workspace.Load(fixture)
	if err != nil {
		t.Fatalf("real selected Workspace.Load: %v", err)
	}
	if len(loaded.AuthoringSources) != 1 ||
		loaded.AuthoringSources[0].Language.LanguageID != "tei.rule-authoring" ||
		loaded.AuthoringSources[0].Language.ContractVersion != "v0" {
		t.Fatalf("real Rule authoring language was not selected: %+v", loaded.AuthoringSources)
	}

	const ruleID = "sales.order_line.quantity_positive"
	const attributeID = "sales.order_line.quantity"
	const sourcePath = "semantic/sales/order-line-rules.tei.md"
	var rule semantic.Validation
	var attributeType core.CanonicalID
	for _, ns := range loaded.World.Semantics {
		for _, candidate := range ns.Validations {
			if string(candidate.ID) == ruleID {
				if rule.ID != "" {
					t.Fatal("duplicate canonical Rule")
				}
				rule = candidate
			}
		}
		for _, data := range ns.Data {
			for _, attribute := range data.Attributes {
				if string(attribute.ID) == attributeID {
					attributeType = attribute.TypeRef
				}
			}
		}
	}
	if rule.ID != ruleID || rule.Kind != semantic.ValidationPositive ||
		rule.Attribute != attributeID || attributeType != "std.decimal" {
		t.Fatalf("wrong resolved Rule or decimal type: %+v, %s", rule, attributeType)
	}
	idx, err := workspace.BuildSourceIndexFromLoad(fixture, loaded)
	if err != nil {
		t.Fatalf("Source Index for same product Load: %v", err)
	}
	original, err := os.ReadFile(filepath.Join(fixture, filepath.FromSlash(sourcePath)))
	if err != nil {
		t.Fatal(err)
	}
	refs := idx.ReferencesFor(sourceindex.Subject{Kind: "semantic-validation", ID: ruleID})
	if len(refs) != 1 || refs[0].ArtifactPath != sourcePath ||
		refs[0].Role != sourceindex.RoleDefinition {
		t.Fatalf("Rule has no unique selected original Source: %+v", refs)
	}
	start, end := refs[0].Range.Start.Offset, refs[0].Range.End.Offset
	if start < 0 || start >= end || end > len(original) || string(original[start:end]) != ruleID {
		t.Fatalf("Rule Source anchor is not the authored Rule: %+v", refs[0])
	}

	out := goemit.Emit(goemit.Input{PackageName: "main", Rules: []goemit.Rule{{Validation: rule, AttributeType: attributeType}}})
	if err := out.Err(); err != nil {
		t.Fatalf("lowering the actual selected Rule failed: %v", err)
	}
	const ruleFile = "quantity_is_positive.go"
	generated := out.File(ruleFile)
	if !bytes.Contains(generated, []byte("return quantity.Sign() > 0")) ||
		bytes.Contains(generated, []byte("validationeval")) {
		t.Fatal("generated Go must implement its own comparison and branch")
	}
	// The harness reads the attribute the Trace binds to the check function.
	// A null value and a value that was not fetched both reach the check
	// function as nil and are undecided.
	const attr = "sales.order_line.quantity"
	program := goemittest.Program(t, out, []goemittest.Case{
		{Name: "positive", Row: map[string]goemittest.Cell{attr: {Text: "2.5"}}},
		{Name: "zero", Row: map[string]goemittest.Cell{attr: {Text: "0"}}},
		{Name: "negative", Row: map[string]goemittest.Cell{attr: {Text: "-1"}}},
		{Name: "null", Row: map[string]goemittest.Cell{attr: {Null: true}}},
		{Name: "missing", Row: map[string]goemittest.Cell{}},
	})
	if imports, err := goemittest.TEIImports(program); err != nil || len(imports) != 0 {
		t.Fatalf("generated program depends on TEI packages: %v %v", imports, err)
	}

	packageDir, err := os.MkdirTemp(root, "tei-rule-build-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(packageDir) })
	relDir, err := filepath.Rel(root, packageDir)
	if err != nil {
		t.Fatal(err)
	}
	relDir = filepath.ToSlash(relDir)
	write := func(name string, data []byte) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(packageDir, name), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	for name, data := range program {
		write(name, data)
	}

	// The generated package and its harness use only the standard library,
	// so the plan names no TEI package. Include the Go module itself so
	// -isolate cannot silently read the working tree's module context during
	// compilation.
	plan := planFor(t, root, relDir, "./"+relDir, "runtime-artifact")
	plan.Target = "host/" + runtime.GOOS + "/" + runtime.GOARCH
	resolver := buildintent.DirSourceResolver{Root: root}
	for _, source := range []struct {
		kind, id, path string
	}{
		{"go-module", "gomod", "go.mod"},
		{"go-module", "gosum", "go.sum"},
	} {
		resolved, err := resolver.Resolve(buildintent.IntentSource{
			Kind: source.kind, ID: source.id, Ref: source.path,
		})
		if err != nil {
			t.Fatalf("resolve planned source %s: %v", source.path, err)
		}
		plan.Sources = append(plan.Sources, plugin.BuildSource{
			Kind: source.kind, ID: source.id, Locator: source.path, Digest: resolved.Source.Digest,
		})
	}
	if err := plan.Validate(); err != nil {
		t.Fatalf("exact BuildPlan invalid: %v", err)
	}
	dir := t.TempDir()
	planPath := writePlan(t, dir, plan)

	// A source edit after plan resolution must be rejected before compilation.
	write(ruleFile, append(append([]byte(nil), generated...), []byte("\n// stale planned input\n")...))
	if _, err := runTool(t,
		"-plan", planPath, "-repo", root,
		"-artifact-out", filepath.Join(dir, "should-not-build"),
		"-state", filepath.Join(dir, "rejected-state"), "-isolate",
	); err == nil || !strings.Contains(err.Error(), "the plan does not describe") {
		t.Fatalf("stale generated-Go input was accepted or misreported: %v", err)
	}
	write(ruleFile, generated)

	artifact := filepath.Join(dir, "generated-rule")
	if runtime.GOOS == "windows" {
		artifact += ".exe"
	}
	rep, err := runTool(t,
		"-plan", planPath, "-repo", root,
		"-artifact-out", artifact,
		"-state", filepath.Join(dir, "build-state"), "-isolate",
	)
	if err != nil {
		t.Fatalf("existing BuildPlan builder rejected the generated Go: %v", err)
	}
	if !rep.BuildOK || rep.IsolatedTree == nil ||
		rep.DependencyCoverage == nil || !rep.DependencyCoverage.Checked ||
		len(rep.DependencyCoverage.Uncovered) != 0 || rep.BuildSource != buildSourcePlanOnly {
		t.Fatalf("builder did not compile the covered staged source tree: %+v", rep)
	}
	wantFP, err := plugin.BuildPlanFingerprint(plan)
	if err != nil {
		t.Fatal(err)
	}
	if rep.PlanFingerprint != wantFP || len(rep.Sources) != len(plan.Sources) ||
		rep.Lock == nil || !rep.Lock.Validated ||
		rep.Lock.PlanFingerprint != wantFP || rep.Lock.ArtifactDigest != rep.ArtifactDigest {
		t.Fatalf("build plan, actual sources, and artifact lock are not bound: %+v", rep)
	}
	for _, source := range rep.Sources {
		if !source.Matched {
			t.Fatalf("actual Builder source digest did not match plan: %+v", source)
		}
	}
	output, err := exec.Command(artifact).CombinedOutput()
	if err != nil {
		t.Fatalf("actual Builder-produced Go executable failed: %v\n%s", err, output)
	}
	const expected = "positive:satisfied:1:" + attr + "\nzero:not-satisfied:1:" + attr +
		"\nnegative:not-satisfied:1:" + attr + "\nnull:undecided:1:" + attr + "\nmissing:undecided:1:" + attr + "\n"
	if string(output) != expected {
		t.Fatalf("Builder-produced binary had incorrect decision/read behavior:\n%s\nexpected:\n%s", output, expected)
	}
}
