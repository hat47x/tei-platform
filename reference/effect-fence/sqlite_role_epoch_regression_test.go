//go:build tei_sqlite_reference && cgo

package effectfence

import (
	"context"
	"math"
	"sync"
	"testing"
)

func sqliteMembershipEpoch(t *testing.T, f *sqliteFixture, s Scope) (string, string) {
	t.Helper()
	f.mu.Lock()
	defer f.mu.Unlock()
	row, exists, err := f.query("SELECT version,allowed FROM authority_dependencies WHERE scope=" + scopeSQL(s) + " AND dimension=" + textSQL("membership"))
	if err != nil || !exists {
		t.Fatalf("所属版を確認できません: %v exists=%v", err, exists)
	}
	return row[0], row[1]
}

func sqliteLegacyUser(t *testing.T, f *sqliteFixture, subject string) (string, string, bool) {
	t.Helper()
	f.mu.Lock()
	defer f.mu.Unlock()
	row, exists, err := f.query("SELECT role,disabled FROM legacy_subjects WHERE subject=" + textSQL(subject))
	if err != nil {
		t.Fatal(err)
	}
	if !exists {
		return "", "", false
	}
	return row[0], row[1], true
}

// 呼出側が版更新を一切行わずSQLでロールを変更しても、旧Permitは失効する。
func TestSQLiteDirectRoleUpdateCannotBypassMembershipEpoch(t *testing.T) {
	a, b, identity, req := setupSQLiteLegacyEpochs(t)
	auth := fixtureLegacyAdapter(identity, nil)
	token, result, err := a.PrepareWithVersionedAuthorization(context.Background(), req.Scope.Subject, req, auth, fixtureAuthoritySource)
	if err != nil || result.Disposition != Prepared {
		t.Fatalf("操作の準備に失敗: %+v %v", result, err)
	}
	if err := b.exec("UPDATE legacy_subjects SET role='VIEWER' WHERE subject=" + textSQL(req.Scope.Subject)); err != nil {
		t.Fatal(err)
	}
	if version, allowed := sqliteMembershipEpoch(t, b, req.Scope); version != "2" || allowed != "1" {
		t.Fatalf("直接SQL更新で所属版が増加していません: version=%s allowed=%s", version, allowed)
	}
	result, err = a.CommitForTest(context.Background(), token)
	if err != nil || result.Disposition != Denied || result.Reason != "authority-dependency-changed" {
		t.Fatalf("古いRoleの許可が確定しました: %+v %v", result, err)
	}
	if _, version := a.Observe(req.Scope); version != 0 {
		t.Fatalf("失効後に副作用が発生: %d", version)
	}
}

func TestSQLiteDirectAccountDisableAndRestoreBumpMembershipEpoch(t *testing.T) {
	a, b, identity, req := setupSQLiteLegacyEpochs(t)
	auth := fixtureLegacyAdapter(identity, nil)
	token, result, err := a.PrepareWithVersionedAuthorization(context.Background(), req.Scope.Subject, req, auth, fixtureAuthoritySource)
	if err != nil || result.Disposition != Prepared {
		t.Fatalf("準備に失敗: %+v %v", result, err)
	}
	if err := b.exec("UPDATE legacy_subjects SET disabled=1 WHERE subject=" + textSQL(req.Scope.Subject)); err != nil {
		t.Fatal(err)
	}
	if version, allowed := sqliteMembershipEpoch(t, b, req.Scope); version != "2" || allowed != "0" {
		t.Fatalf("アカウント無効化と失効版が一致しません: %s %s", version, allowed)
	}
	result, err = a.CommitForTest(context.Background(), token)
	if err != nil || result.Disposition != Denied {
		t.Fatalf("無効アカウントの操作が確定: %+v %v", result, err)
	}
	if err := b.exec("UPDATE legacy_subjects SET disabled=0 WHERE subject=" + textSQL(req.Scope.Subject)); err != nil {
		t.Fatal(err)
	}
	if version, allowed := sqliteMembershipEpoch(t, b, req.Scope); version != "3" || allowed != "1" {
		t.Fatalf("再有効化後に新しい版を発行できません: %s %s", version, allowed)
	}
	token, result, err = a.PrepareWithVersionedAuthorization(context.Background(), req.Scope.Subject, req, auth, fixtureAuthoritySource)
	if err != nil || result.Disposition != Prepared {
		t.Fatalf("新しい版で再準備できません: %+v %v", result, err)
	}
	result, err = a.CommitForTest(context.Background(), token)
	if err != nil || result.Disposition != Committed {
		t.Fatalf("再有効化後の新しい許可で確定できません: %+v %v", result, err)
	}
}

