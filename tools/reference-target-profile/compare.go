package targetprofile

import (
	"context"
	"fmt"
	"sort"
	"strings"

	generatorselection "tei/tools/reference-generator-selection"
)

const ComparisonSchema = "tei.reference.target-profile-comparison/v0"

type CompareOptions struct {
	WorkspaceDir   string
	ContractPath   string
	FixturePath    string
	CandidatesDir  string
	ProfilePaths   []string
	EnforceNetwork bool
}

type ProfileSummary struct {
	ProcessModelEvidence *ProcessModelEvidence `json:"processModelEvidence,omitempty"`
	ID                   string                `json:"id"`
	Version              string                `json:"version"`
	ProcessModel         string                `json:"processModel"`
	Storage              string                `json:"storage"`
	Network              string                `json:"network"`
	Update               string                `json:"update"`
	MemoryLimitBytes     int64                 `json:"memoryLimitBytes"`
	GeneratorID          string                `json:"generatorId,omitempty"`
	Passed               bool                  `json:"passed"`
	Qualified            []string              `json:"qualified"`
	Rejected             []string              `json:"rejected"`
	Enforcement          map[string]string     `json:"enforcement"`
	Unsupported          []string              `json:"unsupported,omitempty"`
}

type ProfileCase struct {
	ProfileID  string `json:"profileId"`
	Generator  string `json:"generator,omitempty"`
	Present    bool   `json:"present"`
	State      string `json:"state,omitempty"`
	Amount     string `json:"amount,omitempty"`
	Currency   string `json:"currency,omitempty"`
	Code       string `json:"code,omitempty"`
	CasePassed bool   `json:"casePassed"`
}

type CaseComparison struct {
	CaseID  string        `json:"caseId"`
	Results []ProfileCase `json:"results"`
	Equal   bool          `json:"equal"`
}

type ComparisonReceipt struct {
	Schema                 string           `json:"schema"`
	ContractID             string           `json:"contractId"`
	ContractVersion        string           `json:"contractVersion"`
	FixtureDigest          string           `json:"fixtureDigest"`
	SuiteVersion           string           `json:"suiteVersion"`
	Profiles               []ProfileSummary `json:"profiles"`
	Cases                  []CaseComparison `json:"cases"`
	SemanticsEqual         bool             `json:"semanticsEqual"`
	ProfileDifferences     []string         `json:"profileDifferences,omitempty"`
	UnsupportedDifferences []string         `json:"unsupportedDifferences,omitempty"`
	EnforcementDifferences []string         `json:"enforcementDifferences,omitempty"`
	Passed                 bool             `json:"passed"`
	GrantsRuntimeAuthority bool             `json:"grantsRuntimeAuthority"`
	ProvesRealTarget       bool             `json:"provesRealTarget"`
	Notes                  []string         `json:"notes"`
}

// Compare runs two or more Target Profiles against the same contract, fixture,
// and candidate set, then compares the selected candidate's per-case output.
// It does not claim that the profiles enforce their declared constraints or
// that the targets are equivalent in production.
func Compare(ctx context.Context, opts CompareOptions) (ComparisonReceipt, error) {
	if len(opts.ProfilePaths) < 2 {
		return ComparisonReceipt{}, fmt.Errorf("compare requires at least two profiles")
	}
	var runs []Receipt
	seen := map[string]bool{}
	for _, profilePath := range opts.ProfilePaths {
		receipt, err := Run(ctx, RunOptions{
			WorkspaceDir: opts.WorkspaceDir, ContractPath: opts.ContractPath,
			FixturePath: opts.FixturePath, CandidatesDir: opts.CandidatesDir,
			ProfilePath: profilePath, EnforceNetwork: opts.EnforceNetwork,
		})
		if err != nil {
			return ComparisonReceipt{}, fmt.Errorf("%s: %w", profilePath, err)
		}
		if seen[receipt.Profile.ID] {
			return ComparisonReceipt{}, fmt.Errorf("duplicate profile id %q", receipt.Profile.ID)
		}
		seen[receipt.Profile.ID] = true
		runs = append(runs, receipt)
	}
	receipt := ComparisonReceipt{
		Schema:                 ComparisonSchema,
		Profiles:               []ProfileSummary{},
		Cases:                  []CaseComparison{},
		GrantsRuntimeAuthority: false,
		ProvesRealTarget:       false,
		Notes: []string{
			"同じ契約・fixture・候補集合を複数Profileで実行し、選択候補のcase別出力を比較した。",
			"Profileの宣言・unsupported・enforcementの差は、意味の一致と別に記録する。",
			"この比較は実機・実OS・実Targetでの動作や、宣言した制約の強制を証明しない。",
		},
	}
	first := runs[0].Selection
	receipt.ContractID = first.ContractID
	receipt.ContractVersion = first.ContractVersion
	receipt.FixtureDigest = first.FixtureDigest
	receipt.SuiteVersion = first.SuiteVersion
	for _, run := range runs {
		receipt.Profiles = append(receipt.Profiles, summarizeProfile(run))
	}
	receipt.Cases, receipt.SemanticsEqual = compareCases(runs)
	receipt.ProfileDifferences = profileDifferences(runs)
	receipt.UnsupportedDifferences = unsupportedDifferences(runs)
	receipt.EnforcementDifferences = enforcementDifferences(runs)
	allPassed := true
	for _, run := range runs {
		if !run.Passed {
			allPassed = false
		}
	}
	receipt.Passed = allPassed && receipt.SemanticsEqual
	return receipt, nil
}

