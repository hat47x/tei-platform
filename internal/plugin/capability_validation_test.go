package plugin_test

import (
	"strings"
	"testing"

	"tei/internal/deployment"
	"tei/internal/plugin"
)

func TestValidateUniqueCapabilitiesUsesKindAndID(t *testing.T) {
	shared := cid("runtime.shared")
	cases := []struct {
		name   string
		caps   []plugin.Capability
		reject bool
	}{
		{
			name: "same identity",
			caps: []plugin.Capability{
				{Kind: plugin.CapabilityDeploymentExecutor, ID: shared},
				{Kind: plugin.CapabilityDeploymentExecutor, ID: shared},
			},
			reject: true,
		},
		{
			name: "different details with same identity",
			caps: []plugin.Capability{
				{Kind: plugin.CapabilityDeploymentExecutor, ID: shared, Actions: []deployment.ActionKind{deployment.ActionTrafficShift}},
				{Kind: plugin.CapabilityDeploymentExecutor, ID: shared, Actions: []deployment.ActionKind{deployment.ActionApplicationStart}},
			},
			reject: true,
		},
		{
			name: "same id with different kind",
			caps: []plugin.Capability{
				{Kind: plugin.CapabilityDeploymentExecutor, ID: shared},
				{Kind: plugin.CapabilityObservationProvider, ID: shared},
			},
		},
		{
			name: "different ids with same kind",
			caps: []plugin.Capability{
				{Kind: plugin.CapabilityDeploymentExecutor, ID: shared},
				{Kind: plugin.CapabilityDeploymentExecutor, ID: cid("runtime.other")},
			},
		},
		{name: "empty"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := plugin.ValidateUniqueCapabilities(tc.caps)
			if (err != nil) != tc.reject {
				t.Fatalf("ValidateUniqueCapabilities() = %v; reject = %v", err, tc.reject)
			}
		})
	}
}

func TestValidateUniqueCapabilitiesReportsSameDuplicateInAnyOrder(t *testing.T) {
	a := plugin.Capability{Kind: plugin.CapabilityDeploymentExecutor, ID: cid("runtime.a")}
	z := plugin.Capability{Kind: plugin.CapabilityObservationProvider, ID: cid("runtime.z")}
	first := []plugin.Capability{z, a, z, a}
	second := []plugin.Capability{a, z, a, z}
	err1 := plugin.ValidateUniqueCapabilities(first)
	err2 := plugin.ValidateUniqueCapabilities(second)
	if err1 == nil || err2 == nil || err1.Error() != err2.Error() {
		t.Fatalf("capability duplicate diagnostic depends on ordering: %v, %v", err1, err2)
	}
	if !strings.Contains(err1.Error(), "runtime.a") {
		t.Fatalf("expected lexically first duplicate: %v", err1)
	}
}

func TestStandalonePackageRejectsDuplicateCapability(t *testing.T) {
	source := artifactFreePackageSource()
	conflicting := source.Provides[0]
	conflicting.Actions = []deployment.ActionKind{deployment.ActionTrafficShift}
	source.Provides = append(source.Provides, conflicting)
	if _, err := plugin.MaterializePluginPackage(source, pluginPackagePlan(source)); err == nil || !strings.Contains(err.Error(), "more than once") {
		t.Fatalf("standalone materializer must reject ambiguous capability: %v", err)
	}
}
