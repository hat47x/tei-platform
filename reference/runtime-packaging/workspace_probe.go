package runtimepackaging

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"tei/internal/core"
	"tei/internal/environment"
	"tei/internal/ir"
	"tei/internal/plugin"
	"tei/internal/workspace"
)

// DatabaseProviderObservation records one existing Application/Environment
// database binding and its exact Plugin Lock selection. It is not an artifact.
type DatabaseProviderObservation struct {
	Alias             string `json:"alias"`
	Database          string `json:"database"`
	Instance          string `json:"instance"`
	Provider          string `json:"provider"`
	Capability        string `json:"capability"`
	Plugin            string `json:"plugin"`
	PluginVersion     string `json:"pluginVersion"`
	PluginFingerprint string `json:"pluginFingerprint"`
	ArtifactDigest    string `json:"artifactDigest,omitempty"`
}
type WorkspaceDatabaseReport struct {
	Application  string                        `json:"application"`
	Environment  string                        `json:"environment"`
	Providers    []DatabaseProviderObservation `json:"providers"`
}

// ObserveWorkspaceDatabaseProviders reuses the existing TEI source loader and
// locked plugin resolver; it creates neither a new authority nor a resolver.
func ObserveWorkspaceDatabaseProviders(root, applicationID, environmentID string) (WorkspaceDatabaseReport, error) {
	loaded, err := workspace.Load(root)
	if err != nil {
		return WorkspaceDatabaseReport{}, fmt.Errorf("load workspace: %w", err)
	}
	raw, err := os.ReadFile(filepath.Join(root, ".tei", "plugin-lock.json"))
	if err != nil {
		return WorkspaceDatabaseReport{}, fmt.Errorf("read runtime plugin lock: %w", err)
	}
	var lock plugin.Lock
	if err := json.Unmarshal(raw, &lock); err != nil {
		return WorkspaceDatabaseReport{}, fmt.Errorf("decode runtime plugin lock: %w", err)
	}
	return ObserveDatabaseProviders(loaded.World, lock, applicationID, environmentID)
}

// ObserveDatabaseProviders establishes only exact database provider wiring;
// it does not derive the business Action implementation or Binary Build Plan.
func ObserveDatabaseProviders(world ir.World, lock plugin.Lock, applicationID, environmentID string) (WorkspaceDatabaseReport, error) {
	appID, err := core.ParseCanonicalID(applicationID)
	if err != nil {
		return WorkspaceDatabaseReport{}, fmt.Errorf("application: %w", err)
	}
	envID, err := core.ParseCanonicalID(environmentID)
	if err != nil {
		return WorkspaceDatabaseReport{}, fmt.Errorf("environment: %w", err)
	}
	var appCount, envCount, appIdx, envIdx int
	for i := range world.Applications {
		if world.Applications[i].ID == appID {
			appIdx, appCount = i, appCount+1
		}
	}
	for i := range world.Environments {
		if world.Environments[i].ID == envID {
			envIdx, envCount = i, envCount+1
		}
	}
	if appCount != 1 || envCount != 1 {
		return WorkspaceDatabaseReport{}, fmt.Errorf("expected one application %s and environment %s; found %d and %d", appID, envID, appCount, envCount)
	}
	app, env := world.Applications[appIdx], world.Environments[envIdx]
	resolved, err := (plugin.Catalog{Plugins: world.Plugins, Bundles: world.Bundles}).ResolveEnvironmentWithLock(env, lock)
	if err != nil {
		return WorkspaceDatabaseReport{}, fmt.Errorf("locked resolution: %w", err)
	}
	locked := map[plugin.Identity]plugin.LockedPlugin{}
	for _, el := range lock.Environments {
		if el.Environment != env.ID {
			continue
		}
		for _, lp := range el.Plugins {
			key := plugin.Identity{ID: lp.ID, Version: lp.Version}
			if _, ok := locked[key]; ok {
				return WorkspaceDatabaseReport{}, fmt.Errorf("duplicate locked plugin %s@%s", lp.ID, lp.Version)
			}
			locked[key] = lp
		}
	}
	out := WorkspaceDatabaseReport{Application: applicationID, Environment: environmentID, Providers: []DatabaseProviderObservation{}}
	seen := map[string]bool{}
	for _, req := range app.Databases {
		alias := string(req.Alias)
		if seen[alias] {
			return WorkspaceDatabaseReport{}, fmt.Errorf("duplicate database alias %s", alias)
		}
		seen[alias] = true
		var instanceID core.CanonicalID
		count := 0
		for _, b := range env.Bindings {
			if b.Application == app.ID && b.Alias == req.Alias {
				instanceID, count = b.Instance, count+1
			}
		}
		if count != 1 {
			return WorkspaceDatabaseReport{}, fmt.Errorf("database alias %s has %d bindings", alias, count)
		}
		var instance environment.DatabaseInstance
		count = 0
		for _, candidate := range env.Instances {
			if candidate.ID == instanceID {
				instance, count = candidate, count+1
			}
		}
		if count != 1 || instance.Definition != req.TargetDatabase {
			return WorkspaceDatabaseReport{}, fmt.Errorf("database alias %s has no unique matching instance", alias)
		}
		var selected plugin.ResolvedCapability
		count = 0
		for _, c := range resolved.Capabilities {
			if c.Capability.Kind == plugin.CapabilityDatabaseProvider && c.Capability.SupportsDatabaseProvider(instance.Provider) {
				selected, count = c, count+1
			}
		}
		if count != 1 {
			return WorkspaceDatabaseReport{}, fmt.Errorf("database alias %s has %d providers for %s", alias, count, instance.Provider)
		}
		lp, ok := locked[selected.Plugin]
		if !ok {
			return WorkspaceDatabaseReport{}, fmt.Errorf("database alias %s has no locked plugin identity", alias)
		}
		out.Providers = append(out.Providers, DatabaseProviderObservation{
			Alias: alias, Database: string(req.TargetDatabase), Instance: string(instance.ID),
			Provider: string(instance.Provider), Capability: string(selected.Capability.ID),
			Plugin: string(lp.ID), PluginVersion: lp.Version, PluginFingerprint: lp.Fingerprint,
			ArtifactDigest: lp.ArtifactDigest,
		})
	}
	sort.Slice(out.Providers, func(i, j int) bool { return out.Providers[i].Alias < out.Providers[j].Alias })
	return out, nil
}
