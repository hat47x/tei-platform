package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
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
	"tei/internal/semantic"
	"tei/internal/semantic/validationeval"
	"tei/internal/sourceindex"
	"tei/internal/workspace"
)

// The selected product Rule is lowered to Go, then consumed through the
// normal tei self-build CLI (not only through a directly invoked reference
// builder). This is an R2 command-build witness, not a published Function
// or a Runtime Provider invocation, and does not grant Owner acceptance.
func TestSelectedRuleGoThroughRealCLISelfBuild(t *testing.T) {
	if testing.Short() {
		t.Skip("real CLI and isolated reproducible Go builds are required")
	}
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("existing test catalog pins the host/linux/amd64 target")
	}
	goExe, err := exec.LookPath("go")
	if err != nil {
		t.Skipf("Go toolchain unavailable: %v", err)
	}
	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	fixture := filepath.Join(repoRoot, "internal", "workspace", "testdata", "tei-md-rule")
	loaded, err := workspace.Load(fixture)
	if err != nil {
		t.Fatalf("select the real Rule Markdown reader: %v", err)
	}
	if len(loaded.AuthoringSources) != 1 ||
		loaded.AuthoringSources[0].Language.LanguageID != "tei.rule-authoring" ||
		loaded.AuthoringSources[0].Language.ContractVersion != "v0" {
		t.Fatalf("the Rule Reader was not explicitly selected: %+v", loaded.AuthoringSources)
	}
	var selected semantic.Validation
	var attributeType core.CanonicalID
	for _, ns := range loaded.World.Semantics {
		for _, rule := range ns.Validations {
			if rule.ID == "sales.order_line.quantity_positive" {
				if selected.ID != "" {
					t.Fatal("duplicate selected Rule")
				}
				selected = rule
			}
		}
		for _, d := range ns.Data {
			for _, attr := range d.Attributes {
				if attr.ID == "sales.order_line.quantity" {
					attributeType = attr.TypeRef
				}
			}
		}
	}
	if selected.Kind != semantic.ValidationPositive ||
		selected.Attribute != "sales.order_line.quantity" ||
		attributeType != "std.decimal" {
		t.Fatalf("wrong selected Rule or resolved type: %+v / %s", selected, attributeType)
	}
	// Recheck the same-load Source Index against the original Markdown bytes.
	// A late edit cannot silently become the provenance of the generated Go.
	const sourcePath = "semantic/sales/order-line-rules.tei.md"
	index, err := workspace.BuildSourceIndexFromLoad(fixture, loaded)
	if err != nil {
		t.Fatalf("selected Rule Source Index: %v", err)
	}
	original, err := os.ReadFile(filepath.Join(fixture, filepath.FromSlash(sourcePath)))
	if err != nil {
		t.Fatal(err)
	}
	if err := index.Validate(func(path string) ([]byte, error) {
		return os.ReadFile(filepath.Join(fixture, filepath.FromSlash(path)))
	}); err != nil {
		t.Fatalf("source changed after selected Workspace.Load: %v", err)
	}
	refs := index.ReferencesFor(sourceindex.Subject{Kind: "semantic-validation", ID: string(selected.ID)})
	if len(refs) != 1 || refs[0].ArtifactPath != sourcePath || refs[0].Role != sourceindex.RoleDefinition {
		t.Fatalf("selected Rule does not have exactly one original Source location: %+v", refs)
	}
	if err := refs[0].Range.Validate(original); err != nil {
		t.Fatalf("selected Rule Source range: %v", err)
	}
	if got := string(original[refs[0].Range.Start.Offset:refs[0].Range.End.Offset]); got != string(selected.ID) {
		t.Fatalf("Source Index bound %q instead of the selected Rule %s", got, selected.ID)
	}

	emitted := goemit.Emit(goemit.Input{PackageName: "main", Rules: []goemit.Rule{{Validation: selected, AttributeType: attributeType}}})
	if err := emitted.Err(); err != nil {
		t.Fatalf("generate positive Rule Go: %v", err)
	}
	const ruleFile = "quantity_is_positive.go"
	generated := emitted.File(ruleFile)
	if !bytes.Contains(generated, []byte("return quantity.Sign() > 0")) ||
		bytes.Contains(generated, []byte("validationeval")) {
		t.Fatal("generated Go must perform the selected decision itself")
	}
	// The harness reads the attribute named by the generated doc comment and
	// passes its value to the generated check function. It and the generated
	// files use only the standard library.
	program := goemittest.Program(t, emitted, generatedRuleCases)
	if imports, err := goemittest.TEIImports(program); err != nil || len(imports) != 0 {
		t.Fatalf("generated program depends on TEI packages: %v %v", imports, err)
	}

	// This package is inside the TEI module because the Build Intent and its
	// plan locate sources relative to the repository. Remove it after the
	// attempted real build.
	pkgDir, err := os.MkdirTemp(repoRoot, "tei-rule-self-build-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(pkgDir) })
	relative, err := filepath.Rel(repoRoot, pkgDir)
	if err != nil {
		t.Fatal(err)
	}
	entry := "./" + filepath.ToSlash(relative)
	program["provenance.go"] = []byte(generatedRuleProvenance)
	program["source.tei.md"] = original
	for name, data := range program {
		if err := os.WriteFile(filepath.Join(pkgDir, name), data, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	// Check the exact generated bytes at the point where the real Builder will
	// read the package. A different but compilable Go file is not the selected
	// Rule output, even if it produces the same answers in this test. Every
	// emitted file is checked; the rule file is the one mutated below.
	for _, file := range emitted.Files {
		if err := requireSelectedGeneratedGo(filepath.Join(pkgDir, file.Path), file.Content); err != nil {
			t.Fatalf("selected generated Go %s was not staged: %v", file.Path, err)
		}
	}
	generatedPath := filepath.Join(pkgDir, ruleFile)
	other := append(append([]byte(nil), generated...), []byte("\n// changed implementation byte sequence\n")...)
	if err := os.WriteFile(generatedPath, other, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := requireSelectedGeneratedGo(generatedPath, generated); err == nil {
		t.Fatal("a staged Go file different from the selected Generator output was accepted")
	}
	if err := os.WriteFile(generatedPath, generated, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := requireSelectedGeneratedGo(generatedPath, generated); err != nil {
		t.Fatalf("restored generated Go is not selected: %v", err)
	}

	cli := buildToolBinary(t, ".", "tei")
	builder := buildToolBinary(t, filepath.Join("..", "..", "tools", "reference-go-build-from-plan"), "go-build-from-plan")
	root := t.TempDir()
	catalogPath, catalogDigest := authorGoCommandCatalog(t, root, builder)
	workspace := outerWorkspace(t, root)
	intentPath, memberCount := writeFullChainDeclaration(t, repoRoot, entry, root)
	raw, err := os.ReadFile(intentPath)
	if err != nil {
		t.Fatal(err)
	}
	intent, err := buildintent.ParseIntent(raw)
	if err != nil {
		t.Fatal(err)
	}
	// The first alphabetical dependency may be core/identity, not the
	// generated package. The Build Intent definition must be the actual entry.
	found := false
	for _, source := range intent.Sources {
		if source.ID == intent.Subject.DefinitionSource {
			found = source.Kind == "go-package" && source.Ref == strings.TrimPrefix(entry, "./")
		}
	}
	if !found {
		t.Fatalf("definitionSource is not the generated Rule's actual Go package: %+v", intent.Subject)
	}
	catalogRaw, err := os.ReadFile(catalogPath)
	if err != nil {
		t.Fatal(err)
	}
	catalog, err := buildintent.ParseBuildCatalog(catalogRaw)
	if err != nil {
		t.Fatal(err)
	}
	receipt, err := buildintent.Resolve(intent, buildintent.NewBuildContext(repoRoot, catalog))
	if err != nil {
		t.Fatalf("actual product Build Intent cannot resolve: %v", err)
	}
	// A prose-only byte change in the packaged original must change the exact
	// BuildPlan even when the generated Go has not changed. The final build
	// uses the restored original; its Source Index qualification is separate.
	packagedSource := filepath.Join(pkgDir, "source.tei.md")
	modified := append(append([]byte(nil), original...), []byte("\nAdditional prose only.\n")...)
	if err := os.WriteFile(packagedSource, modified, 0o600); err != nil {
		t.Fatal(err)
	}
	changed, err := buildintent.Resolve(intent, buildintent.NewBuildContext(repoRoot, catalog))
	if err != nil {
		t.Fatalf("resolve changed packaged Source: %v", err)
	}
	if changed.PlanFingerprint == receipt.PlanFingerprint ||
		changed.SourceClosureFingerprint == receipt.SourceClosureFingerprint {
		t.Fatal("a changed original Markdown byte sequence reused the previous BuildPlan")
	}
	if err := os.WriteFile(packagedSource, original, 0o600); err != nil {
		t.Fatal(err)
	}

	planRaw, err := json.Marshal(receipt.Plan)
	if err != nil {
		t.Fatal(err)
	}
	planPath := filepath.Join(root, "plan.json")
	if err := os.WriteFile(planPath, planRaw, 0o600); err != nil {
		t.Fatal(err)
	}
	args, err := json.Marshal([]string{
		"-plan", planPath, "-intent", intentPath, "-repo", repoRoot,
		"-go", goExe, "-isolate",
	})
	if err != nil {
		t.Fatal(err)
	}
	// A preflight byte check alone cannot establish what the Builder will
	// consume. Mutate generated.go *after* the exact BuildPlan was resolved,
	// while retaining its original plan.json and intent. The normal CLI must
	// refuse the stale-plan attempt rather than publish a different image.
	lateGo := append(append([]byte(nil), generated...), []byte("\n// changed after BuildPlan resolution\n")...)
	if err := os.WriteFile(generatedPath, lateGo, 0o600); err != nil {
		t.Fatal(err)
	}
	lateArtifact := filepath.Join(root, "late-changed-rule-command")
	lateCmd := exec.Command(cli,
		"self-build",
		"-workspace", workspace,
		"-intent", "tei.command.tei",
		"-intent-file", intentPath,
		"-build-catalog", catalogPath,
		"-build-catalog-digest", catalogDigest,
		"-build-source-root", repoRoot,
		"-child", builder,
		"-child-arg", string(args),
		"-state", filepath.Join(root, "late-changed-state"),
		"-output", lateArtifact,
	)
	var lateStdout, lateStderr bytes.Buffer
	lateCmd.Stdout, lateCmd.Stderr = &lateStdout, &lateStderr
	lateErr := lateCmd.Run()
	var lateReport l2bGoBuildResult
	if err := json.Unmarshal(lateStdout.Bytes(), &lateReport); err != nil {
		t.Fatalf("stale-plan CLI did not return JSON: run=%v decode=%v\n%s\n%s",
			lateErr, err, lateStdout.String(), lateStderr.String())
	}
	if lateErr == nil || lateReport.OK || (lateReport.L2B != nil && lateReport.L2B.Claimed) {
		t.Fatalf("post-plan change in generated.go was accepted: run=%v report=%+v\nstderr=%s",
			lateErr, lateReport, lateStderr.String())
	}
	if _, err := os.Stat(lateArtifact); !os.IsNotExist(err) {
		t.Fatalf("failed late-change attempt left a publishable artifact: %v", err)
	}
	if err := os.WriteFile(generatedPath, generated, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := requireSelectedGeneratedGo(generatedPath, generated); err != nil {
		t.Fatalf("restored generated Go does not match the selected Rule: %v", err)
	}

	// The original Markdown is also a real member of the selected Go package.
	// A prose-only edit *after* the BuildPlan was fixed must invalidate that
	// attempt even though generated.go and the pure Rule decision are unchanged.
	// This is separate from the earlier changed-Source re-Resolve check: it
	// exercises the normal self-build CLI and the original stale plan.json.
	if err := os.WriteFile(packagedSource, modified, 0o600); err != nil {
		t.Fatal(err)
	}
	staleSourceArtifact := filepath.Join(root, "late-changed-source-command")
	staleSourceCmd := exec.Command(cli,
		"self-build",
		"-workspace", workspace,
		"-intent", "tei.command.tei",
		"-intent-file", intentPath,
		"-build-catalog", catalogPath,
		"-build-catalog-digest", catalogDigest,
		"-build-source-root", repoRoot,
		"-child", builder,
		"-child-arg", string(args),
		"-state", filepath.Join(root, "late-changed-source-state"),
		"-output", staleSourceArtifact,
	)
	var staleSourceStdout, staleSourceStderr bytes.Buffer
	staleSourceCmd.Stdout, staleSourceCmd.Stderr = &staleSourceStdout, &staleSourceStderr
	staleSourceErr := staleSourceCmd.Run()
	var staleSourceReport l2bGoBuildResult
	if err := json.Unmarshal(staleSourceStdout.Bytes(), &staleSourceReport); err != nil {
		t.Fatalf("stale-original CLI did not return JSON: run=%v decode=%v\n%s\n%s",
			staleSourceErr, err, staleSourceStdout.String(), staleSourceStderr.String())
	}
	if staleSourceErr == nil || staleSourceReport.OK ||
		(staleSourceReport.L2B != nil && staleSourceReport.L2B.Claimed) {
		t.Fatalf("post-plan change in original Markdown was accepted: run=%v report=%+v\nstderr=%s",
			staleSourceErr, staleSourceReport, staleSourceStderr.String())
	}
	if _, err := os.Stat(staleSourceArtifact); !os.IsNotExist(err) {
		t.Fatalf("failed stale-original attempt left a publishable artifact: %v", err)
	}
	if err := os.WriteFile(packagedSource, original, 0o600); err != nil {
		t.Fatal(err)
	}
	restored, err := buildintent.Resolve(intent, buildintent.NewBuildContext(repoRoot, catalog))
	if err != nil {
		t.Fatalf("restored original Source does not resolve: %v", err)
	}
	if restored.PlanFingerprint != receipt.PlanFingerprint ||
		restored.SourceClosureFingerprint != receipt.SourceClosureFingerprint {
		t.Fatal("restored original Source did not restore the descriptive BuildPlan")
	}

	artifact := filepath.Join(root, "rule-command")
	cmd := exec.Command(cli,
		"self-build",
		"-workspace", workspace,
		"-intent", "tei.command.tei",
		"-intent-file", intentPath,
		"-build-catalog", catalogPath,
		"-build-catalog-digest", catalogDigest,
		"-build-source-root", repoRoot,
		"-child", builder,
		"-child-arg", string(args),
		"-state", filepath.Join(root, "state"),
		"-output", artifact,
	)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	runErr := cmd.Run()
	var report l2bGoBuildResult
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("real CLI result is not JSON: run=%v decode=%v\n%s\n%s", runErr, err, stdout.String(), stderr.String())
	}
	if runErr != nil || !report.OK ||
		report.L2B == nil || !report.L2B.Claimed ||
		report.PlanReceipt == nil || report.PlanReceipt.State != intentResolutionResolved ||
		report.PlanReceipt.DerivedBy != derivedByEntryInProcess ||
		report.PlanCrossCheck == nil || report.PlanCrossCheck.State != planCrossCheckMatched ||
		report.PlanArtifactBinding == nil || report.PlanArtifactBinding.State != planArtifactBound ||
		report.ArtifactReproducibility == nil || report.ArtifactReproducibility.State != reproducibilityReproduced ||
		report.DefinitionBinding == nil || report.DefinitionBinding.State != definitionBindingBound ||
		report.ChildRun == nil || report.ChildRun.ChildSchema != "tei.reference.go-build-from-plan/v0" ||
		report.ChildIsolation == nil || report.ChildIsolation.State != childIsolationIsolated ||
		report.ChildIsolation.MemberCount != memberCount || len(report.ChildIsolation.ModuleFiles) != 0 {
		t.Fatalf("real CLI did not bind selected generated-Go input to its artifact: run=%v report=%+v\nstderr=%s", runErr, report, stderr.String())
	}
	out, err := exec.Command(artifact).CombinedOutput()
	if err != nil {
		t.Fatalf("CLI-built generated Go could not run: %v\n%s", err, out)
	}
	const attr = "sales.order_line.quantity"
	want := fmt.Sprintf("source-sha256:%x\n", sha256.Sum256(original)) +
		"positive:satisfied:1:" + attr + "\nzero:not-satisfied:1:" + attr + "\nnegative:not-satisfied:1:" + attr +
		"\nnull:undecided:1:" + attr + "\nmissing:undecided:1:" + attr + "\n"
	if string(out) != want {
		t.Fatalf("CLI-built generated Go changed its decision/read footprint:\n%s\nwanted:\n%s", out, want)
	}

	// The same selected semantic.Validation also has a parameter-driven
	// evaluator. It is an alternative realization, not another Meaning source.
	// The generated binary was checked against independent Expected above;
	// now require the existing generic evaluator to agree on both its result
	// and the number of reads. A null value and a value that was not fetched
	// are undecided for Decide, as for the generated check function. No
	// Runtime Provider is claimed here.
	for _, tc := range []struct {
		name, number, expected string
		state                  validationeval.ValueState
	}{
		{"positive", "2.5", "satisfied", validationeval.ValuePresent},
		{"zero", "0", "not-satisfied", validationeval.ValuePresent},
		{"negative", "-1", "not-satisfied", validationeval.ValuePresent},
		{"null", "", "undecided", validationeval.ValueNull},
		{"missing", "", "undecided", validationeval.ValueNotFetched},
	} {
		t.Run("generic-vs-generated-"+tc.name, func(t *testing.T) {
			source := &documentRuleNumber{reading: validationeval.NumberReading{State: tc.state}}
			if tc.number != "" {
				value, err := semantic.ParseOperatorCaseValue(tc.number)
				if err != nil {
					t.Fatalf("bad independent input %q", tc.number)
				}
				source.reading.Value = value
			}
			decision, evalErr := validationeval.Decide(selected, source, validationeval.Operators{})
			observed := string(decision.Result)
			if decision.Undecided != nil {
				observed = "undecided"
			}
			if evalErr != nil || observed != tc.expected || source.reads != 1 {
				t.Fatalf("generic evaluation: got %s, reads=%d, err=%v; want %s, reads=1",
					observed, source.reads, evalErr, tc.expected)
			}
			generatedLine := fmt.Sprintf("%s:%s:%d:%s\n", tc.name, observed, source.reads, attr)
			if !bytes.Contains(out, []byte(generatedLine)) {
				t.Fatalf("CLI-built generated Rule differs from generic evaluation: %q not in %q",
					generatedLine, out)
			}
		})
	}
}

type documentRuleNumber struct {
	reading validationeval.NumberReading
	reads   int
}

func (s *documentRuleNumber) ReadNumber(id core.CanonicalID) (validationeval.NumberReading, error) {
	if id != "sales.order_line.quantity" {
		return validationeval.NumberReading{}, fmt.Errorf("unexpected attribute %s", id)
	}
	s.reads++
	return s.reading, nil
}

func (s *documentRuleNumber) ReadRelationKeys(relation, attribute core.CanonicalID) (validationeval.RelationReading, error) {
	return validationeval.RelationReading{}, fmt.Errorf("unexpected relation read %s / %s", relation, attribute)
}

// requireSelectedGeneratedGo is a test-only byte boundary between the
// selected Generator output and the Go package supplied to Build Intent.
// It does not establish Reader/Generator authority or runtime Grant.
func requireSelectedGeneratedGo(path string, selected []byte) error {
	staged, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if !bytes.Equal(staged, selected) {
		return fmt.Errorf("staged generated Go differs from the selected Generator output")
	}
	return nil
}

// generatedRuleCases are the inputs the harness gives the generated check
// function. An attribute missing from a row was not fetched.
var generatedRuleCases = []goemittest.Case{
	{Name: "positive", Row: map[string]goemittest.Cell{"sales.order_line.quantity": {Text: "2.5"}}},
	{Name: "zero", Row: map[string]goemittest.Cell{"sales.order_line.quantity": {Text: "0"}}},
	{Name: "negative", Row: map[string]goemittest.Cell{"sales.order_line.quantity": {Text: "-1"}}},
	{Name: "null", Row: map[string]goemittest.Cell{"sales.order_line.quantity": {Null: true}}},
	{Name: "missing", Row: map[string]goemittest.Cell{}},
}

// generatedRuleProvenance is an intentionally small Go source carried in the
// same go-package BuildSource as the original selected Rule bytes. The final
// binary must actually embed and hash the staged Markdown, not merely print a
// hash that the test injected as a literal.
const generatedRuleProvenance = `package main

import (
	"crypto/sha256"
	_ "embed"
	"fmt"
)

//go:embed source.tei.md
var embeddedSelectedRule []byte

func init() {
	fmt.Printf("source-sha256:%x\n", sha256.Sum256(embeddedSelectedRule))
}
`
