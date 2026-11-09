//go:build tei_sqlite_reference && cgo

package effectfence

import (
	"context"
	"path/filepath"
	"testing"
)

// 権限版が一致していても、DBの版更新トリガーが消えていれば確定しない。
func TestSQLiteMissingEpochTriggerRejectsPendingCommit(t *testing.T) {
	a, b, identity, req := setupSQLiteLegacyEpochs(t)
	auth := fixtureLegacyAdapter(identity, nil)
	permit, result, err := a.PrepareWithVersionedAuthorization(context.Background(), req.Scope.Subject, req, auth, fixtureAuthoritySource)
	if err != nil || result.Disposition != Prepared {
		t.Fatalf("準備に失敗: %+v %v", result, err)
	}
	if err := b.exec("DROP TRIGGER legacy_subject_epoch_on_update"); err != nil {
		t.Fatal(err)
	}
	result, err = a.CommitForTest(context.Background(), permit)
	if err == nil || result.Disposition != "" || result.Receipt != (Receipt{}) {
		t.Fatalf("DB制約なしの確定が許されました: %+v %v", result, err)
	}
	if _, version := b.Observe(req.Scope); version != 0 {
		t.Fatalf("制約がないまま書込みました: version=%d", version)
	}
	if _, ok := a.Lookup(req.Scope, req.OperationID); ok {
		t.Fatal("制約欠落後の確定記録照会を許しました")
	}
	if _, result, err = a.PrepareWithVersionedAuthorization(context.Background(), req.Scope.Subject, req, auth, fixtureAuthoritySource); err == nil || result.Receipt != (Receipt{}) {
		t.Fatalf("トリガー欠落後の再準備が許されました: %+v %v", result, err)
	}
}

func TestSQLiteModifiedEpochTriggerRejectsNewPermit(t *testing.T) {
	a, b, identity, req := setupSQLiteLegacyEpochs(t)
	// 名前だけを同じにした無作用のトリガーは、安全な版更新とは扱わない。
	if err := b.exec("DROP TRIGGER legacy_subject_epoch_on_update"); err != nil {
		t.Fatal(err)
	}
	if err := b.exec("CREATE TRIGGER legacy_subject_epoch_on_update AFTER UPDATE ON legacy_subjects BEGIN SELECT 1; END"); err != nil {
		t.Fatal(err)
	}
	_, result, err := a.PrepareWithVersionedAuthorization(context.Background(), req.Scope.Subject, req, fixtureLegacyAdapter(identity, nil), fixtureAuthoritySource)
	if err == nil || result.Disposition != "" || result.Receipt != (Receipt{}) {
		t.Fatalf("同名の改変トリガーを信頼しました: %+v %v", result, err)
	}
}

func TestSQLiteTriggerRestoredRequiresFreshPermit(t *testing.T) {
	a, b, identity, req := setupSQLiteLegacyEpochs(t)
	auth := fixtureLegacyAdapter(identity, nil)
	permit, result, err := a.PrepareWithVersionedAuthorization(context.Background(), req.Scope.Subject, req, auth, fixtureAuthoritySource)
	if err != nil || result.Disposition != Prepared {
		t.Fatalf("準備に失敗: %+v %v", result, err)
	}
	if err := b.exec("DROP TRIGGER legacy_subject_epoch_on_delete"); err != nil {
		t.Fatal(err)
	}
	if result, err = a.CommitForTest(context.Background(), permit); err == nil || result.Disposition != "" {
		t.Fatalf("制約欠落後の確定が成功: %+v %v", result, err)
	}
	// 同じ正規DDLを回復した後も、消費したPermitを再利用しない。
	if err := b.exec(requiredSQLiteLegacyEpochTriggers[1]); err != nil {
		t.Fatal(err)
	}
	result, err = a.CommitForTest(context.Background(), permit)
	if err != nil || result.Disposition != Denied || result.Reason != "unknown-permit" {
		t.Fatalf("失敗済みPermitを再利用しました: %+v %v", result, err)
	}
	permit, result, err = a.PrepareWithVersionedAuthorization(context.Background(), req.Scope.Subject, req, auth, fixtureAuthoritySource)
	if err != nil || result.Disposition != Prepared {
		t.Fatalf("正しい制約の復元後も拒否されました: %+v %v", result, err)
	}
	result, err = a.CommitForTest(context.Background(), permit)
	if err != nil || result.Disposition != Committed {
		t.Fatalf("復旧後の新規確定に失敗: %+v %v", result, err)
	}
}

func TestSQLiteVersionedAuthorizationNeedsSchemaProof(t *testing.T) {
	a, err := openSQLiteFixture(filepath.Join(t.TempDir(), "no-triggers.sqlite"))
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	s := epochScope()
	seedAuthorityDependencies(t, a, s)
	if err := a.exec("CREATE TABLE legacy_subjects(subject TEXT PRIMARY KEY, role TEXT NOT NULL, disabled INTEGER NOT NULL)"); err != nil {
		t.Fatal(err)
	}
	if err := a.exec("INSERT INTO legacy_subjects(subject,role,disabled) VALUES(" + textSQL(s.Subject) + ",'EDITOR',0)"); err != nil {
		t.Fatal(err)
	}
	identity := sqliteLegacyAuthority{f: a, scope: s, source: fixtureAuthoritySource}
	req := Request{Scope: s, OperationID: "unprotected", Value: "must-not-commit"}
	_, result, err := a.PrepareWithVersionedAuthorization(context.Background(), s.Subject, req, fixtureLegacyAdapter(identity, nil), fixtureAuthoritySource)
	if err == nil || result.Disposition != "" {
		t.Fatalf("版変更制約が未設置なのに準備できました: %+v %v", result, err)
	}
	if _, version := a.Observe(s); version != 0 {
		t.Fatal("版変更制約を持たないDBへの書込みが生じました")
	}
}
