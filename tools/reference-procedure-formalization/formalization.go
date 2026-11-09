// Package procedureform turns a business procedure written in prose into meaning
// candidates, surfaces the points that block formalization as questions with source
// ranges, and records confirmed answers separately from the prose. It is read-only: it
// never writes a Source, never fills a missing threshold, and never accepts a candidate
// on its own. An inferred candidate stays inferred until a human supplies the answer.
//
// The reference does not read prose itself. It consumes candidates that a caller (a
// person or a generated-AI pass) proposed, and enforces the boundaries that keep that
// proposal apart from accepted meaning.
package procedureform

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

const Schema = "tei.reference.procedure-formalization/v0"

// Segment is one addressable part of a prose document.
type Segment struct {
	ID   string `json:"id"`
	Text string `json:"text"`
}

// SourceText is one revision of a procedure document.
type SourceText struct {
	ID       string    `json:"id"`
	Revision string    `json:"revision"`
	Segments []Segment `json:"segments"`
}

// Candidate kinds.
const (
	KindActor     = "actor"
	KindAction    = "action"
	KindTrigger   = "trigger"
	KindCondition = "condition"
	KindData      = "data"
	KindFunction  = "function"
	KindException = "exception"
	KindResult    = "result"
)

// Candidate is one proposed meaning element, anchored to a source segment. Stated is
// true only when the prose itself states the element; an inferred candidate is a
// supplement that must not reach accepted meaning without an answer.
type Candidate struct {
	ID      string `json:"id"`
	Kind    string `json:"kind"`
	Segment string `json:"segment"`
	Subject string `json:"subject"`
	Value   string `json:"value"`
	Stated  bool   `json:"stated"`
}

// Answer is a human reply to a question, tied to the segment and revision it was made
// about so that a later revision can be re-checked.
type Answer struct {
	ID       string `json:"id"`
	Question string `json:"question"`
	Segment  string `json:"segment"`
	Revision string `json:"revision"`
	Value    string `json:"value"`
	By       string `json:"by"`
}

// Input is one read-only analysis request.
type Input struct {
	Text           SourceText  `json:"text"`
	Candidates     []Candidate `json:"candidates"`
	Answers        []Answer    `json:"answers"`
	KnownFunctions []string    `json:"knownFunctions"`
}

// Question is a missing or contradictory point. It carries its source segment and the
// candidate IDs it concerns.
type Question struct {
	ID       string   `json:"id"`
	Code     string   `json:"code"`
	Subject  string   `json:"subject"`
	Segment  string   `json:"segment"`
	Evidence []string `json:"evidence"`
	Blocks   []string `json:"blocks"`
	Answered bool     `json:"answered"`
	Answer   string   `json:"answer,omitempty"`
}

// Accepted is a stated candidate that no open question blocks.
type Accepted struct {
	Candidate  string `json:"candidate"`
	Provenance string `json:"provenance"`
}

// Blocked is a candidate that cannot be accepted yet, with the reason and questions.
type Blocked struct {
	Candidate string   `json:"candidate"`
	Reason    string   `json:"reason"`
	Questions []string `json:"questions,omitempty"`
}

// AnswerState reports whether an answer still applies to the current revision.
type AnswerState struct {
	Answer string `json:"answer"`
	State  string `json:"state"`
}

// Finding is a diagnostic. It never accepts a candidate.
type Finding struct {
	Code    string `json:"code"`
	Subject string `json:"subject"`
}

// Report is the read-only result.
type Report struct {
	Schema       string        `json:"schema"`
	TextID       string        `json:"textId"`
	Revision     string        `json:"revision"`
	Questions    []Question    `json:"questions"`
	Accepted     []Accepted    `json:"accepted"`
	Blocked      []Blocked     `json:"blocked"`
	AnswerStates []AnswerState `json:"answerStates"`
	Findings     []Finding     `json:"findings"`
	Fingerprint  string        `json:"fingerprint"`
}

