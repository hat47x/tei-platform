package main

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"io"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
)

var fixtureOwnershipTestDriverSeq atomic.Uint64

type fixtureOwnershipDBState struct {
	mu     sync.Mutex
	events []string
	marker string
}

func (s *fixtureOwnershipDBState) add(event string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = append(s.events, event)
}

func (s *fixtureOwnershipDBState) snapshot() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.events...)
}

func (s *fixtureOwnershipDBState) resetEvents() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events = nil
}

func (s *fixtureOwnershipDBState) setMarker(marker string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.marker = marker
}

func (s *fixtureOwnershipDBState) getMarker() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.marker
}

type fixtureOwnershipDriver struct{ state *fixtureOwnershipDBState }

type fixtureOwnershipConn struct{ state *fixtureOwnershipDBState }

type fixtureOwnershipTx struct{ state *fixtureOwnershipDBState }

type fixtureOwnershipRows struct {
	marker string
	done   bool
}

func (d fixtureOwnershipDriver) Open(string) (driver.Conn, error) {
	return &fixtureOwnershipConn{state: d.state}, nil
}

func (c *fixtureOwnershipConn) Prepare(string) (driver.Stmt, error) {
	return nil, fmt.Errorf("prepare is not supported in fixture ownership test driver")
}

func (c *fixtureOwnershipConn) Close() error { return nil }

func (c *fixtureOwnershipConn) Begin() (driver.Tx, error) {
	c.state.add("begin")
	return &fixtureOwnershipTx{state: c.state}, nil
}

func (c *fixtureOwnershipConn) BeginTx(context.Context, driver.TxOptions) (driver.Tx, error) {
	return c.Begin()
}

func (c *fixtureOwnershipConn) ExecContext(_ context.Context, query string, _ []driver.NamedValue) (driver.Result, error) {
	switch {
	case strings.Contains(query, "CREATE TABLE public.order_lines"):
		c.state.add("create")
	case strings.HasPrefix(strings.TrimSpace(query), "COMMENT ON TABLE public.order_lines"):
		c.state.add("comment")
		first := strings.Index(query, "'")
		last := strings.LastIndex(query, "'")
		if first < 0 || last <= first {
			return nil, fmt.Errorf("ownership comment marker not found")
		}
		c.state.setMarker(query[first+1 : last])
	case strings.Contains(query, "LOCK TABLE public.order_lines IN ACCESS EXCLUSIVE MODE"):
		c.state.add("lock")
	case strings.Contains(query, "DROP TABLE public.order_lines"):
		c.state.add("drop")
		c.state.setMarker("")
	default:
		return nil, fmt.Errorf("unexpected exec query: %s", query)
	}
	return driver.RowsAffected(1), nil
}

func (c *fixtureOwnershipConn) QueryContext(_ context.Context, query string, _ []driver.NamedValue) (driver.Rows, error) {
	if !strings.Contains(query, "obj_description") {
		return nil, fmt.Errorf("unexpected query: %s", query)
	}
	c.state.add("marker")
	return &fixtureOwnershipRows{marker: c.state.getMarker()}, nil
}

func (tx *fixtureOwnershipTx) Commit() error {
	tx.state.add("commit")
	return nil
}

func (tx *fixtureOwnershipTx) Rollback() error {
	tx.state.add("rollback")
	return nil
}

func (r *fixtureOwnershipRows) Columns() []string { return []string{"obj_description"} }
func (r *fixtureOwnershipRows) Close() error      { return nil }
func (r *fixtureOwnershipRows) Next(dest []driver.Value) error {
	if r.done {
		return io.EOF
	}
	r.done = true
	if r.marker == "" {
		dest[0] = nil
	} else {
		dest[0] = r.marker
	}
	return nil
}

func openFixtureOwnershipTestDB(t *testing.T) (*sql.DB, *fixtureOwnershipDBState) {
	t.Helper()
	state := &fixtureOwnershipDBState{}
	name := fmt.Sprintf("fixture-ownership-%d", fixtureOwnershipTestDriverSeq.Add(1))
	sql.Register(name, fixtureOwnershipDriver{state: state})
	db, err := sql.Open(name, "")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return db, state
}

func TestCreateAndRemoveOwnedFixtureUseProtectedTransactionOrder(t *testing.T) {
	db, state := openFixtureOwnershipTestDB(t)
	ctx := context.Background()

	marker, err := createOwnedFixtureTable(ctx, db)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := strings.Join(state.snapshot(), ","), "begin,create,comment,commit"; got != want {
		t.Fatalf("create events=%q want=%q", got, want)
	}

	state.resetEvents()
	if err := removeOwnedFixtureTable(ctx, db, marker); err != nil {
		t.Fatal(err)
	}
	if got, want := strings.Join(state.snapshot(), ","), "begin,lock,marker,drop,commit"; got != want {
		t.Fatalf("cleanup events=%q want=%q", got, want)
	}
}

func TestRemoveOwnedFixtureDoesNotDropMarkerMismatch(t *testing.T) {
	db, state := openFixtureOwnershipTestDB(t)
	ctx := context.Background()
	const expected = fixtureOwnershipPrefix + "0123456789abcdef0123456789abcdef"
	state.setMarker(fixtureOwnershipPrefix + "fedcba9876543210fedcba9876543210")

	if err := removeOwnedFixtureTable(ctx, db, expected); err == nil {
		t.Fatal("marker mismatchを受理しました")
	}
	if got, want := strings.Join(state.snapshot(), ","), "begin,lock,marker,rollback"; got != want {
		t.Fatalf("mismatch events=%q want=%q", got, want)
	}
	for _, event := range state.snapshot() {
		if event == "drop" {
			t.Fatal("marker mismatch時にDROPしました")
		}
	}
}
