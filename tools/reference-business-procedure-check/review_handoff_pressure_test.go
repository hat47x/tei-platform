package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"tei/internal/sourceindex"
)

const (
	referenceSetSchemaVersion = "business-procedure-reference-set-v0"
	referenceSetArtifactRole  = "reference-authoring-set-fixture"
	referenceLocalStable      = "reference-local-stable"
)

type referenceSet struct {
	SchemaVersion string                  `json:"schemaVersion"`
	ArtifactRole  string                  `json:"artifactRole"`
	Procedures    []referenceSetProcedure `json:"procedures"`
	Handoffs      []referenceSetHandoff   `json:"handoffs"`
}

type referenceSetProcedure struct {
	ID       string `json:"id"`
	Artifact string `json:"artifact"`
}

type referenceSetHandoff struct {
	From      referenceSetEndpoint `json:"from"`
	To        referenceSetEndpoint `json:"to"`
	Condition string               `json:"condition"`
}

type referenceSetEndpoint struct {
	Procedure string `json:"procedure"`
	Step      string `json:"step"`
}

type reviewStamp struct {
	TargetProcedure     string
	RepositoryRevision  string
	ArtifactPath        string
	ArtifactFingerprint string
	Decision            string
}

func TestReferenceLocalStableProcedureKeySupportsHandoff(t *testing.T) {
	root := filepath.Join(repoPath(), "reference", "business-system-design")
	set, err := readReferenceSet(filepath.Join(root, "order-entry-procedure-reference-set.json"))
	if err != nil {
		t.Fatal(err)
	}
	fixtures, err := resolveReferenceSet(root, set)
	if err != nil {
		t.Fatal(err)
	}
	if err := validateReferenceSetHandoffs(set, fixtures); err != nil {
		t.Fatalf("reference-local Procedure keyでhandoffを解決できません: %v", err)
	}

	for id, f := range fixtures {
		if f.Canonical {
			t.Fatalf("%sがCanonical Procedureになっています", id)
		}
		if f.Procedure.IdentityStatus != referenceLocalStable {
			t.Fatalf("%s identityStatus=%q; want %q", id, f.Procedure.IdentityStatus, referenceLocalStable)
		}
	}
}

func TestHandoffSurvivesArtifactMoveWhenProcedureKeysStayStable(t *testing.T) {
	root := filepath.Join(repoPath(), "reference", "business-system-design")
	set, err := readReferenceSet(filepath.Join(root, "order-entry-procedure-reference-set.json"))
	if err != nil {
		t.Fatal(err)
	}
	before := handoffSignatures(set)

	temp := t.TempDir()
	moved := set
	moved.Procedures = append([]referenceSetProcedure(nil), set.Procedures...)
	for i := range moved.Procedures {
		sourcePath := filepath.Join(root, moved.Procedures[i].Artifact)
		content, err := os.ReadFile(sourcePath)
		if err != nil {
			t.Fatal(err)
		}
		movedName := fmt.Sprintf("moved-%d.json", i)
		if err := os.WriteFile(filepath.Join(temp, movedName), content, 0o600); err != nil {
			t.Fatal(err)
		}
		moved.Procedures[i].Artifact = movedName
	}

	fixtures, err := resolveReferenceSet(temp, moved)
	if err != nil {
		t.Fatal(err)
	}
	if err := validateReferenceSetHandoffs(moved, fixtures); err != nil {
		t.Fatalf("artifact move後にhandoffが壊れました: %v", err)
	}
	if !reflect.DeepEqual(before, handoffSignatures(moved)) {
		t.Fatalf("artifact locator変更でhandoff semantic endpointが変化しました: before=%v after=%v", before, handoffSignatures(moved))
	}
}

