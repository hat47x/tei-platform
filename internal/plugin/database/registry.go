package database

import (
	"errors"
	"sync"

	"tei/internal/database/sqlgen/postgres"
	"tei/internal/database/sqlgen/sqlite"
	"tei/internal/environment"
)

// DialectRegistry maps database providers to their SQL dialect adapters.
type DialectRegistry struct {
	mu       sync.RWMutex
	adapters map[environment.DatabaseProvider]interface{}
}

// NewDialectRegistry creates a new dialect registry with default providers.
func NewDialectRegistry() *DialectRegistry {
	r := &DialectRegistry{
		adapters: make(map[environment.DatabaseProvider]interface{}),
	}
	r.Register(environment.ProviderPostgres, postgres.NewDialect())
	r.Register(environment.ProviderSQLite, sqlite.NewDialect())
	return r
}

// Register adds a dialect adapter for a provider.
func (r *DialectRegistry) Register(provider environment.DatabaseProvider, adapter interface{}) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.adapters[provider] = adapter
}

// Get returns the dialect adapter for the given provider.
func (r *DialectRegistry) Get(provider environment.DatabaseProvider) (interface{}, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	adapter, ok := r.adapters[provider]
	if !ok {
		return nil, errors.New("no dialect adapter registered for provider: " + string(provider))
	}
	return adapter, nil
}

// Providers returns the list of registered providers.
func (r *DialectRegistry) Providers() []environment.DatabaseProvider {
	r.mu.RLock()
	defer r.mu.RUnlock()
	providers := make([]environment.DatabaseProvider, 0, len(r.adapters))
	for p := range r.adapters {
		providers = append(providers, p)
	}
	return providers
}
