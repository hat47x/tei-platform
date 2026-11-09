package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
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

	// The catalog committed beside the reference resolver: data the caller names,
	// which is what §4.22 requires.
	source := filepath.Join("..", "..", "tools", "reference-build-authoring-intent", "testdata", "catalog.json")
	catalogBytes, err := os.ReadFile(source)
	if err != nil {
		t.Fatalf("read the reference catalog: %v", err)
	}
	catalogPath = filepath.Join(root, "catalog.json")
	if err := os.WriteFile(catalogPath, catalogBytes, 0o600); err != nil {
		t.Fatal(err)
	}
	return root, intentPath, sourceRoot, catalogPath
}

func TestSelfBuildEstablishesL2BEndToEnd(t *testing.T) {
	cli := buildToolBinary(t, ".", "tei")
	builderChild := buildToolBinary(t, filepath.Join("..", "..", "tools", "reference-plugin-package-build-child"), "build-child")
	resolver := buildToolBinary(t, filepath.Join("..", "..", "tools", "reference-build-authoring-intent"), "resolver")

	root, intentPath, sourceRoot, catalogPath := setupL2BFixture(t)
	workspace := outerWorkspace(t, root)

	catalogBytes, err := os.ReadFile(catalogPath)
	if err != nil {
		t.Fatal(err)
	}
	catalogDigest := digestOfBytes(catalogBytes)

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
	catalogBytes, err := os.ReadFile(catalogPath)
	if err != nil {
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
		"-build-catalog-digest", digestOfBytes(catalogBytes),
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
