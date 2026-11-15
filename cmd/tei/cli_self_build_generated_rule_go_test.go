package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"tei/internal/buildintent"
	"tei/internal/core"
	"tei/internal/semantic"
	"tei/internal/semantic/validationgen"
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
	generated, err := validationgen.GeneratePositive(selected, attributeType, "main")
	if err != nil {
		t.Fatalf("generate positive Rule Go: %v", err)
	}
	if !bytes.Contains(generated, []byte("value.Sign() > 0")) ||
		bytes.Contains(generated, []byte("validationeval.Evaluate(")) {
		t.Fatal("generated Go must perform the selected decision itself")
	}

	// This package is deliberately inside the TEI module because generated Go
	// uses tei/internal imports. Remove it after the attempted real build.
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
	for name, data := range map[string][]byte{
		"generated.go": generated,
		"main.go": []byte(generatedRuleDriver),
	} {
		if err := os.WriteFile(filepath.Join(pkgDir, name), data, 0o600); err != nil {
			t.Fatal(err)
		}
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
	const want = "positive:satisfied:1\nzero:not-satisfied:1\nnegative:not-satisfied:1\nnull:error:1\nmissing:error:1\n"
	if string(out) != want {
		t.Fatalf("CLI-built generated Go changed its decision/read footprint:\n%s\nwanted:\n%s", out, want)
	}
}

const generatedRuleDriver = "package main\n\nimport (\n\t\"errors\"\n\t\"fmt\"\n\t\"math/big\"\n\t\"tei/internal/core\"\n)\n\ntype ruleNumber struct { value *big.Rat; missing bool; reads int }\n\nfunc (s *ruleNumber) Number(id core.CanonicalID) (*big.Rat, error) {\n\tif id != \"sales.order_line.quantity\" { return nil, errors.New(\"wrong attribute\") }\n\ts.reads++\n\tif s.missing { return nil, errors.New(\"unavailable\") }\n\treturn s.value, nil\n}\n\nfunc (s *ruleNumber) RelationKeys(_, _ core.CanonicalID) ([]string, error) {\n\treturn nil, errors.New(\"unexpected relation read\")\n}\n\nfunc main() {\n\tfor _, tc := range []struct { name, number string; missing bool }{\n\t\t{\"positive\", \"2.5\", false},\n\t\t{\"zero\", \"0\", false},\n\t\t{\"negative\", \"-1\", false},\n\t\t{\"null\", \"\", false},\n\t\t{\"missing\", \"\", true},\n\t} {\n\t\tvar value *big.Rat\n\t\tif tc.number != \"\" {\n\t\t\tvalue = new(big.Rat)\n\t\t\tif _, ok := value.SetString(tc.number); !ok { panic(tc.number) }\n\t\t}\n\t\tsource := &ruleNumber{value: value, missing: tc.missing}\n\t\tresult, err := EvaluateSelected(source)\n\t\tif err != nil {\n\t\t\tfmt.Printf(\"%s:error:%d\\n\", tc.name, source.reads)\n\t\t} else {\n\t\t\tfmt.Printf(\"%s:%s:%d\\n\", tc.name, result, source.reads)\n\t\t}\n\t}\n}\n"
