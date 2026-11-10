package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"tei/internal/knowledgecontext"
	"tei/internal/subject"
	"tei/internal/workspace"
)

// buildKnowledgeContextWithIdentity is buildKnowledgeContext plus the Workspace
// identity the read observed (#3455). The plain builder stays for callers that
// only need the Context.
func buildKnowledgeContextWithIdentity(root, subjectKind, subjectID string) (knowledgecontext.Context, *workspace.WorkspaceIdentity, error) {
	if strings.TrimSpace(root) == "" {
		return knowledgecontext.Context{}, nil, newArgumentError("knowledge-context requires -workspace")
	}
	kind := strings.TrimSpace(subjectKind)
	id := strings.TrimSpace(subjectID)
	if kind == "" || id == "" {
		return knowledgecontext.Context{}, nil, newArgumentError("knowledge-context requires -subject-kind and -subject-id")
	}

	loaded, err := workspace.Load(root)
	if err != nil {
		return knowledgecontext.Context{}, nil, err
	}
	sources, err := workspace.BuildSourceIndex(root, loaded)
	if err != nil {
		return knowledgecontext.Context{}, nil, err
	}
	return knowledgecontext.Build(
		loaded.World,
		sources,
		subject.Subject{Kind: kind, ID: id},
	), workspaceIdentityPointer(loaded), nil
}

// buildKnowledgeContext keeps the Context-only signature used by the other
// knowledge builders and the existing tests.
func buildKnowledgeContext(root, subjectKind, subjectID string) (knowledgecontext.Context, error) {
	context, _, err := buildKnowledgeContextWithIdentity(root, subjectKind, subjectID)
	return context, err
}
func buildDataDesignCandidateReview(
	root, subjectKind, subjectID, candidateKind, candidateFingerprint,
	expectedEvidenceRequirementFingerprint string,
) (knowledgecontext.DataDesignCandidateReviewPacket, error) {
	ctx, err := buildKnowledgeContext(root, subjectKind, subjectID)
	if err != nil {
		return knowledgecontext.DataDesignCandidateReviewPacket{}, err
	}
	return knowledgecontext.BuildDataDesignCandidateReview(
		ctx,
		candidateKind,
		candidateFingerprint,
		expectedEvidenceRequirementFingerprint,
	)
}

func buildKnowledgeDisclosure(
	ctx knowledgecontext.Context,
	selection string,
) (knowledgecontext.DisclosureProjection, error) {
	return knowledgecontext.BuildDisclosure(ctx, []string{selection})
}

func buildBuilderReviewPlan(
	root, subjectKind, subjectID string,
) (knowledgecontext.BuilderReviewPlan, error) {
	if strings.TrimSpace(root) == "" {
		return knowledgecontext.BuilderReviewPlan{}, newArgumentError("knowledge-review-plan requires -workspace")
	}
	kind := strings.TrimSpace(subjectKind)
	id := strings.TrimSpace(subjectID)
	if kind == "" || id == "" {
		return knowledgecontext.BuilderReviewPlan{}, newArgumentError("knowledge-review-plan requires -subject-kind and -subject-id")
	}
	ctx, err := buildKnowledgeContext(root, kind, id)
	if err != nil {
		return knowledgecontext.BuilderReviewPlan{}, err
	}
	return knowledgecontext.BuildBuilderReviewPlan(ctx), nil
}

func buildBuilderReviewHandoff(
	root, subjectKind, subjectID string,
	requirements knowledgecontext.QualificationRequirements,
) (knowledgecontext.BuilderReviewHandoff, error) {
	if strings.TrimSpace(root) == "" {
		return knowledgecontext.BuilderReviewHandoff{}, newArgumentError("knowledge-review-handoff requires -workspace")
	}
	kind := strings.TrimSpace(subjectKind)
	id := strings.TrimSpace(subjectID)
	if kind == "" || id == "" {
		return knowledgecontext.BuilderReviewHandoff{}, newArgumentError("knowledge-review-handoff requires -subject-kind and -subject-id")
	}

	loaded, err := workspace.Load(root)
	if err != nil {
		return knowledgecontext.BuilderReviewHandoff{}, err
	}
	sources, err := workspace.BuildSourceIndex(root, loaded)
	if err != nil {
		return knowledgecontext.BuilderReviewHandoff{}, err
	}
	ctx := knowledgecontext.Build(
		loaded.World,
		sources,
		subject.Subject{Kind: kind, ID: id},
	)
	return knowledgecontext.BuildBuilderReviewHandoff(ctx, sources, requirements)
}

