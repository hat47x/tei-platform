package deliverybundle

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	deliveryprofile "tei/tools/reference-delivery-profile"
)

type InstallReceipt struct {
	Bundle    Manifest `json:"bundle"`
	Catalog   string   `json:"catalog"`
	Installed []string `json:"installed"`
	Skipped   []string `json:"skipped"`
}

// Install verifies the bundle signature and package, then writes the selected
// plugin descriptors into catalogDir. An existing file is skipped only when its
// digest is identical; a different digest fails closed. Installation is not
// transactional.
func Install(bundlePath, signaturePath, publicKeyPath, catalogDir string) (InstallReceipt, error) {
	if err := VerifySignature(bundlePath, signaturePath, publicKeyPath); err != nil {
		return InstallReceipt{}, fmt.Errorf("verify bundle signature: %w", err)
	}
	receipt, err := Verify(bundlePath)
	if err != nil {
		return InstallReceipt{}, err
	}
	reader, err := zip.OpenReader(bundlePath)
	if err != nil {
		return InstallReceipt{}, err
	}
	defer reader.Close()
	entries := map[string]*zip.File{}
	for _, entry := range reader.File {
		entries[entry.Name] = entry
	}
	if err := os.MkdirAll(catalogDir, 0o755); err != nil {
		return InstallReceipt{}, err
	}
	root, err := os.OpenRoot(catalogDir)
	if err != nil {
		return InstallReceipt{}, err
	}
	defer root.Close()
	result := InstallReceipt{Bundle: receipt.Manifest, Catalog: catalogDir, Installed: []string{}, Skipped: []string{}}
	for _, plugin := range receipt.Manifest.Plugins {
		file := plugin.File
		if file == "" || filepath.Base(file) != file || strings.Contains(file, "..") {
			return InstallReceipt{}, fmt.Errorf("plugin %s has an unsafe file name %q", plugin.ID, file)
		}
		entry, ok := entries["files/"+file]
		if !ok {
			return InstallReceipt{}, fmt.Errorf("plugin %s file %s is missing from the bundle", plugin.ID, file)
		}
		pluginReader, err := entry.Open()
		if err != nil {
			return InstallReceipt{}, err
		}
		content, err := io.ReadAll(pluginReader)
		pluginReader.Close()
		if err != nil {
			return InstallReceipt{}, err
		}
		if digestBytes(content) != plugin.Digest {
			return InstallReceipt{}, fmt.Errorf("plugin %s digest does not match", plugin.ID)
		}
		info, statErr := root.Lstat(file)
		if statErr == nil {
			if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
				return InstallReceipt{}, fmt.Errorf("catalog entry %s is not a regular file", file)
			}
			existing, readErr := root.ReadFile(file)
			if readErr != nil {
				return InstallReceipt{}, readErr
			}
			if digestBytes(existing) == plugin.Digest {
				result.Skipped = append(result.Skipped, file)
				continue
			}
			return InstallReceipt{}, fmt.Errorf("catalog entry %s already exists with a different digest", file)
		}
		if !os.IsNotExist(statErr) {
			return InstallReceipt{}, statErr
		}
		installed, err := root.OpenFile(file, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
		if err != nil {
			return InstallReceipt{}, err
		}
		if _, err := installed.Write(content); err != nil {
			installed.Close()
			return InstallReceipt{}, err
		}
		if err := installed.Sync(); err != nil {
			installed.Close()
			return InstallReceipt{}, err
		}
		if err := installed.Close(); err != nil {
			return InstallReceipt{}, err
		}
		result.Installed = append(result.Installed, file)
	}
	if _, err := deliveryprofile.LoadCatalog(catalogDir); err != nil {
		return InstallReceipt{}, fmt.Errorf("installed catalog is not valid: %w", err)
	}
	return result, nil
}

func extractDescriptors(bundlePath string, manifest Manifest, targetDir string) ([]string, error) {
	reader, err := zip.OpenReader(bundlePath)
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	entries := map[string]*zip.File{}
	for _, entry := range reader.File {
		entries[entry.Name] = entry
	}
	installed := []string{}
	for _, plugin := range manifest.Plugins {
		file := plugin.File
		if file == "" || filepath.Base(file) != file || strings.Contains(file, "..") {
			return nil, fmt.Errorf("plugin %s has an unsafe file name %q", plugin.ID, file)
		}
		entry, ok := entries["files/"+file]
		if !ok {
			return nil, fmt.Errorf("plugin %s file %s is missing from the bundle", plugin.ID, file)
		}
		pluginReader, err := entry.Open()
		if err != nil {
			return nil, err
		}
		content, err := io.ReadAll(pluginReader)
		pluginReader.Close()
		if err != nil {
			return nil, err
		}
		if digestBytes(content) != plugin.Digest {
			return nil, fmt.Errorf("plugin %s digest does not match", plugin.ID)
		}
		path := filepath.Join(targetDir, file)
		out, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
		if err != nil {
			return nil, err
		}
		if _, err := out.Write(content); err != nil {
			out.Close()
			return nil, err
		}
		if err := out.Sync(); err != nil {
			out.Close()
			return nil, err
		}
		if err := out.Close(); err != nil {
			return nil, err
		}
		installed = append(installed, file)
	}
	return installed, nil
}

// InstallFresh builds a complete new catalog in a staging directory and
// publishes it with one Rename. The target directory must not exist. A failed
// install leaves no target directory.
func InstallFresh(bundlePath, signaturePath, publicKeyPath, catalogDir string) (InstallReceipt, error) {
	if err := VerifySignature(bundlePath, signaturePath, publicKeyPath); err != nil {
		return InstallReceipt{}, fmt.Errorf("verify bundle signature: %w", err)
	}
	receipt, err := Verify(bundlePath)
	if err != nil {
		return InstallReceipt{}, err
	}
	absolute, err := filepath.Abs(catalogDir)
	if err != nil {
		return InstallReceipt{}, err
	}
	if _, err := os.Lstat(absolute); err == nil {
		return InstallReceipt{}, fmt.Errorf("catalog directory already exists: %s", absolute)
	} else if !os.IsNotExist(err) {
		return InstallReceipt{}, err
	}
	parent := filepath.Dir(absolute)
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return InstallReceipt{}, err
	}
	staging, err := os.MkdirTemp(parent, ".tei-bundle-install-*")
	if err != nil {
		return InstallReceipt{}, err
	}
	defer os.RemoveAll(staging)
	installed, err := extractDescriptors(bundlePath, receipt.Manifest, staging)
	if err != nil {
		return InstallReceipt{}, err
	}
	if _, err := deliveryprofile.LoadCatalog(staging); err != nil {
		return InstallReceipt{}, fmt.Errorf("staged catalog is not valid: %w", err)
	}
	if err := os.Rename(staging, absolute); err != nil {
		return InstallReceipt{}, err
	}
	return InstallReceipt{
		Bundle: receipt.Manifest, Catalog: absolute, Installed: installed, Skipped: []string{},
	}, nil
}
