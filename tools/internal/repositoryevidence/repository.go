package repositoryevidence

import (
	"fmt"
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
// Checkedは、実行後のHEADとworking treeの両方を取得できた場合だけtrueにする。
// 途中までしか観測できなかった状態を「確認済み」として扱わない。
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

// InspectStartはEvidence実行開始時のHEADとworking tree状態を取得する。
func InspectStart() (Source, error) {
	var result Source

	revision, err := commandOutput("git", "rev-parse", "HEAD")
	if err != nil {
		return result, fmt.Errorf("repository revisionを取得できません: %w", err)
	}
	result.RepositoryRevision = strings.TrimSpace(revision)
	if result.RepositoryRevision == "" {
		return result, fmt.Errorf("repository revisionが空です")
	}

	status, err := commandOutput("git", "status", "--porcelain")
	if err != nil {
		return result, fmt.Errorf("working tree状態を取得できません: %w", err)
	}
	result.WorkingTreeClean = strings.TrimSpace(status) == ""
	return result, nil
}

// InspectPostはEvidence実行後のHEADとworking tree状態を取得し、開始revisionとの同一性を記録する。
func InspectPost(initialRevision string) (PostRun, error) {
	return inspectPost(initialRevision, commandOutput)
}

func inspectPost(initialRevision string, output commandOutputFunc) (PostRun, error) {
	var result PostRun
	if strings.TrimSpace(initialRevision) == "" {
		return result, fmt.Errorf("比較対象のrepository revisionが空です")
	}

	revision, err := output("git", "rev-parse", "HEAD")
	if err != nil {
		return result, fmt.Errorf("Evidence実行後のrepository revisionを取得できません: %w", err)
	}
	result.RepositoryRevision = strings.TrimSpace(revision)
	result.RevisionStable = result.RepositoryRevision == initialRevision

	status, err := output("git", "status", "--porcelain")
	if err != nil {
		return result, fmt.Errorf("Evidence実行後のworking tree状態を取得できません: %w", err)
	}
	result.WorkingTreeClean = strings.TrimSpace(status) == ""
	result.Checked = true
	return result, nil
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

func commandOutput(name string, args ...string) (string, error) {
	out, err := exec.Command(name, args...).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%s: %w: %s", strings.Join(append([]string{name}, args...), " "), err, strings.TrimSpace(string(out)))
	}
	return string(out), nil
}
