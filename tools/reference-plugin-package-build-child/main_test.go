package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"tei/internal/core"
	"tei/internal/plugin"
	"tei/internal/syntax"
)

const testPluginSource = `plugin tei.package.build_child_test v1.0.0 {
    runtime builtin
    provides deployment-executor runtime.build_child_test {
        action application-start
    }
}
`

// writeFixture writes a Plugin definition and a matching exact BuildPlan and
// returns their paths.
func writeFixture(t *testing.T) (string, string) {
	t.Helper()
	root := t.TempDir()
	pluginPath := filepath.Join(root, "plugin.tei")
	if err := os.WriteFile(pluginPath, []byte(testPluginSource), 0o644); err != nil {
		t.Fatal(err)
	}
	definition, err := syntax.NewParser(testPluginSource).ParsePlugin()
	if err != nil {
		t.Fatal(err)
	}
	plan := plugin.BuildPlan{
		FormatVersion:               plugin.BuildPlanFormatVersion,
		Plugin:                      plugin.Identity{ID: definition.ID, Version: definition.Version},
		PluginDefinitionFingerprint: plugin.BuildDefinitionFingerprint(definition),
		Sources: []plugin.BuildSource{{
			Kind:    "plugin-definition",
			ID:      "build-child-test-definition",
			Locator: "plugin.tei",
			Digest:  "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		}},
		Builder: plugin.BuilderIdentity{
			ID:              core.MustCanonicalID("tei.builder.package.reference.seed"),
			Version:         "test",
			ContractVersion: "tei.plugin-build/v1",
			ArtifactDigest:  "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		},
		Recipe: plugin.BuildRecipe{
			Kind:       "tei-plugin-package",
			Entrypoint: "plugin.tei",
		},
		OutputKind: plugin.BuildOutputPluginPackage,
		Target:     "plugin-package",
		Profile:    "reference",
	}
	planBytes, err := json.Marshal(plan)
	if err != nil {
		t.Fatal(err)
	}
	planPath := filepath.Join(root, "plan.json")
	if err := os.WriteFile(planPath, planBytes, 0o644); err != nil {
		t.Fatal(err)
	}
	return pluginPath, planPath
}

// The child reports both the plan bytes it held and the fingerprint that
// internal/plugin derives, and writes an artifact whose digest it also reports.
func TestBuildChildReportsPlanBytesAndFingerprint(t *testing.T) {
	pluginPath, planPath := writeFixture(t)
	root := t.TempDir()
	artifactPath := filepath.Join(root, "package.json")
	stateDir := filepath.Join(root, "state")

	var stdout, stderr bytes.Buffer
	if code := run([]string{
		"-plan", planPath,
		"-plugin", pluginPath,
		"-artifact-out", artifactPath,
		"-state", stateDir,
	}, &stdout, &stderr); code != 0 {
		t.Fatalf("child failed: code=%d stderr=%s", code, stderr.String())
	}

	var report childReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("child did not emit one JSON object: %v", err)
	}
	if report.SchemaVersion != childSchema {
		t.Fatalf("schemaVersion=%q", report.SchemaVersion)
	}
	if report.PlanBytesDigest == "" || report.PlanFingerprint == "" {
		t.Fatalf("plan readings missing: %+v", report)
	}
	if report.PlanBytesDigest == report.PlanFingerprint {
		t.Fatal("byte digest and plan fingerprint collapsed into one value")
	}
	if report.ArtifactDigest == "" {
		t.Fatal("artifact digest missing")
	}
	// The reported artifact digest must match the bytes on disk, otherwise the
	// child would be trusting itself rather than the file.
	artifactBytes, err := os.ReadFile(artifactPath)
	if err != nil {
		t.Fatal(err)
	}
	if report.ArtifactDigest != digestOf(artifactBytes) {
		t.Fatal("reported artifact digest does not match the bytes written")
	}
	if report.Boundary == "" {
		t.Fatal("report lost its boundary statement")
	}

	// The plan fingerprint in the report must equal the one internal/plugin
	// derives, so a caller cannot get a fingerprint from a different plan.
	var plan plugin.BuildPlan
	planBytes, err := os.ReadFile(planPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(planBytes, &plan); err != nil {
		t.Fatal(err)
	}
	want, err := plugin.BuildPlanFingerprint(plan)
	if err != nil {
		t.Fatal(err)
	}
	if report.PlanFingerprint != want {
		t.Fatalf("planFingerprint=%q, want %q", report.PlanFingerprint, want)
	}
}

// A plan that does not validate must not produce an artifact: otherwise a
// failure would leave output behind and could be read as a successful build.
func TestBuildChildRejectsAnInvalidPlanWithoutWritingAnArtifact(t *testing.T) {
	pluginPath, _ := writeFixture(t)
	root := t.TempDir()
	planPath := filepath.Join(root, "bad-plan.json")
	if err := os.WriteFile(planPath, []byte(`{"formatVersion":1}`), 0o644); err != nil {
		t.Fatal(err)
	}
	artifactPath := filepath.Join(root, "package.json")

	var stdout, stderr bytes.Buffer
	if code := run([]string{
		"-plan", planPath,
		"-plugin", pluginPath,
		"-artifact-out", artifactPath,
	}, &stdout, &stderr); code != 1 {
		t.Fatalf("invalid plan was not rejected: code=%d", code)
	}
	if _, err := os.Stat(artifactPath); err == nil {
		t.Fatal("an invalid plan still produced an artifact")
	}
	if stdout.Len() != 0 {
		t.Fatalf("a rejected plan still wrote a JSON report: %s", stdout.String())
	}
	if stderr.Len() == 0 {
		t.Fatal("rejection had no stated reason")
	}
}

// A plan whose plugin identity does not match the definition must fail rather
// than materialize a package for a different plugin.
func TestBuildChildRejectsAMismatchedDefinition(t *testing.T) {
	_, planPath := writeFixture(t)
	root := t.TempDir()
	otherPlugin := filepath.Join(root, "other.tei")
	const otherSource = `plugin tei.package.other_probe v1.0.0 {
    runtime builtin
}
`
	if err := os.WriteFile(otherPlugin, []byte(otherSource), 0o644); err != nil {
		t.Fatal(err)
	}
	artifactPath := filepath.Join(root, "package.json")

	var stdout, stderr bytes.Buffer
	if code := run([]string{
		"-plan", planPath,
		"-plugin", otherPlugin,
		"-artifact-out", artifactPath,
	}, &stdout, &stderr); code != 1 {
		t.Fatalf("mismatched definition was not rejected: code=%d", code)
	}
	if _, err := os.Stat(artifactPath); err == nil {
		t.Fatal("a mismatched definition still produced an artifact")
	}
}

func TestBuildChildRequiresItsGrammar(t *testing.T) {
	for _, args := range [][]string{
		{},
		{"-plan", "p.json"},
		{"-plan", "p.json", "-plugin", "p.tei"},
	} {
		var stdout, stderr bytes.Buffer
		if code := run(args, &stdout, &stderr); code != 2 {
			t.Fatalf("args=%q: code=%d, want 2", args, code)
		}
		if stderr.Len() == 0 {
			t.Fatalf("args=%q: no stated reason", args)
		}
	}
}
