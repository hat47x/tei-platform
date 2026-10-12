//go:build tei_sqlite_reference && cgo

package effectfence

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"testing"

	"tei/internal/security/legacyauth"
	"tei/internal/security/legacyauthorization"
)

const fixtureAuthoritySource = "sqlite-legacy-authority"

type sqliteLegacyAuthority struct {
	f      *sqliteFixture
	scope  Scope
	source string
}

func (a sqliteLegacyAuthority) FindBySubject(_ context.Context, subject string) (legacyauth.User, bool, error) {
	a.f.mu.Lock()
	defer a.f.mu.Unlock()
	row, found, err := a.f.query("SELECT role,disabled FROM legacy_subjects WHERE subject=" + textSQL(subject))
	if err != nil || !found {
		return legacyauth.User{}, found, err
	}
	return legacyauth.User{SubjectID: subject, RoleCode: row[0], Disabled: row[1] == "1"}, true, nil
}

func (a sqliteLegacyAuthority) ReadAuthorizationEpochs(_ context.Context, input legacyauthorization.Input) (legacyauthorization.AuthorityEpochs, error) {
	if input.SubjectID != a.scope.Subject || input.Action != a.scope.Action || input.Resource != a.scope.Resource {
		return legacyauthorization.AuthorityEpochs{}, errors.New("認可要求と版読取りの範囲が異なります")
	}
	a.f.mu.Lock()
	defer a.f.mu.Unlock()
	// 正本の欠落・破損と、完全に読み取れた正本の明示的な失効を区別する。
	// 3種類を同一SQLiteスナップショットから読み、失効だけを専用エラーで返す。
	var versions [3]uint64
	inactive := false
	_, err := a.f.transaction(func() (Result, error) {
		for i, dimension := range authorityDimensions {
			row, found, err := a.f.query("SELECT version,allowed FROM authority_dependencies WHERE scope=" +
				scopeSQL(a.scope) + " AND dimension=" + textSQL(dimension))
			if err != nil {
				return Result{}, err
			}
			if !found || len(row) != 2 {
				return Result{}, fmt.Errorf("認可根拠が欠落しています: %s", dimension)
			}
			version, err := parseVersion(row[0])
			if err != nil || version == 0 {
				return Result{}, fmt.Errorf("認可根拠の版が不正です: %s", dimension)
			}
			if row[1] != "0" && row[1] != "1" {
				return Result{}, fmt.Errorf("認可根拠の有効状態が不正です: %s", dimension)
			}
			versions[i] = version
			if row[1] == "0" {
				inactive = true
			}
		}
		return Result{}, nil
	})
	if err != nil {
		return legacyauthorization.AuthorityEpochs{}, err
	}
	if inactive {
		return legacyauthorization.AuthorityEpochs{}, fmt.Errorf("認可根拠が明示的に失効しています: %w", legacyauthorization.ErrAuthorityInactive)
	}
	return legacyauthorization.AuthorityEpochs{
		Source:      a.source,
		SubjectLink: versions[0],
		Membership:  versions[1],
		Policy:      versions[2],
	}, nil
}

func fixtureLegacyAdapter(a sqliteLegacyAuthority, hook func()) legacyauthorization.Adapter {
	return legacyauthorization.Adapter{
		Store: a, EpochReader: a,
		Rule: legacyRuleFunc(func(_ context.Context, u legacyauth.User, action, resource string) (bool, string, error) {
			if hook != nil {
				hook()
			}
			return u.RoleCode == "EDITOR" && action == a.scope.Action && resource == a.scope.Resource, "not_authorized", nil
		}),
	}
}

func setupSQLiteLegacyEpochs(t *testing.T) (*sqliteFixture, *sqliteFixture, sqliteLegacyAuthority, Request) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "legacy-epochs.sqlite")
	a, err := openSQLiteFixture(path)
	if err != nil {
		t.Fatal(err)
	}
	b, err := openSQLiteFixture(path)
	if err != nil {
		a.Close()
		t.Fatal(err)
	}
	t.Cleanup(a.Close)
	t.Cleanup(b.Close)
	s := epochScope()
	seedAuthorityDependencies(t, a, s)
	if err := a.exec("CREATE TABLE IF NOT EXISTS legacy_subjects(subject TEXT PRIMARY KEY, role TEXT NOT NULL, disabled INTEGER NOT NULL)"); err != nil {
		t.Fatal(err)
	}
	if err := a.exec("INSERT INTO legacy_subjects(subject,role,disabled) VALUES(" + textSQL(s.Subject) + ",'EDITOR',0)"); err != nil {
		t.Fatal(err)
	}
	if err := installSQLiteLegacyRoleEpochTriggers(a); err != nil {
		t.Fatal(err)
	}
	return a, b, sqliteLegacyAuthority{f: a, scope: s, source: fixtureAuthoritySource}, Request{Scope: s, OperationID: "versioned-operation", Value: "authorized"}
}

// 役割だけを更新しても、DBトリガーが所属版を同じ確定境界で改版する。
func changeSQLiteLegacyRole(f *sqliteFixture, s Scope, version uint64, role string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	_, err := f.transaction(func() (Result, error) {
		if err := f.exec("UPDATE legacy_subjects SET role=" + textSQL(role) + " WHERE subject=" + textSQL(s.Subject)); err != nil {
			return Result{}, err
		}
		row, exists, err := f.query("SELECT version FROM authority_dependencies WHERE scope=" + scopeSQL(s) + " AND dimension=" + textSQL("membership"))
		if err != nil {
			return Result{}, err
		}
		if !exists || row[0] != fmt.Sprint(version) {
			return Result{}, fmt.Errorf("DBトリガーによる所属版更新が確認できません: %+v", row)
		}
		return Result{}, nil
	})
	return err
}

