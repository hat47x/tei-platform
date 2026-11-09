// Package accountingperiod is a read-only reference for business dates and
// accounting periods (#2144). It separates the business date from the system
// clock, keeps period states (open / closing / closed), rejects direct updates
// to closed periods, links a red-black correction to the original slip, and
// declares what happens to a slip that arrives after its period closed.
//
// It is not a ledger, a database trigger, a lifecycle runner, or an accepted
// API. It writes nothing: a close or correction is a decision over synthetic
// input, not a persisted transaction.
package accountingperiod

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/big"
	"sort"
	"strings"
)

// Period states.
const (
	StateOpen    = "open"
	StateClosing = "closing"
	StateClosed  = "closed"
)

// Late-arrival policies.
const (
	LateAdjustCurrent = "adjust-current"
	LateHold          = "hold"
)

// Result codes.
const (
	CodePosted                = "POSTED"
	CodeClosedPeriodRejected  = "CLOSED_PERIOD_REJECTED"
	CodeCorrectionPosted      = "CORRECTION_POSTED"
	CodeCorrectionMissingOrig = "CORRECTION_MISSING_ORIGINAL"
	CodeLateAdjusted          = "LATE_ARRIVAL_ADJUSTED"
	CodeLateHeld              = "LATE_ARRIVAL_HELD"
	CodeReopenNotPermitted    = "REOPEN_NOT_PERMITTED"
	CodeCloseAccepted         = "CLOSE_ACCEPTED"
	CodeCloseBlockedPending   = "CLOSE_BLOCKED_PENDING"
	CodeUnknownPeriod         = "UNKNOWN_PERIOD"
	CodeInvalidRequest        = "INVALID_REQUEST"
)

// Period is one accounting period. Start and End are business dates
// (YYYY-MM-DD, inclusive). State is open, closing, or closed.
type Period struct {
	ID    string `json:"id"`
	State string `json:"state"`
	Start string `json:"start"`
	End   string `json:"end"`
}

// Slip is one posted business slip.
type Slip struct {
	ID           string `json:"id"`
	Period       string `json:"period"`
	BusinessDate string `json:"businessDate"`
	Amount       string `json:"amount"`
	Status       string `json:"status,omitempty"` // pending or posted
	// Reverses and Corrects link a red-black correction to the original slip.
	Reverses string `json:"reverses,omitempty"`
	Corrects string `json:"corrects,omitempty"`
}

// Action is one operation to decide.
type Action struct {
	Kind string `json:"kind"` // post, update, correct, close, reopen
	// SlipID is the slip being posted, updated, or corrected.
	SlipID string `json:"slipId,omitempty"`
	// Period is used by close and reopen.
	Period string `json:"period,omitempty"`
	// BusinessDate is the accounting date. It is never the system clock.
	BusinessDate string `json:"businessDate,omitempty"`
	Amount       string `json:"amount,omitempty"`
	// Reverses names the original slip a correction reverses.
	Reverses string `json:"reverses,omitempty"`
}

// Input is the whole synthetic request.
type Input struct {
	// SystemTime is the wall clock when the action was received. It is recorded
	// but never used to decide the period.
	SystemTime string   `json:"systemTime,omitempty"`
	Periods    []Period `json:"periods"`
	Slips      []Slip   `json:"slips,omitempty"`
	Action     *Action  `json:"action,omitempty"`
	LatePolicy string   `json:"latePolicy,omitempty"`
}

// Report is the read-only decision.
type Report struct {
	Code         string   `json:"code"`
	Accepted     bool     `json:"accepted"`
	Action       string   `json:"action,omitempty"`
	BusinessDate string   `json:"businessDate,omitempty"`
	SystemTime   string   `json:"systemTime,omitempty"`
	PeriodState  string   `json:"periodState,omitempty"`
	PostedPeriod string   `json:"postedPeriod,omitempty"`
	CorrectionOf string   `json:"correctionOf,omitempty"`
	LinkedSlips  []string `json:"linkedSlips,omitempty"`
	Reason       string   `json:"reason,omitempty"`
}

func reject(in Input, action, code, reason string) Report {
	return Report{Code: code, Action: action, BusinessDate: businessDateOf(in), SystemTime: in.SystemTime, Reason: reason}
}

func businessDateOf(in Input) string {
	if in.Action == nil {
		return ""
	}
	return in.Action.BusinessDate
}

func findPeriodByDate(periods []Period, businessDate string) (Period, bool) {
	for _, period := range periods {
		if businessDate >= period.Start && businessDate <= period.End {
			return period, true
		}
	}
	return Period{}, false
}

