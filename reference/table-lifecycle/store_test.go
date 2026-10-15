package tablelifecycle

import (
	"context"
	"errors"
	"sync"
	"testing"
)

var allowAll = AuthorizeFunc(func(context.Context, string, string) (bool, error) { return true, nil })

func fixture(t *testing.T) (*Store, Submit) {
	t.Helper()
	s, err := NewStore(20, Order{ID: "o1", State: "pending", Version: 7, Units: 3})
	if err != nil {
		t.Fatal(err)
	}
	return s, Submit{OperationID: "op1", SubjectID: "alice", OrderID: "o1", ExpectedVersion: 7, Units: 3}
}
func assertNoChange(t *testing.T, s *Store) {
	t.Helper()
	snap, _ := s.Inspect("o1")
	if snap.Order.State != "pending" || snap.Order.Version != 7 || snap.Available != 20 || snap.Reserved != 0 || len(snap.History) != 0 || len(snap.Notices) != 0 {
		t.Fatalf("unexpected changes: %+v", snap)
	}
}
func TestSubmitCommitOwnerActionsAndOutbox(t *testing.T) {
	s, c := fixture(t)
	r, err := s.Execute(context.Background(), c, allowAll)
	if err != nil || r.Outcome != Submitted || !r.Committed {
		t.Fatalf("result %+v err %v", r, err)
	}
	want := []Phase{PhaseValidate, PhaseAuthorize, PhaseRead, PhaseBefore, PhaseConsequence, PhaseCommit, PhasePostCommit}
	if len(want) != len(r.Trace) {
		t.Fatalf("trace %+v", r.Trace)
	}
	for i, p := range want {
		if r.Trace[i] != p {
			t.Fatalf("phase[%d]=%s, want %s", i, r.Trace[i], p)
		}
	}
	snap, _ := s.Inspect("o1")
	if snap.Order.State != Submitted || snap.Order.Version != 8 || snap.Available != 17 || snap.Reserved != 3 || len(snap.History) != 1 || len(snap.Notices) != 1 {
		t.Fatalf("invalid commit %+v", snap)
	}
	sends := 0
	for i := 0; i < 2; i++ {
		ok, err := s.SendPending(context.Background(), c.OperationID, func(_ context.Context, n Notice) error {
			if n.OperationID != "op1" {
				t.Errorf("notice %+v", n)
			}
			sends++
			return nil
		})
		if err != nil || ok != (i == 0) {
			t.Fatalf("dispatch %d: ok=%v err=%v", i, ok, err)
		}
	}
	if sends != 1 {
		t.Fatalf("sends %d", sends)
	}
	duplicate, err := s.Execute(context.Background(), c, allowAll)
	if err != nil || duplicate.Outcome != Submitted {
		t.Fatalf("duplicate %+v %v", duplicate, err)
	}
	again, _ := s.Inspect("o1")
	if again.Available != 17 || len(again.History) != 1 {
		t.Fatalf("double effect %+v", again)
	}
}
func TestDenialValidationAndUnknownCurrentState(t *testing.T) {
	s, c := fixture(t)
	denied := AuthorizeFunc(func(_ context.Context, subject, order string) (bool, error) {
		if subject != "alice" || order != "o1" {
			t.Fatal("wrong authorization input")
		}
		return false, nil
	})
	r, err := s.Execute(context.Background(), c, denied)
	if err != nil || r.Outcome != Denied || len(r.Trace) != 2 {
		t.Fatalf("denied %+v %v", r, err)
	}
	assertNoChange(t, s)
	c.Units = 0
	if _, err := s.Execute(context.Background(), c, allowAll); err == nil {
		t.Fatal("invalid request accepted")
	}
	assertNoChange(t, s)
	c.Units = 3
	c.ExpectedVersion = 6
	r, err = s.Execute(context.Background(), c, allowAll)
	if err != nil || r.Outcome != NotSubmitted {
		t.Fatalf("old value %+v %v", r, err)
	}
	assertNoChange(t, s)
	c.ExpectedVersion = 7
	c.OrderID = "unknown"
	r, err = s.Execute(context.Background(), c, allowAll)
	if err != nil || r.Outcome != NotSubmitted {
		t.Fatalf("unknown order %+v %v", r, err)
	}
	assertNoChange(t, s)
}
func TestRollbackAndCancelledRequest(t *testing.T) {
	s, c := fixture(t)
	s.failBeforeCommit = true
	if _, err := s.Execute(context.Background(), c, allowAll); err == nil {
		t.Fatal("expected rollback failure")
	}
	assertNoChange(t, s)
	s.failBeforeCommit = false
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := s.Execute(ctx, c, allowAll); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancel err %v", err)
	}
	assertNoChange(t, s)
	authFailure := AuthorizeFunc(func(context.Context, string, string) (bool, error) { return false, errors.New("provider down") })
	if _, err := s.Execute(context.Background(), c, authFailure); err == nil {
		t.Fatal("provider outage was treated as denial")
	}
	assertNoChange(t, s)
}
func TestLostAcknowledgementLookupAndIdentityCollision(t *testing.T) {
	s, c := fixture(t)
	s.loseAcknowledgement = true
	result, err := s.Execute(context.Background(), c, allowAll)
	if err != nil || result.Outcome != Unknown || result.Committed {
		t.Fatalf("unknown %+v %v", result, err)
	}
	snap, _ := s.Inspect("o1")
	if snap.Available != 17 {
		t.Fatal("commit should survive acknowledgement loss")
	}
	got, found, err := s.Lookup(context.Background(), c, allowAll)
	if err != nil || !found || got.Outcome != Submitted || !got.Committed {
		t.Fatalf("lookup %+v %v %v", got, found, err)
	}
	altered := c
	altered.Units++
	if _, err := s.Execute(context.Background(), altered, allowAll); err == nil {
		t.Fatal("operation identity collision accepted")
	}
	denied := AuthorizeFunc(func(context.Context, string, string) (bool, error) { return false, nil })
	_, found, err = s.Lookup(context.Background(), c, denied)
	if err != nil || found {
		t.Fatalf("receipt leaked: %v %v", found, err)
	}
	retry, err := s.Execute(context.Background(), c, allowAll)
	if err != nil || retry.Outcome != Submitted || !retry.Committed {
		t.Fatalf("retry %+v %v", retry, err)
	}
	snap, _ = s.Inspect("o1")
	if len(snap.History) != 1 {
		t.Fatal("duplicate consequence")
	}
}
func TestConcurrentStaleWriteAtMostOneSuccess(t *testing.T) {
	s, c := fixture(t)
	var wg sync.WaitGroup
	outcomes := make(chan string, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			x := c
			if i == 1 {
				x.OperationID = "op2"
			}
			r, e := s.Execute(context.Background(), x, allowAll)
			if e != nil {
				outcomes <- "error"
			} else {
				outcomes <- r.Outcome
			}
		}(i)
	}
	wg.Wait()
	close(outcomes)
	success, conflict := 0, 0
	for x := range outcomes {
		if x == Submitted {
			success++
		} else if x == NotSubmitted {
			conflict++
		} else {
			t.Fatalf("outcome %s", x)
		}
	}
	if success != 1 || conflict != 1 {
		t.Fatalf("success %d conflict %d", success, conflict)
	}
	snap, _ := s.Inspect("o1")
	if snap.Available != 17 || len(snap.History) != 1 || len(snap.Notices) != 1 {
		t.Fatalf("effects %+v", snap)
	}
}
func TestSenderFailureNotConfusedWithUnsent(t *testing.T) {
	s, c := fixture(t)
	_, err := s.Execute(context.Background(), c, allowAll)
	if err != nil {
		t.Fatal(err)
	}
	sent := 0
	_, err = s.SendPending(context.Background(), c.OperationID, func(context.Context, Notice) error { sent++; return errors.New("ambiguous remote timeout") })
	if err == nil {
		t.Fatal("timeout not surfaced")
	}
	dispatched, err := s.SendPending(context.Background(), c.OperationID, func(context.Context, Notice) error { sent++; return nil })
	if err != nil || !dispatched || sent != 2 {
		t.Fatalf("recovery %+v %v sent=%d", dispatched, err, sent)
	}
	// Retrying transport is safe only when receiver deduplicates by OperationID.
}

