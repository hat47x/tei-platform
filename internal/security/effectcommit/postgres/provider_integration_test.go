package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"tei/internal/security/effectcommit"
	"tei/internal/security/legacyauth"
	"tei/internal/security/legacyauthorization"
	"tei/internal/semantic/capabilityinvoke"
)

type postgresFixture struct {
	pool     *pgxpool.Pool
	schema   string
	provider Provider
	req      effectcommit.Request
}

func openFixture(t *testing.T) *postgresFixture {
	t.Helper()
	dsn := os.Getenv("TEI_EFFECT_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("実PostgreSQLの検証はTEI_EFFECT_POSTGRES_DSNがある場合のみ実行")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("PostgreSQLに接続できません: %v", err)
	}
	hash := sha256.Sum256([]byte(t.Name()))
	schema := "tei_fence_" + hex.EncodeToString(hash[:9])
	if _, err := pool.Exec(ctx, "DROP SCHEMA IF EXISTS "+schema+" CASCADE"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := pool.Exec(context.Background(), "DROP SCHEMA IF EXISTS "+schema+" CASCADE"); err != nil {
			t.Errorf("試験スキーマの削除に失敗: %v", err)
		}
	})
	sql := []string{
		"CREATE SCHEMA " + schema,
		"CREATE TABLE " + schema + ".subjects (subject TEXT PRIMARY KEY, role TEXT NOT NULL, disabled BOOLEAN NOT NULL DEFAULT FALSE)",
		"CREATE TABLE " + schema + ".authority (" +
			"application TEXT NOT NULL, capability TEXT NOT NULL, subject TEXT NOT NULL, action TEXT NOT NULL, resource TEXT NOT NULL, " +
			"source TEXT NOT NULL, subject_link BIGINT NOT NULL CHECK (subject_link > 0), membership BIGINT NOT NULL CHECK (membership > 0), " +
			"policy BIGINT NOT NULL CHECK (policy > 0), enabled BOOLEAN NOT NULL, " +
			"PRIMARY KEY (application,capability,subject,action,resource))",
		"CREATE TABLE " + schema + ".objects (application TEXT NOT NULL, resource TEXT NOT NULL, version BIGINT NOT NULL, value BYTEA NOT NULL, PRIMARY KEY(application,resource))",
		"CREATE TABLE " + schema + ".receipts (" +
			"operation_id TEXT PRIMARY KEY, application TEXT NOT NULL, capability TEXT NOT NULL, subject TEXT NOT NULL, action TEXT NOT NULL, " +
			"resource TEXT NOT NULL, request_hash TEXT NOT NULL, resource_version BIGINT NOT NULL)",
		"CREATE FUNCTION " + schema + ".bump_membership() RETURNS TRIGGER LANGUAGE plpgsql AS $$ BEGIN " +
			"UPDATE " + schema + ".authority SET membership=membership+1, enabled=NOT NEW.disabled WHERE subject=NEW.subject; " +
			"IF NOT FOUND THEN RAISE EXCEPTION 'missing authority version'; END IF; " +
			"RETURN NEW; END $$",
		"CREATE TRIGGER update_membership AFTER UPDATE OF role,disabled ON " + schema +
			".subjects FOR EACH ROW EXECUTE FUNCTION " + schema + ".bump_membership()",
	}
	for _, q := range sql {
		if _, err := pool.Exec(ctx, q); err != nil {
			t.Fatalf("試験用DB定義が不正です: %v / %s", err, q)
		}
	}
	req := effectcommit.Request{
		Scope:       effectcommit.Scope{Application: "orders", Capability: "order.save", Subject: "editor", Action: "write", Resource: "order-1"},
		OperationID: "operation-1", Payload: []byte("updated"),
	}
	for _, entry := range []struct {
		sql  string
		args []any
	}{
		{"INSERT INTO " + schema + ".subjects(subject,role,disabled) VALUES($1,'EDITOR',FALSE)", []any{req.Scope.Subject}},
		{"INSERT INTO " + schema + ".authority(application,capability,subject,action,resource,source,subject_link,membership,policy,enabled) " +
			"VALUES ($1,$2,$3,$4,$5,'postgres-authority',1,1,1,TRUE)",
			[]any{req.Scope.Application, req.Scope.Capability, req.Scope.Subject, req.Scope.Action, req.Scope.Resource}},
		{"INSERT INTO " + schema + ".objects(application,resource,version,value) VALUES ($1,$2,0,'original')",
			[]any{req.Scope.Application, req.Scope.Resource}},
	} {
		if _, err := pool.Exec(ctx, entry.sql, entry.args...); err != nil {
			t.Fatal(err)
		}
	}
	p, err := New(pool, schema)
	if err != nil {
		t.Fatal(err)
	}
	return &postgresFixture{pool: pool, schema: schema, provider: p, req: req}
}

