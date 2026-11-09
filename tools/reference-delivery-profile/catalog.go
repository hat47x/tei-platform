package deliveryprofile

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type CatalogEntry struct {
	Plugin Plugin
	Path   string
	Digest string
}

type Catalog map[string]map[string]CatalogEntry

func digestBytes(raw []byte) string {
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func decodePlugin(path string, raw []byte) (Plugin, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	var plugin Plugin
	if err := decoder.Decode(&plugin); err != nil {
		return Plugin{}, fmt.Errorf("%s: %w", path, err)
	}
	if err := validatePlugin(plugin); err != nil {
		return Plugin{}, fmt.Errorf("%s: %w", path, err)
	}
	return plugin, nil
}

func validatePlugin(plugin Plugin) error {
	if plugin.Schema != PluginSchema {
		return fmt.Errorf("schema %q is not %q", plugin.Schema, PluginSchema)
	}
	if strings.TrimSpace(plugin.ID) == "" || strings.TrimSpace(plugin.Version) == "" {
		return fmt.Errorf("plugin id and version are required")
	}
	if len(plugin.Capabilities) == 0 {
		return fmt.Errorf("plugin declares no capabilities")
	}
	return nil
}

// LoadCatalog reads one directory of plugin descriptors. Each id/version pair
// must appear once; duplicate artifacts are rejected instead of guessed.
func LoadCatalog(dir string) (Catalog, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	catalog := Catalog{}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		raw, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		plugin, err := decodePlugin(path, raw)
		if err != nil {
			return nil, err
		}
		if catalog[plugin.ID] == nil {
			catalog[plugin.ID] = map[string]CatalogEntry{}
		}
		if _, exists := catalog[plugin.ID][plugin.Version]; exists {
			return nil, fmt.Errorf("duplicate plugin %s@%s in catalog", plugin.ID, plugin.Version)
		}
		catalog[plugin.ID][plugin.Version] = CatalogEntry{
			Plugin: plugin, Path: path, Digest: digestBytes(raw),
		}
	}
	return catalog, nil
}

func (c Catalog) Resolve(id, version string) (CatalogEntry, error) {
	versions, ok := c[id]
	if !ok {
		return CatalogEntry{}, fmt.Errorf("plugin %s is not in the catalog", id)
	}
	entry, ok := versions[version]
	if !ok {
		return CatalogEntry{}, fmt.Errorf("plugin %s@%s is not in the catalog", id, version)
	}
	return entry, nil
}
