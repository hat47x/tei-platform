package main

// This package is a fixture-only, read-only contract probe. It does not read
// canonical TEI sources, attest Acceptance, authorize execution, or cause effects.

type RuleFixture struct {
	Meaning        string `json:"meaning"`
	MeaningRef     string `json:"meaningRef"`
	SourceRevision string `json:"sourceRevision"`
	Basis          string `json:"basis"`
	Currency       string `json:"currency"`
	ThresholdMinor *int64 `json:"thresholdMinor"`
}

type OrderFixture struct {
	AmountMinor      *int64 `json:"amountMinor,omitempty"`
	Basis            string `json:"basis,omitempty"`
	Currency         string `json:"currency,omitempty"`
	Revision         string `json:"revision,omitempty"`
	ExpectedRevision string `json:"expectedRevision,omitempty"`
}

type Case struct {
	ID      string       `json:"id"`
	Subject string       `json:"subject"`
	Purpose string       `json:"purpose"`
	Rule    RuleFixture  `json:"rule"`
	Order   OrderFixture `json:"order"`
}

type Result struct {
	ID               string `json:"id"`
	Subject          string `json:"subject"`
	Purpose          string `json:"purpose"`
	Status           string `json:"status"`
	Reason           string `json:"reason"`
	Guard            string `json:"guard,omitempty"`
	ExecutionAllowed bool   `json:"executionAllowed"`
}

// Evaluate never returns execution authority. The 'accepted' tag is a fixture
// statement, not validated Acceptance evidence. Its only function is to prove
// that missing rule meaning differs from missing runtime input.
func Evaluate(c Case) Result {
	r := Result{ID: c.ID, Subject: c.Subject, Purpose: c.Purpose, ExecutionAllowed: false}
	if c.ID == "" || c.Subject == "" {
		r.Status, r.Reason = "blocked", "missing-subject-or-case-id"
		return r
	}
	switch c.Purpose {
	case "describe", "simulate", "execute-preflight":
	default:
		r.Status, r.Reason = "blocked", "unsupported-purpose"
		return r
	}
	if c.Rule.Meaning != "accepted" || c.Rule.MeaningRef == "" || c.Rule.SourceRevision == "" {
		r.Status, r.Reason = "semantic-unresolved", "accepted-rule-reference-not-established"
		return r
	}
	if c.Rule.ThresholdMinor == nil || *c.Rule.ThresholdMinor < 0 || c.Rule.Basis != "tax-included-order-total" || c.Rule.Currency != "JPY" {
		r.Status, r.Reason = "semantic-unresolved", "rule-comparison-basis-not-established"
		return r
	}
	if c.Order.AmountMinor == nil {
		// Already-known contradictions cannot become valid by supplying a
		// missing amount later. Deferral is only for unresolved runtime values.
		switch {
		case c.Order.Currency != "" && c.Order.Currency != c.Rule.Currency:
			r.Status, r.Reason = "indeterminate", "runtime-comparison-input-incompatible"
		case c.Order.Basis != "" && c.Order.Basis != c.Rule.Basis:
			r.Status, r.Reason = "indeterminate", "runtime-comparison-input-incompatible"
		case c.Order.Revision != "" && c.Order.ExpectedRevision != "" && c.Order.Revision != c.Order.ExpectedRevision:
			r.Status, r.Reason = "blocked", "stale-order-revision"
		default:
			r.Status, r.Reason = "deferred-to-runtime", "order-amount-not-yet-known"
		}
		return r
	}
	if c.Order.Currency != c.Rule.Currency || c.Order.Basis != c.Rule.Basis || *c.Order.AmountMinor < 0 {
		r.Status, r.Reason = "indeterminate", "runtime-comparison-input-incompatible"
		return r
	}
	if c.Order.Revision == "" || c.Order.ExpectedRevision == "" {
		r.Status, r.Reason = "indeterminate", "order-revision-not-known"
		return r
	}
	if c.Order.Revision != c.Order.ExpectedRevision {
		r.Status, r.Reason = "blocked", "stale-order-revision"
		return r
	}
	r.Status, r.Reason = "evaluated", "synthetic-business-guard-only"
	if *c.Order.AmountMinor >= *c.Rule.ThresholdMinor {
		r.Guard = "true"
	} else {
		r.Guard = "false"
	}
	return r
}