func TestSQLiteVersionedLegacyGrantCommitsAndReplays(t *testing.T) {
	a, _, identity, req := setupSQLiteLegacyEpochs(t)
	auth := fixtureLegacyAdapter(identity, nil)
	token, result, err := a.PrepareWithVersionedAuthorization(context.Background(), req.Scope.Subject, req, auth, fixtureAuthoritySource)
	if err != nil || result.Disposition != Prepared {
		t.Fatalf("prepare: %+v %v", result, err)
	}
	result, err = a.CommitForTest(context.Background(), token)
	if err != nil || result.Disposition != Committed {
		t.Fatalf("commit: %+v %v", result, err)
	}
	_, result, err = a.PrepareWithVersionedAuthorization(context.Background(), req.Scope.Subject, req, auth, fixtureAuthoritySource)
	if err != nil || result.Disposition != Replayed || result.Receipt.ResourceVersion != 1 {
		t.Fatalf("replay: %+v %v", result, err)
	}
	if value, version := a.Observe(req.Scope); value != req.Value || version != 1 {
		t.Fatalf("effect duplicated: %q %d", value, version)
	}
}

func TestSQLiteVersionedGrantRejectsRoleRevokedBeforeCommit(t *testing.T) {
	a, b, identity, req := setupSQLiteLegacyEpochs(t)
	auth := fixtureLegacyAdapter(identity, nil)
	token, result, err := a.PrepareWithVersionedAuthorization(context.Background(), req.Scope.Subject, req, auth, fixtureAuthoritySource)
	if err != nil || result.Disposition != Prepared {
		t.Fatalf("prepare: %+v %v", result, err)
	}
	if err := changeSQLiteLegacyRole(b, req.Scope, 2, "VIEWER"); err != nil {
		t.Fatal(err)
	}
	result, err = a.CommitForTest(context.Background(), token)
	if err != nil || result.Disposition != Denied || result.Reason != "authority-dependency-changed" || result.Receipt != (Receipt{}) {
		t.Fatalf("revoked commit: %+v %v", result, err)
	}
	_, result, err = a.PrepareWithVersionedAuthorization(context.Background(), req.Scope.Subject, req, auth, fixtureAuthoritySource)
	if err != nil || result.Disposition != Denied {
		t.Fatalf("revoked subject got new permit: %+v %v", result, err)
	}
	if _, version := b.Observe(req.Scope); version != 0 {
		t.Fatalf("revoked write had effect: %d", version)
	}
}

func TestSQLiteVersionedGrantRejectsChangeDuringRoleEvaluation(t *testing.T) {
	a, b, identity, req := setupSQLiteLegacyEpochs(t)
	auth := fixtureLegacyAdapter(identity, func() {
		if err := changeSQLiteLegacyRole(b, req.Scope, 2, "VIEWER"); err != nil {
			t.Fatal(err)
		}
	})
	_, result, err := a.PrepareWithVersionedAuthorization(context.Background(), req.Scope.Subject, req, auth, fixtureAuthoritySource)
	if err != nil || result.Disposition != Denied || result.Reason != "authorization-denied" {
		t.Fatalf("role evaluated across version change: %+v %v", result, err)
	}
	if _, version := a.Observe(req.Scope); version != 0 {
		t.Fatal("version drift wrote data")
	}
}

func TestSQLiteVersionedGrantRequiresSameTrustedSourceAndAllEpochs(t *testing.T) {
	a, _, identity, req := setupSQLiteLegacyEpochs(t)
	for _, tc := range []struct {
		name   string
		auth   legacyauthorization.Adapter
		source string
	}{
		{"wrong-source", fixtureLegacyAdapter(identity, nil), "other-authority"},
		{"legacy-without-epochs", legacyauthorization.Adapter{
			Store: identity,
			Rule: legacyRuleFunc(func(context.Context, legacyauth.User, string, string) (bool, string, error) {
				return true, "", nil
			}),
		}, fixtureAuthoritySource},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, result, err := a.PrepareWithVersionedAuthorization(context.Background(), req.Scope.Subject, req, tc.auth, tc.source)
			if err == nil || result.Disposition != "" {
				t.Fatalf("untrusted grant admitted: %+v %v", result, err)
			}
		})
	}
	if _, version := a.Observe(req.Scope); version != 0 {
		t.Fatal("invalid source produced effect")
	}
}

func TestSQLiteVersionedGrantDoesNotDiscloseReceiptAfterPolicyRevocation(t *testing.T) {
	a, b, identity, req := setupSQLiteLegacyEpochs(t)
	auth := fixtureLegacyAdapter(identity, nil)
	token, result, err := a.PrepareWithVersionedAuthorization(context.Background(), req.Scope.Subject, req, auth, fixtureAuthoritySource)
	if err != nil || result.Disposition != Prepared {
		t.Fatalf("prepare: %+v %v", result, err)
	}
	result, err = a.CommitForTest(context.Background(), token)
	if err != nil || result.Disposition != Committed {
		t.Fatalf("commit: %+v %v", result, err)
	}
	if err := b.SetAuthorityDependency(req.Scope, "policy", 2, false); err != nil {
		t.Fatal(err)
	}
	_, result, err = a.PrepareWithVersionedAuthorization(context.Background(), req.Scope.Subject, req, auth, fixtureAuthoritySource)
	if err == nil || result.Receipt != (Receipt{}) || result.Disposition != "" {
		t.Fatalf("revoked policy disclosed receipt: %+v %v", result, err)
	}
	if _, ok := a.Lookup(req.Scope, req.OperationID); ok {
		t.Fatal("revoked receipt disclosed via lookup")
	}
}
