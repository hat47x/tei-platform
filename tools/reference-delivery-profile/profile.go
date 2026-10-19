// Package deliveryprofile validates a Delivery Profile's plugin selection. The
// profile declares required capabilities and the plugins that provide them; it
// does not implement any delivery form itself.
package deliveryprofile

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	ProfileSchema = "tei.reference.delivery-profile/v0"
	PluginSchema  = "tei.reference.delivery-plugin/v0"
)

type PluginRef struct {
	ID      string `json:"id"`
	Version string `json:"version"`
	Path    string `json:"path"`
}

type Profile struct {
	Schema               string      `json:"schema"`
	ID                   string      `json:"id"`
	Version              string      `json:"version"`
	RequiredCapabilities []string    `json:"requiredCapabilities"`
	Plugins              []PluginRef `json:"plugins"`
}

type Plugin struct {
	Schema        string   `json:"schema"`
	ID            string   `json:"id"`
	Version       string   `json:"version"`
	Kind          string   `json:"kind"`
	Capabilities  []string `json:"capabilities"`
	Requires      []string `json:"requires,omitempty"`
	Conflicts     []string `json:"conflicts,omitempty"`
	Entrypoints   []string `json:"entrypoints,omitempty"`
	Guarantees    []string `json:"guarantees,omitempty"`
	NotGuaranteed []string `json:"notGuaranteed,omitempty"`
}

type PluginSummary struct {
	ID           string   `json:"id"`
	Version      string   `json:"version"`
	Path         string   `json:"path"`
	Digest       string   `json:"digest"`
	Capabilities []string `json:"capabilities"`
	Requires     []string `json:"requires,omitempty"`
	Conflicts    []string `json:"conflicts,omitempty"`
}

type Receipt struct {
	Schema               string            `json:"schema"`
	ProfileID            string            `json:"profileId"`
	ProfileVersion       string            `json:"profileVersion"`
	RequiredCapabilities []string          `json:"requiredCapabilities"`
	Plugins              []PluginSummary   `json:"plugins"`
	Order                []string          `json:"order"`
	ProvidedCapabilities []string          `json:"providedCapabilities"`
	Missing              []string          `json:"missing"`
	Conformance          []CapabilityCheck `json:"conformance,omitempty"`
	Passed               bool              `json:"passed"`
	Notes                []string          `json:"notes"`
}

func readJSON(path string, into any) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(into); err != nil {
		return fmt.Errorf("%s: %w", path, err)
	}
	return nil
}

