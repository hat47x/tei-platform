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

type procedureAcceptanceCandidate struct {
	TargetProcedure       string
	BaseSourceFingerprint string
	Step                  string
	BusinessAction        string
}

type procedureAcceptancePreview struct {
	TargetProcedure  string
	SourcePath       string
	BeforeArtifact   sourceindex.Artifact
	BeforeFixture    fixture
	AfterFixture     fixture
	BeforeProjection report
	AfterProjection  report
}

func TestProcedureAcceptancePreviewDoesNotWriteAuthoritativeSource(t *testing.T) {
	root := repoPath()
	probeRoot := prepareProcedureAuthoringWorkspace(t, root)
	registry := map[string]string{"order-entry.order-handling": "procedures/order-handling.json"}
	path := filepath.Join(probeRoot, filepath.FromSlash(registry["order-entry.order-handling"]))
	beforeBytes, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	artifact, err := sourceindex.NewArtifact(registry["order-entry.order-handling"], beforeBytes)
	if err != nil {
		t.Fatal(err)
	}

	candidate := procedureAcceptanceCandidate{
		TargetProcedure:       "order-entry.order-handling",
		BaseSourceFingerprint: artifact.Fingerprint,
		Step:                  "BP-04",
		BusinessAction:        "decide whether surrounding business circumstances permit submission",
	}
	preview, err := previewProcedureAcceptance(probeRoot, registry, candidate)
	if err != nil {
		t.Fatal(err)
	}
	afterBytes, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(beforeBytes, afterBytes) {
		t.Fatal("Previewがauthoritative Business Procedure sourceを書き換えました")
	}
	if reflect.DeepEqual(preview.BeforeProjection.BusinessActionFunctionRows, preview.AfterProjection.BusinessActionFunctionRows) {
		t.Fatal("PreviewにBusiness-side diffが現れていません")
	}
	if !reflect.DeepEqual(preview.BeforeProjection.BusinessActionPatternMeaningRows, preview.AfterProjection.BusinessActionPatternMeaningRows) {
		t.Fatal("Business-only candidateがPattern-derived Application meaningを変更しました")
	}
}

func TestProcedureAcceptanceAcceptWritesThenReloadsWithoutWorldExpansion(t *testing.T) {
	root := repoPath()
	probeRoot := prepareProcedureAuthoringWorkspace(t, root)
	registry := map[string]string{"order-entry.order-handling": "procedures/order-handling.json"}
	relativePath := registry["order-entry.order-handling"]
	initialBytes, err := os.ReadFile(filepath.Join(probeRoot, filepath.FromSlash(relativePath)))
	if err != nil {
		t.Fatal(err)
	}
	initialArtifact, err := sourceindex.NewArtifact(relativePath, initialBytes)
	if err != nil {
		t.Fatal(err)
	}
	worldBefore, err := workspace.Load(probeRoot)
	if err != nil {
		t.Fatal(err)
	}

	candidate := procedureAcceptanceCandidate{
		TargetProcedure:       "order-entry.order-handling",
		BaseSourceFingerprint: initialArtifact.Fingerprint,
		Step:                  "BP-04",
		BusinessAction:        "decide whether surrounding business circumstances permit submission",
	}
	accepted, err := acceptProcedureAcceptance(probeRoot, registry, candidate)
	if err != nil {
		t.Fatal(err)
	}

	reloaded, err := workspace.Load(probeRoot)
	if err != nil {
		t.Fatal(err)
	}
	reloadedSource, err := acceptProcedureSource(probeRoot, relativePath, &reloaded, accepted.AfterArtifact.Fingerprint)
	if err != nil {
		t.Fatal(err)
	}
	reloadedProjection := evaluate(reloaded.World, reloadedSource.Fixture)

	if !reflect.DeepEqual(worldBefore.World, reloaded.World) {
		t.Fatal("Business Procedure Accept後のreloadでCanonical Worldが変化しました")
	}
	if reloadedSource.Fixture.Procedure.ID != candidate.TargetProcedure {
		t.Fatalf("reload後Procedure key=%q; want %q", reloadedSource.Fixture.Procedure.ID, candidate.TargetProcedure)
	}
	if !reflect.DeepEqual(accepted.AfterProjection, reloadedProjection) {
		t.Fatal("Accept後reloadでderived Projectionを再現できません")
	}
	if accepted.BeforeArtifact.Fingerprint == accepted.AfterArtifact.Fingerprint {
		t.Fatal("Accept後もsource fingerprintが変化していません")
	}
}

