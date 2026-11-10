package main

import (
	"encoding/json"
	"errors"
	"strings"

	"tei/internal/plugin"
)

// This file makes the entry check the resolver's own arithmetic instead of
// copying its plan fingerprint. The receipt carries the full BuildPlan, so the
// entry can recompute plugin.BuildPlanFingerprint over it and compare.
//
// What this adds: the receipt is internally consistent, and the plan the
// resolver reported is the plan the receipt carries. What it does NOT add: a
// proof that the builder child built that plan. The builder is a separate
// process and its own reading is still only what it reports. l2Claimed stays
// false.

const (
	planRecomputeMatched     = "matched"
	planRecomputeMismatched  = "mismatched"
	planRecomputeUnperformed = "unperformed"
)

// planRecompute is the entry's own recomputation of the resolver's plan
// fingerprint. A missing plan in the receipt is unperformed with the reason,
// never a match.
type planRecompute struct {
	State                 string `json:"state"`
	ReportedFingerprint   string `json:"reportedFingerprint,omitempty"`
	RecomputedFingerprint string `json:"recomputedFingerprint,omitempty"`
	Matched               bool   `json:"matched"`
	L2BClaimed            bool   `json:"l2bClaimed"`
	Boundary              string `json:"boundary"`
	UnperformedReason     string `json:"unperformedReason,omitempty"`
	FailureCause          string `json:"failureCause,omitempty"`
}

const planRecomputeBoundary = "the entry recomputed the plan fingerprint from the plan inside the receipt; this checks the resolver's arithmetic and the receipt's internal consistency, not that the builder child built this plan"

// recomputeReceiptPlanFingerprint reads the receipt's plan object and derives
// its fingerprint with the same library function the resolver uses. It never
// trusts the reported value.
func recomputeReceiptPlanFingerprint(receiptJSON string) planRecompute {
	record := planRecompute{
		L2BClaimed: false,
		Boundary:   planRecomputeBoundary,
	}
	var parsed struct {
		PlanFingerprint string          `json:"planFingerprint"`
		Plan            json.RawMessage `json:"plan"`
	}
	if err := json.Unmarshal([]byte(receiptJSON), &parsed); err != nil {
		record.State = planRecomputeUnperformed
		record.UnperformedReason = "the receipt is not one JSON object"
		return record
	}
	record.ReportedFingerprint = parsed.PlanFingerprint
	if strings.TrimSpace(parsed.PlanFingerprint) == "" {
		record.State = planRecomputeUnperformed
		record.UnperformedReason = "the receipt has no planFingerprint to compare"
		return record
	}
	if len(parsed.Plan) == 0 {
		record.State = planRecomputeUnperformed
		record.UnperformedReason = "the receipt carries no plan object, so the fingerprint cannot be recomputed"
		return record
	}
	var plan plugin.BuildPlan
	if err := json.Unmarshal(parsed.Plan, &plan); err != nil {
		record.State = planRecomputeUnperformed
		record.UnperformedReason = "the receipt's plan is not a BuildPlan: " + err.Error()
		return record
	}
	recomputed, err := plugin.BuildPlanFingerprint(plan)
	if err != nil {
		record.State = planRecomputeUnperformed
		record.UnperformedReason = "the receipt's plan does not validate: " + err.Error()
		return record
	}
	record.RecomputedFingerprint = recomputed
	if recomputed == parsed.PlanFingerprint {
		record.State = planRecomputeMatched
		record.Matched = true
		return record
	}
	record.State = planRecomputeMismatched
	record.FailureCause = errors.New("the receipt's plan does not hash to the planFingerprint it reports").Error()
	return record
}
