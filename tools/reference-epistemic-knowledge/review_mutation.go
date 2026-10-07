package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

const (
	reviewCommandContract = "tei.epistemic-review-command/v0"
	reviewCommandSchema   = "tei.reference.epistemic-review-command/v0"
	reviewResultContract  = "tei.epistemic-review-mutation/v0"
	reviewResultSchema    = "tei.reference.epistemic-review-mutation/v0"
)

type ReviewCommand struct {
	Contract                   string          `json:"contract"`
	Schema                     string          `json:"schema"`
	AssertionID                string          `json:"assertionId"`
	Operation                  ReviewEventKind `json:"operation"`
	Reviewer                   string          `json:"reviewer"`
	EventID                    string          `json:"eventId"`
	OccurredAt                 string          `json:"occurredAt"`
	ExpectedMeaningFingerprint string          `json:"expectedMeaningFingerprint"`
	ExpectedReviewSubjectFingerprint string     `json:"expectedReviewSubjectFingerprint"`
	ExpectedTargetBinding      string          `json:"expectedTargetBinding"`
	ExpectedLastSequence       int             `json:"expectedLastSequence"`
	Evidence                   []EvidenceRef   `json:"evidence,omitempty"`
	Conditions                 []string        `json:"conditions,omitempty"`
	AccessRefs                 []string        `json:"accessRefs,omitempty"`
}

type ReviewMutationResult struct {
	Contract        string     `json:"contract"`
	Schema          string     `json:"schema"`
	AppendedEvent   ReviewEvent `json:"appendedEvent"`
	UpdatedCorpus   Corpus     `json:"updatedCorpus"`
	Projection      Projection `json:"projection"`
	AuthorityLimits []string   `json:"authorityLimits"`
}

func applyReviewCommand(
	rawCorpus Corpus,
	rawContext ContextQuery,
	rawCommand ReviewCommand,
) (ReviewMutationResult, error) {
	corpusCopy, err := cloneCorpus(rawCorpus)
	if err != nil {
		return ReviewMutationResult{}, err
	}
	corpus, err := normalizeCorpus(corpusCopy)
	if err != nil {
		return ReviewMutationResult{}, err
	}
	context, asOf, err := normalizeContext(rawContext)
	if err != nil {
		return ReviewMutationResult{}, err
	}
	command, occurredAt, err := normalizeReviewCommand(rawCommand)
	if err != nil {
		return ReviewMutationResult{}, err
	}
	if occurredAt.After(asOf) {
		return ReviewMutationResult{}, fmt.Errorf(
			"review command occurredAt %s is after context asOf %s",
			command.OccurredAt,
			context.AsOf,
		)
	}

	assertionIndex := -1
	for i := range corpus.Assertions {
		if corpus.Assertions[i].ID == command.AssertionID {
			assertionIndex = i
			break
		}
	}
	if assertionIndex < 0 {
		return ReviewMutationResult{}, fmt.Errorf("review command references unknown assertion %q", command.AssertionID)
	}
	assertion := corpus.Assertions[assertionIndex]

	if assertion.MeaningFingerprint != command.ExpectedMeaningFingerprint {
		return ReviewMutationResult{}, fmt.Errorf("review command meaning binding is stale")
	}
	currentSubjectFingerprint := reviewSubjectFingerprint(assertion)
	if currentSubjectFingerprint != command.ExpectedReviewSubjectFingerprint {
		return ReviewMutationResult{}, fmt.Errorf("review command subject binding is stale")
	}

	currentTargetBinding := targetBinding(assertion.Target, context)
	if currentTargetBinding != command.ExpectedTargetBinding {
		return ReviewMutationResult{}, fmt.Errorf(
			"review command target binding is stale: expected %s current %s",
			command.ExpectedTargetBinding,
			currentTargetBinding,
		)
	}

	lastSequence := 0
	for _, event := range assertion.ReviewEvents {
		if event.Sequence > lastSequence {
			lastSequence = event.Sequence
		}
		if event.ID == command.EventID {
			return ReviewMutationResult{}, fmt.Errorf("duplicate review event id %q", command.EventID)
		}
	}
	if lastSequence != command.ExpectedLastSequence {
		return ReviewMutationResult{}, fmt.Errorf(
			"review command sequence is stale: expected last %d current last %d",
			command.ExpectedLastSequence,
			lastSequence,
		)
	}

	if command.Operation != ReviewEventWithdraw {
		if assertion.MetadataState != MetadataComplete {
			return ReviewMutationResult{}, fmt.Errorf(
				"review command cannot %s assertion with metadataState=%s",
				command.Operation,
				assertion.MetadataState,
			)
		}
		switch currentTargetBinding {
		case "ambiguous", "detached":
			return ReviewMutationResult{}, fmt.Errorf(
				"review command cannot %s assertion with targetBinding=%s",
				command.Operation,
				currentTargetBinding,
			)
		}
		compatibility := contextCompatibility(assertion.Context, context.Bindings)
		if compatibility != "compatible" {
			return ReviewMutationResult{}, fmt.Errorf(
				"review command cannot %s assertion with contextCompatibility=%s",
				command.Operation,
				compatibility,
			)
		}
	}

	event := ReviewEvent{
		Sequence:           lastSequence + 1,
		ID:                 command.EventID,
		Kind:               command.Operation,
		Reviewer:           command.Reviewer,
		Method:             reviewMethodHuman,
		MeaningFingerprint: assertion.MeaningFingerprint,
		OccurredAt:         command.OccurredAt,
		Evidence:           append([]EvidenceRef{}, command.Evidence...),
		Conditions:         append([]string{}, command.Conditions...),
		AccessRefs:         append([]string{}, command.AccessRefs...),
	}
	assertion.ReviewEvents = append(assertion.ReviewEvents, event)
	corpus.Assertions[assertionIndex] = assertion

	updated, err := normalizeCorpus(corpus)
	if err != nil {
		return ReviewMutationResult{}, fmt.Errorf("normalize updated corpus: %w", err)
	}
	projection, err := evaluateCorpus(updated, context)
	if err != nil {
		return ReviewMutationResult{}, fmt.Errorf("evaluate updated corpus: %w", err)
	}

	var appended ReviewEvent
	for _, candidate := range updated.Assertions {
		if candidate.ID != command.AssertionID {
			continue
		}
		for _, candidateEvent := range candidate.ReviewEvents {
			if candidateEvent.ID == command.EventID {
				appended = candidateEvent
				break
			}
		}
	}
	if appended.ID == "" {
		return ReviewMutationResult{}, fmt.Errorf("appended review event %q was not preserved", command.EventID)
	}

	return ReviewMutationResult{
		Contract:      reviewResultContract,
		Schema:        reviewResultSchema,
		AppendedEvent: appended,
		UpdatedCorpus: updated,
		Projection:    projection,
		AuthorityLimits: []string{
			"review-command-does-not-grant-reviewer-authority",
			"review-command-does-not-grant-canonical-acceptance",
			"review-command-does-not-change-canonical-content",
			"meaning-fingerprint-is-binding-input-not-semantic-proof",
			"review-subject-fingerprint-is-concurrency-receipt-not-truth-proof",
			"target-binding-precondition-does-not-prove-semantic-identity",
		},
	}, nil
}

