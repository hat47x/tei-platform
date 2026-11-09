package inputprofile

import (
	"context"
	"errors"
	"testing"

	"tei/internal/core"
	"tei/internal/plugin"
	"tei/internal/plugin/transform"
)

func TestCurrentHostPackageRejectsInvalidScopeBeforeHostCallback(t *testing.T) {
	const application core.CanonicalID = "sales.app"
	const environment core.CanonicalID = "production.env"
	for _, tc := range []struct {
		name        string
		application core.CanonicalID
		environment core.CanonicalID
	}{
		{name: "empty application", application: "", environment: environment},
		{name: "empty environment", application: application, environment: ""},
		{name: "application has surrounding space", application: " sales.app ", environment: environment},
		{name: "environment has surrounding space", application: application, environment: "production.env "},
		{name: "uppercase application", application: "Sales.app", environment: environment},
		{name: "uppercase environment", application: application, environment: "Production.env"},
		{name: "invalid separator", application: "sales-app", environment: environment},
		{name: "empty path segment", application: application, environment: "production..env"},
		{name: "newline in environment", application: application, environment: "production.env\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			called := false
			source := func(context.Context, core.CanonicalID, core.CanonicalID) (CurrentPackageBuildBinding, error) {
				called = true
				return CurrentPackageBuildBinding{}, nil
			}
			_, err := PrepareCurrentHostPackageInputs(context.Background(), tc.application, tc.environment, source, transform.Source{}, plugin.BuildPlan{}, nil)
			if !errors.Is(err, ErrCurrentPackageSelectionInvalid) {
				t.Fatalf("noncanonical Host scope was admitted: %v", err)
			}
			if called {
				t.Fatal("Host callback was invoked with noncanonical scope")
			}
		})
	}
}

func TestCurrentHostPackageValidCanonicalScopeReachesHost(t *testing.T) {
	application := core.MustCanonicalID("sales.app")
	environment := core.MustCanonicalID("production.env")
	called := 0
	current := func(_ context.Context, gotApplication, gotEnvironment core.CanonicalID) (CurrentPackageBuildBinding, error) {
		called++
		if gotApplication != application || gotEnvironment != environment {
			t.Fatalf("Host received modified scope: %q %q", gotApplication, gotEnvironment)
		}
		return CurrentPackageBuildBinding{
			Application: application,
			Environment: environment,
			Selection:   BuildInputSelection{}, // No implicit v2 selection.
		}, nil
	}
	_, err := PrepareCurrentHostPackageInputs(context.Background(), application, environment, current, transform.Source{}, plugin.BuildPlan{}, nil)
	if err == nil || err.Error() != `unsupported package input profile ""` {
		t.Fatalf("a missing Host profile was silently accepted: %v", err)
	}
	if called != 1 {
		t.Fatalf("Host callback called %d times, want one", called)
	}
}

func TestCurrentHostPackageCanceledRequestNeverInvokesHost(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	called := false
	current := func(context.Context, core.CanonicalID, core.CanonicalID) (CurrentPackageBuildBinding, error) {
		called = true
		return CurrentPackageBuildBinding{}, nil
	}
	_, err := PrepareCurrentHostPackageInputs(ctx, core.MustCanonicalID("sales.app"), core.MustCanonicalID("production.env"), current, transform.Source{}, plugin.BuildPlan{}, nil)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled Host request was not propagated: %v", err)
	}
	if called {
		t.Fatal("Host callback executed after context cancellation")
	}
}
