package main

import (
	"path/filepath"
	"testing"

	"tei/internal/application"
	"tei/internal/ir"
	"tei/internal/semantic"
)

func writeIntentWorld() ir.World {
	return ir.World{
		Applications: []application.Application{{
			ID: "order_entry",
			Patterns: []application.Pattern{{
				ID: "order_entry.submit_order", Kind: application.PatternCommand, Capability: "sales.order.submit",
			}},
		}},
		Semantics: []semantic.Namespace{{
			ID: "sales",
			Capabilities: []semantic.Capability{{
				ID: "sales.order.submit",
				Effect: &semantic.DataEffect{Kind: semantic.DataEffectCreate, Data: "sales.order"},
			}},
		}},
	}
}

func TestDeclaredWriteIntentSelectsEffectFenceOnlyAsUnverifiedObligation(t *testing.T) {
	world := writeIntentWorld()
	intent := ResolveOperationWriteIntent(world, "order_entry", "order_entry.submit_order")
	if intent.Status != "declared-write" || intent.EffectKind != "create" || intent.EffectData != "sales.order" {
		t.Fatalf("expected explicit write intent: %+v", intent)
	}
	report := WorkspaceAssessment{
		SubjectKind: "application-pattern",
		SubjectID: "order_entry.submit_order",
		OperationWriteIntent: &intent,
	}
	p, err := ProjectForPurpose(report, "execute-preflight")
	if err != nil {
		t.Fatal(err)
	}
	ob, ok := purposeRequirementByName(p, "effect-fence")
	if !ok || ob.Status != "not-evaluated" || p.ExecutionAllowed {
		t.Fatalf("declared effect did not retain a runtime-only obligation: %+v", p)
	}
}

func TestMissingOrAmbiguousCapabilityDoesNotProveReadOnly(t *testing.T) {
	world := writeIntentWorld()
	world.Semantics[0].Capabilities[0].Effect = nil
	intent := ResolveOperationWriteIntent(world, "order_entry", "order_entry.submit_order")
	if intent.Status != "effect-not-declared" {
		t.Fatalf("nil DataEffect was treated as a read-only proof: %+v", intent)
	}
	p, err := ProjectForPurpose(WorkspaceAssessment{
		SubjectKind: "application-pattern", SubjectID: "order_entry.submit_order", OperationWriteIntent: &intent,
	}, "execute-preflight")
	if err != nil {
		t.Fatal(err)
	}
	if ob, ok := purposeRequirementByName(p, "effect-fence"); !ok || ob.Status != "conditional" {
		t.Fatalf("unclassified effect was treated as proven absent: %+v", p)
	}

	world = writeIntentWorld()
	world.Semantics[0].Capabilities = append(world.Semantics[0].Capabilities, world.Semantics[0].Capabilities[0])
	if got := ResolveOperationWriteIntent(world, "order_entry", "order_entry.submit_order"); got.Status != "unresolved" {
		t.Fatalf("duplicate capability silently picked: %+v", got)
	}

	world = writeIntentWorld()
	world.Applications[0].Patterns[0].Capability = "sales.missing"
	if got := ResolveOperationWriteIntent(world, "order_entry", "order_entry.submit_order"); got.Status != "unresolved" {
		t.Fatalf("missing capability silently accepted: %+v", got)
	}

	world = writeIntentWorld()
	world.Applications[0].Patterns[0].Kind = application.PatternReference
	if got := ResolveOperationWriteIntent(world, "order_entry", "order_entry.submit_order"); got.Status != "not-command" {
		t.Fatalf("reference pattern mistaken for write declaration: %+v", got)
	}
}

func TestRealSubmitOrderHasNoDeclaredWriteEffect(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "workspace")
	got, err := InspectWorkspace(root, "order_entry", "application-pattern", "order_entry.submit_order")
	if err != nil {
		t.Fatal(err)
	}
	if got.SourceStatus != "traceable-source-declaration" || got.OperationWriteIntent == nil {
		t.Fatalf("actual command was not resolved: %+v", got)
	}
	if got.OperationWriteIntent.Capability != "sales.order.submit" ||
		got.OperationWriteIntent.Status != "effect-not-declared" ||
		got.ExecutionAllowed {
		t.Fatalf("unexpected write/read-only assertion from current canonical: %+v", got)
	}
}
