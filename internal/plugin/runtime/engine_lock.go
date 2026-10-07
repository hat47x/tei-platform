package runtime

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"

	"tei/internal/core"
	"tei/internal/plugin"
)

const RuntimeEngineLockFormatVersion = 1

type RuntimeEngineBinding struct {
	Plugin plugin.Identity       `json:"plugin"`
	Engine RuntimeEngineIdentity `json:"engine"`
}

type RuntimeEngineLock struct {
	FormatVersion int                    `json:"formatVersion"`
	Bindings      []RuntimeEngineBinding `json:"bindings"`
}

func NormalizeRuntimeEngineLock(lock RuntimeEngineLock) RuntimeEngineLock {
	out := lock
	out.Bindings = append([]RuntimeEngineBinding(nil), lock.Bindings...)
	sort.Slice(out.Bindings, func(i, j int) bool {
		if out.Bindings[i].Plugin.ID != out.Bindings[j].Plugin.ID {
			return out.Bindings[i].Plugin.ID < out.Bindings[j].Plugin.ID
		}
		return out.Bindings[i].Plugin.Version < out.Bindings[j].Plugin.Version
	})
	return out
}

func (lock RuntimeEngineLock) Validate() error {
	if lock.FormatVersion != RuntimeEngineLockFormatVersion {
		return fmt.Errorf("unsupported runtime engine lock format %d", lock.FormatVersion)
	}
	seen := map[string]struct{}{}
	for _, binding := range lock.Bindings {
		if binding.Plugin.ID == "" {
			return fmt.Errorf("runtime engine lock plugin id is required")
		}
		if _, err := core.ParseCanonicalID(string(binding.Plugin.ID)); err != nil {
			return fmt.Errorf("runtime engine lock plugin id: %w", err)
		}
		if !core.ValidNumericVersion(binding.Plugin.Version) {
			return fmt.Errorf(
				"runtime engine lock plugin version %q must be major.minor.patch",
				binding.Plugin.Version,
			)
		}
		if err := binding.Engine.Validate(); err != nil {
			return fmt.Errorf(
				"runtime engine lock for %s@%s: %w",
				binding.Plugin.ID,
				binding.Plugin.Version,
				err,
			)
		}
		key := identityKey(binding.Plugin)
		if _, ok := seen[key]; ok {
			return fmt.Errorf("runtime engine lock plugin %s is declared more than once", key)
		}
		seen[key] = struct{}{}
	}
	return nil
}

func RuntimeEngineLockFingerprint(lock RuntimeEngineLock) (string, error) {
	if err := lock.Validate(); err != nil {
		return "", err
	}
	normalized := NormalizeRuntimeEngineLock(lock)
	raw, err := json.Marshal(normalized)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:]), nil
}

// BindRuntimeEngineLock applies exact Runtime Engine expectations to a derived
// EnvironmentPlan. The lock is separate from Plugin Lock because an
// interpreter/host engine is neither the logical Plugin identity nor a
// build-time dependency of the Plugin realization.
func BindRuntimeEngineLock(plan EnvironmentPlan, lock RuntimeEngineLock) (EnvironmentPlan, error) {
	lockFingerprint, err := RuntimeEngineLockFingerprint(lock)
	if err != nil {
		return EnvironmentPlan{}, err
	}
	out := plan
	out.Plugins = append([]PluginRuntime(nil), plan.Plugins...)
	for i := range out.Plugins {
		if plan.Plugins[i].RuntimeEngine != nil {
			copy := *plan.Plugins[i].RuntimeEngine
			out.Plugins[i].RuntimeEngine = &copy
		}
	}

	for _, binding := range NormalizeRuntimeEngineLock(lock).Bindings {
		out, err = bindBuiltinRuntimeEngine(out, binding.Plugin, binding.Engine)
		if err != nil {
			return EnvironmentPlan{}, err
		}
	}
	for _, pr := range out.Plugins {
		if pr.Boundary == BoundaryInProcess && pr.BuildPlanFingerprint != "" && pr.RuntimeEngine == nil {
			return EnvironmentPlan{}, fmt.Errorf(
				"build-bound builtin plugin %s@%s is missing runtime engine lock binding",
				pr.Plugin.ID,
				pr.Plugin.Version,
			)
		}
	}
	out.RuntimeEngineLockFingerprint = lockFingerprint
	out.RuntimePolicyFingerprint = PolicyFingerprint(out)
	return out, nil
}
