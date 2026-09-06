package main

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	"tei/internal/sourceindex"
)

const canonicalProcedureSubjectKindProbe = "business-procedure"

type canonicalProcedureIdentityShellProbe struct {
	ID string `json:"id"`
}

func TestCanonicalProcedureIdentityShellProbeContainsOnlyID(t *testing.T) {
	typeInfo := reflect.TypeOf(canonicalProcedureIdentityShellProbe{})
	if typeInfo.NumField() != 1 || typeInfo.Field(0).Name != "ID" {
		t.Fatalf("identity shell probeへBusiness detailが侵入しました: %+v", typeInfo)
	}
}

func TestCanonicalProcedureIdentityShellProbeDerivesFromAuthoritativeSource(t *testing.T) {
	root := repoPath()
	paths := []string{
		filepath.Join(root, "reference", "business-system-design", "order-entry-business-procedure.json"),
		filepath.Join(root, "reference", "business-system-design", "order-entry-follow-up-procedure.json"),
	}
	shells, err := deriveCanonicalProcedureIdentityShells(paths)
	if err != nil {
		t.Fatal(err)
	}
	got := []string{shells[0].ID, shells[1].ID}
	want := []string{"order-entry.follow-up", "order-entry.order-handling"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("sourceから期待するidentity shellを導出できません: got=%v want=%v", got, want)
	}
}

func TestCanonicalProcedureIdentityShellProbeRejectsDuplicateSourceIdentity(t *testing.T) {
	root := repoPath()
	source := filepath.Join(root, "reference", "business-system-design", "order-entry-business-procedure.json")
	if _, err := deriveCanonicalProcedureIdentityShells([]string{source, source}); err == nil {
		t.Fatal("同一Business Procedure IDを複数authoritative sourceから導出しました")
	}
}

func TestCanonicalProcedureIdentityShellProbeSurvivesPureFileMove(t *testing.T) {
	root := repoPath()
	source := filepath.Join(root, "reference", "business-system-design", "order-entry-business-procedure.json")
	before, err := deriveCanonicalProcedureIdentityShells([]string{source})
	if err != nil {
		t.Fatal(err)
	}

	content, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	moved := filepath.Join(t.TempDir(), "moved", "order-handling.json")
	if err := os.MkdirAll(filepath.Dir(moved), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(moved, content, 0o600); err != nil {
		t.Fatal(err)
	}
	after, err := deriveCanonicalProcedureIdentityShells([]string{moved})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("pure file moveでidentity shellが変化しました: before=%+v after=%+v", before, after)
	}
}

func TestCanonicalProcedureIdentityShellProbeIgnoresBusinessWordingChange(t *testing.T) {
	root := repoPath()
	source := filepath.Join(root, "reference", "business-system-design", "order-entry-business-procedure.json")
	before, err := deriveCanonicalProcedureIdentityShells([]string{source})
	if err != nil {
		t.Fatal(err)
	}
	f, err := readFixture(source)
	if err != nil {
		t.Fatal(err)
	}
	for i := range f.Steps {
		if f.Steps[i].ID == "BP-04" {
			f.Steps[i].BusinessAction = "changed Business wording must not change identity shell"
		}
	}
	content, err := marshalFixture(f)
	if err != nil {
		t.Fatal(err)
	}
	changed := filepath.Join(t.TempDir(), "order-handling.json")
	if err := os.WriteFile(changed, content, 0o600); err != nil {
		t.Fatal(err)
	}
	after, err := deriveCanonicalProcedureIdentityShells([]string{changed})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("Business wording changeでidentity shellが変化しました: before=%+v after=%+v", before, after)
	}
}

func TestCanonicalProcedureIdentityShellProbeTreatsLogicalRenameAsIdentityChange(t *testing.T) {
	root := repoPath()
	source := filepath.Join(root, "reference", "business-system-design", "order-entry-business-procedure.json")
	before, err := deriveCanonicalProcedureIdentityShells([]string{source})
	if err != nil {
		t.Fatal(err)
	}
	f, err := readFixture(source)
	if err != nil {
		t.Fatal(err)
	}
	f.Procedure.ID = "order-entry.order-handling-renamed"
	content, err := marshalFixture(f)
	if err != nil {
		t.Fatal(err)
	}
	renamed := filepath.Join(t.TempDir(), "order-handling.json")
	if err := os.WriteFile(renamed, content, 0o600); err != nil {
		t.Fatal(err)
	}
	after, err := deriveCanonicalProcedureIdentityShells([]string{renamed})
	if err != nil {
		t.Fatal(err)
	}
	if reflect.DeepEqual(before, after) {
		t.Fatal("logical Procedure renameをidentity shell changeとして観測できていません")
	}
}

func TestCanonicalProcedureIdentityShellProbeCanBackWorldStyleSourceSubjectWithoutSourceMetadata(t *testing.T) {
	root := repoPath()
	source := filepath.Join(root, "reference", "business-system-design", "order-entry-business-procedure.json")
	content, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	shells, err := deriveCanonicalProcedureIdentityShells([]string{source})
	if err != nil {
		t.Fatal(err)
	}
	if len(shells) != 1 {
		t.Fatalf("identity shell count=%d; want 1", len(shells))
	}

	relativePath := "procedures/order-handling.json"
	artifact, err := sourceindex.NewArtifact(relativePath, content)
	if err != nil {
		t.Fatal(err)
	}
	rng, err := sourceindex.RangeFromOffsets(content, 0, len(content))
	if err != nil {
		t.Fatal(err)
	}
	subject := sourceindex.Subject{Kind: canonicalProcedureSubjectKindProbe, ID: shells[0].ID}
	idx := sourceindex.Index{
		Artifacts: []sourceindex.Artifact{artifact},
		References: []sourceindex.Reference{{
			Subject:      subject,
			ArtifactPath: relativePath,
			Range:        rng,
			Role:         sourceindex.RoleDefinition,
		}},
	}
	idx.Normalize()
	if err := idx.Validate(func(path string) ([]byte, error) {
		if path != relativePath {
			return nil, fmt.Errorf("unexpected path %q", path)
		}
		return content, nil
	}); err != nil {
		t.Fatal(err)
	}
	refs := idx.ReferencesFor(subject)
	if len(refs) != 1 {
		t.Fatalf("identity shell subjectをSource Indexで解決できません: %+v", refs)
	}
	if shells[0].ID == artifact.Path || shells[0].ID == artifact.Fingerprint {
		t.Fatal("Canonical identity shellがsource locator / fingerprintへ結合しています")
	}
}

func deriveCanonicalProcedureIdentityShells(paths []string) ([]canonicalProcedureIdentityShellProbe, error) {
	seen := map[string]string{}
	result := make([]canonicalProcedureIdentityShellProbe, 0, len(paths))
	for _, path := range paths {
		f, err := readFixture(path)
		if err != nil {
			return nil, err
		}
		if f.Canonical {
			return nil, fmt.Errorf("authoritative Business Procedure source %s unexpectedly claims canonical=true", path)
		}
		if f.Procedure.IdentityStatus != referenceLocalStable {
			return nil, fmt.Errorf("Business Procedure %s identityStatus=%q; want %q", f.Procedure.ID, f.Procedure.IdentityStatus, referenceLocalStable)
		}
		if previous, duplicate := seen[f.Procedure.ID]; duplicate {
			return nil, fmt.Errorf("duplicate Business Procedure identity %q in %s and %s", f.Procedure.ID, previous, path)
		}
		seen[f.Procedure.ID] = path
		result = append(result, canonicalProcedureIdentityShellProbe{ID: f.Procedure.ID})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result, nil
}
