package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"tei/internal/sourceindex"
	"tei/internal/workspace"
)

const procedureAuthoringSubjectKind = "business-procedure-authoring"

type acceptedProcedureSource struct {
	Fixture  fixture
	Artifact sourceindex.Artifact
	Subject  sourceindex.Subject
}

func TestForwardCompatibleProcedureSourceRootDoesNotChangeWorld(t *testing.T) {
	root := repoPath()
	baseline, err := workspace.Load(filepath.Join(root, "reference", "workspace"))
	if err != nil {
		t.Fatal(err)
	}

	probeRoot := prepareProcedureAuthoringWorkspace(t, root)
	loaded, err := workspace.Load(probeRoot)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(baseline.World, loaded.World) {
		t.Fatal("procedures source rootを追加しただけでCanonical Worldが変化しました")
	}

	managed := false
	for _, file := range loaded.ManagedFiles {
		if file.Path == "procedures/order-handling.json" && file.SourceRoot == "procedures" {
			managed = true
		}
	}
	if !managed {
		t.Fatalf("forward-compatible procedures rootのfileがManagedFilesへ入りません: %+v", loaded.ManagedFiles)
	}
}

func TestAcceptedProcedureSourceUsesAuthoringReferenceWithoutWorldExpansion(t *testing.T) {
	root := repoPath()
	probeRoot := prepareProcedureAuthoringWorkspace(t, root)
	loaded, err := workspace.Load(probeRoot)
	if err != nil {
		t.Fatal(err)
	}
	worldBefore := loaded.World

	accepted, err := acceptProcedureSource(probeRoot, "procedures/order-handling.json", &loaded, "")
	if err != nil {
		t.Fatal(err)
	}
	if accepted.Fixture.Canonical {
		t.Fatal("accepted authoring sourceがCanonical Procedureを要求しています")
	}
	if accepted.Fixture.Procedure.IdentityStatus != referenceLocalStable {
		t.Fatalf("identityStatus=%q; want %q", accepted.Fixture.Procedure.IdentityStatus, referenceLocalStable)
	}
	if !reflect.DeepEqual(worldBefore, loaded.World) {
		t.Fatal("authoring SourceContribution登録でCanonical Worldが変化しました")
	}

	idx, err := workspace.BuildSourceIndex(probeRoot, loaded)
	if err != nil {
		t.Fatal(err)
	}
	refs := idx.ReferencesFor(accepted.Subject)
	if len(refs) != 1 {
		t.Fatalf("Business Procedure authoring subjectをSource Indexでexactに1件解決できません: %+v", refs)
	}
	if refs[0].Role != sourceindex.RoleAuthoring {
		t.Fatalf("Business Procedure source role=%q; want %q", refs[0].Role, sourceindex.RoleAuthoring)
	}
	if refs[0].ArtifactPath != "procedures/order-handling.json" {
		t.Fatalf("unexpected authoring locator %q", refs[0].ArtifactPath)
	}
}

func TestAcceptedProcedureSourceReloadKeepsKeyAndDerivedMeaning(t *testing.T) {
	root := repoPath()
	probeRoot := prepareProcedureAuthoringWorkspace(t, root)

	firstResult, err := workspace.Load(probeRoot)
	if err != nil {
		t.Fatal(err)
	}
	first, err := acceptProcedureSource(probeRoot, "procedures/order-handling.json", &firstResult, "")
	if err != nil {
		t.Fatal(err)
	}
	firstProjection := evaluate(firstResult.World, first.Fixture)

	secondResult, err := workspace.Load(probeRoot)
	if err != nil {
		t.Fatal(err)
	}
	second, err := acceptProcedureSource(probeRoot, "procedures/order-handling.json", &secondResult, first.Artifact.Fingerprint)
	if err != nil {
		t.Fatal(err)
	}
	secondProjection := evaluate(secondResult.World, second.Fixture)

	if first.Subject != second.Subject {
		t.Fatalf("reloadでProcedure authoring subjectが変化しました: before=%+v after=%+v", first.Subject, second.Subject)
	}
	if first.Artifact.Fingerprint != second.Artifact.Fingerprint {
		t.Fatal("source非変更reloadでartifact fingerprintが変化しました")
	}
	if !reflect.DeepEqual(firstProjection, secondProjection) {
		t.Fatal("source非変更reloadでcross-view Projectionが変化しました")
	}
}

