package beta

import "testing"

func TestOwnIdentityAndAcknowledgment(t *testing.T) {
	if PluginID != "tei.reference.bundle.beta" || CapabilityID != "process.bundle.beta" || Version == "" {
		t.Fatal("unstable reference plugin identity")
	}
	message, ref := Accept("intent")
	if message != "beta accepted intent" || ref != "reference-beta://intent" {
		t.Fatalf("unexpected beta receipt: %s %s", message, ref)
	}
}
