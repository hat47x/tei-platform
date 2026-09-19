package postgres

import (
	"context"
	"fmt"

	"tei/internal/database/physical"
	"tei/internal/environment"
)

// RelationIdentity identifies one PostgreSQL relation without leaking catalog OIDs
// into the provider-neutral physical schema.
type RelationIdentity struct {
	Schema string `json:"schema"`
	Name   string `json:"name"`
}

// PartitionRelation records one provider-owned child -> parent partition edge.
// A relation may appear as both Child and Parent when PostgreSQL uses sub-partitioning.
type PartitionRelation struct {
	Child  RelationIdentity `json:"child"`
	Parent RelationIdentity `json:"parent"`
}

// PartitionCatalogReader is an optional extension to CatalogReader. Keeping it
// separate lets existing CatalogReader implementations remain source-compatible.
type PartitionCatalogReader interface {
	Partitions(context.Context) ([]PartitionRelation, error)
}

// SchemaObservation keeps the full PostgreSQL physical observation together
// with provider-owned realization facts. Snapshot intentionally still contains
// partition children; ProjectForConvergence decides which relations participate
// in the generic comparison.
//
// PartitionObservationComplete distinguishes an observed empty partition set
// from a CatalogReader that cannot observe partition relationships at all. That
// distinction is required before destructive DDL can treat "no child observed"
// as a safety precondition.
type SchemaObservation struct {
	Snapshot                     physical.SchemaSnapshot `json:"snapshot"`
	Partitions                   []PartitionRelation     `json:"partitions,omitempty"`
	PartitionObservationComplete bool                    `json:"partitionObservationComplete"`
}

// ObserveSchema preserves the existing Introspect contract while adding
// PostgreSQL-specific partition facts when the configured catalog can provide
// them. PartitionObservationComplete is true only after the partition reader
// has returned successfully, including when the complete result is an empty set.
func (p Provider) ObserveSchema(ctx context.Context, instance environment.DatabaseInstance) (SchemaObservation, error) {
	snapshot, err := p.Introspect(ctx, instance)
	if err != nil {
		return SchemaObservation{}, err
	}
	observation := SchemaObservation{Snapshot: snapshot}
	partitionCatalog, ok := p.Catalog.(PartitionCatalogReader)
	if !ok {
		return observation, nil
	}
	partitions, err := partitionCatalog.Partitions(ctx)
	if err != nil {
		return SchemaObservation{}, fmt.Errorf("postgres partitions: %w", err)
	}
	observation.Partitions = append([]PartitionRelation(nil), partitions...)
	observation.PartitionObservationComplete = true
	return observation, nil
}

// ProjectForConvergence removes known partition children from the generic
// actual snapshot without erasing them from SchemaObservation. If desired
// declares the same qualified name as a standalone generic table, the current
// contract cannot prove that both objects have the same semantic role, so the
// projection fails closed instead of creating, dropping, or equating them.
func (o SchemaObservation) ProjectForConvergence(desired physical.SchemaSnapshot) (physical.SchemaSnapshot, error) {
	desired = desired.Normalize()
	actual := o.Snapshot.Normalize()

	children := make(map[string]PartitionRelation, len(o.Partitions))
	for _, edge := range o.Partitions {
		childKey := key(edge.Child.Schema, edge.Child.Name)
		if prior, exists := children[childKey]; exists {
			if prior.Parent != edge.Parent {
				return physical.SchemaSnapshot{}, fmt.Errorf("postgres partition child %s has multiple observed parents", childKey)
			}
			continue
		}
		children[childKey] = edge
	}

	for _, table := range desired.Tables {
		if _, isPartitionChild := children[key(table.Schema, table.Name)]; isPartitionChild {
			return physical.SchemaSnapshot{}, fmt.Errorf("desired standalone table %s conflicts with observed PostgreSQL partition child", key(table.Schema, table.Name))
		}
	}

	projected := actual
	projected.Tables = make([]physical.Table, 0, len(actual.Tables))
	for _, table := range actual.Tables {
		if _, isPartitionChild := children[key(table.Schema, table.Name)]; isPartitionChild {
			continue
		}
		projected.Tables = append(projected.Tables, table)
	}
	return projected.Normalize(), nil
}
