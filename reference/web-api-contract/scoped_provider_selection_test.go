package webapicontract

import (
	"context"
	"errors"
	"testing"

	"tei/internal/application"
	"tei/internal/driver"
	postgresquery "tei/internal/driver/queryruntime/postgres"
	"tei/internal/query"
)

type nilScopedProvider struct{}

func (*nilScopedProvider) ExecuteScopedBrowseQuery(
	context.Context, query.EmbeddedSQL, application.DatabaseRequirement,
	ReadPlan, map[string]any,
) (driver.BrowseQueryResult, error) {
	panic("nil provider must never execute")
}

func TestLegacyPostgresExecutorCannotBeSelectedAsScopedProvider(t *testing.T) {
	legacy := postgresquery.Executor{}
	candidate, err := RequireScopedCanonicalExecutor(legacy)
	if !errors.Is(err, ErrScopedQueryUnavailable) || candidate != nil {
		t.Fatalf("native SQL executor bypassed scoped Provider boundary: %T %v", candidate, err)
	}
}

func TestScopedProviderSelectionRejectsTypedNil(t *testing.T) {
	var typedNil *nilScopedProvider
	candidate, err := RequireScopedCanonicalExecutor(typedNil)
	if !errors.Is(err, ErrScopedQueryUnavailable) || candidate != nil {
		t.Fatalf("typed-nil Provider was accepted: %T %v", candidate, err)
	}
	pub := scopedPublication(t)
	plan, err := pub.BindRead(nil, TrustedReadContext{
		PrincipalID: "subject-order-editor", RowScopeID: "tenant-a-only", PolicyRevision: "r17",
	})
	if err != nil {
		t.Fatal(err)
	}
	out, err := ExecuteScopedCanonicalRead(context.Background(), bridgeWorld(t),
		pub, plan, scopedSupport(pub, plan), typedNil, nil)
	if !errors.Is(err, ErrScopedQueryUnavailable) || len(out.Columns) != 0 {
		t.Fatalf("typed-nil Provider reached Driver: result=%+v err=%v", out, err)
	}
}

func TestScopedProviderSelectorAcceptsExplicitImplementationOnly(t *testing.T) {
	provider := scopedCanonicalExecutorFunc(func(
		context.Context, query.EmbeddedSQL, application.DatabaseRequirement,
		ReadPlan, map[string]any,
	) (driver.BrowseQueryResult, error) {
		return driver.BrowseQueryResult{}, nil
	})
	chosen, err := RequireScopedCanonicalExecutor(provider)
	if err != nil || chosen == nil {
		t.Fatalf("explicit scoped Provider rejected: %v", err)
	}
}
