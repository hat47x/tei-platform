package main

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeReferenceJSON(t *testing.T, name, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadSemanticNamespaceAuthoritiesUsesStrictJSON(t *testing.T) {
	valid := writeReferenceJSON(
		t,
		"authority.json",
		`{
			"namespace":"std.validation",
			"extension":"tei.std.validation",
			"evidenceFingerprint":"sha256:`+strings.Repeat("a", 64)+`"
		}`,
	)
	values, err := loadSemanticNamespaceAuthorities([]string{valid})
	if err != nil {
		t.Fatal(err)
	}
	if len(values) != 1 ||
		values[0].Namespace != "std.validation" ||
		values[0].Extension != "tei.std.validation" {
		t.Fatalf("namespace authority=%+v", values)
	}

	unknown := writeReferenceJSON(
		t,
		"unknown.json",
		`{
			"namespace":"std.validation",
			"extension":"tei.std.validation",
			"evidenceFingerprint":"sha256:`+strings.Repeat("b", 64)+`",
			"unexpected":true
		}`,
	)
	if _, err := loadSemanticNamespaceAuthorities([]string{unknown}); err == nil {
		t.Fatal("unknown namespace authority field was accepted")
	}

	trailing := writeReferenceJSON(
		t,
		"trailing.json",
		`{
			"namespace":"std.validation",
			"extension":"tei.std.validation",
			"evidenceFingerprint":"sha256:`+strings.Repeat("c", 64)+`"
		}
		{"namespace":"std.data"}`,
	)
	if _, err := loadSemanticNamespaceAuthorities([]string{trailing}); err == nil {
		t.Fatal("trailing namespace authority JSON was accepted")
	}
}

func captureReferenceCatalogRun(
	t *testing.T,
	args []string,
) (int, string, string) {
	t.Helper()
	oldStdout, oldStderr := os.Stdout, os.Stderr
	stdoutReader, stdoutWriter, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	stderrReader, stderrWriter, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout, os.Stderr = stdoutWriter, stderrWriter
	defer func() {
		os.Stdout, os.Stderr = oldStdout, oldStderr
	}()

	code := run(args)
	if err := stdoutWriter.Close(); err != nil {
		t.Fatal(err)
	}
	if err := stderrWriter.Close(); err != nil {
		t.Fatal(err)
	}
	stdout, err := io.ReadAll(stdoutReader)
	if err != nil {
		t.Fatal(err)
	}
	stderr, err := io.ReadAll(stderrReader)
	if err != nil {
		t.Fatal(err)
	}
	_ = stdoutReader.Close()
	_ = stderrReader.Close()
	return code, string(stdout), string(stderr)
}

func acyclicSubjectContextCLIArgs(t *testing.T) []string {
	t.Helper()
	root := repositoryRoot(t)
	conformance := writeReferenceJSON(
		t,
		"acyclic-conformance.json",
		`{
			"providerExtension":"tei.reference.acyclic-evaluator",
			"interface":"reference.graph.acyclic.evaluate",
			"contractFingerprint":"sha256:67fa1472c4ac84a495092064e17a06f68f2be24df87ff15589f2c0bda1baba53",
			"profile":"reference.conformance.acyclic-graph-v1",
			"evidenceFingerprint":"sha256:`+strings.Repeat("a", 64)+`"
		}`,
	)
	return []string{
		"-workspace", filepath.Join(root, "reference", "sui-adoption", "workspace"),
		"-extensions", filepath.Join(root, "tools", "reference-semantic-extension-catalog", "testdata", "extensions"),
		"-require", "tei.reference.sui-acyclic@1",
		"-require", "tei.reference.acyclic-evaluator@1",
		"-require", "tei.reference.acyclic-contract@1",
		"-interface-conformance", conformance,
		"-subject-context",
		"-subject-kind", "semantic-data",
		"-subject-id", "sui.island",
	}
}

func TestSubjectContextCLIRequiresExactSubject(t *testing.T) {
	code, _, stderr := captureReferenceCatalogRun(t, []string{"-subject-context"})
	if code != 2 || !strings.Contains(stderr, "requires exact -subject-kind and -subject-id") {
		t.Fatalf("code=%d stderr=%q", code, stderr)
	}
}

func TestSubjectContextCLICannotMixVerifyOnlyMode(t *testing.T) {
	code, _, stderr := captureReferenceCatalogRun(t, []string{
		"-subject-context",
		"-subject-kind", "semantic-data",
		"-subject-id", "sui.island",
		"-verify-semantic-lock", "unused.json",
	})
	if code != 2 || !strings.Contains(stderr, "cannot be combined with verify-only modes") {
		t.Fatalf("code=%d stderr=%q", code, stderr)
	}
}

func TestSubjectContextCLIEmitsReentrySurfaceWithoutInventingValidationPlan(t *testing.T) {
	code, stdout, stderr := captureReferenceCatalogRun(
		t,
		acyclicSubjectContextCLIArgs(t),
	)
	if code != 0 {
		t.Fatalf("code=%d stderr=%q stdout=%q", code, stderr, stdout)
	}
	var ctx SubjectSemanticContext
	if err := json.Unmarshal([]byte(stdout), &ctx); err != nil {
		t.Fatal(err)
	}
	if ctx.Subject.Kind != "semantic-data" || ctx.Subject.ID != "sui.island" {
		t.Fatalf("subject=%+v", ctx.Subject)
	}
	if ctx.ValidationPlan != nil {
		t.Fatalf("Validation Plan was invented without evaluator binding: %+v", ctx.ValidationPlan)
	}
	consumer, ok := subjectExtensionUseByID(ctx.Extensions, "tei.reference.sui-acyclic")
	if !ok || !hasString(consumer.Roles, "constraint") {
		t.Fatalf("consumer extension is not discoverable: %+v", ctx.Extensions)
	}
	provider, ok := subjectExtensionUseByID(ctx.Extensions, "tei.reference.acyclic-evaluator")
	if !ok || !hasString(provider.Roles, "evaluator-provider") {
		t.Fatalf("evaluator provider is not discoverable: %+v", ctx.Extensions)
	}
}

func TestSubjectContextCLIIncludesExactValidationPlanWhenBindingIsProvided(t *testing.T) {
	binding := writeReferenceJSON(
		t,
		"acyclic-binding.json",
		`{
			"constraintExtension":"tei.reference.sui-acyclic",
			"constraintId":"acyclic",
			"implementationId":"reference.graph.acyclic-evaluator/v1",
			"artifactDigest":"sha256:`+strings.Repeat("8", 64)+`"
		}`,
	)
	args := append(
		acyclicSubjectContextCLIArgs(t),
		"-evaluator-binding", binding,
	)
	code, stdout, stderr := captureReferenceCatalogRun(t, args)
	if code != 0 {
		t.Fatalf("code=%d stderr=%q stdout=%q", code, stderr, stdout)
	}
	var ctx SubjectSemanticContext
	if err := json.Unmarshal([]byte(stdout), &ctx); err != nil {
		t.Fatal(err)
	}
	if ctx.ValidationPlan == nil ||
		len(ctx.ValidationPlan.Bindings) != 1 ||
		ctx.ValidationPlan.Bindings[0].ImplementationID != "reference.graph.acyclic-evaluator/v1" ||
		ctx.ValidationPlan.Bindings[0].ArtifactDigest != "sha256:"+strings.Repeat("8", 64) {
		t.Fatalf("validationPlan=%+v", ctx.ValidationPlan)
	}
}
