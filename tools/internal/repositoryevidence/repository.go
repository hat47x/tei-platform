package repositoryevidence

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// SourceはEvidence実行開始時のRepository source状態である。
//
// Gate Bへ採用するEvidenceは、後から同じsourceへ戻れる必要があるため、
// HEADだけでなくworking treeがcleanであることも記録する。
type Source struct {
	RepositoryRevision string `json:"repositoryRevision"`
	WorkingTreeClean   bool   `json:"workingTreeClean"`
}

// PostRunはEvidence実行後に再確認したRepository source状態である。
//
// Checkedは、実行後のHEADとworking treeを一つの安定したHEAD上で取得できた場合だけtrueにする。
// 途中までしか観測できなかった状態や、観測中にHEADが変化した状態を「確認済み」として扱わない。
//
// 実行中にHEADまたはworking treeが変化したrunを、開始時revisionのEvidenceとして
// 誤採用しないための非Canonicalな実行Evidenceである。
type PostRun struct {
	Checked            bool   `json:"checked"`
	RepositoryRevision string `json:"repositoryRevision,omitempty"`
	RevisionStable     bool   `json:"revisionStable"`
	WorkingTreeClean   bool   `json:"workingTreeClean"`
}

type commandOutputFunc func(name string, args ...string) (string, error)

var gitSourceIdentityOverrideKeys = []string{
	"GIT_DIR",
	"GIT_WORK_TREE",
	"GIT_COMMON_DIR",
	"GIT_INDEX_FILE",
	"GIT_OBJECT_DIRECTORY",
	"GIT_ALTERNATE_OBJECT_DIRECTORIES",
	"GIT_NAMESPACE",
}

// InspectStartはEvidence実行開始時のHEADとworking tree状態を取得する。
// HEADをworking tree観測の前後で確認し、観測中に切り替わった場合はfail closedにする。
func InspectStart() (Source, error) {
	return InspectStartWithEnv(os.Environ())
}

// InspectStartWithEnvは、callerが明示したenvironmentだけをGit subprocessへ渡して開始時sourceを観測する。
// environmentの意味はcaller側が決める。ただしrepository / work tree / index等を別contextへ差し替える
// Git identity overrideやcommand-scope config injectionは、実行sourceとは異なる状態を誤観測し得るため、
// このpackage自身がfail closedで拒否する。
func InspectStartWithEnv(env []string) (Source, error) {
	if err := validateGitSourceEnvironment(env); err != nil {
		return Source{}, err
	}
	return inspectStart(commandOutputWithEnv(env))
}

func inspectStart(output commandOutputFunc) (Source, error) {
	var result Source

	revisionBefore, err := output("git", "rev-parse", "HEAD")
	if err != nil {
		return result, fmt.Errorf("repository revisionを取得できません: %w", err)
	}
	revisionBefore = strings.TrimSpace(revisionBefore)
	if revisionBefore == "" {
		return result, fmt.Errorf("repository revisionが空です")
	}

	status, err := output("git", "status", "--porcelain", "--untracked-files=all")
	if err != nil {
		return result, fmt.Errorf("working tree状態を取得できません: %w", err)
	}

	revisionAfter, err := output("git", "rev-parse", "HEAD")
	if err != nil {
		return result, fmt.Errorf("working tree観測後のrepository revisionを取得できません: %w", err)
	}
	revisionAfter = strings.TrimSpace(revisionAfter)
	if revisionAfter == "" {
		return result, fmt.Errorf("working tree観測後のrepository revisionが空です")
	}
	if revisionBefore != revisionAfter {
		return result, fmt.Errorf("repository source観測中にHEADが変化しました: before=%s after=%s", revisionBefore, revisionAfter)
	}

	result.RepositoryRevision = revisionAfter
	result.WorkingTreeClean = strings.TrimSpace(status) == ""
	return result, nil
}

// InspectPostはEvidence実行後のHEADとworking tree状態を取得し、開始revisionとの同一性を記録する。
func InspectPost(initialRevision string) (PostRun, error) {
	return InspectPostWithEnv(initialRevision, os.Environ())
}

