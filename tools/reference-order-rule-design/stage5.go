package orderruledesign

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"

	procedureexpand "tei/tools/reference-procedure-expansion"
)

// TrialState is the only state a version made by the sample's trial
// acceptance may carry. It is not the state of the preceding sample's
// acceptance, and it is never "accepted".
const TrialState = "trial-acceptance-on-disposable-copy"

// The three execution targets of the unit price rule. They are synthetic:
// the sample declares what each was derived at; it does not generate them.
var executionTargets = []struct{ Target, Ref string }{
	{"screen", "ui.order-line-form"},
	{"server", "server.order-line-validate"},
	{"batch", "batch.order-import-validate"},
}

// Derivation says at which rule version one execution target was derived.
type Derivation struct {
	Target      string `json:"target"`
	Ref         string `json:"ref"`
	RuleVersion string `json:"ruleVersion"`
}

// TargetEntry is one execution target in the impact range.
type TargetEntry struct {
	Target    string `json:"target"`
	Ref       string `json:"ref"`
	DerivedAt string `json:"derivedAt"`
	Derived   bool   `json:"derived"`
	Recheck   bool   `json:"recheck"`
	Reason    string `json:"reason"`
}

// InFlight is a case that started at a baseline version and is not done.
type InFlight struct {
	CaseID         string `json:"caseId"`
	StartedVersion string `json:"startedVersion"`
	Status         string `json:"status"`
	Disposition    string `json:"disposition"` // finish-on-r0 | migrate | "" (undecided)
}

// ReuseCandidate is past evidence the assurance tool offers for reuse. It is
// a candidate: neither a pass nor a permission to run.
type ReuseCandidate struct {
	Evidence         string `json:"evidence"`
	Action           string `json:"action"`
	Pass             bool   `json:"pass"`
	ExecutionAllowed bool   `json:"executionAllowed"`
}

// ImpactRange is what stage 5 hands to stage 6.
type ImpactRange struct {
	Baseline        string           `json:"baselineRevision"`
	CandidateFP     string           `json:"candidateFingerprint"`
	TrialRevision   string           `json:"trialRevision"`
	TrialState      string           `json:"trialState"`
	ProjectAccepted bool             `json:"projectAccepted"`
	Action          string           `json:"action"`
	ExecAllowed     bool             `json:"executionAllowed"`
	ChangedSources  []string         `json:"changedSources"`
	ImpactChanges   []string         `json:"impactChanges"`
	Targets         []TargetEntry    `json:"executionTargets"`
	InFlight        []InFlight       `json:"inFlightCases"`
	Reuse           []ReuseCandidate `json:"reuseCandidates"`
}

type assuranceOut struct {
	Action        string
	Exec          bool
	Reasons       []string
	SourceChanges []string
	Impact        []string
	Digest        string
	HasTargetKeys bool
}

type stage5Out struct {
	R1      string
	Range   ImpactRange
	Review  string
	Carried Carried
}

func (s *Sample) candidatePath() string {
	if s.s1.Evidence != nil {
		return filepath.Join(s.s3.Dir, "candidate-with-evidence.json")
	}
	return filepath.Join(s.s3.Dir, "candidate.json")
}

func (s *Sample) runAssurance(ctx context.Context, before, after string) (assuranceOut, error) {
	raw, err := s.Tools.mustRun(ctx, s.Tools.ChangeAssurance, s.Repo, "-before-workspace", before, "-after-workspace", after,
		"-scope", ReviewScope, "-consumer-kind", "application", "-consumer-id", RuleApplication)
	if err != nil {
		return assuranceOut{}, err
	}
	var v struct {
		SourceChanges []struct {
			Path string `json:"path"`
			Kind string `json:"kind"`
		} `json:"sourceChanges"`
		Impact struct {
			Changes []struct {
				Key struct {
					Subject struct{ Kind, ID string } `json:"subject"`
				} `json:"key"`
				Kind string `json:"kind"`
			} `json:"changes"`
		} `json:"impact"`
		Decision struct {
			Action      string   `json:"action"`
			Reasons     []string `json:"reasonCodes"`
			ExecAllowed bool     `json:"executionAllowed"`
		} `json:"decision"`
	}
	if err := json.Unmarshal(raw, &v); err != nil {
		return assuranceOut{}, err
	}
	o := assuranceOut{Action: v.Decision.Action, Exec: v.Decision.ExecAllowed, Reasons: v.Decision.Reasons}
	for _, c := range v.SourceChanges {
		o.SourceChanges = append(o.SourceChanges, c.Kind+" "+c.Path)
	}
	for _, c := range v.Impact.Changes {
		o.Impact = append(o.Impact, c.Kind+" "+c.Key.Subject.Kind+":"+c.Key.Subject.ID)
	}
	sort.Strings(o.SourceChanges)
	sort.Strings(o.Impact)
	o.Digest = mustDigest(map[string]any{"sourceChanges": o.SourceChanges, "impact": o.Impact, "action": o.Action,
		"reasons": o.Reasons, "executionAllowed": o.Exec})
	// The assurance output has no key for execution targets: the gap the
	// sample records rather than hides.
	var generic map[string]any
	_ = json.Unmarshal(raw, &generic)
	for k := range generic {
		if k == "executionTargets" || k == "derivations" || k == "targets" {
			o.HasTargetKeys = true
		}
	}
	return o, nil
}