func TestCancellationDeletesReservationAtomically(t *testing.T) {
	s, c := fixture(t)
	if _, err := s.Execute(context.Background(), c, allowAll); err != nil {
		t.Fatal(err)
	}
	cancel := Cancel{OperationID: "op-cancel", SubjectID: "alice", OrderID: "o1", ExpectedVersion: 8}
	s.failBeforeCommit = true
	if _, err := s.ExecuteCancel(context.Background(), cancel, allowAll); err == nil {
		t.Fatal("cancel rollback missing")
	}
	snap, _ := s.Inspect("o1")
	if snap.Available != 17 || snap.Reserved != 3 || snap.Order.State != Submitted || len(snap.History) != 1 || len(snap.Notices) != 1 {
		t.Fatalf("rollback %+v", snap)
	}
	s.failBeforeCommit = false
	result, err := s.ExecuteCancel(context.Background(), cancel, allowAll)
	if err != nil || result.Outcome != "cancelled" || !result.Committed {
		t.Fatalf("cancel %+v %v", result, err)
	}
	snap, _ = s.Inspect("o1")
	if snap.Order.State != "cancelled" || snap.Order.Version != 9 || snap.Available != 20 || snap.Reserved != 0 || len(snap.History) != 2 || len(snap.Notices) != 2 {
		t.Fatalf("cancel effects %+v", snap)
	}
	result, err = s.ExecuteCancel(context.Background(), cancel, allowAll)
	if err != nil || result.Outcome != "cancelled" {
		t.Fatalf("retry cancel %+v %v", result, err)
	}
	snap, _ = s.Inspect("o1")
	if snap.Available != 20 || len(snap.History) != 2 {
		t.Fatalf("double cancel %+v", snap)
	}
	if _, err := s.ExecuteCancel(context.Background(), Cancel{OperationID: "op1", SubjectID: "alice", OrderID: "o1", ExpectedVersion: 9}, allowAll); err == nil {
		t.Fatal("cross-action operation ID reuse")
	}
}

