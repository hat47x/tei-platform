package reacandidate

import (
	"fmt"
	"path"
	"sort"
	"strconv"
	"strings"
)

// GenerateCandidates projects one decoded REA attempt into three-element
// analysis structure candidates.
//
// The projection is deliberately conservative:
//
//   - One data-element candidate per source-backed observation, so every
//     candidate names the bytes it came from.
//   - One cross-reference candidate per directed edge, preserving the
//     observed from→to order, the relation label, and the external edge id.
//     Parallel edges and opposite-direction pairs are never merged.
//   - An edge endpoint that this attempt did not observe stays unresolved:
//     no node is fabricated to complete it.
//   - REA's own coverage, limitations and unknowns are carried through, and
//     they cap the qualification at partial.
//
// Nothing here accepts meaning, writes a model, or resolves a Canonical
// subject. Those remain with the Owner.
func GenerateCandidates(evidence AttemptEvidence) (CandidateSet, error) {
	if strings.TrimSpace(evidence.EvidenceID) == "" {
		return CandidateSet{}, refuse("EVIDENCE_ID_MISSING", "attempt evidence has no evidence id")
	}
	if evidence.Provider.ID == "" || evidence.Provider.Version == "" {
		return CandidateSet{}, refuse("PROVIDER_DESCRIPTOR_INCOMPLETE", "attempt evidence has no complete provider descriptor")
	}

	nodesByID := map[string]ExternalNode{}
	var nodeOrder []string
	for _, node := range evidence.Normalized.Graph.Nodes {
		if _, exists := nodesByID[node.NodeID]; exists {
			return CandidateSet{}, refuse("NODE_IDENTITY_DUPLICATE", "attempt graph node_id %q is declared more than once", node.NodeID)
		}
		nodesByID[node.NodeID] = node
		nodeOrder = append(nodeOrder, node.NodeID)
	}

	var candidates []StructureCandidate
	var notes []string
	// observedNodeIDs records which external node identities produced at least
	// one data candidate. An edge endpoint outside this set is unresolved.
	observedNodeIDs := map[string]bool{}

	for _, nodeID := range nodeOrder {
		node := nodesByID[nodeID]
		if len(node.Observations) == 0 {
			notes = append(notes, "node-without-observation:"+nodeID)
			continue
		}
		if len(node.Observations) > 1 {
			notes = append(notes, "node-observed-multiple-times:"+nodeID)
		}
		for _, observation := range node.Observations {
			candidates = append(candidates, newDataCandidate(node, observation))
			observedNodeIDs[nodeID] = true
		}
	}

	// Group directed edges before generating candidates so parallel and
	// opposite-direction pairs can be named as such. Parallel means several
	// edges between the same ordered endpoints; a repeated label on top of
	// that is a stronger, separately named ambiguity.
	type endpointKey struct{ from, to string }
	type labelKey struct{ from, to, relation string }
	parallel := map[endpointKey]int{}
	sameLabel := map[labelKey]int{}
	reverse := map[endpointKey]bool{}
	for _, edge := range evidence.Normalized.Graph.Edges {
		parallel[endpointKey{edge.SourceNodeID, edge.TargetNodeID}]++
		sameLabel[labelKey{edge.SourceNodeID, edge.TargetNodeID, edge.Relation}]++
		reverse[endpointKey{edge.TargetNodeID, edge.SourceNodeID}] = true
	}
	for _, edge := range evidence.Normalized.Graph.Edges {
		candidates = append(candidates, newRelationCandidate(
			edge,
			observedNodeIDs,
			parallel[endpointKey{edge.SourceNodeID, edge.TargetNodeID}],
			sameLabel[labelKey{edge.SourceNodeID, edge.TargetNodeID, edge.Relation}],
			reverse[endpointKey{edge.SourceNodeID, edge.TargetNodeID}],
		))
	}

	annotateSameSuggestedNames(candidates, &notes)
	sortCandidates(candidates)

	set := CandidateSet{
		SchemaVersion: candidateSetSchemaVersion,
		Effect:        EffectReadOnly,
		Attempt: AttemptProvenance{
			EvidenceID:       evidence.EvidenceID,
			Operation:        evidence.Operation,
			Provider:         evidence.Provider,
			SubjectDigest:    evidence.Subject.DigestSHA256,
			GraphCoverage:    evidence.Normalized.Graph.Coverage,
			SemanticCoverage: evidence.Normalized.SemanticGraph.Coverage,
			Limitations:      sortStrings(append([]string(nil), evidence.Limitations...)),
		},
		Candidates:      candidates,
		Unknowns:        append([]ExternalUnknown(nil), evidence.Normalized.SemanticGraph.Unknowns...),
		Notes:           uniqueStrings(notes),
		Qualification:   qualify(evidence, candidates),
		AuthorityLimits: CandidateAuthorityLimits(),
	}
	set.Fingerprint = SetFingerprint(set)
	return set, nil
}

