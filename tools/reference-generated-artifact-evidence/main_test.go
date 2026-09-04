package main

import (
	"reflect"
	"testing"
)

func TestCheckOnlyCommandsNeverUseWriteMode(t *testing.T) {
	got := checkOnlyCommands()
	want := []commandSpec{
		{
			Name: "world-plugin-lock-golden",
			Args: []string{"run", "./tools/reference-golden"},
		},
		{
			Name: "durable-run-golden",
			Args: []string{"run", "./tools/reference-durable-run-golden"},
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("check-only command=%#v want=%#v", got, want)
	}
	for _, spec := range got {
		for _, arg := range spec.Args {
			if arg == "-write" {
				t.Fatalf("完了Evidence runnerが修復操作-writeを含んでいます: %#v", spec)
			}
		}
	}
}

func TestReproducibleRepositoryStateRequiresStableCleanPostRun(t *testing.T) {
	initial := "abc123"
	base := postRunRepositoryEvidence{
		Checked:            true,
		RepositoryRevision: initial,
		RevisionStable:     true,
		WorkingTreeClean:   true,
	}
	if !isReproducibleRepositoryState(initial, true, base) {
		t.Fatal("cleanかつ同一revisionのrunを再現可能と判定できません")
	}

	cases := []struct {
		name         string
		initial      string
		initialClean bool
		postRun      postRunRepositoryEvidence
	}{
		{"missing-initial-revision", "", true, base},
		{"dirty-before", initial, false, base},
		{"not-checked-after", initial, true, postRunRepositoryEvidence{}},
		{"head-changed", initial, true, postRunRepositoryEvidence{Checked: true, RepositoryRevision: "def456", RevisionStable: false, WorkingTreeClean: true}},
		{"dirty-after", initial, true, postRunRepositoryEvidence{Checked: true, RepositoryRevision: initial, RevisionStable: true, WorkingTreeClean: false}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if isReproducibleRepositoryState(tc.initial, tc.initialClean, tc.postRun) {
				t.Fatalf("再現不能なrepository stateをEvidence候補として受理しました: %#v", tc)
			}
		})
	}
}

func TestCommandDisplayKeepsArgumentBoundaries(t *testing.T) {
	spec := commandSpec{Name: "example", Args: []string{"run", "./tools/reference-golden"}}
	got := commandDisplay(spec)
	want := []string{"go", "run", "./tools/reference-golden"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("command display=%#v want=%#v", got, want)
	}
}
