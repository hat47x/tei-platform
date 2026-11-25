// Package lexical is a deterministic, offline baseline provider for the
// reference cognition port.
//
// #5807 and #5809 both require a deterministic baseline to be measured against
// the real local model on the same inputs, with accuracy, misclassification,
// abstention, and format-violation rates reported separately. This package is
// that baseline. It opens no socket, spawns no process, and calls no model: two
// runs over the same request always return the same proposal.
//
// It is deliberately weak. It scores each declared option by the character
// bigram overlap between the observation statement and the option description,
// which is a lexical signal and nothing more. It has no domain lexicon, so it
// cannot know that "請求" means billing beyond the description a caller supplied.
// That weakness is the point: a baseline that already encodes the answer would
// make the comparison meaningless.
//
// It is a reference provider, not a product feature, and it proposes only. The
// port checks subject, option set, basis, and limits; this package never
// accepts, applies, or grants authority.
package lexical

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"unicode"
)

// Model is the provider model identifier this package reports. It names the
// scoring rule, so a stored receipt can be traced to the rule that produced it.
const Model = "lexical-bigram-v0"

// Version is the rule revision. Bump it when the scoring changes: the receipt
// records both, and a comparison that mixes revisions is not a comparison.
const Version = "1"

const (
	// minSharedBigrams is the overlap below which the baseline abstains. A
	// description is often only two or three words long, so requiring two
	// shared pairs would abstain on exactly the short option sets this baseline
	// is meant to handle; one distinctive pair plus the tie and margin checks
	// below is the trade this rule makes. What it cannot do is separate two
	// options whose descriptions share the one word the statement uses, which
	// is why a tie abstains rather than picking.
	minSharedBigrams = 1
	// marginNumerator/marginDenominator require the best option to beat the
	// runner-up by this fraction of its own score. A single shared pair against
	// a long description and a single shared pair against a short one are not
	// equally convincing, and without a margin the longer description's extra
	// unrelated pairs would not matter while equal scores would silently pick
	// the earlier option. 1/4 is a reference constant, not a tuned value.
	marginNumerator   = 1
	marginDenominator = 4
)

type option struct {
	ID          string `json:"id"`
	Description string `json:"description"`
}

type input struct {
	Observations []struct {
		ID        string `json:"id"`
		Statement string `json:"statement"`
	} `json:"observations"`
	Constraints []string `json:"constraints"`
	Choice      *struct {
		ObservationID string   `json:"observationId"`
		Options       []option `json:"options"`
	} `json:"choice"`
}

// Invoke returns a proposal for one choice request. It reports unknown when the
// statement and the descriptions share too little, or when the best two options
// score the same — an ambiguous request is not a coin toss.
func Invoke(_ context.Context, contextRaw []byte) ([]byte, error) {
	var in input
	if err := json.Unmarshal(contextRaw, &in); err != nil {
		return nil, err
	}
	if in.Choice == nil {
		return nil, errors.New("choice question required")
	}
	var statement string
	for _, observation := range in.Observations {
		if observation.ID == in.Choice.ObservationID {
			statement = observation.Statement
			break
		}
	}
	if strings.TrimSpace(statement) == "" {
		return nil, errors.New("choice observation absent")
	}
	if len(in.Choice.Options) < 2 {
		return nil, errors.New("choice requires at least two options")
	}

	unknown := func(reason string) ([]byte, error) {
		return marshal(unknownProposal(reason))
	}

	statementGrams := bigrams(statement)
	bestIndex := -1
	var bestShared, bestSize int
	for index, cand := range in.Choice.Options {
		descriptionGrams := bigrams(cand.Description)
		shared := intersectionSize(statementGrams, descriptionGrams)
		// Exact integer comparison of shared/max(size) avoids float rounding
		// turning a tie into an arbitrary winner.
		if bestIndex == -1 || better(shared, len(descriptionGrams), bestShared, bestSize) {
			bestIndex, bestShared, bestSize = index, shared, len(descriptionGrams)
		}
	}

	if bestShared < minSharedBigrams {
		return unknown("statement and option descriptions share too little to choose")
	}
	// A second option matching equally well means the request does not
	// distinguish them; abstaining is the honest answer.
	for index, cand := range in.Choice.Options {
		if index == bestIndex {
			continue
		}
		shared := intersectionSize(statementGrams, bigrams(cand.Description))
		if equal(shared, len(bigrams(cand.Description)), bestShared, bestSize) {
			return unknown("two options match the statement equally well")
		}
	}
	// Nor is a hair's-breadth win an answer: the runner-up must trail by the
	// declared margin, or the request is treated as undecided.
	runnerUpShared, runnerUpSize := 0, 0
	for index, cand := range in.Choice.Options {
		if index == bestIndex {
			continue
		}
		shared := intersectionSize(statementGrams, bigrams(cand.Description))
		if better(shared, len(bigrams(cand.Description)), runnerUpShared, runnerUpSize) {
			runnerUpShared, runnerUpSize = shared, len(bigrams(cand.Description))
		}
	}
	if !clearsMargin(bestShared, bestSize, runnerUpShared, runnerUpSize) {
		return unknown("the best option does not lead the runner-up clearly enough")
	}

	selected := in.Choice.Options[bestIndex]
	proposal := proposal{
		Schema: "tei.reference.cognition-proposal/v0",
		State:  "proposed",
		Candidates: []candidate{{
			ID:      "choice-1",
			Kind:    "choice",
			Subject: in.Choice.ObservationID,
			Payload: map[string]string{"optionId": selected.ID},
			Basis:   []string{"observation:" + in.Choice.ObservationID},
		}},
	}
	return marshal(proposal)
}

