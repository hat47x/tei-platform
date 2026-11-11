package database

import (
	"context"
	"database/sql"

	"tei/internal/database/sqlgen"
	"tei/internal/environment"
)

// SampleMySQLPlugin is a sample plugin that demonstrates how to add a new database
// provider (MySQL) without modifying existing APIs. This shows the extensibility
// of the plugin system.
type SampleMySQLPlugin struct {
	dialect sqlgen.Dialect
}

// NewSampleMySQLPlugin creates a new sample MySQL plugin.
func NewSampleMySQLPlugin() *SampleMySQLPlugin {
	return &SampleMySQLPlugin{}
}

// Register registers the MySQL dialect adapter with the plugin registry.
// This demonstrates how a new database provider can be added at runtime.
func (p *SampleMySQLPlugin) Register(registry *DialectRegistry) {
	// In a real implementation, this would use a MySQL-specific dialect adapter
	// that implements the sqlgen.Dialect interface. For demonstration purposes,
	// we're using a placeholder that shows the registration pattern.
	registry.Register(environment.ProviderMySQL, p.dialect)
}

// ExecuteQuery demonstrates executing a query using the registered MySQL dialect.
// This shows how the plugin system allows new databases to integrate without
// changing existing code.
func (p *SampleMySQLPlugin) ExecuteQuery(ctx context.Context, db *sql.DB, query string) (*sql.Rows, error) {
	// In a real implementation, this would use the MySQL dialect to:
	// 1. Parse the query
	// 2. Apply MySQL-specific transformations
	// 3. Execute the query
	// 4. Return results
	return db.QueryContext(ctx, query)
}

// Example usage of the plugin system:
//
// 1. Create a new dialect adapter for MySQL that implements sqlgen.Dialect
// 2. Register it with the DialectRegistry
// 3. The rest of the system can now use MySQL without any changes to existing code
//
// This demonstrates the open/closed principle: the system is open for extension
// (new database providers) but closed for modification (existing code doesn't change).
