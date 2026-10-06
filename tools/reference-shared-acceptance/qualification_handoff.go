package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"sort"

	"tei/internal/knowledgecontext"
)

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
