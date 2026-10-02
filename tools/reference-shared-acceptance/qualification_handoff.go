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
		item := Evidence{
			SubjectKind:       evidence.SubjectKind,
			Subject:           evidence.Subject,
			Source:            evidence.Source,
			SourceFingerprint: evidence.SourceFingerprint,
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
	encoded, _ := json.Marshal(struct {
		SubjectKind       string
		Subject           string
		Source            string
		SourceFingerprint string
		Note              string
	}{
		SubjectKind:       e.SubjectKind,
		Subject:           e.Subject,
		Source:            e.Source,
		SourceFingerprint: e.SourceFingerprint,
		Note:              e.Note,
	})
	return string(encoded)
}
