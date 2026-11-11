package sqlgen

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"tei/internal/failure"
)

func TestToDatabaseFailure(t *testing.T) {
	tests := []struct {
		name         string
		provider     string
		err          error
		wantCode     string
		wantKind     failure.Kind
		wantRetry    failure.RetryPolicy
		wantEffect   bool
		wantPreserve bool
	}{
		{
			name:         "nil error",
			provider:     "postgres",
			err:          nil,
			wantCode:     "",
			wantKind:     "",
			wantRetry:    "",
			wantEffect:   false,
			wantPreserve: false,
		},
		{
			name:         "unique violation",
			provider:     "postgres",
			err:          errors.New("pq: duplicate key value violates unique constraint \"orders_id_key\""),
			wantCode:     "POSTGRES_UNIQUE_VIOLATION",
			wantKind:     failure.BusinessRejection,
			wantRetry:    failure.NotRetryable,
			wantEffect:   false,
			wantPreserve: true,
		},
		{
			name:         "foreign key violation",
			provider:     "sqlite",
			err:          errors.New("FOREIGN KEY constraint failed"),
			wantCode:     "SQLITE_FOREIGN_KEY_VIOLATION",
			wantKind:     failure.BusinessRejection,
			wantRetry:    failure.NotRetryable,
			wantEffect:   false,
			wantPreserve: true,
		},
		{
			name:         "not null violation",
			provider:     "postgres",
			err:          errors.New("pq: null value in column \"amount\" violates not-null constraint"),
			wantCode:     "POSTGRES_NOT_NULL_VIOLATION",
			wantKind:     failure.BusinessRejection,
			wantRetry:    failure.NotRetryable,
			wantEffect:   false,
			wantPreserve: true,
		},
		{
			name:         "check violation",
			provider:     "sqlite",
			err:          errors.New("CHECK constraint failed: amount > 0"),
			wantCode:     "SQLITE_CHECK_VIOLATION",
			wantKind:     failure.BusinessRejection,
			wantRetry:    failure.NotRetryable,
			wantEffect:   false,
			wantPreserve: true,
		},
		{
			// The statement was aborted, but an earlier statement in the same
			// transaction may have applied, so the effect is unknown and the
			// retry has to look the result up first.
			name:         "timeout",
			provider:     "postgres",
			err:          errors.New("pq: timeout expired"),
			wantCode:     "POSTGRES_DATABASE_TIMEOUT",
			wantKind:     failure.TechnicalFailure,
			wantRetry:    failure.LookupFirst,
			wantEffect:   true,
			wantPreserve: true,
		},
		{
			name:         "connection error",
			provider:     "sqlite",
			err:          errors.New("sqlite: unable to open database file"),
			wantCode:     "SQLITE_DATABASE_UNAVAILABLE",
			wantKind:     failure.TechnicalFailure,
			wantRetry:    failure.LookupFirst,
			wantEffect:   true,
			wantPreserve: true,
		},
		{
			name:         "generic error",
			provider:     "postgres",
			err:          errors.New("some unknown database error"),
			wantCode:     "POSTGRES_DATABASE_ERROR",
			wantKind:     failure.TechnicalFailure,
			wantRetry:    failure.LookupFirst,
			wantEffect:   true,
			wantPreserve: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var result failure.Failure
			if tt.err != nil {
				result = ToDatabaseFailure(tt.provider, tt.err)
			}

			if tt.err == nil {
				if result != nil {
					t.Errorf("ToDatabaseFailure(nil) = %v, want nil", result)
				}
				return
			}

			if result == nil {
				t.Fatal("ToDatabaseFailure returned nil for non-nil error")
			}

			if result.FailureCode() != tt.wantCode {
				t.Errorf("FailureCode() = %v, want %v", result.FailureCode(), tt.wantCode)
			}
			if result.Kind() != tt.wantKind {
				t.Errorf("Kind() = %v, want %v", result.Kind(), tt.wantKind)
			}
			if result.Retryable() != tt.wantRetry {
				t.Errorf("Retryable() = %v, want %v", result.Retryable(), tt.wantRetry)
			}
			if result.HasEffectUnknown() != tt.wantEffect {
				t.Errorf("HasEffectUnknown() = %v, want %v", result.HasEffectUnknown(), tt.wantEffect)
			}
			if result.FailureCause() != tt.err {
				t.Errorf("FailureCause() = %v, want %v", result.FailureCause(), tt.err)
			}
			if !strings.Contains(result.Error(), tt.err.Error()) {
				t.Errorf("Error() = %v, want to contain original error %v", result.Error(), tt.err.Error())
			}
			// The cause must stay reachable through the standard library, or a
			// caller matching on a driver type stops matching after conversion.
			if !errors.Is(result, tt.err) {
				t.Error("errors.Is did not reach the original error through the failure")
			}
		})
	}
}