func TestSQLiteDeletedSubjectRevokesAllPendingPermits(t *testing.T) {
	a, b, identity, req := setupSQLiteLegacyEpochs(t)
	auth := fixtureLegacyAdapter(identity, nil)
	token, result, err := a.PrepareWithVersionedAuthorization(context.Background(), req.Scope.Subject, req, auth, fixtureAuthoritySource)
	if err != nil || result.Disposition != Prepared {
		t.Fatalf("準備に失敗: %+v %v", result, err)
	}
	if err := b.exec("DELETE FROM legacy_subjects WHERE subject=" + textSQL(req.Scope.Subject)); err != nil {
		t.Fatal(err)
	}
	if version, allowed := sqliteMembershipEpoch(t, b, req.Scope); version != "2" || allowed != "0" {
		t.Fatalf("削除後の所属版が失効していません: %s %s", version, allowed)
	}
	result, err = a.CommitForTest(context.Background(), token)
	if err != nil || result.Disposition != Denied {
		t.Fatalf("削除済み主体の確定: %+v %v", result, err)
	}
	if _, result, err = a.PrepareWithVersionedAuthorization(context.Background(), req.Scope.Subject, req, auth, fixtureAuthoritySource); err == nil || result.Receipt != (Receipt{}) {
		t.Fatalf("削除後に古い結果が開示されました: %+v %v", result, err)
	}
}

func TestSQLiteOneSubjectRoleChangeRevisesEveryScope(t *testing.T) {
	a, b, identity, req := setupSQLiteLegacyEpochs(t)
	other := req.Scope
	other.Application = "inventory"
	other.Resource = "inventory.item"
	seedAuthorityDependencies(t, a, other)
	a.RequireAuthorityDependencies()
	auth1 := fixtureLegacyAdapter(identity, nil)
	auth2 := fixtureLegacyAdapter(sqliteLegacyAuthority{f: a, scope: other, source: fixtureAuthoritySource}, nil)
	req2 := Request{Scope: other, OperationID: "other-operation", Value: "second"}
	token1, result, err := a.PrepareWithVersionedAuthorization(context.Background(), req.Scope.Subject, req, auth1, fixtureAuthoritySource)
	if err != nil || result.Disposition != Prepared {
		t.Fatalf("最初の操作の準備: %+v %v", result, err)
	}
	token2, result, err := a.PrepareWithVersionedAuthorization(context.Background(), other.Subject, req2, auth2, fixtureAuthoritySource)
	if err != nil || result.Disposition != Prepared {
		t.Fatalf("別範囲の操作の準備: %+v %v", result, err)
	}
	if err := b.exec("UPDATE legacy_subjects SET role='VIEWER' WHERE subject=" + textSQL(req.Scope.Subject)); err != nil {
		t.Fatal(err)
	}
	for _, s := range []Scope{req.Scope, other} {
		if version, _ := sqliteMembershipEpoch(t, b, s); version != "2" {
			t.Fatalf("別範囲に版更新が及びません: %+v version=%s", s, version)
		}
	}
	for _, token := range []any{token1, token2} {
		result, err := a.CommitForTest(context.Background(), token)
		if err != nil || result.Disposition != Denied {
			t.Fatalf("別範囲の古い許可が確定しました: %+v %v", result, err)
		}
	}
}

