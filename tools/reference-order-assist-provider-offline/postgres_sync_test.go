package orderassistoffline

import (
	"encoding/json"
	"errors"
	"sort"
	"testing"
)

var basePrices = map[string]int64{"PEN": 250, "BOOK": 400, "INK": 120}

// createAndSync creates a quote offline on a generation-2 terminal and
// synchronises it.
func createAndSync(t *testing.T, h *harness, term *Terminal, id string, discount int64) QuoteState {
	t.Helper()
	h.op(term.CreateQuote(h.ctx, id, "C-100", discount, []NewLine{{SKU: "PEN", Quantity: 3}, {SKU: "BOOK", Quantity: 2, Packaging: "box"}}))
	resp, err := term.Sync(h.ctx)
	if err != nil {
		t.Fatal(err)
	}
	st := resp.Quotes[id]
	if st.Status != "valid" {
		t.Fatalf("created quote %s is %s: %s", id, st.Status, st.Revalidation)
	}
	return st
}

func revalidation(t *testing.T, st QuoteState) Revalidation {
	t.Helper()
	var rv Revalidation
	if err := json.Unmarshal(st.Revalidation, &rv); err != nil {
		t.Fatal(err)
	}
	return rv
}

func opsByOutcome(t *testing.T, h *harness, quoteID string) map[string]string {
	t.Helper()
	rows, err := h.server.Pool.Query(h.ctx, `SELECT op_id, outcome FROM oapo.sync_op WHERE quote_id=$1`, quoteID)
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]string{}
	for rows.Next() {
		var id, o string
		if err := rows.Scan(&id, &o); err != nil {
			t.Fatal(err)
		}
		out[id] = o
	}
	return out
}

