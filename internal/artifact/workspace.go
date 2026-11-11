package artifact

import (
	"fmt"

	"strings"
	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/database/persistence"
	"tei/internal/ir"

	"tei/internal/semantic"
	"tei/internal/sourceindex"
	"tei/internal/workspace"
)

// WorkspaceSet is every artifact one application implies, plus their manifest.
type WorkspaceSet struct {
	InputRevision string     `json:"inputRevision"`
	Artifacts     []Artifact `json:"artifacts"`
	Manifest      Manifest   `json:"manifest"`
	// Findings is the result of CheckSet on this set. A non-empty list means
	// the artifacts disagree with each other; the set is returned so a caller
	// can show what was produced, but it must not be read as a consistent set.
	Findings []Finding `json:"findings"`
}

// renderWorkspace renders the artifacts for one application and returns the
// loaded workspace and source index so callers can record provenance.
func renderWorkspace(root, applicationID string) (WorkspaceSet, workspace.Result, sourceindex.Index, error) {
	loaded, err := workspace.Load(root)
	if err != nil {
		return WorkspaceSet{}, workspace.Result{}, sourceindex.Index{}, fmt.Errorf("load workspace: %w", err)
	}
	index, err := workspace.BuildSourceIndex(root, loaded)
	if err != nil {
		return WorkspaceSet{}, workspace.Result{}, sourceindex.Index{}, fmt.Errorf("build source index: %w", err)
	}
	revision := sourceindex.ModelRevision(index)

	target, ok := findApplication(loaded.World.Applications, applicationID)
	if !ok {
		return WorkspaceSet{}, workspace.Result{}, sourceindex.Index{}, fmt.Errorf("application %q is not declared in this workspace", applicationID)
	}

	data := make(map[core.CanonicalID]semantic.Data)
	capabilities := make(map[core.CanonicalID]semantic.Capability)
	validations := make(map[core.CanonicalID]semantic.Validation)
	periods := make(map[core.CanonicalID]semantic.Period)
	effectRules := make([]semantic.EffectRule, 0)
	runPlans := make([]semantic.RunPlan, 0)
	used := map[core.CanonicalID]bool{}
	for _, namespace := range target.SemanticUses {
		used[namespace] = true
	}
	for _, namespace := range loaded.World.Semantics {
		for _, element := range namespace.Data {
			data[element.ID] = element
		}
		// Only the namespaces the application uses contribute functions: an
		// application must not appear to provide a capability it does not use.
		if used[namespace.ID] {
			for _, capability := range namespace.Capabilities {
				capabilities[capability.ID] = capability
			}
			effectRules = append(effectRules, namespace.EffectRules...)
			runPlans = append(runPlans, namespace.RunPlans...)
			for _, period := range namespace.Periods {
				periods[period.ID] = period
			}
			for _, validation := range namespace.Validations {
				validations[validation.ID] = validation
			}
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
			return WorkspaceSet{}, workspace.Result{}, sourceindex.Index{}, fmt.Errorf("render table definition %s: %w", mapping.ID, err)
		}
		artifacts = append(artifacts, table)
	}

	function, err := RenderFunctionDefinition(FunctionDefinitionInput{
		InputRevision: revision,
		Application:   target,
		Capabilities:  capabilities,
		Validations:   validations,
		Periods:       periods,
		EffectRules:   effectRules,
		RunPlans:      runPlans,
	})
	if err != nil {
		return WorkspaceSet{}, workspace.Result{}, sourceindex.Index{}, fmt.Errorf("render function definition %s: %w", target.ID, err)
	}
	artifacts = append(artifacts, function)

	surfaceArtifact, err := RenderOperationSurface(loaded.World, applicationID, revision)
	if err != nil {
		return WorkspaceSet{}, workspace.Result{}, sourceindex.Index{}, fmt.Errorf("render operation surface: %w", err)
	}
	artifacts = append(artifacts, surfaceArtifact)

	manifest, err := NewManifest(revision, artifacts)
	if err != nil {
		return WorkspaceSet{}, workspace.Result{}, sourceindex.Index{}, fmt.Errorf("build manifest: %w", err)
	}
	set := WorkspaceSet{InputRevision: revision, Artifacts: artifacts, Manifest: manifest}
	set.Findings = append(CheckSet(set), checkSurfaceTargets(loaded.World, target, artifacts)...)
	set.Findings = append(set.Findings, checkPeriodPreconditions(loaded.World, artifacts)...)
	if set.Findings == nil {
		set.Findings = []Finding{}
	}
	return set, loaded, index, nil
}

