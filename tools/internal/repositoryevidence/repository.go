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

// InspectStartはEvidence実行開始時のHEADとworking tree状態を取得する。
// HEADをworking tree観測の前後で確認し、観測中に切り替わった場合はfail closedにする。
func InspectStart() (Source, error) {
	return inspectStart(commandOutput)
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

	status, err := output("git", "status", "--porcelain")
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
	return inspectPost(initialRevision, commandOutput)
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

	status, err := output("git", "status", "--porcelain")
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