func TestSQLiteLegacyMutationFailureRollsBackUserAndEpoch(t *testing.T) {
	a, b, _, req := setupSQLiteLegacyEpochs(t)
	// 版の更新障害が起きると、変更を起こした元のSQL文全体が失敗する。
	if err := a.exec("CREATE TRIGGER reject_membership_version BEFORE UPDATE ON authority_dependencies WHEN NEW.dimension='membership' BEGIN SELECT RAISE(ABORT,'injected epoch fault'); END"); err != nil {
		t.Fatal(err)
	}
	if err := b.exec("UPDATE legacy_subjects SET role='VIEWER' WHERE subject=" + textSQL(req.Scope.Subject)); err == nil {
		t.Fatal("版を更新できないのにRole変更が成功しました")
	}
	if role, _, found := sqliteLegacyUser(t, a, req.Scope.Subject); !found || role != "EDITOR" {
		t.Fatalf("失敗時にRole変更だけ残りました: role=%q found=%v", role, found)
	}
	if version, _ := sqliteMembershipEpoch(t, a, req.Scope); version != "1" {
		t.Fatalf("失敗時に版だけが変わりました: %s", version)
	}
}

func TestSQLiteMissingEpochAndSubjectRekeyAreRejected(t *testing.T) {
	a, b, _, req := setupSQLiteLegacyEpochs(t)
	if err := b.exec("INSERT INTO legacy_subjects(subject,role,disabled) VALUES('unversioned','EDITOR',0)"); err == nil {
		t.Fatal("版のない利用者の追加を許可しました")
	}
	if _, _, exists := sqliteLegacyUser(t, a, "unversioned"); exists {
		t.Fatal("版を持たない利用者が残りました")
	}
	if err := b.exec("UPDATE legacy_subjects SET subject='other-editor' WHERE subject=" + textSQL(req.Scope.Subject)); err == nil {
		t.Fatal("主体の識別子を改版なしで付け替えました")
	}
	if role, _, exists := sqliteLegacyUser(t, a, req.Scope.Subject); !exists || role != "EDITOR" {
		t.Fatalf("不正な主体変更で元の利用者が失われました: role=%q", role)
	}
	if err := a.SetAuthorityDependency(req.Scope, "membership", math.MaxInt64, true); err != nil {
		t.Fatal(err)
	}
	if err := b.exec("UPDATE legacy_subjects SET disabled=1 WHERE subject=" + textSQL(req.Scope.Subject)); err == nil {
		t.Fatal("版の上限到達後の更新が成功しました")
	}
	if _, disabled, exists := sqliteLegacyUser(t, a, req.Scope.Subject); !exists || disabled != "0" {
		t.Fatal("版の上限で失敗したにもかかわらず無効化が残りました")
	}
}

func TestSQLiteDirectRoleUpdateConcurrentWithCommit(t *testing.T) {
	for i := 0; i < 10; i++ {
		a, b, identity, req := setupSQLiteLegacyEpochs(t)
		auth := fixtureLegacyAdapter(identity, nil)
		token, result, err := a.PrepareWithVersionedAuthorization(context.Background(), req.Scope.Subject, req, auth, fixtureAuthoritySource)
		if err != nil || result.Disposition != Prepared {
			t.Fatalf("準備: %+v %v", result, err)
		}
		begin := make(chan struct{})
		var wg sync.WaitGroup
		var changed, committed error
		var outcome Result
		wg.Add(2)
		go func() {
			defer wg.Done()
			<-begin
			outcome, committed = a.CommitForTest(context.Background(), token)
		}()
		go func() {
			defer wg.Done()
			<-begin
			b.mu.Lock()
			changed = b.exec("UPDATE legacy_subjects SET role='VIEWER' WHERE subject=" + textSQL(req.Scope.Subject))
			b.mu.Unlock()
		}()
		close(begin)
		wg.Wait()
		if changed != nil || committed != nil {
			t.Fatalf("並列確定の障害: role=%v commit=%v", changed, committed)
		}
		_, version := a.Observe(req.Scope)
		if (outcome.Disposition == Committed && version != 1) ||
			(outcome.Disposition == Denied && version != 0) ||
			(outcome.Disposition != Committed && outcome.Disposition != Denied) {
			t.Fatalf("失効との直列化に反する結果: %+v version=%d", outcome, version)
		}
	}
}