// reuseFixture runs the assurance tool's own fixture, whose past evidence is
// offered as reuse candidates.
func (s *Sample) reuseFixture(ctx context.Context) ([]ReuseCandidate, error) {
	raw, err := s.Tools.mustRun(ctx, s.Tools.ChangeAssurance, s.Repo, "-input", "tools/reference-change-assurance/example.json")
	if err != nil {
		return nil, err
	}
	var v struct {
		Decisions []struct {
			CaseID  string `json:"caseId"`
			Action  string `json:"action"`
			Allowed bool   `json:"executionAllowed"`
		} `json:"decisions"`
		Results []struct {
			CaseID  string `json:"caseId"`
			Action  string `json:"action"`
			Allowed bool   `json:"executionAllowed"`
		} `json:"results"`
	}
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil, err
	}
	var out []ReuseCandidate
	for _, d := range v.Decisions {
		out = append(out, ReuseCandidate{d.CaseID, d.Action, false, d.Allowed})
	}
	for _, d := range v.Results {
		if len(v.Decisions) > 0 {
			break
		}
		out = append(out, ReuseCandidate{d.CaseID, d.Action, false, d.Allowed})
	}
	if len(out) == 0 {
		var generic any
		_ = json.Unmarshal(raw, &generic)
		return nil, fmt.Errorf("no decisions in the assurance fixture output: %.300s", raw)
	}
	return out, nil
}

// expansionStep calls the procedure expansion library: which realization
// readiness flips when the unit price condition is decided.
func expansionStep() (map[string]any, []string) {
	step := func(cond string) procedureexpand.Input {
		return procedureexpand.Input{Procedure: "order-line-entry",
			Steps: []procedureexpand.Step{{ID: "S1", Action: "check the unit price is positive", Nature: "machine", Authority: "order-clerk", Conditions: []string{cond}}},
			Bindings: []procedureexpand.Binding{
				{Step: "S1", Target: "screen", Ref: executionTargets[0].Ref, Authority: "order-clerk"},
				{Step: "S1", Target: "function", Ref: executionTargets[1].Ref, Authority: "order-clerk"},
				{Step: "S1", Target: "function", Ref: executionTargets[2].Ref, Authority: "order-clerk"}}}
	}
	cmp := procedureexpand.Compare(step("<unresolved>"), step("unit price > 0"))
	return map[string]any{"changedSteps": cmp.ChangedSteps, "affectedAssets": cmp.AffectedAssets,
		"note": "a screen is always available, so only the function bindings flip; the tool lists bound refs, not derived versions"}, cmp.AffectedAssets
}

// BuildImpactRange assembles the range from the assurance output, the
// derivation declarations and the in-flight cases.
func BuildImpactRange(baseline, candidateFP, trial string, a assuranceOut, derivs []Derivation, inflight []InFlight, reuse []ReuseCandidate) ImpactRange {
	r := ImpactRange{Baseline: baseline, CandidateFP: candidateFP, TrialRevision: trial, TrialState: TrialState,
		Action: a.Action, ExecAllowed: a.Exec, ChangedSources: a.SourceChanges, ImpactChanges: a.Impact, InFlight: inflight, Reuse: reuse}
	by := map[string]Derivation{}
	for _, d := range derivs {
		by[d.Ref] = d
	}
	for _, t := range executionTargets {
		e := TargetEntry{Target: t.Target, Ref: t.Ref, DerivedAt: by[t.Ref].RuleVersion}
		e.Derived = e.DerivedAt == candidateFP
		e.Recheck = true
		if e.Derived {
			e.Reason = "derived at the candidate version; the assurance action is " + a.Action + ", so it is rechecked"
		} else {
			e.Reason = "not-derived: still at " + e.DerivedAt + "; stays in the recheck range"
		}
		r.Targets = append(r.Targets, e)
	}
	return r
}

