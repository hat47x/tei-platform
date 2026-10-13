package semanticbuildbridge

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"tei/internal/plugin"
	"tei/internal/semantic"
	"tei/internal/sourceindex"
	"tei/internal/validation"
	"tei/internal/workspace"
	semanticdistribution "tei/reference/semantic-distribution"
)

func writeSource(t *testing.T, filename, data string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(filename), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filename, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
}

func twoConsumers(t *testing.T) (string, string, string) {
	t.Helper()
	base := t.TempDir()
	provider := filepath.Join(base, "provider")
	first := filepath.Join(base, "monitor")
	second := filepath.Join(base, "audit")
	writeSource(t, filepath.Join(provider, "tei.workspace.yaml"),
		"workspace: provider\nsources:\n  - semantic\n")
	writeSource(t, filepath.Join(provider, "semantic", "process.tei"),
		"namespace os\n\ntype ProcessId from std.identifier\n\ndata Process {\n    attribute id type os.process_id\n}\n")
	for _, tc := range []struct {
		root, name, body string
	}{
		{first, "monitor", "namespace monitoring\n\ndata Watch {\n    attribute pid type os.process_id\n}\n"},
		{second, "audit", "namespace auditing\n\ndata Audit {\n    attribute pid type os.process_id\n}\n"},
	} {
		writeSource(t, filepath.Join(tc.root, "tei.workspace.yaml"),
			"workspace: "+tc.name+"\nsources:\n  - semantic\nimports:\n  - alias: os\n    path: ../provider\n    namespace: os\n")
		writeSource(t, filepath.Join(tc.root, "semantic", "consumer.tei"), tc.body)
	}
	return first, second, provider
}

func exportedNamespace(t *testing.T, provider string) semantic.Namespace {
	t.Helper()
	result, err := workspace.Load(provider)
	if err != nil {
		t.Fatal(err)
	}
	for _, ns := range result.World.Semantics {
		if ns.ID == "os" {
			return ns
		}
	}
	t.Fatal("expected provider-owned namespace os")
	return semantic.Namespace{}
}

func planFixture(packageID plugin.Identity, sourceDigest string) plugin.BuildPlan {
	return plugin.BuildPlan{
		FormatVersion:               plugin.BuildPlanFormatVersion,
		Plugin:                      packageID,
		PluginDefinitionFingerprint: plugin.BuildDefinitionFingerprint(plugin.Definition{
			ID: packageID.ID, Version: packageID.Version,
		}),
		Sources: []plugin.BuildSource{{
			Kind: "semantic-namespace-snapshot", ID: "os", Locator: "derived:workspace/os",
			Digest: sourceDigest,
		}},
		Builder: plugin.BuilderIdentity{
			ID: "tei.builders.semantic_export_reference", Version: "1.0.0",
			ContractVersion: "reference.v1", ArtifactDigest: "sha256:" + strings.Repeat("1", 64),
		},
		Recipe:     plugin.BuildRecipe{Kind: "semantic-export-reference", Entrypoint: "os"},
		OutputKind: ReferenceOutputKind,
		Target:     ReferenceTarget,
	}
}

func lockedReference(t *testing.T, ns semantic.Namespace) ([]byte, plugin.BuildPlan, plugin.LockedBuild, plugin.BuildArtifactLock, semanticdistribution.Expectation) {
	t.Helper()
	p := plugin.Identity{ID: "community.os_models", Version: "1.0.0"}
	payload, err := semanticdistribution.MaterializeReference(semanticdistribution.Identity{
		PackageID: string(p.ID), Version: p.Version,
	}, ns)
	if err != nil {
		t.Fatal(err)
	}
	semanticBytes, err := semanticDigestBytes(ns)
	if err != nil {
		t.Fatal(err)
	}
	plan := planFixture(p, sourceindex.Fingerprint(semanticBytes))
	locked, err := plugin.LockBuildOutput(plan, plugin.BuildOutputDigest(payload))
	if err != nil {
		t.Fatal(err)
	}
	lock := plugin.BuildArtifactLock{FormatVersion: plugin.BuildArtifactLockFormatVersion, Builds: []plugin.LockedBuild{locked}}
	if err := lock.Validate(); err != nil {
		t.Fatal(err)
	}
	expected := semanticdistribution.Expectation{
		Package:        semanticdistribution.Identity{PackageID: string(p.ID), Version: p.Version},
		NamespaceID:    "os",
		ArtifactDigest: plugin.BuildOutputDigest(payload),
		SemanticDigest: sourceindex.Fingerprint(semanticBytes),
	}
	return payload, plan, locked, lock, expected
}

func semanticDigestBytes(ns semantic.Namespace) ([]byte, error) {
	return json.Marshal(ns)
}

