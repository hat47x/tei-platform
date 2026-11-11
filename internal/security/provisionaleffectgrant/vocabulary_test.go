package provisionaleffectgrant

import "testing"

// MarkProvisional is the provenance value an effect grant carries when it was
// granted provisionally. A reader must be able to tell it apart from a real
// grant, so the literal is asserted here (#4032).
func TestProvisionalMarkIsCovered(t *testing.T) {
	if MarkProvisional != "provisional-grant" {
		t.Fatalf("MarkProvisional=%q", MarkProvisional)
	}
}
