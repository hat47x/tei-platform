package workspace_test

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"tei/internal/sourceindex"
	"tei/internal/workspace"
)

// Imported provider Source-only drift is invisible to a qualification packet
// that pins only the consumer's World and its managed Source Index.
//
// This is a NEGATIVE cross-layer characterization. It does not create a
// complete import snapshot or grant reuse, source disclosure, or execution.
func TestImportedSourceOnlyDriftNeedsProviderOccurrenceBeyondConsumerRevision(t *testing.T) {
	consumer, provider := externalPair(t)
	sourcePath := filepath.Join(provider, "semantic", "os", "process.tei")
	subject := sourceindex.Subject{Kind: "semantic-type", ID: "os.process_id"}

	rawA, err := os.ReadFile(sourcePath)
	if err != nil {
		t.Fatal(err)
	}
	rawFingerprint := func(raw []byte) string {
		return fmt.Sprintf("sha256:%x", sha256.Sum256(raw))
	}
	before, err := workspace.Load(consumer)
	if err != nil {
		t.Fatal(err)
	}
	if len(before.Imports) != 1 {
		t.Fatalf("expected one selected import, got %d", len(before.Imports))
	}
	consumerIndexA, err := workspace.BuildSourceIndex(consumer, before)
	if err != nil {
		t.Fatal(err)
	}
	providerOriginA, err := workspace.LocateImportedSource(consumer, before, "os", subject)
	if err != nil {
		t.Fatal(err)
	}
	if providerOriginA.Artifact.Path != "semantic/os/process.tei" ||
		providerOriginA.Artifact.Fingerprint != rawFingerprint(rawA) {
		t.Fatalf("initial provider byte witness mismatch: %+v", providerOriginA)
	}

	// Change only the provider's authoring bytes, not the parsed semantics or
	// the consumer-owned files. The independent expected values are the raw
	// bytes, semantic equality, and consumer revision equality.
	rawB := append(append([]byte(nil), rawA...), '\n')
	writeExternalFixture(t, sourcePath, string(rawB))
	if rawFingerprint(rawA) == rawFingerprint(rawB) {
		t.Fatal("the fixture did not change the provider raw Source")
	}
	after, err := workspace.Load(consumer)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before.World, after.World) {
		t.Fatal("provider whitespace changed the consumer's parsed World")
	}
	if len(after.Imports) != 1 ||
		before.Imports[0].SemanticDigest != after.Imports[0].SemanticDigest {
		t.Fatal("provider whitespace unexpectedly changed selected semantic digest")
	}
	consumerIndexB, err := workspace.BuildSourceIndex(consumer, after)
	if err != nil {
		t.Fatal(err)
	}
	if sourceindex.ModelRevision(consumerIndexA) != sourceindex.ModelRevision(consumerIndexB) ||
		!reflect.DeepEqual(consumerIndexA.Artifacts, consumerIndexB.Artifacts) ||
		!reflect.DeepEqual(consumerIndexA.References, consumerIndexB.References) {
		t.Fatal("provider-only rewrite unexpectedly changed consumer-owned Source Index")
	}
	// This old consumer Result is not an immutable provider read attempt:
	// LocateImportedSource performs a fresh provider Load for navigation.
	providerOriginFromOldResult, err := workspace.LocateImportedSource(consumer, before, "os", subject)
	if err != nil {
		t.Fatal(err)
	}
	if providerOriginFromOldResult.Artifact.Fingerprint != rawFingerprint(rawB) ||
		providerOriginFromOldResult.Artifact.Fingerprint == providerOriginA.Artifact.Fingerprint {
		t.Fatalf("fresh provider occurrence not distinguished: initial=%+v current=%+v",
			providerOriginA.Artifact, providerOriginFromOldResult.Artifact)
	}

	// A partial imported Semantic view does not qualify a complete Source
	// read closure, even if the consumer's revision appears unchanged.
	for _, result := range []workspace.Result{before, after} {
		if _, err := workspace.BuildSourceIndexFromLoad(consumer, result); !errors.Is(err, workspace.ErrImportsNotBound) {
			t.Fatalf("imported Source was falsely qualified as a complete Load index: %v", err)
		}
	}
}


