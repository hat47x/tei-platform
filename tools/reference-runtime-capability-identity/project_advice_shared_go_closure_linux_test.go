//go:build linux

package main

import (
	"context"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/plugin"
	customeradvice "tei/plugins/reference-project-customer-advice"
	"tei/tools/internal/gobuilder"
)

// This uses the SAME Go source resolver and hermetic Go builder as TEI's
// self-build reference. A local build receipt is not a trusted Builder issuer,
// an Accepted Service Contract, or an Environment execution grant.
func TestProjectAdviceSharedGoBuilderClosureBindsExactRuntimeArtifact(t *testing.T) {
	repo, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	goPath, err := gobuilder.SelectExecutable("", runtime.GOROOT(), runtime.GOOS)
	if err != nil {
		t.Fatal(err)
	}
	goDigest, err := projectAdviceFileDigest(goPath)
	if err != nil {
		t.Fatal(err)
	}
	toolchainDigest, err := gobuilder.HashToolchain(runtime.GOROOT(), runtime.GOOS, runtime.GOARCH)
	if err != nil {
		t.Fatal(err)
	}

	const entrypoint = "./plugins/reference-project-customer-advice/cmd/reference-project-advice-process"
	state := t.TempDir()
	members, closureDigest, err := gobuilder.ResolveSourceClosure(repo, goPath, entrypoint, state)
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, member := range members {
		if seen[member.Path] {
			t.Fatalf("duplicate selected Go source member: %s", member.Path)
		}
		seen[member.Path] = true
	}
	for _, required := range []string{
		"go.mod",
		"plugins/reference-project-customer-advice/advice.go",
		"plugins/reference-project-customer-advice/cmd/reference-project-advice-process/main.go",
	} {
		if !seen[required] {
			t.Fatalf("selected project Go closure misses %s", required)
		}
	}
	if closureDigest != gobuilder.SourceClosureDigest(members) {
		t.Fatal("selected Go closure fingerprint is not reproducible from exact members")
	}
	artifactPath := filepath.Join(t.TempDir(), "project-advice-built-by-shared-builder")
	if err := gobuilder.BuildPackage(repo, goPath, entrypoint, artifactPath, state); err != nil {
		t.Fatal(err)
	}
	actualArtifactDigest, err := projectAdviceFileDigest(artifactPath)
	if err != nil {
		t.Fatal(err)
	}

	source := plugin.Definition{
		ID:      core.MustCanonicalID("tei.reference.project_customer_advice"),
		Version: "1.0.0",
		Runtime: plugin.Runtime{
			Kind:    plugin.RuntimeProcess,
			Command: "reference-project-advice-process",
			Sandbox: plugin.SandboxRequired,
		},
	}
	plan := plugin.BuildPlan{
		FormatVersion:               plugin.BuildPlanFormatVersion,
		Plugin:                      plugin.Identity{ID: source.ID, Version: source.Version},
		PluginDefinitionFingerprint: plugin.BuildDefinitionFingerprint(source),
		Sources: []plugin.BuildSource{
			{Kind: "go-source-closure", ID: "selected-go-dependencies", Locator: entrypoint, Digest: closureDigest},
			{Kind: "go-toolchain-closure", ID: "host-go-toolchain", Locator: "GOROOT", Digest: toolchainDigest},
		},
		Builder: plugin.BuilderIdentity{
			ID:              core.MustCanonicalID("tei.builder.go"),
			Version:         runtime.Version(),
			ContractVersion: "tei.reference.go-process-build/v0",
			ArtifactDigest:  goDigest,
		},
		Recipe:     plugin.BuildRecipe{Kind: "go-package", Entrypoint: entrypoint},
		OutputKind: plugin.BuildOutputRuntimeArtifact,
		Target:     "linux-amd64",
		Profile:    "reference",
		Parameters: []plugin.BuildParameter{
			{Name: "buildvcs", Value: "false"},
			{Name: "hermeticGoSettings", Value: strings.Join(gobuilder.FixedSettings(runtime.GOOS, runtime.GOARCH), ";")},
			{Name: "mod", Value: "readonly"},
			{Name: "trimpath", Value: "true"},
		},
	}
	built, err := plugin.LockBuildOutput(plan, actualArtifactDigest)
	if err != nil {
		t.Fatal(err)
	}
	definition, err := plugin.RealizeBuiltDefinition(source, actualArtifactDigest)
	if err != nil {
		t.Fatal(err)
	}
	entry := plugin.LockedPlugin{
		ID:                   definition.ID,
		Version:              definition.Version,
		Fingerprint:          plugin.DefinitionFingerprint(definition),
		ArtifactDigest:       actualArtifactDigest,
		BuildPlanFingerprint: built.PlanFingerprint,
	}
	if err := plugin.VerifyRuntimePluginBuild(plan, built, definition, entry); err != nil {
		t.Fatal(err)
	}
	legacy := plugin.Lock{
		FormatVersion: plugin.LockFormatVersion,
		Environments: []plugin.EnvironmentLock{{
			Environment: core.MustCanonicalID("reference_runtime"),
			Plugins:     []plugin.LockedPlugin{{ID: entry.ID, Version: entry.Version, Fingerprint: entry.Fingerprint, ArtifactDigest: entry.ArtifactDigest}},
		}},
	}
	upgraded, err := plugin.BindRuntimeBuild(legacy, core.MustCanonicalID("reference_runtime"), built)
	if err != nil {
		t.Fatal(err)
	}
	if upgraded.FormatVersion != plugin.BuildTraceLockFormatVersion ||
		upgraded.Environments[0].Plugins[0].BuildPlanFingerprint != built.PlanFingerprint {
		t.Fatal("existing Runtime Plugin Lock lost source-closure-bound BuildPlan")
	}

	// The old synthetic Contract/Environment Grant remains separate. This
	// invocation verifies only build identity and pinned-FD transport.
	executionPlan, realization, host, pinned := projectAdvicePinnedFixture(t, artifactPath, actualArtifactDigest)
	invocation, err := invokeWithExistingHostBoundary(
		context.Background(), executionPlan, realization, pinned, nil, host,
		"evaluate-advice", "project-go-complete-closure",
		customeradvice.Input{CustomerCode: "VIP-123"},
	)
	if err != nil {
		t.Fatal(err)
	}
	if invocation.Provenance.RuntimeCapability.ArtifactDigest != entry.ArtifactDigest ||
		!invocation.Provenance.ArtifactVerification.DigestVerified {
		t.Fatal("pinned executed artifact no longer matches exact locked build")
	}
	profile := referenceExecutionProfile(t, "linux-amd64", nil)
	candidate := projectAdviceCandidate(profile)
	candidate.ArtifactDigest = actualArtifactDigest
	ref, err := deriveRuntimeCapabilityRef(candidate, profile)
	if err != nil {
		t.Fatal(err)
	}
	if ref.Fingerprint != invocation.Provenance.RuntimeCapability.Fingerprint {
		t.Fatal("build-derived candidate differs from actual Runtime Capability Ref")
	}

	mutated := append([]gobuilder.SourceMember(nil), members...)
	// A changed selected source invalidates both the closure and the locked
	// Runtime Plugin entry; a docs-only change is outside this selected set.
	for i := range mutated {
		if mutated[i].Path == "plugins/reference-project-customer-advice/advice.go" {
			mutated[i].Digest = "sha256:" + strings.Repeat("f", 64)
		}
	}
	mutatedDigest := gobuilder.SourceClosureDigest(mutated)
	if mutatedDigest == closureDigest {
		t.Fatal("project logic byte drift was invisible to Go source closure")
	}
	changed := plan
	changed.Sources = append([]plugin.BuildSource(nil), plan.Sources...)
	changed.Sources[0].Digest = mutatedDigest
	if err := plugin.VerifyRuntimePluginBuild(changed, built, definition, entry); err == nil {
		t.Fatal("changed resolved Go source closure reused old locked build")
	}
	changed = plan
	changed.Builder.ArtifactDigest = "sha256:" + strings.Repeat("a", 64)
	if err := plugin.VerifyRuntimePluginBuild(changed, built, definition, entry); err == nil {
		t.Fatal("changed exact Go builder identity reused old locked build")
	}
}