func TestTwoWorkspacesConsumeSameSelectedPluginBuildArtifact(t *testing.T) {
	first, second, provider := twoConsumers(t)
	ns := exportedNamespace(t, provider)
	payload, plan, selected, lock, expected := lockedReference(t, ns)
	if selected.ArtifactDigest == expected.SemanticDigest {
		t.Fatal("the artifact digest must not be confused with the semantic namespace digest")
	}
	built, err := InspectLockedExport(plan, lock, selected, payload, expected)
	if err != nil {
		t.Fatal(err)
	}
	for _, root := range []string{first, second} {
		path, err := workspace.Load(root)
		if err != nil {
			t.Fatal(err)
		}
		if len(path.Imports) != 1 || path.Imports[0].SemanticDigest != expected.SemanticDigest {
			t.Fatalf("workspace import digest differs from selected built asset: %#v", path.Imports)
		}
		writeSource(t, filepath.Join(root, "tei.workspace.yaml"), "workspace: consumer\nsources:\n  - semantic\n")
		packaged, err := workspace.Load(root)
		if err != nil {
			t.Fatal(err)
		}
		packaged.World.Semantics = append(packaged.World.Semantics, built)
		workspace.Normalize(&packaged.World)
		if !reflect.DeepEqual(path.World.Semantics, packaged.World.Semantics) {
			t.Fatal("selected built asset changed the consumer semantic world")
		}
		if diagnostics := validation.Validate(packaged.World); validation.HasErrors(diagnostics) {
			t.Fatalf("consumer semantic validation failed: %+v", diagnostics)
		}
		if len(packaged.Imports) != 0 {
			t.Fatal("built artifact pretended to be a Workspace-owned source import")
		}
	}
}

func TestSelectedPluginBuildLockFailClosed(t *testing.T) {
	_, _, provider := twoConsumers(t)
	ns := exportedNamespace(t, provider)
	payload, plan, selected, lock, expected := lockedReference(t, ns)
	cases := []struct {
		name   string
		mutate func(*plugin.BuildPlan, *plugin.BuildArtifactLock, *plugin.LockedBuild, *[]byte, *semanticdistribution.Expectation)
		reason string
	}{
		{"removed-build", func(_ *plugin.BuildPlan, l *plugin.BuildArtifactLock, _ *plugin.LockedBuild, _ *[]byte, _ *semanticdistribution.Expectation) {
			l.Builds = nil
		}, "exact selected build is not present"},
		{"duplicate-lock", func(_ *plugin.BuildPlan, l *plugin.BuildArtifactLock, _ *plugin.LockedBuild, _ *[]byte, _ *semanticdistribution.Expectation) {
			l.Builds = append(l.Builds, l.Builds[0])
		}, "invalid build artifact lock"},
		{"wrong-build-input", func(p *plugin.BuildPlan, _ *plugin.BuildArtifactLock, _ *plugin.LockedBuild, _ *[]byte, _ *semanticdistribution.Expectation) {
			p.Recipe.Entrypoint = "other"
		}, "build plan or artifact digest mismatch"},
		{"artifact-tampering", func(_ *plugin.BuildPlan, _ *plugin.BuildArtifactLock, _ *plugin.LockedBuild, bytes *[]byte, _ *semanticdistribution.Expectation) {
			*bytes = append(append([]byte(nil), (*bytes)...), 'x')
		}, "build plan or artifact digest mismatch"},
		{"wrong-selection-version", func(_ *plugin.BuildPlan, _ *plugin.BuildArtifactLock, s *plugin.LockedBuild, _ *[]byte, _ *semanticdistribution.Expectation) {
			s.Plugin.Version = "9.0.0"
		}, "exact selected build is not present"},
		{"wrong-selection-plan", func(_ *plugin.BuildPlan, _ *plugin.BuildArtifactLock, s *plugin.LockedBuild, _ *[]byte, _ *semanticdistribution.Expectation) {
			s.PlanFingerprint = strings.Repeat("a", 64)
		}, "exact selected build is not present"},
		{"semantic-pinning", func(_ *plugin.BuildPlan, _ *plugin.BuildArtifactLock, _ *plugin.LockedBuild, _ *[]byte, x *semanticdistribution.Expectation) {
			x.SemanticDigest = "sha256:" + strings.Repeat("0", 64)
		}, "semantic digest mismatch"},
		{"mismatched-package", func(_ *plugin.BuildPlan, _ *plugin.BuildArtifactLock, _ *plugin.LockedBuild, _ *[]byte, x *semanticdistribution.Expectation) {
			x.Package.PackageID = "other.os_models"
		}, "selected build package identity mismatch"},
		{"empty-selection", func(_ *plugin.BuildPlan, _ *plugin.BuildArtifactLock, s *plugin.LockedBuild, _ *[]byte, _ *semanticdistribution.Expectation) {
			*s = plugin.LockedBuild{}
		}, "selected build is not a data-only"},
		{"runtime-artifact-not-data", func(p *plugin.BuildPlan, _ *plugin.BuildArtifactLock, s *plugin.LockedBuild, _ *[]byte, _ *semanticdistribution.Expectation) {
			p.OutputKind = plugin.BuildOutputRuntimeArtifact
			s.OutputKind = plugin.BuildOutputRuntimeArtifact
		}, "selected build is not a data-only"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			copiedPlan, copiedLock, copiedSelected := plan, lock, selected
			copiedLock.Builds = append([]plugin.LockedBuild(nil), lock.Builds...)
			copiedPayload := append([]byte(nil), payload...)
			copiedExpected := expected
			tc.mutate(&copiedPlan, &copiedLock, &copiedSelected, &copiedPayload, &copiedExpected)
			if _, err := InspectLockedExport(copiedPlan, copiedLock, copiedSelected, copiedPayload, copiedExpected); err == nil || !strings.Contains(err.Error(), tc.reason) {
				t.Fatalf("want %q rejection, got %v", tc.reason, err)
			}
		})
	}
}
