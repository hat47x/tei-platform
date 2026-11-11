package externalanalysis

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
)

// Candidates are the external structural candidates one received Evidence
// carries: the observations the producer made about the target, the relations it
// drew between them, and the unknowns it declared.
//
// They are external results, not TEI meaning. Nothing here is admitted to the
// Canonical World, no candidate is accepted meaning, and no candidate is a
// database observation — the same boundary the receipt itself keeps (see the
// package documentation). What this type adds to the receipt is the *shape* of
// what the producer found, so a later step can ask whether an existing
// read-only operation can carry it without loss.
type Candidates struct {
	EvidenceID   string        `json:"evidenceId"`
	Operation    string        `json:"operation"`
	GraphID      string        `json:"graphId,omitempty"`
	Observations []Observation `json:"observations"`
	Relations    []Relation    `json:"relations"`
	Unknowns     []Unknown     `json:"unknowns"`
	// Coverage is the producer's own statement about how much of the graph it
	// saw, family by family. It is not a page coverage: a graph is not a page.
	Coverage    *GraphCoverage `json:"coverage,omitempty"`
	Limitations []string       `json:"limitations,omitempty"`
}

// Observation is one external node observation, stated as the producer stated
// it. Every field is copied or measured; nothing is inferred from the target.
type Observation struct {
	// NodeID is the producer's node identity. It is not a TEI subject: the
	// two namespaces are never equated here.
	NodeID string `json:"nodeId"`
	// ObservationID is the producer's observation identity when the delivery
	// states one. A projection and its parent can name the same node with
	// different observation identities.
	ObservationID string `json:"observationId,omitempty"`
	Kind          string `json:"kind"`
	Label         string `json:"label,omitempty"`
	// State, Confidence and Authority are the producer's own words for what
	// it saw. They are copied, not graded: "inferred" is not a worse
	// "observed", it is a different claim.
	State      string `json:"state,omitempty"`
	Confidence string `json:"confidence,omitempty"`
	Authority  string `json:"authority,omitempty"`
	// ArtifactSHA256 is the digest of the exact bytes the observation is
	// about, when the producer states one.
	ArtifactSHA256 string `json:"artifactSha256,omitempty"`
	// Location names where in the target the observation is, in the
	// producer's own vocabulary ("source-range:index.js").
	Location string `json:"location,omitempty"`
	// Span is the producer's line/column span. It is not a
	// sourceindex.Range: that type is a byte span validated against TEI
	// workspace content, and an external producer's line/column claim is a
	// different claim that must not be silently equated with one.
	Span *Span `json:"span,omitempty"`
	// Extractor names the producer component that made the observation.
	Extractor string `json:"extractor,omitempty"`
	// EvidenceContextID is the producer's own handle for the evidence this
	// observation rests on. The observation and its context travel together,
	// because the context is where state, confidence and limits live.
	EvidenceContextID string `json:"evidenceContextId,omitempty"`
	// Limitations are the limits the producer attached to this observation's
	// evidence context.
	Limitations []string `json:"limitations,omitempty"`
	// Unavailable names the parts the producer declared it could not
	// observe. An empty list does not mean everything was observed; the
	// producer's coverage and unknowns carry that.
	Unavailable []string `json:"unavailable,omitempty"`
}

// Relation is one external directed edge.
type Relation struct {
	// RelationID is the producer's own relation identity when it states one.
	// When it does not, the handle is derived by TEI from the Evidence ID and
	// the edge's endpoints and kind, and Derived says so. A derived handle
	// must never be presented as the producer's own identifier.
	RelationID string `json:"relationId"`
	Derived    bool   `json:"derived"`
	From       string `json:"from"`
	To         string `json:"to"`
	Kind       string `json:"kind"`
	// Resolution is the producer's own word: resolved, unresolved,
	// ambiguous. It is copied, not graded.
	Resolution string `json:"resolution,omitempty"`
	Location   string `json:"location,omitempty"`
	Span       *Span  `json:"span,omitempty"`
	// EvidenceContextID is the producer's own handle for the evidence this
	// edge rests on.
	EvidenceContextID string `json:"evidenceContextId,omitempty"`
}

// Unknown is one thing the producer declared it could not resolve. It is a
// first-class candidate field, not prose: an unbound endpoint the producer
// named is different from an endpoint the producer never mentioned.
type Unknown struct {
	UnknownID         string   `json:"unknownId,omitempty"`
	NodeID            string   `json:"nodeId,omitempty"`
	Family            string   `json:"family,omitempty"`
	RelationKinds     []string `json:"relationKinds,omitempty"`
	Reason            string   `json:"reason,omitempty"`
	Detail            string   `json:"detail,omitempty"`
	CandidateNodeIDs  []string `json:"candidateNodeIds,omitempty"`
	EvidenceContextID string   `json:"evidenceContextId,omitempty"`
	Location          string   `json:"location,omitempty"`
	Span              *Span    `json:"span,omitempty"`
}

