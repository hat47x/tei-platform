// Package prototyping checks an exploratory prototyping workflow: it keeps hypotheses
// provisional, distinguishes an AI candidate from a measured observation, promotes only
// a selected part of a prototype into the formal design, and records unmeasured effort
// as unmeasured. It is read-only: it never promotes a hypothesis to an accepted spec and
// never touches the Source of an existing application.
package prototyping

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sort"
)

const Schema = "tei.reference.exploratory-prototyping/v0"

// Iteration is one exploratory loop: hypothesis, input, execution, observation, decision.
type Iteration struct {
	ID           string `json:"id"`
	Hypothesis   string `json:"hypothesis"`
	InputFixture string `json:"inputFixture"`
	Revision     string `json:"revision"`
	Expected     string `json:"expected"`
	Observed     string `json:"observed"`
	// Observation is where the observation came from: execution | ai-candidate | none.
	Observation string `json:"observation"`
	// NextDecision is continue | adopt | discard | unresolved.
	NextDecision string `json:"nextDecision"`
	// SelfDerived is true when the expected value was produced by the prototype itself.
	SelfDerived bool `json:"selfDerived"`
}

// Selection is a proposed adoption unit. Kind is one of
// data | rule | function | adapter | scenario | artifact | discard | hold.
type Selection struct {
	Iteration string `json:"iteration"`
	Kind      string `json:"kind"`
	Ref       string `json:"ref"`
}

// Input is one read-only analysis request.
type Input struct {
	// Profile is exploratory | rigorous.
	Profile         string      `json:"profile"`
	PluginAvailable bool        `json:"pluginAvailable"`
	Iterations      []Iteration `json:"iterations"`
	Selections      []Selection `json:"selections"`
}

// Finding is a diagnostic. It never promotes anything.
type Finding struct {
	Code    string `json:"code"`
	Subject string `json:"subject"`
}

// IterationView is the reproducibility view of one iteration.
type IterationView struct {
	ID           string `json:"id"`
	Status       string `json:"status"`
	Measured     bool   `json:"measured"`
	Reproducible bool   `json:"reproducible"`
}

// AdoptionView records one selection outcome.
type AdoptionView struct {
	Iteration string `json:"iteration"`
	Kind      string `json:"kind"`
	Ref       string `json:"ref"`
	Accepted  bool   `json:"accepted"`
	Reason    string `json:"reason,omitempty"`
}

// Summary counts the adoption outcomes.
type Summary struct {
	Iterations int `json:"iterations"`
	Adopted    int `json:"adopted"`
	Discarded  int `json:"discarded"`
	Held       int `json:"held"`
}

// Report is the read-only result.
type Report struct {
	Schema      string          `json:"schema"`
	Profile     string          `json:"profile"`
	Applicable  bool            `json:"applicable"`
	Iterations  []IterationView `json:"iterations"`
	Adoptions   []AdoptionView  `json:"adoptions"`
	Adopted     []string        `json:"adopted"`
	Discarded   []string        `json:"discarded"`
	Held        []string        `json:"held"`
	Summary     Summary         `json:"summary"`
	Findings    []Finding       `json:"findings"`
	Fingerprint string          `json:"fingerprint"`
}

var promotable = map[string]bool{
	"data": true, "rule": true, "function": true,
	"adapter": true, "scenario": true, "artifact": true,
}

var meaningBearing = map[string]bool{"data": true, "rule": true, "function": true}

func reproducible(it Iteration) bool {
	return it.Observation == "execution" && it.Revision != "" && it.InputFixture != ""
}

func label(sel Selection) string {
	return sel.Kind + ":" + sel.Ref
}

