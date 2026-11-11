package sqlite

import (
	"database/sql"
	"errors"
	"testing"

	"modernc.org/sqlite"

	"tei/internal/failure"

	_ "modernc.org/sqlite"
)

// openConstraintDB opens a table with a NOT NULL column and a primary key, so a
// bad insert produces the driver's real SQLITE_CONSTRAINT error rather than a
// hand-built stand-in. modernc's sqlite.Error keeps its code unexported, so the
// only honest way to obtain one is to make SQLite report it.
func openConstraintDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.Exec("CREATE TABLE orders (id INTEGER PRIMARY KEY, amount REAL NOT NULL)"); err != nil {
		t.Fatalf("create table: %v", err)
	}
	return db
}

func constraintError(t *testing.T, db *sql.DB, statement string) error {
	t.Helper()
	_, err := db.Exec(statement)
	if err == nil {
		t.Fatalf("%q succeeded, want a constraint error", statement)
	}
	var driverErr *sqlite.Error
	if !errors.As(err, &driverErr) {
		t.Fatalf("%q returned %T, want *sqlite.Error", statement, err)
	}
	return err
}

// The primary result code is what SQLite agrees on across versions, so a
// constraint refusal is a business rejection whatever subtype it is.
func TestFailureOfClassifiesConstraintByResultCode(t *testing.T) {
	db := openConstraintDB(t)
	// The first insert establishes the row the second one collides with.
	if _, err := db.Exec("INSERT INTO orders (id, amount) VALUES (1, 10)"); err != nil {
		t.Fatalf("seed insert: %v", err)
	}
	for _, statement := range []string{
		"INSERT INTO orders (id, amount) VALUES (1, NULL)", // NOT NULL
		"INSERT INTO orders (id, amount) VALUES (1, 20)",   // duplicate key
	} {
		err := constraintError(t, db, statement)
		converted := FailureOf(err)
		got, ok := failure.AsFailure(converted)
		if !ok {
			t.Fatalf("%q: FailureOf did not produce a failure.Failure: %T", statement, err)
		}
		if got.FailureCode() != "SQLITE_CONSTRAINT_VIOLATION" {
			t.Errorf("%q: FailureCode() = %q, want SQLITE_CONSTRAINT_VIOLATION", statement, got.FailureCode())
		}
		if got.Kind() != failure.BusinessRejection {
			t.Errorf("%q: Kind() = %v, want %v", statement, got.Kind(), failure.BusinessRejection)
		}
		if got.Retryable() != failure.NotRetryable {
			t.Errorf("%q: Retryable() = %v, want %v", statement, got.Retryable(), failure.NotRetryable)
		}
		if got.HasEffectUnknown() {
			t.Errorf("%q: a refused constraint was reported as having an unknown effect", statement)
		}
		// The developer-facing cause survives the conversion.
		if !errors.Is(converted, err) {
			t.Errorf("%q: the driver error is no longer reachable from the failure", statement)
		}
	}
}

// An error that carries no result code is left alone: the caller decides whether
// to fall back to message matching, instead of that choice being made here.
func TestFailureOfLeavesErrorWithoutResultCode(t *testing.T) {
	plain := errors.New("sqlite: unable to open database file")
	if got := FailureOf(plain); got != plain {
		t.Fatalf("FailureOf(%v) = %v, want the error unchanged", plain, got)
	}
	if got := FailureOf(nil); got != nil {
		t.Fatalf("FailureOf(nil) = %v, want nil", got)
	}
}

// The extended result code puts the subtype in the high byte. Masking to the
// primary code is what keeps the classification stable across SQLite versions,
// so the carrier must report 19 for every constraint subtype.
func TestFailureOfMasksExtendedResultCode(t *testing.T) {
	const (
		primaryConstraint = 19         // SQLITE_CONSTRAINT
		extendedNotnull   = 19 | 11<<8 // CONSTRAINT_NOTNULL
	)
	carrier := &sqliteCodedError{code: extendedNotnull, err: errors.New("NOT NULL constraint failed")}
	if got := carrier.DatabaseCode(); got != "19" {
		t.Fatalf("DatabaseCode() = %q, want 19", got)
	}
	if carrier.DatabaseProvider() != "sqlite" {
		t.Errorf("DatabaseProvider() = %q, want sqlite", carrier.DatabaseProvider())
	}
	// The driver's own message survives, so the developer still sees which
	// constraint was violated.
	if carrier.Error() != "NOT NULL constraint failed" {
		t.Errorf("Error() = %q", carrier.Error())
	}
	if !errors.Is(carrier, carrier.err) {
		t.Error("the underlying driver error is no longer reachable")
	}
	if extendedNotnull&0xff != primaryConstraint {
		t.Errorf("masking %d lost the primary code", extendedNotnull)
	}
}

// A coded error keeps the driver's message for the developer while adding the
// machine-readable code for the caller.
func TestFailureOfPreservesDriverMessage(t *testing.T) {
	db := openConstraintDB(t)
	err := constraintError(t, db, "INSERT INTO orders (id, amount) VALUES (1, NULL)")
	got, ok := failure.AsFailure(FailureOf(err))
	if !ok {
		t.Fatalf("not a failure.Failure: %T", err)
	}
	if got.FailureMessage() == "" {
		t.Error("FailureMessage() is empty")
	}
	if got.FailureCause() == nil {
		t.Error("FailureCause() is nil, so the developer loses the driver diagnostic")
	}
}
