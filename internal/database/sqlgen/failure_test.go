package sqlgen

import (
	"errors"
	"strings"
	"testing"

	"tei/internal/failure"
)

func TestToDatabaseFailure(t *testing.T) {
	tests := []struct {
		name        string
		provider    string
		err         error
		wantCode    string
		wantKind    failure.Kind
		wantRetry   failure.RetryPolicy
		wantEffect  bool
		wantPreserve bool
	}{
		{
			name:        "nil error",
			provider:    "postgres",
			err:         nil,
			wantCode:    "",
			wantKind:    "",
			wantRetry:   "",
			wantEffect:  false,
			wantPreserve: false,
		},
		{
			name:        "unique violation",
			provider:    "postgres",
			err:         errors.New("pq: duplicate key value violates unique constraint \"orders_id_key\""),
			wantCode:    "POSTGRES_UNIQUE_VIOLATION",
			wantKind:    failure.BusinessRejection,
			wantRetry:   failure.NotRetryable,
			wantEffect:  false,
			wantPreserve: true,
		},
		{
			name:        "foreign key violation",
			provider:    "sqlite",
			err:         errors.New("FOREIGN KEY constraint failed"),
			wantCode:    "SQLITE_FOREIGN_KEY_VIOLATION",
			wantKind:    failure.BusinessRejection,
			wantRetry:   failure.NotRetryable,
			wantEffect:  false,
			wantPreserve: true,
		},
		{
			name:        "not null violation",
			provider:    "postgres",
			err:         errors.New("pq: null value in column \"amount\" violates not-null constraint"),
			wantCode:    "POSTGRES_NOT_NULL_VIOLATION",
			wantKind:    failure.BusinessRejection,
			wantRetry:   failure.NotRetryable,
			wantEffect:  false,
			wantPreserve: true,
		},
		{
			name:        "check violation",
			provider:    "sqlite",
			err:         errors.New("CHECK constraint failed: amount > 0"),
			wantCode:    "SQLITE_CHECK_VIOLATION",
			wantKind:    failure.BusinessRejection,
			wantRetry:   failure.NotRetryable,
			wantEffect:  false,
			wantPreserve: true,
		},
		{
			name:        "timeout",
			provider:    "postgres",
			err:         errors.New("pq: timeout expired"),
			wantCode:    "POSTGRES_DATABASE_TIMEOUT",
			wantKind:    failure.TechnicalFailure,
			wantRetry:   failure.Retryable,
			wantEffect:  true,
			wantPreserve: true,
		},
		{
			name:        "connection error",
			provider:    "sqlite",
			err:         errors.New("sqlite: unable to open database file"),
			wantCode:    "SQLITE_DATABASE_UNAVAILABLE",
			wantKind:    failure.TechnicalFailure,
			wantRetry:   failure.Retryable,
			wantEffect:  true,
			wantPreserve: true,
		},
		{
			name:        "generic error",
			provider:    "postgres",
			err:         errors.New("some unknown database error"),
			wantCode:    "POSTGRES_DATABASE_ERROR",
			wantKind:    failure.TechnicalFailure,
			wantRetry:   failure.LookupFirst,
			wantEffect:  true,
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
