// Package rulecandidate proposes rule hypotheses from input and output examples and
// refuses to confirm a single rule from a few examples. It is read-only: it does not
// write a Source, accept a rule, or grant execution authority. A hypothesis is a plain
// string form, so it can establish a display or normalization convention but never a
// business identity or an authorization. A human selection can be executed
// deterministically without any AI.
package rulecandidate

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"unicode"
)

const Schema = "tei.reference.rule-candidate-discovery/v0"

// Delimiter is the separator the prefix hypotheses split on. It is fixed so that
// the reference does not read a separator convention out of the examples themselves.
const Delimiter = "-"

// Example is one observed input and output pair, with the origin the caller supplied.
type Example struct {
	ID     string `json:"id"`
	Input  string `json:"input"`
	Output string `json:"output"`
}

// Kind names a rule form.
type Kind string

const (
	// PrefixBeforeFirstDelimiter keeps the part before the first delimiter.
	PrefixBeforeFirstDelimiter Kind = "prefix-before-first-delimiter"
	// PrefixBeforeLastDelimiter keeps the part before the last delimiter.
	PrefixBeforeLastDelimiter Kind = "prefix-before-last-delimiter"
	// LeadingAlpha keeps the leading run of letters.
	LeadingAlpha Kind = "leading-alpha"
	// StripTrailingDigits removes trailing digits.
	StripTrailingDigits Kind = "strip-trailing-digits"
	// Identity returns the input unchanged.
	Identity Kind = "identity"
	// LiteralMapping looks the input up in the observed pairs. It does not
	// generalize to any unseen input.
	LiteralMapping Kind = "literal-mapping"
)

// OrderedKinds is the stable evaluation order.
var OrderedKinds = []Kind{
	PrefixBeforeFirstDelimiter,
	PrefixBeforeLastDelimiter,
	LeadingAlpha,
	StripTrailingDigits,
	Identity,
	LiteralMapping,
}

// Finding is a diagnostic. It never confirms a rule.
type Finding struct {
	Code    string `json:"code"`
	Subject string `json:"subject"`
}

// HypothesisView describes one hypothesis against the examples.
type HypothesisView struct {
	Kind          Kind `json:"kind"`
	Generalizable bool `json:"generalizable"`
	FormatOnly    bool `json:"formatOnly"`
	FitsAll       bool `json:"fitsAll"`
}

// KindOutput is the output of one hypothesis on one probe.
type KindOutput struct {
	Kind   Kind   `json:"kind"`
	Output string `json:"output"`
}

// ProbeView is one proposed boundary input and what the fitting hypotheses make of it.
type ProbeView struct {
	Input    string       `json:"input"`
	Outputs  []KindOutput `json:"outputs"`
	Disagree bool         `json:"disagree"`
}

// Report is the read-only result of Analyze.
type Report struct {
	Schema         string           `json:"schema"`
	ExampleCount   int              `json:"exampleCount"`
	Examples       []Example        `json:"examples"`
	Hypotheses     []HypothesisView `json:"hypotheses"`
	Probes         []ProbeView      `json:"probes"`
	Ambiguous      bool             `json:"ambiguous"`
	ReviewRequired bool             `json:"reviewRequired"`
	Findings       []Finding        `json:"findings"`
	Fingerprint    string           `json:"fingerprint"`
}

// apply evaluates one hypothesis kind on one input. It reports false when the form
// has no answer for the input, so the caller can tell "did not fit" from "empty".
func apply(kind Kind, literal map[string]string, input string) (string, bool) {
	switch kind {
	case PrefixBeforeFirstDelimiter:
		if i := strings.Index(input, Delimiter); i > 0 {
			return input[:i], true
		}
		return "", false
	case PrefixBeforeLastDelimiter:
		if i := strings.LastIndex(input, Delimiter); i > 0 {
			return input[:i], true
		}
		return "", false
	case LeadingAlpha:
		n := 0
		for _, r := range input {
			if !unicode.IsLetter(r) {
				break
			}
			n += len(string(r))
		}
		if n == 0 {
			return "", false
		}
		return input[:n], true
	case StripTrailingDigits:
		trimmed := strings.TrimRightFunc(input, unicode.IsDigit)
		if trimmed == input {
			return "", false
		}
		return trimmed, true
	case Identity:
		return input, true
	case LiteralMapping:
		out, ok := literal[input]
		return out, ok
	default:
		return "", false
	}
}

// fitsAll reports whether the hypothesis answers every example with its output.
func fitsAll(kind Kind, literal map[string]string, examples []Example) bool {
	for _, e := range examples {
		out, ok := apply(kind, literal, e.Input)
		if !ok || out != e.Output {
			return false
		}
	}
	return true
}

func literalMap(examples []Example) map[string]string {
	m := make(map[string]string, len(examples))
	for _, e := range examples {
		m[e.Input] = e.Output
	}
	return m
}

func generalizable(kind Kind) bool { return kind != LiteralMapping }

// proposeProbes suggests boundary inputs from the example inputs. A probe with no
// delimiter tests the prefix forms against the letter-run form, and an inserted
// segment tests the first-delimiter form against the last-delimiter form.
func proposeProbes(examples []Example) []string {
	seen := map[string]bool{}
	for _, e := range examples {
		seen[e.Input] = true
	}
	probes := map[string]bool{}
	for _, e := range examples {
		if !strings.Contains(e.Input, Delimiter) {
			continue
		}
		probes[strings.ReplaceAll(e.Input, Delimiter, "")] = true
		last := strings.LastIndex(e.Input, Delimiter)
		probes[e.Input[:last]+Delimiter+"X"+e.Input[last:]] = true
	}
	out := make([]string, 0, len(probes))
	for p := range probes {
		if !seen[p] {
			out = append(out, p)
		}
	}
	sort.Strings(out)
	return out
}

