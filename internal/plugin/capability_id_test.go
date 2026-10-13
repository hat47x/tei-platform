package plugin_test

import (
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/plugin"
)

func TestCapabilityIDRejectedInDirectPluginPackage(t *testing.T) {
	for _, tc := range []struct {
		name   string
		id     string
		reject bool
	}{
		{name: "valid canonical id", id: "runtime.valid"},
		{name: "empty id", id: "", reject: true},
		{name: "id with spaces", id: "not a valid id", reject: true},
		{name: "whitespace-padded id", id: " runtime.valid ", reject: true},
		{name: "uppercase id", id: "Runtime.valid", reject: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			definition := artifactFreePackageSource()
			definition.Provides[0].ID = core.CanonicalID(tc.id)

			err := plugin.ValidateCapabilityID(definition.Provides[0].ID)
			if tc.reject {
				if err == nil || !strings.Contains(err.Error(), "invalid canonical id") {
					t.Fatalf("capability id %q was not rejected: %v", tc.id, err)
				}
			} else if err != nil {
				t.Fatalf("valid capability id rejected: %v", err)
			}

			_, err = plugin.MaterializePluginPackage(definition, pluginPackagePlan(definition))
			if tc.reject {
				if err == nil || !strings.Contains(err.Error(), "invalid canonical id") {
					t.Fatalf("direct package accepted invalid capability id %q: %v", tc.id, err)
				}
			} else if err != nil {
				t.Fatalf("valid capability package rejected: %v", err)
			}
		})
	}
}