func TestSenderCannotDuplicateWhileInFlightAndMayReenter(t *testing.T) {
	s, c := fixture(t)
	if _, err := s.Execute(context.Background(), c, allowAll); err != nil {
		t.Fatal(err)
	}
	entered, release := make(chan struct{}), make(chan struct{})
	done := make(chan error, 1)
	go func() {
		_, err := s.SendPending(context.Background(), c.OperationID, func(context.Context, Notice) error { close(entered); <-release; return nil })
		done <- err
	}()
	<-entered
	if _, ok := s.Inspect("o1"); !ok {
		t.Fatal("sender callback blocked store lock")
	}
	if _, err := s.SendPending(context.Background(), c.OperationID, func(context.Context, Notice) error { t.Fatal("duplicate sender entered"); return nil }); err == nil {
		t.Fatal("reentrant delivery accepted")
	}
	close(release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	didSend, err := s.SendPending(context.Background(), c.OperationID, func(context.Context, Notice) error { t.Fatal("already delivered"); return nil })
	if err != nil || didSend {
		t.Fatalf("second delivery %v %v", didSend, err)
	}
}

func TestCancelUnknownOutcomeLookupRecovery(t *testing.T) {
	ctx := context.Background()
	s, submit := fixture(t)
	if result, err := s.Execute(ctx, submit, allowAll); err != nil || result.Outcome != Submitted {
		t.Fatalf("setup: result=%+v err=%v", result, err)
	}
	cancel := Cancel{OperationID: "cancel-1", SubjectID: "alice", OrderID: "o1", ExpectedVersion: 8}

	// Before commit, no receipt is visible; absence is not a conclusion about
	// external systems, only this synthetic Store.
	if result, found, err := s.LookupCancel(ctx, cancel, allowAll); err != nil || found || result.Committed {
		t.Fatalf("unexpected uncommitted receipt: %+v found=%t err=%v", result, found, err)
	}

	s.loseAcknowledgement = true
	result, err := s.ExecuteCancel(ctx, cancel, allowAll)
	if err != nil || result.Outcome != Unknown || result.Committed {
		t.Fatalf("lost acknowledgement: %+v err=%v", result, err)
	}
	snap, _ := s.Inspect("o1")
	if snap.Order.State != "cancelled" || snap.Available != 20 || snap.Reserved != 0 || len(snap.History) != 2 || len(snap.Notices) != 2 {
		t.Fatalf("cancel was not atomically committed: %+v", snap)
	}

	receipt, found, err := s.LookupCancel(ctx, cancel, allowAll)
	if err != nil || !found || receipt.Outcome != "cancelled" || !receipt.Committed {
		t.Fatalf("cancel receipt: %+v found=%t err=%v", receipt, found, err)
	}
	// A caller may inspect their own receipt but cannot mutate the stored
	// receipt's trace through the returned slice.
	if len(receipt.Trace) == 0 {
		t.Fatal("receipt trace missing")
	}
	receipt.Trace[0] = "mutated-by-caller"
	reloaded, found, err := s.LookupCancel(ctx, cancel, allowAll)
	if err != nil || !found || reloaded.Trace[0] != PhaseValidate {
		t.Fatalf("receipt unexpectedly mutated: %+v %v %v", reloaded, found, err)
	}
	retry, err := s.ExecuteCancel(ctx, cancel, allowAll)
	if err != nil || retry.Outcome != "cancelled" || !retry.Committed {
		t.Fatalf("retry should recover the committed result: %+v %v", retry, err)
	}
	snap, _ = s.Inspect("o1")
	if len(snap.History) != 2 || len(snap.Notices) != 2 {
		t.Fatalf("duplicated cancel effects: %+v", snap)
	}
}

func TestLookupCancelAuthorizationIdentityAndFailure(t *testing.T) {
	ctx := context.Background()
	s, submit := fixture(t)
	if _, err := s.Execute(ctx, submit, allowAll); err != nil {
		t.Fatal(err)
	}
	cancel := Cancel{OperationID: "cancel-1", SubjectID: "alice", OrderID: "o1", ExpectedVersion: 8}
	if _, err := s.ExecuteCancel(ctx, cancel, allowAll); err != nil {
		t.Fatal(err)
	}
	revoked := AuthorizeFunc(func(context.Context, string, string) (bool, error) { return false, nil })
	if receipt, found, err := s.LookupCancel(ctx, cancel, revoked); err != nil || found || receipt.Outcome != Denied {
		t.Fatalf("revoked caller saw receipt: %+v %t %v", receipt, found, err)
	}
	if _, found, err := s.LookupCancel(ctx, cancel, nil); err == nil || found {
		t.Fatalf("missing authorization accepted: %t %v", found, err)
	}
	outage := AuthorizeFunc(func(context.Context, string, string) (bool, error) {
		return false, errors.New("authorization provider failure")
	})
	if result, found, err := s.LookupCancel(ctx, cancel, outage); err == nil || found || result.Outcome == Denied {
		t.Fatalf("provider failure converted to denial: %+v %t %v", result, found, err)
	}
	for _, altered := range []Cancel{
		{OperationID: cancel.OperationID, SubjectID: "bob", OrderID: cancel.OrderID, ExpectedVersion: cancel.ExpectedVersion},
		{OperationID: cancel.OperationID, SubjectID: cancel.SubjectID, OrderID: "o2", ExpectedVersion: cancel.ExpectedVersion},
		{OperationID: cancel.OperationID, SubjectID: cancel.SubjectID, OrderID: cancel.OrderID, ExpectedVersion: 9},
		{OperationID: submit.OperationID, SubjectID: cancel.SubjectID, OrderID: cancel.OrderID, ExpectedVersion: cancel.ExpectedVersion},
	} {
		if result, found, err := s.LookupCancel(ctx, altered, allowAll); err == nil || found || result.Committed {
			t.Fatalf("mismatched identity exposed a receipt: %+v %t %v", result, found, err)
		}
	}
	for _, bad := range []Cancel{
		{SubjectID: "alice", OrderID: "o1"},
		{OperationID: "x", OrderID: "o1"},
		{OperationID: "x", SubjectID: "alice"},
	} {
		if _, found, err := s.LookupCancel(ctx, bad, allowAll); err == nil || found {
			t.Fatalf("invalid cancel identity accepted: %+v", bad)
		}
	}
	aborted, stop := context.WithCancel(ctx)
	stop()
	if _, found, err := s.LookupCancel(aborted, cancel, allowAll); !errors.Is(err, context.Canceled) || found {
		t.Fatalf("cancelled context exposed receipt: %t %v", found, err)
	}
}

func TestDeletePendingAtomicAndIdempotent(t *testing.T) {
	ctx := context.Background()
	s, submit := fixture(t)
	req := DeletePendingRequest{OperationID: "delete-1", SubjectID: submit.SubjectID, OrderID: submit.OrderID, ExpectedVersion: submit.ExpectedVersion}

	before, found, err := s.LookupDelete(ctx, req, allowAll)
	if err != nil || found || before.Committed {
		t.Fatalf("unexpected pre-commit receipt: %+v found=%t err=%v", before, found, err)
	}
	if dispatched, err := s.SendPending(ctx, req.OperationID, func(context.Context, Notice) error { t.Fatal("sent before commit"); return nil }); err != nil || dispatched {
		t.Fatalf("notice before commit: dispatched=%t err=%v", dispatched, err)
	}

	result, err := s.DeletePending(ctx, req, allowAll)
	if err != nil || result.Outcome != Deleted || !result.Committed {
		t.Fatalf("delete: %+v err=%v", result, err)
	}
	if _, exists := s.Inspect(req.OrderID); exists {
		t.Fatal("deleted order remains in store")
	}
	s.mu.Lock()
	available := s.state.available
	history := append([]string(nil), s.state.history[req.OrderID]...)
	notice, noticeExists := s.state.pending[req.OperationID]
	_, reservationExists := s.state.reservations[req.OrderID]
	s.mu.Unlock()
	if available != 20 || reservationExists || len(history) != 1 || history[0] != "deleted:"+req.OperationID ||
		!noticeExists || notice.Kind != "order.deleted" {
		t.Fatalf("atomic delete effects available=%d reservation=%t history=%v notice=%+v exists=%t",
			available, reservationExists, history, notice, noticeExists)
	}
	receipt, found, err := s.LookupDelete(ctx, req, allowAll)
	if err != nil || !found || receipt.Outcome != Deleted || !receipt.Committed {
		t.Fatalf("receipt: %+v found=%t err=%v", receipt, found, err)
	}
	duplicate, err := s.DeletePending(ctx, req, allowAll)
	if err != nil || duplicate.Outcome != Deleted || !duplicate.Committed {
		t.Fatalf("idempotent retry: %+v err=%v", duplicate, err)
	}
	s.mu.Lock()
	historyCount, noticeCount, inventory := len(s.state.history[req.OrderID]), len(s.state.pending), s.state.available
	s.mu.Unlock()
	if historyCount != 1 || noticeCount != 1 || inventory != 20 {
		t.Fatalf("duplicate consequences history=%d notices=%d inventory=%d", historyCount, noticeCount, inventory)
	}
	sends := 0
	for i := 0; i < 2; i++ {
		ok, err := s.SendPending(ctx, req.OperationID, func(_ context.Context, n Notice) error {
			if n.Kind != "order.deleted" {
				t.Errorf("wrong notice %+v", n)
			}
			sends++
			return nil
		})
		if err != nil || ok != (i == 0) {
			t.Fatalf("send %d: %t %v", i, ok, err)
		}
	}
	if sends != 1 {
		t.Fatalf("sent %d times", sends)
	}
}

func TestDeletePendingFailsClosedAndRollsBack(t *testing.T) {
	ctx := context.Background()
	s, submit := fixture(t)
	req := DeletePendingRequest{OperationID: "delete-1", SubjectID: submit.SubjectID, OrderID: submit.OrderID, ExpectedVersion: submit.ExpectedVersion}

	checkUnchanged := func() {
		t.Helper()
		assertNoChange(t, s)
		s.mu.Lock()
		receipts, pending := len(s.state.receipts), len(s.state.pending)
		s.mu.Unlock()
		if receipts != 0 || pending != 0 {
			t.Fatalf("uncommitted delete left receipts=%d pending=%d", receipts, pending)
		}
	}
	s.failBeforeCommit = true
	if _, err := s.DeletePending(ctx, req, allowAll); err == nil {
		t.Fatal("delete succeeded despite transaction failure")
	}
	checkUnchanged()
	s.failBeforeCommit = false

	stale := req
	stale.ExpectedVersion--
	if result, err := s.DeletePending(ctx, stale, allowAll); err != nil || result.Outcome != NotDeleted {
		t.Fatalf("stale old version: %+v %v", result, err)
	}
	missing := req
	missing.OrderID = "absent"
	if result, err := s.DeletePending(ctx, missing, allowAll); err != nil || result.Outcome != NotDeleted {
		t.Fatalf("missing order: %+v %v", result, err)
	}
	checkUnchanged()

	denied := AuthorizeFunc(func(context.Context, string, string) (bool, error) { return false, nil })
	if result, err := s.DeletePending(ctx, req, denied); err != nil || result.Outcome != Denied {
		t.Fatalf("denial: %+v %v", result, err)
	}
	failedAuth := AuthorizeFunc(func(context.Context, string, string) (bool, error) {
		return false, errors.New("authority unreachable")
	})
	if result, err := s.DeletePending(ctx, req, failedAuth); err == nil || result.Outcome == Denied {
		t.Fatalf("authority outage changed to denial: %+v %v", result, err)
	}
	aborted, stop := context.WithCancel(ctx)
	stop()
	if _, err := s.DeletePending(aborted, req, allowAll); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled context: %v", err)
	}
	checkUnchanged()

	// A submitted order owns a reservation and can only be cancelled by the
	// separate owner Action; deleting it must never leak inventory or history.
	if _, err := s.Execute(ctx, submit, allowAll); err != nil {
		t.Fatal(err)
	}
	if result, err := s.DeletePending(ctx, req, allowAll); err != nil || result.Outcome != NotDeleted {
		t.Fatalf("submitted order deleted: %+v %v", result, err)
	}
	snap, exists := s.Inspect(req.OrderID)
	if !exists || snap.Order.State != Submitted || snap.Reserved != 3 || snap.Available != 17 || len(snap.History) != 1 {
		t.Fatalf("submitted order effects changed: %+v exists=%t", snap, exists)
	}
}