// Analyze fits every hypothesis to the examples, proposes boundary inputs where the
// fitting hypotheses disagree, and never confirms a single rule. ReviewRequired is
// always true: a few examples cannot settle a rule.
func Analyze(examples []Example) Report {
	sorted := append([]Example(nil), examples...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].ID < sorted[j].ID })

	literal := literalMap(sorted)
	views := make([]HypothesisView, 0, len(OrderedKinds))
	fitting := make([]Kind, 0, len(OrderedKinds))
	for _, k := range OrderedKinds {
		fits := fitsAll(k, literal, sorted)
		views = append(views, HypothesisView{
			Kind:          k,
			Generalizable: generalizable(k),
			FormatOnly:    true,
			FitsAll:       fits,
		})
		if fits {
			fitting = append(fitting, k)
		}
	}

	generalFit := 0
	for _, k := range fitting {
		if generalizable(k) {
			generalFit++
		}
	}

	probes := proposeProbes(sorted)
	probeViews := make([]ProbeView, 0, len(probes))
	for _, p := range probes {
		outputs := make([]KindOutput, 0, len(fitting))
		distinct := map[string]bool{}
		for _, k := range fitting {
			out, ok := apply(k, literal, p)
			if !ok {
				continue
			}
			outputs = append(outputs, KindOutput{Kind: k, Output: out})
			distinct[out] = true
		}
		probeViews = append(probeViews, ProbeView{Input: p, Outputs: outputs, Disagree: len(distinct) > 1})
	}

	findings := []Finding{{Code: "format-only-not-business-meaning", Subject: "all-hypotheses"}}
	if len(sorted) < 3 {
		findings = append(findings, Finding{Code: "insufficient-examples", Subject: fmt.Sprint(len(sorted))})
	}
	if generalFit >= 2 {
		findings = append(findings, Finding{Code: "multiple-hypotheses-fit", Subject: fmt.Sprint(generalFit)})
	}
	sortFindings(findings)

	return Report{
		Schema:         Schema,
		ExampleCount:   len(sorted),
		Examples:       sorted,
		Hypotheses:     views,
		Probes:         probeViews,
		Ambiguous:      generalFit >= 2,
		ReviewRequired: true,
		Findings:       findings,
		Fingerprint:    fingerprint(sorted, fitting),
	}
}

func fingerprint(examples []Example, fitting []Kind) string {
	payload := struct {
		Schema   string    `json:"schema"`
		Examples []Example `json:"examples"`
		Fitting  []Kind    `json:"fitting"`
	}{Schema, examples, fitting}
	raw, err := json.Marshal(payload)
	if err != nil {
		raw = []byte(fmt.Sprintf("%v", payload))
	}
	return fmt.Sprintf("sha256:%x", sha256.Sum256(raw))
}

// Selection is a human choice of one hypothesis, tied to the examples it was chosen
// from.
type Selection struct {
	Kind       Kind     `json:"kind"`
	ExampleIDs []string `json:"exampleIds"`
}

// Confirmed is a recorded selection. It never claims a business meaning.
type Confirmed struct {
	Selected      Kind     `json:"selected"`
	ExampleIDs    []string `json:"exampleIds"`
	Meaning       string   `json:"meaning"`
	Deterministic bool     `json:"deterministic"`
}

// Confirm records a human selection. A format rule can be confirmed only as a format
// rule: a claim that it establishes a business identity or an authorization is refused,
// because the form of a string cannot establish either.
func Confirm(sel Selection, examples []Example, claim string) (Confirmed, []Finding) {
	findings := []Finding{}
	if !knownKind(sel.Kind) {
		findings = append(findings, Finding{Code: "unknown-hypothesis", Subject: string(sel.Kind)})
	}
	ids := map[string]bool{}
	for _, e := range examples {
		ids[e.ID] = true
	}
	chosen := append([]string(nil), sel.ExampleIDs...)
	if len(chosen) == 0 {
		findings = append(findings, Finding{Code: "selection-without-examples", Subject: string(sel.Kind)})
	}
	for _, id := range chosen {
		if !ids[id] {
			findings = append(findings, Finding{Code: "selection-references-unknown-example", Subject: id})
		}
	}
	sort.Strings(chosen)
	switch claim {
	case "format":
	case "business-identity":
		findings = append(findings, Finding{Code: "format-cannot-establish-business-identity", Subject: string(sel.Kind)})
	case "authorization":
		findings = append(findings, Finding{Code: "format-cannot-establish-authorization", Subject: string(sel.Kind)})
	default:
		findings = append(findings, Finding{Code: "unknown-meaning-claim", Subject: claim})
	}
	if len(findings) > 0 {
		sortFindings(findings)
		return Confirmed{}, findings
	}
	return Confirmed{Selected: sel.Kind, ExampleIDs: chosen, Meaning: "format-only", Deterministic: true}, nil
}

// Execute applies a confirmed selection to a new input. It is deterministic and uses
// no AI. It reports false when the selected form has no answer for the input.
func Execute(sel Selection, examples []Example, input string) (string, bool) {
	if !knownKind(sel.Kind) {
		return "", false
	}
	return apply(sel.Kind, literalMap(examples), input)
}

func knownKind(k Kind) bool {
	for _, candidate := range OrderedKinds {
		if candidate == k {
			return true
		}
	}
	return false
}

func sortFindings(findings []Finding) {
	sort.Slice(findings, func(i, j int) bool {
		if findings[i].Code != findings[j].Code {
			return findings[i].Code < findings[j].Code
		}
		return findings[i].Subject < findings[j].Subject
	})
}
