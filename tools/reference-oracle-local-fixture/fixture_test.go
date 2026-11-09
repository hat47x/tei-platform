package main

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"
)

// fakeAdminは、Oracleへ接続せずに管理接続の動きを再現する試験用のドライバーである。
type fakeAdmin struct {
	mu sync.Mutex
	// existing は、すでに存在するユーザー名である。
	existing map[string]bool
	// failCreate は、CREATE USERを失敗させるユーザー名と、そのエラー文である。
	failCreate map[string]string
	executed   []string
}

func (a *fakeAdmin) record(statement string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.executed = append(a.executed, statement)
}

func (a *fakeAdmin) count(prefix string) int {
	a.mu.Lock()
	defer a.mu.Unlock()
	n := 0
	for _, s := range a.executed {
		if strings.HasPrefix(s, prefix) {
			n++
		}
	}
	return n
}

func (a *fakeAdmin) drops() []string {
	a.mu.Lock()
	defer a.mu.Unlock()
	var out []string
	for _, s := range a.executed {
		if strings.HasPrefix(s, "DROP USER") {
			out = append(out, s)
		}
	}
	return out
}

func (a *fakeAdmin) open() *sql.DB { return sql.OpenDB(adminFakeConnector{a}) }

type adminFakeConnector struct{ admin *fakeAdmin }

func (c adminFakeConnector) Connect(context.Context) (driver.Conn, error) {
	return adminFakeConn{c.admin}, nil
}
func (c adminFakeConnector) Driver() driver.Driver { return nil }

type adminFakeConn struct{ admin *fakeAdmin }

func (adminFakeConn) Prepare(string) (driver.Stmt, error) { return nil, errors.New("未対応") }
func (adminFakeConn) Close() error                        { return nil }
func (adminFakeConn) Begin() (driver.Tx, error)           { return nil, errors.New("未対応") }

func (c adminFakeConn) ExecContext(_ context.Context, query string, _ []driver.NamedValue) (driver.Result, error) {
	c.admin.record(query)
	if strings.HasPrefix(query, "CREATE USER") {
		for name, message := range c.admin.failCreate {
			if strings.Contains(query, `"`+name+`"`) {
				return nil, errors.New(message)
			}
		}
	}
	return driver.RowsAffected(0), nil
}

func (c adminFakeConn) QueryContext(_ context.Context, query string, args []driver.NamedValue) (driver.Rows, error) {
	switch {
	case strings.Contains(query, "FROM dba_users WHERE username = :u"):
		count := int64(0)
		if len(args) == 1 {
			if name, ok := args[0].Value.(string); ok && c.admin.existing[name] {
				count = 1
			}
		}
		return &adminFakeRows{columns: []string{"COUNT"}, values: []driver.Value{count}}, nil
	case strings.Contains(query, "database_properties"):
		return &adminFakeRows{columns: []string{"V"}, values: []driver.Value{"USERS"}}, nil
	default:
		// 残存確認のCOUNT。
		return &adminFakeRows{columns: []string{"COUNT"}, values: []driver.Value{int64(0)}}, nil
	}
}

type adminFakeRows struct {
	columns []string
	values  []driver.Value
	done    bool
}

func (r *adminFakeRows) Columns() []string { return r.columns }
func (r *adminFakeRows) Close() error      { return nil }
func (r *adminFakeRows) Next(dest []driver.Value) error {
	if r.done {
		return io.EOF
	}
	r.done = true
	copy(dest, r.values)
	return nil
}

func newTestFixture(t *testing.T) *fixture {
	t.Helper()
	f, err := newFixture(adminTarget{Host: "127.0.0.1", Port: 1521, Service: "S"}, "0123ABCD")
	if err != nil {
		t.Fatal(err)
	}
	f.ownerPassword, f.readOnlyPassword, f.selectControlPassword = "pwOwnerAAAA", "pwReadAAAA", "pwSelectAAAA"
	return f
}

