package validation

import (
	"testing"

	"tei/internal/ir"
	"tei/internal/semantic"
)

func TestCapabilityDeclaredEffectUsesCanonicalData(t *testing.T) {
	base := ir.World{Semantics: []semantic.Namespace{{
		ID: "intake",
		Data: []semantic.Data{{ID: "intake.request"}},
		Capabilities: []semantic.Capability{{ID: "intake.request.create"}},
	}}}
	if ds := validateCapabilityDataEffects(base); len(ds) != 0 {
		t.Fatalf("legacy Capability may have no effect: %+v", ds)
	}
	capability := &base.Semantics[0].Capabilities[0]
	capability.Effect = &semantic.DataEffect{Kind: semantic.DataEffectCreate, Data: "intake.request"}
	if ds := validateCapabilityDataEffects(base); len(ds) != 0 {
		t.Fatalf("declared create effect was rejected: %+v", ds)
	}
	capability.Effect.Kind = semantic.DataEffectUpdate
	if ds := validateCapabilityDataEffects(base); len(ds) != 0 {
		t.Fatalf("declared update effect was rejected: %+v", ds)
	}
}

func TestCapabilityDataEffectFailsClosedOnUnknownDataAndKind(t *testing.T) {
	w := ir.World{Semantics: []semantic.Namespace{{
		ID: "intake",
		Data: []semantic.Data{{ID: "intake.request"}},
		Capabilities: []semantic.Capability{{
			ID: "intake.request.save",
			Effect: &semantic.DataEffect{Kind: semantic.DataEffectKind("upsert"), Data: "intake.missing"},
		}},
	}}}
	ds := validateCapabilityDataEffects(w)
	want := map[string]bool{
		"INVALID_SEMANTIC_CAPABILITY_DATA_EFFECT": false,
		"UNKNOWN_SEMANTIC_CAPABILITY_EFFECT_DATA": false,
	}
	for _, d := range ds {
		if _, ok := want[d.Code]; ok {
			want[d.Code] = true
		}
		if d.Subject == nil || d.Subject.ID != "intake.request.save" {
			t.Fatalf("effect validation lost Source subject: %+v", d)
		}
	}
	for code, seen := range want {
		if !seen {
			t.Fatalf("missing %s diagnostic: %+v", code, ds)
		}
	}
}
