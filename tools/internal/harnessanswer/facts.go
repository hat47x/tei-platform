// Package harnessanswer derives fact tokens from captured TEI CLI output and
// checks whether a harness answer mentions each of them.
//
// This is a mechanical presence check. A token appearing in the answer does not
// show that the answer used it correctly, that the claim is true, or that the
// harness understood the source. The Runbook's reviewer-held Expected remains
// the grading authority; this package only reports which observed facts the
// answer mentions at all, so a human reviewer does not have to re-read raw CLI
// output and answer text side by side.
package harnessanswer

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// sourceDigestPrefixChars is how many digest characters after the algorithm
// label an answer may cite and still match. It is fixed, not derived per
// answer, so the same inputs always produce the same report.
const sourceDigestPrefixChars = 16

// InspectFact is one fact taken from `tei inspect` output for a subject.
type InspectFact struct {
	Key    string   `json:"key"`
	Tokens []string `json:"tokens"`
}

// GroundTruth is the set of facts derived from captured CLI responses.
type GroundTruth struct {
	SubjectKind       string        `json:"subjectKind"`
	SubjectID         string        `json:"subjectId"`
	WorldFingerprint  string        `json:"worldFingerprint"`
	SourceFingerprint string        `json:"sourceIndexFingerprint"`
	Facts             []InspectFact `json:"facts"`
	// SourceArtifacts are artifact paths reported by source-index.
	SourceArtifacts []string `json:"sourceArtifacts"`
	// ValidateOK is the CLI's own ok flag; diagnostics counts follow.
	ValidateOK             *bool          `json:"validateOk"`
	ValidateDiagnostics    int            `json:"validateDiagnostics"`
	ValidateSeverityCounts map[string]int `json:"validateSeverityCounts,omitempty"`
	ValidateCodes          []string       `json:"validateCodes,omitempty"`
}

// Presence is the check result for one fact.
type Presence struct {
	Key     string   `json:"key"`
	Tokens  []string `json:"tokens"`
	Found   bool     `json:"found"`
	Matched string   `json:"matched,omitempty"`
}

// Report is the mechanical comparison of one answer against the ground truth.
type Report struct {
	SubjectID string     `json:"subjectId"`
	Presence  []Presence `json:"presence"`
	Found     int        `json:"found"`
	Missing   int        `json:"missing"`
	// Limits states what this report does not establish.
	Limits []string `json:"limits"`
}

