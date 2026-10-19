// Package exportpackage builds a deterministic delivery package: a zip with a
// manifest and the selected files. It is a Reference foundation, not a signed
// release artifact.
package exportpackage

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
	"strings"
)

const Schema = "tei.reference.export-package/v0"

type File struct {
	Path   string `json:"path"`
	Size   int64  `json:"size"`
	Digest string `json:"digest"`
}

type Manifest struct {
	Schema string `json:"schema"`
	Files  []File `json:"files"`
	Digest string `json:"digest"`
}

func digestBytes(raw []byte) string {
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func manifestDigest(files []File) string {
	hash := sha256.New()
	for _, file := range files {
		fmt.Fprintf(hash, "%s\x00%d\x00%s\n", file.Path, file.Size, file.Digest)
	}
	return "sha256:" + hex.EncodeToString(hash.Sum(nil))
}

func Export(root, out string) (Manifest, error) {
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return Manifest{}, err
	}
	outAbs, err := filepath.Abs(out)
	if err != nil {
		return Manifest{}, err
	}
	info, err := os.Stat(rootAbs)
	if err != nil || !info.IsDir() {
		return Manifest{}, fmt.Errorf("package root is not a directory: %s", rootAbs)
	}
	relative, err := filepath.Rel(rootAbs, outAbs)
	if err == nil && !strings.HasPrefix(relative, "..") && relative != "." {
		return Manifest{}, fmt.Errorf("package output must be outside the package root")
	}
	var files []File
	if err := filepath.WalkDir(rootAbs, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("package contains a symbolic link: %s", path)
		}
		relative, err := filepath.Rel(rootAbs, path)
		if err != nil {
			return err
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		files = append(files, File{
			Path: filepath.ToSlash(relative), Size: int64(len(content)), Digest: digestBytes(content),
		})
		return nil
	}); err != nil {
		return Manifest{}, err
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	manifest := Manifest{Schema: Schema, Files: files}
	manifest.Digest = manifestDigest(files)
	manifestRaw, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return Manifest{}, err
	}
	if err := os.MkdirAll(filepath.Dir(outAbs), 0o755); err != nil {
		return Manifest{}, err
	}
	file, err := os.Create(outAbs)
	if err != nil {
		return Manifest{}, err
	}
	writer := zip.NewWriter(file)
	closeAll := func() error {
		if err := writer.Close(); err != nil {
			file.Close()
			return err
		}
		return file.Close()
	}
	header := &zip.FileHeader{Name: "manifest.json", Method: zip.Deflate}
	header.SetMode(0o644)
	entry, err := writer.CreateHeader(header)
	if err != nil {
		file.Close()
		return Manifest{}, err
	}
	if _, err := entry.Write(append(manifestRaw, '\n')); err != nil {
		file.Close()
		return Manifest{}, err
	}
	for _, item := range files {
		content, err := os.ReadFile(filepath.Join(rootAbs, filepath.FromSlash(item.Path)))
		if err != nil {
			file.Close()
			return Manifest{}, err
		}
		header := &zip.FileHeader{Name: "files/" + item.Path, Method: zip.Deflate}
		header.SetMode(0o644)
		entry, err := writer.CreateHeader(header)
		if err != nil {
			file.Close()
			return Manifest{}, err
		}
		if _, err := entry.Write(content); err != nil {
			file.Close()
			return Manifest{}, err
		}
	}
	if err := closeAll(); err != nil {
		return Manifest{}, err
	}
	return manifest, nil
}

func Verify(path string) (Manifest, error) {
	reader, err := zip.OpenReader(path)
	if err != nil {
		return Manifest{}, err
	}
	defer reader.Close()
	entries := map[string]*zip.File{}
	for _, entry := range reader.File {
		name := entry.Name
		if strings.HasPrefix(name, "/") || strings.Contains(name, "..") {
			return Manifest{}, fmt.Errorf("package entry %q is not a safe relative path", name)
		}
		entries[name] = entry
	}
	manifestFile, ok := entries["manifest.json"]
	if !ok {
		return Manifest{}, fmt.Errorf("package has no manifest.json")
	}
	manifestReader, err := manifestFile.Open()
	if err != nil {
		return Manifest{}, err
	}
	manifestRaw, err := io.ReadAll(manifestReader)
	manifestReader.Close()
	if err != nil {
		return Manifest{}, err
	}
	var manifest Manifest
	if err := json.Unmarshal(manifestRaw, &manifest); err != nil {
		return Manifest{}, fmt.Errorf("manifest is not valid JSON: %w", err)
	}
	if manifest.Schema != Schema {
		return Manifest{}, fmt.Errorf("manifest schema %q is not %q", manifest.Schema, Schema)
	}
	if manifest.Digest != manifestDigest(manifest.Files) {
		return Manifest{}, fmt.Errorf("manifest digest does not match its file list")
	}
	if len(manifest.Files) != len(entries)-1 {
		return Manifest{}, fmt.Errorf("package has %d entries, manifest lists %d files", len(entries)-1, len(manifest.Files))
	}
	for _, item := range manifest.Files {
		entry, ok := entries["files/"+item.Path]
		if !ok {
			return Manifest{}, fmt.Errorf("package is missing %s", item.Path)
		}
		contentReader, err := entry.Open()
		if err != nil {
			return Manifest{}, err
		}
		content, err := io.ReadAll(contentReader)
		contentReader.Close()
		if err != nil {
			return Manifest{}, err
		}
		if int64(len(content)) != item.Size || digestBytes(content) != item.Digest {
			return Manifest{}, fmt.Errorf("package file %s does not match the manifest", item.Path)
		}
	}
	return manifest, nil
}
