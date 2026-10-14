//go:build linux

package main

import (
	"context"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/plugin"
	customeradvice "tei/plugins/reference-project-customer-advice"
)

// BuildPlan/BuildArtifactLock are the EXISTING TEI build-trace format. The
// source list intentionally tracks only the selected project members and
// go.mod: it is not a complete Go dependency closure or a trusted BuildPlan
// issuer. Production admission must use the existing self-build closure,
// Builder acceptance and Host Trust boundaries.
func projectAdviceObservedBuild(t *testing.T) (
	string, plugin.Definition, plugin.BuildPlan, plugin.LockedBuild, plugin.LockedPlugin,
) {
	t.Helper()
	path, digest := buildStandaloneProjectAdviceExecutable(t)
	root := filepath.Clean(filepath.Join("..", ".."))
	goBinary, err := exec.LookPath("go")
	if err != nil {
		t.Fatal(err)
	}
	builderDigest, err := projectAdviceFileDigest(goBinary)
	if err != nil {
		t.Fatal(err)
	}
	definition := plugin.Definition{
		ID:      core.MustCanonicalID("tei.reference.project_customer_advice"),
		Version: "1.0.0",
		Runtime: plugin.Runtime{
			Kind:    plugin.RuntimeProcess,
			Command: "reference-project-advice-process",
			Sandbox: plugin.SandboxRequired,
		},
	}
	sources := make([]plugin.BuildSource, 0, 3)
	for _, item := range []struct{ id, path string }{
		{"project-advice-logic", "plugins/reference-project-customer-advice/advice.go"},
		{"project-advice-process", "plugins/reference-project-customer-advice/cmd/reference-project-advice-process/main.go"},
		{"go-module", "go.mod"},
	} {
		d, err := projectAdviceFileDigest(filepath.Join(root, item.path))
		if err != nil {
			t.Fatal(err)
		}
		sources = append(sources, plugin.BuildSource{
			Kind: "source-observation", ID: item.id, Locator: item.path, Digest: d,
		})
	}
	plan := plugin.BuildPlan{
		FormatVersion:               plugin.BuildPlanFormatVersion,
		Plugin:                      plugin.Identity{ID: definition.ID, Version: definition.Version},
		PluginDefinitionFingerprint: plugin.BuildDefinitionFingerprint(definition),
		Sources:                     sources,
		Builder: plugin.BuilderIdentity{
			ID:              core.MustCanonicalID("tei.builder.go"),
			Version:         runtime.Version(),
			ContractVersion: "tei.reference.go-process-build/v0",
			ArtifactDigest:  builderDigest,
		},
		Recipe: plugin.BuildRecipe{
			Kind:       "go-package",
			Entrypoint: "./plugins/reference-project-customer-advice/cmd/reference-project-advice-process",
		},
		OutputKind: plugin.BuildOutputRuntimeArtifact,
		Target:     "linux-amd64",
		Profile:    "reference",
		Parameters: []plugin.BuildParameter{
			{Name: "buildvcs", Value: "false"},
			{Name: "cgo", Value: "0"},
			{Name: "trimpath", Value: "true"},
		},
	}
	locked, err := plugin.LockBuildOutput(plan, digest)
	if err != nil {
		t.Fatal(err)
	}
	derived, err := plugin.RealizeBuiltDefinition(definition, digest)
	if err != nil {
		t.Fatal(err)
	}
	runtimeEntry := plugin.LockedPlugin{
		ID:                   derived.ID,
		Version:              derived.Version,
		Fingerprint:          plugin.DefinitionFingerprint(derived),
		ArtifactDigest:       digest,
		BuildPlanFingerprint: locked.PlanFingerprint,
	}
	return path, derived, plan, locked, runtimeEntry
}

func TestProjectAdviceBuiltArtifactIsTracedThroughExistingTEIBuildLock(t *testing.T) {
	path, definition, plan, output, entry := projectAdviceObservedBuild(t)
	if err := plugin.VerifyRuntimePluginBuild(plan, output, definition, entry); err != nil {
		t.Fatal(err)
	}
	buildLock := plugin.BuildArtifactLock{
		FormatVersion: plugin.BuildArtifactLockFormatVersion,
		Builds:        []plugin.LockedBuild{output},
	}
	if _, err := plugin.BuildArtifactLockFingerprint(buildLock); err != nil {
		t.Fatal(err)
	}
	// A tracked build is still not an accepted provider/Contract. Its digest
	// only becomes an input to the existing generic Runtime candidate.
	profile := referenceExecutionProfile(t, "linux-amd64", nil)
	candidate := projectAdviceCandidate(profile)
	candidate.ArtifactDigest = output.ArtifactDigest
	ref, err := deriveRuntimeCapabilityRef(candidate, profile)
	if err != nil {
		t.Fatal(err)
	}
	if ref.ArtifactDigest != entry.ArtifactDigest ||
		ref.PluginID != entry.ID || ref.PluginVersion != entry.Version ||
		ref.RealizationTarget != plan.Target {
		t.Fatal("build trace did not bind exact Runtime Capability identity")
	}

	// The already existing pinned-FD verifier and HostSandbox test adapters
	// still own real artifact-byte verification and process dispatch.
	environmentPlan, realization, host, pinned := projectAdvicePinnedFixture(t, path, output.ArtifactDigest)
	inv, err := invokeWithExistingHostBoundary(
		context.Background(), environmentPlan, realization, pinned, nil,
		host, "evaluate-advice", "project-advice-build-lock",
		customeradvice.Input{CustomerCode: "VIP-123"},
	)
	if err != nil {
		t.Fatal(err)
	}
	if inv.Provenance.RuntimeCapability.Fingerprint != ref.Fingerprint ||
		inv.Provenance.RuntimeCapability.ArtifactDigest != output.ArtifactDigest ||
		inv.Provenance.ArtifactVerification == nil ||
		!inv.Provenance.ArtifactVerification.DigestVerified {
		t.Fatalf("build/output and executed Runtime identity diverged: %+v", inv.Provenance)
	}
}

