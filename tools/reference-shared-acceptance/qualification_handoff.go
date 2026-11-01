package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"sort"

	"tei/internal/knowledgecontext"
)

func decodeQualificationInput(data []byte) (knowledgecontext.QualificationResult, error) {
	qualification, qualificationErr := decodeQualificationResult(data)
	if qualificationErr == nil {
		return qualification, nil
	}

	// #2354: tei knowledge-qualify output may arrive as the command result
	// envelope. Unwrap it before the bare-shape decoders reject the unknown
	// schemaVersion field.
	if unwrapped, envelopeErr := unwrapQualificationEnvelope(data); envelopeErr == nil {
		result, resultErr := decodeQualificationResult(unwrapped)
		if resultErr == nil {
			return result, nil
		}
	}

	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var handoff knowledgecontext.BuilderReviewHandoff
	if err := decoder.Decode(&handoff); err != nil {
		return knowledgecontext.QualificationResult{}, fmt.Errorf(
			"qualification入力をQualificationResultまたはBuilder Review Handoffとして読めません: qualification=%v handoff=%w",
			qualificationErr,
			err,
		)
	}
	if _, err := decoder.Token(); err != io.EOF {
		return knowledgecontext.QualificationResult{}, fmt.Errorf(
			"Builder Review Handoff JSONの後ろに余分な内容があります",
		)
	}
	if err := knowledgecontext.ValidateBuilderReviewHandoff(handoff); err != nil {
		return knowledgecontext.QualificationResult{}, fmt.Errorf(
			"Builder Review Handoffが不正です: %w",
			err,
		)
	}
	if handoff.Status != "qualified-for-disclosure" {
		return knowledgecontext.QualificationResult{}, fmt.Errorf(
			"Builder Review Handoff status %q はShared Acceptance Evidenceへ接続できません",
			handoff.Status,
		)
	}
	if handoff.Qualification.Status != knowledgecontext.QualificationApplicable {
		return knowledgecontext.QualificationResult{}, fmt.Errorf(
			"Builder Review Handoffのqualification status %q はapplicableではありません",
			handoff.Qualification.Status,
		)
	}
	return handoff.Qualification, nil
}

// unwrapQualificationEnvelope extracts the result of a tei.cli.knowledge-qualify/v0
// envelope. It fails for any other JSON so the caller can keep trying the bare
// shapes.
func unwrapQualificationEnvelope(data []byte) ([]byte, error) {
	var envelope struct {
		SchemaVersion string          `json:"schemaVersion"`
		OK            bool            `json:"ok"`
		Mode          string          `json:"mode,omitempty"`
		Result        json.RawMessage `json:"result,omitempty"`
		Error         *struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error,omitempty"`
	}
	if err := json.Unmarshal(data, &envelope); err != nil {
		return nil, err
	}
	if envelope.SchemaVersion != "tei.cli.knowledge-qualify/v0" {
		return nil, fmt.Errorf("schemaVersion %q is not tei.cli.knowledge-qualify/v0", envelope.SchemaVersion)
	}
	if envelope.Error != nil {
		return nil, fmt.Errorf("tei knowledge-qualify failed: %s: %s", envelope.Error.Code, envelope.Error.Message)
	}
	if envelope.Mode != "" && envelope.Mode != "qualification" {
		return nil, fmt.Errorf("tei knowledge-qualify returned mode %s", envelope.Mode)
	}
	if len(envelope.Result) == 0 {
		return nil, fmt.Errorf("tei knowledge-qualify envelope has no result")
	}
	return envelope.Result, nil
}

func decodeQualificationResult(data []byte) (knowledgecontext.QualificationResult, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var result knowledgecontext.QualificationResult
	if err := decoder.Decode(&result); err != nil {
		return result, fmt.Errorf("qualificationをJSONとして読めません: %w", err)
	}
	if _, err := decoder.Token(); err != io.EOF {
		return result, fmt.Errorf("qualificationのJSONの後ろに余分な内容があります")
	}
	return result, nil
}

