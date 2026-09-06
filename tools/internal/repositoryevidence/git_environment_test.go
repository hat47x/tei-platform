package repositoryevidence

import (
	"fmt"
	"reflect"
	"testing"
)

func TestValidateGitSourceEnvironmentRejectsCommandScopeConfigInjection(t *testing.T) {
	cases := []string{
		"GIT_CONFIG_COUNT=1",
		"git_config_count=1",
		"GIT_CONFIG_PARAMETERS='status.showUntrackedFiles=no'",
		"git_config_parameters='status.showUntrackedFiles=no'",
		"GIT_CONFIG_KEY_0=status.showUntrackedFiles",
		"git_config_key_12=core.worktree",
		"GIT_CONFIG_VALUE_0=no",
		"git_config_value_12=/other/worktree",
	}
	for _, entry := range cases {
		t.Run(entry, func(t *testing.T) {
			if err := validateGitSourceEnvironment([]string{"PATH=/bin", entry}); err == nil {
				t.Fatalf("Git command-scope config injectionを受理しました: %s", entry)
			}
		})
	}
}

func TestValidateGitSourceEnvironmentDoesNotRejectUnrelatedGitEnvironment(t *testing.T) {
	if err := validateGitSourceEnvironment([]string{
		"PATH=/bin",
		"GIT_PAGER=cat",
		"GIT_TERMINAL_PROMPT=0",
	}); err != nil {
		t.Fatalf("source identity / command-scope configを変更しないGit environmentまで拒否しました: %v", err)
	}
}

func TestIndexStateHasOnlyDefaultFlags(t *testing.T) {
	cases := []struct {
		name       string
		indexState string
		wantClean  bool
	}{
		{name: "empty repository", indexState: "", wantClean: true},
		{name: "default tracked entries", indexState: "H go.mod\x00H main.go\x00", wantClean: true},
		{name: "assume unchanged", indexState: "h main.go\x00", wantClean: false},
		{name: "skip worktree", indexState: "S main.go\x00", wantClean: false},
		{name: "assume and skip", indexState: "s main.go\x00", wantClean: false},
		{name: "unexpected nondefault tag", indexState: "M conflicted.go\x00", wantClean: false},
		{name: "malformed output", indexState: "main.go\x00", wantClean: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := indexStateHasOnlyDefaultFlags(tc.indexState); got != tc.wantClean {
				t.Fatalf("indexStateHasOnlyDefaultFlags(%q)=%v want=%v", tc.indexState, got, tc.wantClean)
			}
		})
	}
}

func TestInspectStartObservesUntrackedIgnoredAndIndexLocalState(t *testing.T) {
	cases := []struct {
		name       string
		status     string
		indexState string
		wantClean  bool
	}{
		{name: "clean", status: "", indexState: "H tracked.go\x00", wantClean: true},
		{name: "untracked", status: "?? untracked.txt\n", indexState: "H tracked.go\x00", wantClean: false},
		{name: "ignored", status: "!! ignored.local.go\n", indexState: "H tracked.go\x00", wantClean: false},
		{name: "assume unchanged", status: "", indexState: "h tracked.go\x00", wantClean: false},
		{name: "skip worktree", status: "", indexState: "S tracked.go\x00", wantClean: false},
		{name: "assume and skip", status: "", indexState: "s tracked.go\x00", wantClean: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			source, err := inspectStart(func(name string, args ...string) (string, error) {
				calls++
				switch calls {
				case 1, 4:
					if name != "git" || !reflect.DeepEqual(args, []string{"rev-parse", "HEAD"}) {
						return "", fmt.Errorf("unexpected revision command: %s %#v", name, args)
					}
					return "abc123\n", nil
				case 2:
					if name != "git" || !reflect.DeepEqual(args, []string{"status", "--porcelain", "--untracked-files=all", "--ignored=matching"}) {
						return "", fmt.Errorf("unexpected status command: %s %#v", name, args)
					}
					return tc.status, nil
				case 3:
					if name != "git" || !reflect.DeepEqual(args, []string{"ls-files", "-v", "-z"}) {
						return "", fmt.Errorf("unexpected index command: %s %#v", name, args)
					}
					return tc.indexState, nil
				default:
					return "", fmt.Errorf("unexpected call=%d", calls)
				}
			})
			if err != nil {
				t.Fatalf("inspectStart() error=%v", err)
			}
			if source.WorkingTreeClean != tc.wantClean {
				t.Fatalf("WorkingTreeClean=%v want=%v source=%#v", source.WorkingTreeClean, tc.wantClean, source)
			}
		})
	}
}

func TestInspectPostObservesUntrackedIgnoredAndIndexLocalState(t *testing.T) {
	cases := []struct {
		name       string
		status     string
		indexState string
		wantClean  bool
	}{
		{name: "clean", status: "", indexState: "H tracked.go\x00", wantClean: true},
		{name: "untracked", status: "?? untracked.txt\n", indexState: "H tracked.go\x00", wantClean: false},
		{name: "ignored", status: "!! ignored.local.go\n", indexState: "H tracked.go\x00", wantClean: false},
		{name: "assume unchanged", status: "", indexState: "h tracked.go\x00", wantClean: false},
		{name: "skip worktree", status: "", indexState: "S tracked.go\x00", wantClean: false},
		{name: "assume and skip", status: "", indexState: "s tracked.go\x00", wantClean: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			post, err := inspectPost("abc123", func(name string, args ...string) (string, error) {
				calls++
				switch calls {
				case 1, 4:
					if name != "git" || !reflect.DeepEqual(args, []string{"rev-parse", "HEAD"}) {
						return "", fmt.Errorf("unexpected revision command: %s %#v", name, args)
					}
					return "abc123\n", nil
				case 2:
					if name != "git" || !reflect.DeepEqual(args, []string{"status", "--porcelain", "--untracked-files=all", "--ignored=matching"}) {
						return "", fmt.Errorf("unexpected status command: %s %#v", name, args)
					}
					return tc.status, nil
				case 3:
					if name != "git" || !reflect.DeepEqual(args, []string{"ls-files", "-v", "-z"}) {
						return "", fmt.Errorf("unexpected index command: %s %#v", name, args)
					}
					return tc.indexState, nil
				default:
					return "", fmt.Errorf("unexpected call=%d", calls)
				}
			})
			if err != nil {
				t.Fatalf("inspectPost() error=%v", err)
			}
			if !post.Checked || !post.RevisionStable || post.WorkingTreeClean != tc.wantClean {
				t.Fatalf("post-run local source state観測が不正です: wantClean=%v post=%#v", tc.wantClean, post)
			}
		})
	}
}
