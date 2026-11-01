package accountingperiod

import (
	"reflect"
	"sort"
	"testing"
)

// The fixture follows #2144: April is closed, May is open. The business date is
// the accounting date; the system time is only recorded.
func periods() []Period {
	return []Period{
		{ID: "2026-04", State: StateClosed, Start: "2026-04-01", End: "2026-04-30"},
		{ID: "2026-05", State: StateOpen, Start: "2026-05-01", End: "2026-05-31"},
	}
}

func slips() []Slip {
	return []Slip{
		{ID: "S-APR-1", Period: "2026-04", BusinessDate: "2026-04-15", Amount: "1000", Status: "posted"},
	}
}

func action(kind, businessDate string) *Action {
	return &Action{Kind: kind, SlipID: "S-NEW", BusinessDate: businessDate, Amount: "500"}
}

func TestBusinessDateIsSeparateFromSystemTime(t *testing.T) {
	in := Input{SystemTime: "2026-05-10T09:00:00Z", Periods: periods(), LatePolicy: LateHold}
	open := in
	open.Action = action("post", "2026-05-10")
	closed := in
	closed.Action = action("post", "2026-04-15")

	openReport := Evaluate(open)
	closedReport := Evaluate(closed)
	if openReport.Code != CodePosted || !openReport.Accepted {
		t.Fatalf("open=%+v", openReport)
	}
	if closedReport.Code != CodeLateHeld {
		t.Fatalf("closed=%+v", closedReport)
	}
	if openReport.SystemTime != closedReport.SystemTime {
		t.Fatalf("system time differed between the two runs")
	}
	if openReport.BusinessDate == closedReport.BusinessDate {
		t.Fatalf("business date was not carried separately")
	}
}

func TestDirectUpdateOfClosedPeriodIsRejected(t *testing.T) {
	in := Input{SystemTime: "2026-05-10", Periods: periods(), Slips: slips()}
	in.Action = &Action{Kind: "update", SlipID: "S-APR-1", BusinessDate: "2026-04-15", Amount: "1200"}
	report := Evaluate(in)
	if report.Code != CodeClosedPeriodRejected || report.Accepted {
		t.Fatalf("report=%+v", report)
	}
	if report.PeriodState != StateClosed {
		t.Fatalf("periodState=%q", report.PeriodState)
	}
}

func TestRedBlackCorrectionLinksTheOriginal(t *testing.T) {
	in := Input{SystemTime: "2026-05-10", Periods: periods(), Slips: slips()}
	in.Action = &Action{Kind: "correct", SlipID: "S-APR-1-C", BusinessDate: "2026-05-10", Amount: "-1000", Reverses: "S-APR-1"}
	report := Evaluate(in)
	if report.Code != CodeCorrectionPosted || !report.Accepted {
		t.Fatalf("report=%+v", report)
	}
	if report.CorrectionOf != "S-APR-1" || report.PostedPeriod != "2026-05" {
		t.Fatalf("report=%+v", report)
	}
	wantLinks := []string{"S-APR-1", "S-APR-1-C"}
	if !reflect.DeepEqual(report.LinkedSlips, wantLinks) {
		t.Fatalf("links=%v want %v", report.LinkedSlips, wantLinks)
	}
}

func TestCorrectionWithoutOriginalIsRejected(t *testing.T) {
	in := Input{SystemTime: "2026-05-10", Periods: periods(), Slips: slips()}
	in.Action = &Action{Kind: "correct", SlipID: "S-X", BusinessDate: "2026-05-10", Reverses: "missing"}
	if report := Evaluate(in); report.Code != CodeCorrectionMissingOrig {
		t.Fatalf("report=%+v", report)
	}
}

func TestCorrectionIntoClosedPeriodIsRejected(t *testing.T) {
	in := Input{SystemTime: "2026-05-10", Periods: periods(), Slips: slips()}
	in.Action = &Action{Kind: "correct", BusinessDate: "2026-04-20", Reverses: "S-APR-1"}
	if report := Evaluate(in); report.Code != CodeClosedPeriodRejected {
		t.Fatalf("report=%+v", report)
	}
}

func TestLateArrivalAdjustsIntoTheOpenPeriod(t *testing.T) {
	in := Input{SystemTime: "2026-05-10", Periods: periods(), LatePolicy: LateAdjustCurrent}
	in.Action = action("post", "2026-04-20")
	report := Evaluate(in)
	if report.Code != CodeLateAdjusted || !report.Accepted {
		t.Fatalf("report=%+v", report)
	}
	if report.PostedPeriod != "2026-05" {
		t.Fatalf("postedPeriod=%q", report.PostedPeriod)
	}
}

func TestLateArrivalIsHeldByDefault(t *testing.T) {
	in := Input{SystemTime: "2026-05-10", Periods: periods()}
	in.Action = action("post", "2026-04-20")
	report := Evaluate(in)
	if report.Code != CodeLateHeld || report.Accepted {
		t.Fatalf("report=%+v", report)
	}
}