func TestProcedureAcceptanceRejectLeavesSourceUntouched(t *testing.T) {
	root := repoPath()
	probeRoot := prepareProcedureAuthoringWorkspace(t, root)
	registry := map[string]string{"order-entry.order-handling": "procedures/order-handling.json"}
	relativePath := registry["order-entry.order-handling"]
	path := filepath.Join(probeRoot, filepath.FromSlash(relativePath))
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	artifact, err := sourceindex.NewArtifact(relativePath, before)
	if err != nil {
		t.Fatal(err)
	}
	candidate := procedureAcceptanceCandidate{
		TargetProcedure:       "order-entry.order-handling",
		BaseSourceFingerprint: artifact.Fingerprint,
		Step:                  "BP-04",
		BusinessAction:        "candidate text that will be rejected",
	}
	if _, err := previewProcedureAcceptance(probeRoot, registry, candidate); err != nil {
		t.Fatal(err)
	}

	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, after) {
		t.Fatal("Reject相当のPreview-only経路でsourceが変化しました")
	}
}

func TestProcedureAcceptanceRejectsStaleCandidate(t *testing.T) {
	root := repoPath()
	probeRoot := prepareProcedureAuthoringWorkspace(t, root)
	registry := map[string]string{"order-entry.order-handling": "procedures/order-handling.json"}
	relativePath := registry["order-entry.order-handling"]
	path := filepath.Join(probeRoot, filepath.FromSlash(relativePath))
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	artifact, err := sourceindex.NewArtifact(relativePath, before)
	if err != nil {
		t.Fatal(err)
	}
	candidate := procedureAcceptanceCandidate{
		TargetProcedure:       "order-entry.order-handling",
		BaseSourceFingerprint: artifact.Fingerprint,
		Step:                  "BP-04",
		BusinessAction:        "candidate based on stale source",
	}

	f, err := decodeFixtureBytes(before)
	if err != nil {
		t.Fatal(err)
	}
	for i := range f.Steps {
		if f.Steps[i].ID == "BP-09" {
			f.Steps[i].BusinessAction = "another accepted edit happened first"
		}
	}
	changed, err := marshalFixture(f)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, changed, 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := previewProcedureAcceptance(probeRoot, registry, candidate); err == nil {
		t.Fatal("stale candidateをPreview可能として扱いました")
	}
}

