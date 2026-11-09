package validation

import (
	"testing"
)

func TestValidateRelationOnDelete(t *testing.T) {
	w := applicationPatternInteractionWorld()
	if len(w.Semantics[0].Data[0].Relations) == 0 {
		t.Skip("world has no relation")
	}
	w.Semantics[0].Data[0].Relations[0].OnDelete = "restrict"
	if diagnosticByCode(Validate(w), "SEMANTIC_RELATION_ON_DELETE_UNSUPPORTED") != nil {
		t.Fatal("supported on-delete was rejected")
	}
}

func TestValidateRelationOnDeleteRejectsUnknown(t *testing.T) {
	w := applicationPatternInteractionWorld()
	if len(w.Semantics[0].Data[0].Relations) == 0 {
		t.Skip("world has no relation")
	}
	w.Semantics[0].Data[0].Relations[0].OnDelete = "block"
	if diagnosticByCode(Validate(w), "SEMANTIC_RELATION_ON_DELETE_UNSUPPORTED") == nil {
		t.Fatal("unknown on-delete was accepted")
	}
}
