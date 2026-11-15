package workspace

import (
	"fmt"
	"sort"

	"tei/internal/plugin"
	"tei/internal/plugin/implementation"
	"tei/internal/sourceindex"
	"tei/internal/syntax"
)

func loadPluginImplementations(root string, r *Result) error {
	files, err := listPinnedDomainSources(root, r, "plugin-implementations", func(name string) bool {
		return name == "implementation.tei"
	})
	if err != nil {
		return err
	}

	for _, path := range files {
		src, err := readSourceForLoad(root, r, path)
		if err != nil {
			return err
		}
		definition, err := syntax.NewParser(string(src)).ParsePluginImplementation()
		if err != nil {
			if err := r.collectSourceError(path, err); err != nil {
				return err
			}
			continue
		}
		key := definition.Identity()
		if previous, exists := r.PluginImplementationSources[key]; exists {
			return fmt.Errorf("%s: plugin implementation %s is already defined in %s", path, definition.Identity(), previous.File)
		}
		r.PluginImplementations = append(r.PluginImplementations, definition)
		relative := relPath(root, path)
		r.PluginImplementationSources[key] = SourceLocation{File: relative, Line: 1, Column: 1}
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
		definition, ok := pluginDefinitionForImplementation(r.World.Plugins, source)
		if !ok {
			return fmt.Errorf("plugin implementation %s has no matching plugin definition", source.Identity())
		}
		if definition.Artifact.Digest != "" || definition.Artifact.SignatureRef != "" || definition.Artifact.Provenance != "" {
			return fmt.Errorf(
				"plugin implementation %s requires artifact-free authoritative Plugin definition",
				source.Identity(),
			)
		}
		if err := implementation.ValidateAgainstDefinition(source, definition); err != nil {
			return fmt.Errorf("plugin implementation %s: %w", source.Identity(), err)
		}
	}
	return nil
}

func pluginDefinitionForImplementation(definitions []plugin.Definition, source implementation.Source) (plugin.Definition, bool) {
	for _, definition := range definitions {
		if definition.ID == source.Plugin.ID && definition.Version == source.Plugin.Version {
			return definition, true
		}
	}
	return plugin.Definition{}, false
}
