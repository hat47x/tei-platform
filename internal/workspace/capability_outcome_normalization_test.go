package workspace_test

import (
	"testing"

	"tei/internal/ir"
	"tei/internal/semantic"
	"tei/internal/workspace"
)

func TestNormalizeSortsCapabilityOutcomesByName(t *testing.T) {
	w := ir.World{Semantics: []semantic.Namespace{{
		ID: "security",
		Capabilities: []semantic.Capability{{
			ID: "security.authenticate",
			Outcomes: []semantic.Outcome{
				{Name: "denied", Payload: "security.authentication_denial"},
				{Name: "authenticated", Payload: "security.authenticated_subject"},
			},
		}},
	}}}

	workspace.Normalize(&w)

	outcomes := w.Semantics[0].Capabilities[0].Outcomes
	if len(outcomes) != 2 {
		t.Fatalf("outcomes=%#v", outcomes)
	}
	if outcomes[0].Name != "authenticated" || outcomes[1].Name != "denied" {
		t.Fatalf("Outcomeがlocal name順に正規化されていません: %#v", outcomes)
	}
}