type candidate struct {
	ID      string            `json:"id"`
	Kind    string            `json:"kind"`
	Subject string            `json:"subject"`
	Payload map[string]string `json:"payload"`
	Basis   []string          `json:"basis"`
}

type proposal struct {
	Schema      string      `json:"schema"`
	State       string      `json:"state"`
	Candidates  []candidate `json:"candidates"`
	Diagnostics []string    `json:"diagnostics,omitempty"`
}

func unknownProposal(reason string) proposal {
	return proposal{
		Schema:      "tei.reference.cognition-proposal/v0",
		State:       "unknown",
		Candidates:  []candidate{},
		Diagnostics: []string{reason},
	}
}

func marshal(p proposal) ([]byte, error) {
	if p.Candidates == nil {
		p.Candidates = []candidate{}
	}
	return json.Marshal(p)
}

// bigrams reduces text to its set of adjacent character pairs, ignoring case,
// whitespace, and punctuation. A set, not a count: repeated words in a long
// description must not outweigh a single distinctive match in a short statement.
func bigrams(text string) map[string]bool {
	runes := make([]rune, 0, len(text))
	for _, r := range strings.ToLower(text) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			runes = append(runes, r)
		}
	}
	grams := make(map[string]bool, len(runes))
	for index := 0; index+1 < len(runes); index++ {
		grams[string(runes[index:index+2])] = true
	}
	return grams
}

func intersectionSize(left, right map[string]bool) int {
	if len(left) > len(right) {
		left, right = right, left
	}
	shared := 0
	for gram := range left {
		if right[gram] {
			shared++
		}
	}
	return shared
}

// better reports shared1/size1 > shared2/size2 by cross-multiplication.
func better(shared1, size1, shared2, size2 int) bool {
	if size1 == 0 || size2 == 0 {
		return size1 != 0
	}
	return shared1*size2 > shared2*size1
}

// equal reports shared1/size1 == shared2/size2 by cross-multiplication.
func equal(shared1, size1, shared2, size2 int) bool {
	if size1 == 0 || size2 == 0 {
		return size1 == size2
	}
	return shared1*size2 == shared2*size1
}

// clearsMargin reports whether the best score leads the runner-up by at least
// marginNumerator/marginDenominator of itself, by cross-multiplication so no
// float rounding decides a borderline case.
func clearsMargin(bestShared, bestSize, runnerUpShared, runnerUpSize int) bool {
	if bestSize == 0 || bestShared == 0 {
		return false
	}
	// gap = best - runnerUp, compared against best * margin.
	gapNumerator := bestShared*runnerUpSize - runnerUpShared*bestSize
	return gapNumerator*marginDenominator >= bestShared*bestSize*marginNumerator
}
