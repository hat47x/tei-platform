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
