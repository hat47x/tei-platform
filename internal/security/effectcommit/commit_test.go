package effectcommit

import (
	"context"
	"errors"
	"math"
	"sync"
	"testing"

	"tei/internal/security/legacyauthorization"
)

// 変更範囲の試験用Store。BeginからCommit/Rollbackまで同一mutexを保持する。
// 実運用Providerのトランザクション能力を証明するものではない。
type memoryProvider struct {
	mu            sync.Mutex
	version       uint64
	value         string
	records       map[string]Receipt
	epochs        legacyauthorization.AuthorityEpochs
	enabled       bool
	receiptError  error
	commitError   error
	commitApplied bool
	failCheck     error
	events        []string
}
type memoryTx struct {
	db      *memoryProvider
	version uint64
	value   string
	records map[string]Receipt
	closed  bool
}

func fixtureRequest() Request {
	return Request{Scope: Scope{Application: "orders", Capability: "order.save", Subject: "editor", Action: "write", Resource: "order-1"}, OperationID: "operation-1", Payload: []byte("value")}
}
func fixtureEpochs() legacyauthorization.AuthorityEpochs {
	return legacyauthorization.AuthorityEpochs{Source: "authorization-primary", SubjectLink: 1, Membership: 2, Policy: 3}
}
func fixtureGrant(req Request) legacyauthorization.AuthorizationGrant {
	e := fixtureEpochs()
	return legacyauthorization.AuthorizationGrant{SubjectID: req.Scope.Subject, Action: req.Scope.Action, Resource: req.Scope.Resource, Epochs: &e}
}
func newMemoryProvider() *memoryProvider {
	return &memoryProvider{enabled: true, epochs: fixtureEpochs(), records: make(map[string]Receipt)}
}
func (db *memoryProvider) Begin(_ context.Context) (Transaction, error) {
	db.mu.Lock()
	db.events = append(db.events, "begin")
	copyRecords := make(map[string]Receipt, len(db.records))
	for k, v := range db.records {
		copyRecords[k] = v
	}
	return &memoryTx{db: db, version: db.version, value: db.value, records: copyRecords}, nil
}
func (tx *memoryTx) CheckAuthority(_ context.Context, s Scope, e legacyauthorization.AuthorityEpochs) (bool, error) {
	tx.db.events = append(tx.db.events, "authority")
	if tx.db.failCheck != nil {
		return false, tx.db.failCheck
	}
	return tx.db.enabled && s.Subject == "editor" && e == tx.db.epochs, nil
}
func (tx *memoryTx) FindReceipt(_ context.Context, operation string) (Receipt, bool, error) {
	tx.db.events = append(tx.db.events, "lookup")
	r, ok := tx.records[operation]
	return r, ok, nil
}
func (tx *memoryTx) WriteResource(_ context.Context, req Request) (uint64, error) {
	tx.db.events = append(tx.db.events, "write")
	if tx.version != req.ExpectedVersion {
		return 0, ErrResourceConflict
	}
	tx.version++
	tx.value = string(req.Payload)
	return tx.version, nil
}
func (tx *memoryTx) SaveReceipt(_ context.Context, rec Receipt) error {
	tx.db.events = append(tx.db.events, "receipt")
	if tx.db.receiptError != nil {
		return tx.db.receiptError
	}
	tx.records[rec.OperationID] = rec
	return nil
}
func (tx *memoryTx) Commit(_ context.Context) error {
	tx.db.events = append(tx.db.events, "commit")
	if tx.closed {
		return errors.New("already closed")
	}
	defer func() { tx.closed = true; tx.db.mu.Unlock() }()
	if tx.db.commitError != nil && !tx.db.commitApplied {
		return tx.db.commitError
	}
	tx.db.version, tx.db.value, tx.db.records = tx.version, tx.value, tx.records
	return tx.db.commitError
}
func (tx *memoryTx) Rollback() error {
	if tx.closed {
		return nil
	}
	tx.db.events = append(tx.db.events, "rollback")
	tx.closed = true
	tx.db.mu.Unlock()
	return nil
}

