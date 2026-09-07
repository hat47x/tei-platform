package main

import (
	"path/filepath"
	"reflect"
	"testing"

	"tei/internal/application"
	"tei/internal/ir"
	"tei/internal/workspace"
)

func TestBusinessModelCanBeValidatedAgainstWorldWithoutExpandingIRWorld(t *testing.T) {
	worldType := reflect.TypeOf(ir.World{})
	for _, forbidden := range []string{"Business", "BusinessModel", "Procedures", "Handoffs", "BusinessProcedures", "BusinessHandoffs"} {
		if _, found := worldType.FieldByName(forbidden); found {
			t.Fatalf("current pressureだけでir.WorldへBusiness field %qを追加しました", forbidden)
		}
	}

	root := repoPath()
	loaded, err := workspace.Load(filepath.Join(root, "reference", "workspace"))
	if err != nil {
		t.Fatal(err)
	}
	setRoot := filepath.Join(root, "reference", "business-system-design")
	set, fixtures := currentReferenceSetAndFixturesD4O1Probe(t, setRoot)
	model, err := deriveD4O1NormalizedBusinessModelProbe(set, fixtures, currentProtectedTargetsD4O1Probe())
	if err != nil {
		t.Fatal(err)
	}

	before := loaded.World
	if err := validateD4O1NormalizedBusinessModelProbe(loaded.World, model); err != nil {
		t.Fatalf("separate Business modelをcurrent Worldへcross-view validateできません: %v", err)
	}
	if !reflect.DeepEqual(before, loaded.World) {
		t.Fatal("Business cross-view validationがir.Worldをmutationしました")
	}
}

func TestBusinessMeaningCanChangeWithoutPretendingRuntimeIRChanged(t *testing.T) {
	root := repoPath()
	loaded, err := workspace.Load(filepath.Join(root, "reference", "workspace"))
	if err != nil {
		t.Fatal(err)
	}
	setRoot := filepath.Join(root, "reference", "business-system-design")
	set, fixtures := currentReferenceSetAndFixturesD4O1Probe(t, setRoot)

	beforeModel, err := deriveD4O1NormalizedBusinessModelProbe(set, fixtures, currentProtectedTargetsD4O1Probe())
	if err != nil {
		t.Fatal(err)
	}
	changedFixtures := make(map[string]fixture, len(fixtures))
	for id, item := range fixtures {
		changedFixtures[id] = item
	}
	primary := changedFixtures["order-entry.order-handling"]
	primary.Steps = append([]step(nil), primary.Steps...)
	found := false
	for i := range primary.Steps {
		if primary.Steps[i].ID == "BP-04" {
			primary.Steps[i].BusinessAction = "decide whether the order may proceed after considering unresolved surrounding business circumstances"
			found = true
		}
	}
	if !found {
		t.Fatal("test setup: BP-04がありません")
	}
	changedFixtures["order-entry.order-handling"] = primary
	afterModel, err := deriveD4O1NormalizedBusinessModelProbe(set, changedFixtures, currentProtectedTargetsD4O1Probe())
	if err != nil {
		t.Fatal(err)
	}

	if semanticFingerprintD4O1Probe(t, beforeModel) == semanticFingerprintD4O1Probe(t, afterModel) {
		t.Fatal("Business-only meaning changeをseparate Business semantic revisionで観測できません")
	}
	beforeWorld := loaded.World
	if err := validateD4O1NormalizedBusinessModelProbe(loaded.World, afterModel); err != nil {
		t.Fatalf("Business-only meaning change後のcross-view validationが不正です: %v", err)
	}
	if !reflect.DeepEqual(beforeWorld, loaded.World) {
		t.Fatal("Business-only meaning changeをir.World changeへ誤変換しました")
	}
}