func TestProjectAdviceExistingBuildLockRejectsPlanAndArtifactDrift(t *testing.T) {
	_, definition, plan, build, entry := projectAdviceObservedBuild(t)

	for _, tc := range []struct {
		name string
		edit func(*plugin.BuildPlan, *plugin.LockedBuild, *plugin.LockedPlugin)
	}{
		{"project source", func(p *plugin.BuildPlan, _ *plugin.LockedBuild, _ *plugin.LockedPlugin) {
			p.Sources[0].Digest = "sha256:" + strings.Repeat("b", 64)
		}},
		{"go compiler", func(p *plugin.BuildPlan, _ *plugin.LockedBuild, _ *plugin.LockedPlugin) {
			p.Builder.ArtifactDigest = "sha256:" + strings.Repeat("c", 64)
		}},
		{"build contract", func(p *plugin.BuildPlan, _ *plugin.LockedBuild, _ *plugin.LockedPlugin) {
			p.Builder.ContractVersion = "tei.reference.go-process-build/v1"
		}},
		{"target", func(p *plugin.BuildPlan, _ *plugin.LockedBuild, _ *plugin.LockedPlugin) {
			p.Target = "wasm32-wasi"
		}},
		{"locked artifact", func(_ *plugin.BuildPlan, l *plugin.LockedBuild, _ *plugin.LockedPlugin) {
			l.ArtifactDigest = "sha256:" + strings.Repeat("d", 64)
		}},
		{"runtime plugin", func(_ *plugin.BuildPlan, _ *plugin.LockedBuild, e *plugin.LockedPlugin) {
			e.ID = core.MustCanonicalID("tei.reference.other_plugin")
		}},
		{"runtime lock", func(_ *plugin.BuildPlan, _ *plugin.LockedBuild, e *plugin.LockedPlugin) {
			e.BuildPlanFingerprint = ""
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			changedPlan := plan
			changedPlan.Sources = append([]plugin.BuildSource(nil), plan.Sources...)
			changedBuild := build
			changedEntry := entry
			tc.edit(&changedPlan, &changedBuild, &changedEntry)
			if err := plugin.VerifyRuntimePluginBuild(changedPlan, changedBuild, definition, changedEntry); err == nil {
				t.Fatal("changed build input/output was accepted under old Runtime Plugin Lock")
			}
		})
	}

	// Even a self-consistent new Plan and Artifact Lock is not a migration
	// of the old Runtime entry. An accepted Host must explicitly replace it.
	newPlan := plan
	newPlan.Sources = append([]plugin.BuildSource(nil), plan.Sources...)
	newPlan.Sources[0].Digest = "sha256:" + strings.Repeat("f", 64)
	newBuild, err := plugin.LockBuildOutput(newPlan, build.ArtifactDigest)
	if err != nil {
		t.Fatal(err)
	}
	if newBuild.PlanFingerprint == build.PlanFingerprint {
		t.Fatal("changed source did not invalidate the resolved build plan")
	}
	if err := plugin.VerifyRuntimePluginBuild(newPlan, newBuild, definition, entry); err == nil {
		t.Fatal("fresh BuildArtifactLock silently reauthorized old Runtime Plugin Lock")
	}
}

func TestProjectAdviceBuildPlanNeedsSourceAndBuilderEvidence(t *testing.T) {
	_, _, plan, _, _ := projectAdviceObservedBuild(t)
	plan.Sources[0].Digest = ""
	if _, err := plugin.BuildPlanFingerprint(plan); err == nil {
		t.Fatal("BuildPlan accepted an unpinned project source")
	}
	plan.Sources[0].Digest = "sha256:" + strings.Repeat("a", 64)
	plan.Builder.ArtifactDigest = ""
	if _, err := plugin.BuildPlanFingerprint(plan); err == nil {
		t.Fatal("BuildPlan accepted an unpinned Go builder")
	}
}
