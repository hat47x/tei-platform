package orderassistoffline

import (
	"errors"
	"testing"
)

var actorFor = map[string]string{"manager": "mgr-kato", "sales-head": "head-sato", "director": "dir-mori", "finance": "fin-ueda"}

// approveThrough approves the case's current stages in order until it
// reaches stop (exclusive) under its pinned policy.
func (h *harness) approveThrough(caseID string, p Policy, stop string) {
	h.t.Helper()
	for {
		c := h.caseState(caseID)
		if c.Stage == stop || c.Status != "pending" {
			return
		}
		st := p.Stages[p.index(c.Stage)]
		h.mustDecide(Decision{CaseID: caseID, Stage: c.Stage, Actor: actorFor[st.Role], Decision: "approve", IfMatch: c.QuoteDigest})
	}
}

func validActions(c CaseState) map[string]ActionRecord {
	out := map[string]ActionRecord{}
	for _, a := range c.Actions {
		if a.Valid && a.Decision == "approve" {
			out[a.Stage] = a
		}
	}
	return out
}

// P-WF-01: approval pinned to the quote digest, period-bound delegation,
// declared sendback scope.
func TestPostgresPWF01ApprovalPinnedToQuoteDigest(t *testing.T) {
	h := newHarness(t, ServerOptions{})
	if err := h.server.PublishPriceList(h.ctx, 1, basePrices); err != nil {
		t.Fatal(err)
	}
	a := h.terminal("term-a", 2, false)
	if _, err := a.DownloadMaster(h.ctx); err != nil {
		t.Fatal(err)
	}
	d1 := createAndSync(t, h, a, "Q-WF01", 500).Digest
	if err := h.server.OpenCase(h.ctx, "CASE-WF01", "Q-WF01"); err != nil {
		t.Fatal(err)
	}
	h.mustDecide(Decision{CaseID: "CASE-WF01", Stage: "S1-manager", Actor: "mgr-kato", Decision: "approve", IfMatch: d1})

	// The amount changes after the first approval (an offline edit synced).
	h.op(a.SetField(h.ctx, "Q-WF01", "lines/PEN/quantity", rawInt(5)))
	if _, err := a.Sync(h.ctx); err != nil {
		t.Fatal(err)
	}
	q := h.quote("Q-WF01")
	c := h.caseState("CASE-WF01")
	if q.Status != "valid" || q.Digest == d1 || c.QuoteDigest != q.Digest || c.Stage != "S1-manager" || len(validActions(c)) != 0 {
		t.Fatalf("after the amount changed: quote %s %s, case %+v", q.Status, q.Digest, c)
	}
	if c.Actions[0].InvalidatedReason != "quote-changed" || c.Actions[0].QuoteDigest != d1 {
		t.Fatalf("first approval: %+v", c.Actions[0])
	}
	d2 := q.Digest
	// Approving or advancing on the old digest is refused.
	if err := h.server.Decide(h.ctx, Decision{CaseID: "CASE-WF01", Stage: "S1-manager", Actor: "mgr-kato", Decision: "approve", IfMatch: d1}); !errors.Is(err, ErrStaleQuote) {
		t.Fatalf("approval of the old digest: %v", err)
	}
	if err := h.server.Decide(h.ctx, Decision{CaseID: "CASE-WF01", Stage: "S2-sales-head", Actor: "head-sato", Decision: "approve", IfMatch: d1}); !errors.Is(err, ErrStaleQuote) {
		t.Fatalf("second stage on the old digest: %v", err)
	}
	h.mustDecide(Decision{CaseID: "CASE-WF01", Stage: "S1-manager", Actor: "mgr-kato", Decision: "approve", IfMatch: d2})

	// Delegation: deputy-ito may act for head-sato from 10-10 to 10-20.
	if err := h.server.AddDelegation(h.ctx, "head-sato", "deputy-ito", "sales-head", "2026-10-10", "2026-10-20"); err != nil {
		t.Fatal(err)
	}
	refusals := map[string]string{}
	for _, date := range []string{"2026-10-09", "2026-10-21"} {
		if err := h.server.SetBusinessDate(h.ctx, date); err != nil {
			t.Fatal(err)
		}
		err := h.server.Decide(h.ctx, Decision{CaseID: "CASE-WF01", Stage: "S2-sales-head", Actor: "deputy-ito", Decision: "approve", IfMatch: d2})
		if !errors.Is(err, ErrDelegationOutOfTerm) {
			t.Fatalf("delegation on %s: %v", date, err)
		}
		refusals[date] = "delegation-out-of-term"
	}
	if err := h.server.Decide(h.ctx, Decision{CaseID: "CASE-WF01", Stage: "S2-sales-head", Actor: "sales-b", Decision: "approve", IfMatch: d2}); !errors.Is(err, ErrNotAuthorized) {
		t.Fatalf("approval without role or delegation: %v", err)
	}
	if err := h.server.SetBusinessDate(h.ctx, "2026-10-20"); err != nil {
		t.Fatal(err)
	}
	h.mustDecide(Decision{CaseID: "CASE-WF01", Stage: "S2-sales-head", Actor: "deputy-ito", Decision: "approve", IfMatch: d2})
	c = h.caseState("CASE-WF01")
	delegated := validActions(c)["S2-sales-head"]
	if delegated.Actor != "deputy-ito" || delegated.OnBehalfOf != "head-sato" || delegated.BusinessDate != "2026-10-20" {
		t.Fatalf("delegated approval: %+v", delegated)
	}

	// Sendback from the director: the policy sends back to S2 and withdraws
	// approvals at S2 and later; S1 stays.
	h.mustDecide(Decision{CaseID: "CASE-WF01", Stage: "S3-director", Actor: "dir-mori", Decision: "sendback", IfMatch: d2})
	c = h.caseState("CASE-WF01")
	va := validActions(c)
	if c.Stage != "S2-sales-head" || len(va) != 1 || va["S1-manager"].ActionID == 0 {
		t.Fatalf("after sendback: stage %s valid %+v", c.Stage, va)
	}
	var sentBack ActionRecord
	for _, a := range c.Actions {
		if a.Stage == "S2-sales-head" && a.Decision == "approve" {
			sentBack = a
		}
	}
	if sentBack.Valid || sentBack.InvalidatedReason != "sendback-from-S3-director" {
		t.Fatalf("withdrawn S2 approval: %+v", sentBack)
	}
	h.mustDecide(Decision{CaseID: "CASE-WF01", Stage: "S2-sales-head", Actor: "head-sato", Decision: "approve", IfMatch: d2})

	// The director stage needs the credit first.
	if err := h.server.Decide(h.ctx, Decision{CaseID: "CASE-WF01", Stage: "S3-director", Actor: "dir-mori", Decision: "approve", IfMatch: d2}); !errors.Is(err, ErrCreditNotReady) {
		t.Fatalf("director before credit: %v", err)
	}
	if _, err := h.saga().Run(h.ctx, "CASE-WF01"); err != nil {
		t.Fatal(err)
	}
	h.mustDecide(Decision{CaseID: "CASE-WF01", Stage: "S3-director", Actor: "dir-mori", Decision: "approve", IfMatch: d2})
	if _, err := h.saga().Run(h.ctx, "CASE-WF01"); err != nil {
		t.Fatal(err)
	}
	conv, err := Convergence(h.ctx, h.server.Pool, h.credit, "CASE-WF01")
	if err != nil || conv != "all-complete" {
		t.Fatalf("convergence: %s %v", conv, err)
	}
	c = h.caseState("CASE-WF01")
	for stage, a := range validActions(c) {
		if a.QuoteDigest != d2 {
			t.Fatalf("valid approval at %s is for %s, not the current digest", stage, a.QuoteDigest)
		}
	}
	observed(t, "pwf01", map[string]any{"refusedEvents": h.countEvents("decision-refused", "CASE-WF01"), "delegationRefusals": refusals,
		"delegated": delegated, "actions": len(c.Actions), "validApprovals": len(validActions(c)), "convergence": conv,
		"invalidatedByQuoteChange": c.Actions[0].InvalidatedReason, "invalidatedBySendback": sentBack.InvalidatedReason})
}

