package generatorselection

import (
	"fmt"
	"os"
	"sort"
	"strings"
)

const RegistrySchema = "tei.reference.generator-registry/v0"

const (
	RegistryActive  = "active"
	RegistryRevoked = "revoked"
)

type RegistryEntry struct {
	ID           string `json:"id"`
	Version      string `json:"version"`
	Status       string `json:"status"`
	SourceDigest string `json:"sourceDigest"`
	Supersedes   string `json:"supersedes,omitempty"`
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
	if registry.Schema != RegistrySchema {
		return Registry{}, fmt.Errorf("registry schema %q is not %q", registry.Schema, RegistrySchema)
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
	seen := map[string]bool{}
	for _, entry := range registry.Entries {
		if strings.TrimSpace(entry.ID) == "" || strings.TrimSpace(entry.Version) == "" {
			return Registry{}, fmt.Errorf("registry entry id and version are required")
		}
		if entry.Status != RegistryActive && entry.Status != RegistryRevoked {
			return Registry{}, fmt.Errorf("registry entry %s@%s has unknown status %q", entry.ID, entry.Version, entry.Status)
		}
		if !digestPattern.MatchString(entry.SourceDigest) {
			return Registry{}, fmt.Errorf("registry entry %s@%s has no source digest", entry.ID, entry.Version)
		}
		key := entry.ID + "\x00" + entry.Version
		if seen[key] {
			return Registry{}, fmt.Errorf("duplicate registry entry %s@%s", entry.ID, entry.Version)
		}
		seen[key] = true
	}
	return registry, nil
}

// ActiveEntry returns the single active version for id. Revoked, missing, and
// ambiguous entries fail closed.
func ActiveEntry(registry Registry, id string) (RegistryEntry, error) {
	var active []RegistryEntry
	var revoked []RegistryEntry
	for _, entry := range registry.Entries {
		if entry.ID != id {
			continue
		}
		switch entry.Status {
		case RegistryActive:
			active = append(active, entry)
		case RegistryRevoked:
			revoked = append(revoked, entry)
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
		return RegistryEntry{}, fmt.Errorf("generator %s is not registered", id)
	}
	return active[0], nil
}

func candidateVersion(manifest CandidateManifest) string {
	if manifest.Version == "" {
		return "1"
	}
	return manifest.Version
}
