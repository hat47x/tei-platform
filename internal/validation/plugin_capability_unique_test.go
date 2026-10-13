package validation_test

import (
	"testing"

	"tei/internal/plugin"
	"tei/internal/validation"
)

func TestWorldValidationRejectsRepeatedCapabilityIdentity(t *testing.T) {
	w := referenceWorld()
	w.Plugins[0].Provides = []plugin.Capability{
		{Kind: plugin.CapabilityDeploymentExecutor, ID: cid("runtime.shared")},
		{Kind: plugin.CapabilityDeploymentExecutor, ID: cid("runtime.shared")},
	}
	for _, d := range validation.Validate(w) {
		if d.Code == "DUPLICATE_PLUGIN_CAPABILITY" && d.Severity == validation.Error {
			return
		}
	}
	t.Fatal("world validation must reject duplicate capability identity")
}
