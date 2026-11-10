package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"

	"tei/internal/buildintent"
)

// §4.39 stage 3: the claim made from a real Go build must be checked every time,
// not measured once. This runs the whole chain — declaration, catalog, plan, a
// builder that reads the plan, the entry's five conditions — and fails if any of
// them stops holding.
//
// The target is a small package rather than ./cmd/tei so the suite stays usable;
// the mechanism is the same, and the ./cmd/tei observation is recorded in
// plan/evidence/TEI_GO_SELF_BUILD_L2B_FROM_REAL_GO_BUILD_2026-11-10.md.

type l2bGoBuildResult struct {
	OK  bool `json:"ok"`
	L2B *struct {
		Claimed bool     `json:"claimed"`
		Unmet   []string `json:"unmet"`
	} `json:"l2b"`
	PlanReceipt *struct {
		State     string `json:"state"`
		DerivedBy string `json:"derivedBy"`
	} `json:"planReceipt"`
	PlanCrossCheck *struct {
		State string `json:"state"`
	} `json:"planCrossCheck"`
	PlanArtifactBinding *struct {
		State string `json:"state"`
	} `json:"planArtifactBinding"`
	ArtifactReproducibility *struct {
		State string `json:"state"`
	} `json:"artifactReproducibility"`
	DefinitionBinding *struct {
		State string `json:"state"`
	} `json:"definitionBinding"`
	Error *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// authorGoCommandCatalog writes a catalog whose provider builds a Go command, and
// returns its path and digest. The builder is copied into the root because the
// authoring tool refuses a path outside it.
func authorGoCommandCatalog(t *testing.T, root, builderArtifact string) (string, string) {
	t.Helper()
	tool := buildToolBinary(t, filepath.Join("..", "..", "tools", "reference-build-catalog-authoring"), "catalog-author")
	bytesOfBuilder, err := os.ReadFile(builderArtifact)
	if err != nil {
		t.Fatalf("read the builder: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "builder"), bytesOfBuilder, 0o755); err != nil {
		t.Fatal(err)
	}
	declaration := `{
  "providers": [
    {
      "plugin": "tei.builder.go",
      "version": "1.2.0",
      "implementation": "go.build-from-plan",
      "contract": "tei.plugin-build",
      "contractMajor": 1,
      "artifact": "builder",
      "recipeKinds": ["go-command"],
      "targets": ["host/linux/amd64"],
      "profiles": ["release"]
    }
  ],
  "targets": [{"target": "host/linux/amd64", "profiles": ["release"]}]
}
`
	declarationPath := filepath.Join(root, "capability-declaration.json")
	if err := os.WriteFile(declarationPath, []byte(declaration), 0o600); err != nil {
		t.Fatal(err)
	}
	catalogPath := filepath.Join(root, "catalog.json")
	var stdout, stderr bytes.Buffer
	cmd := exec.Command(tool, "-declaration", declarationPath, "-root", root, "-output", catalogPath)
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("author the catalog: %v\n%s", err, stderr.String())
	}
	catalogBytes, err := os.ReadFile(catalogPath)
	if err != nil {
		t.Fatal(err)
	}
	return catalogPath, digestOfBytes(catalogBytes)
}

func TestSelfBuildEstablishesL2BFromARealGoBuild(t *testing.T) {
	if testing.Short() {
		// A real build twice (the second one is the reproducibility check) is the
		// point, so this test costs what it costs. -short is for callers who trade
		// the gate away knowingly.
		t.Skip("a real go build twice is the observation; -short skips it")
	}
	if _, err := exec.LookPath("go"); err != nil {
		t.Skipf("Go toolchain unavailable: %v", err)
	}
	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	goExe, err := exec.LookPath("go")
	if err != nil {
		t.Skipf("Go toolchain unavailable: %v", err)
	}

	cli := buildToolBinary(t, ".", "tei")
	builder := buildToolBinary(t, filepath.Join("..", "..", "tools", "reference-go-build-from-plan"), "go-build-from-plan")

	root := t.TempDir()
	catalogPath, catalogDigest := authorGoCommandCatalog(t, root, builder)
	workspace := outerWorkspace(t, root)

	// §4.40: a command build is defined by the package it compiles, so the
	// definitionSource names the go-package source. The plugin declaration is
	// still listed as a source the plan carries, but it does not define the build.
	definitionPath := filepath.Join(repoRoot, "reference", "plugin-self-build", "workspace", "plugins",
		"tei-native-runtime-reference", "1.0.0", "plugin.tei")
	if _, statErr := os.Stat(definitionPath); statErr != nil {
		t.Skipf("the plugin definition used by this test is missing: %v", statErr)
	}
	intentBody := `{
  "formatVersion": 1,
  "subject": {"kind": "plugin", "id": "tei.native.runtime.reference", "version": "1.0.0", "definitionSource": "tei"},
  "sources": [
    {"kind": "plugin-definition", "id": "definition", "ref": "reference/plugin-self-build/workspace/plugins/tei-native-runtime-reference/1.0.0/plugin.tei"},
    {"kind": "go-package", "id": "tei", "ref": "cmd/tei/testdata/l2b-go-command"}
  ],
  "builder": {"contract": "tei.plugin-build", "compatibleMajor": 1},
  "recipe": {"kind": "go-command", "entrypoint": "./cmd/tei/testdata/l2b-go-command"},
  "outputKind": "runtime-artifact",
  "target": "host/linux/amd64",
  "profile": "release"
}
`
	intentPath := filepath.Join(root, "intent.json")
	if err := os.WriteFile(intentPath, []byte(intentBody), 0o600); err != nil {
		t.Fatal(err)
	}

	// The plan is the entry's own derivation, so the child is given exactly the
	// plan the entry will compare against.
	catalogBytes, err := os.ReadFile(catalogPath)
	if err != nil {
		t.Fatal(err)
	}
	catalog, err := buildintent.ParseBuildCatalog(catalogBytes)
	if err != nil {
		t.Fatal(err)
	}
	intent, err := buildintent.ParseIntent([]byte(intentBody))
	if err != nil {
		t.Fatal(err)
	}
	receipt, err := buildintent.Resolve(intent, buildintent.NewBuildContext(repoRoot, catalog))
	if err != nil {
		t.Fatalf("derive the plan: %v", err)
	}
	planPath := filepath.Join(root, "plan.json")
	planJSON, err := json.Marshal(receipt.Plan)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(planPath, planJSON, 0o600); err != nil {
		t.Fatal(err)
	}

	childArg, err := json.Marshal([]string{
		"-plan", planPath,
		"-intent", intentPath,
		"-repo", repoRoot,
		"-go", goExe,
	})
	if err != nil {
		t.Fatal(err)
	}

	output := filepath.Join(root, "artifact")
	var stdout, stderr bytes.Buffer
	cmd := exec.Command(cli,
		"self-build",
		"-workspace", workspace,
		"-intent", "tei.native.runtime.reference",
		"-intent-file", intentPath,
		"-build-catalog", catalogPath,
		"-build-catalog-digest", catalogDigest,
		"-build-source-root", repoRoot,
		"-child", builder,
		"-child-arg", string(childArg),
		"-state", filepath.Join(root, "state"),
		"-output", output,
	)
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	_ = cmd.Run()

	var report l2bGoBuildResult
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("decode the report: %v\n%s\n%s", err, stdout.String(), stderr.String())
	}
	if !report.OK {
		t.Fatalf("the attempt failed: %+v\n%s", report.Error, stderr.String())
	}
	if report.L2B == nil || !report.L2B.Claimed {
		t.Fatalf("l2b was not claimed: %+v", report.L2B)
	}
	for name, state := range map[string]string{
		"planReceipt":             receiptState(report.PlanReceipt),
		"planCrossCheck":          recordState(report.PlanCrossCheck),
		"planArtifactBinding":     recordState(report.PlanArtifactBinding),
		"artifactReproducibility": recordState(report.ArtifactReproducibility),
	} {
		want := map[string]string{
			"planReceipt":             intentResolutionResolved,
			"planCrossCheck":          planCrossCheckMatched,
			"planArtifactBinding":     planArtifactBound,
			"artifactReproducibility": reproducibilityReproduced,
		}[name]
		if state != want {
			t.Fatalf("%s = %q, want %q", name, state, want)
		}
	}
	// The builder reports the definition it checked, so the entry can compare it
	// with the plan rather than trusting the builder to refuse.
	if report.DefinitionBinding == nil || report.DefinitionBinding.State != definitionBindingBound {
		t.Fatalf("definitionBinding = %+v, want %q", report.DefinitionBinding, definitionBindingBound)
	}
	if _, statErr := os.Stat(output); statErr != nil {
		t.Fatalf("no artifact was produced: %v", statErr)
	}
}

func receiptState(r *struct {
	State     string `json:"state"`
	DerivedBy string `json:"derivedBy"`
}) string {
	if r == nil {
		return ""
	}
	return r.State
}

func recordState(r *struct {
	State string `json:"state"`
}) string {
	if r == nil {
		return ""
	}
	return r.State
}

var _ = strconv.Quote