// VerifyImpactRange returns the ways a range overclaims or omits.
func VerifyImpactRange(r ImpactRange, caseStarts []InFlight) []string {
	var out []string
	add := func(f string, a ...any) { out = append(out, fmt.Sprintf(f, a...)) }
	if r.ExecAllowed {
		add("the range allows execution")
	}
	if r.ProjectAccepted || r.TrialState != TrialState {
		add("the trial version is not kept apart from an accepted version (state %q, accepted %v)", r.TrialState, r.ProjectAccepted)
	}
	for _, c := range r.Reuse {
		if c.Pass || c.ExecutionAllowed {
			add("reuse candidate %s is read as a pass or a permission", c.Evidence)
		}
	}
	if len(r.Targets) != len(executionTargets) {
		add("the range lists %d execution targets, want %d", len(r.Targets), len(executionTargets))
	}
	for _, t := range r.Targets {
		if !t.Derived && !t.Recheck {
			add("target %s is not derived and is not in the recheck range", t.Target)
		}
	}
	listed := map[string]bool{}
	for _, c := range r.InFlight {
		listed[c.CaseID] = true
		if c.StartedVersion == "" {
			add("case %s has no start version", c.CaseID)
		}
	}
	for _, c := range caseStarts {
		if !listed[c.CaseID] {
			add("in-flight case %s is missing from the range", c.CaseID)
		}
	}
	return out
}

// inFlightCases are synthetic: no route returns running cases from a
// runtime to the design side, so the sample is handed this list.
func inFlightCases(baseline string, disp map[string]string) []InFlight {
	var out []InFlight
	for _, id := range []string{"SO-1001", "SO-1002", "SO-1003"} {
		out = append(out, InFlight{CaseID: id, StartedVersion: baseline, Status: "in-progress", Disposition: disp[id]})
	}
	return out
}

var mainDispositions = map[string]string{"SO-1001": "finish-on-r0", "SO-1002": "finish-on-r0", "SO-1003": "migrate"}

func (s *Sample) trialAcceptance(ctx context.Context, dir string) (string, string, error) {
	r1 := filepath.Join(dir, "r1")
	if err := copyTree(s.s2.R0, r1); err != nil {
		return "", "", err
	}
	rv, err := s.Tools.mustRun(ctx, s.Tools.SharedAcceptance, s.Repo, "review", "-workspace", r1, "-candidate", s.candidatePath(), "-format", "json")
	if err != nil {
		return "", "", err
	}
	var rev struct {
		ReviewFingerprint string `json:"reviewFingerprint"`
	}
	if err := json.Unmarshal(rv, &rev); err != nil || rev.ReviewFingerprint == "" {
		return "", "", fmt.Errorf("review output: %v", err)
	}
	if _, err := s.Tools.mustRun(ctx, s.Tools.SharedAcceptance, s.Repo, "accept", "-workspace", r1, "-candidate", s.candidatePath(),
		"-expected-review-fingerprint", rev.ReviewFingerprint); err != nil {
		return "", "", err
	}
	return r1, rev.ReviewFingerprint, nil
}

func (s *Sample) stage5(ctx context.Context) error {
	dir := filepath.Join(s.Work, "stage5")
	// The trial acceptance runs on a disposable copy only; the project's
	// acceptance (the preceding sample's) is a different state.
	r1, reviewFP, err := s.trialAcceptance(ctx, dir)
	if err != nil {
		return err
	}
	trial, err := s.revisionOf(ctx, r1)
	if err != nil {
		return err
	}
	if want, _ := s.s3.Preview["modelRevisionAfter"].(string); want != trial {
		return fmt.Errorf("trial revision %s differs from the preview of stage 3 %s", trial, want)
	}
	a, err := s.runAssurance(ctx, s.s2.R0, r1)
	if err != nil {
		return err
	}
	if a.Action != "recheck" || a.Exec {
		return fmt.Errorf("assurance on the candidate: action %q executionAllowed %v", a.Action, a.Exec)
	}
	reuse, err := s.reuseFixture(ctx)
	if err != nil {
		return err
	}
	exp, _ := expansionStep()
	var derivs []Derivation
	for _, t := range executionTargets {
		derivs = append(derivs, Derivation{t.Target, t.Ref, s.s3.Fingerprint})
	}
	rng := BuildImpactRange(s.s2.Revision, s.s3.Fingerprint, trial, a, derivs, inFlightCases(s.s2.Revision, mainDispositions), reuse)
	if p := VerifyImpactRange(rng, inFlightCases(s.s2.Revision, nil)); len(p) > 0 {
		return fmt.Errorf("impact range: %v", p)
	}
	s.assurance = a
	c := s.s4.Carried
	c.AssuranceDigest, c.TrialRevision, c.ImpactRangeDigest = a.Digest, trial, mustDigest(rng)
	s.s5 = stage5Out{R1: r1, Range: rng, Review: reviewFP, Carried: c}
	return s.Chain.Append(5, c, map[string]any{
		"trialState": TrialState, "projectAccepted": false, "trialRevision": trial,
		"assurance": map[string]any{"action": a.Action, "executionAllowed": a.Exec, "reasons": a.Reasons,
			"sourceChanges": a.SourceChanges, "impact": a.Impact, "emitsExecutionTargets": a.HasTargetKeys},
		"impactRange": rng, "procedureExpansion": exp, "reuseFixtureCandidates": reuse,
		"derivationNote": "derivations are declarations of the sample, not generated artifacts",
		"inFlightNote":   "no route returns running cases from a runtime; the list is a sample input",
	})
}