func TestBusinessOnlyAcceptedSourceChangeRequiresFreshAcceptance(t *testing.T) {
	root := repoPath()
	probeRoot := prepareProcedureAuthoringWorkspace(t, root)
	procedurePath := filepath.Join(probeRoot, "procedures", "order-handling.json")

	beforeResult, err := workspace.Load(probeRoot)
	if err != nil {
		t.Fatal(err)
	}
	before, err := acceptProcedureSource(probeRoot, "procedures/order-handling.json", &beforeResult, "")
	if err != nil {
		t.Fatal(err)
	}
	beforeProjection := evaluate(beforeResult.World, before.Fixture)

	changed := before.Fixture
	found := false
	for i := range changed.Steps {
		if changed.Steps[i].ID == "BP-04" {
			changed.Steps[i].BusinessAction = "decide whether surrounding business circumstances permit submission"
			found = true
		}
	}
	if !found {
		t.Fatal("test setupがBP-04を見つけられません")
	}
	content, err := json.MarshalIndent(changed, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	content = append(content, '\n')
	if err := os.WriteFile(procedurePath, content, 0o600); err != nil {
		t.Fatal(err)
	}

	staleResult, err := workspace.Load(probeRoot)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := acceptProcedureSource(probeRoot, "procedures/order-handling.json", &staleResult, before.Artifact.Fingerprint); err == nil || !strings.Contains(err.Error(), "stale base fingerprint") {
		t.Fatalf("stale source acceptanceをfail closedにできていません: %v", err)
	}

	currentBytes, err := os.ReadFile(procedurePath)
	if err != nil {
		t.Fatal(err)
	}
	currentArtifact, err := sourceindex.NewArtifact("procedures/order-handling.json", currentBytes)
	if err != nil {
		t.Fatal(err)
	}
	freshResult, err := workspace.Load(probeRoot)
	if err != nil {
		t.Fatal(err)
	}
	fresh, err := acceptProcedureSource(probeRoot, "procedures/order-handling.json", &freshResult, currentArtifact.Fingerprint)
	if err != nil {
		t.Fatal(err)
	}
	freshProjection := evaluate(freshResult.World, fresh.Fixture)

	if before.Subject != fresh.Subject {
		t.Fatal("Business-only changeでreview / authoring target keyが変化しました")
	}
	if before.Artifact.Fingerprint == fresh.Artifact.Fingerprint {
		t.Fatal("Business-only source changeをfingerprint差として観測できていません")
	}
	if reflect.DeepEqual(beforeProjection.BusinessActionFunctionRows, freshProjection.BusinessActionFunctionRows) {
		t.Fatal("Business-only source changeがBusiness-side Projectionへ現れていません")
	}
	if !reflect.DeepEqual(beforeProjection.BusinessActionPatternMeaningRows, freshProjection.BusinessActionPatternMeaningRows) {
		t.Fatal("Business-only source changeでPattern-derived Application meaningまで変化しました")
	}
	if !reflect.DeepEqual(beforeResult.World, freshResult.World) {
		t.Fatal("Business-only Procedure source changeでCanonical Worldが変化しました")
	}
}

func TestAcceptedProcedureSourceMoveChangesLocatorNotAuthoringSubject(t *testing.T) {
	root := repoPath()
	probeRoot := prepareProcedureAuthoringWorkspace(t, root)

	beforeResult, err := workspace.Load(probeRoot)
	if err != nil {
		t.Fatal(err)
	}
	before, err := acceptProcedureSource(probeRoot, "procedures/order-handling.json", &beforeResult, "")
	if err != nil {
		t.Fatal(err)
	}

	movedRel := "procedures/sales/order-handling.json"
	movedPath := filepath.Join(probeRoot, filepath.FromSlash(movedRel))
	if err := os.MkdirAll(filepath.Dir(movedPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(filepath.Join(probeRoot, "procedures", "order-handling.json"), movedPath); err != nil {
		t.Fatal(err)
	}

	afterResult, err := workspace.Load(probeRoot)
	if err != nil {
		t.Fatal(err)
	}
	after, err := acceptProcedureSource(probeRoot, movedRel, &afterResult, before.Artifact.Fingerprint)
	if err != nil {
		t.Fatal(err)
	}

	if before.Subject != after.Subject {
		t.Fatalf("file moveでBusiness Procedure authoring subjectが変化しました: before=%+v after=%+v", before.Subject, after.Subject)
	}
	if before.Artifact.Fingerprint != after.Artifact.Fingerprint {
		t.Fatal("pure file moveでartifact fingerprintが変化しました")
	}
	if before.Artifact.Path == after.Artifact.Path {
		t.Fatal("test setupがsource locatorを移動できていません")
	}

	idx, err := workspace.BuildSourceIndex(probeRoot, afterResult)
	if err != nil {
		t.Fatal(err)
	}
	refs := idx.ReferencesFor(after.Subject)
	if len(refs) != 1 || refs[0].ArtifactPath != movedRel {
		t.Fatalf("Source Index locatorがmove後pathへ追随していません: %+v", refs)
	}
}

func acceptProcedureSource(root, relativePath string, result *workspace.Result, expectedBaseFingerprint string) (acceptedProcedureSource, error) {
	content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(relativePath)))
	if err != nil {
		return acceptedProcedureSource{}, err
	}
	f, err := decodeFixture(strings.NewReader(string(content)))
	if err != nil {
		return acceptedProcedureSource{}, fmt.Errorf("Business Procedure authoring source %s is invalid: %w", relativePath, err)
	}
	if f.Canonical {
		return acceptedProcedureSource{}, fmt.Errorf("Business Procedure authoring source %s must remain non-Canonical", relativePath)
	}
	if f.Procedure.IdentityStatus != referenceLocalStable {
		return acceptedProcedureSource{}, fmt.Errorf("Business Procedure %s identityStatus=%q; want %q", f.Procedure.ID, f.Procedure.IdentityStatus, referenceLocalStable)
	}
	artifact, err := sourceindex.NewArtifact(relativePath, content)
	if err != nil {
		return acceptedProcedureSource{}, err
	}
	if expectedBaseFingerprint != "" && artifact.Fingerprint != expectedBaseFingerprint {
		return acceptedProcedureSource{}, fmt.Errorf("stale base fingerprint: have %s want %s", artifact.Fingerprint, expectedBaseFingerprint)
	}
	subject := sourceindex.Subject{Kind: procedureAuthoringSubjectKind, ID: f.Procedure.ID}
	result.SourceContributions = append(result.SourceContributions, workspace.SourceContribution{
		Subject: subject,
		File:    relativePath,
		Role:    sourceindex.RoleAuthoring,
	})
	return acceptedProcedureSource{Fixture: f, Artifact: artifact, Subject: subject}, nil
}

func prepareProcedureAuthoringWorkspace(t *testing.T, repoRoot string) string {
	t.Helper()
	sourceRoot := filepath.Join(repoRoot, "reference", "workspace")
	targetRoot := filepath.Join(t.TempDir(), "workspace")
	copyDirectory(t, sourceRoot, targetRoot)

	manifestPath := filepath.Join(targetRoot, "tei.workspace.yaml")
	manifest, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(string(manifest), "\n") {
		manifest = append(manifest, '\n')
	}
	manifest = append(manifest, []byte("  - procedures\n")...)
	if err := os.WriteFile(manifestPath, manifest, 0o600); err != nil {
		t.Fatal(err)
	}

	procedureDir := filepath.Join(targetRoot, "procedures")
	if err := os.MkdirAll(procedureDir, 0o755); err != nil {
		t.Fatal(err)
	}
	procedureSource, err := os.ReadFile(filepath.Join(repoRoot, "reference", "business-system-design", "order-entry-business-procedure.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(procedureDir, "order-handling.json"), procedureSource, 0o600); err != nil {
		t.Fatal(err)
	}
	return targetRoot
}

func copyDirectory(t *testing.T, sourceRoot, targetRoot string) {
	t.Helper()
	err := filepath.WalkDir(sourceRoot, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(sourceRoot, path)
		if err != nil {
			return err
		}
		target := filepath.Join(targetRoot, relative)
		if entry.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, content, 0o600)
	})
	if err != nil {
		t.Fatal(err)
	}
}