// Analyze derives the blocking questions, applies the answers that still hold, and
// accepts only stated candidates that no open question blocks. Every step is
// deterministic; it uses no AI.
func Analyze(in Input) Report {
	segments := map[string]string{}
	for _, s := range in.Text.Segments {
		segments[s.ID] = s.Text
	}

	answerState := map[string]string{}
	answerByQuestion := map[string]Answer{}
	answerStates := make([]AnswerState, 0, len(in.Answers))
	for _, a := range in.Answers {
		state := "applicable"
		switch {
		case segments[a.Segment] == "":
			state = "source-missing"
		case a.Revision != in.Text.Revision:
			state = "stale-revision"
		}
		answerState[a.ID] = state
		answerStates = append(answerStates, AnswerState{Answer: a.ID, State: state})
		if state == "applicable" {
			answerByQuestion[a.Question] = a
		}
	}
	sort.Slice(answerStates, func(i, j int) bool { return answerStates[i].Answer < answerStates[j].Answer })

	questions := deriveQuestions(in)
	// Apply answers: a question is settled only by an answer that still applies and
	// that is not empty.
	for i := range questions {
		a, ok := answerByQuestion[questions[i].ID]
		if !ok || strings.TrimSpace(a.Value) == "" {
			continue
		}
		questions[i].Answered = true
		questions[i].Answer = a.Value
	}
	sortQuestions(questions)

	blocking := map[string][]string{}
	for _, q := range questions {
		if q.Answered {
			continue
		}
		for _, c := range q.Blocks {
			blocking[c] = append(blocking[c], q.ID)
		}
	}

	accepted := []Accepted{}
	blocked := []Blocked{}
	candidates := append([]Candidate(nil), in.Candidates...)
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].ID < candidates[j].ID })
	for _, c := range candidates {
		_, anchored := segments[c.Segment]
		switch {
		case !anchored:
			blocked = append(blocked, Blocked{Candidate: c.ID, Reason: "candidate-anchors-unknown-segment"})
		case !c.Stated:
			blocked = append(blocked, Blocked{Candidate: c.ID, Reason: "inferred-not-accepted"})
		case len(blocking[c.ID]) > 0:
			blocked = append(blocked, Blocked{Candidate: c.ID, Reason: "blocked-by-question", Questions: sortedCopy(blocking[c.ID])})
		default:
			accepted = append(accepted, Accepted{Candidate: c.ID, Provenance: "stated"})
		}
	}

	findings := deriveFindings(in, segments, questions, answerState)
	return Report{
		Schema:       Schema,
		TextID:       in.Text.ID,
		Revision:     in.Text.Revision,
		Questions:    questions,
		Accepted:     accepted,
		Blocked:      blocked,
		AnswerStates: answerStates,
		Findings:     findings,
		Fingerprint:  fingerprint(in),
	}
}

// deriveQuestions surfaces the missing elements and contradictions. It never fills a
// gap: an empty condition becomes a question, not a value.
func deriveQuestions(in Input) []Question {
	questions := []Question{}

	// A stated candidate with an empty value is a missing element.
	for _, c := range in.Candidates {
		if !c.Stated || strings.TrimSpace(c.Value) != "" {
			continue
		}
		code := "missing-" + c.Kind
		q := Question{
			ID:       questionID(code, c.Segment, c.Subject),
			Code:     code,
			Subject:  c.Subject,
			Segment:  c.Segment,
			Evidence: []string{c.ID},
			Blocks:   dependentCandidates(in, c),
		}
		questions = append(questions, q)
	}

	// A stated action with no stated actor in the same segment cannot be formalized.
	for _, c := range in.Candidates {
		if c.Kind != KindAction || !c.Stated {
			continue
		}
		if hasActor(in, c) {
			continue
		}
		code := "missing-actor"
		questions = append(questions, Question{
			ID:       questionID(code, c.Segment, c.Subject),
			Code:     code,
			Subject:  c.Subject,
			Segment:  c.Segment,
			Evidence: []string{c.ID},
			Blocks:   []string{c.ID},
		})
	}

	// Two stated candidates about the same subject with different values contradict.
	groups := map[string][]Candidate{}
	for _, c := range in.Candidates {
		if !c.Stated || c.Kind != KindCondition || strings.TrimSpace(c.Value) == "" {
			continue
		}
		groups[c.Subject] = append(groups[c.Subject], c)
	}
	for subject, group := range groups {
		values := map[string]bool{}
		for _, c := range group {
			values[c.Value] = true
		}
		if len(values) < 2 {
			continue
		}
		evidence := make([]string, 0, len(group))
		blocks := make([]string, 0, len(group))
		for _, c := range group {
			evidence = append(evidence, c.ID)
			blocks = append(blocks, c.ID)
		}
		sort.Strings(evidence)
		sort.Strings(blocks)
		seg := group[0].Segment
		questions = append(questions, Question{
			ID:       questionID("contradictory-candidates", seg, subject),
			Code:     "contradictory-candidates",
			Subject:  subject,
			Segment:  seg,
			Evidence: evidence,
			Blocks:   blocks,
		})
	}

	// A stated function reference that is not known cannot be accepted.
	known := map[string]bool{}
	for _, f := range in.KnownFunctions {
		known[f] = true
	}
	for _, c := range in.Candidates {
		if c.Kind != KindFunction || !c.Stated || known[c.Value] {
			continue
		}
		questions = append(questions, Question{
			ID:       questionID("unknown-function-reference", c.Segment, c.Value),
			Code:     "unknown-function-reference",
			Subject:  c.Value,
			Segment:  c.Segment,
			Evidence: []string{c.ID},
			Blocks:   []string{c.ID},
		})
	}

	return questions
}

