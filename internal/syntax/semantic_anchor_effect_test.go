package syntax

import "testing"

func TestParseSemanticFileAnchoredAcceptsCapabilityDataEffect(t *testing.T) {
	src := `namespace sales

capability sales.order.submit {
    input order sales.order
    output sales.order
    effect create sales.order
}
`

	parsed, err := ParseSemanticFileAnchored(src)
	if err != nil {
		t.Fatalf("DataEffectを持つCapabilityのSource Anchorを構築できません: %v", err)
	}
	if len(parsed.Value.Capabilities) != 1 {
		t.Fatalf("capabilities=%d want=1", len(parsed.Value.Capabilities))
	}
	effect := parsed.Value.Capabilities[0].Effect
	if effect == nil || effect.Kind != "create" || effect.Data != "sales.order" {
		t.Fatalf("effect=%+v", effect)
	}
}
