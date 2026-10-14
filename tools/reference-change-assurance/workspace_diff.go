package main

import (
	"fmt"
	"sort"

	"tei/internal/index"
	"tei/internal/modelref"
	"tei/internal/modelref/derivedbridge"
	"tei/internal/sourceindex"
	"tei/internal/workspace"
)

// SourceArtifactChange is a byte-fingerprint comparison of managed Source,
// not an assertion that the business meaning has changed.
type SourceArtifactChange struct {
	Path              string `json:"path"`
	Kind              string `json:"kind"`
	BeforeFingerprint string `json:"beforeFingerprint,omitempty"`
	AfterFingerprint  string `json:"afterFingerprint,omitempty"`
}

// WorkspaceDiffReport compares two separately loaded Workspace roots.
// Neither the filesystem snapshots nor their origin are authenticated.
type WorkspaceDiffReport struct {
	SchemaVersion              string                 `json:"schemaVersion"`
	Authority                  string                 `json:"authority"`
	Consumer                   modelref.Key           `json:"consumer"`
	BeforeSourceRevision       string                 `json:"beforeSourceRevision"`
	AfterSourceRevision        string                 `json:"afterSourceRevision"`
	SourceChanges              []SourceArtifactChange `json:"sourceChanges"`
	BeforeProjectionUnresolved []string               `json:"beforeProjectionUnresolved"`
	AfterProjectionUnresolved  []string               `json:"afterProjectionUnresolved"`
	BeforeRuleFingerprint      string                 `json:"beforeRuleFingerprint"`
	AfterRuleFingerprint       string                 `json:"afterRuleFingerprint"`
	Impact                     modelref.ImpactReview  `json:"impact"`
	Decision                   Decision               `json:"decision"`
	AuthorityLimits            []string               `json:"authorityLimits"`
}

// diffSourceArtifacts compares only entries in the existing managed Source
// Index. Unmanaged and external files remain outside this observation.
func diffSourceArtifacts(before, after sourceindex.Index) []SourceArtifactChange {
	old := make(map[string]string, len(before.Artifacts))
	next := make(map[string]string, len(after.Artifacts))
	paths := make(map[string]bool, len(before.Artifacts)+len(after.Artifacts))
	for _, artifact := range before.Artifacts {
		old[artifact.Path] = artifact.Fingerprint
		paths[artifact.Path] = true
	}
	for _, artifact := range after.Artifacts {
		next[artifact.Path] = artifact.Fingerprint
		paths[artifact.Path] = true
	}
	names := make([]string, 0, len(paths))
	for p := range paths {
		names = append(names, p)
	}
	sort.Strings(names)
	result := make([]SourceArtifactChange, 0)
	for _, name := range names {
		oldFP, wasPresent := old[name]
		newFP, isPresent := next[name]
		if wasPresent && isPresent && oldFP == newFP {
			continue
		}
		change := SourceArtifactChange{Path: name, BeforeFingerprint: oldFP, AfterFingerprint: newFP}
		switch {
		case !wasPresent:
			change.Kind = "added"
		case !isPresent:
			change.Kind = "removed"
		default:
			change.Kind = "content-changed"
		}
		result = append(result, change)
	}
	return result
}

