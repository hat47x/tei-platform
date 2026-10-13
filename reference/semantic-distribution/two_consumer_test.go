package semanticdistribution

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"tei/internal/semantic"
	"tei/internal/sourceindex"
	"tei/internal/validation"
	"tei/internal/workspace"
)

func selectedNamespaceFromProvider(t *testing.T, providerPath string, identity Identity, namespace string) ([]byte, Expectation) {
	t.Helper()
	provider, err := workspace.Load(providerPath)
	if err != nil {
		t.Fatal(err)
	}
	model := findNamespace(t, provider.World.Semantics, namespace)
	payload, err := MaterializeReference(identity, model)
	if err != nil {
		t.Fatal(err)
	}
	digest, err := digestNamespace(model)
	if err != nil {
		t.Fatal(err)
	}
	return payload, Expectation{
		Package: identity, NamespaceID: namespace,
		ArtifactDigest: sourceindex.Fingerprint(payload), SemanticDigest: digest,
	}
}

func containsDiagnostic(diags []validation.Diagnostic, code string) bool {
	for _, d := range diags {
		if d.Code == code {
			return true
		}
	}
	return false
}

func TestTwoIndependentConsumersShareExactlyOneSemanticExport(t *testing.T) {
	first, provider := preparePair(t)
	second := filepath.Join(filepath.Dir(first), "second")
	writeFixture(t, filepath.Join(second, "tei.workspace.yaml"),
		"workspace: second\nsources:\n  - semantic\nimports:\n  - alias: os\n    path: ../provider\n    namespace: os\n")
	writeFixture(t, filepath.Join(second, "semantic", "audit.tei"),
		"namespace auditing\n\ndata Audit {\n    attribute observedProcess type os.process_id\n}\n")

	identity := Identity{PackageID: "community.os_models", Version: "1.0.0"}
	payload, expected := selectedNamespaceFromProvider(t, provider, identity, "os")
	for _, testCase := range []struct {
		name, root string
	}{
		{"monitoring", first},
		{"auditing", second},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			pathLoaded, err := workspace.Load(testCase.root)
			if err != nil {
				t.Fatal(err)
			}
			if len(pathLoaded.Imports) != 1 || pathLoaded.Imports[0].SemanticDigest != expected.SemanticDigest {
				t.Fatalf("semantic content diverged across independent consumers: %#v", pathLoaded.Imports)
			}
			if diags := validation.Validate(pathLoaded.World); validation.HasErrors(diags) {
				t.Fatalf("path consumer is invalid: %+v", diags)
			}
			model, err := InspectSelectedReference(payload, expected)
			if err != nil {
				t.Fatal(err)
			}
			writeFixture(t, filepath.Join(testCase.root, "tei.workspace.yaml"),
				"workspace: consumer\nsources:\n  - semantic\n")
			packaged, err := workspace.Load(testCase.root)
			if err != nil {
				t.Fatal(err)
			}
			packaged.World.Semantics = append(packaged.World.Semantics, model)
			workspace.Normalize(&packaged.World)
			if !reflect.DeepEqual(pathLoaded.World.Semantics, packaged.World.Semantics) {
				t.Fatalf("path and data-only package produce different meanings for %s", testCase.name)
			}
			if diags := validation.Validate(packaged.World); validation.HasErrors(diags) {
				t.Fatalf("data-only package consumer is invalid: %+v", diags)
			}
			if len(packaged.Imports) != 0 {
				t.Fatal("data-only package falsely claims a provider Workspace import")
			}
			idx, err := workspace.BuildSourceIndex(testCase.root, packaged)
			if err != nil {
				t.Fatal(err)
			}
			for _, a := range idx.Artifacts {
				if strings.HasPrefix(a.Path, "../") || strings.Contains(a.Path, "provider") {
					t.Fatalf("provider path leaked into consumer source index: %+v", a)
				}
			}
		})
	}
}

