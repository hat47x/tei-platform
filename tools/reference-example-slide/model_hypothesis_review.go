package exampleslide

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
)

const modelHypothesesSchema = "tei.reference.example-model-hypotheses/v0"

// ModelHypotheses is a set of proposals to examine, not accepted semantic
// declarations. The supplied IDs name proposals, never physical PK constraints.
type ModelHypotheses struct {
	Schema                 string                   `json:"schema"`
	CandidateKeys          []CandidateKeyHypothesis `json:"candidateKeys,omitempty"`
	FunctionalDependencies []FDHypothesis           `json:"functionalDependencies,omitempty"`
}

// ModelHypothesisIssue ties one read-only result to the actual diagram
// placements and Canonical attribute IDs supplied in the Catalog. Base-only
// evidence is never attributed to a later slide or to accepted Source.
type ModelHypothesisIssue struct {
	ID                  string         `json:"id"`
	Kind                string         `json:"kind"`
	HypothesisID        string         `json:"hypothesisId"`
	Table               string         `json:"table"`
	Stage               string         `json:"stage"`
	FieldIDs            []string       `json:"fieldIds"`
	DiagramPlacementIDs []string       `json:"diagramPlacementIds,omitempty"`
	FrameIDs            []string       `json:"frameIds,omitempty"`
	Result              string         `json:"result"`
	Reason              string         `json:"reason"`
	RecordPairs         []FDRecordPair `json:"recordPairs,omitempty"`
	UnusableRecordIDs   []string       `json:"unusableRecordIds,omitempty"`
	IndeterminatePairs  int            `json:"indeterminatePairs"`
	Question            string         `json:"question"`
	EvidenceNeeded      []string       `json:"evidenceNeeded"`
	ReviewTarget        string         `json:"reviewTarget"`
}

// ModelHypothesisReview is a separate unverified packet. Existing Review and
// AnswerHandoff fingerprints are intentionally unchanged by this experiment.
type ModelHypothesisReview struct {
	Schema                string                 `json:"schema"`
	CaseID                string                 `json:"caseId"`
	ModelRevision         string                 `json:"modelRevision"`
	ExampleFingerprint    string                 `json:"exampleFingerprint"`
	CatalogFingerprint    string                 `json:"catalogFingerprint"`
	HypothesesFingerprint string                 `json:"hypothesesFingerprint"`
	Fingerprint           string                 `json:"fingerprint"`
	Issues                []ModelHypothesisIssue `json:"issues"`
}

