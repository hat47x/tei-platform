package languagereference

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"tei/internal/sourceacceptance"
	"tei/internal/sourceindex"
	"tei/internal/workspace"
)

// This is a synthetic Host Reader, NOT a production authority. The real Host
// is responsible for authenticating current acceptance and revocation.
type bootstrapHostReader struct {
	record sourceacceptance.Record
	err    error
	calls  int
}

func (r *bootstrapHostReader) ReadCurrentAcceptedSource(_ context.Context, _ string) (sourceacceptance.Record, error) {
	r.calls++
	if r.err != nil {
		return sourceacceptance.Record{}, r.err
	}
	return r.record, nil
}

// BS-14's A-B-A case must not revive an old decision merely because Source
// bytes and their content-addressed revision returned to A. This probes the
// existing product Source acceptance boundary, not a new acceptance engine.
func TestBootstrapBS14ABARequiresCurrentHostRead(t *testing.T) {
	expected := bootstrapExpectedStops(t)
	if expected["BS-14"] != "review-current-qualification" {
		t.Fatalf("independent BS-14 expected stop drifted: %q", expected["BS-14"])
	}
	root := t.TempDir()
	semanticDir := filepath.Join(root, "semantic")
	if err := os.MkdirAll(semanticDir, 0o755); err != nil {
		t.Fatal(err)
	}
	write := func(path, value string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(path)), []byte(value), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write("tei.workspace.yaml", "workspace: bs14\nformatVersion: 1\nsources:\n  - semantic\n")
	write("semantic/model.tei", "namespace organization\n\ntype OrgUnitId from std.identifier\n\ndata OrgUnit {\n    attribute orgUnitId type organization.org_unit_id\n}\n")
	const initial = "# Initial README, not interpreted as TEI\n"
	write("semantic/README.md", initial)

	loadedA, err := workspace.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	indexA, err := workspace.BuildSourceIndexFromLoad(root, loadedA)
	if err != nil {
		t.Fatal(err)
	}
	revA := sourceindex.ModelRevision(indexA)
	if len(indexA.Artifacts) < 2 || len(indexA.References) == 0 {
		t.Fatal("fixture lacks meaningful managed Source and interpreted references")
	}
	reader := &bootstrapHostReader{record: sourceacceptance.Record{
		ApplicationID:     "order_entry",
		DecisionRef:       "test://bs14/synthetic-provisional",
		AuthorityRevision: "synthetic-current-1",
		ModelRevision:     revA,
		Artifacts:         append([]sourceindex.Artifact(nil), indexA.Artifacts...),
		AcceptanceKind:    sourceacceptance.AcceptanceKindProvisional,
	}}
	ctx := context.Background()
	verified, err := sourceacceptance.LoadVerifiedWorkspace(ctx, reader, "order_entry", root)
	if err != nil || verified.Acceptance.ModelRevision != revA ||
		verified.Acceptance.AcceptanceKind != sourceacceptance.AcceptanceKindProvisional {
		t.Fatalf("positive control with exact CURRENT witness failed: verified=%+v, err=%v", verified.Acceptance, err)
	}

	// B changes only a managed-but-unread document, so the same syntactic
	// World cannot be accepted under the old exact Source membership.
	write("semantic/README.md", "# Changed README, not interpreted as TEI\n")
	loadedB, err := workspace.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	indexB, err := workspace.BuildSourceIndexFromLoad(root, loadedB)
	if err != nil {
		t.Fatal(err)
	}
	if sourceindex.ModelRevision(indexB) == revA ||
		!reflect.DeepEqual(loadedA.World, loadedB.World) ||
		!reflect.DeepEqual(indexA.References, indexB.References) {
		t.Fatal("B must change Source revision, but not its interpreted meaning")
	}
	refusedB, err := sourceacceptance.LoadVerifiedWorkspace(ctx, reader, "order_entry", root)
	if err == nil || !strings.Contains(err.Error(), "SOURCE_FINGERPRINT_MISMATCH") ||
		!reflect.DeepEqual(refusedB, sourceacceptance.VerifiedWorkspace{}) {
		t.Fatalf("stale current witness unexpectedly accepted changed Source: %v", err)
	}

	// A' restores exact path and bytes. A current Host revocation still
	// refuses; content-hash equality is not itself a current Host decision.
	write("semantic/README.md", initial)
	loadedAgain, err := workspace.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	indexAgain, err := workspace.BuildSourceIndexFromLoad(root, loadedAgain)
	if err != nil {
		t.Fatal(err)
	}
	if sourceindex.ModelRevision(indexAgain) != revA ||
		!reflect.DeepEqual(indexA.References, indexAgain.References) {
		t.Fatal("A' did not restore the exact Source revision/semantic references")
	}
	reader.err = errors.New("private Host revocation token=do-not-disclose")
	refusedAgain, err := sourceacceptance.LoadVerifiedWorkspace(ctx, reader, "order_entry", root)
	if !errors.Is(err, sourceacceptance.ErrCurrentAcceptanceUnavailable) ||
		strings.Contains(err.Error(), "do-not-disclose") ||
		!reflect.DeepEqual(refusedAgain, sourceacceptance.VerifiedWorkspace{}) {
		t.Fatalf("A' bypassed current Host revocation or exposed private details: %v", err)
	}
	if reader.calls != 3 {
		t.Fatalf("expected one CURRENT Host read per attempt, got %d", reader.calls)
	}
}
