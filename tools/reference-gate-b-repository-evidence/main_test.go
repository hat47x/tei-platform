package main

import (
	"reflect"
	"testing"

	"tei/tools/internal/repositoryevidence"
)

func TestInitialRepositorySourceNeverClaimsReproducibleRevision(t *testing.T) {
	result := evidence{ReproducibleRevision: true}
	source := repositoryevidence.Source{
		RepositoryRevision: "abc123",
		WorkingTreeClean:   true,
	}

	recordInitialRepositorySource(&result, source)

	if result.RepositoryRevision != source.RepositoryRevision {
		t.Fatalf("repository revision=%q want=%q", result.RepositoryRevision, source.RepositoryRevision)
	}
	if !result.WorkingTreeClean {
		t.Fatal("開始時clean状態を記録できていません")
	}
	if result.ReproducibleRevision {
		t.Fatal("post-run確認前にreproducibleRevision=trueを記録しています")
	}
}

func TestTargetedChecksKeepGateBMinimumEvidenceSeparate(t *testing.T) {
	got := targetedChecks()
	if len(got) != 3 {
		t.Fatalf("targeted checks=%d want=3", len(got))
	}

	wantScenario := commandSpec{
		Name: "final-ui-less-scenario",
		Args: []string{
			"test", "./internal/driver",
			"-run", "^TestOrderEntryGateBScenarioConnectsMajorBoundariesWithoutUI$",
			"-count=1",
		},
	}
	if !reflect.DeepEqual(got[0], wantScenario) {
		t.Fatalf("final Scenario commandがGate B runbookと一致しません: %#v", got[0])
	}

	wantExternalEvidence := commandSpec{
		Name: "external-evidence-intake",
		Args: []string{
			"test",
			"./internal/database/externalqueryevidence",
			"./tools/reference-external-query-evidence",
			"-count=1",
		},
	}
	if !reflect.DeepEqual(got[1], wantExternalEvidence) {
		t.Fatalf("external Evidence intake commandがGate B runbookと一致しません: %#v", got[1])
	}

	wantRepositoryEvidenceBoundary := commandSpec{
		Name: "repository-evidence-source-boundary",
		Args: []string{
			"test",
			"./tools/internal/repositoryevidence",
			"-count=1",
		},
	}
	if !reflect.DeepEqual(got[2], wantRepositoryEvidenceBoundary) {
		t.Fatalf("Repository Evidence source boundary commandがGate B targeted Evidenceと一致しません: %#v", got[2])
	}
}

func TestBroadCheckIsExplicitRepositoryWideTest(t *testing.T) {
	got := broadCheck()
	want := commandSpec{
		Name: "repository-wide-go-test",
		Args: []string{"test", "./..."},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("broad check=%#v want=%#v", got, want)
	}
}

func TestCommandDisplayDoesNotUseShellString(t *testing.T) {
	spec := commandSpec{Name: "example", Args: []string{"test", "./internal/driver"}}
	got := commandDisplay(spec)
	want := []string{"go", "test", "./internal/driver"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("command display=%#v want=%#v", got, want)
	}
}

func TestReproducibleRepositoryStateRequiresStableCleanSourceBeforeAndAfter(t *testing.T) {
	tests := []struct {
		name            string
		initialRevision string
		initialClean    bool
		postRun         postRunRepositoryEvidence
		want            bool
	}{
		{
			name:            "stable clean revision",
			initialRevision: "abc123",
			initialClean:    true,
			postRun: postRunRepositoryEvidence{
				Checked:            true,
				RepositoryRevision: "abc123",
				RevisionStable:     true,
				WorkingTreeClean:   true,
			},
			want: true,
		},
		{
			name:            "dirty before run",
			initialRevision: "abc123",
			initialClean:    false,
			postRun: postRunRepositoryEvidence{
				Checked:            true,
				RepositoryRevision: "abc123",
				RevisionStable:     true,
				WorkingTreeClean:   true,
			},
			want: false,
		},
		{
			name:            "head changed during run",
			initialRevision: "abc123",
			initialClean:    true,
			postRun: postRunRepositoryEvidence{
				Checked:            true,
				RepositoryRevision: "def456",
				RevisionStable:     false,
				WorkingTreeClean:   true,
			},
			want: false,
		},
		{
			name:            "working tree changed during run",
			initialRevision: "abc123",
			initialClean:    true,
			postRun: postRunRepositoryEvidence{
				Checked:            true,
				RepositoryRevision: "abc123",
				RevisionStable:     true,
				WorkingTreeClean:   false,
			},
			want: false,
		},
		{
			name:            "post run state was not checked",
			initialRevision: "abc123",
			initialClean:    true,
			postRun: postRunRepositoryEvidence{
				Checked:            false,
				RepositoryRevision: "abc123",
				RevisionStable:     true,
				WorkingTreeClean:   true,
			},
			want: false,
		},
		{
			name:            "initial revision missing",
			initialRevision: "",
			initialClean:    true,
			postRun: postRunRepositoryEvidence{
				Checked:            true,
				RepositoryRevision: "abc123",
				RevisionStable:     true,
				WorkingTreeClean:   true,
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isReproducibleRepositoryState(tt.initialRevision, tt.initialClean, tt.postRun)
			if got != tt.want {
				t.Fatalf("isReproducibleRepositoryState()=%v want=%v", got, tt.want)
			}
		})
	}
}
