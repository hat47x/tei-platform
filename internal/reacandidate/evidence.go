package reacandidate

import (
	"encoding/json"
	"fmt"
	"strings"
)

// maxAttemptRecords bounds one decoded attempt. REA output is an external,
// untrusted document: a bound keeps a malformed or hostile file from turning
// into unbounded work, and it is recorded rather than silently truncated.
const maxAttemptRecords = 4096

// reservedControlMarker is the REA JSON Lines control key. It never appears in
// a data record; seeing it inside a decoded occurrence means the caller passed
// a control stream where a data document was promised.
const reservedControlMarker = "$rea"

// supportedGraphSchema is the only graph schema this intake interprets. A
// different schema is refused instead of guessed at, because node/edge field
// meaning is schema-specific.
const supportedGraphSchema = "JavaScriptApplicationGraph"

// AttemptEvidence is ONE REA analysis attempt: the provider descriptor, the
// analyzed subject digest, and the observations and directed relations REA
// reported for it. Every record carries its own occurrence evidence, so a
// candidate generated from this attempt can name the bytes it came from.
//
// This type is the external contract boundary. It is decoded losslessly for
// the fields named below and tolerates additional REA fields; it never
// carries TEI Canonical meaning, acceptance, or authority.
type AttemptEvidence struct {
	EvidenceID   string             `json:"evidenceId"`
	Operation    string             `json:"operation"`
	Provider     ProviderDescriptor `json:"provider"`
	Subject      SubjectDescriptor  `json:"subject"`
	Normalized   NormalizedResult   `json:"normalizedResult"`
	Limitations  []string           `json:"limitations,omitempty"`
	RootDigestOK bool               `json:"rootDigestAgreement"`
}

// ProviderDescriptor names the external analyzer that produced the attempt.
type ProviderDescriptor struct {
	ID      string `json:"id"`
	Version string `json:"version"`
}

// SubjectDescriptor identifies what was analyzed.
type SubjectDescriptor struct {
	DigestSHA256 string `json:"digestSha256"`
}

// NormalizedResult is REA's normalized view of the analyzed subject.
type NormalizedResult struct {
	RootArtifactSHA256 string        `json:"rootArtifactSha256,omitempty"`
	Graph              ExternalGraph `json:"graph"`
	SemanticGraph      SemanticGraph `json:"semanticGraph"`
}

// ExternalGraph holds the observed nodes and directed edges.
type ExternalGraph struct {
	Schema   string          `json:"schema"`
	Nodes    []ExternalNode  `json:"nodes"`
	Edges    []ExternalEdge  `json:"edges"`
	Coverage *CoverageRecord `json:"coverage,omitempty"`
}

// ExternalNode is one observed graph node with its source-backed observations.
type ExternalNode struct {
	NodeID       string                `json:"nodeId"`
	Kind         string                `json:"kind,omitempty"`
	Observations []ExternalObservation `json:"observations"`
}

// ExternalObservation is one source-backed observation of a node.
type ExternalObservation struct {
	ObservationID string             `json:"observationId"`
	Evidence      OccurrenceEvidence `json:"evidence"`
}

// ExternalEdge is one directed relation REA observed between two nodes.
type ExternalEdge struct {
	EdgeID       string             `json:"edgeId"`
	SourceNodeID string             `json:"sourceNodeId"`
	TargetNodeID string             `json:"targetNodeId"`
	Relation     string             `json:"relation,omitempty"`
	Evidence     OccurrenceEvidence `json:"evidence"`
}

// OccurrenceEvidence is the source backing of one observation or relation.
type OccurrenceEvidence struct {
	State     string          `json:"state,omitempty"`
	Authority string          `json:"authority,omitempty"`
	Artifact  ArtifactDigest  `json:"artifact,omitempty"`
	Location  *LocationRecord `json:"location,omitempty"`
	Coverage  *CoverageRecord `json:"coverage,omitempty"`
}

// ArtifactDigest is the analyzed artifact's digest as reported by REA.
type ArtifactDigest struct {
	SHA256 string `json:"sha256,omitempty"`
}