func buildBuilderReviewHandoffFromFile(
	root, subjectKind, subjectID, requirementsPath string,
) (knowledgecontext.BuilderReviewHandoff, error) {
	if strings.TrimSpace(requirementsPath) == "" {
		return knowledgecontext.BuilderReviewHandoff{}, newArgumentError("knowledge-review-handoff requires -qualification-requirements")
	}
	var requirements knowledgecontext.QualificationRequirements
	if err := decodeStrictKnowledgeJSONFile("-qualification-requirements", requirementsPath, &requirements); err != nil {
		return knowledgecontext.BuilderReviewHandoff{}, err
	}
	return buildBuilderReviewHandoff(root, subjectKind, subjectID, requirements)
}

func buildBuilderReviewDisclosureFromFile(
	root, subjectKind, subjectID, handoffPath string,
) (
	knowledgecontext.DisclosureProjection,
	knowledgecontext.BuilderReviewHandoff,
	error,
) {
	if strings.TrimSpace(root) == "" {
		return knowledgecontext.DisclosureProjection{},
			knowledgecontext.BuilderReviewHandoff{},
			newArgumentError("knowledge-review-disclose requires -workspace")
	}
	kind := strings.TrimSpace(subjectKind)
	id := strings.TrimSpace(subjectID)
	if kind == "" || id == "" {
		return knowledgecontext.DisclosureProjection{},
			knowledgecontext.BuilderReviewHandoff{},
			newArgumentError("knowledge-review-disclose requires -subject-kind and -subject-id")
	}
	if strings.TrimSpace(handoffPath) == "" {
		return knowledgecontext.DisclosureProjection{},
			knowledgecontext.BuilderReviewHandoff{},
			newArgumentError("knowledge-review-disclose requires -review-handoff")
	}

	var handoff knowledgecontext.BuilderReviewHandoff
	if err := decodeStrictKnowledgeJSONFile("-review-handoff", handoffPath, &handoff); err != nil {
		return knowledgecontext.DisclosureProjection{},
			knowledgecontext.BuilderReviewHandoff{},
			err
	}

	loaded, err := workspace.Load(root)
	if err != nil {
		return knowledgecontext.DisclosureProjection{},
			knowledgecontext.BuilderReviewHandoff{},
			err
	}
	sources, err := workspace.BuildSourceIndex(root, loaded)
	if err != nil {
		return knowledgecontext.DisclosureProjection{},
			knowledgecontext.BuilderReviewHandoff{},
			err
	}
	ctx := knowledgecontext.Build(
		loaded.World,
		sources,
		subject.Subject{Kind: kind, ID: id},
	)
	disclosure, err := knowledgecontext.BuildBuilderReviewDisclosure(
		ctx,
		sources,
		handoff,
	)
	if err != nil {
		return knowledgecontext.DisclosureProjection{}, handoff, err
	}
	return disclosure, handoff, nil
}

func checkDataDesignCandidateReviewFingerprint(
	packet knowledgecontext.DataDesignCandidateReviewPacket,
	expected string,
) error {
	expected = strings.TrimSpace(expected)
	if expected == "" {
		return nil
	}
	if packet.Fingerprint != expected {
		return fmt.Errorf(
			"Data Design candidate review fingerprint mismatch: expected=%s current=%s",
			expected,
			packet.Fingerprint,
		)
	}
	return nil
}

func checkBuilderReviewHandoffFingerprint(
	handoff knowledgecontext.BuilderReviewHandoff,
	expected string,
) error {
	expected = strings.TrimSpace(expected)
	if expected == "" {
		return nil
	}
	if handoff.Fingerprint != expected {
		return fmt.Errorf(
			"Builder Review Handoff fingerprint mismatch: expected=%s current=%s",
			expected,
			handoff.Fingerprint,
		)
	}
	return nil
}

func checkBuilderReviewPlanFingerprint(
	plan knowledgecontext.BuilderReviewPlan,
	expected string,
) error {
	expected = strings.TrimSpace(expected)
	if expected == "" {
		return nil
	}
	if plan.Fingerprint != expected {
		return fmt.Errorf(
			"Builder Review Plan fingerprint mismatch: expected=%s current=%s",
			expected,
			plan.Fingerprint,
		)
	}
	return nil
}

func checkKnowledgeDisclosureFingerprint(
	disclosure knowledgecontext.DisclosureProjection,
	expected string,
) error {
	expected = strings.TrimSpace(expected)
	if expected == "" {
		return nil
	}
	if disclosure.Status != "complete" {
		return fmt.Errorf(
			"Knowledge Disclosure status is %q; cached disclosure reuse requires complete selected Knowledge",
			disclosure.Status,
		)
	}
	if disclosure.Fingerprint != expected {
		return fmt.Errorf(
			"Knowledge Disclosure fingerprint mismatch: expected=%s current=%s",
			expected,
			disclosure.Fingerprint,
		)
	}
	return nil
}

