// Command tei-cognition-compare runs the declared case set through two providers
// and reports the four rates #5807 and #5809 require separately:
// accuracy, misclassification, abstention, and format violation.
//
// The rates are reported per provider and per case kind, because a provider that
// abstains on every ambiguous case and a provider that guesses are not
// distinguished by a single "accuracy" number. A format violation is counted
// apart from a wrong answer: a provider that returns an undeclared option or
// malformed JSON has not answered at all, and merging that into "incorrect"
// would hide a capability failure behind an accuracy figure.
//
// The command proposes only. It writes no receipt to a shared location, accepts
// nothing, and applies nothing.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"
	"time"

	choice "tei/plugins/reference-cognition-choice"
	lexical "tei/plugins/reference-cognition-lexical"
	port "tei/tools/reference-cognition-port"
)

// Case is one fixed input with an independently written expected answer.
type Case struct {
	ID        string `json:"id"`
	Kind      string `json:"kind"`
	Statement string `json:"statement"`
	Expected  string `json:"expected"`
	Note      string `json:"note,omitempty"`
}

// CaseSet is the fixture the two providers are compared on.
type CaseSet struct {
	Schema      string              `json:"schema"`
	Note        string              `json:"note"`
	Options     []port.ChoiceOption `json:"options"`
	Constraints []string            `json:"constraints"`
	Cases       []Case              `json:"cases"`
}

// Counts are one provider's outcome buckets for one group of cases.
type Counts struct {
	Cases              int `json:"cases"`
	Correct            int `json:"correct"`
	Wrong              int `json:"wrong"`
	AbstainedCorrectly int `json:"abstainedCorrectly"`
	AbstainedWrongly   int `json:"abstainedWrongly"`
	FormatViolation    int `json:"formatViolation"`
	Failed             int `json:"failed"`
}

func (c Counts) add(other Counts) Counts {
	return Counts{
		Cases:              c.Cases + other.Cases,
		Correct:            c.Correct + other.Correct,
		Wrong:              c.Wrong + other.Wrong,
		AbstainedCorrectly: c.AbstainedCorrectly + other.AbstainedCorrectly,
		AbstainedWrongly:   c.AbstainedWrongly + other.AbstainedWrongly,
		FormatViolation:    c.FormatViolation + other.FormatViolation,
		Failed:             c.Failed + other.Failed,
	}
}

// Rate is a percentage with the denominator it was computed over, so a zero
// denominator is visible rather than reported as 0%.
type Rate struct {
	Numerator   int     `json:"numerator"`
	Denominator int     `json:"denominator"`
	Percent     float64 `json:"percent"`
	Note        string  `json:"note,omitempty"`
}

func rate(numerator, denominator int) Rate {
	r := Rate{Numerator: numerator, Denominator: denominator}
	if denominator == 0 {
		r.Note = "no cases in this group"
		return r
	}
	r.Percent = float64(numerator) * 100 / float64(denominator)
	return r
}

// ProviderResult is one provider's full report.
type ProviderResult struct {
	Provider string            `json:"provider"`
	Model    string            `json:"model"`
	Version  string            `json:"version"`
	Overall  Counts            `json:"overall"`
	ByKind   map[string]Counts `json:"byKind"`
	Rates    map[string]Rate   `json:"rates"`
	Outcomes []CaseOutcome     `json:"outcomes"`
}

// CaseOutcome is one case's observed result, kept so a rate can be traced back
// to the individual answers that produced it.
type CaseOutcome struct {
	ID       string `json:"id"`
	Kind     string `json:"kind"`
	Expected string `json:"expected"`
	Observed string `json:"observed"`
	State    string `json:"state"`
	Verdict  string `json:"verdict"`
	Detail   string `json:"detail,omitempty"`
	Millis   int64  `json:"millis"`
}