func TestDeletePendingUncertainOutcomeReceiptAndIdentity(t *testing.T) {
	ctx := context.Background()
	s, submit := fixture(t)
	req := DeletePendingRequest{OperationID: "delete-1", SubjectID: submit.SubjectID, OrderID: submit.OrderID, ExpectedVersion: submit.ExpectedVersion}
	s.loseAcknowledgement = true
	result, err := s.DeletePending(ctx, req, allowAll)
	if err != nil || result.Outcome != Unknown || result.Committed {
		t.Fatalf("lost acknowledgement: %+v %v", result, err)
	}
	if _, exists := s.Inspect(req.OrderID); exists {
		t.Fatal("lost acknowledgement rolled back a committed delete")
	}
	recovered, found, err := s.LookupDelete(ctx, req, allowAll)
	if err != nil || !found || recovered.Outcome != Deleted || !recovered.Committed {
		t.Fatalf("lookup: %+v %t %v", recovered, found, err)
	}
	mutated := recovered
	mutated.Trace[0] = "tampered"
	recovered, found, err = s.LookupDelete(ctx, req, allowAll)
	if err != nil || !found || recovered.Trace[0] != PhaseValidate {
		t.Fatalf("mutable receipt escaped: %+v %t %v", recovered, found, err)
	}
	revoked := AuthorizeFunc(func(context.Context, string, string) (bool, error) { return false, nil })
	if value, found, err := s.LookupDelete(ctx, req, revoked); err != nil || found || value.Outcome != Denied {
		t.Fatalf("unauthorized receipt: %+v %t %v", value, found, err)
	}
	down := AuthorizeFunc(func(context.Context, string, string) (bool, error) {
		return false, errors.New("authority failure")
	})
	if _, found, err := s.LookupDelete(ctx, req, down); err == nil || found {
		t.Fatalf("provider failure leaked receipt: %t %v", found, err)
	}
	for _, altered := range []DeletePendingRequest{
		{OperationID: req.OperationID, SubjectID: "bob", OrderID: req.OrderID, ExpectedVersion: req.ExpectedVersion},
		{OperationID: req.OperationID, SubjectID: req.SubjectID, OrderID: "other", ExpectedVersion: req.ExpectedVersion},
		{OperationID: req.OperationID, SubjectID: req.SubjectID, OrderID: req.OrderID, ExpectedVersion: req.ExpectedVersion + 1},
	} {
		if _, found, err := s.LookupDelete(ctx, altered, allowAll); err == nil || found {
			t.Fatalf("mismatched identity accepted: %+v %t %v", altered, found, err)
		}
		if _, err := s.DeletePending(ctx, altered, allowAll); err == nil {
			t.Fatalf("operation identity reused for another request: %+v", altered)
		}
	}
	retry, err := s.DeletePending(ctx, req, allowAll)
	if err != nil || retry.Outcome != Deleted {
		t.Fatalf("retry lost committed identity: %+v %v", retry, err)
	}
	s.mu.Lock()
	count := len(s.state.history[req.OrderID])
	s.mu.Unlock()
	if count != 1 {
		t.Fatalf("duplicate deletion history count=%d", count)
	}
}