func checkDataDesignFingerprint(ctx knowledgecontext.Context, expected string) error {
	expected = strings.TrimSpace(expected)
	if expected == "" {
		return nil
	}
	if ctx.DataDesign == nil {
		return fmt.Errorf("expected Data Design fingerprint %q, but this subject has no Data Design Context", expected)
	}
	if ctx.DataDesign.ResolutionStatus != "resolved" {
		return fmt.Errorf(
			"Data Design Context resolution status is %q; cached Context reuse requires structurally resolved identity",
			ctx.DataDesign.ResolutionStatus,
		)
	}
	if ctx.DataDesign.EvidenceCoverageStatus != "complete" {
		return fmt.Errorf(
			"Data Design Context evidence coverage is %q; cached Context reuse requires complete Evidence coverage",
			ctx.DataDesign.EvidenceCoverageStatus,
		)
	}
	if ctx.DataDesign.Fingerprint != expected {
		return fmt.Errorf(
			"Data Design Context fingerprint mismatch: expected=%s current=%s",
			expected,
			ctx.DataDesign.Fingerprint,
		)
	}
	return nil
}

func buildKnowledgeQualification(root, subjectKind, subjectID string, requirements knowledgecontext.QualificationRequirements) (knowledgecontext.QualificationResult, error) {
	if strings.TrimSpace(root) == "" {
		return knowledgecontext.QualificationResult{}, newArgumentError("knowledge-qualify requires -workspace")
	}
	kind := strings.TrimSpace(subjectKind)
	id := strings.TrimSpace(subjectID)
	if kind == "" || id == "" {
		return knowledgecontext.QualificationResult{}, newArgumentError("knowledge-qualify requires -subject-kind and -subject-id")
	}
	loaded, err := workspace.Load(root)
	if err != nil {
		return knowledgecontext.QualificationResult{}, err
	}
	sources, err := workspace.BuildSourceIndex(root, loaded)
	if err != nil {
		return knowledgecontext.QualificationResult{}, err
	}
	ctx := knowledgecontext.Build(
		loaded.World,
		sources,
		subject.Subject{Kind: kind, ID: id},
	)
	return knowledgecontext.QualifyWithSources(ctx, sources, requirements)
}

func buildKnowledgeQualificationFromFile(root, subjectKind, subjectID, requirementsPath string) (knowledgecontext.QualificationResult, error) {
	if strings.TrimSpace(requirementsPath) == "" {
		return knowledgecontext.QualificationResult{}, newArgumentError("knowledge-qualify requires -qualification-requirements")
	}
	var requirements knowledgecontext.QualificationRequirements
	if err := decodeStrictKnowledgeJSONFile("-qualification-requirements", requirementsPath, &requirements); err != nil {
		return knowledgecontext.QualificationResult{}, err
	}
	return buildKnowledgeQualification(root, subjectKind, subjectID, requirements)
}

// decodeStrictKnowledgeJSONFile decodes one JSON document from a path the
// caller named. A path that cannot be opened, or that holds something other
// than the expected document, is an argument problem: the caller passed the
// wrong value, not a Workspace whose knowledge failed to build. Reporting it
// as KNOWLEDGE_BUILD_FAILED made a mistyped path look like a broken model
// (#3943), so every failure here is an argument error.
//
// flag is the name of the option the path came from. One loader serves several
// options — `-qualification-requirements` and `-review-handoff` — so naming one
// of them inside the message pointed the caller at a flag they had not used.
func decodeStrictKnowledgeJSONFile(flag, path string, dst any) error {
	f, err := os.Open(path)
	if err != nil {
		return newArgumentError(flag + " must name a readable JSON file: " + err.Error())
	}
	defer f.Close()

	decoder := json.NewDecoder(f)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		return newArgumentError(flag + " is not the expected JSON document: " + err.Error())
	}

	// A second document is a malformed input, not a partially usable file: the
	// command cannot tell which of the two the caller meant, so it must refuse
	// rather than pick the first. Leaving this path on a plain error answered
	// KNOWLEDGE_BUILD_FAILED, which read as a defect in TEI (#3943).
	var trailing any
	err = decoder.Decode(&trailing)
	if err == io.EOF {
		return nil
	}
	if err == nil {
		return newArgumentError(flag + " must hold one JSON document: " + path + " contains trailing JSON value")
	}
	return newArgumentError(flag + " is not the expected JSON document: " + err.Error())
}
