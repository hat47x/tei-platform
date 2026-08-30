package provider

import (
	"context"

	"tei/internal/database/physical"
	"tei/internal/environment"
)

type Capability string

const (
	CapabilityMetadataRead   Capability = "metadata-read"
	CapabilityQuery          Capability = "query"
	CapabilityDataMutation   Capability = "data-mutation"
	CapabilitySchemaChange   Capability = "schema-change"
	CapabilityExplain        Capability = "explain"
	CapabilityExplainAnalyze Capability = "explain-analyze"
	CapabilityTransaction    Capability = "transaction"
	CapabilityCancel         Capability = "cancel"
)

type Capabilities map[Capability]bool

func (c Capabilities) Has(v Capability) bool { return c[v] }

// Provider is the small Database Workbench SPI. It intentionally does not
// expose user/role administration, backup, replication, or server-management
// operations. Those remain outside TEI's application-development DB surface.
type Provider interface {
	ID() environment.DatabaseProvider
	Capabilities(context.Context, environment.DatabaseInstance) (Capabilities, error)
	Introspect(context.Context, environment.DatabaseInstance) (physical.SchemaSnapshot, error)
}
