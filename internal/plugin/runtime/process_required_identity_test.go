package runtime_test

import (
	"context"
	"strings"
	"testing"

	pruntime "tei/internal/plugin/runtime"
)

// These replies come from a real child process, not a mock ProcessLauncher.
// Registry contains historical response defaulting for in-process/mocked
// callers, but an external process must never use that defaulting to attest
// its own identity and descriptor fingerprint.
func TestProcessResponseRequiresCompleteCarrierIdentity(t *testing.T) {
	for _, field := range []string{
		"protocol", "request", "plugin", "plugin-id", "plugin-version",
		"capability", "fingerprint", "status",
	} {
		for _, variant := range []string{"missing", "empty"} {
			t.Run(variant+"-"+field, func(t *testing.T) {
				launcher, command := helperLauncher(t, "wire-"+variant+"-"+field)
				resp, err := launcher.Invoke(context.Background(), command, processRequest())
				if err == nil || !strings.Contains(err.Error(), "missing required carrier identity or status") {
					t.Fatalf("unbound plugin response accepted: %v", err)
				}
				if strings.Contains(err.Error(), processResponseSecretCanary) {
					t.Fatal("untrusted response bytes leaked into error")
				}
				if resp.Status == pruntime.ResponseOK || len(resp.Payload) != 0 {
					t.Fatal("rejected process carrier exposed a success payload")
				}
			})
		}
	}
}

func TestProcessResponseCompleteCarrierRemainsAccepted(t *testing.T) {
	launcher, command := helperLauncher(t, "echo")
	resp, err := launcher.Invoke(context.Background(), command, processRequest())
	if err != nil || resp.Status != pruntime.ResponseOK {
		t.Fatalf("valid plugin response was rejected: %v", err)
	}
}
