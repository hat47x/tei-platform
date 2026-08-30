package observation_test

import (
	"context"
	"testing"

	"tei/internal/core"
	dbrelease "tei/internal/database/release"
	"tei/internal/deployment"
	"tei/internal/deployment/observation"
	"tei/internal/environment"
)

type fakeObserver struct {
	id       string
	kind     deployment.ObservationKind
	fragment observation.Fragment
}

func (f fakeObserver) ID() string                                 { return f.id }
func (f fakeObserver) Supports(k deployment.ObservationKind) bool { return k == f.kind }
func (f fakeObserver) Observe(context.Context, observation.Request) (observation.Fragment, error) {
	return f.fragment, nil
}

func TestObservationRoutingAndMerge(t *testing.T) {
	env := environment.Environment{ID: "production", ObservationProviders: []environment.ObservationProviderBinding{{Kind: deployment.ObservationApplicationRuntime, Provider: "runtime.observer"}, {Kind: deployment.ObservationDatabaseState, Provider: "database.observer"}}}
	r := observation.Router{Providers: map[core.CanonicalID]observation.Provider{
		"runtime.observer":  fakeObserver{id: "runtime.observer", kind: deployment.ObservationApplicationRuntime, fragment: observation.Fragment{ApplicationRuntime: []deployment.ApplicationRuntimeState{{Application: "order_entry", Version: "3.8.0", Lifecycle: deployment.LifecycleServing, TrafficBasisPoints: 10000, AcceptingWork: true}}}},
		"database.observer": fakeObserver{id: "database.observer", kind: deployment.ObservationDatabaseState, fragment: observation.Fragment{Databases: []dbrelease.StateSnapshot{{Database: "sales_core", Source: dbrelease.StateFromObserved, DataReadiness: dbrelease.DataReady}}}},
	}}
	a, err := r.Observe(context.Background(), env, deployment.ObservationApplicationRuntime)
	if err != nil {
		t.Fatal(err)
	}
	d, err := r.Observe(context.Background(), env, deployment.ObservationDatabaseState)
	if err != nil {
		t.Fatal(err)
	}
	base := deployment.Snapshot{Environment: "production"}
	m, err := observation.Merge(base, a, d)
	if err != nil {
		t.Fatal(err)
	}
	if len(m.ApplicationRuntime) != 1 || len(m.Databases) != 1 {
		t.Fatalf("unexpected merged state: %+v", m)
	}
}

func TestApplicationObservationIsAuthoritativeFacet(t *testing.T) {
	base := deployment.Snapshot{Environment: "production", ApplicationRuntime: []deployment.ApplicationRuntimeState{
		{Application: "order_entry", Version: "3.7.0", Lifecycle: deployment.LifecycleQuiescent, DrainComplete: true},
		{Application: "order_entry", Version: "3.8.0", Lifecycle: deployment.LifecycleServing, TrafficBasisPoints: 10000, AcceptingWork: true},
	}}
	f := observation.Fragment{Environment: "production", Kind: deployment.ObservationApplicationRuntime, ApplicationRuntime: []deployment.ApplicationRuntimeState{{Application: "order_entry", Version: "3.8.0", Lifecycle: deployment.LifecycleServing, TrafficBasisPoints: 10000, AcceptingWork: true}}}
	got, err := observation.Merge(base, f)
	if err != nil {
		t.Fatal(err)
	}
	if got.IsActive("order_entry", "3.7.0") {
		t.Fatalf("retired release must disappear from authoritative runtime facet: %+v", got)
	}
}
