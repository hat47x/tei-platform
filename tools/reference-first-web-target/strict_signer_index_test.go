//go:build !tei_order_only && !tei_repair_only

package main

import (
	"reflect"
	"testing"

	"tei/internal/sourceindex"
	"tei/internal/workspace"
)

// strictSignerIndex builds the Source Index a test issuer signs. It uses the
// same Load-bound builder that the Web Target verifies with
// (sourceacceptance.LoadVerifiedWorkspace), so what is signed equals what is
// verified. Test-only helper; it is not an Acceptance decision.
func strictSignerIndex(t *testing.T, root string, loaded workspace.Result) sourceindex.Index {
	t.Helper()
	index, err := workspace.BuildSourceIndexFromLoad(root, loaded)
	if err != nil {
		t.Fatal(err)
	}
	return index
}

// For the reference Workspace the legacy (re-reading) builder and the strict
// (Load-bound) builder must produce the same Source Index. If they ever
// diverge, signers that use one and verifiers that use the other would
// disagree, so this fails first.
func TestSignerStrictAndLegacySourceIndexAgreeForReferenceWorkspace(t *testing.T) {
	root := referenceWorkspacePath(t)
	loaded, err := workspace.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	legacy, err := workspace.BuildSourceIndex(root, loaded)
	if err != nil {
		t.Fatal(err)
	}
	strict := strictSignerIndex(t, root, loaded)
	if !reflect.DeepEqual(legacy.Artifacts, strict.Artifacts) ||
		sourceindex.ModelRevision(legacy) != sourceindex.ModelRevision(strict) {
		t.Fatal("legacy and strict Source Index diverge for the reference Workspace")
	}
}
