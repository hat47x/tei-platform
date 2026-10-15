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