func dependentCandidates(in Input, empty Candidate) []string {
	if empty.Kind != KindCondition {
		return []string{empty.ID}
	}
	// An undetermined condition blocks the actions about the same subject in the same
	// segment, not the whole procedure.
	blocks := []string{empty.ID}
	for _, c := range in.Candidates {
		if c.Stated && c.Kind == KindAction && c.Segment == empty.Segment && c.Subject == empty.Subject {
			blocks = append(blocks, c.ID)
		}
	}
	sort.Strings(blocks)
	return blocks
}

func hasActor(in Input, action Candidate) bool {
	for _, c := range in.Candidates {
		if c.Kind == KindActor && c.Stated && c.Segment == action.Segment && c.Subject == action.Subject && strings.TrimSpace(c.Value) != "" {
			return true
		}
	}
	return false
}

func deriveFindings(in Input, segments map[string]string, questions []Question, answerState map[string]string) []Finding {
	findings := []Finding{}
	inferred := false
	for _, c := range in.Candidates {
		if !c.Stated {
			inferred = true
		}
		if _, ok := segments[c.Segment]; !ok {
			findings = append(findings, Finding{Code: "candidate-anchors-unknown-segment", Subject: c.ID})
		}
	}
	if inferred {
		findings = append(findings, Finding{Code: "inferred-not-accepted", Subject: "all"})
	}
	open := 0
	for _, q := range questions {
		if !q.Answered {
			open++
		}
	}
	if open > 0 {
		findings = append(findings, Finding{Code: "unresolved-questions", Subject: fmt.Sprint(open)})
	}
	for id, state := range answerState {
		if state != "applicable" {
			findings = append(findings, Finding{Code: "answer-needs-recheck:" + state, Subject: id})
		}
	}
	sortFindings(findings)
	return findings
}

func questionID(code, segment, subject string) string {
	return strings.Join([]string{code, segment, subject}, ":")
}

func fingerprint(in Input) string {
	candidates := append([]Candidate(nil), in.Candidates...)
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].ID < candidates[j].ID })
	answers := append([]Answer(nil), in.Answers...)
	sort.Slice(answers, func(i, j int) bool { return answers[i].ID < answers[j].ID })
	functions := append([]string(nil), in.KnownFunctions...)
	sort.Strings(functions)
	payload := struct {
		Schema     string      `json:"schema"`
		Text       SourceText  `json:"text"`
		Candidates []Candidate `json:"candidates"`
		Answers    []Answer    `json:"answers"`
		Functions  []string    `json:"functions"`
	}{Schema, in.Text, candidates, answers, functions}
	raw, err := json.Marshal(payload)
	if err != nil {
		raw = []byte(fmt.Sprintf("%v", payload))
	}
	return fmt.Sprintf("sha256:%x", sha256.Sum256(raw))
}

func sortQuestions(qs []Question) {
	sort.Slice(qs, func(i, j int) bool { return qs[i].ID < qs[j].ID })
}

func sortFindings(findings []Finding) {
	sort.Slice(findings, func(i, j int) bool {
		if findings[i].Code != findings[j].Code {
			return findings[i].Code < findings[j].Code
		}
		return findings[i].Subject < findings[j].Subject
	})
}

func sortedCopy(s []string) []string {
	out := append([]string(nil), s...)
	sort.Strings(out)
	return out
}