func summarizeProfile(run Receipt) ProfileSummary {
	return ProfileSummary{
		ProcessModelEvidence: run.ProcessModel,
		ID:                   run.Profile.ID, Version: run.Profile.Version,
		ProcessModel: run.Profile.ProcessModel, Storage: run.Profile.Storage,
		Network: run.Profile.Network, Update: run.Profile.Update,
		MemoryLimitBytes: run.Profile.MemoryLimitBytes, GeneratorID: run.Profile.GeneratorID,
		Passed: run.Passed, Qualified: append([]string(nil), run.Selection.Qualified...),
		Rejected:    append([]string(nil), run.Selection.Rejected...),
		Enforcement: copyMap(run.Enforcement), Unsupported: append([]string(nil), run.Unsupported...),
	}
}

func copyMap(values map[string]string) map[string]string {
	out := map[string]string{}
	for key, value := range values {
		out[key] = value
	}
	return out
}

func selectedCandidate(run Receipt) *generatorselection.CandidateReceipt {
	if run.Profile.GeneratorID != "" {
		for i := range run.Selection.Candidates {
			if run.Selection.Candidates[i].ID == run.Profile.GeneratorID {
				return &run.Selection.Candidates[i]
			}
		}
		return nil
	}
	if len(run.Selection.Qualified) == 1 {
		for i := range run.Selection.Candidates {
			if run.Selection.Candidates[i].ID == run.Selection.Qualified[0] {
				return &run.Selection.Candidates[i]
			}
		}
	}
	return nil
}

func compareCases(runs []Receipt) ([]CaseComparison, bool) {
	order := []string{}
	byCase := map[string]map[string]ProfileCase{}
	selected := map[string]*generatorselection.CandidateReceipt{}
	for _, run := range runs {
		candidate := selectedCandidate(run)
		selected[run.Profile.ID] = candidate
		if candidate == nil {
			continue
		}
		for _, result := range candidate.Results {
			if _, ok := byCase[result.CaseID]; !ok {
				order = append(order, result.CaseID)
				byCase[result.CaseID] = map[string]ProfileCase{}
			}
			byCase[result.CaseID][run.Profile.ID] = ProfileCase{
				ProfileID: run.Profile.ID, Generator: candidate.ID, Present: true,
				State: result.Got.State, Amount: result.Got.Amount,
				Currency: result.Got.Currency, Code: result.Got.Code, CasePassed: result.Passed,
			}
		}
	}
	if len(order) == 0 {
		return []CaseComparison{}, false
	}
	cases := make([]CaseComparison, 0, len(order))
	semanticsEqual := true
	for _, caseID := range order {
		results := make([]ProfileCase, 0, len(runs))
		var reference *ProfileCase
		equal := true
		for _, run := range runs {
			result, ok := byCase[caseID][run.Profile.ID]
			if !ok {
				generator := ""
				if candidate := selected[run.Profile.ID]; candidate != nil {
					generator = candidate.ID
				}
				result = ProfileCase{ProfileID: run.Profile.ID, Generator: generator, Present: false, CasePassed: false}
				equal = false
			}
			if reference == nil {
				copy := result
				reference = &copy
			} else if !sameCase(*reference, result) {
				equal = false
			}
			results = append(results, result)
		}
		if !equal {
			semanticsEqual = false
		}
		cases = append(cases, CaseComparison{CaseID: caseID, Results: results, Equal: equal})
	}
	return cases, semanticsEqual
}

func sameCase(left, right ProfileCase) bool {
	return left.Present == right.Present && left.State == right.State &&
		left.Amount == right.Amount && left.Currency == right.Currency &&
		left.Code == right.Code && left.CasePassed == right.CasePassed
}

func profileDifferences(runs []Receipt) []string {
	fields := []struct {
		name  string
		value func(Receipt) string
	}{
		{"processModel", func(run Receipt) string { return run.Profile.ProcessModel }},
		{"storage", func(run Receipt) string { return run.Profile.Storage }},
		{"network", func(run Receipt) string { return run.Profile.Network }},
		{"update", func(run Receipt) string { return run.Profile.Update }},
		{"memoryLimitBytes", func(run Receipt) string { return fmt.Sprintf("%d", run.Profile.MemoryLimitBytes) }},
	}
	var out []string
	for _, field := range fields {
		values := map[string][]string{}
		for _, run := range runs {
			value := field.value(run)
			values[value] = append(values[value], run.Profile.ID)
		}
		if len(values) > 1 {
			out = append(out, field.name+": "+formatValues(values))
		}
	}
	sort.Strings(out)
	return out
}

func unsupportedDifferences(runs []Receipt) []string {
	values := map[string][]string{}
	for _, run := range runs {
		for _, value := range run.Unsupported {
			values[value] = append(values[value], run.Profile.ID)
		}
	}
	var out []string
	for value, profiles := range values {
		if len(profiles) != len(runs) {
			sort.Strings(profiles)
			out = append(out, value+": "+strings.Join(profiles, ","))
		}
	}
	sort.Strings(out)
	return out
}

func enforcementDifferences(runs []Receipt) []string {
	keys := map[string]bool{}
	for _, run := range runs {
		for key := range run.Enforcement {
			keys[key] = true
		}
	}
	var out []string
	for key := range keys {
		values := map[string][]string{}
		for _, run := range runs {
			value := run.Enforcement[key]
			values[value] = append(values[value], run.Profile.ID)
		}
		if len(values) > 1 {
			out = append(out, key+": "+formatValues(values))
		}
	}
	sort.Strings(out)
	return out
}

func formatValues(values map[string][]string) string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, key := range keys {
		profiles := values[key]
		sort.Strings(profiles)
		parts = append(parts, strings.Join(profiles, ",")+"="+key)
	}
	return strings.Join(parts, "; ")
}