type pgSubjectStore struct{ f *postgresFixture }

func (p pgSubjectStore) FindBySubject(ctx context.Context, subject string) (legacyauth.User, bool, error) {
	var user legacyauth.User
	err := p.f.pool.QueryRow(ctx, "SELECT subject,role,disabled FROM "+p.f.schema+".subjects WHERE subject=$1", subject).
		Scan(&user.SubjectID, &user.RoleCode, &user.Disabled)
	if errors.Is(err, pgx.ErrNoRows) {
		return legacyauth.User{}, false, nil
	}
	return user, err == nil, err
}

type pgEpochReader struct{ f *postgresFixture }

func (p pgEpochReader) ReadAuthorizationEpochs(ctx context.Context, input legacyauthorization.Input) (legacyauthorization.AuthorityEpochs, error) {
	q := "SELECT source,subject_link,membership,policy,enabled FROM " + p.f.schema +
		".authority WHERE subject=$1 AND action=$2 AND resource=$3 AND application=$4 AND capability=$5"
	var e legacyauthorization.AuthorityEpochs
	var link, membership, policy int64
	var enabled bool
	err := p.f.pool.QueryRow(ctx, q, input.SubjectID, input.Action, input.Resource,
		p.f.req.Scope.Application, p.f.req.Scope.Capability).
		Scan(&e.Source, &link, &membership, &policy, &enabled)
	if err != nil {
		return legacyauthorization.AuthorityEpochs{}, err
	}
	if link <= 0 || membership <= 0 || policy <= 0 {
		return legacyauthorization.AuthorityEpochs{}, errors.New("版が不正です")
	}
	if !enabled {
		return legacyauthorization.AuthorityEpochs{}, legacyauthorization.ErrAuthorityInactive
	}
	e.SubjectLink, e.Membership, e.Policy = uint64(link), uint64(membership), uint64(policy)
	return e, nil
}

type pgRule struct{}

func (pgRule) Evaluate(_ context.Context, user legacyauth.User, _, _ string) (bool, string, error) {
	return user.RoleCode == "EDITOR", "role_denied", nil
}

func (f *postgresFixture) grant(t *testing.T) legacyauthorization.AuthorizationGrant {
	t.Helper()
	adapter := legacyauthorization.Adapter{Store: pgSubjectStore{f}, EpochReader: pgEpochReader{f}, Rule: pgRule{}}
	got, err := adapter.Invoke(context.Background(), legacyauthorization.AuthorizationCapability, legacyauthorization.Input{
		SubjectID: f.req.Scope.Subject, Action: f.req.Scope.Action, Resource: f.req.Scope.Resource,
	})
	if err != nil || got.Outcome != "allowed" {
		t.Fatalf("security.authorize: %+v %v", got, err)
	}
	grant, ok := got.Payload.(legacyauthorization.AuthorizationGrant)
	if !ok || grant.Epochs == nil {
		t.Fatal("認可版がありません")
	}
	return grant
}

func (f *postgresFixture) readObject(t *testing.T) (string, int64) {
	t.Helper()
	var value []byte
	var version int64
	err := f.pool.QueryRow(context.Background(), "SELECT value,version FROM "+f.schema+".objects WHERE application=$1 AND resource=$2",
		f.req.Scope.Application, f.req.Scope.Resource).Scan(&value, &version)
	if err != nil {
		t.Fatal(err)
	}
	return string(value), version
}

