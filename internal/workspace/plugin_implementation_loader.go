package workspace

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"tei/internal/plugin"
	"tei/internal/plugin/implementation"
	"tei/internal/sourceindex"
	"tei/internal/syntax"
)

func loadPluginImplementations(root string, r *Result) error {
	base := filepath.Join(root, "implementations")
	var files []string
	err := filepath.WalkDir(base, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			if os.IsNotExist(walkErr) {
				return nil
			}
			return walkErr
		}
		if d.IsDir() || d.Name() != "implementation.tei" {
			return nil
		}
		files = append(files, path)
		return nil
	})
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	sort.Strings(files)

	for _, path := range files {
		src, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		definition, err := syntax.NewParser(string(src)).ParsePluginImplementation()
		if err != nil {
			return sourceErr(path, err)
		}
		key := "plugin-implementation:" + definition.Identity()
		if previous, exists := r.Sources[key]; exists {
			return fmt.Errorf("%s: plugin implementation %s is already defined in %s", path, definition.Identity(), previous.File)
		}
		r.PluginImplementations = append(r.PluginImplementations, definition)
		relative := relPath(root, path)
		r.Sources[key] = SourceLocation{File: relative, Line: 1, Column: 1}
		r.SourceContributions = append(r.SourceContributions, SourceContribution{
			Subject: sourceindex.Subject{Kind: "plugin-implementation", ID: definition.Identity()},
			File:    relative,
			Role:    sourceindex.RoleImplementation,
		})
	}
	sort.Slice(r.PluginImplementations, func(i, j int) bool {
		return r.PluginImplementations[i].Identity() < r.PluginImplementations[j].Identity()
	})
	return nil
}

func validatePluginImplementationDefinitions(r *Result) error {
	for _, source := range r.PluginImplementations {
		definition, ok := pluginDefinition(r.World.Plugins, source)
		if !ok {
			return fmt.Errorf("plugin implementation %s has no matching plugin definition", source.Identity())
		}
		if err := implementation.ValidateAgainstDefinition(source, definition); err != nil {
			return fmt.Errorf("plugin implementation %s: %w", source.Identity(), err)
		}
	}
	return nil
}

func pluginDefinition(definitions []plugin.Definition, source implementation.Source) (plugin.Definition, bool) {
	for _, definition := range definitions {
		if definition.ID == source.Plugin.ID && definition.Version == source.Plugin.Version {
			return definition, true
		}
	}
	return plugin.Definition{}, false
}
