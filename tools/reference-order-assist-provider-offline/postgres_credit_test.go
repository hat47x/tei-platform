package orderassistoffline

import (
	"errors"
	"fmt"
	"testing"
)

// caseAtDirector prepares a valid quote and a case waiting at the director
// stage (the stage that needs the credit first).
func (h *harness) caseAtDirector(term *Terminal, id string) (caseID, digest string) {
	h.t.Helper()
	st := createAndSync(h.t, h, term, "Q-"+id, 500)
	caseID = "CASE-" + id
	if err := h.server.OpenCase(h.ctx, caseID, "Q-"+id); err != nil {
		h.t.Fatal(err)
	}
	h.approveThrough(caseID, h.policyV1, "S3-director")
	return caseID, st.Digest
}

func (h *harness) creditState(caseID string) (state string, attempts, releaseAttempts int, digest string) {
	h.t.Helper()
	if err := h.server.Pool.QueryRow(h.ctx, `SELECT state, attempts, release_attempts, quote_digest FROM oapo.credit_step WHERE case_id=$1`, caseID).
		Scan(&state, &attempts, &releaseAttempts, &digest); err != nil {
		h.t.Fatal(err)
	}
	return
}

func (h *harness) convergence(caseID string) string {
	h.t.Helper()
	c, err := Convergence(h.ctx, h.server.Pool, h.credit, caseID)
	if err != nil {
		h.t.Fatal(err)
	}
	return c
}

// P-TX-02: the provider consumes the credit and drops the connection without
// answering, twice. The saga resends once with the same key, then resolves
// the unknown outcome by inquiry.
func TestPostgresPTX02CreditUnknownOutcome(t *testing.T) {
	h := newHarness(t, ServerOptions{})
	if err := h.server.PublishPriceList(h.ctx, 1, basePrices); err != nil {
		t.Fatal(err)
	}
	a := h.terminal("term-a", 2, false)
	if _, err := a.DownloadMaster(h.ctx); err != nil {
		t.Fatal(err)
	}
	caseID, digest := h.caseAtDirector(a, "TX02")
	h.setCreditFault("reserve", 2)
	steps, err := h.saga().Run(h.ctx, caseID)
	if err != nil {
		t.Fatal(err)
	}
	var trace []string
	for _, s := range steps {
		trace = append(trace, s.Action+":"+s.State)
	}
	want := []string{"record-reserve-intent:reserve-pending", "reserve:unknown", "reserve:unknown", "inquire:consumed"}
	if fmt.Sprint(trace) != fmt.Sprint(want) {
		t.Fatalf("saga trace %v, want %v", trace, want)
	}
	key := CreditKey(caseID, digest)
	p := h.provider(key)
	state, attempts, _, creditDigest := h.creditState(caseID)
	var net int64
	if err := h.server.Pool.QueryRow(h.ctx, `SELECT amount_minor FROM oapo.credit_step WHERE case_id=$1`, caseID).Scan(&net); err != nil {
		t.Fatal(err)
	}
	if p.Reservations != 1 || p.ConsumedEvents != 1 || p.ReserveCalls != 2 || p.InquireCalls < 1 || p.UsedMinor != net ||
		state != "consumed" || attempts != 2 || creditDigest != digest {
		t.Fatalf("provider %+v, local %s attempts=%d digest=%s", p, state, attempts, creditDigest)
	}
	res, err := h.credit.Inquire(h.ctx, key)
	if err != nil || res.QuoteRef != digest {
		t.Fatalf("provider's quote reference: %+v %v", res, err)
	}
	// Running again does not call the provider again.
	if steps, err := h.saga().Run(h.ctx, caseID); err != nil || len(steps) != 0 {
		t.Fatalf("second run: %+v %v", steps, err)
	}
	if again := h.provider(key); again.ReserveCalls != 2 || again.Reservations != 1 {
		t.Fatalf("second run reached the provider: %+v", again)
	}

	// Negative control: the same resend without an idempotency key consumes twice.
	unsafe := NewCreditClient(h.creditURL)
	unsafe.UnsafeNoIdempotencyKey = true
	h.setCreditFault("reserve", 1)
	if _, err := unsafe.Reserve(h.ctx, "ignored", "C-100", 777, "negative-control"); !errors.Is(err, ErrOutcomeUnknown) {
		t.Fatalf("negative control first send: %v", err)
	}
	if _, err := unsafe.Reserve(h.ctx, "ignored", "C-100", 777, "negative-control"); err != nil {
		t.Fatal(err)
	}
	nc := h.provider("")
	if nc.Reservations != 2 {
		t.Fatalf("negative control should consume twice: %+v", nc)
	}
	observed(t, "ptx02", map[string]any{"trace": trace, "provider": p, "localAttempts": attempts, "negativeControlReservations": nc.Reservations})
}