func findPeriod(periods []Period, id string) (Period, bool) {
	for _, period := range periods {
		if period.ID == id {
			return period, true
		}
	}
	return Period{}, false
}

func currentOpenPeriod(periods []Period) (Period, bool) {
	var open []Period
	for _, period := range periods {
		if period.State == StateOpen {
			open = append(open, period)
		}
	}
	if len(open) == 0 {
		return Period{}, false
	}
	sort.Slice(open, func(i, j int) bool { return open[i].Start < open[j].Start })
	return open[0], true
}

func findSlip(slips []Slip, id string) (Slip, bool) {
	for _, slip := range slips {
		if slip.ID == id {
			return slip, true
		}
	}
	return Slip{}, false
}

// Evaluate decides one action. A direct update of a slip in a closed period is
// rejected; the caller must use a red-black correction posted to an open period.
func Evaluate(in Input) Report {
	if in.Action == nil {
		return reject(in, "", CodeInvalidRequest, "action is required")
	}
	action := in.Action
	switch action.Kind {
	case "post", "update", "correct", "close", "reopen":
	default:
		return reject(in, action.Kind, CodeInvalidRequest, fmt.Sprintf("unknown action %q", action.Kind))
	}
	if action.Kind == "close" || action.Kind == "reopen" {
		return evaluateLifecycle(in, action)
	}
	if strings.TrimSpace(action.BusinessDate) == "" {
		return reject(in, action.Kind, CodeInvalidRequest, "businessDate is required")
	}

	switch action.Kind {
	case "post":
		return evaluatePost(in, action)
	case "update":
		return evaluateUpdate(in, action)
	case "correct":
		return evaluateCorrection(in, action)
	}
	return reject(in, action.Kind, CodeInvalidRequest, "unreachable")
}

func evaluatePost(in Input, action *Action) Report {
	period, ok := findPeriodByDate(in.Periods, action.BusinessDate)
	if !ok {
		return reject(in, action.Kind, CodeUnknownPeriod, fmt.Sprintf("no period contains business date %s", action.BusinessDate))
	}
	if period.State != StateClosed {
		return Report{Code: CodePosted, Accepted: true, Action: action.Kind,
			BusinessDate: action.BusinessDate, SystemTime: in.SystemTime,
			PeriodState: period.State, PostedPeriod: period.ID,
			Reason: "business date falls in an open period; the system clock is not used"}
	}
	// The period is closed. The declared policy decides whether to adjust the
	// current open period or hold the slip. It never silently changes the
	// closed period.
	switch policyOf(in, period) {
	case LateAdjustCurrent:
		open, ok := currentOpenPeriod(in.Periods)
		if !ok {
			return reject(in, action.Kind, CodeLateHeld, "no open period exists to receive the late arrival")
		}
		return Report{Code: CodeLateAdjusted, Accepted: true, Action: action.Kind,
			BusinessDate: action.BusinessDate, SystemTime: in.SystemTime,
			PeriodState: period.State, PostedPeriod: open.ID,
			Reason: fmt.Sprintf("period %s is closed; the late slip is adjusted in open period %s without changing the closed period", period.ID, open.ID)}
	default:
		return Report{Code: CodeLateHeld, Accepted: false, Action: action.Kind,
			BusinessDate: action.BusinessDate, SystemTime: in.SystemTime,
			PeriodState: period.State,
			Reason:      fmt.Sprintf("period %s is closed; the late slip is held for review", period.ID)}
	}
}

func evaluateUpdate(in Input, action *Action) Report {
	period, ok := findPeriodByDate(in.Periods, action.BusinessDate)
	if !ok {
		return reject(in, action.Kind, CodeUnknownPeriod, fmt.Sprintf("no period contains business date %s", action.BusinessDate))
	}
	if period.State == StateClosed {
		return Report{Code: CodeClosedPeriodRejected, Accepted: false, Action: action.Kind,
			BusinessDate: action.BusinessDate, SystemTime: in.SystemTime, PeriodState: period.State,
			Reason: "a slip in a closed period cannot be updated directly; use a red-black correction posted to an open period"}
	}
	return Report{Code: CodePosted, Accepted: true, Action: action.Kind,
		BusinessDate: action.BusinessDate, SystemTime: in.SystemTime,
		PeriodState: period.State, PostedPeriod: period.ID}
}

