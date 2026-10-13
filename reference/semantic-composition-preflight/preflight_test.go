package semanticcompositionpreflight

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
	semanticdistribution "tei/reference/semantic-distribution"
)

var semanticRead = composition.Contract{
	ID: "reference.semantic.asset_read", Version: "1.0.0", Shape: "data-only/select-and-fetch/v0",
}

func writeSemanticFixture(t *testing.T, file, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func selectedSemanticFixture(t *testing.T) ([]semantic.Namespace, []semantic.Namespace) {
	t.Helper()
	root := t.TempDir()
	provider := filepath.Join(root, "provider")
	consumer := filepath.Join(root, "consumer")
	writeSemanticFixture(t, filepath.Join(provider, "tei.workspace.yaml"),
		"workspace: provider\nsources:\n  - semantic\n")
	writeSemanticFixture(t, filepath.Join(provider, "semantic", "common.tei"),
		"namespace common\n\ntype MachineId from std.identifier\n")
	writeSemanticFixture(t, filepath.Join(provider, "semantic", "os.tei"),
		"namespace os\n\ntype ProcessId from common.machine_id\n\ndata Process {\n    attribute id type os.process_id\n}\n")
	writeSemanticFixture(t, filepath.Join(consumer, "tei.workspace.yaml"),
		"workspace: consumer\nsources:\n  - semantic\n")
	writeSemanticFixture(t, filepath.Join(consumer, "semantic", "watch.tei"),
		"namespace monitoring\n\ndata Watch {\n    attribute process type os.process_id\n}\n")
	loadedProvider, err := workspace.Load(provider)
	if err != nil {
		t.Fatal(err)
	}
	loadedConsumer, err := workspace.Load(consumer)
	if err != nil {
		t.Fatal(err)
	}
	if len(loadedConsumer.Imports) != 0 {
		t.Fatal("fixture must not use a path import")
	}
	var external, local []semantic.Namespace
	for _, ns := range loadedProvider.World.Semantics {
		if ns.ID != "std" {
			external = append(external, ns)
		}
	}
	for _, ns := range loadedConsumer.World.Semantics {
		if ns.ID != "std" {
			local = append(local, ns)
		}
	}
	if len(external) != 2 || len(local) != 1 {
		t.Fatalf("invalid fixture: external=%v local=%v", external, local)
	}
	return local, external
}

func buildExactSemanticAsset(t *testing.T, ns semantic.Namespace, providerID string) SelectedAsset {
	t.Helper()
	identity := semanticdistribution.Identity{
		PackageID: "community." + string(ns.ID) + "_models", Version: "1.0.0",
	}
	payload, err := semanticdistribution.MaterializeReference(identity, ns)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(ns)
	if err != nil {
		t.Fatal(err)
	}
	plan := plugin.BuildPlan{
		FormatVersion: plugin.BuildPlanFormatVersion,
		Plugin: plugin.Identity{
			ID: core.MustCanonicalID(identity.PackageID), Version: identity.Version,
		},
		PluginDefinitionFingerprint: strings.Repeat("a", 64),
		Sources: []plugin.BuildSource{{
			Kind: "semantic-namespace", ID: string(ns.ID), Locator: "reference://semantic/" + string(ns.ID),
			Digest: sourceindex.Fingerprint(raw),
		}},
		Builder: plugin.BuilderIdentity{
			ID: "tei.builder.semantic_export_reference", Version: "1.0.0",
			ContractVersion: "tei.semantic-export-reference/v0",
			ArtifactDigest:  "sha256:" + strings.Repeat("b", 64),
		},
		Recipe:     plugin.BuildRecipe{Kind: "deterministic-semantic-reference", Entrypoint: string(ns.ID)},
		OutputKind: semanticdistribution.SemanticReferenceOutputKind,
		Target:     "semantic/reference",
	}
	digest := sourceindex.Fingerprint(payload)
	locked, err := plugin.LockBuildOutput(plan, digest)
	if err != nil {
		t.Fatal(err)
	}
	lock := plugin.BuildArtifactLock{
		FormatVersion: plugin.BuildArtifactLockFormatVersion,
		Builds:        []plugin.LockedBuild{locked},
	}
	if err := lock.Validate(); err != nil {
		t.Fatal(err)
	}
	return SelectedAsset{
		ProviderID: providerID, Contract: semanticRead,
		Lock: lock, Build: locked,
		Expected: semanticdistribution.Expectation{
			Package: identity, NamespaceID: string(ns.ID),
			ArtifactDigest: digest, SemanticDigest: sourceindex.Fingerprint(raw),
		},
		Fetch: func(request string) ([]byte, error) {
			if request != digest {
				return nil, errors.New("unexpected artifact request")
			}
			return append([]byte(nil), payload...), nil
		},
	}
}

func selectedSemanticRegistry(t *testing.T) *composition.Registry {
	t.Helper()
	registry := composition.NewRegistry()
	if _, err := registry.Apply(0, []composition.Provider{
		{ID: "read-common", Scope: "tenant-a", Offers: semanticRead},
		{ID: "read-os", Scope: "tenant-a", Offers: semanticRead},
		{ID: "other-tenant", Scope: "tenant-b", Offers: semanticRead},
	}, nil); err != nil {
		t.Fatal(err)
	}
	return registry
}

func selectedAssetsFixture(t *testing.T) (*composition.Registry, []semantic.Namespace, []SelectedAsset) {
	t.Helper()
	local, external := selectedSemanticFixture(t)
	var selected []SelectedAsset
	for _, ns := range external {
		selected = append(selected, buildExactSemanticAsset(t, ns, "read-"+string(ns.ID)))
	}
	return selectedSemanticRegistry(t), local, selected
}

func TestExplicitCompositionSelectionsAndLockedSemanticAssetsYieldOnlyAReview(t *testing.T) {
	registry, local, assets := selectedAssetsFixture(t)
	original := append([]semantic.Namespace(nil), local...)
	out, err := Inspect(registry, "tenant-a", local, assets)
	if err != nil {
		t.Fatal(err)
	}
	if out.Wiring.Verdict != composition.ProposalCandidate || len(out.Wiring.Findings) != 2 ||
		len(out.References.Unmatched) != 0 || len(out.References.Dependencies) != 2 ||
		len(out.References.CycleCandidates) != 0 || len(out.Diagnostics) != 0 {
		t.Fatalf("incomplete semantic inspection: %+v", out)
	}
	if !reflect.DeepEqual(local, original) {
		t.Fatal("local semantic source was modified")
	}
	if !registry.ProposalCurrent(out.Wiring) {
		t.Fatal("the inspected structural proposal should still be current")
	}
	for i, finding := range out.Wiring.Findings {
		if finding.Selected == nil || finding.Selected.ProviderID != assets[i].ProviderID {
			t.Fatalf("provider selector was substituted for asset %d: %+v", i, finding)
		}
	}
	reverse := []SelectedAsset{assets[1], assets[0]}
	again, err := Inspect(registry, "tenant-a", local, reverse)
	if err != nil || !reflect.DeepEqual(again.References, out.References) {
		t.Fatalf("asset order changed semantic meaning: %+v vs %+v (%v)", again, out, err)
	}
}

func TestCompositionSemanticPreflightDoesNotAutoSelectOrCrossScope(t *testing.T) {
	for _, tc := range []struct {
		name string
		edit func(*[]SelectedAsset, *string)
	}{
		{"missing-selection", func(assets *[]SelectedAsset, _ *string) {
			(*assets)[0].ProviderID = ""
		}},
		{"absent-provider", func(assets *[]SelectedAsset, _ *string) {
			(*assets)[0].ProviderID = "unknown"
		}},
		{"cross-scope-provider", func(assets *[]SelectedAsset, _ *string) {
			(*assets)[0].ProviderID = "other-tenant"
		}},
		{"wrong-scope", func(_ *[]SelectedAsset, scope *string) {
			*scope = "tenant-b"
		}},
		{"wrong-contract", func(assets *[]SelectedAsset, _ *string) {
			(*assets)[0].Contract = composition.Contract{ID: "wrong", Version: "1", Shape: "wrong"}
		}},
		{"duplicate-logical-namespace", func(assets *[]SelectedAsset, _ *string) {
			(*assets)[1].Expected.NamespaceID = (*assets)[0].Expected.NamespaceID
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			registry, local, selected := selectedAssetsFixture(t)
			scope := "tenant-a"
			tc.edit(&selected, &scope)
			calls := 0
			for i := range selected {
				selected[i].Fetch = func(string) ([]byte, error) {
					calls++
					return nil, errors.New("should not fetch before selection")
				}
			}
			if _, err := Inspect(registry, scope, local, selected); err == nil {
				t.Fatal("unresolved/ambiguous selection was accepted")
			}
			if calls != 0 {
				t.Fatalf("unselected artifact was fetched %d times", calls)
			}
		})
	}
}

func TestCompositionSemanticPreflightDetectsTamperAndUnmatchedDependencies(t *testing.T) {
	t.Run("artifact-tamper", func(t *testing.T) {
		registry, local, assets := selectedAssetsFixture(t)
		assets[0].Expected.SemanticDigest = "sha256:" + strings.Repeat("0", 64)
		if _, err := Inspect(registry, "tenant-a", local, assets); err == nil || !strings.Contains(err.Error(), "semantic digest mismatch") {
			t.Fatalf("forged semantic digest passed: %v", err)
		}
	})
	t.Run("missing-explicit-namespace", func(t *testing.T) {
		registry, local, assets := selectedAssetsFixture(t)
		onlyOS := []SelectedAsset{assets[1]}
		report, err := Inspect(registry, "tenant-a", local, onlyOS)
		if !errors.Is(err, ErrInvalidSemanticView) || len(report.References.Unmatched) == 0 {
			t.Fatalf("unselected common dependency was accepted: %+v, %v", report, err)
		}
	})
	t.Run("artifact-package-mismatch", func(t *testing.T) {
		registry, local, assets := selectedAssetsFixture(t)
		assets[0].Expected.Package.Version = "9.9.9"
		if _, err := Inspect(registry, "tenant-a", local, assets); err == nil {
			t.Fatal("wrong pinned package version was accepted")
		}
	})
}

func TestCompositionSemanticPreflightRejectsRegistryChangeDuringFetch(t *testing.T) {
	registry, local, assets := selectedAssetsFixture(t)
	original := assets[0].Fetch
	assets[0].Fetch = func(digest string) ([]byte, error) {
		revision := registry.Snapshot().Revision
		if _, err := registry.Apply(revision, nil, []string{"read-os"}); err != nil {
			return nil, err
		}
		return original(digest)
	}
	result, err := Inspect(registry, "tenant-a", local, assets)
	if !errors.Is(err, ErrProposalChanged) {
		t.Fatalf("stale structural selection was accepted: %+v, %v", result, err)
	}
}

func TestCompositionSemanticPreflightRejectsExistingNamespaceWithMissingTargetSymbol(t *testing.T) {
	registry, local, assets := selectedAssetsFixture(t)
	// AuditSelectedNamespaceSet can only confirm the selected prefix "common";
	// the actual target symbol still must pass canonical semantic validation.
	_, provided := selectedSemanticFixture(t)
	var changed semantic.Namespace
	for _, ns := range provided {
		if ns.ID == "os" {
			changed = ns
			break
		}
	}
	if changed.ID != "os" || len(changed.Types) == 0 {
		t.Fatal("missing os fixture")
	}
	changed.Types[0].Underlying = "common.not_declared"
	for i := range assets {
		if assets[i].Expected.NamespaceID == "os" {
			assets[i] = buildExactSemanticAsset(t, changed, "read-os")
		}
	}
	report, err := Inspect(registry, "tenant-a", local, assets)
	if !errors.Is(err, ErrInvalidSemanticView) {
		t.Fatalf("a prefix match was mistaken for a declared semantic target: %+v, %v", report, err)
	}
	if len(report.References.Unmatched) != 0 {
		t.Fatalf("the selected prefix exists; the missing symbol is a validation failure: %+v", report.References)
	}
	if len(report.Diagnostics) == 0 {
		t.Fatal("the concrete semantic target failure lost its validator diagnostics")
	}
}