func attachQualificationEvidence(candidateData []byte, qualification knowledgecontext.QualificationResult) (Candidate, error) {
	candidate, err := decodeCandidate(candidateData)
	if err != nil {
		return Candidate{}, err
	}
	if qualification.Status != knowledgecontext.QualificationApplicable {
		return Candidate{}, fmt.Errorf("qualification status %q はapplicableではありません", qualification.Status)
	}
	if len(qualification.Evidence) == 0 {
		return Candidate{}, fmt.Errorf("applicable qualificationにexact Evidenceがありません")
	}
	if err := validateQualificationDependencyEvidence(qualification); err != nil {
		return Candidate{}, err
	}

	seen := map[string]struct{}{}
	for _, evidence := range candidate.Evidence {
		seen[evidenceKey(evidence)] = struct{}{}
	}
	for i, evidence := range qualification.Evidence {
		if evidence.SubjectKind != qualification.Candidate.Kind ||
			evidence.Subject != qualification.Candidate.ID {
			return Candidate{}, fmt.Errorf(
				"qualification evidence[%d]のsubject %s:%s がcandidate %s:%s と一致しません",
				i,
				evidence.SubjectKind,
				evidence.Subject,
				qualification.Candidate.Kind,
				qualification.Candidate.ID,
			)
		}
		if evidence.Source == "" || evidence.SourceFingerprint == "" {
			return Candidate{}, fmt.Errorf("qualification evidence[%d]にsourceまたはsourceFingerprintがありません", i)
		}
		requirements := qualification.Requirements
		dependencyEvidence := append([]knowledgecontext.QualificationEvidence(nil), qualification.DependencyEvidence...)
		sort.Slice(dependencyEvidence, func(i, j int) bool {
			return qualificationEvidenceKey(dependencyEvidence[i]) < qualificationEvidenceKey(dependencyEvidence[j])
		})
		item := Evidence{
			SubjectKind:                     evidence.SubjectKind,
			Subject:                         evidence.Subject,
			Source:                          evidence.Source,
			SourceFingerprint:               evidence.SourceFingerprint,
			QualificationRequirements:       &requirements,
			QualificationDependencyEvidence: dependencyEvidence,
		}
		key := evidenceKey(item)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		candidate.Evidence = append(candidate.Evidence, item)
	}
	sort.SliceStable(candidate.Evidence, func(i, j int) bool {
		return evidenceKey(candidate.Evidence[i]) < evidenceKey(candidate.Evidence[j])
	})

	normalized, _, err := normalizeCandidate(candidate)
	if err != nil {
		return Candidate{}, err
	}
	return normalized, nil
}

func evidenceKey(e Evidence) string {
	encoded, _ := json.Marshal(e)
	return string(encoded)
}

func qualificationEvidenceKey(e knowledgecontext.QualificationEvidence) string {
	encoded, _ := json.Marshal(e)
	return string(encoded)
}

func sameQualificationEvidence(left, right []knowledgecontext.QualificationEvidence) bool {
	if len(left) != len(right) {
		return false
	}
	leftKeys := make([]string, len(left))
	rightKeys := make([]string, len(right))
	for i := range left {
		leftKeys[i] = qualificationEvidenceKey(left[i])
	}
	for i := range right {
		rightKeys[i] = qualificationEvidenceKey(right[i])
	}
	sort.Strings(leftKeys)
	sort.Strings(rightKeys)
	for i := range leftKeys {
		if leftKeys[i] != rightKeys[i] {
			return false
		}
	}
	return true
}

func validateQualificationDependencyEvidence(qualification knowledgecontext.QualificationResult) error {
	required := map[string]struct{}{}
	for _, dependency := range qualification.Requirements.RequiredDependencies {
		required[qualificationDependencySubjectKey(dependency.Kind, dependency.ID)] = struct{}{}
	}
	seen := map[string]struct{}{}
	for i, evidence := range qualification.DependencyEvidence {
		key := qualificationDependencySubjectKey(evidence.SubjectKind, evidence.Subject)
		if _, ok := required[key]; !ok {
			return fmt.Errorf(
				"qualification dependencyEvidence[%d]のsubject %s:%s はrequiredDependenciesにありません",
				i,
				evidence.SubjectKind,
				evidence.Subject,
			)
		}
		seen[key] = struct{}{}
	}
	for key := range required {
		if _, ok := seen[key]; !ok {
			return fmt.Errorf("qualification required dependency %q のauthoritative Evidenceがありません", key)
		}
	}
	if len(required) == 0 && len(qualification.DependencyEvidence) != 0 {
		return fmt.Errorf("qualificationにrequiredDependenciesが無いのにdependencyEvidenceがあります")
	}
	return nil
}

func qualificationDependencySubjectKey(kind, id string) string {
	return fmt.Sprintf("%q|%q", kind, id)
}
