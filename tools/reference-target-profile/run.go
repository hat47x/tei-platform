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

type Receipt struct {
	Schema      string                     `json:"schema"`
	Profile     Profile                    `json:"profile"`
	Selection   generatorselection.Receipt `json:"selection"`
	Enforcement map[string]string          `json:"enforcement"`
	Unsupported []string                   `json:"unsupported,omitempty"`
	Passed      bool                       `json:"passed"`
	Notes       []string                   `json:"notes"`
}

// Run applies the profile's memory limit to the selection subprocesses, runs
// the same fixed fixtures, and records which constraints were actually
// enforced. Process model, storage, network, update, and observability are
// declarations in this Reference; they are not enforced by the harness.
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
	selection, err := generatorselection.Select(ctx, opts.WorkspaceDir, opts.ContractPath, opts.FixturePath, opts.CandidatesDir)
	if err != nil {
		return Receipt{}, err
	}
	return Receipt{
		Schema:    Schema,
		Profile:   profile,
		Selection: selection,
		Enforcement: map[string]string{
			"memory":        "env:GOMEMLIMIT (soft)",
			"processModel":  "declared-only",
			"storage":       "declared-only",
			"network":       "declared-only",
			"update":        "declared-only",
			"observability": "declared-only",
		},
		Unsupported: append([]string(nil), profile.Unsupported...),
		Passed:      selection.Passed,
		Notes: []string{
			"同じ受理済み契約・fixture・候補を、Target Profileの宣言とmemory limit付きで実行した。",
			"processModel・storage・network・update・observabilityは宣言であり、このReferenceでは強制していない。",
			"profileの合格は、実機・実OS・実Targetでの動作を証明しない。",
		},
	}, nil
}
