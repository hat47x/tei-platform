package generatorselection

import (
	"fmt"
	"os"
	"sort"
	"strings"
)

const (
	// RegistrySchemaV0 is the original registry shape. It understands only
	// active and revoked entries and has no replacement history.
	RegistrySchemaV0 = "tei.reference.generator-registry/v0"
	// RegistrySchemaV1 adds superseded entries, replacement links, and reasons.
	// A v1 registry is still only a project-local constraint, not an authority.
	RegistrySchemaV1 = "tei.reference.generator-registry/v1"
	// RegistrySchema is the schema written by new fixtures and tools.
	RegistrySchema = RegistrySchemaV1
)

const (
	// RegistryActive marks the one version a project currently treats as the
	// usable generator for an id.
	RegistryActive = "active"
	// RegistrySuperseded marks a version replaced by a newer one. It must have
	// exactly one successor in the same registry.
	RegistrySuperseded = "superseded"
	// RegistryRevoked marks a version that must not be used. A revoked version
	// may or may not have a successor.
	RegistryRevoked = "revoked"
)

type RegistryEntry struct {
	ID           string `json:"id"`
	Version      string `json:"version"`
	Status       string `json:"status"`
	Language     string `json:"language,omitempty"`
	SourceDigest string `json:"sourceDigest"`
	Supersedes   string `json:"supersedes,omitempty"`
	Reason       string `json:"reason,omitempty"`
	Note         string `json:"note,omitempty"`
}

type Registry struct {
	Schema          string          `json:"schema"`
	ContractID      string          `json:"contractId"`
	ContractVersion string          `json:"contractVersion"`
	ContractDigest  string          `json:"contractDigest"`
	Entries         []RegistryEntry `json:"entries"`
	Note            string          `json:"note,omitempty"`
}

type registryKey struct {
	id      string
	version string
}

// LoadRegistry validates a registry against the contract it names. A registry
// does not grant authority; it only says which generator versions the project
// currently treats as active.
func LoadRegistry(registryPath, contractPath string) (Registry, error) {
	var contract Contract
	if _, err := readJSON(contractPath, &contract); err != nil {
		return Registry{}, err
	}
	if err := validateContract(contract); err != nil {
		return Registry{}, err
	}
	contractRaw, err := os.ReadFile(contractPath)
	if err != nil {
		return Registry{}, err
	}
	var registry Registry
	if _, err := readJSON(registryPath, &registry); err != nil {
		return Registry{}, err
	}
	if registry.Schema != RegistrySchemaV0 && registry.Schema != RegistrySchemaV1 {
		return Registry{}, fmt.Errorf("registry schema %q is neither %q nor %q",
			registry.Schema, RegistrySchemaV0, RegistrySchemaV1)
	}
	if registry.ContractID != contract.ID || registry.ContractVersion != contract.Version {
		return Registry{}, fmt.Errorf("registry targets %s/%s, contract is %s/%s",
			registry.ContractID, registry.ContractVersion, contract.ID, contract.Version)
	}
	if registry.ContractDigest != digest(contractRaw) {
		return Registry{}, fmt.Errorf("registry contract digest does not match %s", contractPath)
	}
	if len(registry.Entries) == 0 {
		return Registry{}, fmt.Errorf("registry has no entries")
	}
	if err := validateRegistryEntries(registry); err != nil {
		return Registry{}, err
	}
	return registry, nil
}

