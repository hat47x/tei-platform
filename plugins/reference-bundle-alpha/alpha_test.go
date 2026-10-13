package alpha

import "testing"

func TestOwnIdentityAndAcknowledgment(t *testing.T) {
	if PluginID != "tei.reference.bundle.alpha" || CapabilityID != "process.bundle.alpha" || Version == "" {
		t.Fatal("unstable reference plugin identity")
	}
	message, ref := Accept("intent")
	if message != "alpha accepted intent" || ref != "reference-alpha://intent" {
		t.Fatalf("unexpected alpha receipt: %s %s", message, ref)
	}
}