func uniqueSorted(values []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

// Validate loads the profile and every referenced plugin. A missing required
// capability fails closed; it is not replaced by another plugin or a per-form
// implementation.
func Validate(profilePath string) (Receipt, error) {
	return ValidateWithCatalog(profilePath, "")
}

// ValidateWithCatalog resolves plugin references that omit path from the
// catalog directory. A path reference still loads that exact descriptor; the
// catalog is only used for references without a path.
func ValidateWithCatalog(profilePath, catalogDir string) (Receipt, error) {
	var catalog Catalog
	if catalogDir != "" {
		loaded, err := LoadCatalog(catalogDir)
		if err != nil {
			return Receipt{}, err
		}
		catalog = loaded
	}
	var profile Profile
	if err := readJSON(profilePath, &profile); err != nil {
		return Receipt{}, err
	}
	if profile.Schema != ProfileSchema {
		return Receipt{}, fmt.Errorf("profile schema %q is not %q", profile.Schema, ProfileSchema)
	}
	if strings.TrimSpace(profile.ID) == "" || strings.TrimSpace(profile.Version) == "" {
		return Receipt{}, fmt.Errorf("profile id and version are required")
	}
	if len(profile.RequiredCapabilities) == 0 {
		return Receipt{}, fmt.Errorf("requiredCapabilities must not be empty")
	}
	if len(profile.Plugins) == 0 {
		return Receipt{}, fmt.Errorf("plugins must not be empty")
	}
	profileDir := filepath.Dir(profilePath)
	receipt := Receipt{
		Schema: ProfileSchema, ProfileID: profile.ID, ProfileVersion: profile.Version,
		RequiredCapabilities: uniqueSorted(profile.RequiredCapabilities),
		Plugins:              []PluginSummary{}, ProvidedCapabilities: []string{}, Missing: []string{},
		Notes: []string{
			"Profileはpluginの選択と能力の検証だけを行い、提供形態の実装を持たない。",
			"pluginの能力宣言は、実機での保証を証明しない。",
		},
	}
	seenPlugin := map[string]bool{}
	var provided []string
	for _, ref := range profile.Plugins {
		if strings.TrimSpace(ref.ID) == "" || strings.TrimSpace(ref.Version) == "" {
			return Receipt{}, fmt.Errorf("plugin id and version are required")
		}
		if seenPlugin[ref.ID] {
			return Receipt{}, fmt.Errorf("duplicate plugin id %q", ref.ID)
		}
		seenPlugin[ref.ID] = true
		var plugin Plugin
		var pluginPath, pluginDigest string
		if strings.TrimSpace(ref.Path) != "" {
			pluginPath = ref.Path
			if !filepath.IsAbs(pluginPath) {
				pluginPath = filepath.Join(profileDir, filepath.FromSlash(pluginPath))
			}
			raw, err := os.ReadFile(pluginPath)
			if err != nil {
				return Receipt{}, err
			}
			plugin, err = decodePlugin(pluginPath, raw)
			if err != nil {
				return Receipt{}, err
			}
			pluginDigest = digestBytes(raw)
			if plugin.ID != ref.ID {
				return Receipt{}, fmt.Errorf("plugin path %s declares id %q, profile requires %q", pluginPath, plugin.ID, ref.ID)
			}
			if plugin.Version != ref.Version {
				return Receipt{}, fmt.Errorf("plugin %s version %q does not match profile version %q", ref.ID, plugin.Version, ref.Version)
			}
		} else {
			if catalog == nil {
				return Receipt{}, fmt.Errorf("plugin %s has no path; -catalog is required", ref.ID)
			}
			entry, err := catalog.Resolve(ref.ID, ref.Version)
			if err != nil {
				return Receipt{}, err
			}
			plugin, pluginPath, pluginDigest = entry.Plugin, entry.Path, entry.Digest
		}
		capabilities := uniqueSorted(plugin.Capabilities)
		provided = append(provided, capabilities...)
		receipt.Plugins = append(receipt.Plugins, PluginSummary{
			ID: ref.ID, Version: ref.Version, Path: pluginPath, Digest: pluginDigest,
			Capabilities: capabilities,
			Requires:     uniqueSorted(plugin.Requires), Conflicts: uniqueSorted(plugin.Conflicts),
		})
	}
	byID := map[string]PluginSummary{}
	for _, plugin := range receipt.Plugins {
		byID[plugin.ID] = plugin
	}
	for _, plugin := range receipt.Plugins {
		for _, dependency := range plugin.Requires {
			if _, ok := byID[dependency]; !ok {
				return Receipt{}, fmt.Errorf("plugin %s requires %s, but %s is not selected", plugin.ID, dependency, dependency)
			}
		}
		for _, conflict := range plugin.Conflicts {
			if _, ok := byID[conflict]; ok {
				return Receipt{}, fmt.Errorf("plugin %s conflicts with selected plugin %s", plugin.ID, conflict)
			}
		}
	}
	order, err := resolveOrder(receipt.Plugins)
	if err != nil {
		return Receipt{}, err
	}
	receipt.Order = order
	receipt.ProvidedCapabilities = uniqueSorted(provided)
	required := map[string]bool{}
	for _, capability := range receipt.RequiredCapabilities {
		required[capability] = true
	}
	for _, capability := range receipt.ProvidedCapabilities {
		delete(required, capability)
	}
	for capability := range required {
		receipt.Missing = append(receipt.Missing, capability)
	}
	sort.Strings(receipt.Missing)
	receipt.Passed = len(receipt.Missing) == 0
	return receipt, nil
}

func resolveOrder(plugins []PluginSummary) ([]string, error) {
	byID := map[string]PluginSummary{}
	for _, plugin := range plugins {
		byID[plugin.ID] = plugin
	}
	state := map[string]int{}
	order := []string{}
	var visit func(string) error
	visit = func(id string) error {
		switch state[id] {
		case 2:
			return nil
		case 1:
			return fmt.Errorf("plugin dependency cycle includes %s", id)
		}
		state[id] = 1
		for _, dependency := range byID[id].Requires {
			if err := visit(dependency); err != nil {
				return err
			}
		}
		state[id] = 2
		order = append(order, id)
		return nil
	}
	ids := make([]string, 0, len(byID))
	for id := range byID {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		if err := visit(id); err != nil {
			return nil, err
		}
	}
	return order, nil
}
