package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"tei/internal/validation"
	"tei/internal/workspace"
	"tei/tools/internal/repositoryevidence"
)

const evidenceNature = "deterministic-repository"

type commandSpec struct {
	Name string
	Args []string
}

type checkEvidence struct {
	Name    string   `json:"name"`
	Command []string `json:"command"`
	Passed  bool     `json:"passed"`
}

type validationEvidence struct {
	Passed bool     `json:"passed"`
	Errors []string `json:"errors,omitempty"`
}

type evidence struct {
	EvidenceNature       string                     `json:"evidenceNature"`
	RepositoryRevision   string                     `json:"repositoryRevision"`
	WorkingTreeClean     bool                       `json:"workingTreeClean"`
	ReproducibleRevision bool                       `json:"reproducibleRevision"`
	PostRunRepository    repositoryevidence.PostRun `json:"postRunRepository"`
	GoVersion            string                     `json:"goVersion"`
	Workspace            string                     `json:"workspace"`
	WorkspaceValidation  validationEvidence         `json:"workspaceValidation"`
	Checks               []checkEvidence            `json:"checks"`
	BroadRequested       bool                       `json:"broadRequested"`
}

func main() {
	workspacePath := flag.String("workspace", "reference/workspace", "検証対象Reference Workspace directory")
	broad := flag.Bool("broad", false, "targeted Evidence成功後にgo test ./...も実行する")
	allowDirty := flag.Bool("allow-dirty", false, "dirty working treeでも診断目的で実行する。Gate B完了Evidenceには使用しない")
	flag.Parse()

	result, err := run(*workspacePath, *broad, *allowDirty)
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

func run(workspacePath string, broad, allowDirty bool) (evidence, error) {
	result := evidence{
		EvidenceNature: evidenceNature,
		Workspace:      workspacePath,
		BroadRequested: broad,
		Checks:         []checkEvidence{},
	}

	source, err := repositoryevidence.InspectStart()
	if err != nil {
		return result, err
	}
	result.RepositoryRevision = source.RepositoryRevision
	result.WorkingTreeClean = source.WorkingTreeClean
	result.ReproducibleRevision = source.WorkingTreeClean
	if !result.WorkingTreeClean && !allowDirty {
		return result, fmt.Errorf("working treeに未commit差分があるためGate B Repository Evidenceを停止します。差分を識別・整理してclean revisionで再実行してください。診断目的だけなら-allow-dirtyを使用できます")
	}

	goVersion, err := commandOutput("go", "version")
	if err != nil {
		return result, fmt.Errorf("Go toolchainを確認できません: %w", err)
	}
	result.GoVersion = strings.TrimSpace(goVersion)

	loaded, err := workspace.Load(workspacePath)
	if err != nil {
		return result, fmt.Errorf("Workspace %sを読み込めません: %w", workspacePath, err)
	}
	for _, diagnostic := range validation.Validate(loaded.World) {
		if diagnostic.Severity != validation.Error {
			continue
		}
		result.WorkspaceValidation.Errors = append(
			result.WorkspaceValidation.Errors,
			fmt.Sprintf("%s: %s", diagnostic.Code, diagnostic.Message),
		)
	}
	if len(result.WorkspaceValidation.Errors) > 0 {
		return result, fmt.Errorf("Workspace Validation ErrorがあるためRepository Evidence testを実行しません")
	}
	result.WorkspaceValidation.Passed = true

	targetedFailed := false
	for _, spec := range targetedChecks() {
		passed := runStreamingCommand(spec)
		result.Checks = append(result.Checks, checkEvidence{
			Name:    spec.Name,
			Command: commandDisplay(spec),
			Passed:  passed,
		})
		if !passed {
			targetedFailed = true
		}
	}

	broadFailed := false
	if !targetedFailed && broad {
		spec := broadCheck()
		passed := runStreamingCommand(spec)
		result.Checks = append(result.Checks, checkEvidence{
			Name:    spec.Name,
			Command: commandDisplay(spec),
			Passed:  passed,
		})
		if !passed {
			broadFailed = true
		}
	}

	postRun, postRunErr := repositoryevidence.InspectPost(source.RepositoryRevision)
	result.PostRunRepository = postRun
	result.ReproducibleRevision = repositoryevidence.IsReproducible(source, postRun)
	if postRunErr != nil {
		return result, postRunErr
	}
	if !postRun.RevisionStable {
		return result, fmt.Errorf("Evidence実行中にrepository HEADが変化したため、単一revisionのGate B Evidenceとして採用できません。cleanかつ固定されたrevisionで再実行してください")
	}
	if !postRun.WorkingTreeClean {
		return result, fmt.Errorf("Evidence実行後にworking tree差分が残ったため、記録revisionと実行sourceの同一性を保証できません。差分を確認してclean revisionで再実行してください")
	}
	if targetedFailed {
		return result, fmt.Errorf("targeted Gate B Repository Evidence testに失敗があります")
	}
	if broadFailed {
		return result, fmt.Errorf("broad Repository testに失敗があります")
	}
	if !result.ReproducibleRevision {
		return result, fmt.Errorf("-allow-dirtyで診断実行しました。testが成功してもclean revisionへ固定できないためGate B完了Evidenceには使用できません")
	}
	return result, nil
}

func targetedChecks() []commandSpec {
	return []commandSpec{
		{
			Name: "final-ui-less-scenario",
			Args: []string{
				"test", "./internal/driver",
				"-run", "^TestOrderEntryGateBScenarioConnectsMajorBoundariesWithoutUI$",
				"-count=1",
			},
		},
		{
			Name: "external-evidence-intake",
			Args: []string{
				"test",
				"./internal/database/externalqueryevidence",
				"./tools/reference-external-query-evidence",
				"-count=1",
			},
		},
		{
			Name: "repository-evidence-source-boundary",
			Args: []string{
				"test",
				"./tools/internal/repositoryevidence",
				"./tools/reference-gate-b-repository-evidence",
				"-count=1",
			},
		},
	}
}

func broadCheck() commandSpec {
	return commandSpec{
		Name: "repository-wide-go-test",
		Args: []string{"test", "./..."},
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
	fmt.Fprintf(os.Stderr, "[Gate B Evidence] go %s\n", strings.Join(spec.Args, " "))
	cmd := exec.Command("go", spec.Args...)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	return cmd.Run() == nil
}

func commandDisplay(spec commandSpec) []string {
	return append([]string{"go"}, spec.Args...)
}
