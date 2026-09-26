package teild0a

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strings"
	"time"
)

const (
	CorpusSchema = "tei.research.ld0a-proposal-corpus/v0"

	CandidateKindCanonical = "canonical"
	CandidateKindUnknown   = "unknown_or_needs_new_concept"

	DecisionAccepted    = "accepted"
	DecisionRejected    = "rejected"
	DecisionUnknown     = "unknown_or_unresolved"
	DecisionNotProposed = "not_proposed"

	StalenessCurrent = "current_at_decision"
	StalenessStale   = "stale_rebase_required"
	StalenessUnknown = "unknown"

	ConformancePass        = "pass"
	ConformanceFail        = "fail"
	ConformanceUnavailable = "unavailable_not_run"
	ConformanceNA          = "not_applicable"

	CompatibilityCompatible   = "compatible"
	CompatibilityIncompatible = "incompatible"
	CompatibilityUnknown      = "unknown"

	RevisionCompatible = "current"
	RevisionStale      = "stale_or_incompatible"
	RevisionUnknown    = "unknown"

	SplitTrain      = "train"
	SplitValid      = "valid"
	SplitTestA      = "test_a"
	SplitTestB      = "test_b"
	SplitTestC      = "test_c"
	SplitUnassigned = "unassigned"
)

var sha256Revision = regexp.MustCompile("^sha256:[0-9a-f]{64}$")

type Corpus struct {
	Schema string `json:"schema"`
	Rows   []Row  `json:"rows"`
}

type Row struct {
	RowID              string            `json:"row_id"`
	SchemaVersion      string            `json:"schema_version"`
	SourceID           string            `json:"source_id"`
	SourceRevision     string            `json:"source_revision"`
	ApplicationID      string            `json:"application_id"`
	ContextID          string            `json:"context_id"`
	BaseModelRevision  string            `json:"base_model_revision"`
	CandidateSetID     string            `json:"candidate_set_id"`
	CandidateSetSHA256 string            `json:"candidate_set_sha256"`
	CandidateSet       []CandidateRef    `json:"candidate_set"`
	Candidate          CandidateRef      `json:"candidate"`
	ProposalID         string            `json:"proposal_id,omitempty"`
	CollectedAt        string            `json:"collected_at"`
	TemporalGroupID    string            `json:"temporal_group_id"`
	Features           FeatureProjection `json:"features"`
	Outcome            Outcome           `json:"outcome"`
	Split              SplitIdentity     `json:"split"`
}

type CandidateRef struct {
	Kind              string `json:"kind"`
	CanonicalID       string `json:"canonical_id,omitempty"`
	CanonicalRevision string `json:"canonical_revision,omitempty"`
}

type FeatureProjection struct {
	Source    SourceFeatures    `json:"source"`
	Candidate CandidateFeatures `json:"candidate"`
}

type SourceFeatures struct {
	FieldKind           string   `json:"field_kind"`
	Shape               string   `json:"shape"`
	ReferencedSourceIDs []string `json:"referenced_source_ids"`
	EvidenceIDs         []string `json:"evidence_ids"`
	EvidenceCount       int      `json:"evidence_count"`
	RelationCount       int      `json:"relation_count"`
	RevisionAgeSteps    int      `json:"revision_age_steps"`
}

type CandidateFeatures struct {
	CandidateKind          string   `json:"candidate_kind"`
	Compatibility          string   `json:"compatibility"`
	RevisionCompatibility  string   `json:"revision_compatibility"`
	EvidenceAvailable      bool     `json:"evidence_available"`
	ReuseCount             int      `json:"reuse_count"`
	IncompatibilityReasons []string `json:"incompatibility_reasons,omitempty"`
}

