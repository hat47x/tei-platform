package postgres

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"tei/internal/command"
	"tei/internal/command/lifecyclepg"
	"tei/internal/security/effectcommit"
	"tei/internal/security/effectcommit/commandadapter"
	"tei/internal/security/legacyauthorization"
	"tei/internal/semantic"
)

// #1174 RC-03 / RC-05 on a real provider. The effectcommit Runner commits in
// its own PostgreSQL transaction; the Command's operation record and its
// lifecycle stage are written afterwards and are NOT part of that commit. These
// tests inject a failure into each of those later writes and check that the
// committed effect is reported as unknown (never as a clean failure), that the
// durable lifecycle refuses a blind re-run, and that the facts can be recovered
// from the existing receipt and an independent read on another connection.
//
// They do not show atomicity between the provider commit and the Command
// records. There is none, and the tests pin what holds without it.

type failingRecorder struct{}

func (failingRecorder) Record(context.Context, command.OperationRecord) error {
	return errors.New("operation record store unavailable")
}

// observeFailingStore lets Dispatch/MarkUnknown/Takeover reach the real
// PostgreSQL lifecycle table but fails the outcome write, as if the lifecycle
// store dropped out right after the provider committed.
type observeFailingStore struct{ *lifecyclepg.Store }

func (observeFailingStore) Observe(string, string, int, string) error {
	return errors.New("lifecycle store unavailable")
}

type recordFailureFixture struct {
	*postgresFixture
	grant legacyauthorization.AuthorizationGrant
	dsn   string
}

func openRecordFailureFixture(t *testing.T) recordFailureFixture {
	t.Helper()
	f := openFixture(t)
	for _, statement := range []string{
		"INSERT INTO " + f.schema + ".authority(application,capability,subject,action,resource,source,subject_link,membership,policy,enabled) " +
			"VALUES ('order_entry','sales.submit','editor','write','order-1','postgres-authority',1,1,1,TRUE)",
		"INSERT INTO " + f.schema + ".objects(application,resource,version,value) VALUES ('order_entry','order-1',0,'original')",
	} {
		if _, err := f.pool.Exec(context.Background(), statement); err != nil {
			t.Fatal(err)
		}
	}
	return recordFailureFixture{postgresFixture: f, grant: f.grant(t), dsn: os.Getenv("TEI_EFFECT_POSTGRES_DSN")}
}

// openLifecycle opens the durable lifecycle on its own connection. The table
// lives in the fixture's disposable schema and is dropped with it.
func (f recordFailureFixture) openLifecycle(t *testing.T) *lifecyclepg.Store {
	t.Helper()
	store, err := lifecyclepg.Open(f.dsn, f.schema)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	return store
}

func (f recordFailureFixture) executor(operation string) commandadapter.Executor {
	return commandadapter.Executor{
		Runner:          effectcommit.Runner{Provider: f.provider},
		Grant:           f.grant,
		OperationID:     operation,
		ExpectedVersion: 0,
		Payload:         []byte("updated"),
		Effect:          &semantic.DataEffect{Kind: semantic.DataEffectUpdate, Data: "sales.order"},
	}
}

func (f recordFailureFixture) receipts(t *testing.T) int {
	t.Helper()
	var count int
	if err := f.pool.QueryRow(context.Background(), "SELECT count(*) FROM "+f.schema+".receipts").Scan(&count); err != nil {
		t.Fatal(err)
	}
	return count
}

