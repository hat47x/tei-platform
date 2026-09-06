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

func TestInspectStartForcesAllUntrackedFiles(t *testing.T) {
	calls := 0
	source, err := inspectStart(func(name string, args ...string) (string, error) {
		calls++
		switch calls {
		case 1, 3:
			if name != "git" || !reflect.DeepEqual(args, []string{"rev-parse", "HEAD"}) {
				return "", fmt.Errorf("unexpected revision command: %s %#v", name, args)
			}
			return "abc123\n", nil
		case 2:
			if name != "git" || !reflect.DeepEqual(args, []string{"status", "--porcelain", "--untracked-files=all"}) {
				return "", fmt.Errorf("unexpected status command: %s %#v", name, args)
			}
			return "?? untracked.txt\n", nil
		default:
			return "", fmt.Errorf("unexpected call=%d", calls)
		}
	})
	if err != nil {
		t.Fatalf("inspectStart() error=%v", err)
	}
	if source.WorkingTreeClean {
		t.Fatalf("明示的に観測したuntracked fileをcleanと判定しました: %#v", source)
	}
}

func TestInspectPostForcesAllUntrackedFiles(t *testing.T) {
	calls := 0
	post, err := inspectPost("abc123", func(name string, args ...string) (string, error) {
		calls++
		switch calls {
		case 1, 3:
			if name != "git" || !reflect.DeepEqual(args, []string{"rev-parse", "HEAD"}) {
				return "", fmt.Errorf("unexpected revision command: %s %#v", name, args)
			}
			return "abc123\n", nil
		case 2:
			if name != "git" || !reflect.DeepEqual(args, []string{"status", "--porcelain", "--untracked-files=all"}) {
				return "", fmt.Errorf("unexpected status command: %s %#v", name, args)
			}
			return "?? untracked.txt\n", nil
		default:
			return "", fmt.Errorf("unexpected call=%d", calls)
		}
	})
	if err != nil {
		t.Fatalf("inspectPost() error=%v", err)
	}
	if !post.Checked || !post.RevisionStable || post.WorkingTreeClean {
		t.Fatalf("post-run untracked file観測が不正です: %#v", post)
	}
}
