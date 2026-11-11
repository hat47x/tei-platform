package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"

	"tei/internal/database/sqlgen"
	"tei/internal/environment"
	"tei/internal/failure"
)

// mockDialect is a test-only dialect adapter that implements the sqlgen.Dialect interface.
// It demonstrates that any new database provider can be added by implementing the interface.
type mockDialect struct {
	name string
}

func (m *mockDialect) PhysicalType(storage sqlgen.StorageType) (string, error) {
	return m.name + "_" + string(storage), nil
}

func (m *mockDialect) CanStore(storage sqlgen.StorageType) bool {
	return true
}

func (m *mockDialect) PlaceholderStyle() sqlgen.PlaceholderStyle {
	return "mock"
}

func (m *mockDialect) Name() string {
	return m.name
}

func (m *mockDialect) BeginTransaction(ctx context.Context, db *sql.DB) (*sql.Tx, error) {
	return nil, nil
}

func (m *mockDialect) IsUniqueViolation(err error) bool {
	return false
}

func (m *mockDialect) IsForeignKeyViolation(err error) bool {
	return false
}

func (m *mockDialect) IsNullViolation(err error) bool {
	return false
}

func (m *mockDialect) IsCheckViolation(err error) bool {
	return false
}

// TestExtensibility verifies that a new database provider can be added without
// modifying existing code. This is the open/closed principle in action.
func TestExtensibility(t *testing.T) {
	// 1. Create a new registry
	registry := NewDialectRegistry()

	// 2. Create a new dialect adapter for a hypothetical new database
	newProvider := environment.DatabaseProvider("newdb")
	newDialect := &mockDialect{name: "newdb"}

	// 3. Register the new dialect - this is the only step needed to add a new database
	registry.Register(newProvider, newDialect)

	// 4. Verify the new provider works
	adapter, err := registry.Get(newProvider)
	if err != nil {
		t.Fatalf("Failed to get new provider adapter: %v", err)
	}

	// 5. Verify the adapter implements the expected interface
	if _, ok := adapter.(*mockDialect); !ok {
		t.Errorf("Expected *mockDialect, got %T", adapter)
	}

	// 6. Verify existing providers still work
	if _, err := registry.Get(environment.ProviderPostgres); err != nil {
		t.Errorf("PostgreSQL adapter broken after adding new provider: %v", err)
	}
	if _, err := registry.Get(environment.ProviderSQLite); err != nil {
		t.Errorf("SQLite adapter broken after adding new provider: %v", err)
	}
}

// TestPluginIsolation verifies that plugins are isolated from each other.
func TestPluginIsolation(t *testing.T) {
	registry1 := NewDialectRegistry()
	registry2 := NewDialectRegistry()

	// Register a custom provider only in registry1
	customProvider := environment.DatabaseProvider("custom")
	registry1.Register(customProvider, &mockDialect{name: "custom"})

	// Verify registry2 doesn't have the custom provider
	if _, err := registry2.Get(customProvider); err == nil {
		t.Error("registry2 should not have custom provider")
	}

	// Verify registry1 still has the custom provider
	if _, err := registry1.Get(customProvider); err != nil {
		t.Errorf("registry1 should have custom provider: %v", err)
	}
}

// TestErrorPreservation verifies that database errors are preserved when wrapped.
func TestErrorPreservation(t *testing.T) {
	originalErr := errors.New("original database error")
	wrapped := sqlgen.ToDatabaseFailure("postgres", originalErr)

	if wrapped.FailureCause() != originalErr {
		t.Errorf("FailureCause() = %v, want %v", wrapped.FailureCause(), originalErr)
	}

	// Verify the error message contains the original error
	if !contains(wrapped.Error(), originalErr.Error()) {
		t.Errorf("Error message %v does not contain original error %v", wrapped.Error(), originalErr.Error())
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// TestConcurrentAccess verifies thread-safe access to the registry.
func TestConcurrentAccess(t *testing.T) {
	registry := NewDialectRegistry()
	done := make(chan bool)

	// Concurrent reads
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				registry.Get(environment.ProviderPostgres)
				registry.Get(environment.ProviderSQLite)
			}
			done <- true
		}()
	}

	// Concurrent writes
	for i := 0; i < 5; i++ {
		go func(id int) {
			for j := 0; j < 100; j++ {
				registry.Register(environment.DatabaseProvider(fmt.Sprintf("provider-%d-%d", id, j)), &mockDialect{})
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines to finish
	for i := 0; i < 15; i++ {
		<-done
	}
}

// TestFailureKindClassification verifies that database errors are classified correctly.
func TestFailureKindClassification(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		provider string
		wantKind failure.Kind
	}{
		{
			name:     "unique violation is business rejection",
			err:      errors.New("unique constraint violated"),
			provider: "postgres",
			wantKind: failure.BusinessRejection,
		},
		{
			name:     "connection error is technical failure",
			err:      errors.New("connection refused"),
			provider: "sqlite",
			wantKind: failure.TechnicalFailure,
		},
		{
			name:     "timeout is technical failure",
			err:      errors.New("query timeout"),
			provider: "postgres",
			wantKind: failure.TechnicalFailure,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := sqlgen.ToDatabaseFailure(tt.provider, tt.err)
			if f == nil {
				t.Fatal("ToDatabaseFailure returned nil")
			}
			if f.Kind() != tt.wantKind {
				t.Errorf("Kind() = %v, want %v", f.Kind(), tt.wantKind)
			}
		})
	}
}
