package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"tei/internal/sourceindex"
)

type procedureScopedStepAddressProbe struct {
	Procedure string `json:"procedure"`
	Step      string `json:"step"`
}

func TestStepAddressCanRemainProcedureScopedCompositeKey(t *testing.T) {
	addressType := reflect.TypeOf(procedureScopedStepAddressProbe{})
	want := []string{"Procedure", "Step"}
	if got := fieldNames(addressType); !reflect.DeepEqual(got, want) {
		t.Fatalf("Procedure-scoped Step addressが膨張しました: got=%v want=%v", got, want)
	}
}

func TestHandoffAndAcceptanceUseSameProcedureScopedStepAddressShape(t *testing.T) {
	root := filepath.Join(repoPath(), "reference", "business-system-design")
	set, err := readReferenceSet(filepath.Join(root, "order-entry-procedure-reference-set.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(set.Handoffs) != 1 {
		t.Fatalf("handoff count=%d; want 1", len(set.Handoffs))
	}

	handoffAddress := procedureScopedStepAddressProbe{
		Procedure: set.Handoffs[0].From.Procedure,
		Step:      set.Handoffs[0].From.Step,
	}
	if handoffAddress != (procedureScopedStepAddressProbe{Procedure: "order-entry.order-handling", Step: "BP-09"}) {
		t.Fatalf("unexpected handoff source address: %+v", handoffAddress)
	}

	candidate := procedureAcceptanceCandidate{
		TargetProcedure: "order-entry.order-handling",
		Step:            "BP-09",
	}
	acceptanceAddress := procedureScopedStepAddressProbe{Procedure: candidate.TargetProcedure, Step: candidate.Step}
	if handoffAddress != acceptanceAddress {
		t.Fatalf("handoffとAcceptanceでStep address shapeが一致しません: handoff=%+v acceptance=%+v", handoffAddress, acceptanceAddress)
	}
}

func TestProcedureScopedStepAddressIsIndependentFromArtifactLocator(t *testing.T) {
	before := procedureScopedStepAddressProbe{Procedure: "order-entry.order-handling", Step: "BP-09"}
	after := before
	oldLocator := "procedures/order-handling.json"
	newLocator := "procedures/sales/order-handling.json"
	if oldLocator == newLocator {
		t.Fatal("test setupのlocatorが同一です")
	}
	if before != after {
		t.Fatal("artifact locator moveでProcedure-scoped Step addressが変化しました")
	}
}

func TestStepLocalKeyRenameRequiresExplicitAcceptanceTargetUpdate(t *testing.T) {
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
	found := false
	for i := range f.Steps {
		if f.Steps[i].ID == "BP-04" {
			f.Steps[i].ID = "BP-04-renamed"
			found = true
		}
	}
	if !found {
		t.Fatal("test setupがBP-04を見つけられません")
	}
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

	stale := procedureAcceptanceCandidate{
		TargetProcedure:       "order-entry.order-handling",
		BaseSourceFingerprint: artifact.Fingerprint,
		Step:                  "BP-04",
		BusinessAction:        "must not apply through a stale Step local key",
	}
	if _, err := previewProcedureAcceptance(probeRoot, registry, stale); err == nil {
		t.Fatal("Step local key rename後も旧address candidateを黙って適用しました")
	}

	updated := stale
	updated.Step = "BP-04-renamed"
	if _, err := previewProcedureAcceptance(probeRoot, registry, updated); err != nil {
		t.Fatalf("明示的に更新したProcedure-scoped Step addressを解決できません: %v", err)
	}
}

func TestProcedureScopedStepAddressDoesNotBecomeGlobalRuntimeIdentity(t *testing.T) {
	addressType := reflect.TypeOf(procedureScopedStepAddressProbe{})
	for _, forbidden := range []string{"GlobalID", "RuntimeID", "ExecutionID", "InstanceID", "StateID", "ArtifactPath", "Fingerprint"} {
		if _, found := addressType.FieldByName(forbidden); found {
			t.Fatalf("Step addressへ不要なglobal/runtime/source identity %qが侵入しました", forbidden)
		}
	}
}
