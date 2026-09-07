package main

import (
	"path/filepath"
	"reflect"
	"testing"
)

func TestBusinessHandoffRemainsAddressedByProcedureAndStepWhenActorContextChanges(t *testing.T) {
	root := filepath.Join(repoPath(), "reference", "business-system-design")
	set, fixtures := currentReferenceSetAndFixturesD4O1Probe(t, root)

	before, err := deriveCanonicalBusinessHandoffProbes(set, fixtures)
	if err != nil {
		t.Fatal(err)
	}

	changed := make(map[string]fixture, len(fixtures))
	for id, item := range fixtures {
		changed[id] = item
	}
	followUp := changed["order-entry.follow-up"]
	followUp.Procedure.Actor = candidateRef{
		Status: "candidate",
		Label:  "sales follow-up coordinator",
	}
	changed["order-entry.follow-up"] = followUp

	after, err := deriveCanonicalBusinessHandoffProbes(set, changed)
	if err != nil {
		t.Fatalf("target ProcedureのBusiness actor context変更だけでHandoff endpoint解決が壊れました: %v", err)
	}
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("Procedure actor contextをHandoff relation meaningへ複製しました: before=%+v after=%+v", before, after)
	}
}

func TestBusinessHandoffConditionAndTargetProcedureTriggerRemainSeparateAuthorities(t *testing.T) {
	root := filepath.Join(repoPath(), "reference", "business-system-design")
	set, fixtures := currentReferenceSetAndFixturesD4O1Probe(t, root)

	beforeHandoffs, err := deriveCanonicalBusinessHandoffProbes(set, fixtures)
	if err != nil {
		t.Fatal(err)
	}
	beforeTrigger := fixtures["order-entry.follow-up"].Procedure.Trigger

	t.Run("target trigger wording can change without rewriting Handoff", func(t *testing.T) {
		changed := make(map[string]fixture, len(fixtures))
		for id, item := range fixtures {
			changed[id] = item
		}
		followUp := changed["order-entry.follow-up"]
		followUp.Procedure.Trigger = candidateRef{
			Status: "candidate",
			Text:   "an earlier order interaction has finished and follow-up work may now be required",
		}
		changed["order-entry.follow-up"] = followUp

		after, err := deriveCanonicalBusinessHandoffProbes(set, changed)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(beforeHandoffs, after) {
			t.Fatalf("target Procedure trigger wordingをHandoff conditionへ暗黙同期しました: before=%+v after=%+v", beforeHandoffs, after)
		}
	})

	t.Run("Handoff condition can change without rewriting target trigger", func(t *testing.T) {
		changedSet := set
		changedSet.Handoffs = append([]referenceSetHandoff(nil), set.Handoffs...)
		if len(changedSet.Handoffs) == 0 {
			t.Fatal("test setup: Handoffがありません")
		}
		changedSet.Handoffs[0].Condition = "follow-up is required because unresolved surrounding business work remains"

		after, err := deriveCanonicalBusinessHandoffProbes(changedSet, fixtures)
		if err != nil {
			t.Fatal(err)
		}
		if reflect.DeepEqual(beforeHandoffs, after) {
			t.Fatal("Handoff condition changeを独立したBusiness relation meaningとして観測できません")
		}
		if !reflect.DeepEqual(fixtures["order-entry.follow-up"].Procedure.Trigger, beforeTrigger) {
			t.Fatal("Handoff condition変更がtarget Procedure triggerへ逆書きされました")
		}
	})
}