type Outcome struct {
	ProposalDecision      string        `json:"proposal_decision"`
	DecisionProvenance    string        `json:"decision_provenance,omitempty"`
	Staleness             string        `json:"staleness"`
	PostAcceptConformance string        `json:"post_accept_conformance"`
	PostAcceptEvidenceRef string        `json:"post_accept_evidence_ref,omitempty"`
	FinalCanonicalMapping *CanonicalRef `json:"final_canonical_mapping,omitempty"`
}

type CanonicalRef struct {
	CanonicalID string `json:"canonical_id"`
	Revision    string `json:"revision"`
}

type SplitIdentity struct {
	Name               string `json:"name"`
	ApplicationPattern string `json:"application_pattern"`
	SourceFamily       string `json:"source_family"`
	RevisionLineage    string `json:"revision_lineage"`
	ProposalEpisode    string `json:"proposal_episode"`
	TimeRegime         string `json:"time_regime"`
}

type CanonicalResolver func(CanonicalRef) bool

type ReuseRankingCase struct {
	SourceID           string         `json:"source_id"`
	SourceRevision     string         `json:"source_revision"`
	ApplicationID      string         `json:"application_id"`
	ContextID          string         `json:"context_id"`
	BaseModelRevision  string         `json:"base_model_revision"`
	ProposalEpisode    string         `json:"proposal_episode"`
	CandidateSetID     string         `json:"candidate_set_id"`
	PositiveCandidate  CanonicalRef   `json:"positive_candidate"`
	NegativeCandidates []CanonicalRef `json:"negative_candidates"`
}

type IncompatibilityExample struct {
	RowID        string `json:"row_id"`
	Incompatible bool   `json:"incompatible"`
}

type EscalationExample struct {
	SourceID          string `json:"source_id"`
	SourceRevision    string `json:"source_revision"`
	ApplicationID     string `json:"application_id"`
	ContextID         string `json:"context_id"`
	BaseModelRevision string `json:"base_model_revision"`
	ProposalEpisode   string `json:"proposal_episode"`
	CandidateSetID    string `json:"candidate_set_id"`
	Escalate          bool   `json:"escalate"`
}

func DecodeCorpus(data []byte) (Corpus, error) {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	var corpus Corpus
	if err := dec.Decode(&corpus); err != nil {
		return Corpus{}, fmt.Errorf("decode corpus: %w", err)
	}
	if _, err := dec.Token(); err != io.EOF {
		return Corpus{}, fmt.Errorf("corpus has trailing JSON content")
	}
	if err := ValidateCorpus(corpus); err != nil {
		return Corpus{}, err
	}
	return corpus, nil
}

