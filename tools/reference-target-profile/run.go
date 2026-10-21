package targetprofile

import (
	"context"
	"os"
	"strconv"

	generatorselection "tei/tools/reference-generator-selection"
)

type RunOptions struct {
	WorkspaceDir  string
	ContractPath  string
	FixturePath   string
	CandidatesDir string
	ProfilePath   string
}

// ObservabilityEvidence records which declared observation channels the
// harness actually observed from the selected candidate's case executions.
type ObservabilityEvidence struct {
	Declared          []string `json:"declared"`
	Observed          []string `json:"observed"`
	Unsupported       []string `json:"unsupported,omitempty"`
	Cases             int      `json:"cases"`
	CasesWithStdout   int      `json:"casesWithStdout"`
	CasesWithStderr   int      `json:"casesWithStderr"`
	CasesWithExitCode int      `json:"casesWithExitCode"`
}

type Receipt struct {
	Schema       string                     `json:"schema"`
	Profile      Profile                    `json:"profile"`
	Selection    generatorselection.Receipt `json:"selection"`
	Observation  *ObservabilityEvidence     `json:"observation,omitempty"`
	ProcessModel *ProcessModelEvidence      `json:"processModel,omitempty"`
	Enforcement  map[string]string          `json:"enforcement"`
	Unsupported  []string                   `json:"unsupported,omitempty"`
	Passed       bool                       `json:"passed"`
	Notes        []string                   `json:"notes"`
}

// Run applies the profile's memory limit to the selection subprocesses, runs
// the same fixed fixtures, and records which constraints were actually
// enforced. Storage, network, and update remain declarations;
// observability channels stdout, stderr, and exit-code are observed from the
// selected candidate's case executions.
func Run(ctx context.Context, opts RunOptions) (Receipt, error) {
	profile, err := Load(opts.ProfilePath)
	if err != nil {
		return Receipt{}, err
	}
	previous, hadPrevious := os.LookupEnv("GOMEMLIMIT")
	if err := os.Setenv("GOMEMLIMIT", strconv.FormatInt(profile.MemoryLimitBytes, 10)); err != nil {
		return Receipt{}, err
	}
	defer func() {
		if hadPrevious {
			_ = os.Setenv("GOMEMLIMIT", previous)
		} else {
			_ = os.Unsetenv("GOMEMLIMIT")
		}
	}()
	var allowed []string
	if profile.GeneratorID != "" {
		allowed = []string{profile.GeneratorID}
	}
	selection, err := generatorselection.SelectForTarget(ctx, opts.WorkspaceDir, opts.ContractPath, opts.FixturePath, opts.CandidatesDir, "", allowed)
	if err != nil {
		return Receipt{}, err
	}
	observation := observe(profile, selection)
	processModel := enforceProcessModel(ctx, profile, opts, selection)
	unsupported := append([]string(nil), profile.Unsupported...)
	for _, value := range observation.Unsupported {
		unsupported = appendUnique(unsupported, value)
	}
	if !processModel.Passed && processModel.Mode == ProcessBatch {
		unsupported = appendUnique(unsupported, "batch-process-model")
	}
	observabilityEnforcement := "observed"
	if len(observation.Unsupported) > 0 {
		observabilityEnforcement = "partial"
		if len(observation.Observed) == 0 {
			observabilityEnforcement = "unsupported"
		}
	}
	processEnforcement := "unsupported"
	switch processModel.Mode {
	case ProcessOneShot:
		processEnforcement = "oneshot:failed"
		if processModel.Passed {
			processEnforcement = "oneshot:process-per-case"
		}
	case ProcessLongRunning:
		processEnforcement = "long-running:failed"
		if processModel.Passed {
			processEnforcement = "long-running:single-process"
		}
	}
	memoryEnforcement := "env:GOMEMLIMIT (soft)"
	if candidate := selectedCandidate(Receipt{Profile: profile, Selection: selection}); candidate != nil && candidate.Language != generatorselection.LanguageGo {
		memoryEnforcement = "not-enforced: GOMEMLIMIT does not limit non-Go runtimes"
	}
	return Receipt{
		Schema:       Schema,
		Profile:      profile,
		Selection:    selection,
		Observation:  observation,
		ProcessModel: processModel,
		Enforcement: map[string]string{
			"memory":        memoryEnforcement,
			"processModel":  processEnforcement,
			"storage":       "declared-only",
			"network":       "declared-only",
			"update":        "declared-only",
			"observability": observabilityEnforcement,
		},
		Unsupported: unsupported,
		Passed:      selection.Passed && len(observation.Unsupported) == 0 && processModel.Passed,
		Notes: []string{
			"同じ受理済み契約・fixture・候補を、Target Profileの宣言とともに実行した。memoryの強制範囲はenforcementへ記録する。",
			"observabilityのstdout・stderr・exit-codeはcase実行から観測した。未対応channelはUnsupportedとして返す。",
			"processModelはoneshotのprocess-per-case、long-runningのsingle-processを実行で確認した。",
			"storage・network・updateは宣言であり、このReferenceでは強制していない。",
			"profileの合格は、実機・実OS・実Targetでの動作を証明しない。",
		},
	}, nil
}

func observe(profile Profile, selection generatorselection.Receipt) *ObservabilityEvidence {
	observation := &ObservabilityEvidence{
		Declared: append([]string(nil), profile.Observability...),
		Observed: []string{}, Unsupported: []string{},
	}
	candidate := selectedCandidate(Receipt{Profile: profile, Selection: selection})
	if candidate == nil || len(candidate.Results) == 0 {
		observation.Unsupported = append(observation.Unsupported, "no-execution-evidence")
		return observation
	}
	observation.Cases = len(candidate.Results)
	for _, result := range candidate.Results {
		if result.Execution == nil {
			continue
		}
		observation.CasesWithStdout++
		observation.CasesWithStderr++
		observation.CasesWithExitCode++
	}
	for _, channel := range profile.Observability {
		switch channel {
		case "stdout":
			if observation.CasesWithStdout == observation.Cases {
				observation.Observed = appendUnique(observation.Observed, channel)
			} else {
				observation.Unsupported = appendUnique(observation.Unsupported, channel)
			}
		case "stderr":
			if observation.CasesWithStderr == observation.Cases {
				observation.Observed = appendUnique(observation.Observed, channel)
			} else {
				observation.Unsupported = appendUnique(observation.Unsupported, channel)
			}
		case "exit-code":
			if observation.CasesWithExitCode == observation.Cases {
				observation.Observed = appendUnique(observation.Observed, channel)
			} else {
				observation.Unsupported = appendUnique(observation.Unsupported, channel)
			}
		default:
			observation.Unsupported = appendUnique(observation.Unsupported, channel)
		}
	}
	return observation
}

func appendUnique(values []string, value string) []string {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}
