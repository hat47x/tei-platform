// This file adds the isolated-tree build: instead of checking the working tree and
// then compiling it (which leaves a window for the files to change), the planned
// sources are copied into a fresh tree and that tree is compiled. What was checked
// is then exactly what was built.
//
// The copy is not a proof of anything by itself: the module context (go.mod,
// go.sum) is taken from the working tree and recorded, because a package build
// cannot happen without it. That is stated in the recorded boundary rather than
// left for a reader to assume.

package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"tei/internal/buildintent"
	"tei/internal/plugin"
)

// isolatedTree is the result of staging the planned sources into a fresh
// directory and confirming the staged bytes are the planned bytes.
type isolatedTree struct {
	Root string `json:"root"`
	// Members are the planned sources, with the digest read back from the copy.
	Members []stagedMember `json:"members"`
	// ModuleFiles are the module context files taken from the working tree. They
	// are not sources, and the recorded boundary says so.
	ModuleFiles []string `json:"moduleFiles,omitempty"`
	Boundary    string   `json:"boundary"`
}

type stagedMember struct {
	Locator string `json:"locator"`
	Planned string `json:"plannedDigest"`
	Staged  string `json:"stagedDigest"`
	Changed bool   `json:"changed"`
}

const isolatedTreeBoundary = "the planned sources were copied into a fresh tree and the copy was re-hashed before building; the module context is whatever the plan named, and anything taken from the working tree instead is recorded under moduleFiles and is not part of the claim"

// checkModuleContextIsReproducible refuses a module whose build would depend on
// something the staged tree does not carry. A replace directive points at a path
// outside the module, and a vendor directory holds the dependencies beside the
// module rather than in the cache: neither is staged, so a build that used them
// would not be the build the plan describes.
func checkModuleContextIsReproducible(stageRoot string) error {
	goMod, err := os.ReadFile(filepath.Join(stageRoot, "go.mod"))
	if err != nil {
		// A module context that is not there is the caller's problem to see, and the
		// build reports it. This check is about what the file says.
		return nil
	}
	for _, line := range strings.Split(string(goMod), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "replace ") || strings.HasPrefix(trimmed, "replace(") || trimmed == "replace (" {
			return fmt.Errorf("the module context has a replace directive (%q); this build stages only the plan's sources, so it would not reproduce what that directive points at", trimmed)
		}
	}
	if info, statErr := os.Stat(filepath.Join(stageRoot, "vendor")); statErr == nil && info.IsDir() {
		return fmt.Errorf("the staged tree has a vendor directory; this build stages only the plan's sources, so vendored dependencies would be taken from outside the plan")
	}
	return nil
}

// planNamesModuleFile reports whether the plan already names this file as a
// go-module source, so staging does not take a second copy from the working tree.
func planNamesModuleFile(plan plugin.BuildPlan, name string) bool {
	for _, source := range plan.Sources {
		if source.Kind == "go-module" && filepath.ToSlash(source.Locator) == name {
			return true
		}
	}
	return false
}

// stageSources copies each planned source into root, preserving the locator, and
// refuses when the copy does not hash to the planned value.
func stageSources(repoRoot, stageRoot string, plan plugin.BuildPlan, checked []checkedSource) (isolatedTree, error) {
	tree := isolatedTree{Root: stageRoot, Boundary: isolatedTreeBoundary}
	resolver := buildintent.DirSourceResolver{Root: repoRoot}
	for _, source := range plan.Sources {
		resolved, err := resolver.Resolve(buildintent.IntentSource{Kind: source.Kind, ID: source.ID, Ref: source.Locator})
		if err != nil {
			return isolatedTree{}, fmt.Errorf("resolve source %q: %w", source.Locator, err)
		}
		// A source is either one file (resolved under the name ".") or a directory
		// of members. The locator is the file's own path in the first case, so the
		// copy must not turn it into a directory.
		destination := filepath.Join(stageRoot, filepath.FromSlash(source.Locator))
		names := make([]string, 0, len(resolved.Files))
		for name := range resolved.Files {
			names = append(names, name)
		}
		sort.Strings(names)
		for _, name := range names {
			target := destination
			if name != "." {
				target = filepath.Join(destination, filepath.FromSlash(name))
			}
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return isolatedTree{}, err
			}
			if err := os.WriteFile(target, resolved.Files[name], 0o644); err != nil {
				return isolatedTree{}, err
			}
		}
		// Re-read the staged tree through the same resolver and compare digests.
		staged, err := (buildintent.DirSourceResolver{Root: stageRoot}).Resolve(
			buildintent.IntentSource{Kind: source.Kind, ID: source.ID, Ref: source.Locator})
		if err != nil {
			return isolatedTree{}, fmt.Errorf("re-resolve the staged source %q: %w", source.Locator, err)
		}
		member := stagedMember{
			Locator: source.Locator,
			Planned: source.Digest,
			Staged:  staged.Source.Digest,
			Changed: staged.Source.Digest != source.Digest,
		}
		if member.Changed {
			return isolatedTree{}, fmt.Errorf("the staged copy of %q does not hash to the planned digest (%s, planned %s); what was checked is not what would be built", source.Locator, member.Staged, member.Planned)
		}
		tree.Members = append(tree.Members, member)
	}

	// A plan may name the module context itself (kind go-module). When it does,
	// staging already copied those files and nothing is taken from the working
	// tree, so the isolated tree depends on the plan alone.
	for _, name := range []string{"go.mod", "go.sum"} {
		if planNamesModuleFile(plan, name) {
			continue
		}
		source := filepath.Join(repoRoot, name)
		data, err := os.ReadFile(source)
		if err != nil {
			if os.IsNotExist(err) && name == "go.sum" {
				continue
			}
			return isolatedTree{}, fmt.Errorf("read %s for the module context: %w", name, err)
		}
		if err := os.WriteFile(filepath.Join(stageRoot, name), data, 0o644); err != nil {
			return isolatedTree{}, err
		}
		tree.ModuleFiles = append(tree.ModuleFiles, name+":"+digestOfBytes(data))
	}
	sort.Strings(tree.ModuleFiles)
	return tree, nil
}

// digestTree hashes a directory the same way the resolver hashes a source, so a
// caller can compare the staged tree with the plan without a second convention.
func digestTree(root string) (string, error) {
	h := sha256.New()
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		relative, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}
		file, openErr := os.Open(path)
		if openErr != nil {
			return openErr
		}
		defer file.Close()
		data, readErr := io.ReadAll(file)
		if readErr != nil {
			return readErr
		}
		fmt.Fprintf(h, "%d:%s\n%d\n", len(relative), filepath.ToSlash(relative), len(data))
		h.Write(data)
		return nil
	})
	if err != nil {
		return "", err
	}
	return "sha256:" + hex.EncodeToString(h.Sum(nil)), nil
}

var _ = strings.TrimSpace
