package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"

	"tei/internal/sourceindex"
	"tei/internal/validation"
	"tei/internal/workspace"
)

// snapshotは、ある時点のauthoritative sourceと、そこから導出したCanonical World / Source Indexを束ねる。
type snapshot struct {
	root     string
	loaded   workspace.Result
	index    sourceindex.Index
	revision string
}

func takeSnapshot(root string) (snapshot, error) {
	loaded, err := workspace.Load(root)
	if err != nil {
		return snapshot{}, fmt.Errorf("workspaceを読み込めません: %w", err)
	}
	index, err := workspace.BuildSourceIndex(root, loaded)
	if err != nil {
		return snapshot{}, fmt.Errorf("Source Indexを構築できません: %w", err)
	}
	return snapshot{root: root, loaded: loaded, index: index, revision: revisionOf(index)}, nil
}

// revisionOfは、管理対象source全fileの（相対path, 内容fingerprint）からmodel revisionを導く。
// file pathの変更もsourceの変更として現れる。意味が変わらないfile移動を意味変更と区別するのはSemantic Diffの責務である。
func revisionOf(index sourceindex.Index) string {
	artifacts := append([]sourceindex.Artifact(nil), index.Artifacts...)
	sort.Slice(artifacts, func(i, j int) bool { return artifacts[i].Path < artifacts[j].Path })
	hash := sha256.New()
	for _, artifact := range artifacts {
		fmt.Fprintf(hash, "%s\x00%s\n", artifact.Path, artifact.Fingerprint)
	}
	return "sha256:" + hex.EncodeToString(hash.Sum(nil))
}

func (s snapshot) fingerprints() map[string]string {
	out := make(map[string]string, len(s.index.Artifacts))
	for _, artifact := range s.index.Artifacts {
		out[artifact.Path] = artifact.Fingerprint
	}
	return out
}

func (s snapshot) diagnostics() []validation.Diagnostic {
	return validation.Validate(s.loaded.World)
}

func copyTree(src, dst string) error {
	return filepath.WalkDir(src, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if entry.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		if !entry.Type().IsRegular() {
			return fmt.Errorf("通常fileではないため複製できません: %s", path)
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, content, 0o644)
	})
}

// writeFileAtomicは同じdirectory内のtemp fileを経由して置き換え、途中状態のsourceを残さない。
func writeFileAtomic(path string, content []byte) error {
	dir := filepath.Dir(path)
	temp, err := os.CreateTemp(dir, ".tei-accept-*")
	if err != nil {
		return err
	}
	tempPath := temp.Name()
	cleanup := func() { _ = os.Remove(tempPath) }
	if _, err := temp.Write(content); err != nil {
		_ = temp.Close()
		cleanup()
		return err
	}
	if err := temp.Close(); err != nil {
		cleanup()
		return err
	}
	if info, err := os.Stat(path); err == nil {
		if err := os.Chmod(tempPath, info.Mode().Perm()); err != nil {
			cleanup()
			return err
		}
	}
	if err := os.Rename(tempPath, path); err != nil {
		cleanup()
		return err
	}
	return nil
}