// LocationRecord is REA's source location descriptor.
type LocationRecord struct {
	Available bool            `json:"available"`
	Value     *LocationValues `json:"value,omitempty"`
}

// LocationValues carries either an artifact path or a source range.
type LocationValues struct {
	Kind   string `json:"kind"`
	Path   string `json:"path,omitempty"`
	Source string `json:"source,omitempty"`
	Start  *Point `json:"start,omitempty"`
	End    *Point `json:"end,omitempty"`
}

// Point is a source position as REA reports it.
type Point struct {
	Line   int `json:"line"`
	Column int `json:"column"`
}

// SemanticGraph is REA's semantic view, including what it could not resolve.
type SemanticGraph struct {
	Unknowns []ExternalUnknown `json:"unknowns,omitempty"`
	Coverage *CoverageRecord   `json:"coverage,omitempty"`
}

// ExternalUnknown is one unresolved item REA declared.
type ExternalUnknown struct {
	UnknownID string `json:"unknownId"`
	Family    string `json:"family,omitempty"`
	Reason    string `json:"reason,omitempty"`
}

// CoverageRecord is REA's own coverage statement. exhausted==false means the
// attempt is partial by REA's own account; a nil record means REA said
// nothing, which is unknown and never complete.
type CoverageRecord struct {
	Status    string   `json:"status,omitempty"`
	Exhausted *bool    `json:"exhausted,omitempty"`
	Notes     []string `json:"notes,omitempty"`
}

// EvidenceError reports why an external document was refused. The code is
// stable so a caller can branch without matching prose.
type EvidenceError struct {
	Code    string
	Message string
}

func (e *EvidenceError) Error() string { return e.Message }

func refuse(code, format string, args ...any) error {
	return &EvidenceError{Code: code, Message: fmt.Sprintf(format, args...)}
}

// Wire types mirror REA's own document keys. The internal AttemptEvidence
// uses TEI's camelCase, so a change in REA's key names is confined here.
type reaEnvelope struct {
	OK   *bool          `json:"ok"`
	Data *reaAttemptDoc `json:"data"`
}

type reaAttemptDoc struct {
	Operation        string         `json:"operation"`
	EvidenceID       string         `json:"evidence_id"`
	Provider         *reaProvider   `json:"provider"`
	Subject          *reaSubject    `json:"subject"`
	NormalizedResult *reaNormalized `json:"normalized_result"`
	Limitations      []string       `json:"limitations"`
}

type reaProvider struct {
	ID      string `json:"id"`
	Version string `json:"version"`
}

type reaSubject struct {
	Digest *reaDigest `json:"digest"`
}

type reaDigest struct {
	SHA256 string `json:"sha256"`
}

type reaNormalized struct {
	RootArtifactSHA256 string       `json:"root_artifact_sha256"`
	Graph              *reaGraph    `json:"graph"`
	SemanticGraph      *reaSemantic `json:"semantic_graph"`
}

type reaGraph struct {
	Schema   string       `json:"schema"`
	Nodes    []reaNode    `json:"nodes"`
	Edges    []reaEdge    `json:"edges"`
	Coverage *reaCoverage `json:"coverage"`
}

type reaNode struct {
	NodeID       string           `json:"node_id"`
	Kind         string           `json:"kind"`
	Observations []reaObservation `json:"observations"`
	Reserved     *json.RawMessage `json:"$rea"`
}

type reaObservation struct {
	ObservationID string           `json:"observation_id"`
	Evidence      *reaEvidence     `json:"evidence"`
	Reserved      *json.RawMessage `json:"$rea"`
}

type reaEdge struct {
	EdgeID       string           `json:"edge_id"`
	SourceNodeID string           `json:"source_node_id"`
	TargetNodeID string           `json:"target_node_id"`
	Relation     string           `json:"relation"`
	Evidence     *reaEvidence     `json:"evidence"`
	Reserved     *json.RawMessage `json:"$rea"`
}

type reaEvidence struct {
	State     string       `json:"state"`
	Authority string       `json:"authority"`
	Artifact  *reaArtifact `json:"artifact"`
	Location  *reaLocation `json:"location"`
	Coverage  *reaCoverage `json:"coverage"`
}