// independentFacts reads the receipt and the stored resource through a pool
// that shares nothing with the one that committed.
func (f recordFailureFixture) independentFacts(t *testing.T, operation string) (effectcommit.Result, effectcommit.ResourceState) {
	t.Helper()
	ctx := context.Background()
	other, err := pgxpool.New(ctx, f.dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer other.Close()
	provider, err := New(other, f.schema)
	if err != nil {
		t.Fatal(err)
	}
	runner := effectcommit.Runner{Provider: provider}
	scope := effectcommit.Scope{Application: "order_entry", Capability: "sales.submit", Subject: f.grant.SubjectID, Action: f.grant.Action, Resource: f.grant.Resource}
	receipt, err := runner.Lookup(ctx, effectcommit.Request{Scope: scope, OperationID: operation, ExpectedVersion: 0, Payload: []byte("updated")}, f.grant)
	if err != nil {
		t.Fatal(err)
	}
	state, err := runner.ReadResource(ctx, scope)
	if err != nil {
		t.Fatal(err)
	}
	return receipt, state
}

func executionCodeOf(t *testing.T, err error) string {
	t.Helper()
	var execution *command.ExecutionError
	if !errors.As(err, &execution) {
		t.Fatalf("error is not an ExecutionError: %v", err)
	}
	return execution.Code
}

// RC-03: PostgreSQL committed, then the operation record failed.
func TestPostgresRecorderFailureAfterCommitIsUnknownAndRecoverableByReceipt(t *testing.T) {
	f := openRecordFailureFixture(t)
	ctx := context.Background()
	const operation = "record-failure-op"
	boundary := func(stages command.StageLifecycle, recorder command.Recorder) command.LifecycleInvoker {
		return command.LifecycleInvoker{
			Lifecycle: stages,
			Invoker: command.Invoker{
				Authorizer: command.ReferenceAuthorizer{},
				World:      commandWorldE2E(),
				Executor:   f.executor(operation),
				Recorder:   recorder,
			},
		}
	}
	options := command.DispatchOptions{CommandID: "cmd-record", Owner: "owner-a", Generation: 1}

	_, err := boundary(f.openLifecycle(t), failingRecorder{}).Invoke(ctx, commandRequestE2E(), options)
	if code := executionCodeOf(t, err); code != "COMMAND_RECORD_FAILED" {
		t.Fatalf("code=%s, want COMMAND_RECORD_FAILED", code)
	}
	if !command.EffectMayBeUnknown(err) {
		t.Fatal("a recorder failure after a real commit was reported as a definite failure")
	}
	// The effect is committed although the Command failed.
	if value, version := readE2EObject(t, f.postgresFixture); value != "updated" || version != 1 {
		t.Fatalf("object=%q/%d, want the committed effect", value, version)
	}

	// "Restart": a new lifecycle connection sees the durable unknown stage and
	// refuses the same attempt before the Executor runs.
	restarted := f.openLifecycle(t)
	if attempt, ok := restarted.Attempt(options.CommandID); !ok || attempt.Stage != command.StageUnknown {
		t.Fatalf("durable attempt=%+v ok=%v, want unknown", attempt, ok)
	}
	if restarted.MayReplay(options.CommandID) {
		t.Fatal("an unknown command was reported as replayable after restart")
	}
	recorder := &command.MemoryRecorder{}
	_, err = boundary(restarted, recorder).Invoke(ctx, commandRequestE2E(), options)
	if code := executionCodeOf(t, err); code != "COMMAND_LIFECYCLE_REFUSED" {
		t.Fatalf("replay code=%s, want COMMAND_LIFECYCLE_REFUSED", code)
	}
	if command.EffectMayBeUnknown(err) {
		t.Fatal("a refusal before execution was reported as an unknown effect")
	}

	// The existing receipt and an independent read establish what happened.
	receipt, state := f.independentFacts(t, operation)
	if receipt.Status != effectcommit.StatusReplayed || receipt.Receipt.ResourceVersion != 1 {
		t.Fatalf("receipt lookup=%+v, want the committed receipt", receipt)
	}
	if !state.Found || string(state.Payload) != "updated" {
		t.Fatalf("independent read=%+v, want the committed payload", state)
	}

	// A takeover re-submits the same operation id. The receipt makes it a
	// replay: no second write, no second receipt, and the record is now kept.
	if _, err := boundary(restarted, recorder).Invoke(ctx, commandRequestE2E(), command.DispatchOptions{
		CommandID: options.CommandID, Owner: "owner-b", Generation: 2, Takeover: true,
	}); err != nil {
		t.Fatal(err)
	}
	if attempt, _ := restarted.Attempt(options.CommandID); attempt.Stage != command.StageObserved || attempt.TakeoverFrom != 1 {
		t.Fatalf("takeover attempt=%+v", attempt)
	}
	if value, version := readE2EObject(t, f.postgresFixture); value != "updated" || version != 1 {
		t.Fatalf("takeover duplicated the effect: %q/%d", value, version)
	}
	if count := f.receipts(t); count != 1 {
		t.Fatalf("receipts=%d, want 1", count)
	}
	if records := recorder.Records; len(records) != 1 {
		t.Fatalf("operation records=%d, want 1", len(records))
	}
}

// RC-05: PostgreSQL committed, then the lifecycle outcome write failed. The
// durable stage stays dispatched: the effect is committed but no stage says so.
func TestPostgresLifecycleObserveFailureAfterCommitStaysInDoubt(t *testing.T) {
	f := openRecordFailureFixture(t)
	ctx := context.Background()
	const operation = "observe-failure-op"
	options := command.DispatchOptions{CommandID: "cmd-observe", Owner: "owner-a", Generation: 1}
	invoker := command.Invoker{
		Authorizer: command.ReferenceAuthorizer{},
		World:      commandWorldE2E(),
		Executor:   f.executor(operation),
	}

	_, err := command.LifecycleInvoker{
		Lifecycle: observeFailingStore{f.openLifecycle(t)},
		Invoker:   invoker,
	}.Invoke(ctx, commandRequestE2E(), options)
	if code := executionCodeOf(t, err); code != "COMMAND_LIFECYCLE_RECORD_FAILED" {
		t.Fatalf("code=%s, want COMMAND_LIFECYCLE_RECORD_FAILED", code)
	}
	if !command.EffectMayBeUnknown(err) {
		t.Fatal("a lifecycle write failure after a real commit was reported as a definite failure")
	}
	if value, version := readE2EObject(t, f.postgresFixture); value != "updated" || version != 1 {
		t.Fatalf("object=%q/%d, want the committed effect", value, version)
	}

	restarted := f.openLifecycle(t)
	if attempt, ok := restarted.Attempt(options.CommandID); !ok || attempt.Stage != command.StageDispatched {
		t.Fatalf("durable attempt=%+v ok=%v, want dispatched", attempt, ok)
	}
	// Before the fix the durable store answered true here.
	if restarted.MayReplay(options.CommandID) {
		t.Fatal("a committed command with no recorded outcome was reported as replayable")
	}
	boundary := command.LifecycleInvoker{Lifecycle: restarted, Invoker: invoker}
	if _, err := boundary.Invoke(ctx, commandRequestE2E(), options); executionCodeOf(t, err) != "COMMAND_LIFECYCLE_REFUSED" {
		t.Fatalf("re-dispatch error=%v, want COMMAND_LIFECYCLE_REFUSED", err)
	}
	// Stop condition: the lifecycle offers no automatic way out of dispatched.
	// A takeover needs an unknown stage, and nothing recorded one.
	if _, err := boundary.Invoke(ctx, commandRequestE2E(), command.DispatchOptions{
		CommandID: options.CommandID, Owner: "owner-b", Generation: 2, Takeover: true,
	}); executionCodeOf(t, err) != "COMMAND_LIFECYCLE_REFUSED" {
		t.Fatalf("takeover error=%v, want COMMAND_LIFECYCLE_REFUSED", err)
	}
	if value, version := readE2EObject(t, f.postgresFixture); value != "updated" || version != 1 {
		t.Fatalf("refused attempts changed the object: %q/%d", value, version)
	}

	// The facts are still recoverable without re-running the command: the
	// receipt and an independent read show the commit, and the original
	// owner/generation can then record the outcome it failed to write.
	receipt, state := f.independentFacts(t, operation)
	if receipt.Status != effectcommit.StatusReplayed || receipt.Receipt.ResourceVersion != 1 {
		t.Fatalf("receipt lookup=%+v, want the committed receipt", receipt)
	}
	if !state.Found || string(state.Payload) != "updated" {
		t.Fatalf("independent read=%+v, want the committed payload", state)
	}
	if err := restarted.Observe(options.CommandID, options.Owner, options.Generation, command.OutcomeCommitted); err != nil {
		t.Fatal(err)
	}
	if attempt, _ := restarted.Attempt(options.CommandID); attempt.Stage != command.StageObserved {
		t.Fatalf("recorded attempt=%+v, want observed", attempt)
	}
	if count := f.receipts(t); count != 1 {
		t.Fatalf("receipts=%d, want 1", count)
	}
}
