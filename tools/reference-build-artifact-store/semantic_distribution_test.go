package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"tei/internal/composition"
	"tei/internal/core"
	"tei/internal/plugin"
	"tei/internal/semantic"
	"tei/internal/sourceindex"
	"tei/internal/workspace"
	semanticcompositionpreflight "tei/reference/semantic-composition-preflight"
	semanticdistribution "tei/reference/semantic-distribution"
)

func TestBuildLockedDataOnlySemanticExportUsesExistingArtifactStore(t *testing.T) {
	providerRoot := filepath.Join(t.TempDir(), "provider")
	semanticRoot := filepath.Join(providerRoot, "semantic")
	if err := os.MkdirAll(semanticRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(providerRoot, "tei.workspace.yaml"),
		[]byte("workspace: provider\nsources:\n  - semantic\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(semanticRoot, "os.tei"),
		[]byte("namespace os\n\ntype ProcessId from std.identifier\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	provider, err := workspace.Load(providerRoot)
	if err != nil {
		t.Fatal(err)
	}
	var ns semantic.Namespace
	for _, item := range provider.World.Semantics {
		if string(item.ID) == "os" {
			ns = item
		}
	}
	if ns.ID != "os" {
		t.Fatal("provider did not own the desired semantic namespace")
	}
	identity := semanticdistribution.Identity{PackageID: "community.os_models", Version: "1.0.0"}
	data, err := semanticdistribution.MaterializeReference(identity, ns)
	if err != nil {
		t.Fatal(err)
	}
	store := openTestStore(t, 1<<20)
	put, err := store.PutBytes(data)
	if err != nil {
		t.Fatal(err)
	}
	// Existing BuildPlan and BuildArtifactLock, not a new semantic registry or
	// a forged Runtime plugin. Metadata is a reference fixture, not an
	// attestation of a production builder or signed artifact.
	plan := plugin.BuildPlan{
		FormatVersion:               plugin.BuildPlanFormatVersion,
		Plugin:                      plugin.Identity{ID: core.MustCanonicalID(identity.PackageID), Version: identity.Version},
		PluginDefinitionFingerprint: strings.Repeat("a", 64),
		Sources: []plugin.BuildSource{{
			Kind: "semantic-namespace", ID: "os", Locator: "reference://semantic/os",
			Digest: "sha256:" + strings.Repeat("b", 64),
		}},
		Builder: plugin.BuilderIdentity{
			ID: core.MustCanonicalID("tei.builder.semantic_export_reference"), Version: "1.0.0",
			ContractVersion: "tei.semantic-export-reference/v0",
			ArtifactDigest:  "sha256:" + strings.Repeat("c", 64),
		},
		Recipe:     plugin.BuildRecipe{Kind: "deterministic-semantic-reference", Entrypoint: "os"},
		OutputKind: semanticdistribution.SemanticReferenceOutputKind,
		Target:     "semantic/reference",
	}
	locked, err := plugin.LockBuildOutput(plan, put.Digest)
	if err != nil {
		t.Fatal(err)
	}
	buildLock := plugin.BuildArtifactLock{
		FormatVersion: plugin.BuildArtifactLockFormatVersion,
		Builds:        []plugin.LockedBuild{locked},
	}
	if err := buildLock.Validate(); err != nil {
		t.Fatal(err)
	}
	expected := semanticdistribution.Expectation{
		Package: identity, NamespaceID: "os", ArtifactDigest: put.Digest,
		SemanticDigest: sourceindex.Fingerprint(mustJSONSemanticNamespace(t, ns)),
	}
	fetch := func(digest string) ([]byte, error) {
		value, _, err := store.GetBytes(digest)
		return value, err
	}
	selected, err := semanticdistribution.InspectLockedSemanticAsset(buildLock, locked, expected, fetch)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(selected, ns) {
		t.Fatal("verified BuildArtifactLock/Store bytes changed the selected semantic Namespace")
	}

	// This is the first reference crossing all of these actual boundaries:
	// explicit Composition service selection, BuildArtifactLock, real CAS
	// retrieval, the exported Namespace, and a real consumer Workspace.
	consumerRoot := filepath.Join(t.TempDir(), "monitoring")
	if err := os.MkdirAll(filepath.Join(consumerRoot, "semantic"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(consumerRoot, "tei.workspace.yaml"),
		[]byte("workspace: monitoring\nsources:\n  - semantic\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(consumerRoot, "semantic", "watch.tei"),
		[]byte("namespace monitoring\n\ndata Watch {\n    attribute processId type os.process_id\n}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	consumer, err := workspace.Load(consumerRoot)
	if err != nil {
		t.Fatal(err)
	}
	if len(consumer.Imports) != 0 {
		t.Fatal("data-only distribution was disguised as a path import")
	}
	var consumerSemantic []semantic.Namespace
	for _, item := range consumer.World.Semantics {
		if item.ID != "std" {
			consumerSemantic = append(consumerSemantic, item)
		}
	}
	if len(consumerSemantic) != 1 || consumerSemantic[0].ID != "monitoring" {
		t.Fatalf("unexpected consumer semantic view: %+v", consumerSemantic)
	}
	contract := composition.Contract{
		ID: "reference.semantic.asset_read", Version: "1.0.0", Shape: "data-only/select-and-fetch/v0",
	}
	registry := composition.NewRegistry()
	if _, err := registry.Apply(0, []composition.Provider{{
		ID: "semantic-store", Scope: "tenant-a", Offers: contract,
	}}, nil); err != nil {
		t.Fatal(err)
	}
	asset := semanticcompositionpreflight.SelectedAsset{
		ProviderID: "semantic-store", Contract: contract,
		Lock: buildLock, Build: locked, Expected: expected, Fetch: fetch,
	}
	report, err := semanticcompositionpreflight.Inspect(
		registry, "tenant-a", consumerSemantic, []semanticcompositionpreflight.SelectedAsset{asset},
	)
	if err != nil {
		t.Fatal(err)
	}
	if report.Wiring.Verdict != composition.ProposalCandidate ||
		len(report.References.Dependencies) != 1 ||
		report.References.Dependencies[0].From != "monitoring" ||
		report.References.Dependencies[0].To != "os" ||
		len(report.References.Unmatched) != 0 ||
		len(report.Diagnostics) != 0 {
		t.Fatalf("service/lock/CAS/consumer semantic chain was not inspected: %+v", report)
	}

	t.Run("composition-selected-asset-cannot-fallback-on-store-miss", func(t *testing.T) {
		missing := openTestStore(t, 1<<20)
		unavailable := asset
		unavailable.Fetch = func(digest string) ([]byte, error) {
			value, _, err := missing.GetBytes(digest)
			return value, err
		}
		if _, err := semanticcompositionpreflight.Inspect(
			registry, "tenant-a", consumerSemantic, []semanticcompositionpreflight.SelectedAsset{unavailable},
		); !errors.Is(err, errArtifactNotFound) {
			t.Fatalf("missing CAS asset fell back to an unselected provider: %v", err)
		}
	})
	t.Run("composition-selection-revoked-during-real-cas-read", func(t *testing.T) {
		revoked := asset
		revoked.Fetch = func(digest string) ([]byte, error) {
			value, err := fetch(digest)
			if err != nil {
				return nil, err
			}
			revision := registry.Snapshot().Revision
			if _, err := registry.Apply(revision, nil, []string{"semantic-store"}); err != nil {
				return nil, err
			}
			return value, nil
		}
		if _, err := semanticcompositionpreflight.Inspect(
			registry, "tenant-a", consumerSemantic, []semanticcompositionpreflight.SelectedAsset{revoked},
		); !errors.Is(err, semanticcompositionpreflight.ErrProposalChanged) {
			t.Fatalf("revoked provider was accepted after actual CAS read: %v", err)
		}
	})

	t.Run("wrong-exact-lock", func(t *testing.T) {
		mutated := locked
		mutated.PlanFingerprint = strings.Repeat("d", 64)
		if _, err := semanticdistribution.InspectLockedSemanticAsset(buildLock, mutated, expected, fetch); err == nil || !strings.Contains(err.Error(), "not present in BuildArtifactLock") {
			t.Fatalf("forged exact plan was accepted: %v", err)
		}
	})
	t.Run("runtime-artifact-is-not-semantic", func(t *testing.T) {
		mutated := locked
		mutated.OutputKind = plugin.BuildOutputRuntimeArtifact
		if _, err := semanticdistribution.InspectLockedSemanticAsset(buildLock, mutated, expected, fetch); err == nil || !strings.Contains(err.Error(), "data-only") {
			t.Fatalf("executable artifact was interpreted as semantic export: %v", err)
		}
	})
	t.Run("wrong-provider", func(t *testing.T) {
		mutated := expected
		mutated.Package.PackageID = "other.provider"
		if _, err := semanticdistribution.InspectLockedSemanticAsset(buildLock, locked, mutated, fetch); err == nil || !strings.Contains(err.Error(), "identity differs") {
			t.Fatalf("wrong provider was used: %v", err)
		}
	})
	t.Run("missing-artifact", func(t *testing.T) {
		absent := openTestStore(t, 1<<20)
		read := func(digest string) ([]byte, error) {
			value, _, err := absent.GetBytes(digest)
			return value, err
		}
		if _, err := semanticdistribution.InspectLockedSemanticAsset(buildLock, locked, expected, read); !errors.Is(err, errArtifactNotFound) {
			t.Fatalf("missing exact artifact fell back silently: %v", err)
		}
	})
	t.Run("corrupt-artifact", func(t *testing.T) {
		if err := os.Chmod(receiptPath(store, put), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(receiptPath(store, put), []byte("corrupt"), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := semanticdistribution.InspectLockedSemanticAsset(buildLock, locked, expected, fetch); !errors.Is(err, errArtifactCorrupt) {
			t.Fatalf("corrupt exact bytes were accepted: %v", err)
		}
	})
}

func mustJSONSemanticNamespace(t *testing.T, ns semantic.Namespace) []byte {
	t.Helper()
	b, err := json.Marshal(ns)
	if err != nil {
		t.Fatal(err)
	}
	return b
}
