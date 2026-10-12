//go:build tei_sqlite_reference && cgo

package effectfence

/*
#cgo LDFLAGS: -lsqlite3
#include <stdlib.h>
#include <sqlite3.h>
*/
import "C"

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"sync"
	"unsafe"
)

// SQLite is used only by this opt-in integration test. It is not a TEI runtime provider.
// All rights and resource versions must live in the same SQLite database for the
// transaction below to protect against revocation; a remote identity store does not qualify.
type sqliteFixture struct {
	mu      sync.Mutex
	db      *C.sqlite3
	next    uint64
	pending map[uint64]prepared
}

func openSQLiteFixture(path string) (*sqliteFixture, error) {
	cpath := C.CString(path)
	defer C.free(unsafe.Pointer(cpath))
	var db *C.sqlite3
	rc := C.sqlite3_open_v2(cpath, &db, C.SQLITE_OPEN_READWRITE|C.SQLITE_OPEN_CREATE|C.SQLITE_OPEN_FULLMUTEX, nil)
	if rc != C.SQLITE_OK {
		if db != nil {
			C.sqlite3_close(db)
		}
		return nil, fmt.Errorf("SQLite open: %d", int(rc))
	}
	f := &sqliteFixture{db: db, pending: map[uint64]prepared{}}
	C.sqlite3_busy_timeout(db, 5000)
	if err := f.exec("PRAGMA journal_mode=WAL;"); err != nil {
		f.Close()
		return nil, err
	}
	if err := f.exec("PRAGMA synchronous=FULL;"); err != nil {
		f.Close()
		return nil, err
	}
	for _, stmt := range []string{
		"CREATE TABLE IF NOT EXISTS authority (scope TEXT PRIMARY KEY, version INTEGER NOT NULL, allowed INTEGER NOT NULL)",
		"CREATE TABLE IF NOT EXISTS resources (resource TEXT PRIMARY KEY, version INTEGER NOT NULL, value TEXT NOT NULL)",
		"CREATE TABLE IF NOT EXISTS committed (operation TEXT PRIMARY KEY, request TEXT NOT NULL, application TEXT NOT NULL, resource TEXT NOT NULL, version INTEGER NOT NULL)",
	} {
		if err := f.exec(stmt); err != nil {
			f.Close()
			return nil, err
		}
	}
	return f, nil
}

func (f *sqliteFixture) Close() {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.db != nil {
		C.sqlite3_close(f.db)
		f.db = nil
	}
}
func (f *sqliteFixture) exec(q string) error {
	sql := C.CString(q)
	defer C.free(unsafe.Pointer(sql))
	rc := C.sqlite3_exec(f.db, sql, nil, nil, nil)
	if rc != C.SQLITE_OK {
		return fmt.Errorf("SQLite SQL (%d): %s", int(rc), C.GoString(C.sqlite3_errmsg(f.db)))
	}
	return nil
}
func (f *sqliteFixture) query(q string) ([]string, bool, error) {
	sql := C.CString(q)
	defer C.free(unsafe.Pointer(sql))
	var stmt *C.sqlite3_stmt
	rc := C.sqlite3_prepare_v2(f.db, sql, -1, &stmt, nil)
	if rc != C.SQLITE_OK {
		return nil, false, fmt.Errorf("SQLite prepare (%d): %s", int(rc), C.GoString(C.sqlite3_errmsg(f.db)))
	}
	defer C.sqlite3_finalize(stmt)
	rc = C.sqlite3_step(stmt)
	if rc == C.SQLITE_DONE {
		return nil, false, nil
	}
	if rc != C.SQLITE_ROW {
		return nil, false, fmt.Errorf("SQLite step (%d): %s", int(rc), C.GoString(C.sqlite3_errmsg(f.db)))
	}
	columns := int(C.sqlite3_column_count(stmt))
	result := make([]string, columns)
	for i := 0; i < columns; i++ {
		n := C.int(i)
		if C.sqlite3_column_type(stmt, n) == C.SQLITE_NULL {
			continue
		}
		size := C.sqlite3_column_bytes(stmt, n)
		result[i] = string(C.GoBytes(unsafe.Pointer(C.sqlite3_column_text(stmt, n)), size))
	}
	return result, true, nil
}
func textSQL(s string) string { return fmt.Sprintf("CAST(X'%x' AS TEXT)", []byte(s)) }
func scopeSQL(s Scope) string { b, _ := json.Marshal(s); return textSQL(string(b)) }
func resourceSQL(s Scope) string {
	b, _ := json.Marshal([2]string{s.Application, s.Resource})
	return textSQL(string(b))
}
func requestSQL(r Request) string           { b, _ := json.Marshal(r); return textSQL(string(b)) }
func parseVersion(s string) (uint64, error) { return strconv.ParseUint(s, 10, 63) }