// InspectPostWithEnvは、callerが明示したenvironmentだけをGit subprocessへ渡して実行後sourceを観測する。
// source同一性の判定契約はInspectPostと同じで、Git identity overrideとcommand-scope config injectionは
// 開始時と同様に拒否する。
func InspectPostWithEnv(initialRevision string, env []string) (PostRun, error) {
	if err := validateGitSourceEnvironment(env); err != nil {
		return PostRun{}, err
	}
	return inspectPost(initialRevision, commandOutputWithEnv(env))
}

func inspectPost(initialRevision string, output commandOutputFunc) (PostRun, error) {
	var result PostRun
	if strings.TrimSpace(initialRevision) == "" {
		return result, fmt.Errorf("比較対象のrepository revisionが空です")
	}

	revisionBefore, err := output("git", "rev-parse", "HEAD")
	if err != nil {
		return result, fmt.Errorf("Evidence実行後のrepository revisionを取得できません: %w", err)
	}
	result.RepositoryRevision = strings.TrimSpace(revisionBefore)
	if result.RepositoryRevision == "" {
		return result, fmt.Errorf("Evidence実行後のrepository revisionが空です")
	}
	result.RevisionStable = result.RepositoryRevision == initialRevision

	status, err := output("git", "status", "--porcelain", "--untracked-files=all")
	if err != nil {
		return result, fmt.Errorf("Evidence実行後のworking tree状態を取得できません: %w", err)
	}
	result.WorkingTreeClean = strings.TrimSpace(status) == ""

	revisionAfter, err := output("git", "rev-parse", "HEAD")
	if err != nil {
		return result, fmt.Errorf("Evidence実行後のworking tree観測後にrepository revisionを取得できません: %w", err)
	}
	revisionAfter = strings.TrimSpace(revisionAfter)
	if revisionAfter == "" {
		return result, fmt.Errorf("Evidence実行後のworking tree観測後のrepository revisionが空です")
	}
	if result.RepositoryRevision != revisionAfter {
		before := result.RepositoryRevision
		result.RepositoryRevision = revisionAfter
		result.RevisionStable = false
		return result, fmt.Errorf("Evidence実行後のrepository source観測中にHEADが変化しました: before=%s after=%s", before, revisionAfter)
	}

	result.RepositoryRevision = revisionAfter
	result.RevisionStable = result.RepositoryRevision == initialRevision
	result.Checked = true
	return result, nil
}

func validateGitSourceEnvironment(env []string) error {
	for _, entry := range env {
		key, _, ok := strings.Cut(entry, "=")
		if !ok {
			continue
		}
		for _, forbidden := range gitSourceIdentityOverrideKeys {
			if strings.EqualFold(key, forbidden) {
				return fmt.Errorf("repository source観測ではGit identity override %sを使用できません。実行checkoutとは別のrepository / work tree / index等を観測する可能性があるため、環境変数を解除して再実行してください", forbidden)
			}
		}
		if isGitCommandScopeConfigOverrideKey(key) {
			return fmt.Errorf("repository source観測ではGit command-scope config override %sを使用できません。git status等の観測意味をambient environmentから変更できるため、環境変数を解除して再実行してください", key)
		}
	}
	return nil
}

func isGitCommandScopeConfigOverrideKey(key string) bool {
	upper := strings.ToUpper(key)
	return upper == "GIT_CONFIG_COUNT" ||
		upper == "GIT_CONFIG_PARAMETERS" ||
		strings.HasPrefix(upper, "GIT_CONFIG_KEY_") ||
		strings.HasPrefix(upper, "GIT_CONFIG_VALUE_")
}

// IsReproducibleは、実行開始前後が同一のclean revisionへ固定できる場合だけtrueを返す。
func IsReproducible(initial Source, post PostRun) bool {
	return initial.RepositoryRevision != "" &&
		initial.WorkingTreeClean &&
		post.Checked &&
		post.RepositoryRevision == initial.RepositoryRevision &&
		post.RevisionStable &&
		post.WorkingTreeClean
}

func commandOutputWithEnv(env []string) commandOutputFunc {
	environment := append([]string{}, env...)
	return func(name string, args ...string) (string, error) {
		cmd := exec.Command(name, args...)
		cmd.Env = append([]string{}, environment...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			return "", fmt.Errorf("%s: %w: %s", strings.Join(append([]string{name}, args...), " "), err, strings.TrimSpace(string(out)))
		}
		return string(out), nil
	}
}