// dataCandidateBasis names the structural observations that support treating
// one REA node observation as a data-element candidate. Every entry is a
// structural fact, never a business claim.
func dataCandidateBasis(node ExternalNode, revision SourceRevision) []string {
	basis := []string{"rea-graph-node-observation"}
	if node.Kind != "" {
		basis = append(basis, "node-kind:"+node.Kind)
	}
	if revision.ArtifactPath != "" {
		basis = append(basis, "source:"+revision.ArtifactPath)
	}
	if revision.Span != nil {
		basis = append(basis, "source-span:"+spanLabel(revision.Span))
	}
	return basis
}

func spanLabel(span *Span) string {
	endLine, endColumn := span.Start.Line, span.Start.Column
	if span.End != nil {
		endLine, endColumn = span.End.Line, span.End.Column
	}
	return strconv.Itoa(span.Start.Line) + ":" + strconv.Itoa(span.Start.Column) +
		"-" + strconv.Itoa(endLine) + ":" + strconv.Itoa(endColumn)
}

// newDataCandidate builds one data-element candidate from one observation.
func newDataCandidate(node ExternalNode, observation ExternalObservation) StructureCandidate {
	occurrence := observation.Evidence
	revision := newSourceRevision(occurrence)
	candidate := StructureCandidate{
		CandidateID: candidateID(
			CandidateKindDataElement,
			node.NodeID,
			observation.ObservationID,
			"", "",
		),
		Kind: CandidateKindDataElement,
		Facts: StructuralFacts{
			ExternalOccurrenceID: observation.ObservationID,
			ExternalIdentity:     node.NodeID,
			ObservedKind:         node.Kind,
			ObservedState:        occurrence.State,
			ObservedAuthority:    occurrence.Authority,
			EvidenceCoverage:     coverageStatus(occurrence.Coverage),
		},
		Inference: CandidateInference{
			TargetElement: "data",
			Basis:         uniqueStrings(dataCandidateBasis(node, revision)),
			Unresolved: []string{
				"business-meaning",
				"canonical-subject",
				"attribute-set",
				"identity-key",
				"lifecycle",
			},
		},
		SourceRevision: revision,
		MeaningLoss: []string{
			"external-node-identity-has-no-three-element-field",
			"external-observation-identity-has-no-three-element-field",
			"observed-state-has-no-three-element-field",
			"observed-authority-has-no-three-element-field",
			"evidence-coverage-has-no-three-element-field",
			"source-span-has-no-three-element-field",
			"artifact-digest-has-no-three-element-field",
		},
		ReviewStatus: ReviewStatusRequiresOwner,
		Resolution:   ResolutionUnresolved,
	}
	if name := suggestedName(revision.ArtifactPath); name != "" {
		candidate.Inference.SuggestedName = name
		candidate.Inference.Basis = append(candidate.Inference.Basis, "suggested-name-from-source-artifact-basename")
		candidate.Inference.Unresolved = append(candidate.Inference.Unresolved, "display-name-business-meaning")
		candidate.Inference.Basis = sortStrings(candidate.Inference.Basis)
		candidate.Inference.Unresolved = sortStrings(candidate.Inference.Unresolved)
	}
	candidate.Unresolved = []string{"canonical-subject-mapping"}
	if revision.Binding == SourceBindingUnavailable {
		candidate.Unresolved = append(candidate.Unresolved, "source-location")
	}
	if revision.Fingerprint == "" {
		candidate.Unresolved = append(candidate.Unresolved, "artifact-digest")
	}
	candidate.Unresolved = sortStrings(candidate.Unresolved)
	return candidate
}