func main() {
	casesPath := flag.String("cases", "tools/reference-cognition-port/testdata/cases/choice-baseline.json", "case set JSON")
	providerPath := flag.String("provider", "", "provider JSON (execution external, network none)")
	baseline := flag.Bool("baseline", true, "run the deterministic lexical baseline")
	model := flag.Bool("model", false, "also run the loopback model provider declared by -provider")
	flag.Parse()
	if !*baseline && !*model {
		fail(fmt.Errorf("select -baseline, -model, or both"))
	}
	var set CaseSet
	if err := readJSON(*casesPath, &set); err != nil {
		fail(err)
	}
	if len(set.Cases) == 0 {
		fail(fmt.Errorf("case set contains no cases"))
	}

	report := map[string]ProviderResult{}
	if *baseline {
		report["lexical"] = runSet(set, lexicalProvider())
	}
	if *model {
		if *providerPath == "" {
			fail(fmt.Errorf("-model requires -provider"))
		}
		provider, err := readProvider(*providerPath)
		if err != nil {
			fail(err)
		}
		report["model"] = runSet(set, provider)
	}

	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(report); err != nil {
		fail(err)
	}
}

// provider is the seam this command runs a case set through. The implementation
// is either the in-process baseline or the loopback model; the harness below
// does not know which, so both are measured by identical code.
type provider struct {
	name    string
	model   string
	version string
	invoke  func(context.Context, port.Provider, []byte) ([]byte, error)
}

func lexicalProvider() provider {
	return provider{
		name:    "reference-cognition-lexical",
		model:   lexical.Model,
		version: lexical.Version,
		invoke: func(ctx context.Context, p port.Provider, raw []byte) ([]byte, error) {
			if p.Network != port.NetworkNone {
				return nil, fmt.Errorf("lexical baseline requires network none")
			}
			return lexical.Invoke(ctx, raw)
		},
	}
}

func runSet(set CaseSet, p provider) ProviderResult {
	declared := port.Provider{
		Schema:       port.ProviderSchema,
		ID:           p.name,
		Model:        p.model,
		Version:      p.version,
		Execution:    port.ExecutionExternal,
		Network:      port.NetworkNone,
		Capabilities: []string{"choice"},
		Limits:       port.Limits{MaxContextBytes: 8192, MaxOutputBytes: 8192, TimeoutSeconds: 60},
	}
	result := ProviderResult{
		Provider: p.name,
		Model:    p.model,
		Version:  p.version,
		ByKind:   map[string]Counts{},
		Rates:    map[string]Rate{},
	}
	for _, c := range set.Cases {
		outcome, counts := measureCase(set, c, p, declared)
		result.Overall = result.Overall.add(counts)
		result.ByKind[c.Kind] = result.ByKind[c.Kind].add(counts)
		result.Outcomes = append(result.Outcomes, outcome)
	}

	// The four rates are reported apart, not combined: they answer different
	// questions and a provider can trade one for another.
	result.Rates = map[string]Rate{
		"accuracy": rate(result.Overall.Correct,
			result.Overall.Correct+result.Overall.Wrong+result.Overall.AbstainedCorrectly+result.Overall.AbstainedWrongly),
		"misclassification": rate(result.Overall.Wrong,
			result.Overall.Correct+result.Overall.Wrong+result.Overall.AbstainedCorrectly+result.Overall.AbstainedWrongly),
		"abstention": rate(result.Overall.AbstainedCorrectly+result.Overall.AbstainedWrongly,
			result.Overall.Correct+result.Overall.Wrong+result.Overall.AbstainedCorrectly+result.Overall.AbstainedWrongly),
		// A format violation is measured over every case, including the ones
		// where no answer was needed: returning undeclared output on any case
		// is the capability fact, whatever the case expected.
		"formatViolation": rate(result.Overall.FormatViolation, len(set.Cases)),
		"providerFailure": rate(result.Overall.Failed, len(set.Cases)),
	}
	return result
}

