package repositoryevidence

import (
	"errors"
	"testing"
)

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

func TestInspectPostMarksCheckedOnlyAfterBothObservationsSucceed(t *testing.T) {
	t.Run("revision observation fails", func(t *testing.T) {
		post, err := inspectPost("abc123", func(name string, args ...string) (string, error) {
			return "", errors.New("rev-parse failed")
		})
		if err == nil {
			t.Fatal("revision観測失敗をerrorにしていません")
		}
		if post.Checked {
			t.Fatalf("revisionを観測できていないのにChecked=trueです: %#v", post)
		}
	})

	t.Run("working tree observation fails", func(t *testing.T) {
		calls := 0
		post, err := inspectPost("abc123", func(name string, args ...string) (string, error) {
			calls++
			if calls == 1 {
				return "abc123\n", nil
			}
			return "", errors.New("status failed")
		})
		if err == nil {
			t.Fatal("working tree観測失敗をerrorにしていません")
		}
		if post.Checked {
			t.Fatalf("working treeを観測できていないのにChecked=trueです: %#v", post)
		}
		if post.RepositoryRevision != "abc123" || !post.RevisionStable {
			t.Fatalf("取得済みrevision観測まで失っています: %#v", post)
		}
	})

	t.Run("both observations succeed", func(t *testing.T) {
		calls := 0
		post, err := inspectPost("abc123", func(name string, args ...string) (string, error) {
			calls++
			if calls == 1 {
				return "abc123\n", nil
			}
			return "", nil
		})
		if err != nil {
			t.Fatalf("post-run観測に失敗しました: %v", err)
		}
		if !post.Checked || !post.RevisionStable || !post.WorkingTreeClean {
			t.Fatalf("両観測成功後の状態が不正です: %#v", post)
		}
	})

	t.Run("dirty tree is checked but not reproducible", func(t *testing.T) {
		calls := 0
		post, err := inspectPost("abc123", func(name string, args ...string) (string, error) {
			calls++
			if calls == 1 {
				return "abc123\n", nil
			}
			return " M changed.go\n", nil
		})
		if err != nil {
			t.Fatalf("dirty working treeの観測自体は成功するはずです: %v", err)
		}
		if !post.Checked {
			t.Fatalf("両観測が成功しているのにChecked=falseです: %#v", post)
		}
		if post.WorkingTreeClean {
			t.Fatalf("dirty working treeをcleanと判定しました: %#v", post)
		}
	})
}