// reviewWorkspaceDiff calls the existing Source Index and modelref
// ReviewImpactDiff. Both roots are read independently; neither Load is an
// atomic multi-file revision nor evidence of human acceptance.
func reviewWorkspaceDiff(beforeRoot, afterRoot, scope string, consumer modelref.Key) (WorkspaceDiffReport, error) {
	if beforeRoot == "" || afterRoot == "" || scope == "" || consumer.Scope != scope ||
		consumer.Subject.Kind == "" || consumer.Subject.ID == "" {
		return WorkspaceDiffReport{}, fmt.Errorf("before/after Workspace, scope and typed consumer are required")
	}
	before, err := workspace.Load(beforeRoot)
	if err != nil {
		return WorkspaceDiffReport{}, fmt.Errorf("load before Workspace: %w", err)
	}
	after, err := workspace.Load(afterRoot)
	if err != nil {
		return WorkspaceDiffReport{}, fmt.Errorf("load after Workspace: %w", err)
	}
	beforeSources, err := workspace.BuildSourceIndex(beforeRoot, before)
	if err != nil {
		return WorkspaceDiffReport{}, fmt.Errorf("index before Workspace: %w", err)
	}
	afterSources, err := workspace.BuildSourceIndex(afterRoot, after)
	if err != nil {
		return WorkspaceDiffReport{}, fmt.Errorf("index after Workspace: %w", err)
	}
	beforeProjection := derivedbridge.FromIndex(index.Build(before.World), scope)
	afterProjection := derivedbridge.FromIndex(index.Build(after.World), scope)
	review := modelref.ReviewImpactDiff(beforeProjection.Graph, afterProjection.Graph, afterProjection.Rules)
	report := WorkspaceDiffReport{
		SchemaVersion:              "tei.reference.workspace-change-diff/v0",
		Authority:                  "two-workspace-source-index-candidate-no-execution",
		Consumer:                   consumer,
		BeforeSourceRevision:       sourceindex.ModelRevision(beforeSources),
		AfterSourceRevision:        sourceindex.ModelRevision(afterSources),
		SourceChanges:              diffSourceArtifacts(beforeSources, afterSources),
		BeforeProjectionUnresolved: append([]string{}, beforeProjection.Unresolved...),
		AfterProjectionUnresolved:  append([]string{}, afterProjection.Unresolved...),
		BeforeRuleFingerprint:      modelref.RuleFingerprint(beforeProjection.Rules),
		AfterRuleFingerprint:       modelref.RuleFingerprint(afterProjection.Rules),
		Impact:                     review,
		AuthorityLimits: []string{
			"separate-workspace-loads-are-not-atomic-revisions",
			"source-index-covers-managed-files-only",
			"typed-graph-does-not-cover-source-meaning-or-native-effects",
			"no-verified-oracle-receipt-or-execution-authority",
		},
	}
	report.Decision = classifyWorkspaceDiff(report, beforeProjection.Graph, afterProjection.Graph, len(beforeSources.Artifacts) > 0 && len(afterSources.Artifacts) > 0)
	return report, nil
}

func graphHasConsumer(g modelref.Graph, consumer modelref.Key) bool {
	for _, node := range g.Nodes {
		if node.Key == consumer {
			return true
		}
	}
	return false
}

// Positive dependency impact recommends recheck. A source change without a
// corresponding typed graph change is NEVER evidence of no impact or reuse.
func classifyWorkspaceDiff(report WorkspaceDiffReport, before, after modelref.Graph, sourceIndexed bool) Decision {
	d := Decision{
		CaseID:           "workspace-two-snapshot-diff",
		Purpose:          "review-observed-managed-source-change",
		ClaimID:          report.Consumer.Subject.Kind + "/" + report.Consumer.Subject.ID,
		RuntimeChecks:    []string{"runtime-authority-must-be-evaluated-separately"},
		ExecutionAllowed: false,
	}
	unresolved := func(reason string) Decision {
		d.Action, d.ReasonCodes = "unresolved", []string{reason}
		return d
	}
	if !sourceIndexed {
		return unresolved("managed-source-inventory-missing")
	}
	if !report.Impact.GraphValid {
		return unresolved("typed-before-or-after-graph-invalid")
	}
	if report.BeforeRuleFingerprint != report.AfterRuleFingerprint {
		return unresolved("typed-projection-rules-changed")
	}
	if !graphHasConsumer(before, report.Consumer) && !graphHasConsumer(after, report.Consumer) {
		return unresolved("consumer-not-in-either-typed-graph")
	}
	for _, change := range report.Impact.Changes {
		if change.Key == report.Consumer {
			d.Action, d.ReasonCodes = "recheck", []string{"consumer-directly-changed"}
			return d
		}
	}
	for _, affected := range report.Impact.Affected {
		if affected.Consumer == report.Consumer {
			d.Action, d.ReasonCodes = "recheck", []string{"typed-reference-impact-candidate"}
			return d
		}
	}
	if len(report.SourceChanges) > 0 {
		if len(report.Impact.Changes) == 0 {
			return unresolved("source-changed-without-typed-graph-diff")
		}
		return unresolved("source-change-without-proven-consumer-impact")
	}
	if len(report.Impact.Changes) > 0 {
		return unresolved("typed-change-without-managed-source-diff")
	}
	if len(report.BeforeProjectionUnresolved) > 0 || len(report.AfterProjectionUnresolved) > 0 {
		return unresolved("workspace-projection-unresolved")
	}
	return unresolved("no-independent-receipt-or-proven-dependency-closure")
}
