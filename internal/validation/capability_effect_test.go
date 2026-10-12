package validation

import (
	"testing"

	"tei/internal/ir"
	"tei/internal/semantic"
)

func testCapabilityEffectWorld() ir.World {
	namespace := semantic.Namespace{ID: "intake"}
	namespace.Data = []semantic.Data{{ID: "intake.request"}}
	namespace.Capabilities = []semantic.Capability{{ID: "intake.request.save"}}
	return ir.World{Semantics: []semantic.Namespace{namespace}}
}

func TestCapabilityDeclaredEffectUsesCanonicalData(t *testing.T) {
	w := testCapabilityEffectWorld()
	if ds := validateCapabilityDataEffects(w); len(ds) != 0 {
		t.Fatalf("legacy Capability may have no effect: %+v", ds)
	}
	capability := &w.Semantics[0].Capabilities[0]
	capability.Effect = &semantic.DataEffect{Kind: semantic.DataEffectCreate, Data: "intake.request"}
	if ds := validateCapabilityDataEffects(w); len(ds) != 0 {
		t.Fatalf("declared create effect was rejected: %+v", ds)
	}
	capability.Effect.Kind = semantic.DataEffectUpdate
	if ds := validateCapabilityDataEffects(w); len(ds) != 0 {
		t.Fatalf("declared update effect was rejected: %+v", ds)
	}
}

func TestCapabilityDataEffectFailsClosedOnUnknownDataAndKind(t *testing.T) {
	w := testCapabilityEffectWorld()
	w.Semantics[0].Capabilities[0].Effect = &semantic.DataEffect{
		Kind: semantic.DataEffectKind("upsert"),
		Data: "intake.missing",
	}
	diagnostics := validateCapabilityDataEffects(w)
	invalidKind := false
	unknownData := false
	for _, d := range diagnostics {
		if d.Code == "INVALID_SEMANTIC_CAPABILITY_DATA_EFFECT" {
			invalidKind = true
		}
		if d.Code == "UNKNOWN_SEMANTIC_CAPABILITY_EFFECT_DATA" {
			unknownData = true
		}
		if d.Subject == nil || d.Subject.ID != "intake.request.save" {
			t.Fatalf("effect validation lost Source subject: %+v", d)
		}
	}
	if !invalidKind || !unknownData {
		t.Fatalf("invalid kind or unknown Data did not fail closed: %+v", diagnostics)
	}
}
