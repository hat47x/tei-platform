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
