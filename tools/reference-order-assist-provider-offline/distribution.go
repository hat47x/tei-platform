package orderassistoffline

import (
	"context"
	"errors"
	"fmt"
)

// Site is an execution site of the quote rule (P-MIX-01).
type Site string

const (
	SiteTerminal Site = "terminal" // sales terminal, JavaScript
	SiteServer   Site = "server"   // order server, native executable
	SiteBatch    Site = "batch"    // nightly review, WASI module
)

// Deployment is what one site reports it is running.
type Deployment struct {
	Site           Site   `json:"site"`
	Realization    string `json:"realization"`
	Generation     int    `json:"generation"`
	ArtifactDigest string `json:"artifactDigest"`
}

// Distribution is the rollout state of one rule revision to the sites.
type Distribution struct {
	TargetGeneration int          `json:"targetGeneration"`
	Sites            []Deployment `json:"sites"`
}

// Exclusion records why a site's realization was not used for a decision.
type Exclusion struct {
	Site               Site   `json:"site"`
	Reason             string `json:"reason"`
	ReportedGeneration int    `json:"reportedGeneration"`
	ReportedDigest     string `json:"reportedDigest"`
}

// SiteOutcome is one used site's answer.
type SiteOutcome struct {
	Site    Site    `json:"site"`
	Outcome Outcome `json:"outcome"`
}

// Judgment is a quote decision made only from realizations on the target
// rule version, with the rule version and the excluded sites recorded.
type Judgment struct {
	RuleVersion  string        `json:"ruleVersion"`
	Generation   int           `json:"generation"`
	Used         []SiteOutcome `json:"used"`
	Excluded     []Exclusion   `json:"excluded"`
	Agreed       bool          `json:"agreed"`
	Rejected     bool          `json:"rejected"`
	Result       QuoteResult   `json:"result"`
	ResultDigest string        `json:"resultDigest,omitempty"`
}

// ErrDivergence means the used realizations returned different answers; no
// decision is made.
var ErrDivergence = errors.New("realizations on the same rule version disagree")

// ErrNoEligibleSite means no site runs the target rule version.
var ErrNoEligibleSite = errors.New("no site runs the target rule version")

// JudgeOptions holds the negative control.
type JudgeOptions struct {
	// UnsafeIncludeStale uses every site with its own generation, as a naive
	// rollout would. Only for the negative control.
	UnsafeIncludeStale bool
}

// Judge evaluates one quote request at every eligible site and returns the
// agreed decision. A site is eligible when it reports the target generation
// and its reported artifact digest is the registered digest of that
// realization; an ineligible site is never invoked.
func (s *RealizationSet) Judge(ctx context.Context, dist Distribution, requestID string, payload []byte, opts JudgeOptions) (Judgment, error) {
	target, ok := Generations[dist.TargetGeneration]
	if !ok {
		return Judgment{}, fmt.Errorf("unknown target generation %d", dist.TargetGeneration)
	}
	j := Judgment{RuleVersion: target.RuleVersion, Generation: target.Number}
	type pick struct {
		site Site
		r    Realization
		gen  Generation
	}
	var picks []pick
	for _, d := range dist.Sites {
		r, known := s.ByName[d.Realization]
		gen, genKnown := Generations[d.Generation]
		switch {
		case !known || !genKnown || r.Generation != d.Generation:
			j.Excluded = append(j.Excluded, Exclusion{d.Site, "unknown-realization", d.Generation, d.ArtifactDigest})
			continue
		case r.ArtifactDigest != d.ArtifactDigest:
			j.Excluded = append(j.Excluded, Exclusion{d.Site, "artifact-digest-mismatch", d.Generation, d.ArtifactDigest})
			continue
		case d.Generation != target.Number && !opts.UnsafeIncludeStale:
			j.Excluded = append(j.Excluded, Exclusion{d.Site, "rule-version-mismatch", d.Generation, d.ArtifactDigest})
			continue
		}
		picks = append(picks, pick{d.Site, r, gen})
	}
	if len(picks) == 0 {
		return j, ErrNoEligibleSite
	}
	for i, p := range picks {
		out, err := p.r.Quote(ctx, p.gen, fmt.Sprintf("%s-%s-%d", requestID, p.site, i), payload)
		if err != nil {
			return j, err
		}
		j.Used = append(j.Used, SiteOutcome{Site: p.site, Outcome: out})
	}
	first := j.Used[0].Outcome
	for _, u := range j.Used[1:] {
		if u.Outcome.Rejected != first.Rejected || string(u.Outcome.Raw) != string(first.Raw) {
			return j, fmt.Errorf("%w: %s and %s", ErrDivergence, j.Used[0].Site, u.Site)
		}
	}
	j.Agreed = true
	j.Rejected = first.Rejected
	if !first.Rejected {
		j.Result = first.Result
		j.ResultDigest = digestBytes(first.Raw)
	}
	return j, nil
}
