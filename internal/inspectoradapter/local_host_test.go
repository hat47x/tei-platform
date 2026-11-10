package inspectoradapter

import (
	"errors"
	"path/filepath"
	"testing"

	"tei/internal/composition"
	"tei/internal/operations"
	"tei/internal/sourceindex"
)

func TestLocalReferenceHostChoosesRealCompositionRoute(t *testing.T) {
	host, err := newLocalInspectorHostFromObservedDigest(inspectorArtifact)
	if err != nil {
		t.Fatal(err)
	}
	if host.selection.ProviderID != LocalInspectorProviderID ||
		host.selection.Scope != "authoring-tool" ||
		host.selection.Contract != host.contract {
		t.Fatalf("incorrect local selection: %+v", host.selection)
	}
	root := filepath.Join("..", "..", "reference", "workspace")
	selected, err := host.Bind(root)
	if err != nil {
		t.Fatal(err)
	}
	want := sourceindex.Subject{Kind: "semantic-data", ID: "sales.order"}
	got, err := selected.Inspect(want)
	if err != nil || got.Subject != want ||
		got.Effect != "read-only" || got.WorldFingerprint == "" {
		t.Fatalf("local selected Inspector did not call real Operation: %+v %v", got, err)
	}
}

func TestLocalReferenceHostFailsWithoutBindingAndAfterRemoval(t *testing.T) {
	host, err := newLocalInspectorHostFromObservedDigest(inspectorArtifact)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := host.Bind(""); err == nil {
		t.Fatal("missing trusted root accepted")
	}
	root := filepath.Join("..", "..", "reference", "workspace")
	selected, err := host.Bind(root)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := host.registry.Apply(host.selection.Revision, nil, []string{LocalInspectorProviderID}); err != nil {
		t.Fatal(err)
	}
	if _, err := selected.Inspect(sourceindex.Subject{Kind: "semantic-data", ID: "sales.order"}); !errors.Is(err, composition.ErrStaleRegistry) {
		t.Fatalf("withdrawn local Provider executed: %v", err)
	}
}

func TestLocalReferenceHostRejectsInvalidDigest(t *testing.T) {
	if _, err := newLocalInspectorHostFromObservedDigest("sha256:synthetic"); err == nil {
		t.Fatal("invalid host executable observation digest accepted")
	}
}

func TestLocalReferenceHostKeepsTypedInspectionFailure(t *testing.T) {
	host, err := newLocalInspectorHostFromObservedDigest(inspectorArtifact)
	if err != nil {
		t.Fatal(err)
	}
	selected, err := host.Bind(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	_, err = selected.Inspect(sourceindex.Subject{Kind: "semantic-data", ID: "sales.order"})
	var failed *operations.ValidationFailure
	if !errors.As(err, &failed) || failed.Stage != operations.StageLoad {
		t.Fatalf("missing manifest did not preserve Loader failure: %v", err)
	}
}