// P-WF-02: the policy is revised while cases are in flight.
func TestPostgresPWF02PolicyRevisionAndMigration(t *testing.T) {
	h := newHarness(t, ServerOptions{})
	if err := h.server.PublishPriceList(h.ctx, 1, basePrices); err != nil {
		t.Fatal(err)
	}
	a := h.terminal("term-a", 2, false)
	if _, err := a.DownloadMaster(h.ctx); err != nil {
		t.Fatal(err)
	}
	cases := []string{"C1", "C2", "C3", "C4", "C5"}
	for _, id := range cases {
		createAndSync(t, h, a, "Q-"+id, 500)
	}
	for _, id := range cases[:4] {
		if err := h.server.OpenCase(h.ctx, "CASE-"+id, "Q-"+id); err != nil {
			t.Fatal(err)
		}
	}
	h.approveThrough("CASE-C2", h.policyV1, "S2-sales-head")
	h.approveThrough("CASE-C3", h.policyV1, "S3-director")
	h.approveThrough("CASE-C4", h.policyV1, "S3-director")

	// Revision: v2 becomes the policy for new cases.
	if err := h.server.InstallPolicy(h.ctx, h.policyV2, true); err != nil {
		t.Fatal(err)
	}
	if err := h.server.OpenCase(h.ctx, "CASE-C5", "Q-C5"); err != nil {
		t.Fatal(err)
	}
	if c := h.caseState("CASE-C5"); c.PolicyVersion != h.policyV2.Version || c.Stage != "T1-manager" {
		t.Fatalf("new case: %+v", c)
	}

	// C4 completes on v1.
	if _, err := h.saga().Run(h.ctx, "CASE-C4"); err != nil {
		t.Fatal(err)
	}
	h.approveThrough("CASE-C4", h.policyV1, "")
	if _, err := h.saga().Run(h.ctx, "CASE-C4"); err != nil {
		t.Fatal(err)
	}
	c4 := h.caseState("CASE-C4")
	if c4.Status != "approved" || c4.PolicyVersion != h.policyV1.Version {
		t.Fatalf("C4: %+v", c4)
	}
	for _, a := range c4.Actions {
		if a.PolicyVersion != h.policyV1.Version {
			t.Fatalf("C4 decision recorded under %s", a.PolicyVersion)
		}
	}
	var handoffPolicy string
	if err := h.server.Pool.QueryRow(h.ctx, `SELECT policy_version FROM oapo.handoff WHERE case_id='CASE-C4'`).Scan(&handoffPolicy); err != nil || handoffPolicy != h.policyV1.Version {
		t.Fatalf("C4 handoff policy: %s %v", handoffPolicy, err)
	}

	// Migration by the table: C1 (S1) lands on T1, C3 (S3) lands on T2 with
	// the S1 approval carried; C2 (S2, unmapped) is refused and stays on v1.
	landed := map[string]string{}
	for _, id := range []string{"CASE-C1", "CASE-C3"} {
		to, err := h.server.MigrateCase(h.ctx, id, h.migration)
		if err != nil {
			t.Fatalf("migrate %s: %v", id, err)
		}
		landed[id] = to
	}
	if landed["CASE-C1"] != "T1-manager" || landed["CASE-C3"] != "T2-finance" {
		t.Fatalf("landing: %v", landed)
	}
	if _, err := h.server.MigrateCase(h.ctx, "CASE-C2", h.migration); !errors.Is(err, ErrUnmappedStage) {
		t.Fatalf("unmapped stage: %v", err)
	}
	c2 := h.caseState("CASE-C2")
	if c2.PolicyVersion != h.policyV1.Version || c2.Stage != "S2-sales-head" {
		t.Fatalf("refused case changed: %+v", c2)
	}
	c3 := h.caseState("CASE-C3")
	va := validActions(c3)
	if c3.PolicyVersion != h.policyV2.Version || c3.Stage != "T2-finance" || len(va) != 1 ||
		va["T1-manager"].MigratedFrom != h.policyV1.Version+"#S1-manager" {
		t.Fatalf("C3 after migration: %+v", c3)
	}
	notCarried := 0
	for _, a := range c3.Actions {
		if a.InvalidatedReason == "not-carried-by-migration" {
			notCarried++
		}
	}
	if notCarried != 1 {
		t.Fatalf("S2 approval of C3 should be withdrawn: %+v", c3.Actions)
	}
	// An old-stage decision on a migrated case is refused.
	if err := h.server.Decide(h.ctx, Decision{CaseID: "CASE-C3", Stage: "S3-director", Actor: "dir-mori", Decision: "approve", IfMatch: c3.QuoteDigest}); !errors.Is(err, ErrWrongStage) {
		t.Fatalf("v1 stage on a migrated case: %v", err)
	}

	// Each case finishes on its own version.
	h.approveThrough("CASE-C3", h.policyV2, "T3-director")
	if _, err := h.saga().Run(h.ctx, "CASE-C3"); err != nil {
		t.Fatal(err)
	}
	h.approveThrough("CASE-C3", h.policyV2, "")
	h.approveThrough("CASE-C2", h.policyV1, "S3-director")
	if _, err := h.saga().Run(h.ctx, "CASE-C2"); err != nil {
		t.Fatal(err)
	}
	h.approveThrough("CASE-C2", h.policyV1, "")
	h.approveThrough("CASE-C1", h.policyV2, "T3-director")
	versions := map[string][]string{}
	for _, id := range []string{"CASE-C1", "CASE-C2", "CASE-C3"} {
		c := h.caseState(id)
		for _, a := range c.Actions {
			if a.Valid {
				versions[id] = append(versions[id], a.PolicyVersion+"#"+a.Stage)
			}
		}
	}
	if h.caseState("CASE-C2").Status != "approved" || h.caseState("CASE-C3").Status != "approved" {
		t.Fatal("C2 or C3 did not complete")
	}
	for _, s := range versions["CASE-C2"] {
		if s[:len(h.policyV1.Version)] != h.policyV1.Version {
			t.Fatalf("C2 decision under %s", s)
		}
	}
	for _, s := range versions["CASE-C3"] {
		if s[:len(h.policyV2.Version)] != h.policyV2.Version {
			t.Fatalf("C3 valid decision under %s", s)
		}
	}
	observed(t, "pwf02", map[string]any{"landed": landed, "refused": "CASE-C2@S2-sales-head", "c4Policy": c4.PolicyVersion,
		"c5Policy": h.policyV2.Version, "validDecisions": versions, "migrationRefusedEvents": h.countEvents("case-migration-refused", "CASE-C2")})
}
