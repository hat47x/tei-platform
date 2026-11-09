// Package procedureexpand expands an accepted business procedure into realization
// options (documentation, human work list, non-screen function, external service, AI)
// and decides which options are ready. It is read-only: it never writes a Source,
// never auto-determines an authority, data item or condition the procedure does not
// state, and never treats a screen as execution completion.
package procedureexpand

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

const Schema = "tei.reference.procedure-expansion/v0"

// Step is one action of an accepted business procedure.
type Step struct {
	ID             string   `json:"id"`
	Action         string   `json:"action"`
	Nature         string   `json:"nature"` // machine | manual | external | undetermined
	Authority      string   `json:"authority"`
	Conditions     []string `json:"conditions"`
	ExternalEffect bool     `json:"externalEffect"`
	ResultUnknown  bool     `json:"resultUnknown"`
}

// Binding connects a step to a realization target.
type Binding struct {
	Step      string `json:"step"`
	Target    string `json:"target"` // function | screen | external | ai | explain
	Ref       string `json:"ref"`
	Authority string `json:"authority"`
}

// Input is one read-only analysis request.
type Input struct {
	Procedure string    `json:"procedure"`
	Steps     []Step    `json:"steps"`
	Bindings  []Binding `json:"bindings"`
}

// Finding is a diagnostic. It never enables execution.
type Finding struct {
	Code    string `json:"code"`
	Subject string `json:"subject"`
}

// Option is the readiness of one realization target for one step.
type Option struct {
	Step    string   `json:"step"`
	Target  string   `json:"target"`
	Ref     string   `json:"ref"`
	Ready   bool     `json:"ready"`
	Missing []string `json:"missing"`
}

// StepPlan is the realization status of one step.
type StepPlan struct {
	Step             string   `json:"step"`
	Action           string   `json:"action"`
	Targets          []string `json:"targets"`
	Status           string   `json:"status"` // executable | manual | external | unresolved
	MeaningUnchanged bool     `json:"meaningUnchanged"`
}

// Summary counts the step statuses.
type Summary struct {
	Executable int `json:"executable"`
	Manual     int `json:"manual"`
	External   int `json:"external"`
	Unresolved int `json:"unresolved"`
}

// Report is the read-only result.
type Report struct {
	Schema      string     `json:"schema"`
	Steps       []StepPlan `json:"steps"`
	Options     []Option   `json:"options"`
	Summary     Summary    `json:"summary"`
	Findings    []Finding  `json:"findings"`
	Fingerprint string     `json:"fingerprint"`
}

func hasUndetermined(conds []string) bool {
	for _, c := range conds {
		t := strings.TrimSpace(c)
		if t == "" || strings.EqualFold(t, "<unresolved>") {
			return true
		}
	}
	return false
}

func evaluate(s Step, b Binding) Option {
	missing := []string{}
	switch b.Target {
	case "function":
		if s.Nature != "machine" {
			missing = append(missing, "nature-not-machine")
		}
		if hasUndetermined(s.Conditions) {
			missing = append(missing, "undetermined-condition")
		}
		if s.ResultUnknown {
			missing = append(missing, "result-unknown")
		}
		if b.Authority != "" && b.Authority != s.Authority {
			missing = append(missing, "authority-not-in-procedure")
		}
	case "external":
		if !s.ExternalEffect {
			missing = append(missing, "external-effect-not-declared")
		}
		if s.ResultUnknown {
			missing = append(missing, "result-unknown")
		}
	case "ai":
		missing = append(missing, "requires-acceptance")
	case "screen", "explain":
		// A screen or a document never establishes execution; it is always available.
	default:
		missing = append(missing, "unknown-target")
	}
	sort.Strings(missing)
	return Option{Step: s.ID, Target: b.Target, Ref: b.Ref, Ready: len(missing) == 0, Missing: missing}
}

func statusOf(s Step, bs []Binding) string {
	if len(bs) == 0 {
		return "unresolved"
	}
	hasAuto := false
	functionReady := false
	externalReady := false
	for _, b := range bs {
		switch b.Target {
		case "function":
			hasAuto = true
			if evaluate(s, b).Ready {
				functionReady = true
			}
		case "external":
			hasAuto = true
			if evaluate(s, b).Ready {
				externalReady = true
			}
		case "ai":
			hasAuto = true
		}
	}
	switch {
	case functionReady:
		return "executable"
	case externalReady:
		return "external"
	case !hasAuto:
		return "manual"
	default:
		// The step intends automatic execution but no automatic option is ready.
		return "unresolved"
	}
}

func onlyHuman(bs []Binding) bool {
	for _, b := range bs {
		if b.Target != "screen" && b.Target != "explain" {
			return false
		}
	}
	return true
}