func CandidateSetFingerprint(baseModelRevision string, members []CandidateRef) (string, error) {
	if !sha256Revision.MatchString(baseModelRevision) {
		return "", fmt.Errorf("base model revision must be sha256:<64 lowercase hex>")
	}
	normalized, err := normalizeCandidateSet(members)
	if err != nil {
		return "", err
	}
	raw, err := json.Marshal(struct {
		BaseModelRevision string         `json:"base_model_revision"`
		Candidates        []CandidateRef `json:"candidates"`
	}{baseModelRevision, normalized})
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

func ValidateCorpus(corpus Corpus) error {
	if corpus.Schema != CorpusSchema {
		return fmt.Errorf("schema %q must be %q", corpus.Schema, CorpusSchema)
	}
	if len(corpus.Rows) == 0 {
		return fmt.Errorf("corpus must contain rows")
	}

	rowIDs := map[string]bool{}
	setIdentity := map[string]string{}
	setAccepted := map[string]string{}
	groupSplit := map[string]string{}

	for i, row := range corpus.Rows {
		if err := validateRow(row); err != nil {
			return fmt.Errorf("rows[%d] %q: %w", i, row.RowID, err)
		}
		if rowIDs[row.RowID] {
			return fmt.Errorf("duplicate row_id %q", row.RowID)
		}
		rowIDs[row.RowID] = true

		if previous, ok := setIdentity[row.CandidateSetID]; ok && previous != row.CandidateSetSHA256 {
			return fmt.Errorf("candidate_set_id %q maps to multiple hashes", row.CandidateSetID)
		}
		setIdentity[row.CandidateSetID] = row.CandidateSetSHA256

		if row.Outcome.ProposalDecision == DecisionAccepted {
			contextKey := decisionContextKey(row)
			if previous, ok := setAccepted[contextKey]; ok {
				return fmt.Errorf("decision context %q has multiple accepted rows: %s and %s", contextKey, previous, candidateKey(row.Candidate))
			}
			setAccepted[contextKey] = candidateKey(row.Candidate)
		}

		groups := map[string]string{
			"application_pattern": row.Split.ApplicationPattern,
			"source_family":       row.Split.SourceFamily,
			"revision_lineage":    row.Split.RevisionLineage,
			"proposal_episode":    row.Split.ProposalEpisode,
			"time_regime":         row.Split.TimeRegime,
			"temporal_group":      row.TemporalGroupID,
		}
		for kind, value := range groups {
			key := kind + ":" + value
			if previous, ok := groupSplit[key]; ok && previous != row.Split.Name {
				return fmt.Errorf("%s %q crosses split %q -> %q", kind, value, previous, row.Split.Name)
			}
			groupSplit[key] = row.Split.Name
		}
	}
	return nil
}

func ValidateCorpusWithCanonicalResolver(corpus Corpus, resolve CanonicalResolver) error {
	if resolve == nil {
		return fmt.Errorf("canonical resolver is required")
	}
	if err := ValidateCorpus(corpus); err != nil {
		return err
	}
	resolved := map[string]bool{}
	check := func(ref CanonicalRef) error {
		key := canonicalRefKey(ref)
		if ok, seen := resolved[key]; seen {
			if !ok {
				return fmt.Errorf("canonical identity %s cannot be resolved at recorded revision", key)
			}
			return nil
		}
		ok := resolve(ref)
		resolved[key] = ok
		if !ok {
			return fmt.Errorf("canonical identity %s cannot be resolved at recorded revision", key)
		}
		return nil
	}
	for i, row := range corpus.Rows {
		for j, candidate := range row.CandidateSet {
			if candidate.Kind != CandidateKindCanonical {
				continue
			}
			if err := check(CanonicalRef{
				CanonicalID: candidate.CanonicalID,
				Revision:    candidate.CanonicalRevision,
			}); err != nil {
				return fmt.Errorf("rows[%d] candidate_set[%d]: %w", i, j, err)
			}
		}
		if row.Outcome.FinalCanonicalMapping != nil {
			if err := check(*row.Outcome.FinalCanonicalMapping); err != nil {
				return fmt.Errorf("rows[%d] final_canonical_mapping: %w", i, err)
			}
		}
	}
	return nil
}

func validateRow(row Row) error {
	for field, value := range map[string]string{
		"row_id":            row.RowID,
		"schema_version":    row.SchemaVersion,
		"source_id":         row.SourceID,
		"application_id":    row.ApplicationID,
		"context_id":        row.ContextID,
		"candidate_set_id":  row.CandidateSetID,
		"collected_at":      row.CollectedAt,
		"temporal_group_id": row.TemporalGroupID,
	} {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("%s is required", field)
		}
	}
	if row.SchemaVersion != CorpusSchema {
		return fmt.Errorf("schema_version %q must be %q", row.SchemaVersion, CorpusSchema)
	}
	if !sha256Revision.MatchString(row.SourceRevision) {
		return fmt.Errorf("source_revision must be sha256:<64 lowercase hex>")
	}
	if !sha256Revision.MatchString(row.BaseModelRevision) {
		return fmt.Errorf("base_model_revision must be sha256:<64 lowercase hex>")
	}
	if _, err := time.Parse(time.RFC3339, row.CollectedAt); err != nil {
		return fmt.Errorf("collected_at must be RFC3339: %w", err)
	}
	expected, err := CandidateSetFingerprint(row.BaseModelRevision, row.CandidateSet)
	if err != nil {
		return fmt.Errorf("candidate_set: %w", err)
	}
	if row.CandidateSetSHA256 != expected {
		return fmt.Errorf("candidate_set_sha256 mismatch")
	}
	if !candidateInSet(row.Candidate, row.CandidateSet) {
		return fmt.Errorf("candidate is not in candidate_set")
	}
	if err := validateCandidate(row.Candidate); err != nil {
		return fmt.Errorf("candidate: %w", err)
	}
	if row.Features.Candidate.CandidateKind != row.Candidate.Kind {
		return fmt.Errorf("features.candidate.candidate_kind must match candidate.kind")
	}
	if err := validateFeatures(row.Features); err != nil {
		return err
	}
	if err := validateOutcome(row); err != nil {
		return err
	}
	if postAcceptEvidenceRef := strings.TrimSpace(row.Outcome.PostAcceptEvidenceRef); postAcceptEvidenceRef != "" {
		for _, evidenceID := range row.Features.Source.EvidenceIDs {
			if evidenceID == postAcceptEvidenceRef {
				return fmt.Errorf("source feature evidence_ids must not include post_accept_evidence_ref")
			}
		}
	}
	if err := validateSplit(row.Split); err != nil {
		return err
	}
	return nil
}

