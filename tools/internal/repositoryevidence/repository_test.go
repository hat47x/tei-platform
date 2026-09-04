package repositoryevidence

import "testing"

func TestIsReproducibleRequiresSameCleanRevisionBeforeAndAfter(t *testing.T) {
	initial := Source{
		RepositoryRevision: "abc123",
		WorkingTreeClean:   true,
	}
	base := PostRun{
		Checked:            true,
		RepositoryRevision: "abc123",
		RevisionStable:     true,
		WorkingTreeClean:   true,
	}
	if !IsReproducible(initial, base) {
		t.Fatal("同一のclean revisionを再現可能と判定できません")
	}

	cases := []struct {
		name    string
		initial Source
		post    PostRun
	}{
		{
			name: "missing initial revision",
			initial: Source{
				WorkingTreeClean: true,
			},
			post: base,
		},
		{
			name: "dirty before run",
			initial: Source{
				RepositoryRevision: "abc123",
				WorkingTreeClean:   false,
			},
			post: base,
		},
		{
			name:    "post run not checked",
			initial: initial,
			post: PostRun{
				RepositoryRevision: "abc123",
				RevisionStable:     true,
				WorkingTreeClean:   true,
			},
		},
		{
			name:    "head changed during run",
			initial: initial,
			post: PostRun{
				Checked:            true,
				RepositoryRevision: "def456",
				RevisionStable:     false,
				WorkingTreeClean:   true,
			},
		},
		{
			name:    "revision string mismatch even if stable flag is wrong",
			initial: initial,
			post: PostRun{
				Checked:            true,
				RepositoryRevision: "def456",
				RevisionStable:     true,
				WorkingTreeClean:   true,
			},
		},
		{
			name:    "dirty after run",
			initial: initial,
			post: PostRun{
				Checked:            true,
				RepositoryRevision: "abc123",
				RevisionStable:     true,
				WorkingTreeClean:   false,
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if IsReproducible(tc.initial, tc.post) {
				t.Fatalf("再現不能なrepository stateを受理しました: initial=%#v post=%#v", tc.initial, tc.post)
			}
		})
	}
}