// newRelationCandidate builds one cross-reference candidate from one directed
// edge. Direction, relation label and edge identity are preserved exactly.
func newRelationCandidate(edge ExternalEdge, observedNodeIDs map[string]bool, parallelCount, sameLabelCount int, hasReverse bool) StructureCandidate {
	occurrence := edge.Evidence
	revision := newSourceRevision(occurrence)
	direction := &Direction{FromExternalID: edge.SourceNodeID, ToExternalID: edge.TargetNodeID}

	basis := []string{"rea-directed-edge"}
	if edge.Relation != "" {
		basis = append(basis, "relation-label:"+edge.Relation)
	}
	unresolved := []string{
		"relation-business-meaning",
		"cardinality",
		"ownership-direction",
		"endpoint-canonical-subjects",
	}
	var notes []string
	if observedNodeIDs[edge.SourceNodeID] {
		basis = append(basis, "endpoint-observed-as-data-candidate:"+edge.SourceNodeID)
	} else {
		unresolved = append(unresolved, "source-endpoint-not-observed-in-this-attempt:"+edge.SourceNodeID)
		notes = append(notes, "unresolved-source-endpoint")
	}
	if observedNodeIDs[edge.TargetNodeID] {
		basis = append(basis, "endpoint-observed-as-data-candidate:"+edge.TargetNodeID)
	} else {
		unresolved = append(unresolved, "target-endpoint-not-observed-in-this-attempt:"+edge.TargetNodeID)
		notes = append(notes, "unresolved-target-endpoint")
	}
	if edge.SourceNodeID == edge.TargetNodeID {
		notes = append(notes, "self-relation")
		unresolved = append(unresolved, "self-relation-meaning")
	}
	if parallelCount > 1 {
		notes = append(notes, "parallel-relation")
		unresolved = append(unresolved, "parallel-relation-meaning")
	}
	if sameLabelCount > 1 {
		notes = append(notes, "same-endpoints-same-label")
		unresolved = append(unresolved, "same-endpoints-same-label-meaning")
	}
	if hasReverse {
		notes = append(notes, "direction-conflict-pair")
		unresolved = append(unresolved, "direction-conflict-resolution")
	}

	candidate := StructureCandidate{
		CandidateID: candidateID(
			CandidateKindCrossReference,
			edge.EdgeID,
			edge.EdgeID,
			edge.SourceNodeID,
			edge.TargetNodeID,
		),
		Kind: CandidateKindCrossReference,
		Facts: StructuralFacts{
			ExternalOccurrenceID: edge.EdgeID,
			ExternalIdentity:     edge.EdgeID,
			ObservedKind:         edge.Relation,
			Direction:            direction,
			ObservedState:        occurrence.State,
			ObservedAuthority:    occurrence.Authority,
			EvidenceCoverage:     coverageStatus(occurrence.Coverage),
		},
		Inference: CandidateInference{
			TargetElement: "cross-reference",
			SuggestedKind: "data-to-data",
			Basis:         uniqueStrings(basis),
			Unresolved:    sortStrings(unresolved),
		},
		SourceRevision: revision,
		MeaningLoss: []string{
			"external-edge-identity-has-no-cross-reference-field",
			"external-endpoint-identities-have-no-cross-reference-field",
			"observed-direction-is-encoded-in-from-to-order-only",
			"observed-state-has-no-three-element-field",
			"observed-authority-has-no-three-element-field",
			"evidence-coverage-has-no-three-element-field",
			"relation-label-has-no-typed-vocabulary",
		},
		Notes:        uniqueStrings(notes),
		ReviewStatus: ReviewStatusRequiresOwner,
		Resolution:   ResolutionUnresolved,
	}
	candidate.Unresolved = []string{"canonical-subject-mapping"}
	if revision.Binding == SourceBindingUnavailable {
		candidate.Unresolved = append(candidate.Unresolved, "source-location")
	}
	if revision.Fingerprint == "" {
		candidate.Unresolved = append(candidate.Unresolved, "artifact-digest")
	}
	candidate.Unresolved = sortStrings(candidate.Unresolved)
	return candidate
}

// newSourceRevision copies the provider-reported source of one occurrence.
// Without an external artifact set the digest is provider-reported, which is
// recorded as such rather than as an independent verification.
func newSourceRevision(occurrence OccurrenceEvidence) SourceRevision {
	revision := SourceRevision{Binding: SourceBindingUnavailable}
	location := occurrence.Location
	if location == nil || !location.Available || location.Value == nil {
		revision.BindingNotes = append(revision.BindingNotes, "provider-reported-no-location")
		return revision
	}
	value := location.Value
	switch value.Kind {
	case "artifact-path":
		if !safeRelativePath(value.Path) {
			revision.BindingNotes = append(revision.BindingNotes, "provider-reported-invalid-artifact-path")
			return revision
		}
		revision.ArtifactPath = value.Path
	case "source-range":
		source := value.Source
		if source == "" {
			source = value.Path
		}
		if !safeRelativePath(source) {
			revision.BindingNotes = append(revision.BindingNotes, "provider-reported-invalid-source-path")
			return revision
		}
		revision.ArtifactPath = source
		if value.Start != nil {
			revision.Span = &Span{Start: Position{Line: value.Start.Line, Column: value.Start.Column}}
			if value.End != nil {
				revision.Span.End = &Position{Line: value.End.Line, Column: value.End.Column}
			}
		}
	default:
		revision.BindingNotes = append(revision.BindingNotes, "provider-reported-unknown-location-kind:"+value.Kind)
		return revision
	}
	if occurrence.Artifact.SHA256 != "" {
		if !isSHA256Hex(occurrence.Artifact.SHA256) {
			revision.BindingNotes = append(revision.BindingNotes, "provider-reported-invalid-artifact-digest")
			return revision
		}
		revision.Fingerprint = "sha256:" + occurrence.Artifact.SHA256
	}
	if revision.ArtifactPath != "" && revision.Fingerprint != "" {
		revision.Binding = SourceBindingProviderReported
	} else {
		revision.Binding = SourceBindingUnavailable
		if revision.ArtifactPath == "" {
			revision.BindingNotes = append(revision.BindingNotes, "provider-reported-no-artifact-path")
		}
		if revision.Fingerprint == "" {
			revision.BindingNotes = append(revision.BindingNotes, "provider-reported-no-artifact-digest")
		}
	}
	return revision
}

