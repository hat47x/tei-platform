// Package rmtlink records a candidate-only association between a source
// observation and an intended TEI semantic change. It is not an importer,
// model translation, human Decision, source verifier, or runtime authority.
package rmtlink

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"regexp"

	"tei/tools/internal/rmtevidence"
)

const Schema = "tei.reference.rmt-semantic-link-candidate/v0"

const (
	StatusProposed = "proposed"
	MeaningUnresolved = "unresolved"
)

type SemanticTarget struct {
	Kind string `json:"kind"`
	ID string `json:"id"`
	Change string `json:"change"`
}

type Link struct {
	Schema string `json:"schema"`
	Status string `json:"status"`
	MeaningStatus string `json:"meaningStatus"`
	SourceArtifactDigest string `json:"sourceArtifactDigest"`
	SourceToken string `json:"sourceToken"`
	SourceEvidenceFingerprint string `json:"sourceEvidenceFingerprint"`
	MappingContractFingerprint string `json:"mappingContractFingerprint"`
	CandidateFingerprint string `json:"candidateFingerprint"`
	BaseModelRevision string `json:"baseModelRevision"`
	ReviewFingerprint string `json:"reviewFingerprint"`
	Target SemanticTarget `json:"target"`
	AuthorityLimits []string `json:"authorityLimits"`
	LinkFingerprint string `json:"linkFingerprint"`
}

type Context struct {
	Source rmtevidence.Evidence
	CandidateFingerprint string
	BaseModelRevision string
	ReviewFingerprint string
	Target SemanticTarget
}

var digestPattern = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)
var semanticIDPattern = regexp.MustCompile(`^[a-z][a-z0-9_-]*(?:\.[a-z][a-z0-9_-]*)+$`)

var limits = []string{
	"source-observation-is-not-semantic-authority",
	"target-correlation-is-a-proposal-not-source-derived-meaning",
	"review-fingerprint-is-not-human-approval",
	"typed-evidence-fingerprint-is-not-source-observer-conformance",
	"link-cannot-author-acceptance-or-runtime-claims",
}

// Prepare creates a deterministic, source-token-scoped association. The caller
// must supply a concrete semantic change observed in the actual Review diff;
// the link only checks the binding, not that the interpretation is correct.
func Prepare(ctx Context) (Link, error) {
	source, err := rmtevidence.Verify(ctx.Source)
	if err != nil {
		return Link{}, fmt.Errorf("RMT source Evidence: %w", err)
	}
	if !digestPattern.MatchString(ctx.CandidateFingerprint) ||
		!digestPattern.MatchString(ctx.BaseModelRevision) ||
		!digestPattern.MatchString(ctx.ReviewFingerprint) {
		return Link{}, fmt.Errorf("candidate, base model and Review require exact sha256 fingerprints")
	}
	if !semanticIDPattern.MatchString(ctx.Target.ID) ||
		ctx.Target.Kind != "validation" || ctx.Target.Change != "added" {
		return Link{}, fmt.Errorf("v0 RMT link only supports an explicit added Validation target")
	}
	// A mapped source construct can still be semantically unresolved. No
	// classification or extension grants equivalence or Acceptance.
	value := Link{
		Schema: Schema,
		Status: StatusProposed,
		MeaningStatus: MeaningUnresolved,
		SourceArtifactDigest: source.SourceArtifactDigest,
		SourceToken: source.SourceToken,
		SourceEvidenceFingerprint: source.EvidenceFingerprint,
		MappingContractFingerprint: source.MappingContractFingerprint,
		CandidateFingerprint: ctx.CandidateFingerprint,
		BaseModelRevision: ctx.BaseModelRevision,
		ReviewFingerprint: ctx.ReviewFingerprint,
		Target: ctx.Target,
		AuthorityLimits: append([]string(nil), limits...),
	}
	value.LinkFingerprint = fingerprint(value)
	return value, nil
}

// Verify refuses stale/retargeted untrusted links. The Review and candidate
// fingerprints are supplied by independent current evaluation, not read back
// from the link. Source replay itself remains a separate required check.
func Verify(link Link, ctx Context) error {
	expected, err := Prepare(ctx)
	if err != nil {
		return err
	}
	if !digestPattern.MatchString(link.LinkFingerprint) {
		return fmt.Errorf("invalid linkFingerprint")
	}
	if link.LinkFingerprint != fingerprint(link) {
		return fmt.Errorf("link has been modified")
	}
	if link.LinkFingerprint != expected.LinkFingerprint {
		return fmt.Errorf("link does not belong to current exact source Evidence / candidate / Review / target")
	}
	return nil
}

// Decode rejects unknown fields, trailing input and reconstructed fingerprints.
// It cannot replace a fresh source replay or human acceptance verification.
func Decode(data []byte, ctx Context) (Link, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var proposal Link
	if err := decoder.Decode(&proposal); err != nil {
		return Link{}, fmt.Errorf("decode RMT semantic link: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return Link{}, fmt.Errorf("RMT semantic link contains trailing content")
	}
	if err := Verify(proposal, ctx); err != nil {
		return Link{}, err
	}
	return proposal, nil
}

func fingerprint(value Link) string {
	value.LinkFingerprint = ""
	encoded, _ := json.Marshal(value)
	sum := sha256.Sum256(encoded)
	return "sha256:" + hex.EncodeToString(sum[:])
}
