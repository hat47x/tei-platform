package workspace_test

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"tei/internal/sourceindex"
	"tei/internal/workspace"
)

// A provider Source anchor is a fresh navigation observation, not an immutable
// witness of the exact raw bytes consumed by the earlier consumer Load.
//
// This characterization is deliberately a NEGATIVE P0b test: a stable selected
// namespace digest is not a proof that the provider's physical Source bytes,
// source occurrence, or multi-file read cohort remained unchanged.
func TestImportedSourceReresolutionIsNotConsumerLoadByteWitness(t *testing.T) {
	consumer, provider := externalPair(t)
	target := sourceindex.Subject{Kind: "semantic-type", ID: "os.process_id"}
	providerSource := filepath.Join(provider, "semantic", "os", "process.tei")

	rawBefore, err := os.ReadFile(providerSource)
	if err != nil {
		t.Fatal(err)
	}
	fingerprint := func(raw []byte) string {
		sum := sha256.Sum256(raw)
		return fmt.Sprintf("sha256:%x", sum)
	}
	loadedBefore, err := workspace.Load(consumer)
	if err != nil {
		t.Fatal(err)
	}
	if len(loadedBefore.Imports) != 1 {
		t.Fatalf("expected an explicitly selected provider namespace: %+v", loadedBefore.Imports)
	}
	consumerIndexBefore, err := workspace.BuildSourceIndex(consumer, loadedBefore)
	if err != nil {
		t.Fatal(err)
	}
	originBefore, err := workspace.LocateImportedSource(consumer, loadedBefore, "os", target)
	if err != nil {
		t.Fatal(err)
	}
	if originBefore.Artifact.Path != "semantic/os/process.tei" ||
		originBefore.Artifact.Fingerprint != fingerprint(rawBefore) {
		t.Fatalf("initial provider Source origin does not match raw fixture: %+v", originBefore)
	}

	// Whitespace-only changes are a concrete counterexample: the provider has
	// new raw Source bytes, but its selected normalized semantic meaning does
	// not change. The consumer's managed Source is not edited.
	rawAfter := append(append([]byte(nil), rawBefore...), '\n')
	writeExternalFixture(t, providerSource, string(rawAfter))
	loadedAfter, err := workspace.Load(consumer)
	if err != nil {
		t.Fatal(err)
	}
	if len(loadedAfter.Imports) != 1 ||
		loadedAfter.Imports[0].SemanticDigest != loadedBefore.Imports[0].SemanticDigest {
		t.Fatalf("whitespace-only provider change unexpectedly changed selected meaning: before=%+v after=%+v", loadedBefore.Imports, loadedAfter.Imports)
	}
	consumerIndexAfter, err := workspace.BuildSourceIndex(consumer, loadedAfter)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(consumerIndexBefore.Artifacts, consumerIndexAfter.Artifacts) ||
		sourceindex.ModelRevision(consumerIndexBefore) != sourceindex.ModelRevision(consumerIndexAfter) {
		t.Fatal("provider-only raw Source change unexpectedly changed consumer-owned Source")
	}

	// Even when supplied the earlier loaded consumer Result, the navigation
	// helper deliberately reloads the provider and yields CURRENT source bytes,
	// not the provider bytes consumed by that earlier Load.
	originFromOldLoad, err := workspace.LocateImportedSource(consumer, loadedBefore, "os", target)
	if err != nil {
		t.Fatal(err)
	}
	if originFromOldLoad.Artifact.Path != originBefore.Artifact.Path ||
		originFromOldLoad.Artifact.Fingerprint != fingerprint(rawAfter) ||
		originFromOldLoad.Artifact.Fingerprint == originBefore.Artifact.Fingerprint {
		t.Fatalf("provider Source was not re-resolved as a new raw observation: before=%+v after=%+v", originBefore, originFromOldLoad)
	}

	// This test must not be used as a passing P0b input-closure or snapshot
	// test. A same-Attempt source/manifest byte witness is still not available
	// through the consumer Result or the navigation helper.
}