// Span is a line/column span in the analyzed target, as the producer stated
// it. Lines and columns are 1-based, the way the producers state them.
type Span struct {
	StartLine   int `json:"startLine"`
	StartColumn int `json:"startColumn"`
	EndLine     int `json:"endLine"`
	EndColumn   int `json:"endColumn"`
}

// GraphCoverage is the producer's statement about how much of the graph it saw.
type GraphCoverage struct {
	Status           string           `json:"status,omitempty"`
	Truncated        bool             `json:"truncated"`
	OmittedNodes     int              `json:"omittedNodes"`
	OmittedRelations int              `json:"omittedRelations"`
	Limits           []string         `json:"limits,omitempty"`
	Families         []CoverageFamily `json:"families,omitempty"`
}

// CoverageFamily is one relation family's coverage. A family that is partial
// names the unknown IDs it could not resolve, so "partial" is not a blanket
// statement about the whole graph.
type CoverageFamily struct {
	Family            string   `json:"family"`
	Status            string   `json:"status"`
	RetainedRelations int      `json:"retainedRelations"`
	OmittedRelations  int      `json:"omittedRelations"`
	UnknownIDs        []string `json:"unknownIds,omitempty"`
}

// ReadCandidates reads the observations, relations and unknowns one received
// Evidence carries, from its own semantic graph.
//
// The Evidence body is the producer's bytes; the graph is read as the producer
// wrote it. An Evidence that carries no semantic graph is not an error of the
// caller's making and not an empty success either: the caller is told the
// Evidence does not state one, because "no graph" and "a graph with no nodes"
// are different facts about the delivery.
func ReadCandidates(body []byte) (Candidates, error) {
	if len(body) == 0 {
		return Candidates{}, fmt.Errorf("Evidence本体のbytesが空です: 受領したEvidenceをそのまま渡してください")
	}
	_, text, err := decodeText(body)
	if err != nil {
		return Candidates{}, err
	}
	var record evidenceRecord
	if err := json.Unmarshal([]byte(text), &record); err != nil {
		return Candidates{}, fmt.Errorf("Evidence本体がJSON objectではありません: %v", err)
	}
	if _, problems := parseEvidenceRecord(body); len(problems.missing) > 0 || len(problems.invalid) > 0 {
		return Candidates{}, fmt.Errorf("Evidence recordとして不完全です: 受領したbytesをそのまま渡してください")
	}
	candidates := Candidates{
		EvidenceID: rawString(record.EvidenceID),
		Operation:  rawString(record.Operation),
	}
	graph, contexts, ok := decodeSemanticGraph(record.NormalizedResult)
	if !ok {
		return Candidates{}, fmt.Errorf("Evidenceにsemantic graphの宣言がありません: 観測の集合はこのEvidenceから読めません")
	}
	candidates.GraphID = graph.ID
	candidates.Coverage = decodeGraphCoverage(graph.Coverage)
	candidates.Limitations = graph.Limitations
	candidates.Observations = readObservations(graph.Nodes, contexts)
	candidates.Relations = readRelations(candidates.EvidenceID, graph.Relations)
	candidates.Unknowns = readUnknowns(graph.Unknowns)
	return candidates, nil
}

type semanticGraph struct {
	ID               string          `json:"graph_id"`
	Coverage         json.RawMessage `json:"coverage"`
	Limitations      []string        `json:"limitations"`
	Unknowns         []graphUnknown  `json:"unknowns"`
	Nodes            []graphNode     `json:"nodes"`
	Relations        []graphRelation `json:"relations"`
	EvidenceContexts []graphContext  `json:"evidence_contexts"`
}

type graphNode struct {
	NodeID   string          `json:"node_id"`
	Kind     string          `json:"kind"`
	Label    string          `json:"label"`
	Identity json.RawMessage `json:"identity"`
	Evidence graphEvidence   `json:"evidence"`
}

type graphRelation struct {
	RelationID string        `json:"relation_id"`
	From       string        `json:"source_node_id"`
	To         string        `json:"target_node_id"`
	Kind       string        `json:"relation"`
	Resolution string        `json:"resolution"`
	Evidence   graphEvidence `json:"evidence"`
}