func evaluateCorrection(in Input, action *Action) Report {
	if strings.TrimSpace(action.Reverses) == "" {
		return reject(in, action.Kind, CodeCorrectionMissingOrig, "a correction must name the original slip it reverses")
	}
	original, ok := findSlip(in.Slips, action.Reverses)
	if !ok {
		return reject(in, action.Kind, CodeCorrectionMissingOrig, fmt.Sprintf("original slip %q is not present", action.Reverses))
	}
	originalPeriod, ok := findPeriod(in.Periods, original.Period)
	if !ok {
		return reject(in, action.Kind, CodeUnknownPeriod, fmt.Sprintf("original slip period %q is not declared", original.Period))
	}
	if originalPeriod.State != StateClosed {
		return reject(in, action.Kind, CodeInvalidRequest,
			fmt.Sprintf("correction is only for a closed period; %s is %s", originalPeriod.ID, originalPeriod.State))
	}
	targetPeriod, ok := findPeriodByDate(in.Periods, action.BusinessDate)
	if !ok {
		return reject(in, action.Kind, CodeUnknownPeriod, fmt.Sprintf("no period contains business date %s", action.BusinessDate))
	}
	if targetPeriod.State == StateClosed {
		return Report{Code: CodeClosedPeriodRejected, Accepted: false, Action: action.Kind,
			BusinessDate: action.BusinessDate, SystemTime: in.SystemTime, PeriodState: targetPeriod.State,
			Reason: "a correction cannot be posted into a closed period"}
	}
	slipID := strings.TrimSpace(action.SlipID)
	if slipID == "" {
		slipID = original.ID + "-corrected"
	}
	return Report{Code: CodeCorrectionPosted, Accepted: true, Action: action.Kind,
		BusinessDate: action.BusinessDate, SystemTime: in.SystemTime,
		PeriodState: targetPeriod.State, PostedPeriod: targetPeriod.ID,
		CorrectionOf: original.ID,
		LinkedSlips:  []string{original.ID, slipID},
		Reason:       "the reversal and the correction form a red-black pair; the original closed slip is not rewritten"}
}

func evaluateLifecycle(in Input, action *Action) Report {
	period, ok := findPeriod(in.Periods, action.Period)
	if !ok {
		return reject(in, action.Kind, CodeUnknownPeriod, fmt.Sprintf("period %q is not declared", action.Period))
	}
	if action.Kind == "reopen" {
		// Reopening a closed period needs an authority and audit decision that
		// this reference does not own.
		return Report{Code: CodeReopenNotPermitted, Accepted: false, Action: action.Kind,
			SystemTime: in.SystemTime, PeriodState: period.State,
			Reason: "reopening a closed period is not permitted by this reference; it needs an authority and audit decision"}
	}
	if period.State == StateClosed {
		return Report{Code: CodeCloseAccepted, Accepted: true, Action: action.Kind,
			SystemTime: in.SystemTime, PeriodState: period.State,
			Reason: "the period is already closed; the close is idempotent"}
	}
	for _, slip := range in.Slips {
		if slip.Period == period.ID && slip.Status == "pending" {
			return Report{Code: CodeCloseBlockedPending, Accepted: false, Action: action.Kind,
				SystemTime: in.SystemTime, PeriodState: period.State,
				Reason: fmt.Sprintf("slip %q in period %s is still pending", slip.ID, period.ID)}
		}
	}
	return Report{Code: CodeCloseAccepted, Accepted: true, Action: action.Kind,
		SystemTime: in.SystemTime, PeriodState: StateClosed,
		Reason: "all slips in the period are posted; the close is accepted as a decision, not a stored write"}
}

func policyOf(in Input, closed Period) string {
	_ = closed
	if strings.TrimSpace(in.LatePolicy) == "" {
		return LateHold
	}
	return in.LatePolicy
}

// Event is one received business event for ordered replay.
type Event struct {
	ID           string `json:"id"`
	BusinessDate string `json:"businessDate"`
	Amount       string `json:"amount"`
	OrderKey     int    `json:"orderKey"`
	Version      int    `json:"version"`
}

// ReplayInput replays events whose arrival order may differ from their business
// order. The declared OrderKey orders them, and the highest Version wins for
// the same event ID.
type ReplayInput struct {
	Periods    []Period `json:"periods"`
	Events     []Event  `json:"events"`
	LatePolicy string   `json:"latePolicy,omitempty"`
}

