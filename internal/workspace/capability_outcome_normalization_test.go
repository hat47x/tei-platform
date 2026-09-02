package workspace_test

import (
	"testing"

	"tei/internal/ir"
	"tei/internal/semantic"
	"tei/internal/workspace"
)

func TestCapabilityOutcomesNormalizeByLocalName(t *testing.T) {
	world := ir.World{Semantics: []semantic.Namespace{
		{
			ID: "security",
			Capabilities: []semantic.Capability{
				{
					ID:     "security.authenticate",
					Symbol: "authenticate",
					Outcomes: []semantic.Outcome{
						{Name: "denied", Payload: "security.authentication_denial"},
						{Name: "authenticated", Payload: "security.authenticated_subject"},
					},
				},
			},
		},
	}}

	workspace.Normalize(&world)

	got := world.Semantics[0].Capabilities[0].Outcomes
	if len(got) != 2 {
		t.Fatalf("outcomes=%#v", got)
	}
	if got[0].Name != "authenticated" || got[1].Name != "denied" {
		t.Fatalf("Outcomeの宣言順がCanonicalな意味へ残っています: %#v", got)
	}
}