func TestCreateUsersStopsOnNameCollisionAndTouchesNothing(t *testing.T) {
	for _, existing := range []func(*fixture) string{
		func(f *fixture) string { return f.Owner },
		func(f *fixture) string { return f.ReadOnly },
		func(f *fixture) string { return f.SelectControl },
	} {
		f := newTestFixture(t)
		admin := &fakeAdmin{existing: map[string]bool{existing(f): true}}
		db := admin.open()
		if err := f.createUsers(context.Background(), db, &redactor{}); err == nil {
			t.Fatal("名前が衝突しているのに作成が続行されました")
		}
		if n := admin.count("CREATE USER"); n != 0 {
			t.Fatalf("衝突時にCREATE USERが%d回実行されました", n)
		}
		if len(f.created) != 0 {
			t.Fatalf("作成していないユーザーが記録されています: %v", f.created)
		}
		result := f.teardown(context.Background(), db, &redactor{})
		if drops := admin.drops(); len(drops) != 0 {
			t.Fatalf("作成していないユーザーが削除されました: %v", drops)
		}
		if len(result.DroppedUsers) != 0 {
			t.Fatalf("削除したと記録されています: %v", result.DroppedUsers)
		}
		db.Close()
	}
}

func TestTeardownDropsOnlyUsersCreatedByThisRunAfterRaceFailure(t *testing.T) {
	f := newTestFixture(t)
	// 衝突確認の後に、別のセッションが読取り専用ユーザーを先に作った状況（ORA-01920）を再現する。
	admin := &fakeAdmin{failCreate: map[string]string{f.ReadOnly: "ORA-01920: user name conflicts with another user or role name"}}
	db := admin.open()
	defer db.Close()
	if err := f.createUsers(context.Background(), db, &redactor{}); err == nil {
		t.Fatal("CREATE USERの失敗が報告されていません")
	}
	if len(f.created) != 1 || f.created[0] != f.Owner {
		t.Fatalf("作成に成功した所有者だけが記録されるべきです: %v", f.created)
	}
	f.teardown(context.Background(), db, &redactor{})
	drops := admin.drops()
	if len(drops) != 1 || !strings.Contains(drops[0], f.Owner) {
		t.Fatalf("削除対象が作成済みの所有者だけになっていません: %v", drops)
	}
	for _, drop := range drops {
		if strings.Contains(drop, f.ReadOnly) || strings.Contains(drop, f.SelectControl) {
			t.Fatalf("作成していないユーザーを削除しようとしました: %s", drop)
		}
	}
}

func TestTeardownDropsAllThreeAfterSuccessfulCreation(t *testing.T) {
	f := newTestFixture(t)
	admin := &fakeAdmin{}
	db := admin.open()
	defer db.Close()
	if err := f.createUsers(context.Background(), db, &redactor{}); err != nil {
		t.Fatal(err)
	}
	if got := admin.count("CREATE USER"); got != 3 || len(f.created) != 3 {
		t.Fatalf("created=%v CREATE USER=%d", f.created, got)
	}
	result := f.teardown(context.Background(), db, &redactor{})
	if len(admin.drops()) != 3 || !result.Verified {
		t.Fatalf("drops=%v verified=%t errors=%v", admin.drops(), result.Verified, result.Errors)
	}
}

func TestTeardownRefusesNamesOutsideFixtureFormat(t *testing.T) {
	f := newTestFixture(t)
	f.created = []string{"SYSTEM", "LEMORADM", "TEI_REF_0123ABCD_OWN2"}
	admin := &fakeAdmin{}
	db := admin.open()
	defer db.Close()
	result := f.teardown(context.Background(), db, &redactor{})
	if drops := admin.drops(); len(drops) != 0 {
		t.Fatalf("形式に合わない名前のユーザーが削除されました: %v", drops)
	}
	if len(result.Errors) != 3 {
		t.Fatalf("拒否が記録されていません: %v", result.Errors)
	}
}

func TestAdoptExistingForCleanupTargetsOnlyExistingRunUsers(t *testing.T) {
	f := newTestFixture(t)
	admin := &fakeAdmin{existing: map[string]bool{f.Owner: true, f.SelectControl: true}}
	db := admin.open()
	defer db.Close()
	if err := f.adoptExistingForCleanup(context.Background(), db, &redactor{}); err != nil {
		t.Fatal(err)
	}
	f.teardown(context.Background(), db, &redactor{})
	drops := admin.drops()
	if len(drops) != 2 {
		t.Fatalf("存在する二つだけが対象になるべきです: %v", drops)
	}
	for _, drop := range drops {
		if strings.Contains(drop, f.ReadOnly) {
			t.Fatalf("存在しないユーザーが対象になっています: %s", drop)
		}
	}
}