func TestDataOnlyExportDoesNotSilentlyImportDependencyClosure(t *testing.T) {
	consumer, provider := preparePair(t)
	// This provider owns both namespace declarations; the consumer initially
	// imports only os. Neither Path nor data-only distributions may synthesize
	// an undeclared common dependency on the consumer's behalf.
	writeFixture(t, filepath.Join(provider, "semantic", "common.tei"),
		"namespace common\n\ntype MachineId from std.identifier\n")
	writeFixture(t, filepath.Join(provider, "semantic", "os", "process.tei"),
		"namespace os\n\ntype ProcessId from common.machine_id\n\ndata Process {\n    attribute processId type os.process_id\n}\n")
	osID := Identity{PackageID: "community.os_models", Version: "1.0.0"}
	osPayload, osExpected := selectedNamespaceFromProvider(t, provider, osID, "os")
	commonID := Identity{PackageID: "community.common_models", Version: "1.0.0"}
	commonPayload, commonExpected := selectedNamespaceFromProvider(t, provider, commonID, "common")

	pathOnly, err := workspace.Load(consumer)
	if err != nil {
		t.Fatal(err)
	}
	if !containsDiagnostic(validation.Validate(pathOnly.World), "UNKNOWN_SEMANTIC_TYPE_UNDERLYING") {
		t.Fatal("path import unexpectedly resolved unlisted dependency")
	}
	osModel, err := InspectSelectedReference(osPayload, osExpected)
	if err != nil {
		t.Fatal(err)
	}
	writeFixture(t, filepath.Join(consumer, "tei.workspace.yaml"),
		"workspace: consumer\nsources:\n  - semantic\n")
	packaged, err := workspace.Load(consumer)
	if err != nil {
		t.Fatal(err)
	}
	packaged.World.Semantics = append(packaged.World.Semantics, osModel)
	workspace.Normalize(&packaged.World)
	if !containsDiagnostic(validation.Validate(packaged.World), "UNKNOWN_SEMANTIC_TYPE_UNDERLYING") {
		t.Fatal("data-only export unexpectedly resolved unlisted dependency")
	}
	// Both dependencies are explicitly selected; there is no auto-discovery,
	// transitive fallback or additional Plugin resolver here.
	commonModel, err := InspectSelectedReference(commonPayload, commonExpected)
	if err != nil {
		t.Fatal(err)
	}
	packaged.World.Semantics = append(packaged.World.Semantics, commonModel)
	workspace.Normalize(&packaged.World)
	if diags := validation.Validate(packaged.World); validation.HasErrors(diags) {
		t.Fatalf("explicitly selected namespace dependencies should validate: %+v", diags)
	}
	writeFixture(t, filepath.Join(consumer, "tei.workspace.yaml"),
		"workspace: consumer\nsources:\n  - semantic\nimports:\n  - alias: os\n    path: ../provider\n    namespace: os\n  - alias: common\n    path: ../provider\n    namespace: common\n")
	explicitPath, err := workspace.Load(consumer)
	if err != nil {
		t.Fatal(err)
	}
	if diags := validation.Validate(explicitPath.World); validation.HasErrors(diags) {
		t.Fatalf("explicit Workspace imports should validate: %+v", diags)
	}
	if !reflect.DeepEqual(explicitPath.World.Semantics, packaged.World.Semantics) {
		t.Fatal("explicit closure differs between Path and data-only distributions")
	}
}

func TestCollidingProvidersCannotSilentlyReplaceNamespace(t *testing.T) {
	consumer, provider := preparePair(t)
	same, err := workspace.Load(consumer)
	if err != nil {
		t.Fatal(err)
	}
	original := findNamespace(t, same.World.Semantics, "os")
	changed := original
	changed.Types = append([]semantic.Type(nil), original.Types...)
	changed.Types[0].Underlying = "std.text"

	firstID := Identity{PackageID: "provider.one", Version: "1.0.0"}
	secondID := Identity{PackageID: "provider.two", Version: "2.0.0"}
	firstPayload, err := MaterializeReference(firstID, original)
	if err != nil {
		t.Fatal(err)
	}
	secondPayload, err := MaterializeReference(secondID, changed)
	if err != nil {
		t.Fatal(err)
	}
	firstExpected := Expectation{
		Package: firstID, NamespaceID: "os",
		ArtifactDigest: sourceindex.Fingerprint(firstPayload),
		SemanticDigest: same.Imports[0].SemanticDigest,
	}
	// An explicitly pinned first Provider rejects another Provider even if
	// the second Provider has a larger numeric version or the same namespace.
	if _, err := InspectSelectedReference(secondPayload, firstExpected); err == nil {
		t.Fatal("a different distribution provider replaced an explicitly selected one")
	}
	firstModel, err := InspectSelectedReference(firstPayload, firstExpected)
	if err != nil {
		t.Fatal(err)
	}
	secondDigest, err := digestNamespace(changed)
	if err != nil {
		t.Fatal(err)
	}
	secondModel, err := InspectSelectedReference(secondPayload, Expectation{
		Package: secondID, NamespaceID: "os",
		ArtifactDigest: sourceindex.Fingerprint(secondPayload), SemanticDigest: secondDigest,
	})
	if err != nil {
		t.Fatal(err)
	}
	// Supplying both without making an explicit exclusive selection must fail
	// existing Canonical duplicate validation, not allow last-wins override.
	writeFixture(t, filepath.Join(consumer, "tei.workspace.yaml"),
		"workspace: consumer\nsources:\n  - semantic\n")
	loaded, err := workspace.Load(consumer)
	if err != nil {
		t.Fatal(err)
	}
	loaded.World.Semantics = append(loaded.World.Semantics, firstModel, secondModel)
	workspace.Normalize(&loaded.World)
	if !containsDiagnostic(validation.Validate(loaded.World), "DUPLICATE_SEMANTIC_NAMESPACE") {
		t.Fatal("two providers exported the same namespace without duplicate diagnostic")
	}
	_ = provider // the source of the chosen fixture is not used as an authority to resolve a tie
}
