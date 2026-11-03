package validation

import (
	"testing"

	"tei/internal/ir"
	"tei/internal/semantic"
)

func deletePolicyWorld(policy string) ir.World {
	w := testCapabilityEffectWorld()
	w.Semantics[0].Data = append(w.Semantics[0].Data, semantic.Data{
		ID: "intake.line",
		Relations: []semantic.Relation{{
			ID:       "intake.line.request",
			Target:   "intake.request",
			OnDelete: policy,
		}},
	})
	w.Semantics[0].Capabilities[0].Effect = &semantic.DataEffect{
		Kind: semantic.DataEffectDelete,
		Data: "intake.request",
	}
	return w
}

func TestDeleteEffectRequiresInboundPolicy(t *testing.T) {
	if diagnosticByCode(validateCapabilityDataEffects(deletePolicyWorld("")), "DELETE_EFFECT_RELATION_POLICY_MISSING") == nil {
		t.Fatal("delete effect without inbound policy was accepted")
	}
	if diagnosticByCode(validateCapabilityDataEffects(deletePolicyWorld("restrict")), "DELETE_EFFECT_RELATION_POLICY_MISSING") != nil {
		t.Fatal("delete effect with inbound policy was reported")
	}
}
