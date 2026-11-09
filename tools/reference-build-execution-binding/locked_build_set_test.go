package main

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/plugin"
)

func artifactDigest(data []byte) string {
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func writeMultiBuildFile(t *testing.T, root, rel, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// buildTwoActualArtifacts runs one `go build` invocation that writes two
// distinct executables. The bytes are real compiler outputs, not fixtures.
func buildTwoActualArtifacts(t *testing.T) [][]byte {
	t.Helper()
	if _, err := exec.LookPath("go"); err != nil {
		t.Skip("go command is not on PATH")
	}
	module := t.TempDir()
	writeMultiBuildFile(t, module, "go.mod", "module example.test/multibuild\n\ngo 1.22\n")
	writeMultiBuildFile(t, module, "cmd/one/main.go", "package main\n\nfunc main() { println(\"one\") }\n")
	writeMultiBuildFile(t, module, "cmd/two/main.go", "package main\n\nfunc main() { println(\"two\") }\n")
	out := t.TempDir()
	command := exec.Command("go", "build", "-trimpath", "-buildvcs=false",
		"-o", out+string(os.PathSeparator), "./cmd/one", "./cmd/two")
	command.Dir = module
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("actual multi-package Go build failed: %v\n%s", err, output)
	}
	files, err := filepath.Glob(filepath.Join(out, "*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 2 {
		t.Fatalf("one build execution must produce two artifacts, got %v", files)
	}
	sort.Strings(files)
	artifacts := make([][]byte, 0, len(files))
	for _, file := range files {
		info, err := os.Stat(file)
		if err != nil || !info.Mode().IsRegular() || info.Size() == 0 {
			t.Fatalf("compiler output %s is not a regular non-empty file: %v", file, err)
		}
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		artifacts = append(artifacts, data)
	}
	if artifactDigest(artifacts[0]) == artifactDigest(artifacts[1]) {
		t.Fatal("multi-member fixture must produce distinct member bytes")
	}
	return artifacts
}

func lockedBuildSetFixture(t *testing.T, artifacts [][]byte) (plugin.LockedBuild, BuildExecutionResult, ArtifactSet, string) {
	t.Helper()
	set := ArtifactSet{
		FormatVersion: artifactSetFormatVersion,
		Members: []ArtifactMember{
			{Role: "executable-one", Format: "application/octet-stream", Digest: artifactDigest(artifacts[0])},
			{Role: "executable-two", Format: "application/octet-stream", Digest: artifactDigest(artifacts[1])},
		},
	}
	plan := referencePluginBuildPlan("tei.reference.multi_member_plugin", strings.Repeat("a", 64))
	lockedPlanFingerprint, err := plugin.BuildPlanFingerprint(plan)
	if err != nil {
		t.Fatal(err)
	}
	locked := plugin.LockedBuild{
		Plugin:          plan.Plugin,
		OutputKind:      plan.OutputKind,
		Target:          plan.Target,
		Profile:         plan.Profile,
		PlanFingerprint: lockedPlanFingerprint,
		ArtifactDigest:  artifactDigest(artifacts[0]),
	}
	executionPlan := BuildExecutionPlan{
		FormatVersion: buildExecutionPlanFormatVersion,
		Sources: []plugin.BuildSource{{
			Kind: "source", ID: "multi", Locator: "source://multi", Digest: digestA,
		}},
		Builder: plugin.BuilderIdentity{
			ID: core.MustCanonicalID("tei.builder.multi_member_reference"), Version: "1.0.0",
			ContractVersion: "tei.reference.build/v1", ArtifactDigest: digestB,
		},
		Recipe:         plugin.BuildRecipe{Kind: "multi-member-reference", Entrypoint: "multi"},
		OutputContract: "multi-member-artifact-set",
		Target:         "host/reference",
	}
	executionFingerprint, err := buildExecutionPlanFingerprint(executionPlan)
	if err != nil {
		t.Fatal(err)
	}
	setFingerprint, err := artifactSetFingerprint(set)
	if err != nil {
		t.Fatal(err)
	}
	result := BuildExecutionResult{
		FormatVersion:            buildExecutionResultFormatVersion,
		ExecutionPlanFingerprint: executionFingerprint,
		ArtifactSetFingerprint:   setFingerprint,
	}
	return locked, result, set, "executable-one"
}

func TestActualMultiMemberBuildBindsLockedBuildToArtifactSet(t *testing.T) {
	artifacts := buildTwoActualArtifacts(t)
	locked, result, set, role := lockedBuildSetFixture(t, artifacts)
	binding, err := bindLockedBuildToArtifactSet(locked, result, set, role)
	if err != nil {
		t.Fatal(err)
	}
	if err := binding.Validate(); err != nil {
		t.Fatal(err)
	}
	if binding.OutputRole != role || binding.MemberDigest != artifactDigest(artifacts[0]) {
		t.Fatalf("binding did not select the locked member: %+v", binding)
	}
	if binding.ArtifactSetFingerprint == "" || binding.BuildExecutionResultFingerprint == "" {
		t.Fatalf("binding lost the set/result identity: %+v", binding)
	}
	if binding.LockedBuild.ArtifactDigest != artifactDigest(artifacts[0]) {
		t.Fatalf("binding rewrote the LockedBuild digest: %+v", binding.LockedBuild)
	}
	first, err := lockedBuildArtifactSetBindingFingerprint(binding)
	if err != nil {
		t.Fatal(err)
	}
	second, err := lockedBuildArtifactSetBindingFingerprint(binding)
	if err != nil || first != second {
		t.Fatalf("binding fingerprint is not deterministic: %q %q %v", first, second, err)
	}
}

func TestLockedBuildSetBindingRejectsWrongRoleOrDigest(t *testing.T) {
	artifacts := buildTwoActualArtifacts(t)
	locked, result, set, _ := lockedBuildSetFixture(t, artifacts)
	if _, err := bindLockedBuildToArtifactSet(locked, result, set, "absent-role"); err == nil {
		t.Fatal("absent Artifact Set role was accepted")
	}
	// The lock fixes executable-one; claiming it is executable-two must fail.
	if _, err := bindLockedBuildToArtifactSet(locked, result, set, "executable-two"); err == nil {
		t.Fatal("LockedBuild digest was rebound to a different member role")
	}
	wrongResult := result
	wrongResult.ArtifactSetFingerprint = digestC
	if _, err := bindLockedBuildToArtifactSet(locked, wrongResult, set, "executable-one"); err == nil {
		t.Fatal("execution result not binding the Artifact Set was accepted")
	}
	tampered := locked
	tampered.ArtifactDigest = artifactDigest(artifacts[1])
	if _, err := bindLockedBuildToArtifactSet(tampered, result, set, "executable-one"); err == nil {
		t.Fatal("tampered LockedBuild digest was accepted")
	}
}

func TestLockedBuildSetBindingDoesNotPromoteAuthority(t *testing.T) {
	artifacts := buildTwoActualArtifacts(t)
	locked, result, set, role := lockedBuildSetFixture(t, artifacts)
	binding, err := bindLockedBuildToArtifactSet(locked, result, set, role)
	if err != nil {
		t.Fatal(err)
	}
	// The binding carries no Acceptance/Trust/Runtime fields; its presence must
	// not change the LockedBuild or the execution result.
	if binding.LockedBuild.PlanFingerprint != locked.PlanFingerprint ||
		binding.LockedBuild.ArtifactDigest != locked.ArtifactDigest ||
		result.ArtifactSetFingerprint == "" {
		t.Fatalf("binding changed the locked inputs: %+v", binding)
	}
}
