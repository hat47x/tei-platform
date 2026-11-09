package validation

import (
	"testing"
)

func TestValidateAttributeRole(t *testing.T) {
	w := applicationPatternInteractionWorld()
	w.Semantics[0].Data[0].Attributes[0].Role = "business-key"
	if diagnosticByCode(Validate(w), "SEMANTIC_ATTRIBUTE_ROLE_UNSUPPORTED") != nil {
		t.Fatal("supported role was rejected")
	}
}

func TestValidateAttributeRoleRejectsUnknown(t *testing.T) {
	w := applicationPatternInteractionWorld()
	w.Semantics[0].Data[0].Attributes[0].Role = "primary"
	if diagnosticByCode(Validate(w), "SEMANTIC_ATTRIBUTE_ROLE_UNSUPPORTED") == nil {
		t.Fatal("unknown role was accepted")
	}
}
