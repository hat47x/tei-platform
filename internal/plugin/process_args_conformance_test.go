package plugin_test

import (
	"testing"

	"tei/internal/core"
	"tei/internal/plugin"
)

func TestProcessArgsChangeDefinitionAndLockFingerprints(t *testing.T) {
	pluginID := core.MustCanonicalID("tei.process.reference")
	environmentID := core.MustCanonicalID("reference")

	base := plugin.Definition{
		ID:      pluginID,
		Version: "1.0.0",
		Runtime: plugin.Runtime{
			Kind:    plugin.RuntimeProcess,
			Command: "python3",
			Args:    []string{"-m", "worker-a"},
		},
	}
	changed := base
	changed.Runtime.Args = []string{"-m", "worker-b"}

	baseFP := plugin.DefinitionFingerprint(base)
	changedFP := plugin.DefinitionFingerprint(changed)
	if baseFP == changedFP {
		t.Fatal("process argv change did not change plugin definition fingerprint")
	}

	baseLock := plugin.Lock{
		FormatVersion: plugin.LockFormatVersion,
		Environments: []plugin.EnvironmentLock{{
			Environment: environmentID,
			Plugins: []plugin.LockedPlugin{{
				ID:          pluginID,
				Version:     "1.0.0",
				Fingerprint: baseFP,
			}},
		}},
	}
	changedLock := baseLock
	changedLock.Environments = []plugin.EnvironmentLock{{
		Environment: environmentID,
		Plugins: []plugin.LockedPlugin{{
			ID:          pluginID,
			Version:     "1.0.0",
			Fingerprint: changedFP,
		}},
	}}

	baseLockFP, err := plugin.EnvironmentLockFingerprint(baseLock, environmentID)
	if err != nil {
		t.Fatal(err)
	}
	changedLockFP, err := plugin.EnvironmentLockFingerprint(changedLock, environmentID)
	if err != nil {
		t.Fatal(err)
	}
	if baseLockFP == changedLockFP {
		t.Fatal("process argv change did not change environment plugin lock fingerprint")
	}
}
