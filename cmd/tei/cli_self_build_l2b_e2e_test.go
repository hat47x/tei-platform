package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

// This is the end-to-end gate for the whole arc: one command, run against the
// reference resolver's committed catalog and the reference builder child, must
// produce an attempt where every L2-b condition holds.
//
// The unit tests around assessL2B check the predicate; this checks that the entry,
// the reference catalog and the reference child actually produce the fields the
// predicate needs. A change that breaks any one condition shows up here as a named
// condition in l2b.unmet rather than as a silently weaker claim.
//
// The reference resolver is used only to write the plan file the builder child
// consumes. The entry still derives the plan itself, which is what condition 1
// requires.

func buildToolBinary(t *testing.T, packagePath, name string) string {
	t.Helper()
	goExe, err := exec.LookPath("go")
	if err != nil {
		t.Skipf("Go toolchain unavailable: %v", err)
	}
	binary := filepath.Join(t.TempDir(), name)
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	cmd := exec.Command(goExe, "build", "-trimpath", "-o", binary, packagePath)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build %s: %v\n%s", packagePath, err, out)
	}
	return binary
}

type l2bEndToEndReport struct {
	OK        bool `json:"ok"`
	Qualified bool `json:"qualified"`
	L2B       *struct {
		Claimed bool     `json:"claimed"`
		Unmet   []string `json:"unmet"`
	} `json:"l2b"`
	ArtifactReproducibility *struct {
		State             string `json:"state"`
		UnperformedReason string `json:"unperformedReason"`
	} `json:"artifactReproducibility"`
	Error *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// setupL2BFixture writes a declaration, its sources and a workspace, and returns
// the paths the attempt needs.
func setupL2BFixture(t *testing.T) (root, intentPath, sourceRoot, catalogPath string) {
	t.Helper()
	root = t.TempDir()

	sourceRoot = filepath.Join(root, "src")
	writeWorkspaceFile(t, sourceRoot, "plugin.tei", `plugin tei.plugin.l2b_e2e v1.0.0 {
    runtime builtin
    provides deployment-executor runtime.l2b_e2e {
        action application-start
    }
}
`)
	writeWorkspaceFile(t, sourceRoot, "implementation.go.txt", "package impl\n")

	intentPath = filepath.Join(root, "intent.json")
	writeWorkspaceFile(t, root, "intent.json", `{
  "formatVersion": 1,
  "subject": {"kind": "plugin", "id": "tei.plugin.l2b_e2e", "version": "1.0.0", "definitionSource": "definition"},
  "sources": [
    {"kind": "plugin-definition", "id": "definition", "ref": "plugin.tei"},
    {"kind": "go-package", "id": "implementation", "ref": "implementation.go.txt"}
  ],
  "builder": {"contract": "tei.plugin-build", "compatibleMajor": 1},
  "recipe": {"kind": "go-package", "entrypoint": "."},
  "outputKind": "plugin-package",
  "target": "host/linux/amd64",
  "profile": "release"
}
`)

	// The catalog is not copied from a fixture: it is authored from a capability
	// declaration (§4.33), so this gate runs the chain the design describes —
	// declaration, catalog, claim — rather than starting from a catalog someone
	// wrote by hand.
	catalogPath = filepath.Join(root, "catalog.json")
	return root, intentPath, sourceRoot, catalogPath
}

// authorCatalog runs the authoring tool over a capability declaration that names
// the builder child, and returns the digest the caller must pass.
func authorCatalog(t *testing.T, root, sourceRoot, builderArtifact string) (string, string) {
	t.Helper()
	authoringTool := buildToolBinary(t, filepath.Join("..", "..", "tools", "reference-build-catalog-authoring"), "catalog-author")

	// The tool refuses a path outside -root, so the builder this catalog describes
	// is placed inside the root. The bytes — and therefore the digest — are the
	// same ones the attempt will run.
	builderBytes, err := os.ReadFile(builderArtifact)
	if err != nil {
		t.Fatalf("read the builder: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "builder"), builderBytes, 0o755); err != nil {
		t.Fatal(err)
	}
	builderArtifact = "builder"

	declaration := `{
  "providers": [
    {
      "plugin": "tei.builder.go",
      "version": "1.2.0",
      "implementation": "go.builder",
      "contract": "tei.plugin-build",
      "contractMajor": 1,
      "artifact": ` + strconv.Quote(builderArtifact) + `,
      "recipeKinds": ["go-package"],
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
	cmd := exec.Command(authoringTool,
		"-declaration", declarationPath,
		"-root", root,
		"-output", catalogPath,
	)
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("author the catalog: %v\n%s\n%s", err, stdout.String(), stderr.String())
	}
	var result struct {
		CatalogDigest string `json:"catalogDigest"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("decode the authoring result: %v\n%s", err, stdout.String())
	}
	if result.CatalogDigest == "" {
		t.Fatal("the authoring tool did not report a catalog digest")
	}

	// The catalog must name the bytes that will actually run. If it named a
	// different digest the attempt would still resolve (the lookup does not match
	// on digest), so nothing downstream would notice — which is why the gate
	// checks it here.
	var written struct {
		Providers []struct {
			ArtifactDigest string `json:"artifactDigest"`
		} `json:"providers"`
	}
	catalogBytes, err := os.ReadFile(catalogPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(catalogBytes, &written); err != nil {
		t.Fatalf("decode the authored catalog: %v", err)
	}
	if len(written.Providers) != 1 {
		t.Fatalf("the authored catalog has %d providers", len(written.Providers))
	}
	if want := digestOfBytes(builderBytes); written.Providers[0].ArtifactDigest != want {
		t.Fatalf("the catalog names %s but the builder is %s",
			written.Providers[0].ArtifactDigest, want)
	}
	return catalogPath, result.CatalogDigest
}

func TestSelfBuildEstablishesL2BEndToEnd(t *testing.T) {
	cli := buildToolBinary(t, ".", "tei")
	builderChild := buildToolBinary(t, filepath.Join("..", "..", "tools", "reference-plugin-package-build-child"), "build-child")
	resolver := buildToolBinary(t, filepath.Join("..", "..", "tools", "reference-build-authoring-intent"), "resolver")

	root, intentPath, sourceRoot, _ := setupL2BFixture(t)
	workspace := outerWorkspace(t, root)

	// The declaration names the builder this attempt will actually run, so the
	// catalog cannot describe a builder the repository does not have.
	catalogPath, catalogDigest := authorCatalog(t, root, sourceRoot, builderChild)

	// The builder child consumes a plan file. Produce it with the reference
	// resolver so the child has an input; the entry's own derivation is separate.
	planPath := filepath.Join(root, "plan.json")
	resolveCmd := exec.Command(resolver,
		"-intent", intentPath,
		"-catalog", catalogPath,
		"-source-root", sourceRoot,
	)
	var resolveStdout, resolveStderr bytes.Buffer
	resolveCmd.Stdout, resolveCmd.Stderr = &resolveStdout, &resolveStderr
	if err := resolveCmd.Run(); err != nil {
		t.Fatalf("reference resolver: %v\n%s", err, resolveStderr.String())
	}
	var receipt struct {
		Plan json.RawMessage `json:"plan"`
	}
	if err := json.Unmarshal(resolveStdout.Bytes(), &receipt); err != nil {
		t.Fatalf("decode the reference receipt: %v", err)
	}
	if err := os.WriteFile(planPath, append(receipt.Plan, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}

	childArg, err := json.Marshal([]string{
		"-plan", planPath,
		"-plugin", filepath.Join(sourceRoot, "plugin.tei"),
		"-intent", intentPath,
	})
	if err != nil {
		t.Fatal(err)
	}

	output := filepath.Join(root, "package.json")
	var stdout, stderr bytes.Buffer
	cmd := exec.Command(cli,
		"self-build",
		"-workspace", workspace,
		"-intent", "tei.plugin.l2b_e2e",
		"-intent-file", intentPath,
		"-build-catalog", catalogPath,
		"-build-catalog-digest", catalogDigest,
		"-build-source-root", sourceRoot,
		// No -rebuild-check: naming a catalog turns the second build on (§4.30).
		"-child", builderChild,
		"-child-arg", string(childArg),
		"-state", filepath.Join(root, "state"),
		"-output", output,
	)
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	runErr := cmd.Run()
	code := 0
	if runErr != nil {
		var exit *exec.ExitError
		if !errors.As(runErr, &exit) {
			t.Fatalf("run: %v", runErr)
		}
		code = exit.ExitCode()
	}

	var report l2bEndToEndReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("decode the report: %v\n%s", err, stdout.String())
	}
	if code != 0 {
		t.Fatalf("attempt failed: code=%d error=%+v l2b=%+v stderr=%s",
			code, report.Error, report.L2B, stderr.String())
	}
	if report.L2B == nil {
		t.Fatalf("the report has no l2b record: %s", stdout.String())
	}
	if !report.L2B.Claimed {
		t.Fatalf("L2-b was not claimed; unmet=%v", report.L2B.Unmet)
	}
	if len(report.L2B.Unmet) != 0 {
		t.Fatalf("claimed with unmet conditions: %v", report.L2B.Unmet)
	}
	// Claiming L2-b must not read as qualification.
	if report.Qualified {
		t.Fatal("claiming L2-b also reported the attempt as qualified")
	}
	if !report.OK {
		t.Fatal("the attempt reported ok=false")
	}
	if _, err := os.Stat(output); err != nil {
		t.Fatalf("no artifact was produced: %v", err)
	}
}

// Without the reproducibility check the same attempt must state the one missing
// condition instead of claiming less loudly.
func TestSelfBuildWithoutTheRebuildCheckNamesTheMissingCondition(t *testing.T) {
	cli := buildToolBinary(t, ".", "tei")

	root, intentPath, sourceRoot, catalogPath := setupL2BFixture(t)
	workspace := outerWorkspace(t, root)

	// This test is about -rebuild-check=false, not about authoring, so it uses a
	// catalog committed for the reference resolver.
	source := filepath.Join("..", "..", "tools", "reference-build-authoring-intent", "testdata", "catalog.json")
	catalogBytes, err := os.ReadFile(source)
	if err != nil {
		t.Fatalf("read the reference catalog: %v", err)
	}
	if err := os.WriteFile(catalogPath, catalogBytes, 0o600); err != nil {
		t.Fatal(err)
	}

	child := buildSelfBuildChild(t, root)

	var stdout, stderr bytes.Buffer
	cmd := exec.Command(cli,
		"self-build",
		"-workspace", workspace,
		"-intent", "tei.plugin.l2b_e2e",
		"-intent-file", intentPath,
		"-build-catalog", catalogPath,
		"-build-catalog-digest", handWrittenCatalogDigest(t, catalogPath),
		"-build-source-root", sourceRoot,
		// Turn the default off explicitly: this test is about that path (§4.30).
		"-rebuild-check=false",
		"-child", child,
		"-child-arg", childArgs(t, "-artifact", filepath.Join(root, "pkg.json")),
		"-state", filepath.Join(root, "state"),
		"-output", filepath.Join(root, "package.json"),
	)
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	_ = cmd.Run()

	var report l2bEndToEndReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("decode the report: %v\n%s", err, stdout.String())
	}
	if report.L2B == nil {
		t.Fatal("no l2b record")
	}
	if report.L2B.Claimed {
		t.Fatal("L2-b was claimed without the reproducibility check")
	}
	joined := strings.Join(report.L2B.Unmet, "\n")
	if !strings.Contains(joined, "not reproduced from the plan") {
		t.Fatalf("unmet does not name the missing reproduction:\n%s", joined)
	}
	// The reason must distinguish "the caller turned it off" from "nobody asked",
	// because in-process the second build is the default (§4.30).
	repro := report.ArtifactReproducibility
	if repro == nil {
		t.Fatal("no reproducibility record")
	}
	if !strings.Contains(repro.UnperformedReason, "turned off") {
		t.Fatalf("the reason does not say the caller turned it off: %q", repro.UnperformedReason)
	}
}

// handWrittenCatalogDigest is the digest of a catalog file this test wrote itself.
// It exists so the negative case can keep using a committed catalog while the
// positive case runs the authoring chain.
func handWrittenCatalogDigest(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read the catalog: %v", err)
	}
	return digestOfBytes(data)
}