func TestAsDatabaseFailure(t *testing.T) {
	t.Run("database failure", func(t *testing.T) {
		original := errors.New("pq: duplicate key")
		f := ToDatabaseFailure("postgres", original)
		got, ok := AsDatabaseFailure(f)
		if !ok {
			t.Fatal("AsDatabaseFailure returned false for DatabaseFailure")
		}
		if got.FailureCode() != "POSTGRES_UNIQUE_VIOLATION" {
			t.Errorf("AsDatabaseFailure code = %v, want POSTGRES_UNIQUE_VIOLATION", got.FailureCode())
		}
	})

	t.Run("non-database failure", func(t *testing.T) {
		_, ok := AsDatabaseFailure(errors.New("generic error"))
		if ok {
			t.Error("AsDatabaseFailure returned true for non-DatabaseFailure")
		}
	})

	t.Run("nil error", func(t *testing.T) {
		_, ok := AsDatabaseFailure(nil)
		if ok {
			t.Error("AsDatabaseFailure returned true for nil")
		}
	})
}

func TestDatabaseFailureImplementsFailure(t *testing.T) {
	var f failure.Failure = DatabaseFailure{}
	if f == nil {
		t.Fatal("DatabaseFailure does not implement failure.Failure")
	}
}

// codedCarrier stands in for a driver error that exposes a machine-readable
// code, which is what pgconn.PgError and modernc sqlite's Error do.
type codedCarrier struct {
	provider string
	code     string
}

func (c codedCarrier) Error() string            { return "driver error " + c.code }
func (c codedCarrier) DatabaseProvider() string { return c.provider }
func (c codedCarrier) DatabaseCode() string     { return c.code }

// A typed code is preferred over the message, and it wins even when the message
// would say something else. A driver that reports SQLSTATE 23505 with a message
// that happens to contain "timeout" is still a unique violation.
func TestToDatabaseFailurePrefersTypedCode(t *testing.T) {
	err := fmt.Errorf("insert failed: %w", codedCarrier{provider: "postgres", code: "23505"})
	got := ToDatabaseFailure("postgres", err)
	if got.FailureCode() != "POSTGRES_UNIQUE_VIOLATION" {
		t.Fatalf("FailureCode() = %q, want POSTGRES_UNIQUE_VIOLATION", got.FailureCode())
	}
	if got.Kind() != failure.BusinessRejection {
		t.Errorf("Kind() = %v, want %v", got.Kind(), failure.BusinessRejection)
	}
	if got.HasEffectUnknown() {
		t.Error("a refused constraint was reported as having an unknown effect")
	}
}