func TestDeletePendingRejectsPendingOrderWithReservation(t *testing.T) {
	// This deliberately inconsistent owner state is an independent negative
	// fixture: the row still says pending, but inventory already has a
	// reservation. Neither the order nor owner-side data may be deleted.
	s, submit := fixture(t)
	s.mu.Lock()
	s.state.available = 17
	s.state.reservations[submit.OrderID] = submit.Units
	s.mu.Unlock()

	request := DeletePendingRequest{
		OperationID:     "delete-reserved",
		SubjectID:       submit.SubjectID,
		OrderID:         submit.OrderID,
		ExpectedVersion: submit.ExpectedVersion,
	}
	result, err := s.DeletePending(context.Background(), request, allowAll)
	if err != nil || result.Outcome != NotDeleted || result.Committed {
		t.Fatalf("reserved pending row deleted: result=%+v err=%v", result, err)
	}
	expected := []Phase{PhaseValidate, PhaseAuthorize, PhaseRead}
	if len(result.Trace) != len(expected) {
		t.Fatalf("consequence phases ran for refused delete: %v", result.Trace)
	}
	for i, phase := range expected {
		if result.Trace[i] != phase {
			t.Fatalf("trace[%d]=%s, want %s", i, result.Trace[i], phase)
		}
	}
	snap, exists := s.Inspect(submit.OrderID)
	if !exists || snap.Order.State != "pending" || snap.Order.Version != 7 ||
		snap.Available != 17 || snap.Reserved != 3 || len(snap.History) != 0 || len(snap.Notices) != 0 {
		t.Fatalf("refused delete mutated owner state: %+v exists=%t", snap, exists)
	}
	s.mu.Lock()
	receipts, notices := len(s.state.receipts), len(s.state.pending)
	s.mu.Unlock()
	if receipts != 0 || notices != 0 {
		t.Fatalf("refused delete committed side effects: receipts=%d notices=%d", receipts, notices)
	}
	if receipt, found, err := s.LookupDelete(context.Background(), request, allowAll); err != nil || found || receipt.Committed {
		t.Fatalf("refused delete created a receipt: %+v found=%t err=%v", receipt, found, err)
	}
	if dispatched, err := s.SendPending(context.Background(), request.OperationID, func(context.Context, Notice) error {
		t.Fatal("external delivery ran for refused deletion")
		return nil
	}); err != nil || dispatched {
		t.Fatalf("refused delete published a notice: dispatched=%t err=%v", dispatched, err)
	}
}

