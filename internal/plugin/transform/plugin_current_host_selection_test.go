package transform_test

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/plugin"
	"tei/internal/plugin/inputprofile"
	"tei/internal/plugin/transform"
)

func TestCurrentHostPackageSelectionBindsScopedV2Inputs(t *testing.T) {
	def, plan, source, selection, raw := admissionFixture(t)
	app, env := core.CanonicalID("order_entry"), core.CanonicalID("development")
	reads := 0
	current := func(_ context.Context, application, environment core.CanonicalID) (inputprofile.CurrentPackageBuildBinding, error) {
		reads++
		if application != app || environment != env {
			t.Fatalf("Host selection used different scope: %q/%q", application, environment)
		}
		return inputprofile.CurrentPackageBuildBinding{
			Application: app, Environment: env, Selection: selection,
		}, nil
	}
	type spoofedScope struct{}
	ctx := context.WithValue(context.Background(), spoofedScope{}, "other_application/production")
	prepared, err := inputprofile.PrepareCurrentHostPackageInputs(ctx, app, env, current, source, plan, raw)
	if err != nil || reads != 1 {
		t.Fatalf("Host-selected v2 admission: %v reads=%d", err, reads)
	}
	if prepared.Evidence.Profile != inputprofile.PluginDefinitionJSONV2 ||
		prepared.Evidence.SourceFingerprint != selection.LockedSourceFingerprint ||
		prepared.Evidence.BuildPlanFingerprint != selection.LockedBuildPlanFingerprint {
		t.Fatalf("Host-owned selection was not preserved: %+v", prepared.Evidence)
	}
	actual, err := transform.Evaluate(source, prepared.Inputs)
	if err != nil {
		t.Fatal(err)
	}
	want, err := plugin.MaterializePluginPackage(def, plan)
	if err != nil || !bytes.Equal(want, actual.Output.Bytes) {
		t.Fatalf("Host-scoped v2 inputs did not produce canonical SB-1 bytes: %v", err)
	}
	if _, err := inputprofile.ObservePackageOutput(prepared.Evidence, actual); err != nil {
		t.Fatalf("Host-selected inputs did not retain output evidence: %v", err)
	}
	// This fixture is not a production Host or an execution authorization.
}

func TestCurrentHostPackageSelectionAlwaysReadsCurrentAndFailsClosed(t *testing.T) {
	_, plan, source, selection, raw := admissionFixture(t)
	app, env := core.CanonicalID("order_entry"), core.CanonicalID("development")
	reads := 0
	currentSelection := selection
	currentError := error(nil)
	current := func(_ context.Context, application, environment core.CanonicalID) (inputprofile.CurrentPackageBuildBinding, error) {
		reads++
		if currentError != nil {
			return inputprofile.CurrentPackageBuildBinding{}, currentError
		}
		return inputprofile.CurrentPackageBuildBinding{
			Application: app, Environment: env, Selection: currentSelection,
		}, nil
	}
	prepare := func() error {
		_, err := inputprofile.PrepareCurrentHostPackageInputs(context.Background(), app, env, current, source, plan, raw)
		return err
	}
	if err := prepare(); err != nil {
		t.Fatal(err)
	}
	currentError = errors.New("revoked; private Host authorization details")
	if err := prepare(); !errors.Is(err, inputprofile.ErrCurrentPackageSelectionUnavailable) ||
		strings.Contains(err.Error(), "private Host") {
		t.Fatalf("revoked Host state was ignored or exposed: %v", err)
	}
	currentError = nil
	currentSelection.Profile = ""
	if err := prepare(); err == nil || !strings.Contains(err.Error(), "unsupported package input profile") {
		t.Fatalf("missing current opt-in became an implicit v2 selection: %v", err)
	}
	currentSelection = selection
	currentSelection.LockedBuildPlanFingerprint = "forged-request-plan"
	if err := prepare(); err == nil || !strings.Contains(err.Error(), "Host-approved BuildPlan fingerprint") {
		t.Fatalf("untrusted Plan was admitted: %v", err)
	}
	currentSelection = selection
	currentSelection.LockedSourceFingerprint = "forged-request-source"
	if err := prepare(); err == nil || !strings.Contains(err.Error(), "exact locked SB-1 Source fingerprint") {
		t.Fatalf("untrusted Source was admitted: %v", err)
	}
	currentSelection = selection
	if err := prepare(); err != nil || reads != 6 {
		t.Fatalf("a current authorized selection could not recover: %v reads=%d", err, reads)
	}
}