func TestReopenIsNotPermitted(t *testing.T) {
	in := Input{SystemTime: "2026-05-10", Periods: periods()}
	in.Action = &Action{Kind: "reopen", Period: "2026-04"}
	report := Evaluate(in)
	if report.Code != CodeReopenNotPermitted || report.Accepted {
		t.Fatalf("report=%+v", report)
	}
}

func TestCloseIsBlockedByPendingSlip(t *testing.T) {
	pending := []Slip{{ID: "S-1", Period: "2026-05", BusinessDate: "2026-05-10", Amount: "100", Status: "pending"}}
	in := Input{SystemTime: "2026-05-31", Periods: periods(), Slips: pending}
	in.Action = &Action{Kind: "close", Period: "2026-05"}
	if report := Evaluate(in); report.Code != CodeCloseBlockedPending {
		t.Fatalf("report=%+v", report)
	}
	in.Slips[0].Status = "posted"
	if report := Evaluate(in); report.Code != CodeCloseAccepted {
		t.Fatalf("report=%+v", report)
	}
}

func TestUnknownPeriodIsRejected(t *testing.T) {
	in := Input{Periods: periods()}
	in.Action = action("post", "2025-01-01")
	if report := Evaluate(in); report.Code != CodeUnknownPeriod {
		t.Fatalf("report=%+v", report)
	}
}

func replayInput(events []Event) ReplayInput {
	return ReplayInput{Periods: periods(), Events: events, LatePolicy: LateAdjustCurrent}
}

func TestReplayIsIndependentOfArrivalOrder(t *testing.T) {
	ordered := []Event{
		{ID: "E1", BusinessDate: "2026-05-02", Amount: "100", OrderKey: 1, Version: 1},
		{ID: "E2", BusinessDate: "2026-05-03", Amount: "200", OrderKey: 2, Version: 1},
		{ID: "E3", BusinessDate: "2026-04-20", Amount: "50", OrderKey: 3, Version: 1},
	}
	shuffled := []Event{ordered[2], ordered[0], ordered[1]}
	a := Replay(replayInput(ordered))
	b := Replay(replayInput(shuffled))
	if !reflect.DeepEqual(a, b) {
		t.Fatalf("replay depends on arrival order:\n%+v\n%+v", a, b)
	}
	if a.Totals["2026-05"] != "350" {
		t.Fatalf("totals=%v want 350 (late 50 adjusted into May)", a.Totals)
	}
	if len(a.Late) != 1 || a.Late[0] != "E3" {
		t.Fatalf("late=%v", a.Late)
	}
}

func TestReplayHighestVersionWins(t *testing.T) {
	events := []Event{
		{ID: "E1", BusinessDate: "2026-05-02", Amount: "100", OrderKey: 1, Version: 1},
		{ID: "E1", BusinessDate: "2026-05-02", Amount: "300", OrderKey: 1, Version: 2},
	}
	report := Replay(replayInput(events))
	if report.Totals["2026-05"] != "300" {
		t.Fatalf("totals=%v want the highest version 300", report.Totals)
	}
	if len(report.Stale) == 0 {
		t.Fatalf("the superseded version was not recorded as stale: %+v", report)
	}
}

func TestReplayHoldsLateEventsWhenPolicyIsHold(t *testing.T) {
	in := replayInput([]Event{{ID: "E1", BusinessDate: "2026-04-20", Amount: "50", OrderKey: 1, Version: 1}})
	in.LatePolicy = LateHold
	report := Replay(in)
	if len(report.Held) != 1 || report.Held[0] != "E1" {
		t.Fatalf("held=%v", report.Held)
	}
	if len(report.Totals) != 0 {
		t.Fatalf("a held late event changed the totals: %v", report.Totals)
	}
}

func TestCloseDigestSurvivesLateArrival(t *testing.T) {
	closedSlips := slips()
	before := CloseDigest(closedSlips)
	report := Replay(replayInput([]Event{{ID: "E1", BusinessDate: "2026-04-20", Amount: "50", OrderKey: 1, Version: 1}}))
	if len(report.Late) != 1 {
		t.Fatalf("late=%v", report.Late)
	}
	after := CloseDigest(closedSlips)
	if before != after {
		t.Fatalf("the close-time record changed after a late arrival: %s != %s", before, after)
	}
	reprinted := CloseDigest(append([]Slip(nil), closedSlips...))
	if reprinted != before {
		t.Fatalf("reprinting from the same record is not stable")
	}
}

func TestReplayDeterministicAndSorted(t *testing.T) {
	in := replayInput([]Event{
		{ID: "B", BusinessDate: "2026-05-02", Amount: "1", OrderKey: 2, Version: 1},
		{ID: "A", BusinessDate: "2026-05-03", Amount: "2", OrderKey: 1, Version: 1},
	})
	first := Replay(in)
	second := Replay(in)
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("not deterministic")
	}
	if !sort.StringsAreSorted(first.Applied) {
		t.Fatalf("applied is not sorted: %v", first.Applied)
	}
}