// RenderWorkspace renders the table definitions and the function definition of
// one application from its workspace. The input revision is the workspace's
// model revision, so every artifact in the set is bound to the same meaning.
func RenderWorkspace(root, applicationID string) (WorkspaceSet, error) {
	set, _, _, err := renderWorkspace(root, applicationID)
	return set, err
}

// RenderWorkspaceWithIndex renders the artifacts and returns the loaded workspace
// and source index so callers can record provenance.
func RenderWorkspaceWithIndex(root, applicationID string) (WorkspaceSet, workspace.Result, sourceindex.Index, error) {
	return renderWorkspace(root, applicationID)
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

// checkSurfaceTargets verifies that every operation names a data the
// application can reach through the mappings it may use. The derivation builds
// the surface from those mappings, so a mismatch means the surface and the
// persistence disagree.
func checkSurfaceTargets(world ir.World, app application.Application, artifacts []Artifact) []Finding {
	reachable := map[core.CanonicalID]bool{}
	for _, mapping := range persistenceForApplication(world.Persistence, app) {
		reachable[mapping.SemanticData] = true
	}
	var findings []Finding
	for _, artifact := range artifacts {
		if artifact.Kind != string(KindOperationSurface) {
			continue
		}
		payload, err := decodeOperationSurface(artifact.Content)
		if err != nil {
			continue
		}
		for _, operation := range payload.Operations {
			if !reachable[core.CanonicalID(operation.Target)] {
				findings = append(findings, Finding{
					Code:    FindingSurfaceTargetUnreachable,
					Subject: operation.ID,
					Detail:  "the application can not reach " + operation.Target,
				})
			}
		}
	}
	return findings
}

// checkPeriodPreconditions verifies that every "period:<id>:open" precondition
// names a period that exists for the operation target and that the period
// actually rejects that operation once closed. A precondition without a basis
// would read as a requirement the surface invented.
func checkPeriodPreconditions(world ir.World, artifacts []Artifact) []Finding {
	periods := map[string]semantic.Period{}
	for _, namespace := range world.Semantics {
		for _, period := range namespace.Periods {
			periods[string(period.ID)] = period
		}
	}
	var findings []Finding
	for _, artifact := range artifacts {
		if artifact.Kind != string(KindOperationSurface) {
			continue
		}
		payload, err := decodeOperationSurface(artifact.Content)
		if err != nil {
			continue
		}
		for _, operation := range payload.Operations {
			for _, precondition := range operation.Preconditions {
				if !strings.HasPrefix(precondition, "period:") || !strings.HasSuffix(precondition, ":open") {
					continue
				}
				id := strings.TrimSuffix(strings.TrimPrefix(precondition, "period:"), ":open")
				period, ok := periods[id]
				if !ok {
					findings = append(findings, Finding{
						Code:    "PERIOD_PRECONDITION_UNKNOWN",
						Subject: operation.ID,
						Detail:  "the precondition names an unknown period " + id,
					})
					continue
				}
				if string(period.Data) != operation.Target {
					findings = append(findings, Finding{
						Code:    "PERIOD_PRECONDITION_MISMATCH",
						Subject: operation.ID,
						Detail:  "the period " + id + " governs " + string(period.Data) + ", not " + operation.Target,
					})
				}
			}
		}
	}
	return findings
}
