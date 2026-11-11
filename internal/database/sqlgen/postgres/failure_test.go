package postgres

import (
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"

	"tei/internal/failure"
)

// A SQLSTATE beats the message text. A localized server may report 23505 with a
// message that contains none of the English words the fallback matches, and the
// caller still has to see a unique violation rather than an unknown technical
// failure.
func TestFailureOfClassifiesBySQLState(t *testing.T) {
	driverErr := &pgconn.PgError{Code: "23505", Message: "llave duplicada"}
	converted := FailureOf(driverErr)
	got, ok := failure.AsFailure(converted)
	if !ok {
		t.Fatalf("FailureOf did not produce a failure.Failure: %T", converted)
	}
	if got.FailureCode() != "POSTGRES_UNIQUE_VIOLATION" {
		t.Errorf("FailureCode() = %q, want POSTGRES_UNIQUE_VIOLATION", got.FailureCode())
	}
	if got.Kind() != failure.BusinessRejection {
		t.Errorf("Kind() = %v, want %v", got.Kind(), failure.BusinessRejection)
	}
	if got.Retryable() != failure.NotRetryable {
		t.Errorf("Retryable() = %v, want %v", got.Retryable(), failure.NotRetryable)
	}
	// The developer-facing cause survives the conversion.
	if !errors.Is(converted, driverErr) {
		t.Error("the driver error is no longer reachable from the failure")
	}
}

// A concurrency failure (serialization, deadlock, lock timeout) aborts the
// transaction, but an earlier statement in it may already have applied. The
// policy must therefore be lookup_first, not a blind retry.
func TestFailureOfConcurrencyFailureNeedsLookupFirst(t *testing.T) {
	for _, state := range []string{"40001", "40P01", "55P03", "57014", "08006"} {
		got, ok := failure.AsFailure(FailureOf(&pgconn.PgError{Code: state, Message: "concurrency"}))
		if !ok {
			t.Fatalf("%s: not a failure.Failure", state)
		}
		if got.Kind() != failure.TechnicalFailure {
			t.Errorf("%s: Kind() = %v, want %v", state, got.Kind(), failure.TechnicalFailure)
		}
		if got.Retryable() != failure.LookupFirst {
			t.Errorf("%s: Retryable() = %v, want %v", state, got.Retryable(), failure.LookupFirst)
		}
		if !got.HasEffectUnknown() {
			t.Errorf("%s: HasEffectUnknown() = false", state)
		}
	}
}

// An error that carries no SQLSTATE is left alone: the caller decides whether
// to fall back to message matching, instead of that choice being made here.
func TestFailureOfLeavesErrorWithoutSQLState(t *testing.T) {
	plain := errors.New("dial tcp: connection refused")
	if got := FailureOf(plain); got != plain {
		t.Fatalf("FailureOf(%v) = %v, want the error unchanged", plain, got)
	}
	if got := FailureOf(nil); got != nil {
		t.Fatalf("FailureOf(nil) = %v, want nil", got)
	}
}