func normalizeReviewCommand(input ReviewCommand) (ReviewCommand, time.Time, error) {
	input.Contract = strings.TrimSpace(input.Contract)
	input.Schema = strings.TrimSpace(input.Schema)
	input.AssertionID = strings.TrimSpace(input.AssertionID)
	input.Reviewer = strings.TrimSpace(input.Reviewer)
	input.EventID = strings.TrimSpace(input.EventID)
	input.OccurredAt = strings.TrimSpace(input.OccurredAt)
	input.ExpectedMeaningFingerprint = strings.TrimSpace(input.ExpectedMeaningFingerprint)
	input.ExpectedReviewSubjectFingerprint = strings.TrimSpace(input.ExpectedReviewSubjectFingerprint)
	input.ExpectedTargetBinding = strings.TrimSpace(input.ExpectedTargetBinding)

	if input.Contract != reviewCommandContract {
		return input, time.Time{}, fmt.Errorf("unsupported review command contract %q", input.Contract)
	}
	if input.Schema != reviewCommandSchema {
		return input, time.Time{}, fmt.Errorf("unsupported review command schema %q", input.Schema)
	}
	if input.AssertionID == "" || input.Reviewer == "" || input.EventID == "" {
		return input, time.Time{}, fmt.Errorf("review command requires assertionId, reviewer, and eventId")
	}
	if !input.Operation.valid() {
		return input, time.Time{}, fmt.Errorf("invalid review command operation %q", input.Operation)
	}
	if !sha256Pattern.MatchString(input.ExpectedMeaningFingerprint) {
		return input, time.Time{}, fmt.Errorf(
			"review command expectedMeaningFingerprint %q is not exact sha256",
			input.ExpectedMeaningFingerprint,
		)
	}
	if !sha256Pattern.MatchString(input.ExpectedReviewSubjectFingerprint) {
		return input, time.Time{}, fmt.Errorf(
			"review command expectedReviewSubjectFingerprint %q is not exact sha256",
			input.ExpectedReviewSubjectFingerprint,
		)
	}
	if !validTargetBinding(input.ExpectedTargetBinding) {
		return input, time.Time{}, fmt.Errorf(
			"review command expectedTargetBinding %q is invalid",
			input.ExpectedTargetBinding,
		)
	}
	if input.ExpectedLastSequence < 0 {
		return input, time.Time{}, fmt.Errorf("review command expectedLastSequence must be >= 0")
	}
	occurredAt, err := time.Parse(time.RFC3339, input.OccurredAt)
	if err != nil {
		return input, time.Time{}, fmt.Errorf("invalid review command occurredAt: %w", err)
	}
	input.OccurredAt = occurredAt.Format(time.RFC3339)
	evidence, err := normalizeEvidenceRefs(input.Evidence)
	if err != nil {
		return input, time.Time{}, fmt.Errorf("review command evidence: %w", err)
	}
	input.Evidence = evidence
	input.Conditions = normalizeStrings(input.Conditions)
	input.AccessRefs = normalizeStrings(input.AccessRefs)
	return input, occurredAt, nil
}

func validTargetBinding(value string) bool {
	switch value {
	case "not-required", "exact", "reanchored", "ambiguous", "detached":
		return true
	default:
		return false
	}
}

func cloneCorpus(input Corpus) (Corpus, error) {
	encoded, err := json.Marshal(input)
	if err != nil {
		return Corpus{}, fmt.Errorf("clone corpus: %w", err)
	}
	var cloned Corpus
	if err := json.Unmarshal(encoded, &cloned); err != nil {
		return Corpus{}, fmt.Errorf("clone corpus: %w", err)
	}
	return cloned, nil
}
