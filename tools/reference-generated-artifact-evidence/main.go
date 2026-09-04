package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

const evidenceNature = "generated-artifact-reproducibility"

type commandSpec struct {
	Name string
	Args []string
}

type checkEvidence struct {
	Name    string   `json:"name"`
	Command []string `json:"command"`
	Passed  bool     `json:"passed"`
}

type postRunRepositoryEvidence struct {
	Checked            bool   `json:"checked"`
	RepositoryRevision string `json:"repositoryRevision,omitempty"`
	RevisionStable     bool   `json:"revisionStable"`
	WorkingTreeClean   bool   `json:"workingTreeClean"`
}

type evidence struct {
	EvidenceNature       string                    `json:"evidenceNature"`
	RepositoryRevision   string                    `json:"repositoryRevision"`
	WorkingTreeClean     bool                      `json:"workingTreeClean"`
	ReproducibleRevision bool                      `json:"reproducibleRevision"`
	PostRunRepository    postRunRepositoryEvidence `json:"postRunRepository"`
	GoVersion            string                    `json:"goVersion"`
	Checks               []checkEvidence           `json:"checks"`
}

func main() {
	result, err := run()
	encoded, encodeErr := json.MarshalIndent(result, "", "  ")
	if encodeErr != nil {
		fmt.Fprintln(os.Stderr, encodeErr)
		os.Exit(1)
	}
	fmt.Println(string(encoded))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() (evidence, error) {
	result := evidence{
		EvidenceNature: evidenceNature,
		Checks:         []checkEvidence{},
	}

	revision, err := commandOutput("git", "rev-parse", "HEAD")
	if err != nil {
		return result, fmt.Errorf("repository revisionを取得できません: %w", err)
	}
	result.RepositoryRevision = strings.TrimSpace(revision)

	status, err := commandOutput("git", "status", "--porcelain")
	if err != nil {
		return result, fmt.Errorf("working tree状態を取得できません: %w", err)
	}
	result.WorkingTreeClean = strings.TrimSpace(status) == ""
	if !result.WorkingTreeClean {
		return result, fmt.Errorf("generated artifact完了Evidenceはclean revisionに固定するため、working treeに未commit差分がある状態では実行しません。差分を確認・commitした後に再実行してください")
	}

	goVersion, err := commandOutput("go", "version")
	if err != nil {
		return result, fmt.Errorf("Go toolchainを確認できません: %w", err)
	}
	result.GoVersion = strings.TrimSpace(goVersion)

	failed := false
	for _, spec := range checkOnlyCommands() {
		passed := runStreamingCommand(spec)
		result.Checks = append(result.Checks, checkEvidence{
			Name:    spec.Name,
			Command: commandDisplay(spec),
			Passed:  passed,
		})
		if !passed {
			failed = true
		}
	}

	postRun, postRunErr := inspectPostRunRepository(result.RepositoryRevision)
	result.PostRunRepository = postRun
	result.ReproducibleRevision = isReproducibleRepositoryState(
		result.RepositoryRevision,
		result.WorkingTreeClean,
		postRun,
	)
	if postRunErr != nil {
		return result, postRunErr
	}
	if !postRun.RevisionStable {
		return result, fmt.Errorf("generated artifact Evidence実行中にrepository HEADが変化しました。単一revisionの完了Evidenceとして採用せず、固定されたclean revisionで再実行してください")
	}
	if !postRun.WorkingTreeClean {
		return result, fmt.Errorf("check-only Evidence実行後にworking tree差分が残りました。generated artifactやsourceの差分を確認し、clean revisionで再実行してください")
	}
	if failed {
		return result, fmt.Errorf("generated artifact checkに不一致があります。個別toolの診断を確認し、必要な場合だけ各toolの-writeで修復して差分をcommitした後、新しいclean revisionで本runnerを再実行してください")
	}
	if !result.ReproducibleRevision {
		return result, fmt.Errorf("generated artifact Evidenceを単一のclean revisionへ固定できません")
	}
	return result, nil
}

func checkOnlyCommands() []commandSpec {
	return []commandSpec{
		{
			Name: "world-plugin-lock-golden",
			Args: []string{"run", "./tools/reference-golden"},
		},
		{
			Name: "durable-run-golden",
			Args: []string{"run", "./tools/reference-durable-run-golden"},
		},
	}
}

func inspectPostRunRepository(initialRevision string) (postRunRepositoryEvidence, error) {
	result := postRunRepositoryEvidence{Checked: true}

	revision, err := commandOutput("git", "rev-parse", "HEAD")
	if err != nil {
		return result, fmt.Errorf("generated artifact Evidence実行後のrepository revisionを取得できません: %w", err)
	}
	result.RepositoryRevision = strings.TrimSpace(revision)
	result.RevisionStable = result.RepositoryRevision == initialRevision

	status, err := commandOutput("git", "status", "--porcelain")
	if err != nil {
		return result, fmt.Errorf("generated artifact Evidence実行後のworking tree状態を取得できません: %w", err)
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

func commandOutput(name string, args ...string) (string, error) {
	out, err := exec.Command(name, args...).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%s: %w: %s", strings.Join(append([]string{name}, args...), " "), err, strings.TrimSpace(string(out)))
	}
	return string(out), nil
}

func runStreamingCommand(spec commandSpec) bool {
	fmt.Fprintf(os.Stderr, "[Generated Artifact Evidence] go %s\n", strings.Join(spec.Args, " "))
	cmd := exec.Command("go", spec.Args...)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	return cmd.Run() == nil
}

func commandDisplay(spec commandSpec) []string {
	return append([]string{"go"}, spec.Args...)
}
