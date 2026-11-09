package languagereference

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"

	"tei/internal/core"
	"tei/internal/ir"
	"tei/internal/semantic"
)

var ErrExpandMeaningDrift = errors.New("bound expand meaning differs from its reference fingerprint")

// BoundExpandMeaning is an inspectable, unaccepted typed semantic candidate.
// SourceSpan is provenance rather than part of the meaning fingerprint.
// Fingerprint is a deterministic change detector, NOT an acceptance seal,
// a cryptographic authority, or a replacement for Semantic Set Lock.
type BoundExpandMeaning struct {
	Expansion   RelationExpansionCandidate
	Policy      RelationMatchPolicy
	SourceSpan  PipelineSpan
	Fingerprint string
}

// expandMeaningEnvelope is intentionally versioned and map-free. Encoding its
// ordered struct fields yields deterministic bytes independent of source
// whitespace, argument order, location, or provider catalog order.
// If its semantic schema changes, the schema version must also change.
type expandMeaningEnvelope struct {
	Schema      string
	SourceID    core.CanonicalID
	RelationID  core.CanonicalID
	TargetID    core.CanonicalID
	Cardinality semantic.Cardinality
	Keys        []semantic.RelationKey
	Operator    OperatorContract
	OutputType  string
	Unmatched   UnmatchedPolicy
	NullKey     NullKeyPolicy
}

// LowerPinnedExpandMeaning binds reference Language syntax to an existing
// canonical Relation and an externally supplied, exact-pinned operator.
// It does not validate Semantic Set Lock, issue identity, or execute a join.
func LowerPinnedExpandMeaning(w ir.World, sourceID core.CanonicalID, syntax ExpandInvocationSyntax, call PinnedCall, catalog []OperatorContract) (BoundExpandMeaning, error) {
	if err := syntax.Policy.Validate(); err != nil {
		return BoundExpandMeaning{}, &PipelineDiagnostic{Span: syntax.CallSpan, Step: 0, Cause: err}
	}
	if syntax.RelationID == "" {
		return BoundExpandMeaning{}, &PipelineDiagnostic{Span: syntax.CallSpan, Step: 0, Cause: fmt.Errorf("expand relation is required")}
	}
	expansion, err := InspectPinnedRelationExpansion(w, sourceID, syntax.RelationID, call, catalog)
	if err != nil {
		return BoundExpandMeaning{}, &PipelineDiagnostic{Span: syntax.RelationSpan, Step: 0, Cause: err}
	}
	fingerprint, err := fingerprintExpandMeaning(expansion, syntax.Policy)
	if err != nil {
		return BoundExpandMeaning{}, &PipelineDiagnostic{Span: syntax.CallSpan, Step: 0, Cause: err}
	}
	return BoundExpandMeaning{
		Expansion:   expansion,
		Policy:      syntax.Policy,
		SourceSpan:  syntax.CallSpan,
		Fingerprint: fingerprint,
	}, nil
}

// VerifyBoundExpandMeaning detects accidental modification to the detached
// candidate. This digest is not trusted input authentication; the caller
// must separately revalidate the externally accepted Semantic Set.
func VerifyBoundExpandMeaning(meaning BoundExpandMeaning) error {
	expected, err := fingerprintExpandMeaning(meaning.Expansion, meaning.Policy)
	if err != nil {
		return err
	}
	if meaning.Fingerprint == "" || expected != meaning.Fingerprint {
		return ErrExpandMeaningDrift
	}
	return nil
}

// ProbeBoundExpandMeaning uses only the policy carried by the bound meaning.
// No caller-supplied override can silently change NULL or unmatched behavior.
func ProbeBoundExpandMeaning(meaning BoundExpandMeaning, source []ProbeKeyPart, targets [][]ProbeKeyPart) (RelationMatchProbeResult, error) {
	if err := VerifyBoundExpandMeaning(meaning); err != nil {
		return RelationMatchProbeResult{}, err
	}
	return ProbePinnedExpansion(meaning.Expansion, meaning.Policy, source, targets)
}

func fingerprintExpandMeaning(expansion RelationExpansionCandidate, policy RelationMatchPolicy) (string, error) {
	if err := policy.Validate(); err != nil {
		return "", err
	}
	if expansion.SourceID == "" || expansion.RelationID == "" || expansion.TargetID == "" ||
		len(expansion.Keys) == 0 || !expansion.Cardinality.Valid() ||
		expansion.Operator.ID != "std.op.expand" || expansion.Operator.Version == "" ||
		expansion.Operator.Fingerprint == "" || expansion.Operator.Effect != "pure" ||
		!expandTypesMatch(expansion.Operator, expansion.SourceID, expansion.TargetID) ||
		expansion.OutputType != "rows/"+string(expansion.TargetID) {
		return "", fmt.Errorf("%w: incomplete expansion candidate", ErrOperatorInvalid)
	}
	payload := expandMeaningEnvelope{
		Schema:      "reference.expand-meaning/v1",
		SourceID:    expansion.SourceID,
		RelationID:  expansion.RelationID,
		TargetID:    expansion.TargetID,
		Cardinality: expansion.Cardinality,
		Keys:        expansion.Keys,
		Operator:    expansion.Operator,
		OutputType:  expansion.OutputType,
		Unmatched:   policy.Unmatched,
		NullKey:     policy.NullKey,
	}
	bytes, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("encode expand meaning: %w", err)
	}
	digest := sha256.Sum256(bytes)
	return hex.EncodeToString(digest[:]), nil
}
