package orderassistoffline

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// C5: an offline sales terminal's quote meets approval and an unknown
// external credit outcome. One sequence on one database, no state reset
// between the requirements (P-MIX-02, P-MIX-03, P-WF-01, P-TX-02, P-TX-03).
// Each step appends to a linked run record that names the quote digest the
// step acted on; from the final revalidation on, every step must act on the
// same digest.
func TestPostgresC5ThroughOfflineToCredit(t *testing.T) {
	h := newHarness(t, ServerOptions{})
	const quoteID, caseID = "Q-C5", "CASE-C5"
	var chain Chain
	id := Identity{
		QuoteID: quoteID, RuleVersion: Gen2.RuleVersion,
		TerminalAGeneration2: h.real.Get(2, RuntimeNodeJS).ArtifactDigest,
		TerminalBGeneration1: h.real.Get(1, RuntimeNodeJS).ArtifactDigest,
		ServerRealization:    h.real.Get(2, RuntimeGoNative).ArtifactDigest,
		BatchRealization:     h.real.Get(2, RuntimeWASI).ArtifactDigest,
		WASIRunner:           h.real.Get(2, RuntimeWASI).HostDigest,
		PolicyDigest:         policyDigest(h.policyV1),
	}
	step := func(name, digest string, passed bool, result map[string]any) {
		t.Helper()
		chain.Append(name, id, digest, passed, result)
		if !passed {
			t.Fatalf("C5 step %s failed: %v", name, result)
		}
	}
	if err := h.server.PublishPriceList(h.ctx, 1, basePrices); err != nil {
		t.Fatal(err)
	}
	a := h.terminal("term-a", 2, false) // current generation
	b := h.terminal("term-b", 1, false) // old generation, preserving
	for _, term := range []*Terminal{a, b} {
		if _, err := term.DownloadMaster(h.ctx); err != nil {
			t.Fatal(err)
		}
	}
	step("terminals-loaded-master", "", true, map[string]any{"priceListVersion": 1})

	// 1. Terminal A creates the quote offline and synchronises it.
	create := h.op(a.CreateQuote(h.ctx, quoteID, "C-100", 500, []NewLine{{SKU: "PEN", Quantity: 3}, {SKU: "BOOK", Quantity: 2, Packaging: "box"}}))
	step("offline-create", "", create.Basis.LocalResult.NetMinor == 1473, map[string]any{"op": create.OpID, "localNet": create.Basis.LocalResult.NetMinor})
	if _, err := a.Sync(h.ctx); err != nil {
		t.Fatal(err)
	}
	q1 := h.quote(quoteID)
	step("sync-create", q1.Digest, q1.Status == "valid", map[string]any{"status": q1.Status})

	// 2. The case opens on that digest and the manager approves it.
	if err := h.server.OpenCase(h.ctx, caseID, quoteID); err != nil {
		t.Fatal(err)
	}
	h.mustDecide(Decision{CaseID: caseID, Stage: "S1-manager", Actor: "mgr-kato", Decision: "approve", IfMatch: q1.Digest})
	step("first-approval", q1.Digest, h.caseState(caseID).Stage == "S2-sales-head", nil)

	// 3. Both terminals download the quote, go offline and edit it.
	if _, err := b.DownloadQuote(h.ctx, quoteID); err != nil {
		t.Fatal(err)
	}
	h.op(a.SetField(h.ctx, quoteID, "lines/PEN/quantity", rawInt(5)))
	opB1 := h.op(b.EditLineQuantity(h.ctx, quoteID, "PEN", 4))
	opB2 := h.op(b.EditLineQuantity(h.ctx, quoteID, "BOOK", 6))
	bLocal, _ := b.LocalDoc(quoteID)
	_, bKeptDiscount := bLocal.Get("discountRateBp")
	_, bKeptPackaging := bLocal.Get("lines/BOOK/packaging")
	step("offline-edits", q1.Digest, bKeptDiscount && bKeptPackaging && opB2.Basis.LocalResult.SubtotalMinor == 4*250+6*400,
		map[string]any{"bKeptUnknown": bKeptDiscount && bKeptPackaging, "bLocalSubtotal": opB2.Basis.LocalResult.SubtotalMinor, "bGeneration": opB2.Basis.Generation})

	// 4. The server revises the BOOK price while both are offline.
	if _, err := h.server.PublishRevisedPrices(h.ctx, map[string]int64{"BOOK": 450}); err != nil {
		t.Fatal(err)
	}
	step("price-revised", q1.Digest, true, map[string]any{"BOOK": 450, "priceListVersion": 2})

	// 5. A synchronises: the quote changes, stops on the old price, and the
	// first approval is withdrawn.
	if _, err := a.Sync(h.ctx); err != nil {
		t.Fatal(err)
	}
	q2 := h.quote(quoteID)
	c := h.caseState(caseID)
	step("sync-a-revalidated", q2.Digest, q2.Status == "stale-price" && q2.Digest != q1.Digest && c.Stage == "S1-manager" && len(validActions(c)) == 0 &&
		c.QuoteDigest == q2.Digest, map[string]any{"status": q2.Status, "staleLines": revalidation(t, q2).StaleLines, "caseStage": c.Stage})

	// 6. Old-generation B synchronises: PEN conflicts per field, BOOK applies,
	// unknown members survive.
	rb, err := b.Sync(h.ctx)
	if err != nil {
		t.Fatal(err)
	}
	q3 := h.quote(quoteID)
	conflicts, err := h.server.Conflicts(h.ctx, quoteID)
	if err != nil {
		t.Fatal(err)
	}
	doc3, _ := ParseDoc(q3.Doc)
	_, packagingKept := doc3.Get("lines/BOOK/packaging")
	outcomes := map[string]string{}
	for _, r := range rb.Results {
		outcomes[r.OpID] = r.Outcome
	}
	step("sync-b-old-generation", q3.Digest, q3.Status == "conflict" && len(conflicts) == 1 && conflicts[0].Path == "lines/PEN/quantity" &&
		conflicts[0].OpID == opB1.OpID && EqualJSON(conflicts[0].OpValue, rawInt(4)) && EqualJSON(conflicts[0].ServerValue, rawInt(5)) &&
		outcomes[opB2.OpID] == "applied" && packagingKept,
		map[string]any{"status": q3.Status, "outcomes": outcomes, "conflict": conflicts[0], "packagingKept": packagingKept})

	// 7. A person resolves the conflict and reprices: the quote is valid again.
	if err := h.server.ResolveConflict(h.ctx, conflicts[0].ID, rawInt(4), "mgr-kato"); err != nil {
		t.Fatal(err)
	}
	if err := h.server.Reprice(h.ctx, quoteID, "sales-b"); err != nil {
		t.Fatal(err)
	}
	final := h.quote(quoteID)
	frv := revalidation(t, final)
	d := final.Digest
	step("revalidated-final", d, final.Status == "valid" && frv.Judgment.Result.NetMinor == 3515 && frv.PriceListVersion == 2 &&
		len(frv.Judgment.Used) == 2, map[string]any{"net": frv.Judgment.Result.NetMinor, "ruleVersion": frv.Judgment.RuleVersion,
		"sites": []Site{frv.Judgment.Used[0].Site, frv.Judgment.Used[1].Site}})

	// 8. Approvals on the old digests are refused; the current one proceeds,
	// with a delegated approval that is first out of term.
	errOld := h.server.Decide(h.ctx, Decision{CaseID: caseID, Stage: "S1-manager", Actor: "mgr-kato", Decision: "approve", IfMatch: q1.Digest})
	step("stale-approval-refused", d, errors.Is(errOld, ErrStaleQuote), map[string]any{"error": errString(errOld)})
	h.mustDecide(Decision{CaseID: caseID, Stage: "S1-manager", Actor: "mgr-kato", Decision: "approve", IfMatch: d})
	if err := h.server.AddDelegation(h.ctx, "head-sato", "deputy-ito", "sales-head", "2026-10-10", "2026-10-20"); err != nil {
		t.Fatal(err)
	}
	errTerm := h.server.Decide(h.ctx, Decision{CaseID: caseID, Stage: "S2-sales-head", Actor: "deputy-ito", Decision: "approve", IfMatch: d})
	if err := h.server.SetBusinessDate(h.ctx, "2026-10-12"); err != nil {
		t.Fatal(err)
	}
	h.mustDecide(Decision{CaseID: caseID, Stage: "S2-sales-head", Actor: "deputy-ito", Decision: "approve", IfMatch: d})
	va := validActions(h.caseState(caseID))
	step("approvals-on-final-digest", d, errors.Is(errTerm, ErrDelegationOutOfTerm) && va["S1-manager"].QuoteDigest == d &&
		va["S2-sales-head"].QuoteDigest == d && va["S2-sales-head"].OnBehalfOf == "head-sato",
		map[string]any{"outOfTerm": errString(errTerm), "delegated": va["S2-sales-head"]})

	// 9. External credit: the provider consumes and drops the answer twice.
	h.setCreditFault("reserve", 2)
	steps, err := h.saga().Run(h.ctx, caseID)
	if err != nil {
		t.Fatal(err)
	}
	key := CreditKey(caseID, d)
	res, err := h.credit.Inquire(h.ctx, key)
	if err != nil {
		t.Fatal(err)
	}
	state, attempts, _, creditDigest := h.creditState(caseID)
	pc := h.provider(key)
	step("credit-unknown-resolved", creditDigest, state == "consumed" && pc.Reservations == 1 && pc.ConsumedEvents == 1 && res.QuoteRef == d &&
		steps[len(steps)-1].Action == "inquire" && res.AmountMinor == 3515,
		map[string]any{"steps": steps, "attempts": attempts, "provider": pc, "providerQuoteRef": res.QuoteRef})

	// 10. The director rejects; the compensation is interrupted and resumed.
	h.mustDecide(Decision{CaseID: caseID, Stage: "S3-director", Actor: "dir-mori", Decision: "reject", IfMatch: d})
	code, out := h.sagaProcess(caseID, "exit-after-release-response")
	mid := h.convergence(caseID)
	midState, _, _, _ := h.creditState(caseID)
	step("compensation-interrupted", d, code == FaultExitCode && midState == "release-pending" && mid != "all-compensated",
		map[string]any{"exit": code, "localState": midState, "convergence": mid, "output": out})
	code, out = h.sagaProcess(caseID, "")
	conv := h.convergence(caseID)
	pc = h.provider(key)
	var handoffDigest, handoffOutcome string
	if err := h.server.Pool.QueryRow(h.ctx, `SELECT quote_digest, outcome FROM oapo.handoff WHERE case_id=$1`, caseID).Scan(&handoffDigest, &handoffOutcome); err != nil {
		t.Fatal(err)
	}
	step("compensated-and-withdrawn", handoffDigest, code == 0 && conv == "all-compensated" && handoffOutcome == "withdrawn" &&
		pc.Reservations == 1 && pc.ConsumedEvents == 1 && pc.ReleaseCount == 1 && pc.ReleasedEvents == 1 && pc.UsedMinor == 0,
		map[string]any{"exit": code, "convergence": conv, "provider": pc, "handoff": handoffOutcome})

	// The terminal origin of the quote stays on record.
	var createdBy string
	if err := h.server.Pool.QueryRow(h.ctx, `SELECT created_by_op FROM oapo.quote WHERE quote_id=$1`, quoteID).Scan(&createdBy); err != nil || createdBy != create.OpID {
		t.Fatalf("quote origin: %s %v", createdBy, err)
	}
	required := []string{"terminals-loaded-master", "offline-create", "sync-create", "first-approval", "offline-edits", "price-revised",
		"sync-a-revalidated", "sync-b-old-generation", "revalidated-final", "stale-approval-refused", "approvals-on-final-digest",
		"credit-unknown-resolved", "compensation-interrupted", "compensated-and-withdrawn"}
	if err := chain.Verify(required, "revalidated-final"); err != nil {
		t.Fatal(err)
	}
	// A record that names another digest after the carry step breaks it.
	broken := Chain{Records: append([]Record(nil), chain.Records...)}
	broken.Records = broken.Records[:len(broken.Records)-1]
	broken.Append("compensated-and-withdrawn", id, q2.Digest, true, nil)
	if err := broken.Verify(required, "revalidated-final"); !errors.Is(err, ErrChain) {
		t.Fatalf("a different digest after the carry step was accepted: %v", err)
	}
	if dir := os.Getenv(outDirEnv); dir != "" {
		raw, _ := json.MarshalIndent(chain, "", "  ")
		if err := os.WriteFile(filepath.Join(dir, "chain.json"), raw, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	observed(t, "c5", map[string]any{"records": len(chain.Records), "carriedDigest": d, "createdByOp": createdBy,
		"digests": []string{q1.Digest, q2.Digest, q3.Digest, d}, "provider": pc, "convergence": conv})
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