// P-TX-03: stop the saga worker process at each point around the credit
// consumption and its compensation, and resume it. Every case must end
// all-complete or all-compensated, with at most one consumption and one
// release at the provider.
func TestPostgresPTX03CompensationConverges(t *testing.T) {
	h := newHarness(t, ServerOptions{})
	if err := h.server.PublishPriceList(h.ctx, 1, basePrices); err != nil {
		t.Fatal(err)
	}
	a := h.terminal("term-a", 2, false)
	if _, err := a.DownloadMaster(h.ctx); err != nil {
		t.Fatal(err)
	}
	type scenario struct {
		name          string
		reserveFault  string // saga process fault during the reservation
		decision      string // director: approve | reject
		releaseFault  string // saga process fault during the release
		providerDrop  string // provider drops the answer for this op
		wantConverged string
	}
	scenarios := []scenario{
		{"crash-after-reserve-answer-then-approve", "exit-after-reserve-response", "approve", "", "", "all-complete"},
		{"approve-then-crash-before-handoff", "", "approve", "", "", "all-complete"},
		{"reject-crash-before-release-send", "", "reject", "exit-before-release-send", "", "all-compensated"},
		{"reject-crash-after-release-answer", "", "reject", "exit-after-release-response", "", "all-compensated"},
		{"reject-release-answer-dropped", "", "reject", "", "release", "all-compensated"},
		{"crash-after-reserve-answer-then-reject", "exit-after-reserve-response", "reject", "exit-after-release-response", "", "all-compensated"},
	}
	results := map[string]any{}
	for i, sc := range scenarios {
		caseID, digest := h.caseAtDirector(a, fmt.Sprintf("TX03-%d", i))
		key := CreditKey(caseID, digest)
		intermediate := []string{}
		if sc.reserveFault != "" {
			code, out := h.sagaProcess(caseID, sc.reserveFault)
			if code != FaultExitCode {
				t.Fatalf("%s: worker exit %d: %s", sc.name, code, out)
			}
			if st, _, _, _ := h.creditState(caseID); st != "reserve-pending" {
				t.Fatalf("%s: after the stop the local state is %s", sc.name, st)
			}
			if p := h.provider(key); p.Reservations != 1 {
				t.Fatalf("%s: provider should have consumed once before the stop: %+v", sc.name, p)
			}
			intermediate = append(intermediate, h.convergence(caseID))
		}
		if code, out := h.sagaProcess(caseID, ""); code != 0 {
			t.Fatalf("%s: resume exit %d: %s", sc.name, code, out)
		}
		if st, _, _, _ := h.creditState(caseID); st != "consumed" {
			t.Fatalf("%s: credit %s", sc.name, st)
		}
		h.mustDecide(Decision{CaseID: caseID, Stage: "S3-director", Actor: "dir-mori", Decision: sc.decision, IfMatch: digest})
		intermediate = append(intermediate, h.convergence(caseID)) // decided, saga not yet run
		if sc.providerDrop != "" {
			h.setCreditFault(sc.providerDrop, 1)
		}
		if sc.releaseFault != "" {
			code, out := h.sagaProcess(caseID, sc.releaseFault)
			if code != FaultExitCode {
				t.Fatalf("%s: worker exit %d: %s", sc.name, code, out)
			}
			intermediate = append(intermediate, h.convergence(caseID))
		}
		if code, out := h.sagaProcess(caseID, ""); code != 0 {
			t.Fatalf("%s: final run exit %d: %s", sc.name, code, out)
		}
		got := h.convergence(caseID)
		p := h.provider(key)
		_, attempts, releaseAttempts, _ := h.creditState(caseID)
		results[sc.name] = map[string]any{"converged": got, "intermediate": intermediate, "provider": p,
			"localAttempts": attempts, "localReleaseAttempts": releaseAttempts}
		if got != sc.wantConverged {
			t.Fatalf("%s: converged to %s, want %s (intermediate %v)", sc.name, got, sc.wantConverged, intermediate)
		}
		for _, s := range intermediate {
			if s == "all-complete" || s == "all-compensated" {
				t.Fatalf("%s: an interrupted state was classified as final: %v", sc.name, intermediate)
			}
		}
		wantRelease := 0
		if sc.decision == "reject" {
			wantRelease = 1
		}
		if p.Reservations != 1 || p.ConsumedEvents != 1 || p.ReleaseCount != wantRelease || p.ReleasedEvents != wantRelease {
			t.Fatalf("%s: provider %+v", sc.name, p)
		}
		// Another run changes nothing.
		code, out := h.sagaProcess(caseID, "")
		after := h.provider(key)
		if code != 0 || after.Reservations != p.Reservations || after.ReserveCalls != p.ReserveCalls || after.ReleaseCalls != p.ReleaseCalls ||
			after.ReleaseCount != p.ReleaseCount || h.convergence(caseID) != got {
			t.Fatalf("%s: rerun moved the state: %d %s %+v", sc.name, code, out, after)
		}
	}
	var used int64
	if err := h.creditPool.QueryRow(h.ctx, `SELECT used_minor FROM oapo_credit.account WHERE customer_code='C-100'`).Scan(&used); err != nil {
		t.Fatal(err)
	}
	var stillConsumed int64
	if err := h.creditPool.QueryRow(h.ctx, `SELECT coalesce(sum(amount_minor),0) FROM oapo_credit.reservation WHERE status='consumed'`).Scan(&stillConsumed); err != nil {
		t.Fatal(err)
	}
	if used != stillConsumed {
		t.Fatalf("provider limit %d does not match consumed reservations %d", used, stillConsumed)
	}
	results["providerUsedMinor"] = used
	observed(t, "ptx03", results)
}