func TestPostgresCommitReplayAndRestartLookup(t *testing.T) {
	f := openFixture(t)
	runner := effectcommit.Runner{Provider: f.provider}
	grant := f.grant(t)
	first, err := runner.Execute(context.Background(), f.req, grant)
	if err != nil || first.Status != effectcommit.StatusCommitted || first.Receipt.ResourceVersion != 1 {
		t.Fatalf("initial: %+v %v", first, err)
	}
	replay, err := runner.Execute(context.Background(), f.req, grant)
	if err != nil || replay.Status != effectcommit.StatusReplayed || replay.Receipt != first.Receipt {
		t.Fatalf("replay: %+v %v", replay, err)
	}
	// A different pool has no in-memory permit or receipt cache.
	second, err := pgxpool.New(context.Background(), os.Getenv("TEI_EFFECT_POSTGRES_DSN"))
	if err != nil {
		t.Fatal(err)
	}
	defer second.Close()
	p, err := New(second, f.schema)
	if err != nil {
		t.Fatal(err)
	}
	got, err := (effectcommit.Runner{Provider: p}).Lookup(context.Background(), f.req, grant)
	if err != nil || got.Status != effectcommit.StatusReplayed || got.Receipt != first.Receipt {
		t.Fatalf("reconnect: %+v %v", got, err)
	}
	if val, ver := f.readObject(t); val != "updated" || ver != 1 {
		t.Fatalf("duplicate data: %q %d", val, ver)
	}
}

func TestPostgresRoleRevocationPreventsCommitAndReceipt(t *testing.T) {
	f := openFixture(t)
	runner := effectcommit.Runner{Provider: f.provider}
	grant := f.grant(t)
	if _, err := f.pool.Exec(context.Background(), "UPDATE "+f.schema+".subjects SET role='VIEWER' WHERE subject=$1", f.req.Scope.Subject); err != nil {
		t.Fatal(err)
	}
	for _, operation := range []func() (effectcommit.Result, error){
		func() (effectcommit.Result, error) { return runner.Execute(context.Background(), f.req, grant) },
		func() (effectcommit.Result, error) { return runner.Lookup(context.Background(), f.req, grant) },
	} {
		result, err := operation()
		if err != nil || result.Status != effectcommit.StatusDenied || result.Receipt != (effectcommit.Receipt{}) {
			t.Fatalf("revoked: %+v %v", result, err)
		}
	}
	if val, ver := f.readObject(t); val != "original" || ver != 0 {
		t.Fatalf("revoked write: %q %d", val, ver)
	}
}

func TestPostgresRevocationAfterCommitHidesReceipt(t *testing.T) {
	f := openFixture(t)
	runner := effectcommit.Runner{Provider: f.provider}
	grant := f.grant(t)
	if _, err := runner.Execute(context.Background(), f.req, grant); err != nil {
		t.Fatal(err)
	}
	if _, err := f.pool.Exec(context.Background(), "UPDATE "+f.schema+".subjects SET disabled=TRUE WHERE subject=$1", f.req.Scope.Subject); err != nil {
		t.Fatal(err)
	}
	got, err := runner.Lookup(context.Background(), f.req, grant)
	if err != nil || got.Status != effectcommit.StatusDenied || got.Receipt != (effectcommit.Receipt{}) {
		t.Fatalf("revoked receipt disclosed: %+v %v", got, err)
	}
	if val, ver := f.readObject(t); val != "updated" || ver != 1 {
		t.Fatalf("prior effect modified: %q %d", val, ver)
	}
}

func TestPostgresResourceCASAndReceiptRollback(t *testing.T) {
	f := openFixture(t)
	runner := effectcommit.Runner{Provider: f.provider}
	grant := f.grant(t)
	if _, err := f.pool.Exec(context.Background(), "UPDATE "+f.schema+".objects SET version=1 WHERE application='orders'"); err != nil {
		t.Fatal(err)
	}
	got, err := runner.Execute(context.Background(), f.req, grant)
	if err != nil || got.Status != effectcommit.StatusConflict {
		t.Fatalf("wrong version committed: %+v %v", got, err)
	}
	if _, err := f.pool.Exec(context.Background(), "UPDATE "+f.schema+".objects SET version=0 WHERE application='orders'"); err != nil {
		t.Fatal(err)
	}
	_, err = f.pool.Exec(context.Background(), "CREATE FUNCTION "+f.schema+".deny_receipt() RETURNS TRIGGER LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'receipt unavailable'; END $$")
	if err != nil {
		t.Fatal(err)
	}
	_, err = f.pool.Exec(context.Background(), "CREATE TRIGGER deny_receipt BEFORE INSERT ON "+f.schema+".receipts FOR EACH ROW EXECUTE FUNCTION "+f.schema+".deny_receipt()")
	if err != nil {
		t.Fatal(err)
	}
	got, err = runner.Execute(context.Background(), f.req, grant)
	if err == nil || got != (effectcommit.Result{}) {
		t.Fatalf("receipt fault should fail: %+v %v", got, err)
	}
	if val, ver := f.readObject(t); val != "original" || ver != 0 {
		t.Fatalf("partial effect after receipt failure: %q %d", val, ver)
	}
	if _, err := f.pool.Exec(context.Background(), "DROP TRIGGER deny_receipt ON "+f.schema+".receipts"); err != nil {
		t.Fatal(err)
	}
	got, err = runner.Execute(context.Background(), f.req, grant)
	if err != nil || got.Status != effectcommit.StatusCommitted {
		t.Fatalf("retry after rollback: %+v %v", got, err)
	}
}