func validateIdentityList(field string, values []string, required bool) error {
	if required && len(values) == 0 {
		return fmt.Errorf("source feature %s must contain at least one identity", field)
	}
	seen := map[string]bool{}
	for i, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			return fmt.Errorf("source feature %s[%d] must not be blank", field, i)
		}
		if seen[value] {
			return fmt.Errorf("source feature %s contains duplicate identity %q", field, value)
		}
		seen[value] = true
	}
	return nil
}

func validateFeatures(features FeatureProjection) error {
	if strings.TrimSpace(features.Source.FieldKind) == "" || strings.TrimSpace(features.Source.Shape) == "" {
		return fmt.Errorf("source feature field_kind and shape are required")
	}
	if features.Source.EvidenceCount < 0 || features.Source.RelationCount < 0 || features.Source.RevisionAgeSteps < 0 {
		return fmt.Errorf("source feature counts must be non-negative")
	}
	if err := validateIdentityList("referenced_source_ids", features.Source.ReferencedSourceIDs, true); err != nil {
		return err
	}
	if err := validateIdentityList("evidence_ids", features.Source.EvidenceIDs, features.Source.EvidenceCount > 0); err != nil {
		return err
	}
	if len(features.Source.EvidenceIDs) != features.Source.EvidenceCount {
		return fmt.Errorf("source feature evidence_ids count=%d does not match evidence_count=%d", len(features.Source.EvidenceIDs), features.Source.EvidenceCount)
	}
	if features.Candidate.ReuseCount < 0 {
		return fmt.Errorf("candidate reuse_count must be non-negative")
	}
	switch features.Candidate.Compatibility {
	case CompatibilityCompatible, CompatibilityIncompatible, CompatibilityUnknown:
	default:
		return fmt.Errorf("invalid candidate compatibility %q", features.Candidate.Compatibility)
	}
	switch features.Candidate.RevisionCompatibility {
	case RevisionCompatible, RevisionStale, RevisionUnknown:
	default:
		return fmt.Errorf("invalid candidate revision_compatibility %q", features.Candidate.RevisionCompatibility)
	}
	if features.Candidate.Compatibility == CompatibilityIncompatible {
		if !features.Candidate.EvidenceAvailable || len(features.Source.EvidenceIDs) == 0 {
			return fmt.Errorf("deterministic incompatibility requires available referenced evidence")
		}
		if len(features.Candidate.IncompatibilityReasons) == 0 {
			return fmt.Errorf("deterministic incompatibility requires reasons")
		}
	} else if len(features.Candidate.IncompatibilityReasons) != 0 {
		return fmt.Errorf("incompatibility reasons require compatibility=incompatible")
	}
	return nil
}

