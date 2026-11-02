package artifact

import (
	"fmt"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/database/persistence"
	"tei/internal/semantic"
	"tei/internal/sourceindex"
	"tei/internal/workspace"
)

// WorkspaceSet is every artifact one application implies, plus their manifest.
type WorkspaceSet struct {
	InputRevision string
	Artifacts     []Artifact
	Manifest      Manifest
	// Findings is the result of CheckSet on this set. A non-empty list means
	// the artifacts disagree with each other; the set is returned so a caller
	// can show what was produced, but it must not be read as a consistent set.
	Findings []Finding
}

// RenderWorkspace renders the table definitions and the function definition of
// one application from its workspace. The input revision is the workspace's
// model revision, so every artifact in the set is bound to the same meaning.
func RenderWorkspace(root, applicationID string) (WorkspaceSet, error) {
	loaded, err := workspace.Load(root)
	if err != nil {
		return WorkspaceSet{}, fmt.Errorf("load workspace: %w", err)
	}
	index, err := workspace.BuildSourceIndex(root, loaded)
	if err != nil {
		return WorkspaceSet{}, fmt.Errorf("build source index: %w", err)
	}
	revision := sourceindex.ModelRevision(index)

	target, ok := findApplication(loaded.World.Applications, applicationID)
	if !ok {
		return WorkspaceSet{}, fmt.Errorf("application %q is not declared in this workspace", applicationID)
	}

	data := make(map[core.CanonicalID]semantic.Data)
	capabilities := make(map[core.CanonicalID]semantic.Capability)
	validations := make(map[core.CanonicalID]semantic.Validation)
	for _, namespace := range loaded.World.Semantics {
		for _, element := range namespace.Data {
			data[element.ID] = element
		}
		for _, capability := range namespace.Capabilities {
			capabilities[capability.ID] = capability
		}
		for _, validation := range namespace.Validations {
			validations[validation.ID] = validation
		}
	}

	mappingByID := map[core.CanonicalID]persistence.Mapping{}
	for _, candidate := range loaded.World.Persistence {
		mappingByID[candidate.ID] = candidate
	}
	var artifacts []Artifact
	for _, mapping := range persistenceForApplication(loaded.World.Persistence, target) {
		element, ok := data[mapping.SemanticData]
		if !ok {
			continue
		}
		table, err := RenderTableDefinitionWith(TableDefinitionInput{
			InputRevision: revision,
			Mapping:       mapping,
			Data:          element,
		}, func(id core.CanonicalID) (string, bool) {
			if referenced, ok := mappingByID[id]; ok {
				return referenced.Table, true
			}
			return "", false
		})
		if err != nil {
			return WorkspaceSet{}, fmt.Errorf("render table definition %s: %w", mapping.ID, err)
		}
		artifacts = append(artifacts, table)
	}

	function, err := RenderFunctionDefinition(FunctionDefinitionInput{
		InputRevision: revision,
		Application:   target,
		Capabilities:  capabilities,
		Validations:   validations,
	})
	if err != nil {
		return WorkspaceSet{}, fmt.Errorf("render function definition %s: %w", target.ID, err)
	}
	artifacts = append(artifacts, function)

	surfaceArtifact, err := RenderOperationSurface(loaded.World, applicationID, revision)
	if err != nil {
		return WorkspaceSet{}, fmt.Errorf("render operation surface: %w", err)
	}
	artifacts = append(artifacts, surfaceArtifact)

	manifest, err := NewManifest(revision, artifacts)
	if err != nil {
		return WorkspaceSet{}, fmt.Errorf("build manifest: %w", err)
	}
	set := WorkspaceSet{InputRevision: revision, Artifacts: artifacts, Manifest: manifest}
	set.Findings = CheckSet(set)
	return set, nil
}

func findApplication(applications []application.Application, id string) (application.Application, bool) {
	for _, candidate := range applications {
		if string(candidate.ID) == id {
			return candidate, true
		}
	}
	return application.Application{}, false
}

// persistenceForApplication returns the mappings the application can reach
// through its database aliases, ordered by mapping id.
func persistenceForApplication(mappings []persistence.Mapping, app application.Application) []persistence.Mapping {
	targets := map[core.CanonicalID]bool{}
	for _, requirement := range app.Databases {
		targets[requirement.TargetDatabase] = true
	}
	var out []persistence.Mapping
	for _, mapping := range mappings {
		if targets[mapping.Database] {
			out = append(out, mapping)
		}
	}
	// Stable order by mapping id so the manifest does not depend on file order.
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j].ID < out[j-1].ID; j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}