// P-MIX-02: two terminals update the same quote offline while the server
// revises a unit price; synchronisation is per operation, conflicts are per
// field, nothing is lost, and the old-price quote stops at revalidation.
func TestPostgresPMIX02OfflineSyncConflictAndReprice(t *testing.T) {
	h := newHarness(t, ServerOptions{})
	if err := h.server.PublishPriceList(h.ctx, 1, basePrices); err != nil {
		t.Fatal(err)
	}
	a, b := h.terminal("term-a", 2, false), h.terminal("term-b", 2, false)
	for _, term := range []*Terminal{a, b} {
		if _, err := term.DownloadMaster(h.ctx); err != nil {
			t.Fatal(err)
		}
	}
	created := createAndSync(t, h, a, "Q-MIX02", 500)
	if _, err := b.DownloadQuote(h.ctx, "Q-MIX02"); err != nil {
		t.Fatal(err)
	}

	// Offline on both terminals.
	sent := map[string]json.RawMessage{}
	op := h.op(a.SetField(h.ctx, "Q-MIX02", "lines/PEN/quantity", rawInt(5)))
	sent[op.OpID] = rawInt(5)
	op = h.op(a.SetField(h.ctx, "Q-MIX02", "discountRateBp", rawInt(300)))
	sent[op.OpID] = rawInt(300)
	op = h.op(b.SetField(h.ctx, "Q-MIX02", "lines/PEN/quantity", rawInt(4)))
	sent[op.OpID] = rawInt(4)
	op = h.op(b.SetField(h.ctx, "Q-MIX02", "discountRateBp", rawInt(700)))
	sent[op.OpID] = rawInt(700)
	op = h.op(b.SetField(h.ctx, "Q-MIX02", "lines/BOOK/quantity", rawInt(6)))
	sent[op.OpID] = rawInt(6)
	bOutbox, _ := b.Outbox()

	// Meanwhile the server revises the BOOK price.
	newVersion, err := h.server.PublishRevisedPrices(h.ctx, map[string]int64{"BOOK": 450})
	if err != nil || newVersion != 2 {
		t.Fatalf("revise: %d %v", newVersion, err)
	}

	ra, err := a.Sync(h.ctx)
	if err != nil {
		t.Fatal(err)
	}
	afterA := ra.Quotes["Q-MIX02"]
	if afterA.Status != "stale-price" {
		t.Fatalf("after A: %s", afterA.Status)
	}
	rv := revalidation(t, afterA)
	if len(rv.StaleLines) != 1 || rv.StaleLines[0] != (StaleLine{SKU: "BOOK", QuotedUnitPriceMinor: 400, BasisVersion: 1, CurrentUnitPrice: 450, CurrentVersion: 2}) {
		t.Fatalf("stale lines: %+v", rv.StaleLines)
	}
	if _, err := h.server.Quote(h.ctx, "Q-MIX02"); err != nil {
		t.Fatal(err)
	}
	if err := h.server.OpenCase(h.ctx, "CASE-STALE", "Q-MIX02"); !errors.Is(err, ErrQuoteNotValid) {
		t.Fatalf("a quote on an old price was accepted for approval: %v", err)
	}

	rb, err := b.Sync(h.ctx)
	if err != nil {
		t.Fatal(err)
	}
	afterB := rb.Quotes["Q-MIX02"]
	if afterB.Status != "conflict" {
		t.Fatalf("after B: %s", afterB.Status)
	}
	outcomes := map[string]string{}
	for _, r := range rb.Results {
		outcomes[r.OpID] = r.Outcome
	}
	// Replaying B's request (a lost response) must not apply anything twice.
	replay, refusal, err := h.server.Sync(h.ctx, SyncRequest{TerminalID: b.ID, Generation: 2, RealizationDigest: b.Realization.ArtifactDigest, Ops: bOutbox})
	if err != nil || refusal != nil {
		t.Fatal(err, refusal)
	}
	for _, r := range replay.Results {
		if !r.Replayed || r.Outcome != outcomes[r.OpID] {
			t.Fatalf("replayed op %s: %+v", r.OpID, r)
		}
	}
	if replay.Quotes["Q-MIX02"].Version != afterB.Version || replay.Quotes["Q-MIX02"].Digest != afterB.Digest {
		t.Fatalf("replay changed the quote: %s -> %s", afterB.Digest, replay.Quotes["Q-MIX02"].Digest)
	}

	conflicts, err := h.server.Conflicts(h.ctx, "Q-MIX02")
	if err != nil {
		t.Fatal(err)
	}
	var paths []string
	for _, c := range conflicts {
		paths = append(paths, c.Path)
		if !EqualJSON(c.OpValue, sent[c.OpID]) {
			t.Errorf("conflict %d lost the terminal value: %s vs sent %s", c.ID, c.OpValue, sent[c.OpID])
		}
	}
	sort.Strings(paths)
	if len(conflicts) != 2 || paths[0] != "discountRateBp" || paths[1] != "lines/PEN/quantity" {
		t.Fatalf("conflicts: %+v", conflicts)
	}

	// No lost update: every operation is on the server, and its value is
	// either in the document or in a conflict record.
	doc, _ := ParseDoc(afterB.Doc)
	ops := opsByOutcome(t, h, "Q-MIX02")
	inConflict := map[string]bool{}
	for _, c := range conflicts {
		inConflict[c.OpID] = true
	}
	lost := 0
	for id, value := range sent {
		if _, ok := ops[id]; !ok {
			t.Errorf("op %s missing on the server", id)
			lost++
			continue
		}
		var op Op
		var body []byte
		_ = h.server.Pool.QueryRow(h.ctx, `SELECT body FROM oapo.sync_op WHERE op_id=$1`, id).Scan(&body)
		_ = json.Unmarshal(body, &op)
		cur, _ := doc.Get(op.Path)
		if !inConflict[id] && !EqualJSON(cur, value) {
			t.Errorf("op %s (%s=%s) neither applied nor in conflict (now %s)", id, op.Path, value, cur)
			lost++
		}
	}
	var basis []int
	rows, _ := h.server.Pool.Query(h.ctx, `SELECT DISTINCT price_list_version FROM oapo.sync_op WHERE quote_id='Q-MIX02' AND terminal_id LIKE 'term-%' ORDER BY 1`)
	for rows.Next() {
		var v int
		_ = rows.Scan(&v)
		basis = append(basis, v)
	}
	if len(basis) != 1 || basis[0] != 1 {
		t.Fatalf("terminal operations must record master version 1 as their basis: %v", basis)
	}
	if b2, _ := b.Outbox(); len(b2) != 0 {
		t.Fatalf("acknowledged ops left in B's outbox: %d", len(b2))
	}

	// A person resolves both conflicts, then reprices from the current list.
	for _, c := range conflicts {
		chosen := c.OpValue // the terminal's PEN quantity (4)
		if c.Path == "discountRateBp" {
			chosen = rawInt(500) // neither 300 nor 700
		}
		if err := h.server.ResolveConflict(h.ctx, c.ID, chosen, "mgr-kato"); err != nil {
			t.Fatal(err)
		}
	}
	if st := h.quote("Q-MIX02"); st.Status != "stale-price" {
		t.Fatalf("after resolution: %s", st.Status)
	}
	if err := h.server.Reprice(h.ctx, "Q-MIX02", "sales-b"); err != nil {
		t.Fatal(err)
	}
	final := h.quote("Q-MIX02")
	frv := revalidation(t, final)
	if final.Status != "valid" || frv.Judgment.Result.NetMinor != 3515 || frv.Judgment.Result.SubtotalMinor != 3700 || frv.PriceListVersion != 2 ||
		len(frv.Judgment.Used) != 2 || len(frv.Judgment.Excluded) != 0 {
		t.Fatalf("final: %s %+v", final.Status, frv)
	}
	if err := h.server.OpenCase(h.ctx, "CASE-MIX02", "Q-MIX02"); err != nil {
		t.Fatal(err)
	}
	observed(t, "pmix02", map[string]any{"opsSent": len(sent), "opsOnServer": len(ops), "lost": lost, "conflictPaths": paths,
		"outcomesB": outcomes, "staleLines": rv.StaleLines, "statusAfterA": afterA.Status, "statusAfterB": afterB.Status,
		"terminalBasisPriceListVersions": basis, "createdDigest": created.Digest, "finalDigest": final.Digest,
		"finalNetMinor": frv.Judgment.Result.NetMinor, "replayedOps": len(replay.Results)})
	if lost != 0 {
		t.Fatalf("%d updates lost", lost)
	}
}