func TestHandoffFailsClosedAfterLogicalProcedureRenameWithoutLinkUpdate(t *testing.T) {
	root := filepath.Join(repoPath(), "reference", "business-system-design")
	set, err := readReferenceSet(filepath.Join(root, "order-entry-procedure-reference-set.json"))
	if err != nil {
		t.Fatal(err)
	}
	fixtures, err := resolveReferenceSet(root, set)
	if err != nil {
		t.Fatal(err)
	}

	followUp := fixtures["order-entry.follow-up"]
	followUp.Procedure.ID = "order-entry.follow-up-renamed"
	delete(fixtures, "order-entry.follow-up")
	fixtures[followUp.Procedure.ID] = followUp

	if err := validateReferenceSetHandoffs(set, fixtures); err == nil {
		t.Fatal("logical Procedure rename後もstale handoffを黙って解決しました")
	}
}

func TestReviewTargetContinuesAcrossContentChangeButApprovalDoesNotCarryForward(t *testing.T) {
	root := filepath.Join(repoPath(), "reference", "business-system-design")
	path := filepath.Join(root, "order-entry-business-procedure.json")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	f, err := readFixture(path)
	if err != nil {
		t.Fatal(err)
	}

	before := newReviewStamp(f.Procedure.ID, "R1", filepath.Base(path), content, "accepted")
	changed := false
	for i := range f.Steps {
		if f.Steps[i].ID == "BP-04" {
			f.Steps[i].BusinessAction = "decide whether surrounding business circumstances permit submission"
			changed = true
		}
	}
	if !changed {
		t.Fatal("test setupがBP-04を見つけられません")
	}
	afterContent, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	after := newReviewStamp(f.Procedure.ID, "R2", "moved-order-entry-business-procedure.json", afterContent, "pending")

	if before.TargetProcedure != after.TargetProcedure {
		t.Fatalf("Business authoring target continuityが失われました: before=%s after=%s", before.TargetProcedure, after.TargetProcedure)
	}
	if before.ArtifactFingerprint == after.ArtifactFingerprint {
		t.Fatal("content changeをfingerprint差として観測できていません")
	}
	if reviewApprovalAppliesTo(before, after) {
		t.Fatal("旧revision / fingerprintのapprovalを変更後artifactへ自動carry-forwardしました")
	}
}

func TestReviewTargetSurvivesPureFileMoveButEvidenceRemainsRevisionBound(t *testing.T) {
	root := filepath.Join(repoPath(), "reference", "business-system-design")
	path := filepath.Join(root, "order-entry-business-procedure.json")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	f, err := readFixture(path)
	if err != nil {
		t.Fatal(err)
	}

	before := newReviewStamp(f.Procedure.ID, "R1", filepath.Base(path), content, "accepted")
	after := newReviewStamp(f.Procedure.ID, "R2", "moved-order-entry-business-procedure.json", content, "pending")

	if before.TargetProcedure != after.TargetProcedure {
		t.Fatal("pure file moveでreview targetが変化しました")
	}
	if before.ArtifactFingerprint != after.ArtifactFingerprint {
		t.Fatal("pure file moveでcontent fingerprintが変化しました")
	}
	if reviewApprovalAppliesTo(before, after) {
		t.Fatal("同一contentでも別repository revisionへapprovalを暗黙carry-forwardしました")
	}
}

func TestLogicalProcedureRenameCreatesDifferentReviewTarget(t *testing.T) {
	before := reviewStamp{TargetProcedure: "order-entry.follow-up", RepositoryRevision: "R1", ArtifactFingerprint: "sha256:x", Decision: "accepted"}
	after := reviewStamp{TargetProcedure: "order-entry.follow-up-renamed", RepositoryRevision: "R2", ArtifactFingerprint: "sha256:x", Decision: "pending"}
	if before.TargetProcedure == after.TargetProcedure {
		t.Fatal("logical Procedure renameを同じreview targetとして扱っています")
	}
	if reviewApprovalAppliesTo(before, after) {
		t.Fatal("logical Procedure rename後へapprovalを暗黙carry-forwardしました")
	}
}

