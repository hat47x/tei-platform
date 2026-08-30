package runtime_test

import (
	"testing"

	"tei/internal/core"
	"tei/internal/plugin"
	pruntime "tei/internal/plugin/runtime"
)

func TestProcessArgsChangeRuntimePolicyFingerprint(t *testing.T) {
	identity := plugin.Identity{ID: core.MustCanonicalID("tei.process.reference"), Version: "1.0.0"}
	base := pruntime.EnvironmentPlan{
		Environment:           core.MustCanonicalID("reference"),
		PluginLockFingerprint: "lock-fingerprint",
		Plugins: []pruntime.PluginRuntime{{
			Plugin:                identity,
			DescriptorFingerprint: "descriptor-fingerprint",
			Boundary:              pruntime.BoundaryProcess,
			Trust:                 pruntime.TrustExternalProcess,
			Command:               "python3",
			Args:                  []string{"-m", "worker-a"},
		}},
	}
	changed := base
	changed.Plugins = append([]pruntime.PluginRuntime(nil), base.Plugins...)
	changed.Plugins[0].Args = []string{"-m", "worker-b"}

	if got, want := pruntime.PolicyFingerprint(base), pruntime.PolicyFingerprint(changed); got == want {
		t.Fatal("process argv change did not change runtime policy fingerprint")
	}
}