func TestAtomicCommitAndReceiptReplay(t *testing.T) {
	db := newMemoryProvider()
	runner := Runner{Provider: db}
	req := fixtureRequest()
	first, err := runner.Execute(context.Background(), req, fixtureGrant(req))
	if err != nil || first.Status != StatusCommitted || first.Receipt.ResourceVersion != 1 {
		t.Fatalf("initial: %+v %v", first, err)
	}
	replay, err := runner.Execute(context.Background(), req, fixtureGrant(req))
	if err != nil || replay.Status != StatusReplayed || replay.Receipt != first.Receipt {
		t.Fatalf("replay: %+v %v", replay, err)
	}
	observed, err := runner.Lookup(context.Background(), req, fixtureGrant(req))
	if err != nil || observed.Status != StatusReplayed || observed.Receipt != first.Receipt {
		t.Fatalf("lookup: %+v %v", observed, err)
	}
	if db.version != 1 || db.value != "value" || len(db.records) != 1 {
		t.Fatalf("duplicate effect: %+v", db)
	}
	want := []string{"begin", "authority", "lookup", "write", "receipt", "commit"}
	for i, step := range want {
		if db.events[i] != step {
			t.Fatalf("incorrect atomic ordering: %+v", db.events)
		}
	}
}

func TestRevokeBeforeReplayAndLookup(t *testing.T) {
	db := newMemoryProvider()
	runner := Runner{Provider: db}
	req := fixtureRequest()
	if _, err := runner.Execute(context.Background(), req, fixtureGrant(req)); err != nil {
		t.Fatal(err)
	}
	db.mu.Lock()
	db.enabled = false
	db.mu.Unlock()
	for _, action := range []func() (Result, error){
		func() (Result, error) { return runner.Execute(context.Background(), req, fixtureGrant(req)) },
		func() (Result, error) { return runner.Lookup(context.Background(), req, fixtureGrant(req)) },
	} {
		got, err := action()
		if err != nil || got.Status != StatusDenied || got.Receipt != (Receipt{}) {
			t.Fatalf("revoked receipt disclosed: %+v %v", got, err)
		}
	}
	if db.version != 1 {
		t.Fatal("revocation undid past effect")
	}
}

func TestGrantScopeEpochAndOperationMismatch(t *testing.T) {
	db := newMemoryProvider()
	runner := Runner{Provider: db}
	req := fixtureRequest()
	invalids := []struct {
		name  string
		req   Request
		grant legacyauthorization.AuthorizationGrant
	}{
		{"wrong-subject", req, func() legacyauthorization.AuthorizationGrant { g := fixtureGrant(req); g.SubjectID = "other"; return g }()},
		{"wrong-action", req, func() legacyauthorization.AuthorizationGrant { g := fixtureGrant(req); g.Action = "read"; return g }()},
		{"wrong-resource", req, func() legacyauthorization.AuthorizationGrant { g := fixtureGrant(req); g.Resource = "other"; return g }()},
		{"no-epochs", req, legacyauthorization.AuthorizationGrant{SubjectID: req.Scope.Subject, Action: req.Scope.Action, Resource: req.Scope.Resource}},
		{"missing-source", req, func() legacyauthorization.AuthorizationGrant { g := fixtureGrant(req); g.Epochs.Source = ""; return g }()},
		{"overflow-version", func() Request { r := req; r.ExpectedVersion = math.MaxUint64; return r }(), fixtureGrant(req)},
	}
	for _, item := range invalids {
		t.Run(item.name, func(t *testing.T) {
			got, err := runner.Execute(context.Background(), item.req, item.grant)
			if !errors.Is(err, ErrInvalidRequest) || got != (Result{}) {
				t.Fatalf("invalid request accepted: %+v %v", got, err)
			}
		})
	}
	if len(db.events) != 0 {
		t.Fatalf("invalid requests reached provider: %+v", db.events)
	}
}

func TestSameOperationIdRequiresIdenticalRequest(t *testing.T) {
	db := newMemoryProvider()
	runner := Runner{Provider: db}
	req := fixtureRequest()
	if _, err := runner.Execute(context.Background(), req, fixtureGrant(req)); err != nil {
		t.Fatal(err)
	}
	for _, changed := range []Request{
		func() Request { r := req; r.Payload = []byte("different"); return r }(),
		func() Request { r := req; r.Scope.Application = "other"; return r }(),
		func() Request { r := req; r.ExpectedVersion = 1; return r }(),
	} {
		got, err := runner.Execute(context.Background(), changed, fixtureGrant(changed))
		if err != nil || got.Status != StatusConflict || got.Receipt != (Receipt{}) {
			t.Fatalf("operation ID confused: %+v %v", got, err)
		}
		got, err = runner.Lookup(context.Background(), changed, fixtureGrant(changed))
		if err != nil || got.Status != StatusConflict || got.Receipt != (Receipt{}) {
			t.Fatalf("lookup leaked record: %+v %v", got, err)
		}
	}
	if db.version != 1 || db.value != "value" {
		t.Fatal("conflict modified data")
	}
}