func validateRegistryEntries(registry Registry) error {
	entries := map[registryKey]RegistryEntry{}
	active := map[string]int{}
	v1 := registry.Schema == RegistrySchemaV1
	for _, entry := range registry.Entries {
		if strings.TrimSpace(entry.ID) == "" || strings.TrimSpace(entry.Version) == "" {
			return fmt.Errorf("registry entry id and version are required")
		}
		switch entry.Status {
		case RegistryActive, RegistryRevoked:
		case RegistrySuperseded:
			if !v1 {
				return fmt.Errorf("registry entry %s@%s uses status %q; it requires %s",
					entry.ID, entry.Version, entry.Status, RegistrySchemaV1)
			}
		default:
			return fmt.Errorf("registry entry %s@%s has unknown status %q", entry.ID, entry.Version, entry.Status)
		}
		if !digestPattern.MatchString(entry.SourceDigest) {
			return fmt.Errorf("registry entry %s@%s has no source digest", entry.ID, entry.Version)
		}
		if entry.Language != "" && entry.Language != LanguageGo && entry.Language != LanguagePython {
			return fmt.Errorf("registry entry %s@%s has unsupported language %q", entry.ID, entry.Version, entry.Language)
		}
		if !v1 && (entry.Supersedes != "" || entry.Reason != "") {
			return fmt.Errorf("registry entry %s@%s uses replacement fields; they require %s",
				entry.ID, entry.Version, RegistrySchemaV1)
		}
		if v1 && entry.Status == RegistryRevoked && strings.TrimSpace(entry.Reason) == "" {
			return fmt.Errorf("revoked registry entry %s@%s must state a reason", entry.ID, entry.Version)
		}
		key := registryKey{id: entry.ID, version: entry.Version}
		if _, ok := entries[key]; ok {
			return fmt.Errorf("duplicate registry entry %s@%s", entry.ID, entry.Version)
		}
		entries[key] = entry
		if entry.Status == RegistryActive {
			active[entry.ID]++
		}
	}
	for id, count := range active {
		if count > 1 {
			return fmt.Errorf("generator %s has multiple active versions", id)
		}
	}
	if !v1 {
		return nil
	}
	supersededBy := map[registryKey]registryKey{}
	for _, entry := range registry.Entries {
		if entry.Supersedes == "" {
			continue
		}
		if entry.Supersedes == entry.Version {
			return fmt.Errorf("registry entry %s@%s supersedes itself", entry.ID, entry.Version)
		}
		targetKey := registryKey{id: entry.ID, version: entry.Supersedes}
		target, ok := entries[targetKey]
		if !ok {
			return fmt.Errorf("registry entry %s@%s supersedes missing version %s", entry.ID, entry.Version, entry.Supersedes)
		}
		if target.Status == RegistryActive {
			return fmt.Errorf("active registry entry %s@%s cannot be superseded", target.ID, target.Version)
		}
		if existing, ok := supersededBy[targetKey]; ok {
			return fmt.Errorf("registry entry %s@%s is superseded by both %s and %s",
				target.ID, target.Version, existing.version, entry.Version)
		}
		supersededBy[targetKey] = registryKey{id: entry.ID, version: entry.Version}
	}
	for _, entry := range registry.Entries {
		key := registryKey{id: entry.ID, version: entry.Version}
		if entry.Status == RegistrySuperseded {
			if _, ok := supersededBy[key]; !ok {
				return fmt.Errorf("superseded registry entry %s@%s has no successor", entry.ID, entry.Version)
			}
		}
	}
	for _, entry := range registry.Entries {
		seen := map[registryKey]bool{}
		current := registryKey{id: entry.ID, version: entry.Version}
		for {
			if seen[current] {
				return fmt.Errorf("registry replacement chain for %s@%s contains a cycle", entry.ID, entry.Version)
			}
			seen[current] = true
			successor, ok := supersededBy[current]
			if !ok {
				if entries[current].Status == RegistrySuperseded {
					return fmt.Errorf("registry replacement chain for %s@%s ends at a superseded entry",
						entry.ID, entry.Version)
				}
				break
			}
			current = successor
		}
	}
	return nil
}

// ActiveEntry returns the single active version for id. Revoked, superseded,
// missing, and ambiguous entries fail closed and are distinguished so a caller
// does not report every lifecycle state as the same failure.
func ActiveEntry(registry Registry, id string) (RegistryEntry, error) {
	var active []RegistryEntry
	var revoked []RegistryEntry
	var superseded []RegistryEntry
	for _, entry := range registry.Entries {
		if entry.ID != id {
			continue
		}
		switch entry.Status {
		case RegistryActive:
			active = append(active, entry)
		case RegistryRevoked:
			revoked = append(revoked, entry)
		case RegistrySuperseded:
			superseded = append(superseded, entry)
		}
	}
	if len(active) > 1 {
		sort.Slice(active, func(i, j int) bool { return active[i].Version < active[j].Version })
		return RegistryEntry{}, fmt.Errorf("generator %s has multiple active versions", id)
	}
	if len(active) == 0 {
		if len(revoked) > 0 {
			return RegistryEntry{}, fmt.Errorf("generator %s is revoked", id)
		}
		if len(superseded) > 0 {
			return RegistryEntry{}, fmt.Errorf("generator %s is superseded and has no active version", id)
		}
		return RegistryEntry{}, fmt.Errorf("generator %s is not registered", id)
	}
	return active[0], nil
}

// RegistryIDs returns the distinct generator ids in stable order.
func RegistryIDs(registry Registry) []string {
	seen := map[string]bool{}
	var ids []string
	for _, entry := range registry.Entries {
		if !seen[entry.ID] {
			seen[entry.ID] = true
			ids = append(ids, entry.ID)
		}
	}
	sort.Strings(ids)
	return ids
}

// Entry returns the exact version entry for id, if present.
func Entry(registry Registry, id, version string) (RegistryEntry, bool) {
	for _, entry := range registry.Entries {
		if entry.ID == id && entry.Version == version {
			return entry, true
		}
	}
	return RegistryEntry{}, false
}

func candidateVersion(manifest CandidateManifest) string {
	if manifest.Version == "" {
		return "1"
	}
	return manifest.Version
}