type graphUnknown struct {
	UnknownID        string        `json:"unknown_id"`
	NodeID           string        `json:"node_id"`
	Family           string        `json:"family"`
	RelationKinds    []string      `json:"relation_kinds"`
	Reason           string        `json:"reason"`
	Detail           string        `json:"detail"`
	CandidateNodeIDs []string      `json:"candidate_node_ids"`
	Evidence         graphEvidence `json:"evidence"`
}

type graphEvidence struct {
	ContextID string          `json:"context_id"`
	Location  json.RawMessage `json:"location"`
}

// graphContext is the producer's evidence context: the place where state,
// confidence, authority, the extractor and the per-context limits live. Nodes
// and relations reference it by ID, so a candidate that drops the context drops
// the claim's qualification.
type graphContext struct {
	ContextID   string          `json:"context_id"`
	Authority   string          `json:"authority"`
	State       string          `json:"state"`
	Confidence  string          `json:"confidence"`
	Artifact    json.RawMessage `json:"artifact"`
	Extractor   json.RawMessage `json:"extractor"`
	Coverage    json.RawMessage `json:"coverage"`
	Limitations []string        `json:"limitations"`
}

func decodeSemanticGraph(raw json.RawMessage) (semanticGraph, map[string]graphContext, bool) {
	if len(raw) == 0 || string(raw) == "null" {
		return semanticGraph{}, nil, false
	}
	var result struct {
		SemanticGraph json.RawMessage `json:"semantic_graph"`
	}
	if err := json.Unmarshal(raw, &result); err != nil || len(result.SemanticGraph) == 0 {
		return semanticGraph{}, nil, false
	}
	var graph semanticGraph
	if err := json.Unmarshal(result.SemanticGraph, &graph); err != nil {
		return semanticGraph{}, nil, false
	}
	if len(graph.Nodes) == 0 && len(graph.Relations) == 0 {
		return semanticGraph{}, nil, false
	}
	contexts := make(map[string]graphContext, len(graph.EvidenceContexts))
	for _, context := range graph.EvidenceContexts {
		if context.ContextID != "" {
			contexts[context.ContextID] = context
		}
	}
	return graph, contexts, true
}

func readObservations(nodes []graphNode, contexts map[string]graphContext) []Observation {
	out := make([]Observation, 0, len(nodes))
	for _, node := range nodes {
		observation := Observation{
			NodeID:            node.NodeID,
			Kind:              node.Kind,
			Label:             node.Label,
			EvidenceContextID: node.Evidence.ContextID,
		}
		var identity struct {
			ArtifactSHA256 string `json:"artifact_sha256"`
		}
		if err := json.Unmarshal(node.Identity, &identity); err == nil {
			observation.ArtifactSHA256 = identity.ArtifactSHA256
		}
		if context, ok := contexts[node.Evidence.ContextID]; ok {
			observation.State = context.State
			observation.Confidence = context.Confidence
			observation.Authority = context.Authority
			observation.Limitations = context.Limitations
			observation.Extractor = extractorName(context.Extractor)
			if digest := contextArtifactSHA256(context.Artifact); digest != "" {
				observation.ArtifactSHA256 = digest
			}
		}
		location, span, available := decodeExternalLocation(node.Evidence.Location)
		if !available {
			observation.Unavailable = append(observation.Unavailable, "location")
		}
		observation.Location = location
		observation.Span = span
		out = append(out, observation)
	}
	return out
}

func readRelations(evidenceID string, relations []graphRelation) []Relation {
	out := make([]Relation, 0, len(relations))
	for _, relation := range relations {
		edge := Relation{
			// The producer states its own relation identity for this version;
			// it is read, not invented. Only a producer that states none gets
			// a derived handle, and the handle says it was derived.
			RelationID:        relation.RelationID,
			From:              relation.From,
			To:                relation.To,
			Kind:              relation.Kind,
			Resolution:        relation.Resolution,
			EvidenceContextID: relation.Evidence.ContextID,
		}
		if edge.RelationID == "" {
			edge.RelationID = derivedRelationID(evidenceID, relation.From, relation.To, relation.Kind)
			edge.Derived = true
		}
		if location, span, available := decodeExternalLocation(relation.Evidence.Location); available {
			edge.Location = location
			edge.Span = span
		}
		out = append(out, edge)
	}
	return out
}