// The selected provider Manifest is a separate dependency from the
// Namespace's normalized meaning and from its current Source anchor.
// A manifest-only formatting edit must not be "proved absent" by matching
// consumer World/SourceIndex and provider Source fingerprint.
func TestImportedProviderManifestOnlyDriftNeedsSeparateManifestIdentity(t *testing.T) {
	consumer, provider := externalPair(t)
	manifestPath := filepath.Join(provider, "tei.workspace.yaml")
	target := sourceindex.Subject{Kind: "semantic-type", ID: "os.process_id"}

	rawManifestA, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	providerA, err := workspace.Load(provider)
	if err != nil {
		t.Fatal(err)
	}
	consumerA, err := workspace.Load(consumer)
	if err != nil {
		t.Fatal(err)
	}
	indexA, err := workspace.BuildSourceIndex(consumer, consumerA)
	if err != nil {
		t.Fatal(err)
	}
	sourceA, err := workspace.LocateImportedSource(consumer, consumerA, "os", target)
	if err != nil {
		t.Fatal(err)
	}
	// This explicit provider Load and the nested Load above are separate
	// observations. Their equality in this stable fixture is NOT atomicity.
	fingerprint := func(raw []byte) string {
		return fmt.Sprintf("sha256:%x", sha256.Sum256(raw))
	}
	if providerA.ManifestIdentity().Path != "tei.workspace.yaml" ||
		providerA.ManifestIdentity().SHA256 != fingerprint(rawManifestA) {
		t.Fatalf("original provider Manifest hash does not match raw fixture: %+v", providerA.ManifestIdentity())
	}

	// A second newline is syntactically inert in this controlled fixture.
	rawManifestB := append(append([]byte(nil), rawManifestA...), '\n')
	writeExternalFixture(t, manifestPath, string(rawManifestB))
	providerB, err := workspace.Load(provider)
	if err != nil {
		t.Fatal(err)
	}
	if providerB.ManifestIdentity().SHA256 != fingerprint(rawManifestB) ||
		providerB.ManifestIdentity().SHA256 == providerA.ManifestIdentity().SHA256 {
		t.Fatal("provider-only Manifest byte change was not observed")
	}
	consumerB, err := workspace.Load(consumer)
	if err != nil {
		t.Fatal(err)
	}
	indexB, err := workspace.BuildSourceIndex(consumer, consumerB)
	if err != nil {
		t.Fatal(err)
	}
	sourceFromOldConsumer, err := workspace.LocateImportedSource(consumer, consumerA, "os", target)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(consumerA.World, consumerB.World) ||
		!reflect.DeepEqual(indexA.Artifacts, indexB.Artifacts) ||
		!reflect.DeepEqual(indexA.References, indexB.References) ||
		sourceindex.ModelRevision(indexA) != sourceindex.ModelRevision(indexB) ||
		consumerA.ManifestIdentity() != consumerB.ManifestIdentity() {
		t.Fatal("provider-only Manifest edit unexpectedly changed consumer World, managed index or own Manifest")
	}
	if len(consumerA.Imports) != 1 || len(consumerB.Imports) != 1 ||
		consumerA.Imports[0].SemanticDigest != consumerB.Imports[0].SemanticDigest ||
		sourceFromOldConsumer.Artifact.Fingerprint != sourceA.Artifact.Fingerprint ||
		sourceFromOldConsumer.Artifact.Path != sourceA.Artifact.Path {
		t.Fatal("manifest-only edit unexpectedly changed selected meaning or provider Source occurrence")
	}
	// A caller has no basis to call the entire import input set current from
	// these equal consumer and Source-anchor values.
	for _, loaded := range []workspace.Result{consumerA, consumerB} {
		if _, err := workspace.BuildSourceIndexFromLoad(consumer, loaded); !errors.Is(err, workspace.ErrImportsNotBound) {
			t.Fatalf("manifest-only drift was falsely qualified as a complete import: %v", err)
		}
	}
}
