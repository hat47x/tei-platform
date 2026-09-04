package main

import "testing"

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
			name:            "post run state not checked",
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
			name:            "post run revision mismatched even if flag is wrong",
			initialRevision: "abc123",
			initialClean:    true,
			postRun: postRunRepositoryEvidence{
				Checked:            true,
				RepositoryRevision: "def456",
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
