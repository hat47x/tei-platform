//go:build tei_sqlite_reference && cgo

package effectfence

import (
	"context"
	"errors"
	"testing"

	"tei/internal/security/legacyauth"
	"tei/internal/security/legacyauthorization"
)

// 外部からの確定応答が失われたときに、別の接続で再認可して記録を回収する。
func TestSQLiteVerifiedReceiptSurvivesConnectionRestart(t *testing.T) {
	a, _, identity, req := setupSQLiteLegacyEpochs(t)
	auth := fixtureLegacyAdapter(identity, nil)
	permit, prepared, err := a.PrepareWithVersionedAuthorization(context.Background(), req.Scope.Subject, req, auth, fixtureAuthoritySource)
	if err != nil || prepared.Disposition != Prepared {
		t.Fatalf("prepare: %+v %v", prepared, err)
	}
	committed, err := a.CommitForTest(context.Background(), permit)
	if err != nil || committed.Disposition != Committed {
		t.Fatalf("commit: %+v %v", committed, err)
	}
	a.mu.Lock()
	db, found, err := a.query("PRAGMA database_list")
	a.mu.Unlock()
	if err != nil || !found || len(db) != 3 {
		t.Fatalf("db path: %v %+v", err, db)
	}
	a.Close()
	reopened, err := openSQLiteFixture(db[2])
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	// 元の接続内のschemaCheckやrequireEpochsが無くても、実DBを再検査する。
	if reopened.schemaCheck != nil || reopened.requireEpochs {
		t.Fatal("再接続時の検証設定が予期せず残っています")
	}
	refreshed := fixtureLegacyAdapter(sqliteLegacyAuthority{f: reopened, scope: req.Scope, source: fixtureAuthoritySource}, nil)
	result, err := reopened.LookupWithVersionedAuthorization(context.Background(), req.Scope.Subject, req, refreshed, fixtureAuthoritySource)
	if err != nil || result.Status != ReceiptRecorded || result.Receipt != committed.Receipt {
		t.Fatalf("verified receipt recovery: %+v %v", result, err)
	}
	if _, version := reopened.Observe(req.Scope); version != 1 {
		t.Fatalf("照会による二重副作用: %d", version)
	}
	if len(reopened.pending) != 0 {
		t.Fatal("確定記録照会が新しいPermitを発行しました")
	}
}

func TestSQLiteVerifiedReceiptDistinguishesMissingFromDenied(t *testing.T) {
	a, b, identity, req := setupSQLiteLegacyEpochs(t)
	auth := fixtureLegacyAdapter(identity, nil)
	result, err := a.LookupWithVersionedAuthorization(context.Background(), req.Scope.Subject, req, auth, fixtureAuthoritySource)
	if err != nil || result.Status != ReceiptNotRecorded || result.Receipt != (Receipt{}) {
		t.Fatalf("記録のない状態を表現できません: %+v %v", result, err)
	}
	if len(a.pending) != 0 {
		t.Fatal("確定記録の照会で保留操作が作られました")
	}
	if err := changeSQLiteLegacyRole(b, req.Scope, 2, "VIEWER"); err != nil {
		t.Fatal(err)
	}
	result, err = a.LookupWithVersionedAuthorization(context.Background(), req.Scope.Subject, req, auth, fixtureAuthoritySource)
	if err != nil || result.Status != ReceiptNotAllowed || result.Receipt != (Receipt{}) {
		t.Fatalf("権限失効を記録不存在と混同しました: %+v %v", result, err)
	}
}

func TestSQLiteVerifiedReceiptCannotBeDisclosedAfterRevocation(t *testing.T) {
	a, b, identity, req := setupSQLiteLegacyEpochs(t)
	auth := fixtureLegacyAdapter(identity, nil)
	permit, prepared, err := a.PrepareWithVersionedAuthorization(context.Background(), req.Scope.Subject, req, auth, fixtureAuthoritySource)
	if err != nil || prepared.Disposition != Prepared {
		t.Fatalf("prepare: %+v %v", prepared, err)
	}
	if _, err := a.CommitForTest(context.Background(), permit); err != nil {
		t.Fatal(err)
	}
	if err := b.exec("UPDATE legacy_subjects SET disabled=1 WHERE subject=" + textSQL(req.Scope.Subject)); err != nil {
		t.Fatal(err)
	}
	result, err := a.LookupWithVersionedAuthorization(context.Background(), req.Scope.Subject, req, auth, fixtureAuthoritySource)
	if err != nil || result.Status != ReceiptNotAllowed || result.Receipt != (Receipt{}) {
		t.Fatalf("失効した主体に記録を開示しました: %+v %v", result, err)
	}
}