func validateOutcome(row Row) error {
	switch row.Outcome.ProposalDecision {
	case DecisionAccepted, DecisionRejected:
		if strings.TrimSpace(row.ProposalID) == "" {
			return fmt.Errorf("%s decision requires proposal_id", row.Outcome.ProposalDecision)
		}
		if strings.TrimSpace(row.Outcome.DecisionProvenance) == "" {
			return fmt.Errorf("%s decision requires decision_provenance", row.Outcome.ProposalDecision)
		}
	case DecisionUnknown:
	case DecisionNotProposed:
		if row.ProposalID != "" || row.Outcome.DecisionProvenance != "" {
			return fmt.Errorf("not_proposed row must not invent proposal provenance")
		}
	default:
		return fmt.Errorf("invalid proposal_decision %q", row.Outcome.ProposalDecision)
	}
	switch row.Outcome.Staleness {
	case StalenessCurrent, StalenessStale, StalenessUnknown:
	default:
		return fmt.Errorf("invalid staleness %q", row.Outcome.Staleness)
	}
	switch row.Outcome.PostAcceptConformance {
	case ConformancePass, ConformanceFail:
		if row.Outcome.ProposalDecision != DecisionAccepted {
			return fmt.Errorf("post-Accept conformance cannot exist before accepted decision")
		}
		if strings.TrimSpace(row.Outcome.PostAcceptEvidenceRef) == "" {
			return fmt.Errorf("%s post-Accept conformance requires post_accept_evidence_ref", row.Outcome.PostAcceptConformance)
		}
	case ConformanceUnavailable:
		if row.Outcome.ProposalDecision != DecisionAccepted {
			return fmt.Errorf("post-Accept conformance cannot exist before accepted decision")
		}
		if row.Outcome.PostAcceptEvidenceRef != "" {
			return fmt.Errorf("unavailable post-Accept conformance must not claim post_accept_evidence_ref")
		}
	case ConformanceNA:
		if row.Outcome.ProposalDecision == DecisionAccepted {
			return fmt.Errorf("accepted decision must preserve pass/fail/unavailable post-Accept conformance state")
		}
		if row.Outcome.PostAcceptEvidenceRef != "" {
			return fmt.Errorf("not-applicable post-Accept conformance must not claim post_accept_evidence_ref")
		}
	default:
		return fmt.Errorf("invalid post_accept_conformance %q", row.Outcome.PostAcceptConformance)
	}
	if row.Outcome.FinalCanonicalMapping != nil {
		if row.Outcome.ProposalDecision != DecisionAccepted {
			return fmt.Errorf("final canonical mapping requires accepted decision")
		}
		if strings.TrimSpace(row.Outcome.FinalCanonicalMapping.CanonicalID) == "" ||
			!sha256Revision.MatchString(row.Outcome.FinalCanonicalMapping.Revision) {
			return fmt.Errorf("invalid final canonical mapping")
		}
	}
	return nil
}

func validateSplit(split SplitIdentity) error {
	switch split.Name {
	case SplitTrain, SplitValid, SplitTestA, SplitTestB, SplitTestC, SplitUnassigned:
	default:
		return fmt.Errorf("invalid split %q", split.Name)
	}
	for field, value := range map[string]string{
		"application_pattern": split.ApplicationPattern,
		"source_family":       split.SourceFamily,
		"revision_lineage":    split.RevisionLineage,
		"proposal_episode":    split.ProposalEpisode,
		"time_regime":         split.TimeRegime,
	} {
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("split.%s is required", field)
		}
	}
	return nil
}

