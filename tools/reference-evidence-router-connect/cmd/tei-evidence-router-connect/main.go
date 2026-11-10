// tei-evidence-router-connect resolves the Environment's evidence providers and
// enters them through the existing evidence.Router.
//
// The provider supplied here is reference-only: it reads nothing external and
// answers with a fixed pass. This shows the connect point runs, not that the
// evidence is true.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"tei/internal/core"
	"tei/internal/deployment"
	"tei/internal/deployment/durable"
	"tei/internal/deployment/evidence"
	"tei/internal/environment"
	"tei/internal/workspace"
	evidencerouterconnect "tei/tools/reference-evidence-router-connect"
)

// referenceCollector answers every request with a fixed pass outcome, so a
// reader can see entry happened without mistaking it for real evidence.
type referenceCollector struct {
	id   core.CanonicalID
	kind deployment.EvidenceKind
}

func (c referenceCollector) ID() string { return string(c.id) }

func (c referenceCollector) Supports(k deployment.EvidenceKind) bool { return k == c.kind }

func (c referenceCollector) Collect(_ context.Context, req evidence.Request) (durable.Evidence, error) {
	return durable.Evidence{
		ID:        "reference-evidence",
		ActionID:  req.ActionID,
		Kind:      req.Kind,
		Outcome:   durable.EvidencePass,
		SourceRef: "reference-only",
	}, nil
}

func main() {
	root := flag.String("workspace", "", "Source Workspace root to read")
	environmentID := flag.String("environment", "", "Environment whose declarations are used")
	actionID := flag.String("action-id", "reference-gate", "durable action record id")
	register := flag.Bool("register", true, "register a reference-only provider for the declared kind")
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
	if len(env.EvidenceProviders) == 0 {
		fail(fmt.Errorf("environment %s declares no evidence provider", *environmentID))
	}

	declared := env.EvidenceProviders[0]
	providers := map[core.CanonicalID]evidence.Provider{}
	if *register && !*registerNone {
		providers[declared.Provider] = referenceCollector{id: declared.Provider, kind: declared.Kind}
	}

	records := []durable.ActionRecord{{
		Action: durable.Action{
			ID: *actionID, Kind: durable.ActionRolloutHealthGate, Mode: durable.ModeEvidence,
			EvidenceKind: declared.Kind,
		},
	}}

	report := evidencerouterconnect.Run(context.Background(), evidencerouterconnect.Input{
		Environment: env,
		Providers:   providers,
		Records:     records,
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