func TestProcedureAcceptanceTargetSurvivesFileMoveThroughRegistry(t *testing.T) {
	root := repoPath()
	probeRoot := prepareProcedureAuthoringWorkspace(t, root)
	registry := map[string]string{"order-entry.order-handling": "procedures/order-handling.json"}
	oldRel := registry["order-entry.order-handling"]
	oldPath := filepath.Join(probeRoot, filepath.FromSlash(oldRel))
	content, err := os.ReadFile(oldPath)
	if err != nil {
		t.Fatal(err)
	}
	artifact, err := sourceindex.NewArtifact(oldRel, content)
	if err != nil {
		t.Fatal(err)
	}

	newRel := "procedures/sales/order-handling.json"
	newPath := filepath.Join(probeRoot, filepath.FromSlash(newRel))
	if err := os.MkdirAll(filepath.Dir(newPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(oldPath, newPath); err != nil {
		t.Fatal(err)
	}
	registry["order-entry.order-handling"] = newRel

	candidate := procedureAcceptanceCandidate{
		TargetProcedure:       "order-entry.order-handling",
		BaseSourceFingerprint: artifact.Fingerprint,
		Step:                  "BP-04",
		BusinessAction:        "candidate after a pure source move",
	}
	preview, err := previewProcedureAcceptance(probeRoot, registry, candidate)
	if err != nil {
		t.Fatal(err)
	}
	if preview.SourcePath != newRel {
		t.Fatalf("Procedure keyからmove後locatorを解決できません: got %q want %q", preview.SourcePath, newRel)
	}
	if preview.BeforeArtifact.Fingerprint != artifact.Fingerprint {
		t.Fatal("pure file moveでcandidate base fingerprintが変化しました")
	}
}

func TestProcedureAcceptanceLogicalRenameRequiresExplicitTargetUpdate(t *testing.T) {
	root := repoPath()
	probeRoot := prepareProcedureAuthoringWorkspace(t, root)
	registry := map[string]string{"order-entry.order-handling": "procedures/order-handling.json"}
	relativePath := registry["order-entry.order-handling"]
	path := filepath.Join(probeRoot, filepath.FromSlash(relativePath))
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	f, err := decodeFixtureBytes(content)
	if err != nil {
		t.Fatal(err)
	}
	f.Procedure.ID = "order-entry.order-handling-renamed"
	renamed, err := marshalFixture(f)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, renamed, 0o600); err != nil {
		t.Fatal(err)
	}
	artifact, err := sourceindex.NewArtifact(relativePath, renamed)
	if err != nil {
		t.Fatal(err)
	}

	candidate := procedureAcceptanceCandidate{
		TargetProcedure:       "order-entry.order-handling",
		BaseSourceFingerprint: artifact.Fingerprint,
		Step:                  "BP-04",
		BusinessAction:        "must not target renamed Procedure implicitly",
	}
	if _, err := previewProcedureAcceptance(probeRoot, registry, candidate); err == nil {
		t.Fatal("logical Procedure rename後も旧key candidateを黙って適用可能にしました")
	}
}

type acceptedProcedureChange struct {
	BeforeArtifact   sourceindex.Artifact
	AfterArtifact    sourceindex.Artifact
	BeforeProjection report
	AfterProjection  report
}

func previewProcedureAcceptance(root string, registry map[string]string, candidate procedureAcceptanceCandidate) (procedureAcceptancePreview, error) {
	relativePath, ok := registry[candidate.TargetProcedure]
	if !ok {
		return procedureAcceptancePreview{}, fmt.Errorf("Business Procedure target %q is not registered", candidate.TargetProcedure)
	}
	content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(relativePath)))
	if err != nil {
		return procedureAcceptancePreview{}, err
	}
	artifact, err := sourceindex.NewArtifact(relativePath, content)
	if err != nil {
		return procedureAcceptancePreview{}, err
	}
	if candidate.BaseSourceFingerprint != artifact.Fingerprint {
		return procedureAcceptancePreview{}, fmt.Errorf("stale Business Procedure candidate: base=%s current=%s", candidate.BaseSourceFingerprint, artifact.Fingerprint)
	}
	f, err := decodeFixtureBytes(content)
	if err != nil {
		return procedureAcceptancePreview{}, err
	}
	if f.Procedure.ID != candidate.TargetProcedure {
		return procedureAcceptancePreview{}, fmt.Errorf("Business Procedure registry target %q resolves source identity %q", candidate.TargetProcedure, f.Procedure.ID)
	}
	changed := f
	changed.Steps = append(changed.Steps[:0:0], f.Steps...)
	found := false
	for i := range changed.Steps {
		if changed.Steps[i].ID == candidate.Step {
			changed.Steps[i].BusinessAction = candidate.BusinessAction
			found = true
		}
	}
	if !found {
		return procedureAcceptancePreview{}, fmt.Errorf("Business Procedure step %q/%q not found", candidate.TargetProcedure, candidate.Step)
	}
	loaded, err := workspace.Load(root)
	if err != nil {
		return procedureAcceptancePreview{}, err
	}
	return procedureAcceptancePreview{
		TargetProcedure:  candidate.TargetProcedure,
		SourcePath:       relativePath,
		BeforeArtifact:   artifact,
		BeforeFixture:    f,
		AfterFixture:     changed,
		BeforeProjection: evaluate(loaded.World, f),
		AfterProjection:  evaluate(loaded.World, changed),
	}, nil
}

func acceptProcedureAcceptance(root string, registry map[string]string, candidate procedureAcceptanceCandidate) (acceptedProcedureChange, error) {
	preview, err := previewProcedureAcceptance(root, registry, candidate)
	if err != nil {
		return acceptedProcedureChange{}, err
	}
	content, err := marshalFixture(preview.AfterFixture)
	if err != nil {
		return acceptedProcedureChange{}, err
	}
	path := filepath.Join(root, filepath.FromSlash(preview.SourcePath))
	current, err := os.ReadFile(path)
	if err != nil {
		return acceptedProcedureChange{}, err
	}
	currentArtifact, err := sourceindex.NewArtifact(preview.SourcePath, current)
	if err != nil {
		return acceptedProcedureChange{}, err
	}
	if currentArtifact.Fingerprint != preview.BeforeArtifact.Fingerprint {
		return acceptedProcedureChange{}, fmt.Errorf("Business Procedure source changed after Preview")
	}
	if err := os.WriteFile(path, content, 0o600); err != nil {
		return acceptedProcedureChange{}, err
	}
	afterArtifact, err := sourceindex.NewArtifact(preview.SourcePath, content)
	if err != nil {
		return acceptedProcedureChange{}, err
	}
	return acceptedProcedureChange{
		BeforeArtifact:   preview.BeforeArtifact,
		AfterArtifact:    afterArtifact,
		BeforeProjection: preview.BeforeProjection,
		AfterProjection:  preview.AfterProjection,
	}, nil
}

func decodeFixtureBytes(content []byte) (fixture, error) {
	return decodeFixture(strings.NewReader(string(content)))
}

func marshalFixture(f fixture) ([]byte, error) {
	content, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(content, '\n'), nil
}