func TestConcurrentDeletePendingAtMostOneCommit(t *testing.T) {
	// Two independently authorized deletes race against the same pending row
	// and old version. Only one may leave a tombstone, receipt and outbox item.
	s, submit := fixture(t)
	type attempt struct {
		request DeletePendingRequest
		result  Result
		err     error
	}
	start := make(chan struct{})
	done := make(chan attempt, 2)
	for _, id := range []string{"delete-a", "delete-b"} {
		request := DeletePendingRequest{
			OperationID:     id,
			SubjectID:       submit.SubjectID,
			OrderID:         submit.OrderID,
			ExpectedVersion: submit.ExpectedVersion,
		}
		go func(c DeletePendingRequest) {
			<-start
			result, err := s.DeletePending(context.Background(), c, allowAll)
			done <- attempt{request: c, result: result, err: err}
		}(request)
	}
	close(start)

	var committed, rejected DeletePendingRequest
	successes, conflicts := 0, 0
	for i := 0; i < 2; i++ {
		a := <-done
		if a.err != nil {
			t.Fatalf("concurrent delete %s: %v", a.request.OperationID, a.err)
		}
		switch a.result.Outcome {
		case Deleted:
			if !a.result.Committed {
				t.Fatalf("successful deletion has no commit: %+v", a.result)
			}
			committed = a.request
			successes++
		case NotDeleted:
			if a.result.Committed || len(a.result.Trace) != 3 || a.result.Trace[2] != PhaseRead {
				t.Fatalf("refused deletion committed or reached consequence: %+v", a.result)
			}
			rejected = a.request
			conflicts++
		default:
			t.Fatalf("unexpected delete outcome: %+v", a.result)
		}
	}
	if successes != 1 || conflicts != 1 {
		t.Fatalf("delete commits=%d conflicts=%d", successes, conflicts)
	}
	if _, exists := s.Inspect(submit.OrderID); exists {
		t.Fatal("successfully deleted row remains readable")
	}

	s.mu.Lock()
	available := s.state.available
	history := append([]string(nil), s.state.history[submit.OrderID]...)
	receipts, outbox := len(s.state.receipts), len(s.state.pending)
	_, reserved := s.state.reservations[submit.OrderID]
	winnerReceipt, hasReceipt := s.state.receipts[committed.OperationID]
	winnerNotice, hasNotice := s.state.pending[committed.OperationID]
	_, losingReceipt := s.state.receipts[rejected.OperationID]
	_, losingNotice := s.state.pending[rejected.OperationID]
	s.mu.Unlock()
	if available != 20 || reserved || len(history) != 1 || history[0] != "deleted:"+committed.OperationID {
		t.Fatalf("duplicated owner effect: available=%d reserved=%t history=%v", available, reserved, history)
	}
	if receipts != 1 || outbox != 1 || !hasReceipt || !hasNotice || losingReceipt || losingNotice ||
		winnerReceipt.Request != committed || winnerReceipt.Result.Outcome != Deleted ||
		winnerNotice.Kind != "order.deleted" || winnerNotice.OperationID != committed.OperationID {
		t.Fatalf("invalid commit boundary: receipts=%d outbox=%d winner=%+v notice=%+v losing=%t/%t",
			receipts, outbox, winnerReceipt, winnerNotice, losingReceipt, losingNotice)
	}
	if result, found, err := s.LookupDelete(context.Background(), rejected, allowAll); err != nil || found || result.Committed {
		t.Fatalf("rejected concurrent operation has a receipt: %+v found=%t err=%v", result, found, err)
	}
}