type reaArtifact struct {
	SHA256 string `json:"sha256"`
}

type reaLocation struct {
	Available bool          `json:"available"`
	Value     *reaLocValues `json:"value"`
}

type reaLocValues struct {
	Kind   string    `json:"kind"`
	Path   string    `json:"path"`
	Source string    `json:"source"`
	Start  *reaPoint `json:"start"`
	End    *reaPoint `json:"end"`
}

type reaPoint struct {
	Line   int `json:"line"`
	Column int `json:"column"`
}

type reaSemantic struct {
	Unknowns []reaUnknown `json:"unknowns"`
	Coverage *reaCoverage `json:"coverage"`
}

type reaUnknown struct {
	UnknownID string `json:"unknown_id"`
	Family    string `json:"family"`
	Reason    string `json:"reason"`
}

type reaCoverage struct {
	Status    string   `json:"status"`
	Exhausted *bool    `json:"exhausted"`
	Notes     []string `json:"notes"`
}

// DecodeAttemptEvidence decodes and validates one REA attempt document.
//
// The document is untrusted external input. Decoding fails closed: a reported
// failure, a missing provider descriptor, a digest conflict, duplicate
// identities, a reserved control marker inside a record, or an unbounded
// record count are refused with a code instead of being repaired. Unknown
// extra fields are tolerated, because REA retains context this intake does
// not interpret.
func DecodeAttemptEvidence(raw []byte) (AttemptEvidence, error) {
	var envelope reaEnvelope
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return AttemptEvidence{}, refuse("EVIDENCE_INVALID_JSON", "REA evidence is not one JSON document: %v", err)
	}
	if envelope.OK == nil {
		return AttemptEvidence{}, refuse("EVIDENCE_STATUS_MISSING", "REA evidence envelope has no ok field")
	}
	if !*envelope.OK {
		return AttemptEvidence{}, refuse("EVIDENCE_REPORTED_FAILURE", "REA evidence reports a failed analysis; a failure is not a partial success")
	}
	if envelope.Data == nil {
		return AttemptEvidence{}, refuse("EVIDENCE_DATA_MISSING", "REA evidence envelope has no data")
	}
	doc := envelope.Data
	if strings.TrimSpace(doc.Operation) == "" {
		return AttemptEvidence{}, refuse("OPERATION_MISSING", "REA evidence has no operation")
	}
	if strings.TrimSpace(doc.EvidenceID) == "" {
		return AttemptEvidence{}, refuse("EVIDENCE_ID_MISSING", "REA evidence has no evidence_id")
	}
	if doc.Provider == nil || strings.TrimSpace(doc.Provider.ID) == "" || strings.TrimSpace(doc.Provider.Version) == "" {
		return AttemptEvidence{}, refuse("PROVIDER_DESCRIPTOR_INCOMPLETE", "REA evidence has no complete provider descriptor (id and version)")
	}
	if doc.Subject == nil || doc.Subject.Digest == nil || !isSHA256Hex(doc.Subject.Digest.SHA256) {
		return AttemptEvidence{}, refuse("SUBJECT_DIGEST_INVALID", "REA evidence subject digest is missing or not a 64-character lowercase hex sha256")
	}
	if doc.NormalizedResult == nil || doc.NormalizedResult.Graph == nil {
		return AttemptEvidence{}, refuse("GRAPH_MISSING", "REA evidence has no normalized graph")
	}
	graph := doc.NormalizedResult.Graph
	if graph.Schema != supportedGraphSchema {
		return AttemptEvidence{}, refuse("GRAPH_SCHEMA_UNSUPPORTED", "REA graph schema %q is not the interpreted schema %q", graph.Schema, supportedGraphSchema)
	}
	if len(graph.Nodes) > maxAttemptRecords || len(graph.Edges) > maxAttemptRecords {
		return AttemptEvidence{}, refuse("ATTEMPT_TOO_LARGE", "REA graph exceeds the %d-record intake bound", maxAttemptRecords)
	}
	subjectDigest := doc.Subject.Digest.SHA256
	rootDigest := doc.NormalizedResult.RootArtifactSHA256
	if rootDigest != "" && rootDigest != subjectDigest {
		return AttemptEvidence{}, refuse("DIGEST_CONFLICT", "REA subject digest %s disagrees with root artifact digest %s", subjectDigest, rootDigest)
	}

	evidence := AttemptEvidence{
		EvidenceID:  doc.EvidenceID,
		Operation:   doc.Operation,
		Provider:    ProviderDescriptor{ID: doc.Provider.ID, Version: doc.Provider.Version},
		Subject:     SubjectDescriptor{DigestSHA256: subjectDigest},
		Limitations: append([]string(nil), doc.Limitations...),
		Normalized: NormalizedResult{
			RootArtifactSHA256: rootDigest,
			Graph: ExternalGraph{
				Schema:   graph.Schema,
				Coverage: decodeCoverage(graph.Coverage),
			},
			SemanticGraph: SemanticGraph{Coverage: decodeCoverage(semanticCoverageOf(doc.NormalizedResult))},
		},
	}
	evidence.RootDigestOK = rootDigest == "" || rootDigest == subjectDigest

	seenNodes := map[string]bool{}
	seenObservations := map[string]bool{}
	for _, node := range graph.Nodes {
		if strings.TrimSpace(node.NodeID) == "" {
			return AttemptEvidence{}, refuse("NODE_IDENTITY_MISSING", "REA graph node has no node_id")
		}
		if node.Reserved != nil {
			return AttemptEvidence{}, refuse("RESERVED_CONTROL_MARKER", "REA graph node %q carries the reserved %s control marker", node.NodeID, reservedControlMarker)
		}
		if seenNodes[node.NodeID] {
			return AttemptEvidence{}, refuse("NODE_IDENTITY_DUPLICATE", "REA graph node_id %q is declared more than once", node.NodeID)
		}
		seenNodes[node.NodeID] = true
		if len(node.Observations) > maxAttemptRecords {
			return AttemptEvidence{}, refuse("ATTEMPT_TOO_LARGE", "REA node %q exceeds the %d-observation intake bound", node.NodeID, maxAttemptRecords)
		}
		decoded := ExternalNode{NodeID: node.NodeID, Kind: node.Kind}
		for _, observation := range node.Observations {
			if strings.TrimSpace(observation.ObservationID) == "" {
				return AttemptEvidence{}, refuse("OBSERVATION_IDENTITY_MISSING", "REA node %q has an observation with no observation_id", node.NodeID)
			}
			if observation.Reserved != nil {
				return AttemptEvidence{}, refuse("RESERVED_CONTROL_MARKER", "REA observation %q carries the reserved %s control marker", observation.ObservationID, reservedControlMarker)
			}
			if seenObservations[observation.ObservationID] {
				return AttemptEvidence{}, refuse("OBSERVATION_IDENTITY_DUPLICATE", "REA observation_id %q is declared more than once", observation.ObservationID)
			}
			seenObservations[observation.ObservationID] = true
			if observation.Evidence == nil {
				return AttemptEvidence{}, refuse("OCCURRENCE_EVIDENCE_MISSING", "REA observation %q has no evidence", observation.ObservationID)
			}
			decoded.Observations = append(decoded.Observations, ExternalObservation{
				ObservationID: observation.ObservationID,
				Evidence:      decodeOccurrenceEvidence(observation.Evidence),
			})
		}
		evidence.Normalized.Graph.Nodes = append(evidence.Normalized.Graph.Nodes, decoded)
	}

	seenEdges := map[string]bool{}
	for _, edge := range graph.Edges {
		if strings.TrimSpace(edge.EdgeID) == "" {
			return AttemptEvidence{}, refuse("EDGE_IDENTITY_MISSING", "REA graph edge has no edge_id")
		}
		if edge.Reserved != nil {
			return AttemptEvidence{}, refuse("RESERVED_CONTROL_MARKER", "REA edge %q carries the reserved %s control marker", edge.EdgeID, reservedControlMarker)
		}
		if seenEdges[edge.EdgeID] {
			return AttemptEvidence{}, refuse("EDGE_IDENTITY_DUPLICATE", "REA edge_id %q is declared more than once", edge.EdgeID)
		}
		seenEdges[edge.EdgeID] = true
		if strings.TrimSpace(edge.SourceNodeID) == "" || strings.TrimSpace(edge.TargetNodeID) == "" {
			return AttemptEvidence{}, refuse("EDGE_ENDPOINT_MISSING", "REA edge %q has no source or target node identity", edge.EdgeID)
		}
		if edge.Evidence == nil {
			return AttemptEvidence{}, refuse("OCCURRENCE_EVIDENCE_MISSING", "REA edge %q has no evidence", edge.EdgeID)
		}
		evidence.Normalized.Graph.Edges = append(evidence.Normalized.Graph.Edges, ExternalEdge{
			EdgeID:       edge.EdgeID,
			SourceNodeID: edge.SourceNodeID,
			TargetNodeID: edge.TargetNodeID,
			Relation:     edge.Relation,
			Evidence:     decodeOccurrenceEvidence(edge.Evidence),
		})
	}

	if semantic := doc.NormalizedResult.SemanticGraph; semantic != nil {
		if len(semantic.Unknowns) > maxAttemptRecords {
			return AttemptEvidence{}, refuse("ATTEMPT_TOO_LARGE", "REA semantic graph exceeds the %d-unknown intake bound", maxAttemptRecords)
		}
		seenUnknowns := map[string]bool{}
		for _, unknown := range semantic.Unknowns {
			if strings.TrimSpace(unknown.UnknownID) == "" {
				return AttemptEvidence{}, refuse("UNKNOWN_IDENTITY_MISSING", "REA semantic unknown has no unknown_id")
			}
			if seenUnknowns[unknown.UnknownID] {
				return AttemptEvidence{}, refuse("UNKNOWN_IDENTITY_DUPLICATE", "REA unknown_id %q is declared more than once", unknown.UnknownID)
			}
			seenUnknowns[unknown.UnknownID] = true
			evidence.Normalized.SemanticGraph.Unknowns = append(evidence.Normalized.SemanticGraph.Unknowns, ExternalUnknown{
				UnknownID: unknown.UnknownID,
				Family:    unknown.Family,
				Reason:    unknown.Reason,
			})
		}
	}
	return evidence, nil
}