func measureCase(set CaseSet, c Case, p provider, declared port.Provider) (CaseOutcome, Counts) {
	request := port.Request{
		Schema:  port.RequestSchema,
		ID:      c.ID,
		Kind:    "choice",
		Profile: port.ProfileOffline,
		Context: port.Context{
			WorkspaceRevision: "case-set-v1",
			Observations:      []port.Observation{{ID: c.ID, Statement: c.Statement}},
			Constraints:       set.Constraints,
			Choice:            &port.ChoiceQuestion{ObservationID: c.ID, Options: set.Options},
		},
	}
	raw, err := json.Marshal(request.Context)
	if err != nil {
		return CaseOutcome{ID: c.ID, Verdict: "failed", Detail: err.Error()}, Counts{Cases: 1, Failed: 1}
	}

	started := time.Now()
	proposalRaw, err := p.invoke(context.Background(), declared, raw)
	elapsed := time.Since(started)

	outcome := CaseOutcome{ID: c.ID, Kind: c.Kind, Expected: c.Expected, Millis: elapsed.Milliseconds()}
	counts := Counts{Cases: 1}
	if err != nil {
		outcome.Verdict = "failed"
		outcome.Detail = err.Error()
		counts.Failed = 1
		return outcome, counts
	}

	var proposal port.Proposal
	if err := json.Unmarshal(proposalRaw, &proposal); err != nil {
		// Undecodable output is a format violation, not a wrong answer.
		outcome.Verdict = "format-violation"
		outcome.Detail = "proposal is not JSON: " + err.Error()
		// The port's own validator also refuses an undeclared option, so a
		// provider that returned well-formed but illegal output is caught there,
		// not here. Both are format violations for the purposes of this report.
		if validationErr := port.ValidateProposalForTest(proposal, request); validationErr != nil {
			outcome.Detail = "proposal is not valid: " + validationErr.Error()
		}
		counts.FormatViolation = 1
		return outcome, counts
	}
	if validationErr := port.ValidateProposalForTest(proposal, request); validationErr != nil {
		outcome.Verdict = "format-violation"
		outcome.Detail = "proposal is not valid: " + validationErr.Error()
		counts.FormatViolation = 1
		return outcome, counts
	}

	outcome.State = proposal.State
	switch proposal.State {
	case port.StateProposed:
		if len(proposal.Candidates) != 1 {
			outcome.Verdict = "format-violation"
			outcome.Detail = fmt.Sprintf("proposed with %d candidates", len(proposal.Candidates))
			counts.FormatViolation = 1
			return outcome, counts
		}
		observed, _ := proposal.Candidates[0].Payload["optionId"].(string)
		outcome.Observed = observed
		if observed == c.Expected {
			outcome.Verdict = "correct"
			counts.Correct = 1
		} else {
			outcome.Verdict = "wrong"
			counts.Wrong = 1
		}
	case port.StateUnknown:
		outcome.Observed = "unknown"
		if c.Expected == "unknown" {
			outcome.Verdict = "abstained-correctly"
			counts.AbstainedCorrectly = 1
		} else {
			outcome.Verdict = "abstained-wrongly"
			counts.AbstainedWrongly = 1
		}
	default:
		outcome.Verdict = "failed"
		outcome.Detail = "state " + proposal.State
		counts.Failed = 1
	}
	return outcome, counts
}

func readProvider(path string) (provider, error) {
	var declared port.Provider
	raw, err := os.ReadFile(path)
	if err != nil {
		return provider{}, err
	}
	if err := json.Unmarshal(raw, &declared); err != nil {
		return provider{}, fmt.Errorf("%s: %w", path, err)
	}
	if declared.Network != port.NetworkNone || declared.Execution != port.ExecutionExternal {
		return provider{}, fmt.Errorf("the model provider must declare execution external and network none")
	}
	endpoint, _ := declared.Settings["endpoint"].(string)
	if endpoint == "" {
		return provider{}, fmt.Errorf("the model provider must declare settings.endpoint")
	}
	settings := declared.Settings
	return provider{
		name:    declared.ID,
		model:   declared.Model,
		version: declared.Version,
		invoke: func(ctx context.Context, p port.Provider, raw []byte) ([]byte, error) {
			return choice.Invoke(ctx, endpoint, declared.Model, settings, raw)
		},
	}, nil
}

func readJSON(path string, into any) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(stripBOM(raw)))
	decoder.DisallowUnknownFields()
	return decoder.Decode(into)
}

// stripBOM removes a byte-order mark so a fixture saved by a Windows editor
// decodes as JSON rather than as a foreign character.
func stripBOM(raw []byte) []byte {
	return bytes.TrimPrefix(raw, []byte{0xEF, 0xBB, 0xBF})
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

// sortedKinds keeps the by-kind report stable between runs.
func sortedKinds(byKind map[string]Counts) []string {
	kinds := make([]string, 0, len(byKind))
	for kind := range byKind {
		kinds = append(kinds, kind)
	}
	sort.Strings(kinds)
	return kinds
}

var _ = sortedKinds
