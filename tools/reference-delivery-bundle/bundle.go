// Package deliverybundle packages a resolved Delivery Profile and its plugin
// descriptors into one zip. It does not sign the bundle or install it.
package deliverybundle

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	deliveryprofile "tei/tools/reference-delivery-profile"
	exportpackage "tei/tools/reference-export-package"
)

const Schema = "tei.reference.plugin-bundle/v0"

type Plugin struct {
	ID      string `json:"id"`
	Version string `json:"version"`
	Digest  string `json:"digest"`
	File    string `json:"file"`
}

type Manifest struct {
	Schema         string   `json:"schema"`
	ProfileID      string   `json:"profileId"`
	ProfileVersion string   `json:"profileVersion"`
	CatalogDigest  string   `json:"catalogDigest"`
	LockDigest     string   `json:"lockDigest"`
	Plugins        []Plugin `json:"plugins"`
}

type Receipt struct {
	Manifest      Manifest `json:"manifest"`
	PackageDigest string   `json:"packageDigest"`
	Passed        bool     `json:"passed"`
}

func digestBytes(raw []byte) string {
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func catalogDigest(plugins []Plugin) string {
	sorted := append([]Plugin(nil), plugins...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].ID < sorted[j].ID })
	hash := sha256.New()
	for _, plugin := range sorted {
		fmt.Fprintf(hash, "%s@%s\x00%s\n", plugin.ID, plugin.Version, plugin.Digest)
	}
	return "sha256:" + hex.EncodeToString(hash.Sum(nil))
}

func Pack(profilePath, catalogDir, out string) (Receipt, error) {
	composition, err := deliveryprofile.ValidateWithCatalog(profilePath, catalogDir)
	if err != nil {
		return Receipt{}, err
	}
	if !composition.Passed {
		return Receipt{}, fmt.Errorf("profile did not pass validation; bundle not written")
	}
	lock := deliveryprofile.BuildLock(composition)
	staging, err := os.MkdirTemp("", "tei-plugin-bundle-*")
	if err != nil {
		return Receipt{}, err
	}
	defer os.RemoveAll(staging)
	plugins := make([]Plugin, 0, len(composition.Plugins))
	for _, source := range composition.Plugins {
		raw, err := os.ReadFile(source.Path)
		if err != nil {
			return Receipt{}, err
		}
		file := source.ID + "-" + source.Version + ".json"
		if err := os.WriteFile(filepath.Join(staging, file), raw, 0o644); err != nil {
			return Receipt{}, err
		}
		plugins = append(plugins, Plugin{ID: source.ID, Version: source.Version, Digest: source.Digest, File: file})
	}
	manifest := Manifest{
		Schema: Schema, ProfileID: composition.ProfileID, ProfileVersion: composition.ProfileVersion,
		CatalogDigest: catalogDigest(plugins), LockDigest: lock.Digest, Plugins: plugins,
	}
	raw, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return Receipt{}, err
	}
	if err := os.WriteFile(filepath.Join(staging, "bundle.json"), append(raw, '\n'), 0o644); err != nil {
		return Receipt{}, err
	}
	packageManifest, err := exportpackage.Export(staging, out)
	if err != nil {
		return Receipt{}, err
	}
	return Receipt{Manifest: manifest, PackageDigest: packageManifest.Digest, Passed: true}, nil
}

func Verify(path string) (Receipt, error) {
	packageManifest, err := exportpackage.Verify(path)
	if err != nil {
		return Receipt{}, err
	}
	reader, err := zip.OpenReader(path)
	if err != nil {
		return Receipt{}, err
	}
	defer reader.Close()
	entries := map[string]*zip.File{}
	for _, entry := range reader.File {
		entries[entry.Name] = entry
	}
	bundleFile, ok := entries["files/bundle.json"]
	if !ok {
		return Receipt{}, fmt.Errorf("bundle.json is missing")
	}
	bundleReader, err := bundleFile.Open()
	if err != nil {
		return Receipt{}, err
	}
	raw, err := io.ReadAll(bundleReader)
	bundleReader.Close()
	if err != nil {
		return Receipt{}, err
	}
	var manifest Manifest
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return Receipt{}, fmt.Errorf("bundle.json is not valid JSON: %w", err)
	}
	if manifest.Schema != Schema {
		return Receipt{}, fmt.Errorf("bundle schema %q is not %q", manifest.Schema, Schema)
	}
	if manifest.CatalogDigest != catalogDigest(manifest.Plugins) {
		return Receipt{}, fmt.Errorf("catalog digest does not match the plugin list")
	}
	seen := map[string]bool{}
	for _, plugin := range manifest.Plugins {
		if seen[plugin.ID] {
			return Receipt{}, fmt.Errorf("duplicate plugin id %s", plugin.ID)
		}
		seen[plugin.ID] = true
		entry, ok := entries["files/"+plugin.File]
		if !ok {
			return Receipt{}, fmt.Errorf("plugin file %s is missing", plugin.File)
		}
		pluginReader, err := entry.Open()
		if err != nil {
			return Receipt{}, err
		}
		content, err := io.ReadAll(pluginReader)
		pluginReader.Close()
		if err != nil {
			return Receipt{}, err
		}
		if digestBytes(content) != plugin.Digest {
			return Receipt{}, fmt.Errorf("plugin %s digest does not match", plugin.ID)
		}
	}
	return Receipt{Manifest: manifest, PackageDigest: packageManifest.Digest, Passed: true}, nil
}
