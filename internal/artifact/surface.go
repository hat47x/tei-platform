package artifact

import (
	"encoding/json"
	"fmt"

	"tei/internal/ir"
	"tei/internal/operations"
	"tei/internal/sourceindex"
)

const (
	// OperationSurfaceRuleID and OperationSurfaceRuleVersion identify the rules
	// that turn the derived operations into an artifact.
	OperationSurfaceRuleID      = "tei.artifact.operation-surface"
	OperationSurfaceRuleVersion = "v1"

	// OperationSurfaceGenerator identifies the implementation.
	OperationSurfaceGenerator = "tei-artifact-surface/v1"
)

const operationSurfaceRulePolicy = `operation-surface/v1
content: the derived operations, gaps and summary as deterministic JSON
unresolved: one entry per gap (target, reason, kind)
order: operations and gaps as the derivation orders them`

type operationSurfacePayload struct {
	SchemaVersion string                    `json:"schemaVersion"`
	Application   string                    `json:"application,omitempty"`
	Rule          operations.SurfaceRule    `json:"rule"`
	Operations    []operations.Operation    `json:"operations"`
	Gaps          []operations.SurfaceGap   `json:"gaps"`
	Summary       operations.SurfaceSummary `json:"summary"`
}

// RenderOperationSurface binds the derived standard operations to the artifact
// contract, so "the same meaning produces the same surface" can be checked with
// a digest instead of by reading the JSON. Gaps become unresolved entries: an
// operation that could not be derived is not silently absent.
func RenderOperationSurface(world ir.World, applicationID, inputRevision string) (Artifact, error) {
	surface := operations.DeriveOperationSurface(world, applicationID)
	encoded, err := json.MarshalIndent(operationSurfacePayload{
		SchemaVersion: surface.SchemaVersion,
		Application:   surface.Application,
		Rule:          surface.Rule,
		Operations:    surface.Operations,
		Gaps:          surface.Gaps,
		Summary:       surface.Summary,
	}, "", "  ")
	if err != nil {
		return Artifact{}, fmt.Errorf("encode operation surface: %w", err)
	}

	unresolved := make([]Unresolved, 0, len(surface.Gaps))
	for _, gap := range surface.Gaps {
		unresolved = append(unresolved, Unresolved{
			Subject: gap.Target,
			Reason:  gap.Reason,
			Detail:  fmt.Sprintf("%s: %s", gap.Kind, gap.Detail),
		})
	}

	return New(
		KindOperationSurface,
		Rule{
			ID:          OperationSurfaceRuleID,
			Version:     OperationSurfaceRuleVersion,
			Fingerprint: sourceindex.Fingerprint([]byte(operationSurfaceRulePolicy)),
		},
		OperationSurfaceGenerator,
		inputRevision,
		string(encoded)+"\n",
		unresolved,
	)
}

// decodeOperationSurface reads the derived surface back from the artifact. The
// artifact content is the contract, so consistency checks read the same bytes a
// consumer would.
func decodeOperationSurface(content string) (operationSurfacePayload, error) {
	var payload operationSurfacePayload
	if err := json.Unmarshal([]byte(content), &payload); err != nil {
		return operationSurfacePayload{}, fmt.Errorf("decode operation surface: %w", err)
	}
	return payload, nil
}