// Plan keeps the hypotheses provisional and accepts only a selected part of a prototype.
func Plan(in Input) Report {
	iterations := append([]Iteration(nil), in.Iterations...)
	sort.Slice(iterations, func(i, j int) bool { return iterations[i].ID < iterations[j].ID })
	byID := map[string]Iteration{}
	for _, it := range iterations {
		byID[it.ID] = it
	}

	findings := []Finding{}
	views := make([]IterationView, 0, len(iterations))
	for _, it := range iterations {
		measured := it.Observation == "execution"
		view := IterationView{ID: it.ID, Status: it.NextDecision, Measured: measured, Reproducible: reproducible(it)}
		views = append(views, view)
		if !measured {
			findings = append(findings, Finding{Code: "observation-not-measured", Subject: it.ID})
		}
		if it.SelfDerived {
			findings = append(findings, Finding{Code: "self-derived-expectation", Subject: it.ID})
		}
	}
	if len(iterations) < 2 {
		findings = append(findings, Finding{Code: "insufficient-iterations", Subject: fmt.Sprint(len(iterations))})
	}

	applicable := in.PluginAvailable
	if !in.PluginAvailable {
		findings = append(findings, Finding{Code: "plugin-unavailable", Subject: "profile"})
	}
	if in.Profile != "exploratory" {
		findings = append(findings, Finding{Code: "profile-not-exploratory", Subject: in.Profile})
	}

	selections := append([]Selection(nil), in.Selections...)
	sort.Slice(selections, func(i, j int) bool {
		if selections[i].Iteration != selections[j].Iteration {
			return selections[i].Iteration < selections[j].Iteration
		}
		if selections[i].Kind != selections[j].Kind {
			return selections[i].Kind < selections[j].Kind
		}
		return selections[i].Ref < selections[j].Ref
	})

	adoptions := make([]AdoptionView, 0, len(selections))
	adopted := []string{}
	discarded := []string{}
	held := []string{}
	for _, sel := range selections {
		v := AdoptionView{Iteration: sel.Iteration, Kind: sel.Kind, Ref: sel.Ref}
		it, ok := byID[sel.Iteration]
		switch {
		case sel.Kind == "discard":
			v.Reason = "discarded"
			discarded = append(discarded, label(sel))
		case sel.Kind == "hold":
			v.Reason = "held"
			held = append(held, label(sel))
		case !promotable[sel.Kind]:
			v.Reason = "kind-not-promotable"
			findings = append(findings, Finding{Code: "kind-not-promotable", Subject: sel.Ref})
		case !applicable:
			v.Reason = "plugin-unavailable"
		case in.Profile != "exploratory":
			v.Reason = "profile-not-exploratory"
		case !ok:
			v.Reason = "unknown-iteration"
			findings = append(findings, Finding{Code: "unknown-iteration", Subject: sel.Iteration})
		case it.SelfDerived:
			v.Reason = "self-derived-expectation"
		case !reproducible(it):
			v.Reason = "iteration-not-reproducible"
			findings = append(findings, Finding{Code: "iteration-not-reproducible", Subject: it.ID})
		case meaningBearing[sel.Kind] && it.NextDecision != "adopt":
			v.Reason = "iteration-not-accepted"
			findings = append(findings, Finding{Code: "iteration-not-accepted", Subject: it.ID})
		default:
			v.Accepted = true
			adopted = append(adopted, label(sel))
		}
		adoptions = append(adoptions, v)
	}

	if len(adopted) > 0 && len(discarded) == 0 && len(held) == 0 {
		findings = append(findings, Finding{Code: "no-discard-or-hold", Subject: "selection"})
	}

	sort.Strings(adopted)
	sort.Strings(discarded)
	sort.Strings(held)
	findings = dedupFindings(findings)

	summary := Summary{Iterations: len(iterations), Adopted: len(adopted), Discarded: len(discarded), Held: len(held)}
	return Report{
		Schema:      Schema,
		Profile:     in.Profile,
		Applicable:  applicable,
		Iterations:  views,
		Adoptions:   adoptions,
		Adopted:     adopted,
		Discarded:   discarded,
		Held:        held,
		Summary:     summary,
		Findings:    findings,
		Fingerprint: fingerprint(views, adoptions),
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

func fingerprint(views []IterationView, adoptions []AdoptionView) string {
	payload := struct {
		Schema     string          `json:"schema"`
		Iterations []IterationView `json:"iterations"`
		Adoptions  []AdoptionView  `json:"adoptions"`
	}{Schema, views, adoptions}
	raw, err := json.Marshal(payload)
	if err != nil {
		raw = []byte(fmt.Sprintf("%v", payload))
	}
	return fmt.Sprintf("sha256:%x", sha256.Sum256(raw))
}

// Effort is the measured effort of one workflow under the same finish condition.
type Effort struct {
	Revisions      int  `json:"revisions"`
	ManualInputs   int  `json:"manualInputs"`
	ReusableAssets int  `json:"reusableAssets"`
	DefectsFound   int  `json:"defectsFound"`
	Measured       bool `json:"measured"`
}

// EffortComparison compares a rigorous workflow and an exploratory one. When either side
// is not measured, the comparison is recorded as unmeasured rather than guessed.
type EffortComparison struct {
	Rigorous    Effort `json:"rigorous"`
	Exploratory Effort `json:"exploratory"`
	Measured    bool   `json:"measured"`
	Note        string `json:"note,omitempty"`
}

// CompareEffort compares the two workflows under the same finish condition.
func CompareEffort(rigorous, exploratory Effort) EffortComparison {
	measured := rigorous.Measured && exploratory.Measured
	c := EffortComparison{Rigorous: rigorous, Exploratory: exploratory, Measured: measured}
	if !measured {
		c.Note = "unmeasured: 少なくとも一方が未実測のため、数を比較に使わない"
	}
	return c
}