func TestToDatabaseFailureClassifiesSQLState(t *testing.T) {
	cases := []struct {
		state      string
		wantCode   string
		wantKind   failure.Kind
		wantRetry  failure.RetryPolicy
		wantEffect bool
	}{
		{"23505", "POSTGRES_UNIQUE_VIOLATION", failure.BusinessRejection, failure.NotRetryable, false},
		{"23503", "POSTGRES_FOREIGN_KEY_VIOLATION", failure.BusinessRejection, failure.NotRetryable, false},
		{"23502", "POSTGRES_NOT_NULL_VIOLATION", failure.BusinessRejection, failure.NotRetryable, false},
		{"23514", "POSTGRES_CHECK_VIOLATION", failure.BusinessRejection, failure.NotRetryable, false},
		{"23P01", "POSTGRES_EXCLUSION_VIOLATION", failure.BusinessRejection, failure.NotRetryable, false},
		{"42501", "POSTGRES_INSUFFICIENT_PRIVILEGE", failure.BusinessRejection, failure.NotRetryable, false},
		{"40001", "POSTGRES_SERIALIZATION_FAILURE", failure.TechnicalFailure, failure.LookupFirst, true},
		{"40P01", "POSTGRES_DEADLOCK_DETECTED", failure.TechnicalFailure, failure.LookupFirst, true},
		{"55P03", "POSTGRES_LOCK_NOT_AVAILABLE", failure.TechnicalFailure, failure.LookupFirst, true},
		{"57014", "POSTGRES_QUERY_CANCELED", failure.TechnicalFailure, failure.LookupFirst, true},
		{"08006", "POSTGRES_DATABASE_UNAVAILABLE", failure.TechnicalFailure, failure.LookupFirst, true},
		{"53300", "POSTGRES_TOO_MANY_CONNECTIONS", failure.TechnicalFailure, failure.LookupFirst, true},
	}
	for _, tc := range cases {
		t.Run(tc.state, func(t *testing.T) {
			got := ToDatabaseFailure("postgres", codedCarrier{provider: "postgres", code: tc.state})
			if got.FailureCode() != tc.wantCode {
				t.Errorf("FailureCode() = %q, want %q", got.FailureCode(), tc.wantCode)
			}
			if got.Kind() != tc.wantKind {
				t.Errorf("Kind() = %v, want %v", got.Kind(), tc.wantKind)
			}
			if got.Retryable() != tc.wantRetry {
				t.Errorf("Retryable() = %v, want %v", got.Retryable(), tc.wantRetry)
			}
			if got.HasEffectUnknown() != tc.wantEffect {
				t.Errorf("HasEffectUnknown() = %v, want %v", got.HasEffectUnknown(), tc.wantEffect)
			}
		})
	}
}

func TestToDatabaseFailureClassifiesSQLiteResultCode(t *testing.T) {
	cases := []struct {
		code       string
		wantCode   string
		wantKind   failure.Kind
		wantEffect bool
	}{
		{"19", "SQLITE_CONSTRAINT_VIOLATION", failure.BusinessRejection, false},
		{"5", "SQLITE_DATABASE_BUSY", failure.TechnicalFailure, true},
		{"6", "SQLITE_DATABASE_LOCKED", failure.TechnicalFailure, true},
		{"14", "SQLITE_DATABASE_UNAVAILABLE", failure.TechnicalFailure, true},
	}
	for _, tc := range cases {
		t.Run(tc.code, func(t *testing.T) {
			got := ToDatabaseFailure("sqlite", codedCarrier{provider: "sqlite", code: tc.code})
			if got.FailureCode() != tc.wantCode {
				t.Errorf("FailureCode() = %q, want %q", got.FailureCode(), tc.wantCode)
			}
			if got.Kind() != tc.wantKind {
				t.Errorf("Kind() = %v, want %v", got.Kind(), tc.wantKind)
			}
			if got.HasEffectUnknown() != tc.wantEffect {
				t.Errorf("HasEffectUnknown() = %v, want %v", got.HasEffectUnknown(), tc.wantEffect)
			}
		})
	}
}

// An unknown SQLSTATE is not guessed from its digits: it stays technical and
// requires a lookup first, so an unrecognised failure is never reported as a
// permanent business rejection the caller would stop retrying.
func TestToDatabaseFailureUnknownCodeIsNotGuessed(t *testing.T) {
	got := ToDatabaseFailure("postgres", codedCarrier{provider: "postgres", code: "ZZ999"})
	if got.Kind() != failure.TechnicalFailure {
		t.Errorf("Kind() = %v, want %v", got.Kind(), failure.TechnicalFailure)
	}
	if got.Retryable() != failure.LookupFirst {
		t.Errorf("Retryable() = %v, want %v", got.Retryable(), failure.LookupFirst)
	}
}

// The contract's own rule: an effect-unknown failure is never blindly
// retryable. Deriving the policy instead of storing it is what keeps this true
// for every code, including ones added after this test.
func TestDatabaseFailureEffectUnknownIsNeverBlindlyRetryable(t *testing.T) {
	for _, c := range []databaseClassification{
		{"X", failure.TechnicalFailure, true, true},
		{"X", failure.TechnicalFailure, false, true},
		{"X", failure.BusinessRejection, true, true},
	} {
		f := newDatabaseFailure("postgres", c, errors.New("cause"))
		if f.Retryable() == failure.Retryable {
			t.Errorf("effect-unknown failure %+v reported retryable", c)
		}
		if f.Retryable() != failure.LookupFirst {
			t.Errorf("effect-unknown failure %+v reported %v, want lookup_first", c, f.Retryable())
		}
	}
}
