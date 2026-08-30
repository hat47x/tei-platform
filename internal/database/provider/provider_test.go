package provider_test

import (
	"context"
	"testing"

	"tei/internal/core"
	"tei/internal/database/physical"
	"tei/internal/database/provider"
	"tei/internal/environment"
)

type fakeProvider struct {
	id   environment.DatabaseProvider
	caps provider.Capabilities
	snap physical.SchemaSnapshot
}

func (f fakeProvider) ID() environment.DatabaseProvider { return f.id }
func (f fakeProvider) Capabilities(context.Context, environment.DatabaseInstance) (provider.Capabilities, error) {
	return f.caps, nil
}
func (f fakeProvider) Introspect(context.Context, environment.DatabaseInstance) (physical.SchemaSnapshot, error) {
	return f.snap, nil
}

func TestIntrospectionIsProviderAndDatabaseScoped(t *testing.T) {
	inst := environment.DatabaseInstance{ID: core.MustCanonicalID("sales_core.dev"), Definition: core.MustCanonicalID("sales_core"), Provider: environment.ProviderPostgres}
	p := fakeProvider{id: environment.ProviderPostgres, caps: provider.Capabilities{provider.CapabilityMetadataRead: true}, snap: physical.SchemaSnapshot{Database: core.MustCanonicalID("sales_core")}}
	if _, err := provider.Introspect(context.Background(), p, inst); err != nil {
		t.Fatal(err)
	}
}

func TestIntrospectionRejectsProviderMismatch(t *testing.T) {
	inst := environment.DatabaseInstance{ID: core.MustCanonicalID("sales_core.dev"), Definition: core.MustCanonicalID("sales_core"), Provider: environment.ProviderPostgres}
	p := fakeProvider{id: environment.ProviderOracle, caps: provider.Capabilities{provider.CapabilityMetadataRead: true}}
	if _, err := provider.Introspect(context.Background(), p, inst); err == nil {
		t.Fatal("expected provider mismatch")
	}
}

func TestIntrospectionRequiresMetadataCapability(t *testing.T) {
	inst := environment.DatabaseInstance{ID: core.MustCanonicalID("sales_core.dev"), Definition: core.MustCanonicalID("sales_core"), Provider: environment.ProviderPostgres}
	p := fakeProvider{id: environment.ProviderPostgres, caps: provider.Capabilities{}}
	if _, err := provider.Introspect(context.Background(), p, inst); err == nil {
		t.Fatal("expected metadata capability error")
	}
}