func (f *sqliteFixture) transaction(fn func() (Result, error)) (result Result, err error) {
	if err = f.exec("BEGIN IMMEDIATE"); err != nil {
		return result, err
	}
	defer func() {
		if err != nil {
			_ = f.exec("ROLLBACK")
		}
	}()
	result, err = fn()
	if err != nil {
		return result, err
	}
	if err = f.exec("COMMIT"); err != nil {
		_ = f.exec("ROLLBACK")
		return Result{}, err
	}
	return result, nil
}
func (f *sqliteFixture) SetAuthority(s Scope, version uint64, allowed bool) error {
	if !s.valid() || version == 0 || version > math.MaxInt64 {
		return errors.New("invalid version/scope")
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	_, err := f.transaction(func() (Result, error) {
		row, ok, err := f.query("SELECT version FROM authority WHERE scope=" + scopeSQL(s))
		if err != nil {
			return Result{}, err
		}
		if ok {
			prior, err := parseVersion(row[0])
			if err != nil {
				return Result{}, err
			}
			if version <= prior {
				return Result{}, errors.New("nonmonotone authority version")
			}
		}
		allow := 0
		if allowed {
			allow = 1
		}
		sql := fmt.Sprintf("INSERT INTO authority(scope,version,allowed) VALUES(%s,%d,%d) ON CONFLICT(scope) DO UPDATE SET version=excluded.version,allowed=excluded.allowed", scopeSQL(s), version, allow)
		return Result{}, f.exec(sql)
	})
	return err
}
func (f *sqliteFixture) lookupCommitted(r Request) (Result, bool, error) {
	row, ok, err := f.query("SELECT request,application,resource,version FROM committed WHERE operation=" + textSQL(r.OperationID))
	if err != nil || !ok {
		return Result{}, ok, err
	}
	if row[0] != mustJSON(r) {
		return Result{Disposition: Denied, Reason: "operation-id-conflict"}, true, nil
	}
	version, err := parseVersion(row[3])
	if err != nil {
		return Result{}, true, err
	}
	return Result{Disposition: Replayed, Receipt: Receipt{OperationID: r.OperationID, Application: row[1], Resource: row[2], ResourceVersion: version}}, true, nil
}
func mustJSON(v any) string { b, _ := json.Marshal(v); return string(b) }
func (f *sqliteFixture) PrepareForTest(ctx context.Context, r Request) (any, Result, error) {
	if err := ctx.Err(); err != nil {
		return nil, Result{}, err
	}
	if !r.Scope.valid() || !validToken(r.OperationID) {
		return nil, Result{Disposition: Denied, Reason: "invalid-request"}, nil
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	row, ok, err := f.query("SELECT version,allowed FROM authority WHERE scope=" + scopeSQL(r.Scope))
	if err != nil {
		return nil, Result{}, err
	}
	if !ok || row[1] != "1" {
		return nil, Result{Disposition: Denied, Reason: "authority-unavailable"}, nil
	}
	old, exists, err := f.lookupCommitted(r)
	if err != nil {
		return nil, Result{}, err
	}
	if exists {
		return nil, old, nil
	}
	version, err := parseVersion(row[0])
	if err != nil {
		return nil, Result{}, err
	}
	if f.next == math.MaxUint64 {
		return nil, Result{}, errors.New("permit sequence overflow")
	}
	f.next++
	token := Permit{id: f.next, owner: f}
	f.pending[token.id] = prepared{request: r, authorityVersion: version}
	return token, Result{Disposition: Prepared}, nil
}
func (f *sqliteFixture) CommitForTest(ctx context.Context, token any) (Result, error) {
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	p, ok := token.(Permit)
	if !ok || p.id == 0 || p.owner != f {
		return Result{Disposition: Denied, Reason: "unknown-permit"}, nil
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	entry, ok := f.pending[p.id]
	if !ok {
		return Result{Disposition: Denied, Reason: "unknown-permit"}, nil
	}
	// A retry after cancellation still has a valid permit; after attempted transaction it is consumed.
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	delete(f.pending, p.id)
	r := entry.request
	return f.transaction(func() (Result, error) {
		if err := ctx.Err(); err != nil {
			return Result{}, err
		}
		row, ok, err := f.query("SELECT version,allowed FROM authority WHERE scope=" + scopeSQL(r.Scope))
		if err != nil {
			return Result{}, err
		}
		if !ok || row[1] != "1" || row[0] != strconv.FormatUint(entry.authorityVersion, 10) {
			return Result{Disposition: Denied, Reason: "authority-changed"}, nil
		}
		if old, exists, err := f.lookupCommitted(r); err != nil {
			return Result{}, err
		} else if exists {
			return old, nil
		}
		resource := resourceSQL(r.Scope)
		row, ok, err = f.query("SELECT version FROM resources WHERE resource=" + resource)
		if err != nil {
			return Result{}, err
		}
		version := uint64(0)
		if ok {
			version, err = parseVersion(row[0])
			if err != nil {
				return Result{}, err
			}
		}
		if version != r.ExpectedResourceVersion {
			return Result{Disposition: Denied, Reason: "resource-version-changed"}, nil
		}
		if version == math.MaxInt64 {
			return Result{}, errors.New("resource version overflow")
		}
		next := version + 1
		if err = f.exec(fmt.Sprintf("INSERT INTO resources(resource,version,value) VALUES(%s,%d,%s) ON CONFLICT(resource) DO UPDATE SET version=excluded.version,value=excluded.value", resource, next, textSQL(r.Value))); err != nil {
			return Result{}, err
		}
		if err = f.exec(fmt.Sprintf("INSERT INTO committed(operation,request,application,resource,version) VALUES(%s,%s,%s,%s,%d)", textSQL(r.OperationID), requestSQL(r), textSQL(r.Scope.Application), textSQL(r.Scope.Resource), next)); err != nil {
			return Result{}, err
		}
		return Result{Disposition: Committed, Receipt: Receipt{OperationID: r.OperationID, Application: r.Scope.Application, Resource: r.Scope.Resource, ResourceVersion: next}}, nil
	})
}
func (f *sqliteFixture) Observe(s Scope) (string, uint64) {
	f.mu.Lock()
	defer f.mu.Unlock()
	row, ok, err := f.query("SELECT value,version FROM resources WHERE resource=" + resourceSQL(s))
	if err != nil || !ok {
		return "", 0
	}
	version, err := parseVersion(row[1])
	if err != nil {
		return "", 0
	}
	return row[0], version
}
func (f *sqliteFixture) Lookup(s Scope, opID string) (Receipt, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	row, ok, err := f.query("SELECT allowed FROM authority WHERE scope=" + scopeSQL(s))
	if err != nil || !ok || row[0] != "1" {
		return Receipt{}, false
	}
	row, ok, err = f.query("SELECT request,application,resource,version FROM committed WHERE operation=" + textSQL(opID))
	if err != nil || !ok {
		return Receipt{}, false
	}
	var req Request
	if err = json.Unmarshal([]byte(row[0]), &req); err != nil || req.Scope != s {
		return Receipt{}, false
	}
	version, err := parseVersion(row[3])
	if err != nil {
		return Receipt{}, false
	}
	return Receipt{OperationID: opID, Application: row[1], Resource: row[2], ResourceVersion: version}, true
}
func (f *sqliteFixture) AdvanceResource(s Scope, value string) error {
	if !s.valid() {
		return errors.New("invalid scope")
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	_, err := f.transaction(func() (Result, error) {
		key := resourceSQL(s)
		row, ok, err := f.query("SELECT version FROM resources WHERE resource=" + key)
		if err != nil {
			return Result{}, err
		}
		v := uint64(0)
		if ok {
			v, err = parseVersion(row[0])
			if err != nil {
				return Result{}, err
			}
		}
		if v == math.MaxInt64 {
			return Result{}, errors.New("version overflow")
		}
		return Result{}, f.exec(fmt.Sprintf("INSERT INTO resources(resource,version,value) VALUES(%s,%d,%s) ON CONFLICT(resource) DO UPDATE SET version=excluded.version,value=excluded.value", key, v+1, textSQL(value)))
	})
	return err
}