func TestCancelStaleVersionRetryWithSameOperationID(t *testing.T) {
	ctx := context.Background()
	s, submit := fixture(t)
	if submitted, err := s.Execute(ctx, submit, allowAll); err != nil || submitted.Outcome != Submitted {
		t.Fatalf("setup submit: %+v err=%v", submitted, err)
	}
	stale := Cancel{
		OperationID:     "cancel-after-refresh",
		SubjectID:       submit.SubjectID,
		OrderID:         submit.OrderID,
		ExpectedVersion: submit.ExpectedVersion,
	}
	rejected, err := s.ExecuteCancel(ctx, stale, allowAll)
	if err != nil || rejected.Outcome != NotSubmitted || rejected.Committed {
		t.Fatalf("stale cancellation accepted: %+v err=%v", rejected, err)
	}
	want := []Phase{PhaseValidate, PhaseAuthorize, PhaseRead}
	if len(rejected.Trace) != len(want) {
		t.Fatalf("stale cancellation reached mutation phase: %v", rejected.Trace)
	}
	for i, phase := range want {
		if rejected.Trace[i] != phase {
			t.Fatalf("stale cancellation phase %d: got %s want %s", i, rejected.Trace[i], phase)
		}
	}
	snap, exists := s.Inspect(submit.OrderID)
	if !exists || snap.Order.State != Submitted || snap.Order.Version != submit.ExpectedVersion+1 ||
		snap.Available != 17 || snap.Reserved != submit.Units || len(snap.History) != 1 || len(snap.Notices) != 1 {
		t.Fatalf("stale cancellation changed owner state: %+v exists=%t", snap, exists)
	}
	s.mu.Lock()
	receipts, pending := len(s.state.receipts), len(s.state.pending)
	_, staleReceipt := s.state.receipts[stale.OperationID]
	_, staleNotice := s.state.pending[stale.OperationID]
	s.mu.Unlock()
	if receipts != 1 || pending != 1 || staleReceipt || staleNotice {
		t.Fatalf("stale cancellation left effects: receipts=%d pending=%d stale=%t/%t",
			receipts, pending, staleReceipt, staleNotice)
	}
	if result, found, err := s.LookupCancel(ctx, stale, allowAll); err != nil || found || result.Committed {
		t.Fatalf("stale cancellation acquired receipt: %+v found=%t err=%v", result, found, err)
	}

	// A failed optimistic version check must not consume the idempotency key:
	// retrying after refreshing the expected version can commit exactly once.
	refreshed := stale
	refreshed.ExpectedVersion = submit.ExpectedVersion + 1
	committed, err := s.ExecuteCancel(ctx, refreshed, allowAll)
	if err != nil || committed.Outcome != "cancelled" || !committed.Committed {
		t.Fatalf("refreshed cancellation failed: %+v err=%v", committed, err)
	}
	retried, err := s.ExecuteCancel(ctx, refreshed, allowAll)
	if err != nil || retried.Outcome != "cancelled" || !retried.Committed {
		t.Fatalf("idempotent refreshed retry: %+v err=%v", retried, err)
	}
	snap, exists = s.Inspect(submit.OrderID)
	if !exists || snap.Order.State != "cancelled" || snap.Order.Version != submit.ExpectedVersion+2 ||
		snap.Available != 20 || snap.Reserved != 0 || len(snap.History) != 2 || len(snap.Notices) != 2 {
		t.Fatalf("refreshed retry duplicated owner effects: %+v exists=%t", snap, exists)
	}
	s.mu.Lock()
	receipts, pending = len(s.state.receipts), len(s.state.pending)
	s.mu.Unlock()
	if receipts != 2 || pending != 2 {
		t.Fatalf("refreshed retry duplicated receipts or outbox: receipts=%d pending=%d", receipts, pending)
	}
	if result, found, err := s.LookupCancel(ctx, refreshed, allowAll); err != nil || !found ||
		result.Outcome != "cancelled" || !result.Committed {
		t.Fatalf("refreshed receipt not recoverable: %+v found=%t err=%v", result, found, err)
	}
}