func normalizeCandidateSet(members []CandidateRef) ([]CandidateRef, error) {
	if len(members) < 2 {
		return nil, fmt.Errorf("candidate set must contain at least one Canonical candidate plus explicit unknown")
	}
	normalized := append([]CandidateRef(nil), members...)
	seen := map[string]bool{}
	unknownCount := 0
	for i, member := range normalized {
		if err := validateCandidate(member); err != nil {
			return nil, fmt.Errorf("candidate[%d]: %w", i, err)
		}
		key := candidateKey(member)
		if seen[key] {
			return nil, fmt.Errorf("duplicate candidate %q", key)
		}
		seen[key] = true
		if member.Kind == CandidateKindUnknown {
			unknownCount++
		}
	}
	if unknownCount != 1 {
		return nil, fmt.Errorf("candidate set must contain exactly one explicit unknown candidate")
	}
	sort.Slice(normalized, func(i, j int) bool {
		return candidateKey(normalized[i]) < candidateKey(normalized[j])
	})
	return normalized, nil
}

func validateCandidate(candidate CandidateRef) error {
	switch candidate.Kind {
	case CandidateKindCanonical:
		if strings.TrimSpace(candidate.CanonicalID) == "" {
			return fmt.Errorf("canonical candidate requires canonical_id")
		}
		if !sha256Revision.MatchString(candidate.CanonicalRevision) {
			return fmt.Errorf("canonical candidate revision must be sha256:<64 lowercase hex>")
		}
	case CandidateKindUnknown:
		if candidate.CanonicalID != "" || candidate.CanonicalRevision != "" {
			return fmt.Errorf("unknown candidate must not claim Canonical identity/revision")
		}
	default:
		return fmt.Errorf("invalid candidate kind %q", candidate.Kind)
	}
	return nil
}

func candidateKey(candidate CandidateRef) string {
	if candidate.Kind == CandidateKindUnknown {
		return CandidateKindUnknown
	}
	return candidate.Kind + ":" + candidate.CanonicalID + "@" + candidate.CanonicalRevision
}

func candidateInSet(candidate CandidateRef, members []CandidateRef) bool {
	want := candidateKey(candidate)
	for _, member := range members {
		if candidateKey(member) == want {
			return true
		}
	}
	return false
}

func decisionContextKey(row Row) string {
	return strings.Join([]string{
		row.SourceID,
		row.SourceRevision,
		row.ApplicationID,
		row.ContextID,
		row.BaseModelRevision,
		row.CandidateSetID,
		row.Split.ProposalEpisode,
	}, "\x1f")
}

func canonicalRefKey(ref CanonicalRef) string {
	return CandidateKindCanonical + ":" + ref.CanonicalID + "@" + ref.Revision
}

func canonicalRefInSet(ref CanonicalRef, members []CandidateRef) bool {
	want := canonicalRefKey(ref)
	for _, member := range members {
		if member.Kind == CandidateKindCanonical && candidateKey(member) == want {
			return true
		}
	}
	return false
}