func readReferenceSet(path string) (referenceSet, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return referenceSet{}, err
	}
	decoder := json.NewDecoder(bytes.NewReader(content))
	decoder.DisallowUnknownFields()
	var set referenceSet
	if err := decoder.Decode(&set); err != nil {
		return referenceSet{}, err
	}
	if err := requireEOF(decoder); err != nil {
		return referenceSet{}, err
	}
	if set.SchemaVersion != referenceSetSchemaVersion {
		return referenceSet{}, fmt.Errorf("unsupported reference set schemaVersion %q", set.SchemaVersion)
	}
	if set.ArtifactRole != referenceSetArtifactRole {
		return referenceSet{}, fmt.Errorf("unsupported reference set artifactRole %q", set.ArtifactRole)
	}
	return set, nil
}

func resolveReferenceSet(root string, set referenceSet) (map[string]fixture, error) {
	fixtures := make(map[string]fixture, len(set.Procedures))
	for _, declaration := range set.Procedures {
		if _, exists := fixtures[declaration.ID]; exists {
			return nil, fmt.Errorf("duplicate Procedure key %q", declaration.ID)
		}
		f, err := readFixture(filepath.Join(root, declaration.Artifact))
		if err != nil {
			return nil, err
		}
		if f.Canonical {
			return nil, fmt.Errorf("Procedure %q unexpectedly uses Canonical identity", declaration.ID)
		}
		if f.Procedure.ID != declaration.ID {
			return nil, fmt.Errorf("Procedure declaration %q resolves artifact containing %q", declaration.ID, f.Procedure.ID)
		}
		if f.Procedure.IdentityStatus != referenceLocalStable {
			return nil, fmt.Errorf("Procedure %q identityStatus=%q; want %q", declaration.ID, f.Procedure.IdentityStatus, referenceLocalStable)
		}
		fixtures[declaration.ID] = f
	}
	return fixtures, nil
}

func validateReferenceSetHandoffs(set referenceSet, fixtures map[string]fixture) error {
	for _, handoff := range set.Handoffs {
		from, ok := fixtures[handoff.From.Procedure]
		if !ok {
			return fmt.Errorf("handoff source Procedure %q not found", handoff.From.Procedure)
		}
		to, ok := fixtures[handoff.To.Procedure]
		if !ok {
			return fmt.Errorf("handoff target Procedure %q not found", handoff.To.Procedure)
		}
		if !fixtureHasStep(from, handoff.From.Step) {
			return fmt.Errorf("handoff source step %q/%q not found", handoff.From.Procedure, handoff.From.Step)
		}
		if !fixtureHasStep(to, handoff.To.Step) {
			return fmt.Errorf("handoff target step %q/%q not found", handoff.To.Procedure, handoff.To.Step)
		}
		if handoff.Condition == "" {
			return fmt.Errorf("handoff %q/%q -> %q/%q has no Business condition", handoff.From.Procedure, handoff.From.Step, handoff.To.Procedure, handoff.To.Step)
		}
	}
	return nil
}

func fixtureHasStep(f fixture, id string) bool {
	for _, item := range f.Steps {
		if item.ID == id {
			return true
		}
	}
	return false
}

func handoffSignatures(set referenceSet) []string {
	result := make([]string, 0, len(set.Handoffs))
	for _, handoff := range set.Handoffs {
		result = append(result, fmt.Sprintf("%s/%s->%s/%s:%s", handoff.From.Procedure, handoff.From.Step, handoff.To.Procedure, handoff.To.Step, handoff.Condition))
	}
	return result
}

func newReviewStamp(target, revision, artifact string, content []byte, decision string) reviewStamp {
	return reviewStamp{
		TargetProcedure:     target,
		RepositoryRevision:  revision,
		ArtifactPath:        artifact,
		ArtifactFingerprint: sourceindex.Fingerprint(content),
		Decision:            decision,
	}
}

func reviewApprovalAppliesTo(accepted, candidate reviewStamp) bool {
	return accepted.Decision == "accepted" &&
		accepted.TargetProcedure == candidate.TargetProcedure &&
		accepted.RepositoryRevision == candidate.RepositoryRevision &&
		accepted.ArtifactFingerprint == candidate.ArtifactFingerprint
}