// ReviewModelHypotheses validates the source presentation and both classes of
// proposed rules, then projects base-snapshot findings into the existing ER
// diagram identities. It does not infer a key, issue a Patch or accept a rule.
func ReviewModelHypotheses(input Input, proposed ModelHypotheses) (ModelHypothesisReview, error) {
	var empty ModelHypothesisReview
	if proposed.Schema != modelHypothesesSchema {
		return empty, fmt.Errorf("unsupported model hypotheses schema %q", proposed.Schema)
	}
	if len(proposed.CandidateKeys) == 0 && len(proposed.FunctionalDependencies) == 0 {
		return empty, fmt.Errorf("at least one model hypothesis required")
	}
	if _, err := Build(input); err != nil {
		return empty, err
	}
	keys, err := ProbeBaseCandidateKeys(input.Catalog, input.Example, proposed.CandidateKeys)
	if err != nil {
		return empty, fmt.Errorf("candidate key hypotheses: %w", err)
	}
	fds, err := ProbeBaseFunctionalDependencies(input.Catalog, input.Example, proposed.FunctionalDependencies)
	if err != nil {
		return empty, fmt.Errorf("functional dependency hypotheses: %w", err)
	}

	// Normalize unordered declarations on copies: changing the order of
	// key components or hypotheses must not invalidate the review.
	normalized := ModelHypotheses{Schema: proposed.Schema}
	for _, h := range proposed.CandidateKeys {
		h.Fields = append([]string(nil), h.Fields...)
		sort.Strings(h.Fields)
		normalized.CandidateKeys = append(normalized.CandidateKeys, h)
	}
	for _, h := range proposed.FunctionalDependencies {
		h.Determinants = append([]string(nil), h.Determinants...)
		h.Dependents = append([]string(nil), h.Dependents...)
		sort.Strings(h.Determinants)
		sort.Strings(h.Dependents)
		normalized.FunctionalDependencies = append(normalized.FunctionalDependencies, h)
	}
	sort.Slice(normalized.CandidateKeys, func(i, j int) bool {
		return normalized.CandidateKeys[i].ID < normalized.CandidateKeys[j].ID
	})
	sort.Slice(normalized.FunctionalDependencies, func(i, j int) bool {
		return normalized.FunctionalDependencies[i].ID < normalized.FunctionalDependencies[j].ID
	})

	packet := ModelHypothesisReview{
		Schema:                "tei.reference.example-model-hypothesis-review/v0",
		CaseID:                input.Example.ID,
		ModelRevision:         input.Catalog.Revision,
		ExampleFingerprint:    digestHypothesisJSON(input.Example),
		CatalogFingerprint:    digestHypothesisJSON(input.Catalog),
		HypothesesFingerprint: digestHypothesisJSON(normalized),
		Issues:                make([]ModelHypothesisIssue, 0, len(keys)+len(fds)),
	}
	placements := map[string][]string{}
	for _, placement := range input.Diagram.Placements {
		placements[placement.Table] = append(placements[placement.Table], placement.ID)
	}
	for table := range placements {
		sort.Strings(placements[table])
	}
	baseFrames := make([]string, 0)
	for _, frame := range input.Presentation.Frames {
		if frame.Stage == "base" {
			baseFrames = append(baseFrames, frame.ID)
		}
	}
	sort.Strings(baseFrames)
	newIssue := func(kind, id, table, result, reason string, fields []string) ModelHypothesisIssue {
		// Bind issue identity to both immutable input sources and proposal.
		// Layout-only changes do not alter the identity of a model question.
		identity := digestHypothesisJSON([]string{
			packet.CaseID, packet.ModelRevision, packet.ExampleFingerprint,
			packet.CatalogFingerprint, packet.HypothesesFingerprint,
			kind, id, table, reason,
		})
		return ModelHypothesisIssue{
			ID:   "model-review-" + identity[len("sha256:"):len("sha256:")+24],
			Kind: kind, HypothesisID: id, Table: table, Stage: "base",
			FieldIDs:            append([]string(nil), fields...),
			DiagramPlacementIDs: append([]string(nil), placements[table]...),
			FrameIDs:            append([]string(nil), baseFrames...),
			Result:              result, Reason: reason,
			ReviewTarget: "business-identity-proposal",
			EvidenceNeeded: []string{
				"independent business identity or functional dependency evidence",
				"current Source Index and authority review",
			},
		}
	}
	keyDefs := make(map[string]CandidateKeyHypothesis, len(normalized.CandidateKeys))
	for _, h := range normalized.CandidateKeys {
		keyDefs[h.ID] = h
	}
	for _, finding := range keys {
		h := keyDefs[finding.HypothesisID]
		issue := newIssue("candidate-key", h.ID, h.Table, finding.Result, finding.Reason, h.Fields)
		issue.RecordPairs = append([]FDRecordPair(nil), finding.Collisions...)
		issue.UnusableRecordIDs = append([]string(nil), finding.UnusableRecordIDs...)
		issue.IndeterminatePairs = finding.IndeterminatePairs
		issue.Question = "この項目の組は業務上の識別子ですか。具体例の重複・不明値と、物理行IDの違いを確認してください。"
		packet.Issues = append(packet.Issues, issue)
	}
	fdDefs := make(map[string]FDHypothesis, len(normalized.FunctionalDependencies))
	for _, h := range normalized.FunctionalDependencies {
		fdDefs[h.ID] = h
	}
	for _, finding := range fds {
		h := fdDefs[finding.HypothesisID]
		fields := append(append([]string(nil), h.Determinants...), h.Dependents...)
		issue := newIssue("functional-dependency", h.ID, h.Table, finding.Result, finding.Reason, fields)
		issue.RecordPairs = append([]FDRecordPair(nil), finding.Counterexamples...)
		issue.IndeterminatePairs = finding.IndeterminatePairs
		issue.Question = "この決定項目から従属項目が一つに決まる業務上の根拠はありますか。具体例との矛盾を確認してください。"
		packet.Issues = append(packet.Issues, issue)
	}
	sort.Slice(packet.Issues, func(i, j int) bool {
		a, b := packet.Issues[i], packet.Issues[j]
		if a.Kind != b.Kind {
			return a.Kind < b.Kind
		}
		return a.HypothesisID < b.HypothesisID
	})
	packet.Fingerprint = digestHypothesisJSON(packet)
	return packet, nil
}

func digestHypothesisJSON(value any) string {
	raw, _ := json.Marshal(value)
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:])
}