func semanticCoverageOf(normalized *reaNormalized) *reaCoverage {
	if normalized == nil || normalized.SemanticGraph == nil {
		return nil
	}
	return normalized.SemanticGraph.Coverage
}

func decodeCoverage(coverage *reaCoverage) *CoverageRecord {
	if coverage == nil {
		return nil
	}
	return &CoverageRecord{
		Status:    coverage.Status,
		Exhausted: coverage.Exhausted,
		Notes:     append([]string(nil), coverage.Notes...),
	}
}

func decodeOccurrenceEvidence(evidence *reaEvidence) OccurrenceEvidence {
	decoded := OccurrenceEvidence{State: evidence.State, Authority: evidence.Authority}
	if evidence.Artifact != nil {
		decoded.Artifact = ArtifactDigest{SHA256: evidence.Artifact.SHA256}
	}
	if evidence.Location != nil {
		location := &LocationRecord{Available: evidence.Location.Available}
		if value := evidence.Location.Value; value != nil {
			location.Value = &LocationValues{
				Kind:   value.Kind,
				Path:   value.Path,
				Source: value.Source,
				Start:  decodePoint(value.Start),
				End:    decodePoint(value.End),
			}
		}
		decoded.Location = location
	}
	decoded.Coverage = decodeCoverage(evidence.Coverage)
	return decoded
}

func decodePoint(point *reaPoint) *Point {
	if point == nil {
		return nil
	}
	return &Point{Line: point.Line, Column: point.Column}
}

func isSHA256Hex(value string) bool {
	if len(value) != 64 {
		return false
	}
	for _, r := range value {
		if !(r >= '0' && r <= '9' || r >= 'a' && r <= 'f') {
			return false
		}
	}
	return true
}