// P-MIX-03: a terminal still running generation 1 synchronises with a
// server on generation 2.
func TestPostgresPMIX03OldGenerationTerminal(t *testing.T) {
	h := newHarness(t, ServerOptions{})
	if err := h.server.PublishPriceList(h.ctx, 1, basePrices); err != nil {
		t.Fatal(err)
	}
	a := h.terminal("term-a", 2, false)
	if _, err := a.DownloadMaster(h.ctx); err != nil {
		t.Fatal(err)
	}
	createAndSync(t, h, a, "Q-MIX03", 500)
	old := h.terminal("term-old", 1, false)
	naive := h.terminal("term-naive", 1, true)
	for _, term := range []*Terminal{old, naive} {
		if _, err := term.DownloadMaster(h.ctx); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := old.DownloadQuote(h.ctx, "Q-MIX03"); err != nil {
		t.Fatal(err)
	}
	// The preserving terminal keeps members it does not know in its store.
	local, _ := old.LocalDoc("Q-MIX03")
	if v, ok := local.Get("discountRateBp"); !ok || !EqualJSON(v, rawInt(500)) {
		t.Fatalf("old terminal lost discountRateBp locally: %s", v)
	}
	if v, ok := local.Get("lines/BOOK/packaging"); !ok || !EqualJSON(v, rawString("box")) {
		t.Fatalf("old terminal lost packaging locally: %s", v)
	}

	// Old terminal edits BOOK offline and writes the whole line back.
	h.op(old.EditLineQuantity(h.ctx, "Q-MIX03", "BOOK", 3))
	r1, err := old.Sync(h.ctx)
	if err != nil {
		t.Fatal(err)
	}
	fields := map[string]string{}
	for _, f := range r1.Results[0].Fields {
		fields[f.Path] = f.Outcome
	}
	if fields["lines/BOOK/quantity"] != "applied" || fields["lines/BOOK/packaging"] != "unchanged" {
		t.Fatalf("old terminal write-back: %+v", fields)
	}
	st := h.quote("Q-MIX03")
	doc, _ := ParseDoc(st.Doc)
	if v, ok := doc.Get("lines/BOOK/packaging"); !ok || !EqualJSON(v, rawString("box")) {
		t.Fatal("server lost packaging after the old terminal's write-back")
	}
	rv := revalidation(t, st)
	if st.Status != "valid" || len(rv.TerminalChecks) != 1 || !rv.TerminalChecks[0].Matches || rv.TerminalChecks[0].TerminalGeneration != 1 {
		t.Fatalf("old terminal's local quote: %s %+v", st.Status, rv.TerminalChecks)
	}

	// A naive generation-1 terminal drops the member; the server keeps it.
	if _, err := naive.DownloadQuote(h.ctx, "Q-MIX03"); err != nil {
		t.Fatal(err)
	}
	if nd, _ := naive.LocalDoc("Q-MIX03"); func() bool { _, ok := nd.Get("lines/BOOK/packaging"); return ok }() {
		t.Fatal("negative control: the naive terminal kept packaging")
	}
	h.op(naive.EditLineQuantity(h.ctx, "Q-MIX03", "BOOK", 4))
	r2, err := naive.Sync(h.ctx)
	if err != nil {
		t.Fatal(err)
	}
	guard := ""
	for _, f := range r2.Results[0].Fields {
		if f.Path == "lines/BOOK/packaging" {
			guard = f.Outcome
		}
	}
	doc, _ = ParseDoc(h.quote("Q-MIX03").Doc)
	if _, ok := doc.Get("lines/BOOK/packaging"); guard != "preserved-unknown" || !ok {
		t.Fatalf("server guard: %s", guard)
	}

	// Negative control: without the guard the naive write-back loses it.
	h.server.Opts.UnsafeDropUnknownOnPutLine = true
	if _, err := naive.DownloadQuote(h.ctx, "Q-MIX03"); err != nil {
		t.Fatal(err)
	}
	h.op(naive.EditLineQuantity(h.ctx, "Q-MIX03", "BOOK", 5))
	if _, err := naive.Sync(h.ctx); err != nil {
		t.Fatal(err)
	}
	h.server.Opts.UnsafeDropUnknownOnPutLine = false
	doc, _ = ParseDoc(h.quote("Q-MIX03").Doc)
	_, keptUnsafe := doc.Get("lines/BOOK/packaging")
	if keptUnsafe {
		t.Fatal("negative control did not lose the member")
	}

	// The server narrows the accepted range: the old terminal is refused in a
	// form it understands and keeps its operation.
	if err := h.server.SetAcceptedGenerations(h.ctx, 2, 2); err != nil {
		t.Fatal(err)
	}
	pending := h.op(old.EditLineQuantity(h.ctx, "Q-MIX03", "PEN", 7))
	_, err = old.Sync(h.ctx)
	var refused *ErrSyncRefused
	if !errors.As(err, &refused) || refused.Refusal != (SyncRefusal{Code: "generation-out-of-range", TerminalGeneration: 1, AcceptedMin: 2, AcceptedMax: 2}) {
		t.Fatalf("old terminal after narrowing: %v", err)
	}
	oldRefusal := refused.Refusal
	if ob, _ := old.Outbox(); len(ob) != 1 || ob[0].OpID != pending.OpID {
		t.Fatalf("refused op not kept in the outbox: %+v", ob)
	}
	future := &Terminal{ID: "term-future", Generation: 3, Realization: h.real.Get(2, RuntimeNodeJS), Store: old.Store, ServerURL: h.http.URL, HTTP: old.HTTP}
	_, err = future.Sync(h.ctx)
	if !errors.As(err, &refused) || refused.Refusal.Code != "generation-out-of-range" || refused.Refusal.TerminalGeneration != 3 {
		t.Fatalf("future generation: %v", err)
	}
	impostor := &Terminal{ID: "term-impostor", Generation: 2, Realization: h.real.Get(1, RuntimeNodeJS), Store: old.Store, ServerURL: h.http.URL, HTTP: old.HTTP}
	_, err = impostor.Sync(h.ctx)
	if !errors.As(err, &refused) || refused.Refusal.Code != "unknown-realization" {
		t.Fatalf("generation 2 claimed with a generation-1 realization: %v", err)
	}
	if n := opsByOutcome(t, h, "Q-MIX03")[pending.OpID]; n != "" {
		t.Fatalf("a refused sync applied an op: %s", n)
	}

	// Range restored: the kept operation goes through.
	if err := h.server.SetAcceptedGenerations(h.ctx, 1, 2); err != nil {
		t.Fatal(err)
	}
	r3, err := old.Sync(h.ctx)
	if err != nil || len(r3.Results) != 1 || r3.Results[0].Outcome != "applied" {
		t.Fatalf("after restoring the range: %v %+v", err, r3)
	}
	observed(t, "pmix03", map[string]any{"oldWriteBack": fields, "serverGuard": guard, "negativeControlKept": keptUnsafe,
		"oldTerminalRefusal": oldRefusal, "keptOpApplied": r3.Results[0].Outcome})
}
