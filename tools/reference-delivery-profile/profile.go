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
	Entrypoints   []string `json:"entrypoints,omitempty"`
	Guarantees    []string `json:"guarantees,omitempty"`
	NotGuaranteed []string `json:"notGuaranteed,omitempty"`
}

type PluginSummary struct {
	ID           string   `json:"id"`
	Version      string   `json:"version"`
	Path         string   `json:"path"`
	Capabilities []string `json:"capabilities"`
}

type Receipt struct {
	Schema               string          `json:"schema"`
	ProfileID            string          `json:"profileId"`
	ProfileVersion       string          `json:"profileVersion"`
	RequiredCapabilities []string        `json:"requiredCapabilities"`
	Plugins              []PluginSummary `json:"plugins"`
	ProvidedCapabilities []string        `json:"providedCapabilities"`
	Missing              []string        `json:"missing"`
	Passed               bool            `json:"passed"`
	Notes                []string        `json:"notes"`
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
		if strings.TrimSpace(ref.ID) == "" || strings.TrimSpace(ref.Version) == "" || strings.TrimSpace(ref.Path) == "" {
			return Receipt{}, fmt.Errorf("plugin id, version, and path are required")
		}
		if seenPlugin[ref.ID] {
			return Receipt{}, fmt.Errorf("duplicate plugin id %q", ref.ID)
		}
		seenPlugin[ref.ID] = true
		pluginPath := ref.Path
		if !filepath.IsAbs(pluginPath) {
			pluginPath = filepath.Join(profileDir, filepath.FromSlash(pluginPath))
		}
		var plugin Plugin
		if err := readJSON(pluginPath, &plugin); err != nil {
			return Receipt{}, err
		}
		if plugin.Schema != PluginSchema {
			return Receipt{}, fmt.Errorf("plugin %s schema %q is not %q", ref.ID, plugin.Schema, PluginSchema)
		}
		if plugin.ID != ref.ID {
			return Receipt{}, fmt.Errorf("plugin path %s declares id %q, profile requires %q", pluginPath, plugin.ID, ref.ID)
		}
		if plugin.Version != ref.Version {
			return Receipt{}, fmt.Errorf("plugin %s version %q does not match profile version %q", ref.ID, plugin.Version, ref.Version)
		}
		if len(plugin.Capabilities) == 0 {
			return Receipt{}, fmt.Errorf("plugin %s declares no capabilities", ref.ID)
		}
		capabilities := uniqueSorted(plugin.Capabilities)
		provided = append(provided, capabilities...)
		receipt.Plugins = append(receipt.Plugins, PluginSummary{
			ID: ref.ID, Version: ref.Version, Path: ref.Path, Capabilities: capabilities,
		})
	}
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
