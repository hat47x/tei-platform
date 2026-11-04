package command

import (
	"context"
	"testing"

	"tei/internal/core"
	"tei/internal/ir"
	"tei/internal/semantic"
)

func candidateKeyWorld() ir.World {
	world := valueDomainWorld()
	for i := range world.Semantics[0].Data {
		if world.Semantics[0].Data[i].ID == "sales.order" {
			world.Semantics[0].Data[i].CandidateKeys = []semantic.CandidateKey{{
				ID:         "sales.order.status",
				Symbol:     "status",
				Attributes: []core.CanonicalID{"sales.order.status"},
			}}
		}
	}
	return world
}

func TestReferenceDataExecutorEnforcesCandidateKey(t *testing.T) {
	executor := ReferenceDataExecutor{World: candidateKeyWorld(), Store: NewMemoryStore()}

	first := request()
	first.Inputs = map[string]string{"order": "ORD-1", "status": "open"}
	if _, err := executor.Execute(context.Background(), first); err != nil {
		t.Fatal(err)
	}

	duplicate := request()
	duplicate.Inputs = map[string]string{"order": "ORD-2", "status": "open"}
	if _, err := executor.Execute(context.Background(), duplicate); err == nil {
		t.Fatal("duplicate candidate key was accepted")
	}

	distinct := request()
	distinct.Inputs = map[string]string{"order": "ORD-2", "status": "closed"}
	if _, err := executor.Execute(context.Background(), distinct); err != nil {
		t.Fatalf("distinct candidate key was rejected: %v", err)
	}
}