func TestAuthorityVersionChangeBlocksCommit(t *testing.T) {
	db := newMemoryProvider()
	runner := Runner{Provider: db}
	req := fixtureRequest()
	old := fixtureGrant(req)
	db.mu.Lock()
	db.epochs.Membership++
	db.mu.Unlock()
	for _, act := range []func() (Result, error){
		func() (Result, error) { return runner.Execute(context.Background(), req, old) },
		func() (Result, error) { return runner.Lookup(context.Background(), req, old) },
	} {
		got, err := act()
		if err != nil || got.Status != StatusDenied || got.Receipt != (Receipt{}) {
			t.Fatalf("stale authority passed: %+v %v", got, err)
		}
	}
	if db.version != 0 || len(db.records) != 0 {
		t.Fatal("revoked request committed")
	}
}

func TestReceiptFailureRollsBackEffect(t *testing.T) {
	db := newMemoryProvider()
	db.receiptError = errors.New("receipt table full")
	req := fixtureRequest()
	result, err := (Runner{Provider: db}).Execute(context.Background(), req, fixtureGrant(req))
	if err == nil || result != (Result{}) || db.version != 0 || db.value != "" || len(db.records) != 0 {
		t.Fatalf("partial effect: %+v %v", result, err)
	}
	if db.events[len(db.events)-1] != "rollback" {
		t.Fatalf("not rolled back: %+v", db.events)
	}
}

func TestCommitErrorIsUncertainAndRetryFindsRecord(t *testing.T) {
	db := newMemoryProvider()
	db.commitApplied = true
	db.commitError = errors.New("connection lost during COMMIT")
	runner := Runner{Provider: db}
	req := fixtureRequest()
	got, err := runner.Execute(context.Background(), req, fixtureGrant(req))
	var uncertain *CommitUncertainError
	if got != (Result{}) || !errors.As(err, &uncertain) || !errors.Is(err, db.commitError) {
		t.Fatalf("commit error disguised: %+v %v", got, err)
	}
	db.commitError = nil
	retry, err := runner.Execute(context.Background(), req, fixtureGrant(req))
	if err != nil || retry.Status != StatusReplayed || retry.Receipt.ResourceVersion != 1 || db.version != 1 {
		t.Fatalf("retry after unknown outcome: %+v %v", retry, err)
	}
}

func TestVersionConflictNeverCreatesReceipt(t *testing.T) {
	db := newMemoryProvider()
	db.version = 2
	runner := Runner{Provider: db}
	req := fixtureRequest()
	got, err := runner.Execute(context.Background(), req, fixtureGrant(req))
	if err != nil || got.Status != StatusConflict || len(db.records) != 0 || db.version != 2 {
		t.Fatalf("CAS bypassed: %+v %v", got, err)
	}
}

func TestConcurrentSameResourceOnlyOneCommit(t *testing.T) {
	db := newMemoryProvider()
	runner := Runner{Provider: db}
	start := make(chan struct{})
	var wg sync.WaitGroup
	outcomes := make([]Result, 2)
	errs := make([]error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			r := fixtureRequest()
			r.OperationID = "operation-" + string(rune('A'+i))
			<-start
			outcomes[i], errs[i] = runner.Execute(context.Background(), r, fixtureGrant(r))
		}(i)
	}
	close(start)
	wg.Wait()
	nCommitted, nConflict := 0, 0
	for i, r := range outcomes {
		if errs[i] != nil {
			t.Fatal(errs[i])
		}
		switch r.Status {
		case StatusCommitted:
			nCommitted++
		case StatusConflict:
			nConflict++
		default:
			t.Fatalf("unexpected concurrent result: %+v", r)
		}
	}
	if nCommitted != 1 || nConflict != 1 || db.version != 1 || len(db.records) != 1 {
		t.Fatalf("concurrency violated: %+v %+v version=%d", outcomes, errs, db.version)
	}
}

func TestProviderFailureAndCanceledContext(t *testing.T) {
	db := newMemoryProvider()
	db.failCheck = errors.New("authority db unavailable")
	runner := Runner{Provider: db}
	req := fixtureRequest()
	got, err := runner.Execute(context.Background(), req, fixtureGrant(req))
	if !errors.Is(err, db.failCheck) || got != (Result{}) || len(db.records) != 0 {
		t.Fatalf("authority error converted to denial: %+v %v", got, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	db.events = nil
	got, err = runner.Execute(ctx, req, fixtureGrant(req))
	if !errors.Is(err, context.Canceled) || got != (Result{}) || len(db.events) != 0 {
		t.Fatalf("canceled action reached provider: %+v %v", got, err)
	}
}
