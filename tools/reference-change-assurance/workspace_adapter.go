package main

import (
	"fmt"

	"tei/internal/index"
	"tei/internal/modelref"
	"tei/internal/modelref/derivedbridge"
	"tei/internal/workspace"
)

// WorkspaceChangeReport inspects a proposed typed change in the real Workspace
// Derived Index. It never certifies a closed dependency graph or an old receipt.
type WorkspaceChangeReport struct {
	SchemaVersion        string                `json:"schemaVersion"`
	Authority            string                `json:"authority"`
	Changed              modelref.Change       `json:"changed"`
	Consumer             modelref.Key          `json:"consumer"`
	GraphNodes           int                   `json:"graphNodes"`
	GraphEdges           int                   `json:"graphEdges"`
	ProjectionUnresolved []string              `json:"projectionUnresolved"`
	Impact               modelref.ImpactReview `json:"impact"`
	Decision             Decision              `json:"decision"`
	AuthorityLimits      []string              `json:"authorityLimits"`
}

// reviewWorkspaceChange uses TEI's existing read-only projection and typed
// impact review. changeKind is a proposed scenario, not an observed Source diff.
// No semantic acceptance, Provider result, independent oracle, or runtime
// authority is inferred from the index.
func reviewWorkspaceChange(root, scope string, changed modelref.Key, changeKind string, consumer modelref.Key) (WorkspaceChangeReport, error) {
	if root == "" || scope == "" || changed.Scope != scope || consumer.Scope != scope {
		return WorkspaceChangeReport{}, fmt.Errorf("workspace, scope and matching typed key scopes are required")
	}
	loaded, err := workspace.Load(root)
	if err != nil {
		return WorkspaceChangeReport{}, fmt.Errorf("load workspace: %w", err)
	}
	projection := derivedbridge.FromIndex(index.Build(loaded.World), scope)
	proposed := modelref.Change{Key: changed, Kind: changeKind}
	impact := modelref.ReviewImpact(projection.Graph, []modelref.Change{proposed}, projection.Rules)
	result := WorkspaceChangeReport{
		SchemaVersion:        "tei.reference.workspace-change-assurance/v0",
		Authority:            "derived-index-candidate-no-execution",
		Changed:              proposed,
		Consumer:             consumer,
		GraphNodes:           len(projection.Graph.Nodes),
		GraphEdges:           len(projection.Graph.Edges),
		ProjectionUnresolved: append([]string{}, projection.Unresolved...),
		Impact:               impact,
		AuthorityLimits: []string{
			"proposed-change-not-verified-source-diff",
			"derived-index-not-accepted-business-meaning",
			"dependency-closure-and-evidence-receipts-not-proven",
			"runtime-authority-not-evaluated",
		},
	}
	result.Decision = classifyWorkspaceImpact(result, projection.Graph)
	return result, nil
}

// A positive typed impact can propose a recheck. A negative candidate impact
// can NEVER infer reuse, since native effects, provenance and oracle closure
// are outside this projection's contract.
func classifyWorkspaceImpact(report WorkspaceChangeReport, graph modelref.Graph) Decision {
	decision := Decision{
		CaseID:           "workspace-typed-impact",
		Purpose:          "review-proposed-change",
		ClaimID:          report.Consumer.Subject.Kind + "/" + report.Consumer.Subject.ID,
		RuntimeChecks:    []string{"runtime-authority-must-be-evaluated-separately"},
		ExecutionAllowed: false,
	}
	switch {
	case !report.Impact.GraphValid:
		decision.Action, decision.ReasonCodes = "unresolved", []string{"typed-graph-or-change-invalid"}
		return decision
	case len(report.ProjectionUnresolved) > 0:
		decision.Action, decision.ReasonCodes = "unresolved", []string{"workspace-projection-unresolved"}
		return decision
	}
	found := false
	for _, node := range graph.Nodes {
		if node.Key == report.Consumer {
			found = true
			break
		}
	}
	if !found {
		decision.Action, decision.ReasonCodes = "unresolved", []string{"consumer-not-in-typed-graph"}
		return decision
	}
	if report.Changed.Key == report.Consumer {
		decision.Action, decision.ReasonCodes = "recheck", []string{"directly-changed-subject"}
		return decision
	}
	for _, affected := range report.Impact.Affected {
		if affected.Consumer == report.Consumer && affected.Changed == report.Changed.Key {
			decision.Action, decision.ReasonCodes = "recheck", []string{"typed-reference-impact-candidate"}
			return decision
		}
	}
	decision.Action, decision.ReasonCodes = "unresolved", []string{"no-bound-receipt-or-proven-dependency-closure"}
	return decision
}
