package database

import (
	"testing"

	"tei/internal/database/sqlgen/postgres"
	"tei/internal/database/sqlgen/sqlite"
	"tei/internal/environment"
)

func TestDialectRegistry(t *testing.T) {
	registry := NewDialectRegistry()

	// Test default providers are registered
	if _, err := registry.Get(environment.ProviderPostgres); err != nil {
		t.Errorf("PostgreSQL adapter not registered by default: %v", err)
	}
	if _, err := registry.Get(environment.ProviderSQLite); err != nil {
		t.Errorf("SQLite adapter not registered by default: %v", err)
	}
	if _, err := registry.Get(environment.ProviderOracle); err == nil {
		t.Error("Oracle adapter should not be registered by default")
	}

	// Test custom provider registration
	customAdapter := postgres.NewDialect()
	registry.Register("custom-db", customAdapter)
	if _, err := registry.Get("custom-db"); err != nil {
		t.Errorf("Custom adapter not registered: %v", err)
	}

	// Test providers list
	providers := registry.Providers()
	if len(providers) != 3 {
		t.Errorf("Expected 3 providers, got %d", len(providers))
	}

	// Test PostgreSQL adapter type
	adapter, err := registry.Get(environment.ProviderPostgres)
	if err != nil {
		t.Fatalf("Failed to get PostgreSQL adapter: %v", err)
	}
	if _, ok := adapter.(*postgres.Dialect); !ok {
		t.Errorf("Expected *postgres.Dialect, got %T", adapter)
	}

	// Test SQLite adapter type
	adapter, err = registry.Get(environment.ProviderSQLite)
	if err != nil {
		t.Fatalf("Failed to get SQLite adapter: %v", err)
	}
	if _, ok := adapter.(*sqlite.Dialect); !ok {
		t.Errorf("Expected *sqlite.Dialect, got %T", adapter)
	}
}

func TestNewDialectRegistry(t *testing.T) {
	registry := NewDialectRegistry()
	if registry == nil {
		t.Error("NewDialectRegistry returned nil")
	}
	if len(registry.adapters) == 0 {
		t.Error("NewDialectRegistry created empty registry")
	}
}