type inspectEnvelope struct {
	OK     bool `json:"ok"`
	Result struct {
		Subject struct {
			Kind string `json:"kind"`
			ID   string `json:"id"`
		} `json:"subject"`
		Canonical struct {
			ID         string `json:"id"`
			Symbol     string `json:"symbol"`
			Attributes []struct {
				ID     string `json:"id"`
				Symbol string `json:"symbol"`
				Type   string `json:"type"`
			} `json:"attributes"`
			Relations []struct {
				ID     string `json:"id"`
				Symbol string `json:"symbol"`
				Target string `json:"target"`
			} `json:"relations"`
		} `json:"canonical"`
		WorldFingerprint       string `json:"worldFingerprint"`
		SourceIndexFingerprint string `json:"sourceIndexFingerprint"`
		Sources                struct {
			Artifacts []struct {
				Path        string `json:"path"`
				Fingerprint string `json:"fingerprint"`
			} `json:"artifacts"`
		} `json:"sources"`
	} `json:"result"`
	Error *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

type validateEnvelope struct {
	OK          bool `json:"ok"`
	Diagnostics []struct {
		Diagnostic struct {
			Code     string `json:"code"`
			Severity string `json:"severity"`
		} `json:"diagnostic"`
	} `json:"diagnostics"`
	Error *struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// decodeFirstJSON reads the first JSON value in body and ignores whatever
// follows it. Captured CLI output holds the JSON envelope followed by the
// PowerShell host's own diagnostic lines ("[stderr] ..."), so decoding the
// trailing text would reject a response that the CLI itself produced.
func decodeFirstJSON(body []byte, target any) error {
	decoder := json.NewDecoder(bytes.NewReader(body))
	if err := decoder.Decode(target); err != nil {
		return err
	}
	return nil
}

// FromInspect builds the subject facts from a captured `tei inspect -json`
// response. The fact tokens are the identifiers the CLI returned, so they stay
// tied to the exact response rather than to a hand-written expectation list.
func FromInspect(body []byte) (GroundTruth, error) {
	var env inspectEnvelope
	if err := decodeFirstJSON(body, &env); err != nil {
		return GroundTruth{}, fmt.Errorf("inspect response is not JSON: %w", err)
	}
	if !env.OK {
		code := "unknown"
		if env.Error != nil && env.Error.Code != "" {
			code = env.Error.Code
		}
		return GroundTruth{}, fmt.Errorf("inspect response is not ok: %s", code)
	}
	gt := GroundTruth{
		SubjectKind:       env.Result.Subject.Kind,
		SubjectID:         env.Result.Subject.ID,
		WorldFingerprint:  env.Result.WorldFingerprint,
		SourceFingerprint: env.Result.SourceIndexFingerprint,
	}
	if env.Result.Canonical.ID != "" {
		gt.Facts = append(gt.Facts, InspectFact{
			Key: "canonical-id", Tokens: []string{env.Result.Canonical.ID},
		})
	}
	for _, attr := range env.Result.Canonical.Attributes {
		// The canonical ID is what the CLI returns, but answers commonly cite
		// the symbol instead ("customerId" rather than
		// "sales.order.customer_id"). Both are accepted, and the matched token
		// is reported so a reviewer can see which form the answer used.
		tokens := []string{}
		if attr.ID != "" {
			tokens = append(tokens, attr.ID)
		}
		if attr.Type != "" {
			tokens = append(tokens, attr.Type)
		}
		if attr.Symbol != "" {
			tokens = append(tokens, attr.Symbol)
		}
		if len(tokens) == 0 {
			continue
		}
		gt.Facts = append(gt.Facts, InspectFact{Key: "attribute:" + attr.ID, Tokens: tokens})
	}
	for _, rel := range env.Result.Canonical.Relations {
		tokens := []string{}
		if rel.ID != "" {
			tokens = append(tokens, rel.ID)
		}
		if rel.Target != "" {
			tokens = append(tokens, rel.Target)
		}
		if rel.Symbol != "" {
			tokens = append(tokens, rel.Symbol)
		}
		if len(tokens) == 0 {
			continue
		}
		gt.Facts = append(gt.Facts, InspectFact{Key: "relation:" + rel.ID, Tokens: tokens})
	}
	for _, artifact := range env.Result.Sources.Artifacts {
		if artifact.Path == "" {
			continue
		}
		tokens := []string{artifact.Path}
		if artifact.Fingerprint != "" {
			// Answers normally cite a digest prefix, so keep the full digest
			// and a bounded prefix. The prefix length is fixed here rather than
			// guessed per answer, so the check stays deterministic.
			tokens = append(tokens, artifact.Fingerprint)
			if idx := strings.Index(artifact.Fingerprint, ":"); idx >= 0 {
				digest := artifact.Fingerprint[idx+1:]
				if len(digest) > sourceDigestPrefixChars {
					digest = digest[:sourceDigestPrefixChars]
				}
				if digest != "" {
					tokens = append(tokens, digest)
				}
			}
		}
		gt.Facts = append(gt.Facts, InspectFact{Key: "source:" + artifact.Path, Tokens: tokens})
		gt.SourceArtifacts = append(gt.SourceArtifacts, artifact.Path)
	}
	if gt.WorldFingerprint != "" {
		gt.Facts = append(gt.Facts, InspectFact{Key: "worldFingerprint", Tokens: []string{gt.WorldFingerprint}})
	}
	return gt, nil
}

// WithValidate adds validate facts to the ground truth.
func (g GroundTruth) WithValidate(body []byte) (GroundTruth, error) {
	var env validateEnvelope
	if err := decodeFirstJSON(body, &env); err != nil {
		return g, fmt.Errorf("validate response is not JSON: %w", err)
	}
	ok := env.OK
	g.ValidateOK = &ok
	g.ValidateDiagnostics = len(env.Diagnostics)
	counts := map[string]int{}
	seen := map[string]bool{}
	for _, diag := range env.Diagnostics {
		severity := diag.Diagnostic.Severity
		if severity == "" {
			severity = "unknown"
		}
		counts[severity]++
		code := diag.Diagnostic.Code
		if code == "" || seen[code] {
			continue
		}
		seen[code] = true
		g.ValidateCodes = append(g.ValidateCodes, code)
	}
	g.ValidateSeverityCounts = counts
	sort.Strings(g.ValidateCodes)
	for _, code := range g.ValidateCodes {
		g.Facts = append(g.Facts, InspectFact{Key: "diagnostic:" + code, Tokens: []string{code}})
	}
	return g, nil
}

// Check reports which facts the answer text mentions. Tokens are matched as
// plain substrings: the check is deliberately mechanical, so it cannot detect
// a token used with the wrong meaning.
func (g GroundTruth) Check(answer string) Report {
	report := Report{
		SubjectID: g.SubjectID,
		Limits: []string{
			"that mentioning a token means the answer used it correctly",
			"that the answer is complete, ordered or free of wrong claims",
			"that the harness understood the source rather than copied a token",
			"anything about source acceptance, authority or provider effects",
		},
	}
	for _, fact := range g.Facts {
		p := Presence{Key: fact.Key, Tokens: fact.Tokens}
		for _, token := range fact.Tokens {
			if token != "" && strings.Contains(answer, token) {
				p.Found = true
				p.Matched = token
				break
			}
		}
		if p.Found {
			report.Found++
		} else {
			report.Missing++
		}
		report.Presence = append(report.Presence, p)
	}
	return report
}
