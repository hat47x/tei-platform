package plugin

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"

	"tei/internal/core"
	"tei/internal/environment"
)

const (
	// LockFormatVersion is the legacy/current unbound Runtime Plugin Lock
	// format. It is kept for compatibility with existing checked-in locks.
	LockFormatVersion = 2
	// BuildTraceLockFormatVersion adds a per-Plugin BuildPlan fingerprint so a
	// Runtime lock can trace the selected implementation back to exact build
	// inputs. A lock is upgraded to this format only when build provenance is
	// explicitly bound.
	BuildTraceLockFormatVersion = 3
)

type LockedPlugin struct {
	ID                   core.CanonicalID `json:"id"`
	Version              string           `json:"version"`
	Fingerprint          string           `json:"fingerprint"`
	ArtifactDigest       string           `json:"artifactDigest,omitempty"`
	BuildPlanFingerprint string           `json:"buildPlanFingerprint,omitempty"`
}

type EnvironmentLock struct {
	Environment core.CanonicalID `json:"environment"`
	Plugins     []LockedPlugin   `json:"plugins"`
}

type Lock struct {
	FormatVersion int               `json:"formatVersion"`
	Environments  []EnvironmentLock `json:"environments"`
}

func DefinitionFingerprint(d Definition) string {
	n := NormalizeDefinition(d)
	b, _ := json.Marshal(n)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

func BuildLock(c Catalog, envs []environment.Environment) (Lock, error) {
	out := Lock{FormatVersion: LockFormatVersion}
	for _, env := range envs {
		r, err := c.ResolveEnvironment(env)
		if err != nil {
			return Lock{}, fmt.Errorf("environment %s: %w", env.ID, err)
		}
		el := EnvironmentLock{Environment: env.ID}
		for _, ident := range r.Plugins {
			def, ok := exactDefinition(c.Plugins, ident)
			if !ok {
				return Lock{}, fmt.Errorf("resolved plugin %s@%s disappeared from catalog", ident.ID, ident.Version)
			}
			el.Plugins = append(el.Plugins, LockedPlugin{ID: ident.ID, Version: ident.Version, Fingerprint: DefinitionFingerprint(def), ArtifactDigest: def.Artifact.Digest})
		}
		out.Environments = append(out.Environments, el)
	}
	sort.Slice(out.Environments, func(i, j int) bool { return out.Environments[i].Environment < out.Environments[j].Environment })
	return out, nil
}

func exactDefinition(defs []Definition, id Identity) (Definition, bool) {
	for _, d := range defs {
		if d.ID == id.ID && d.Version == id.Version {
			return d, true
		}
	}
	return Definition{}, false
}

func supportedLockFormatVersion(version int) bool {
	return version == LockFormatVersion || version == BuildTraceLockFormatVersion
}

func validateLockBuildTrace(lock Lock) error {
	for _, environmentLock := range lock.Environments {
		for _, locked := range environmentLock.Plugins {
			if locked.BuildPlanFingerprint == "" {
				continue
			}
			if lock.FormatVersion != BuildTraceLockFormatVersion {
				return fmt.Errorf(
					"plugin lock format %d cannot contain build plan fingerprint for %s@%s",
					lock.FormatVersion, locked.ID, locked.Version,
				)
			}
			if locked.ArtifactDigest == "" {
				return fmt.Errorf(
					"plugin lock build plan fingerprint for %s@%s requires an artifact digest",
					locked.ID, locked.Version,
				)
			}
			if err := validateHexFingerprint(
				"plugin lock build plan fingerprint for "+string(locked.ID)+"@"+locked.Version,
				locked.BuildPlanFingerprint,
			); err != nil {
				return err
			}
		}
	}
	return nil
}

func (l Lock) environment(id core.CanonicalID) (EnvironmentLock, bool) {
	for _, e := range l.Environments {
		if e.Environment == id {
			return e, true
		}
	}
	return EnvironmentLock{}, false
}

// EnvironmentLockFingerprint returns a stable fingerprint for the exact plugin
// identities and descriptor fingerprints pinned for one environment. Durable
// execution artifacts bind to this value rather than to a mutable catalog.
func EnvironmentLockFingerprint(l Lock, environmentID core.CanonicalID) (string, error) {
	if !supportedLockFormatVersion(l.FormatVersion) {
		return "", fmt.Errorf("unsupported plugin lock format %d", l.FormatVersion)
	}
	if err := validateLockBuildTrace(l); err != nil {
		return "", err
	}
	el, ok := l.environment(environmentID)
	if !ok {
		return "", fmt.Errorf("plugin lock has no environment %s", environmentID)
	}
	n := EnvironmentLock{Environment: el.Environment, Plugins: append([]LockedPlugin(nil), el.Plugins...)}
	sort.Slice(n.Plugins, func(i, j int) bool {
		if n.Plugins[i].ID != n.Plugins[j].ID {
			return n.Plugins[i].ID < n.Plugins[j].ID
		}
		return n.Plugins[i].Version < n.Plugins[j].Version
	})
	b, _ := json.Marshal(n)
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:]), nil
}

func (c Catalog) ResolveEnvironmentWithLock(env environment.Environment, lock Lock) (ResolvedSet, error) {
	if !supportedLockFormatVersion(lock.FormatVersion) {
		return ResolvedSet{}, fmt.Errorf("unsupported plugin lock format %d", lock.FormatVersion)
	}
	if err := validateLockBuildTrace(lock); err != nil {
		return ResolvedSet{}, err
	}
	el, ok := lock.environment(env.ID)
	if !ok {
		return ResolvedSet{}, fmt.Errorf("plugin lock has no environment %s", env.ID)
	}
	var subset []Definition
	for _, lp := range el.Plugins {
		def, ok := exactDefinition(c.Plugins, Identity{ID: lp.ID, Version: lp.Version})
		if !ok {
			return ResolvedSet{}, fmt.Errorf("locked plugin %s@%s is not available", lp.ID, lp.Version)
		}
		if got := DefinitionFingerprint(def); got != lp.Fingerprint {
			return ResolvedSet{}, fmt.Errorf("locked plugin %s@%s fingerprint diverged", lp.ID, lp.Version)
		}
		if lp.ArtifactDigest != def.Artifact.Digest {
			return ResolvedSet{}, fmt.Errorf("locked plugin %s@%s artifact digest diverged", lp.ID, lp.Version)
		}
		subset = append(subset, def)
	}
	r, err := (Catalog{Plugins: subset, Bundles: c.Bundles}).ResolveEnvironment(env)
	if err != nil {
		return ResolvedSet{}, fmt.Errorf("locked resolution for %s: %w", env.ID, err)
	}
	if len(r.Plugins) != len(el.Plugins) {
		return ResolvedSet{}, fmt.Errorf("plugin lock for %s contains a different plugin set than current requirements", env.ID)
	}
	for i := range r.Plugins {
		if r.Plugins[i].ID != el.Plugins[i].ID || r.Plugins[i].Version != el.Plugins[i].Version {
			return ResolvedSet{}, fmt.Errorf("plugin lock for %s does not match current requirements", env.ID)
		}
	}
	return r, nil
}