func BuildReuseRankingView(corpus Corpus) ([]ReuseRankingCase, error) {
	if err := ValidateCorpus(corpus); err != nil {
		return nil, err
	}
	grouped := map[string][]Row{}
	for _, row := range corpus.Rows {
		key := decisionContextKey(row)
		grouped[key] = append(grouped[key], row)
	}
	keys := make([]string, 0, len(grouped))
	for key := range grouped {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	out := []ReuseRankingCase{}
	for _, key := range keys {
		rows := grouped[key]
		var positive CanonicalRef
		positiveKey := ""
		for _, row := range rows {
			if row.Outcome.ProposalDecision != DecisionAccepted ||
				row.Outcome.Staleness != StalenessCurrent ||
				row.Outcome.FinalCanonicalMapping == nil {
				continue
			}
			mapping := *row.Outcome.FinalCanonicalMapping
			if !canonicalRefInSet(mapping, row.CandidateSet) {
				continue
			}
			mappingKey := canonicalRefKey(mapping)
			if positiveKey != "" && positiveKey != mappingKey {
				return nil, fmt.Errorf("decision context %q has multiple final canonical mappings", key)
			}
			positive = mapping
			positiveKey = mappingKey
		}
		if positiveKey == "" {
			continue
		}

		negativeSet := map[string]CanonicalRef{}
		for _, row := range rows {
			if row.Candidate.Kind != CandidateKindCanonical {
				continue
			}
			candidateRef := CanonicalRef{
				CanonicalID: row.Candidate.CanonicalID,
				Revision:    row.Candidate.CanonicalRevision,
			}
			candidateRefKey := canonicalRefKey(candidateRef)
			if candidateRefKey == positiveKey {
				continue
			}
			if row.Outcome.ProposalDecision == DecisionRejected ||
				row.Features.Candidate.Compatibility == CompatibilityIncompatible {
				negativeSet[candidateRefKey] = candidateRef
			}
		}
		negativeKeys := make([]string, 0, len(negativeSet))
		for negativeKey := range negativeSet {
			negativeKeys = append(negativeKeys, negativeKey)
		}
		sort.Strings(negativeKeys)
		negatives := make([]CanonicalRef, 0, len(negativeKeys))
		for _, negativeKey := range negativeKeys {
			negatives = append(negatives, negativeSet[negativeKey])
		}

		first := rows[0]
		out = append(out, ReuseRankingCase{
			SourceID:           first.SourceID,
			SourceRevision:     first.SourceRevision,
			ApplicationID:      first.ApplicationID,
			ContextID:          first.ContextID,
			BaseModelRevision:  first.BaseModelRevision,
			ProposalEpisode:    first.Split.ProposalEpisode,
			CandidateSetID:     first.CandidateSetID,
			PositiveCandidate:  positive,
			NegativeCandidates: negatives,
		})
	}
	return out, nil
}

func BuildIncompatibilityView(corpus Corpus) ([]IncompatibilityExample, error) {
	if err := ValidateCorpus(corpus); err != nil {
		return nil, err
	}
	out := []IncompatibilityExample{}
	for _, row := range corpus.Rows {
		if row.Candidate.Kind != CandidateKindCanonical {
			continue
		}
		if !row.Features.Candidate.EvidenceAvailable || len(row.Features.Source.EvidenceIDs) == 0 {
			continue
		}
		switch row.Features.Candidate.Compatibility {
		case CompatibilityCompatible:
			out = append(out, IncompatibilityExample{RowID: row.RowID, Incompatible: false})
		case CompatibilityIncompatible:
			out = append(out, IncompatibilityExample{RowID: row.RowID, Incompatible: true})
		}
	}
	return out, nil
}

func BuildEscalationView(corpus Corpus) ([]EscalationExample, error) {
	if err := ValidateCorpus(corpus); err != nil {
		return nil, err
	}
	grouped := map[string][]Row{}
	for _, row := range corpus.Rows {
		key := decisionContextKey(row)
		grouped[key] = append(grouped[key], row)
	}
	keys := make([]string, 0, len(grouped))
	for key := range grouped {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	out := make([]EscalationExample, 0, len(keys))
	for _, key := range keys {
		rows := grouped[key]
		acceptedCurrent := false
		for _, row := range rows {
			if row.Outcome.ProposalDecision == DecisionAccepted &&
				row.Outcome.Staleness == StalenessCurrent &&
				row.Outcome.FinalCanonicalMapping != nil &&
				canonicalRefInSet(*row.Outcome.FinalCanonicalMapping, row.CandidateSet) {
				acceptedCurrent = true
			}
		}
		first := rows[0]
		out = append(out, EscalationExample{
			SourceID:          first.SourceID,
			SourceRevision:    first.SourceRevision,
			ApplicationID:     first.ApplicationID,
			ContextID:         first.ContextID,
			BaseModelRevision: first.BaseModelRevision,
			ProposalEpisode:   first.Split.ProposalEpisode,
			CandidateSetID:    first.CandidateSetID,
			Escalate:          !acceptedCurrent,
		})
	}
	return out, nil
}
