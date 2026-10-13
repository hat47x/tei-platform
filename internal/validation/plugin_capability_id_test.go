package validation_test

import (
	"testing"

	"tei/internal/core"
	"tei/internal/plugin"
	"tei/internal/validation"
)

func TestWorldRejectsMalformedPluginCapabilityIDs(t *testing.T) {
	for _, tc := range []struct {
		name   string
		id     core.CanonicalID
		reject bool
	}{
		{name: "valid", id: core.CanonicalID("runtime.valid")},
		{name: "empty", id: "", reject: true},
		{name: "spaces", id: core.CanonicalID("not a valid id"), reject: true},
		{name: "padded", id: core.CanonicalID(" runtime.valid "), reject: true},
		{name: "uppercase", id: core.CanonicalID("Runtime.valid"), reject: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w := referenceWorld()
			w.Plugins[0].Provides = []plugin.Capability{{
				Kind: plugin.CapabilityDeploymentExecutor,
				ID:   tc.id,
			}}
			found := false
			for _, d := range validation.Validate(w) {
				if d.Code == "INVALID_PLUGIN_CAPABILITY_ID" && d.Severity == validation.Error {
					found = true
				}
			}
			if found != tc.reject {
				t.Fatalf("world Capability ID %q rejected=%t, want %t", tc.id, found, tc.reject)
			}
		})
	}
}
