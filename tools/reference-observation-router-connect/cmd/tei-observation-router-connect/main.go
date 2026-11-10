// tei-observation-router-connect resolves the Environment's observation
// providers and enters them through the existing observation.Router.
//
// The providers supplied here are reference-only: they read nothing external and
// answer with a fixed fragment. This shows the connect point runs, not that an
// observation reflects the world.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"tei/internal/core"
	"tei/internal/deployment"
	"tei/internal/deployment/observation"
	"tei/internal/environment"
	"tei/internal/workspace"
	observationrouterconnect "tei/tools/reference-observation-router-connect"
)

// referenceObserver answers every request with a fragment carrying only the
// identity fields, so a reader can see entry happened without mistaking it for
// real state.
type referenceObserver struct {
	id   core.CanonicalID
	kind deployment.ObservationKind
}

func (o referenceObserver) ID() string { return string(o.id) }

func (o referenceObserver) Supports(k deployment.ObservationKind) bool { return k == o.kind }

func (o referenceObserver) Observe(_ context.Context, req observation.Request) (observation.Fragment, error) {
	return observation.Fragment{
		ProviderID:  o.id,
		Environment: req.Environment,
		Kind:        req.Kind,
		ObservedAt:  "reference-only",
	}, nil
}

func main() {
	root := flag.String("workspace", "", "Source Workspace root to read")
	environmentID := flag.String("environment", "", "Environment whose declarations are used")
	registerAll := flag.Bool("register", true, "register a reference-only provider for every declared kind")
	registerNone := flag.Bool("register-none", false, "register no provider, to show the refusal")
	flag.Parse()
	if *root == "" || *environmentID == "" {
		fail(fmt.Errorf("-workspace and -environment are required"))
	}

	loaded, err := workspace.Load(*root)
	if err != nil {
		fail(fmt.Errorf("load Workspace: %w", err))
	}
	env, ok := findEnvironment(loaded.World.Environments, *environmentID)
	if !ok {
		fail(fmt.Errorf("environment %s not found", *environmentID))
	}

	providers := map[core.CanonicalID]observation.Provider{}
	if *registerAll && !*registerNone {
		for _, b := range env.ObservationProviders {
			providers[b.Provider] = referenceObserver{id: b.Provider, kind: b.Kind}
		}
	}

	report := observationrouterconnect.Run(context.Background(), observationrouterconnect.Input{
		Environment: env,
		Providers:   providers,
	})
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(report); err != nil {
		fail(err)
	}
}

func findEnvironment(envs []environment.Environment, id string) (environment.Environment, bool) {
	for _, env := range envs {
		if string(env.ID) == id {
			return env, true
		}
	}
	return environment.Environment{}, false
}

func fail(err error) { fmt.Fprintln(os.Stderr, err); os.Exit(2) }