func TestConcurrentCancelAtMostOneRefund(t *testing.T) {
	ctx := context.Background()
	s, submit := fixture(t)
	if result, err := s.Execute(ctx, submit, allowAll); err != nil || result.Outcome != Submitted || !result.Committed {
		t.Fatalf("submit setup: %+v err=%v", result, err)
	}

	// Distinct idempotency identities race against the same old row version.
	// Cancellation must refund the reservation and publish at most once.
	type attempt struct {
		request Cancel
		result  Result
		err     error
	}
	start := make(chan struct{})
	done := make(chan attempt, 2)
	for _, id := range []string{"cancel-concurrent-a", "cancel-concurrent-b"} {
		request := Cancel{
			OperationID:     id,
			SubjectID:       submit.SubjectID,
			OrderID:         submit.OrderID,
			ExpectedVersion: submit.ExpectedVersion + 1,
		}
		go func(c Cancel) {
			<-start
			result, err := s.ExecuteCancel(ctx, c, allowAll)
			done <- attempt{request: c, result: result, err: err}
		}(request)
	}
	close(start)

	var winner, loser Cancel
	successes, conflicts := 0, 0
	for i := 0; i < 2; i++ {
		a := <-done
		if a.err != nil {
			t.Fatalf("concurrent cancellation %s: %v", a.request.OperationID, a.err)
		}
		switch a.result.Outcome {
		case "cancelled":
			if !a.result.Committed {
				t.Fatalf("cancelled without commit: %+v", a.result)
			}
			winner = a.request
			successes++
		case NotSubmitted:
			if a.result.Committed || len(a.result.Trace) != 3 || a.result.Trace[2] != PhaseRead {
				t.Fatalf("losing cancellation reached mutation: %+v", a.result)
			}
			loser = a.request
			conflicts++
		default:
			t.Fatalf("unexpected cancellation result: %+v", a.result)
		}
	}
	if successes != 1 || conflicts != 1 {
		t.Fatalf("cancellation successes=%d conflicts=%d", successes, conflicts)
	}

	snap, exists := s.Inspect(submit.OrderID)
	if !exists || snap.Order.State != "cancelled" || snap.Order.Version != submit.ExpectedVersion+2 ||
		snap.Available != 20 || snap.Reserved != 0 || len(snap.History) != 2 || len(snap.Notices) != 2 ||
		snap.History[1] != "cancelled:"+winner.OperationID {
		t.Fatalf("duplicate refund or history: %+v exists=%t", snap, exists)
	}
	s.mu.Lock()
	receipts, outbox := len(s.state.receipts), len(s.state.pending)
	winningReceipt, winnerReceiptFound := s.state.receipts[winner.OperationID]
	winningNotice, winnerNoticeFound := s.state.pending[winner.OperationID]
	_, loserReceiptFound := s.state.receipts[loser.OperationID]
	_, loserNoticeFound := s.state.pending[loser.OperationID]
	s.mu.Unlock()
	if receipts != 2 || outbox != 2 || !winnerReceiptFound || !winnerNoticeFound ||
		loserReceiptFound || loserNoticeFound ||
		winningReceipt.Request != winner || winningReceipt.Result.Outcome != "cancelled" ||
		winningNotice.Kind != "order.cancelled" || winningNotice.OperationID != winner.OperationID {
		t.Fatalf("cancel effect boundary: receipts=%d outbox=%d winner=%+v notice=%+v loser=%t/%t",
			receipts, outbox, winningReceipt, winningNotice, loserReceiptFound, loserNoticeFound)
	}
	if result, found, err := s.LookupCancel(ctx, loser, allowAll); err != nil || found || result.Committed {
		t.Fatalf("losing cancellation has receipt: %+v found=%t err=%v", result, found, err)
	}
	if result, found, err := s.LookupCancel(ctx, winner, allowAll); err != nil || !found ||
		result.Outcome != "cancelled" || !result.Committed {
		t.Fatalf("winning cancellation receipt missing: %+v found=%t err=%v", result, found, err)
	}
}

func TestCancelRejectsMismatchedOwnerReservation(t *testing.T) {
	// A submitted row's Units and the inventory owner's reservation must
	// agree before cancellation refunds the held quantity. A positive
	// reservation alone is insufficient to authorize an owner-side refund.
	for _, tc := range []struct {
		name        string
		reservation int
		allowCancel bool
	}{
		{name: "under-reserved", reservation: 2},
		{name: "matched", reservation: 3, allowCancel: true},
		{name: "over-reserved", reservation: 4},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			s, submit := fixture(t)
			if result, err := s.Execute(ctx, submit, allowAll); err != nil || result.Outcome != Submitted || !result.Committed {
				t.Fatalf("submit setup: %+v err=%v", result, err)
			}

			// Reconstruct an owner reservation that may have drifted
			// independently of the submitted order (synthetic fault fixture).
			s.mu.Lock()
			s.state.reservations[submit.OrderID] = tc.reservation
			s.state.available = 20 - tc.reservation
			s.mu.Unlock()
			cancel := Cancel{
				OperationID:     "cancel-reservation-check",
				SubjectID:       submit.SubjectID,
				OrderID:         submit.OrderID,
				ExpectedVersion: submit.ExpectedVersion + 1,
			}
			result, err := s.ExecuteCancel(ctx, cancel, allowAll)
			if err != nil {
				t.Fatalf("cancel result=%+v err=%v", result, err)
			}
			snap, exists := s.Inspect(submit.OrderID)
			if !exists {
				t.Fatal("cancellation unexpectedly removed the order")
			}
			s.mu.Lock()
			receipts, outbox := len(s.state.receipts), len(s.state.pending)
			_, cancelReceipt := s.state.receipts[cancel.OperationID]
			_, cancelNotice := s.state.pending[cancel.OperationID]
			s.mu.Unlock()
			if tc.allowCancel {
				if result.Outcome != "cancelled" || !result.Committed ||
					snap.Order.State != "cancelled" || snap.Order.Version != 9 ||
					snap.Available != 20 || snap.Reserved != 0 ||
					len(snap.History) != 2 || len(snap.Notices) != 2 ||
					receipts != 2 || outbox != 2 || !cancelReceipt || !cancelNotice {
					t.Fatalf("matching reservation not committed exactly once: result=%+v snap=%+v receipts=%d outbox=%d",
						result, snap, receipts, outbox)
				}
				return
			}
			if result.Outcome != NotSubmitted || result.Committed ||
				len(result.Trace) != 3 || result.Trace[2] != PhaseRead ||
				snap.Order.State != Submitted || snap.Order.Version != 8 ||
				snap.Available != 20-tc.reservation || snap.Reserved != tc.reservation ||
				len(snap.History) != 1 || len(snap.Notices) != 1 ||
				receipts != 1 || outbox != 1 || cancelReceipt || cancelNotice {
				t.Fatalf("mismatched reservation mutated owner: result=%+v snap=%+v receipts=%d outbox=%d receipt=%t notice=%t",
					result, snap, receipts, outbox, cancelReceipt, cancelNotice)
			}
			if receipt, found, lookupErr := s.LookupCancel(ctx, cancel, allowAll); lookupErr != nil || found || receipt.Committed {
				t.Fatalf("rejected mismatch left a cancel receipt: %+v found=%t err=%v", receipt, found, lookupErr)
			}
		})
	}
}
