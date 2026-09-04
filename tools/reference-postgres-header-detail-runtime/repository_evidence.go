package main

import (
	"fmt"
	"os/exec"
	"strings"
)

type postRunRepositoryEvidence struct {
	Checked            bool   `json:"checked"`
	RepositoryRevision string `json:"repositoryRevision,omitempty"`
	RevisionStable     bool   `json:"revisionStable"`
	WorkingTreeClean   bool   `json:"workingTreeClean"`
}

type repositorySourceEvidence struct {
	RepositoryRevision   string
	WorkingTreeClean     bool
	ReproducibleRevision bool
	PostRunRepository    postRunRepositoryEvidence
	GoVersion            string
}

func inspectInitialRepository() (repositorySourceEvidence, error) {
	result := repositorySourceEvidence{}

	revision, err := repositoryCommandOutput("git", "rev-parse", "HEAD")
	if err != nil {
		return result, fmt.Errorf("PostgreSQL Evidenceのrepository revisionを取得できません: %w", err)
	}
	result.RepositoryRevision = strings.TrimSpace(revision)
	if result.RepositoryRevision == "" {
		return result, fmt.Errorf("PostgreSQL Evidenceのrepository revisionが空です")
	}

	status, err := repositoryCommandOutput("git", "status", "--porcelain")
	if err != nil {
		return result, fmt.Errorf("PostgreSQL Evidenceのworking tree状態を取得できません: %w", err)
	}
	result.WorkingTreeClean = strings.TrimSpace(status) == ""
	if !result.WorkingTreeClean {
		return result, fmt.Errorf("working treeに未commit差分があるためPostgreSQL Evidenceを開始しません。Gate Bへ採用するProvider Evidenceはclean revisionから実行してください")
	}

	goVersion, err := repositoryCommandOutput("go", "version")
	if err != nil {
		return result, fmt.Errorf("PostgreSQL EvidenceのGo toolchain identityを取得できません: %w", err)
	}
	result.GoVersion = strings.TrimSpace(goVersion)
	return result, nil
}

func finalizeRepositoryEvidence(result *repositorySourceEvidence) error {
	postRun, err := inspectPostRunRepository(result.RepositoryRevision)
	result.PostRunRepository = postRun
	result.ReproducibleRevision = isReproducibleRepositoryState(
		result.RepositoryRevision,
		result.WorkingTreeClean,
		postRun,
	)
	if err != nil {
		return err
	}
	if !postRun.RevisionStable {
		return fmt.Errorf("PostgreSQL Evidence実行中にrepository HEADが変化したため、単一revisionのEvidenceとして採用できません")
	}
	if !postRun.WorkingTreeClean {
		return fmt.Errorf("PostgreSQL Evidence実行後にworking tree差分が残ったため、記録revisionと実行sourceの同一性を保証できません")
	}
	if !result.ReproducibleRevision {
		return fmt.Errorf("PostgreSQL Evidenceをcleanかつ固定されたrepository revisionへ再現可能に紐付けられません")
	}
	return nil
}

func inspectPostRunRepository(initialRevision string) (postRunRepositoryEvidence, error) {
	result := postRunRepositoryEvidence{Checked: true}

	revision, err := repositoryCommandOutput("git", "rev-parse", "HEAD")
	if err != nil {
		return result, fmt.Errorf("PostgreSQL Evidence実行後のrepository revisionを取得できません: %w", err)
	}
	result.RepositoryRevision = strings.TrimSpace(revision)
	result.RevisionStable = result.RepositoryRevision == initialRevision

	status, err := repositoryCommandOutput("git", "status", "--porcelain")
	if err != nil {
		return result, fmt.Errorf("PostgreSQL Evidence実行後のworking tree状態を取得できません: %w", err)
	}
	result.WorkingTreeClean = strings.TrimSpace(status) == ""
	return result, nil
}

func isReproducibleRepositoryState(
	initialRevision string,
	initialClean bool,
	postRun postRunRepositoryEvidence,
) bool {
	return initialRevision != "" &&
		initialClean &&
		postRun.Checked &&
		postRun.RepositoryRevision == initialRevision &&
		postRun.RevisionStable &&
		postRun.WorkingTreeClean
}

func repositoryCommandOutput(name string, args ...string) (string, error) {
	out, err := exec.Command(name, args...).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%s: %w: %s", strings.Join(append([]string{name}, args...), " "), err, strings.TrimSpace(string(out)))
	}
	return string(out), nil
}