func readUnknowns(unknowns []graphUnknown) []Unknown {
	out := make([]Unknown, 0, len(unknowns))
	for _, unknown := range unknowns {
		entry := Unknown{
			UnknownID:         unknown.UnknownID,
			NodeID:            unknown.NodeID,
			Family:            unknown.Family,
			RelationKinds:     unknown.RelationKinds,
			Reason:            unknown.Reason,
			Detail:            unknown.Detail,
			CandidateNodeIDs:  unknown.CandidateNodeIDs,
			EvidenceContextID: unknown.Evidence.ContextID,
		}
		if location, span, available := decodeExternalLocation(unknown.Evidence.Location); available {
			entry.Location = location
			entry.Span = span
		}
		out = append(out, entry)
	}
	return out
}

// derivedRelationID is the handle for an edge the producer did not name. It is
// a digest of the Evidence ID and the edge's endpoints and kind, so the same
// edge read twice gets the same handle and a different edge does not.
func derivedRelationID(evidenceID, from, to, kind string) string {
	sum := sha256.Sum256([]byte(strings.Join([]string{evidenceID, from, to, kind}, "\x00")))
	return "sha256:" + hex.EncodeToString(sum[:])
}

func extractorName(raw json.RawMessage) string {
	if len(raw) == 0 || string(raw) == "null" {
		return ""
	}
	var extractor struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	}
	if err := json.Unmarshal(raw, &extractor); err != nil {
		return ""
	}
	if extractor.Version == "" {
		return extractor.Name
	}
	return extractor.Name + "@" + extractor.Version
}

func contextArtifactSHA256(raw json.RawMessage) string {
	if len(raw) == 0 || string(raw) == "null" {
		return ""
	}
	var artifact struct {
		SHA256 string `json:"sha256"`
	}
	if err := json.Unmarshal(raw, &artifact); err != nil {
		return ""
	}
	return artifact.SHA256
}

func decodeGraphCoverage(raw json.RawMessage) *GraphCoverage {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	var document struct {
		Status           string   `json:"status"`
		Truncated        bool     `json:"truncated"`
		OmittedNodes     int      `json:"omitted_nodes"`
		OmittedRelations int      `json:"omitted_relations"`
		Limits           []string `json:"limits"`
		Families         []struct {
			Family            string   `json:"family"`
			Status            string   `json:"status"`
			RetainedRelations int      `json:"retained_relations"`
			OmittedRelations  int      `json:"omitted_relations"`
			UnknownIDs        []string `json:"unknown_ids"`
		} `json:"families"`
	}
	if err := json.Unmarshal(raw, &document); err != nil {
		return nil
	}
	coverage := &GraphCoverage{
		Status:           document.Status,
		Truncated:        document.Truncated,
		OmittedNodes:     document.OmittedNodes,
		OmittedRelations: document.OmittedRelations,
		Limits:           document.Limits,
	}
	for _, family := range document.Families {
		coverage.Families = append(coverage.Families, CoverageFamily{
			Family:            family.Family,
			Status:            family.Status,
			RetainedRelations: family.RetainedRelations,
			OmittedRelations:  family.OmittedRelations,
			UnknownIDs:        family.UnknownIDs,
		})
	}
	return coverage
}

// decodeExternalLocation reads a producer location. A location the producer
// declared unavailable is reported as unavailable with no place name, rather
// than as an empty string that reads like "no location".
func decodeExternalLocation(raw json.RawMessage) (location string, span *Span, available bool) {
	if len(raw) == 0 || string(raw) == "null" {
		return "", nil, false
	}
	var declared struct {
		Available *bool           `json:"available"`
		Value     json.RawMessage `json:"value"`
	}
	if err := json.Unmarshal(raw, &declared); err != nil {
		return "", nil, false
	}
	if declared.Available != nil && !*declared.Available {
		return "", nil, false
	}
	if len(declared.Value) == 0 || string(declared.Value) == "null" {
		return "", nil, false
	}
	var value struct {
		Kind   string `json:"kind"`
		Source string `json:"source"`
		Path   string `json:"path"`
		Start  *struct {
			Line   int `json:"line"`
			Column int `json:"column"`
		} `json:"start"`
		End *struct {
			Line   int `json:"line"`
			Column int `json:"column"`
		} `json:"end"`
	}
	if err := json.Unmarshal(declared.Value, &value); err != nil {
		return "", nil, false
	}
	where := value.Source
	if where == "" {
		where = value.Path
	}
	if value.Kind != "" {
		location = value.Kind + ":" + where
	} else {
		location = where
	}
	if value.Start != nil && value.End != nil {
		span = &Span{
			StartLine:   value.Start.Line,
			StartColumn: value.Start.Column,
			EndLine:     value.End.Line,
			EndColumn:   value.End.Column,
		}
	}
	return location, span, true
}