func TestPostgresSameOperationDifferentRequestIsConflict(t *testing.T) {
	f := openFixture(t)
	runner := effectcommit.Runner{Provider: f.provider}
	grant := f.grant(t)
	if _, err := runner.Execute(context.Background(), f.req, grant); err != nil {
		t.Fatal(err)
	}
	conflict := f.req
	conflict.Payload = []byte("different")
	got, err := runner.Execute(context.Background(), conflict, grant)
	if err != nil || got.Status != effectcommit.StatusConflict || got.Receipt != (effectcommit.Receipt{}) {
		t.Fatalf("payload conflict: %+v %v", got, err)
	}
}

func TestPostgresConcurrentRevokeAndCommitLinearize(t *testing.T) {
	for iteration := 0; iteration < 8; iteration++ {
		f := openFixture(t)
		runner := effectcommit.Runner{Provider: f.provider}
		grant := f.grant(t)
		begin := make(chan struct{})
		var wg sync.WaitGroup
		var result effectcommit.Result
		var writeErr, revokeErr error
		wg.Add(2)
		go func() {
			defer wg.Done()
			<-begin
			result, writeErr = runner.Execute(context.Background(), f.req, grant)
		}()
		go func() {
			defer wg.Done()
			<-begin
			_, revokeErr = f.pool.Exec(context.Background(), "UPDATE "+f.schema+".subjects SET role='VIEWER' WHERE subject=$1", f.req.Scope.Subject)
		}()
		close(begin)
		wg.Wait()
		if writeErr != nil || revokeErr != nil {
			t.Fatalf("concurrent database failure: %v / %v", writeErr, revokeErr)
		}
		_, ver := f.readObject(t)
		if (result.Status == effectcommit.StatusCommitted && ver != 1) || (result.Status == effectcommit.StatusDenied && ver != 0) ||
			(result.Status != effectcommit.StatusCommitted && result.Status != effectcommit.StatusDenied) {
			t.Fatalf("serial order violated: %+v version=%d", result, ver)
		}
	}
}

func TestPostgresSchemaConfigurationRejectsUnsafeNames(t *testing.T) {
	for _, s := range []string{"", "public; DROP TABLE users", "schema name", "x.y", "UPPER", strings.Repeat("a", 64)} {
		if _, err := New(&pgxpool.Pool{}, s); err == nil {
			t.Fatalf("unsafe schema: %q", s)
		}
	}
}

func TestPostgresCommitErrorRemainsUncertain(t *testing.T) {
	f := openFixture(t)
	base := f.provider
	wrapped := uncertainProvider{inner: base}
	runner := effectcommit.Runner{Provider: wrapped}
	grant := f.grant(t)
	got, err := runner.Execute(context.Background(), f.req, grant)
	var uncertain *effectcommit.CommitUncertainError
	if got != (effectcommit.Result{}) || !errors.As(err, &uncertain) {
		t.Fatalf("commit response loss hidden: %+v %v", got, err)
	}
	retry, err := (effectcommit.Runner{Provider: base}).Execute(context.Background(), f.req, grant)
	if err != nil || retry.Status != effectcommit.StatusReplayed || retry.Receipt.ResourceVersion != 1 {
		t.Fatalf("receipt recovery: %+v %v", retry, err)
	}
}

type uncertainProvider struct{ inner Provider }

func (u uncertainProvider) Begin(ctx context.Context) (effectcommit.Transaction, error) {
	tx, err := u.inner.Begin(ctx)
	if err != nil {
		return nil, err
	}
	return uncertainTx{Transaction: tx}, nil
}

type uncertainTx struct{ effectcommit.Transaction }

func (t uncertainTx) Commit(ctx context.Context) error {
	if err := t.Transaction.Commit(ctx); err != nil {
		return err
	}
	return fmt.Errorf("network response lost after server committed")
}

var _ capabilityinvoke.Adapter = legacyauthorization.Adapter{}