func TestSQLiteVerifiedReceiptFailsClosedWhenTriggerDroppedAfterRestart(t *testing.T) {
	a, b, identity, req := setupSQLiteLegacyEpochs(t)
	auth := fixtureLegacyAdapter(identity, nil)
	permit, prepared, err := a.PrepareWithVersionedAuthorization(context.Background(), req.Scope.Subject, req, auth, fixtureAuthoritySource)
	if err != nil || prepared.Disposition != Prepared {
		t.Fatalf("prepare: %+v %v", prepared, err)
	}
	if _, err := a.CommitForTest(context.Background(), permit); err != nil {
		t.Fatal(err)
	}
	b.mu.Lock()
	db, found, err := b.query("PRAGMA database_list")
	b.mu.Unlock()
	if err != nil || !found || len(db) != 3 {
		t.Fatal("database file unavailable")
	}
	if err := b.exec("DROP TRIGGER legacy_subject_epoch_on_update"); err != nil {
		t.Fatal(err)
	}
	reopened, err := openSQLiteFixture(db[2])
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	refreshed := fixtureLegacyAdapter(sqliteLegacyAuthority{f: reopened, scope: req.Scope, source: fixtureAuthoritySource}, nil)
	result, err := reopened.LookupWithVersionedAuthorization(context.Background(), req.Scope.Subject, req, refreshed, fixtureAuthoritySource)
	if err == nil || result != (ReceiptLookupResult{}) {
		t.Fatalf("トリガー削除後に再接続から記録を取得しました: %+v %v", result, err)
	}
	if err := b.exec(requiredSQLiteLegacyEpochTriggers[0]); err != nil {
		t.Fatal(err)
	}
	result, err = reopened.LookupWithVersionedAuthorization(context.Background(), req.Scope.Subject, req, refreshed, fixtureAuthoritySource)
	if err != nil || result.Status != ReceiptRecorded {
		t.Fatalf("制約回復後の再照会が失敗: %+v %v", result, err)
	}
}

func TestSQLiteVerifiedReceiptRejectsOperationConflictAndWrongAuthority(t *testing.T) {
	a, _, identity, req := setupSQLiteLegacyEpochs(t)
	auth := fixtureLegacyAdapter(identity, nil)
	permit, prepared, err := a.PrepareWithVersionedAuthorization(context.Background(), req.Scope.Subject, req, auth, fixtureAuthoritySource)
	if err != nil || prepared.Disposition != Prepared {
		t.Fatalf("prepare: %+v %v", prepared, err)
	}
	if _, err := a.CommitForTest(context.Background(), permit); err != nil {
		t.Fatal(err)
	}
	conflicting := req
	conflicting.Value = "not-original"
	result, err := a.LookupWithVersionedAuthorization(context.Background(), req.Scope.Subject, conflicting, auth, fixtureAuthoritySource)
	if err != nil || result.Status != ReceiptNotAllowed || result.Receipt != (Receipt{}) {
		t.Fatalf("異なる要求本文に確定記録を開示しました: %+v %v", result, err)
	}
	result, err = a.LookupWithVersionedAuthorization(context.Background(), "another-subject", req, auth, fixtureAuthoritySource)
	if err != nil || result.Status != ReceiptNotAllowed {
		t.Fatalf("別主体の要求を許可しました: %+v %v", result, err)
	}
	result, err = a.LookupWithVersionedAuthorization(context.Background(), req.Scope.Subject, req, auth, "wrong-source")
	if err == nil || result != (ReceiptLookupResult{}) {
		t.Fatalf("正本識別子の不一致を許可しました: %+v %v", result, err)
	}
	withoutEpoch := legacyauthorization.Adapter{
		Store: identity,
		Rule: legacyRuleFunc(func(context.Context, legacyauth.User, string, string) (bool, string, error) {
			return true, "", nil
		}),
	}
	result, err = a.LookupWithVersionedAuthorization(context.Background(), req.Scope.Subject, req, withoutEpoch, fixtureAuthoritySource)
	if err == nil || result != (ReceiptLookupResult{}) {
		t.Fatalf("版のない認可結果で再照会できました: %+v %v", result, err)
	}
}

func TestSQLiteVerifiedReceiptProviderFailureIsNotNonexistent(t *testing.T) {
	a, b, identity, req := setupSQLiteLegacyEpochs(t)
	auth := fixtureLegacyAdapter(identity, nil)
	if err := b.exec("DROP TABLE legacy_subjects"); err != nil {
		t.Fatal(err)
	}
	result, err := a.LookupWithVersionedAuthorization(context.Background(), req.Scope.Subject, req, auth, fixtureAuthoritySource)
	if err == nil || result != (ReceiptLookupResult{}) {
		t.Fatalf("認可正本の障害を記録不存在に変換しました: %+v %v", result, err)
	}
	closed, cancel := context.WithCancel(context.Background())
	cancel()
	result, err = a.LookupWithVersionedAuthorization(closed, req.Scope.Subject, req, auth, fixtureAuthoritySource)
	if !errors.Is(err, context.Canceled) || result != (ReceiptLookupResult{}) {
		t.Fatalf("中止した操作が記録を開示しました: %+v %v", result, err)
	}
}