// ReplayReport is the read-only replay result.
type ReplayReport struct {
	Code    string            `json:"code"`
	Applied []string          `json:"applied"`
	Late    []string          `json:"late,omitempty"`
	Held    []string          `json:"held,omitempty"`
	Stale   []string          `json:"stale,omitempty"`
	Totals  map[string]string `json:"totals"`
	Digest  string            `json:"digest"`
	Note    string            `json:"note,omitempty"`
}

// Replay applies events in OrderKey order. A shuffled arrival order produces
// the same totals and digest. An event whose period is closed is adjusted or
// held by the same policy as Evaluate; it never changes the closed period.
func Replay(in ReplayInput) ReplayReport {
	report := ReplayReport{Code: "REPLAYED", Applied: []string{}, Totals: map[string]string{},
		Note: "business order is declared by orderKey; the arrival order is not used. The highest version wins for the same event id."}
	chosen := map[string]Event{}
	for _, event := range in.Events {
		existing, ok := chosen[event.ID]
		if !ok || event.Version > existing.Version || (event.Version == existing.Version && event.OrderKey < existing.OrderKey) {
			if ok {
				report.Stale = append(report.Stale, existing.ID)
			}
			chosen[event.ID] = event
			continue
		}
		if event.Version < existing.Version || event.OrderKey > existing.OrderKey {
			report.Stale = append(report.Stale, event.ID)
		}
	}
	ordered := make([]Event, 0, len(chosen))
	for _, event := range chosen {
		ordered = append(ordered, event)
	}
	sort.Slice(ordered, func(i, j int) bool {
		if ordered[i].OrderKey == ordered[j].OrderKey {
			return ordered[i].ID < ordered[j].ID
		}
		return ordered[i].OrderKey < ordered[j].OrderKey
	})

	totals := map[string]*big.Rat{}
	for _, event := range ordered {
		period, ok := findPeriodByDate(in.Periods, event.BusinessDate)
		if !ok {
			report.Code = CodeUnknownPeriod
			continue
		}
		if period.State == StateClosed {
			if policyOf(Input{LatePolicy: in.LatePolicy}, period) == LateAdjustCurrent {
				open, ok := currentOpenPeriod(in.Periods)
				if !ok {
					report.Held = append(report.Held, event.ID)
					continue
				}
				addAmount(totals, open.ID, event.Amount)
				report.Late = append(report.Late, event.ID)
				report.Applied = append(report.Applied, event.ID)
				continue
			}
			report.Held = append(report.Held, event.ID)
			continue
		}
		addAmount(totals, period.ID, event.Amount)
		report.Applied = append(report.Applied, event.ID)
	}
	sort.Strings(report.Applied)
	sort.Strings(report.Late)
	sort.Strings(report.Held)
	sort.Strings(report.Stale)
	for period, total := range totals {
		report.Totals[period] = formatRat(total)
	}
	report.Digest = replayDigest(report)
	return report
}

func addAmount(totals map[string]*big.Rat, period, amount string) {
	value, ok := new(big.Rat).SetString(strings.TrimSpace(amount))
	if !ok {
		return
	}
	if totals[period] == nil {
		totals[period] = new(big.Rat)
	}
	totals[period].Add(totals[period], value)
}

func formatRat(value *big.Rat) string {
	if value.IsInt() {
		return value.Num().String()
	}
	return strings.TrimRight(strings.TrimRight(value.FloatString(6), "0"), ".")
}

func replayDigest(report ReplayReport) string {
	periods := make([]string, 0, len(report.Totals))
	for period := range report.Totals {
		periods = append(periods, period)
	}
	sort.Strings(periods)
	hash := sha256.New()
	for _, period := range periods {
		fmt.Fprintf(hash, "%s\x00%s\n", period, report.Totals[period])
	}
	for _, id := range report.Applied {
		fmt.Fprintf(hash, "applied\x00%s\n", id)
	}
	for _, id := range report.Held {
		fmt.Fprintf(hash, "held\x00%s\n", id)
	}
	return "sha256:" + hex.EncodeToString(hash.Sum(nil))
}

// CloseDigest hashes the posted slips of a closed period. Reprinting the close
// report from the same close-time record yields the same digest even after late
// arrivals were handled elsewhere.
func CloseDigest(slips []Slip) string {
	sorted := append([]Slip(nil), slips...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].ID < sorted[j].ID })
	hash := sha256.New()
	for _, slip := range sorted {
		fmt.Fprintf(hash, "%s\x00%s\x00%s\x00%s\x00%s\n", slip.ID, slip.Period, slip.BusinessDate, slip.Amount, slip.Status)
	}
	return "sha256:" + hex.EncodeToString(hash.Sum(nil))
}
