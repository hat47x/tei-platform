package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"tei/tools/internal/repositoryevidence"
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

type evidence struct {
	EvidenceNature       string                     `json:"evidenceNature"`
	RepositoryRevision   string                     `json:"repositoryRevision"`
	WorkingTreeClean     bool                       `json:"workingTreeClean"`
	ReproducibleRevision bool                       `json:"reproducibleRevision"`
	PostRunRepository    repositoryevidence.PostRun `json:"postRunRepository"`
	GoVersion            string                     `json:"goVersion"`
	Checks               []checkEvidence            `json:"checks"`
}

type inspectStartFunc func() (repositoryevidence.Source, error)
type inspectPostFunc func(string) (repositoryevidence.PostRun, error)
type checkRunnerFunc func(commandSpec) bool
type commandOutputFunc func(string, ...string) (string, error)

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
	return runWith(
		repositoryevidence.InspectStart,
		repositoryevidence.InspectPost,
		runStreamingCommand,
		commandOutput,
	)
}

func runWith(
	inspectStart inspectStartFunc,
	inspectPost inspectPostFunc,
	checkRunner checkRunnerFunc,
	output commandOutputFunc,
) (evidence, error) {
	result := evidence{
		EvidenceNature: evidenceNature,
		Checks:         []checkEvidence{},
	}

	source, err := inspectStart()
	if err != nil {
		return result, err
	}
	result.RepositoryRevision = source.RepositoryRevision
	result.WorkingTreeClean = source.WorkingTreeClean
	if !result.WorkingTreeClean {
		return result, fmt.Errorf("generated artifact完了Evidenceはclean revisionに固定するため、working treeに未commit差分がある状態では実行しません。差分を確認・commitした後に再実行してください")
	}

	goVersion, err := output("go", "version")
	if err != nil {
		return result, fmt.Errorf("Go toolchainを確認できません: %w", err)
	}
	result.GoVersion = strings.TrimSpace(goVersion)

	failed := false
	for _, spec := range checkOnlyCommands() {
		passed := checkRunner(spec)
		result.Checks = append(result.Checks, checkEvidence{
			Name:    spec.Name,
			Command: commandDisplay(spec),
			Passed:  passed,
		})
		if !passed {
			failed = true
		}
	}

	postRun, postRunErr := inspectPost(source.RepositoryRevision)
	result.PostRunRepository = postRun
	if postRunErr != nil {
		return result, postRunErr
	}
	if !postRun.Checked {
		return result, fmt.Errorf("generated artifact Evidence実行後のrepository source確認が完了していません。単一revisionの完了Evidenceとして採用しません")
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

	result.ReproducibleRevision = repositoryevidence.IsReproducible(source, postRun)
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