func TestCurrentHostPackageSelectionRejectsWrongScopeAndCancellation(t *testing.T) {
	_, plan, source, selection, raw := admissionFixture(t)
	app, env := core.CanonicalID("order_entry"), core.CanonicalID("development")
	call := func(ctx context.Context, application, environment core.CanonicalID, current inputprofile.CurrentPackageBuildSource) error {
		_, err := inputprofile.PrepareCurrentHostPackageInputs(ctx, application, environment, current, source, plan, raw)
		return err
	}
	valid := func(_ context.Context, _, _ core.CanonicalID) (inputprofile.CurrentPackageBuildBinding, error) {
		return inputprofile.CurrentPackageBuildBinding{Application: app, Environment: env, Selection: selection}, nil
	}
	for _, tc := range []struct {
		name    string
		ctx     context.Context
		app     core.CanonicalID
		env     core.CanonicalID
		current inputprofile.CurrentPackageBuildSource
	}{
		{name: "nil context", app: app, env: env, current: valid},
		{name: "nil Host source", ctx: context.Background(), app: app, env: env},
		{name: "missing Application", ctx: context.Background(), env: env, current: valid},
		{name: "missing Environment", ctx: context.Background(), app: app, current: valid},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := call(tc.ctx, tc.app, tc.env, tc.current); !errors.Is(err, inputprofile.ErrCurrentPackageSelectionInvalid) {
				t.Fatalf("invalid Host source was admitted: %v", err)
			}
		})
	}
	wrongApp := func(_ context.Context, _, _ core.CanonicalID) (inputprofile.CurrentPackageBuildBinding, error) {
		return inputprofile.CurrentPackageBuildBinding{Application: "other_application", Environment: env, Selection: selection}, nil
	}
	if err := call(context.Background(), app, env, wrongApp); !errors.Is(err, inputprofile.ErrCurrentPackageSelectionScope) {
		t.Fatalf("cross-Application selection was admitted: %v", err)
	}
	wrongEnv := func(_ context.Context, _, _ core.CanonicalID) (inputprofile.CurrentPackageBuildBinding, error) {
		return inputprofile.CurrentPackageBuildBinding{Application: app, Environment: "production", Selection: selection}, nil
	}
	if err := call(context.Background(), app, env, wrongEnv); !errors.Is(err, inputprofile.ErrCurrentPackageSelectionScope) {
		t.Fatalf("cross-Environment selection was admitted: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	reads := 0
	neverRead := func(_ context.Context, _, _ core.CanonicalID) (inputprofile.CurrentPackageBuildBinding, error) {
		reads++
		return inputprofile.CurrentPackageBuildBinding{}, nil
	}
	if err := call(ctx, app, env, neverRead); !errors.Is(err, context.Canceled) || reads != 0 {
		t.Fatalf("cancelled Host request read selection: %v reads=%d", err, reads)
	}
	ctx, cancel = context.WithCancel(context.Background())
	lateCancel := func(_ context.Context, _, _ core.CanonicalID) (inputprofile.CurrentPackageBuildBinding, error) {
		reads++
		cancel()
		return inputprofile.CurrentPackageBuildBinding{Application: app, Environment: env, Selection: selection}, nil
	}
	if err := call(ctx, app, env, lateCancel); !errors.Is(err, context.Canceled) || reads != 1 {
		t.Fatalf("Host accepted selection after cancellation: %v reads=%d", err, reads)
	}
}