// Plan enumerates the realization options of each step and decides the status. It never
// changes the procedure's meaning or authority.
func Plan(in Input) Report {
	steps := append([]Step(nil), in.Steps...)
	sort.Slice(steps, func(i, j int) bool { return steps[i].ID < steps[j].ID })

	byStep := map[string][]Binding{}
	for _, b := range in.Bindings {
		byStep[b.Step] = append(byStep[b.Step], b)
	}

	plans := make([]StepPlan, 0, len(steps))
	options := []Option{}
	findings := []Finding{}
	summary := Summary{}

	for _, s := range steps {
		bs := byStep[s.ID]
		sort.Slice(bs, func(i, j int) bool {
			if bs[i].Target != bs[j].Target {
				return bs[i].Target < bs[j].Target
			}
			return bs[i].Ref < bs[j].Ref
		})
		targets := []string{}
		for _, b := range bs {
			opt := evaluate(s, b)
			options = append(options, opt)
			targets = append(targets, b.Target)
			for _, m := range opt.Missing {
				findings = append(findings, Finding{Code: m, Subject: s.ID})
			}
		}
		sort.Strings(targets)
		status := statusOf(s, bs)
		plans = append(plans, StepPlan{
			Step:             s.ID,
			Action:           s.Action,
			Targets:          targets,
			Status:           status,
			MeaningUnchanged: onlyHuman(bs),
		})
		switch status {
		case "executable":
			summary.Executable++
		case "manual":
			summary.Manual++
		case "external":
			summary.External++
		default:
			summary.Unresolved++
		}
	}

	findings = dedupFindings(findings)
	return Report{
		Schema:      Schema,
		Steps:       plans,
		Options:     options,
		Summary:     summary,
		Findings:    findings,
		Fingerprint: fingerprint(plans, options),
	}
}

func dedupFindings(findings []Finding) []Finding {
	sort.Slice(findings, func(i, j int) bool {
		if findings[i].Code != findings[j].Code {
			return findings[i].Code < findings[j].Code
		}
		return findings[i].Subject < findings[j].Subject
	})
	out := findings[:0]
	for i, f := range findings {
		if i > 0 && findings[i-1] == f {
			continue
		}
		out = append(out, f)
	}
	return out
}

func fingerprint(plans []StepPlan, options []Option) string {
	payload := struct {
		Schema  string     `json:"schema"`
		Steps   []StepPlan `json:"steps"`
		Options []Option   `json:"options"`
	}{Schema, plans, options}
	raw, err := json.Marshal(payload)
	if err != nil {
		raw = []byte(fmt.Sprintf("%v", payload))
	}
	return fmt.Sprintf("sha256:%x", sha256.Sum256(raw))
}

// Comparison measures the effect of a procedure change.
type Comparison struct {
	AddedSteps     []string `json:"addedSteps"`
	RemovedSteps   []string `json:"removedSteps"`
	ChangedSteps   []string `json:"changedSteps"`
	AffectedAssets []string `json:"affectedAssets"`
}

// Compare plans both procedures and reports the affected steps and bound assets, so an
// unrelated asset is not touched by a procedure change.
func Compare(before, after Input) Comparison {
	b, a := Plan(before), Plan(after)

	status := func(r Report) map[string]string {
		m := map[string]string{}
		for _, s := range r.Steps {
			m[s.Step] = s.Status
		}
		return m
	}
	bs, as := status(b), status(a)

	added, removed, changed := []string{}, []string{}, []string{}
	for id, st := range as {
		prev, ok := bs[id]
		if !ok {
			added = append(added, id)
			continue
		}
		if prev != st {
			changed = append(changed, id)
		}
	}
	for id := range bs {
		if _, ok := as[id]; !ok {
			removed = append(removed, id)
		}
	}

	ready := func(r Report) map[string]bool {
		m := map[string]bool{}
		for _, o := range r.Options {
			m[o.Step+"|"+o.Target+"|"+o.Ref] = o.Ready
		}
		return m
	}
	br, ar := ready(b), ready(a)
	assets := map[string]bool{}
	for key, r := range ar {
		if prev, ok := br[key]; ok && prev != r {
			parts := strings.Split(key, "|")
			if parts[2] != "" {
				assets[parts[2]] = true
			}
		}
	}
	affected := []string{}
	for ref := range assets {
		affected = append(affected, ref)
	}
	sort.Strings(affected)
	sort.Strings(added)
	sort.Strings(removed)
	sort.Strings(changed)

	return Comparison{AddedSteps: added, RemovedSteps: removed, ChangedSteps: changed, AffectedAssets: affected}
}
