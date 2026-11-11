package resource

import "testing"

// These are audit vocabularies written into durable records. A reader of a
// stored record branches on them, so each value has to be shown here rather than
// only declared (#4032).
func TestResourceVocabulariesAreCovered(t *testing.T) {
	t.Run("supervision states", func(t *testing.T) {
		states := []SupervisionState{
			SupervisionActive,
			SupervisionRenewalDue,
			SupervisionRenewed,
			SupervisionRenewalFailed,
			SupervisionRenewalDeadlineMissed,
			SupervisionRefreshDue,
			SupervisionRefreshed,
			SupervisionRefreshFailed,
			SupervisionRevokePending,
			SupervisionRevoked,
			SupervisionRevokeFailed,
		}
		want := []string{
			"active", "renewal-due", "renewed", "renewal-failed", "renewal-deadline-missed",
			"refresh-due", "refreshed", "refresh-failed", "revoke-pending", "revoked", "revoke-failed",
		}
		if len(states) != len(want) {
			t.Fatalf("a supervision state was dropped: %d values", len(states))
		}
		seen := map[string]bool{}
		for i, state := range states {
			if string(state) != want[i] {
				t.Fatalf("states[%d]=%q want %q", i, state, want[i])
			}
			if seen[string(state)] {
				t.Fatalf("state %q is declared twice", state)
			}
			seen[string(state)] = true
		}
	})

	t.Run("ownership states", func(t *testing.T) {
		states := []OwnershipState{
			OwnershipClaimed,
			OwnershipHeartbeat,
			OwnershipTakeover,
			OwnershipLost,
			OwnershipReleased,
		}
		want := []string{"ownership-claimed", "ownership-heartbeat", "ownership-takeover", "ownership-lost", "ownership-released"}
		seen := map[string]bool{}
		for i, state := range states {
			if string(state) != want[i] {
				t.Fatalf("states[%d]=%q want %q", i, state, want[i])
			}
			if seen[string(state)] {
				t.Fatalf("state %q is declared twice", state)
			}
			seen[string(state)] = true
		}
	})

	t.Run("secret delivery", func(t *testing.T) {
		deliveries := []SecretDelivery{SecretDeliveryHandle, SecretDeliveryEnvironment, SecretDeliveryFile}
		want := []string{"handle", "environment", "file"}
		for i, delivery := range deliveries {
			if string(delivery) != want[i] {
				t.Fatalf("deliveries[%d]=%q want %q", i, delivery, want[i])
			}
		}
		// A declared value has to be accepted by Valid, and an unknown one not.
		for _, delivery := range deliveries {
			if !delivery.Valid() {
				t.Fatalf("%q is declared but not accepted by Valid", delivery)
			}
		}
		if SecretDelivery("carrier-pigeon").Valid() {
			t.Fatal("an undeclared secret delivery was accepted")
		}
	})
}