// annotateSameSuggestedNames records display-name collisions between
// structurally distinct external nodes. The candidates are never merged by
// name; the collision is surfaced for the Owner to disambiguate.
func annotateSameSuggestedNames(candidates []StructureCandidate, notes *[]string) {
	byName := map[string][]string{}
	for _, candidate := range candidates {
		if candidate.Kind != CandidateKindDataElement {
			continue
		}
		name := candidate.Inference.SuggestedName
		if name == "" {
			continue
		}
		byName[name] = append(byName[name], candidate.Facts.ExternalIdentity)
	}
	names := make([]string, 0, len(byName))
	for name := range byName {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		identities := uniqueStrings(byName[name])
		if len(identities) < 2 {
			continue
		}
		*notes = append(*notes, "same-suggested-name-distinct-external-identity:"+name+":"+strings.Join(identities, ","))
		for i := range candidates {
			if candidates[i].Kind == CandidateKindDataElement && candidates[i].Inference.SuggestedName == name {
				candidates[i].Notes = append(candidates[i].Notes, "same-suggested-name-distinct-external-identity")
			}
		}
	}
}

// qualify caps the attempt qualification using REA's own statements. A nil
// coverage record means REA said nothing, which is unknown and never
// complete.
func qualify(evidence AttemptEvidence, candidates []StructureCandidate) string {
	if evidence.Normalized.Graph.Coverage == nil {
		return "unknown"
	}
	exhausted := evidence.Normalized.Graph.Coverage.Exhausted
	if exhausted == nil || !*exhausted {
		return "partial"
	}
	if len(evidence.Limitations) > 0 || len(evidence.Normalized.SemanticGraph.Unknowns) > 0 {
		return "partial"
	}
	if semantic := evidence.Normalized.SemanticGraph.Coverage; semantic != nil && semantic.Exhausted != nil && !*semantic.Exhausted {
		return "partial"
	}
	for _, candidate := range candidates {
		if candidate.SourceRevision.Binding != SourceBindingProviderReported {
			return "partial"
		}
		unresolved := append(append([]string(nil), candidate.Unresolved...), candidate.Inference.Unresolved...)
		for _, item := range unresolved {
			if strings.HasPrefix(item, "source-endpoint-not-observed") ||
				strings.HasPrefix(item, "target-endpoint-not-observed") {
				return "partial"
			}
		}
	}
	return "complete"
}

func coverageStatus(coverage *CoverageRecord) string {
	if coverage == nil {
		return ""
	}
	return coverage.Status
}

func suggestedName(artifactPath string) string {
	if artifactPath == "" {
		return ""
	}
	base := path.Base(artifactPath)
	if base == "." || base == "/" || base == ".." {
		return ""
	}
	return strings.TrimSuffix(base, path.Ext(base))
}

// safeRelativePath accepts only a workspace-relative POSIX path, mirroring the
// Source Index path rule so an external path cannot escape a workspace.
func safeRelativePath(value string) bool {
	if value == "" || strings.ContainsAny(value, "\\\x00") || strings.HasPrefix(value, "/") {
		return false
	}
	clean := path.Clean(value)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") {
		return false
	}
	if strings.Contains(strings.SplitN(clean, "/", 2)[0], ":") {
		return false
	}
	return clean == value
}

// String renders a candidate for logs; it never includes source bytes.
func (c StructureCandidate) String() string {
	return fmt.Sprintf("%s %s external=%s", c.Kind, c.CandidateID, c.Facts.ExternalIdentity)
}
