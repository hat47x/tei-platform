package provider

import (
	"context"
	"fmt"

	"tei/internal/database/physical"
	"tei/internal/environment"
)

func Introspect(ctx context.Context, p Provider, instance environment.DatabaseInstance) (physical.SchemaSnapshot, error) {
	if p.ID() != instance.Provider {
		return physical.SchemaSnapshot{}, fmt.Errorf("database provider mismatch: instance %s requires %s, provider is %s", instance.ID, instance.Provider, p.ID())
	}
	caps, err := p.Capabilities(ctx, instance)
	if err != nil {
		return physical.SchemaSnapshot{}, err
	}
	if !caps.Has(CapabilityMetadataRead) {
		return physical.SchemaSnapshot{}, fmt.Errorf("database provider %s does not expose metadata-read", p.ID())
	}
	s, err := p.Introspect(ctx, instance)
	if err != nil {
		return physical.SchemaSnapshot{}, err
	}
	if s.Database != instance.Definition {
		return physical.SchemaSnapshot{}, fmt.Errorf("introspection snapshot database %s does not match instance definition %s", s.Database, instance.Definition)
	}
	return s, nil
}