func TestProtectedTargetMeaningCanChangeWithoutPretendingRuntimeIRChanged(t *testing.T) {
	root := repoPath()
	loaded, err := workspace.Load(filepath.Join(root, "reference", "workspace"))
	if err != nil {
		t.Fatal(err)
	}
	setRoot := filepath.Join(root, "reference", "business-system-design")
	set, fixtures := currentReferenceSetAndFixturesD4O1Probe(t, setRoot)

	before, err := deriveD4O1NormalizedBusinessModelProbe(set, fixtures, currentProtectedTargetsD4O1Probe())
	if err != nil {
		t.Fatal(err)
	}
	changedTargets := currentProtectedTargetsD4O1Probe()
	changedTargets["order-entry.order-handling/BP-06"] = []string{"order_entry.order_record"}
	after, err := deriveD4O1NormalizedBusinessModelProbe(set, fixtures, changedTargets)
	if err != nil {
		t.Fatal(err)
	}
	if semanticFingerprintD4O1Probe(t, before) == semanticFingerprintD4O1Probe(t, after) {
		t.Fatal("protected-target-only changeをBusiness semantic revisionが見逃しました")
	}

	beforeWorld := loaded.World
	if err := validateD4O1NormalizedBusinessModelProbe(loaded.World, after); err != nil {
		t.Fatalf("exactな別Application Pattern targetへのBusiness-only changeをcross-view validateできません: %v", err)
	}
	if !reflect.DeepEqual(beforeWorld, loaded.World) {
		t.Fatal("protected-target-only changeをir.World changeへ誤変換しました")
	}
}

func TestHandoffMeaningCanChangeWithoutPretendingRuntimeIRChanged(t *testing.T) {
	root := repoPath()
	loaded, err := workspace.Load(filepath.Join(root, "reference", "workspace"))
	if err != nil {
		t.Fatal(err)
	}
	setRoot := filepath.Join(root, "reference", "business-system-design")
	set, fixtures := currentReferenceSetAndFixturesD4O1Probe(t, setRoot)

	before, err := deriveD4O1NormalizedBusinessModelProbe(set, fixtures, currentProtectedTargetsD4O1Probe())
	if err != nil {
		t.Fatal(err)
	}
	changedSet := set
	changedSet.Handoffs = append([]referenceSetHandoff(nil), set.Handoffs...)
	if len(changedSet.Handoffs) == 0 {
		t.Fatal("test setup: Handoffがありません")
	}
	changedSet.Handoffs[0].Condition = "follow-up is required because unresolved external work remains"
	after, err := deriveD4O1NormalizedBusinessModelProbe(changedSet, fixtures, currentProtectedTargetsD4O1Probe())
	if err != nil {
		t.Fatal(err)
	}
	if semanticFingerprintD4O1Probe(t, before) == semanticFingerprintD4O1Probe(t, after) {
		t.Fatal("handoff-only changeをBusiness semantic revisionが見逃しました")
	}

	beforeWorld := loaded.World
	if err := validateD4O1NormalizedBusinessModelProbe(loaded.World, after); err != nil {
		t.Fatalf("handoff-only Business meaning change後のcross-view validationが不正です: %v", err)
	}
	if !reflect.DeepEqual(beforeWorld, loaded.World) {
		t.Fatal("handoff-only Business meaning changeをir.World changeへ誤変換しました")
	}
}

func TestSeparateBusinessPlaneStillFailsClosedAgainstStaleApplicationTarget(t *testing.T) {
	root := repoPath()
	loaded, err := workspace.Load(filepath.Join(root, "reference", "workspace"))
	if err != nil {
		t.Fatal(err)
	}
	setRoot := filepath.Join(root, "reference", "business-system-design")
	set, fixtures := currentReferenceSetAndFixturesD4O1Probe(t, setRoot)
	model, err := deriveD4O1NormalizedBusinessModelProbe(set, fixtures, currentProtectedTargetsD4O1Probe())
	if err != nil {
		t.Fatal(err)
	}

	changed := loaded.World
	changed.Applications = append([]application.Application(nil), loaded.World.Applications...)
	removed := false
	for i := range changed.Applications {
		if string(changed.Applications[i].ID) != "order_entry" {
			continue
		}
		original := changed.Applications[i].Patterns
		patterns := make([]application.Pattern, 0, len(original))
		for _, pattern := range original {
			if string(pattern.ID) == "order_entry.submit_order" {
				removed = true
				continue
			}
			patterns = append(patterns, pattern)
		}
		changed.Applications[i].Patterns = patterns
	}
	if !removed {
		t.Fatal("test setup: order_entry.submit_order Patternがありません")
	}

	if err := validateD4O1NormalizedBusinessModelProbe(changed, model); err == nil {
		t.Fatal("separate Business planeがstale protected Application targetを見逃しました")
	}
}
