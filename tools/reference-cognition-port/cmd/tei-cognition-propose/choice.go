package main

import (
	"context"
	"crypto/sha256"
	"fmt"
	"strings"

	"tei/internal/composition"
	"tei/plugins/local-choice"
	cognitionport "tei/tools/reference-cognition-port"
)

// The CLI is a static, read-only consumer. It selects an exact Contract and
// Provider through the existing Composition registry. This is not the
// application-runtime carrier, an authority issuer or a new runtime Host.
func selectedChoice(ctx context.Context, request cognitionport.Request, provider cognitionport.Provider) (*cognitionport.Proposal, error) {
	if provider.ID != localchoice.ID || provider.Execution != cognitionport.ExecutionExternal || provider.Network != cognitionport.NetworkRequired {
		return nil, fmt.Errorf("descriptor does not match the selected loopback provider")
	}
	if request.Kind != "choice" {
		return &cognitionport.Proposal{Schema: cognitionport.ProposalSchema, State: cognitionport.StateUnsupported}, nil
	}
	contract := composition.Contract{ID: localchoice.ContractID, Version: localchoice.Version, Shape: fmt.Sprintf("%x", sha256.Sum256([]byte(localchoice.Shape)))}
	const scope = "reference-cli"
	registry := composition.NewRegistry()
	if _, err := registry.Apply(0, []composition.Provider{{ID: localchoice.ID, Scope: scope, Offers: contract}}, nil); err != nil {
		return nil, err
	}
	selection, err := registry.Select(provider.ID, scope)
	if err != nil || selection.Contract != contract || selection.ProviderID != localchoice.ID {
		return nil, fmt.Errorf("choice contract selection failed")
	}
	endpoint, ok := provider.Settings["endpoint"].(string)
	if !ok || len(provider.Settings) != 1 {
		return nil, fmt.Errorf("choice provider requires only the endpoint setting")
	}
	if len(request.Context.AcceptedReferences) != 0 || len(request.Context.Constraints) != 0 {
		return nil, fmt.Errorf("choice contract accepts only input and option observations")
	}
	var input cognitionport.Observation
	var options []localchoice.Option
	for _, o := range request.Context.Observations {
		switch {
		case o.ID == "input":
			input = o
		case strings.HasPrefix(o.ID, "option.") && o.Reference == "":
			options = append(options, localchoice.Option{ID: strings.TrimPrefix(o.ID, "option."), Criterion: o.Statement})
		default:
			return nil, fmt.Errorf("unknown choice observation")
		}
	}
	if strings.TrimSpace(input.Reference) == "" {
		return nil, fmt.Errorf("choice input requires a subject reference")
	}
	answer, err := localchoice.Choose(ctx, endpoint, provider.Model, input.Statement, options)
	if err != nil {
		return nil, err
	}
	if answer == localchoice.Unknown {
		return &cognitionport.Proposal{Schema: cognitionport.ProposalSchema, State: cognitionport.StateUnknown, Diagnostics: []string{"provider abstained"}}, nil
	}
	return &cognitionport.Proposal{
		Schema: cognitionport.ProposalSchema, State: cognitionport.StateProposed,
		Candidates: []cognitionport.Candidate{{
			ID: "choice-" + answer, Kind: request.Kind, Subject: input.Reference,
			Payload: map[string]any{"choice": answer},
			Basis:   []string{"observation:input", "observation:option." + answer},
		}},
	}, nil
}
