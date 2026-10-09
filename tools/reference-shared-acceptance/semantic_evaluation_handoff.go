package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"tei/internal/sourceindex"
	"tei/tools/internal/rmtevidence"
	semanticevidence "tei/tools/internal/semanticevidence"
)

const (
	evidenceArtifactKindSemanticEvaluation = "semantic-evaluation"
	evidenceArtifactKindRMTTranscription   = "rmt-transcription"
	semanticEvaluationEvidenceSchema       = semanticevidence.Schema
)

type EvidenceArtifact struct {
	Kind                string          `json:"kind"`
	Schema              string          `json:"schema"`
	ArtifactFingerprint string          `json:"artifactFingerprint"`
	Payload             json.RawMessage `json:"payload"`
}

type semanticEvaluationArtifact = semanticevidence.Evidence
type semanticEvaluationDiagnostic = semanticevidence.Diagnostic

func decodeSemanticEvaluationArtifact(data []byte) (semanticEvaluationArtifact, error) {
	return semanticevidence.Decode(data)
}

func semanticEvaluationArtifactEvidenceFingerprint(value semanticEvaluationArtifact) string {
	return semanticevidence.Fingerprint(value)
}

func normalizeEvidenceArtifact(value EvidenceArtifact) (EvidenceArtifact, error) {
	value.Kind = strings.TrimSpace(value.Kind)
	value.Schema = strings.TrimSpace(value.Schema)
	value.ArtifactFingerprint = strings.TrimSpace(value.ArtifactFingerprint)
	if value.Kind == "" || value.Schema == "" || value.ArtifactFingerprint == "" || len(value.Payload) == 0 {
		return value, fmt.Errorf("artifact Evidenceにはkind / schema / artifactFingerprint / payloadが必要です")
	}
	if !revisionPattern.MatchString(value.ArtifactFingerprint) {
		return value, fmt.Errorf("artifact Evidence.artifactFingerprint %q は exact sha256 fingerprintではありません", value.ArtifactFingerprint)
	}

	switch value.Kind {
	case evidenceArtifactKindSemanticEvaluation:
		if value.Schema != semanticEvaluationEvidenceSchema {
			return value, fmt.Errorf(
				"artifact Evidence kind %q のschema %q は未対応です",
				value.Kind,
				value.Schema,
			)
		}
		normalized, err := semanticevidence.Decode(value.Payload)
		if err != nil {
			return value, err
		}
		payload, err := semanticevidence.CanonicalBytes(normalized)
		if err != nil {
			return value, err
		}
		calculated := sourceindex.Fingerprint(payload)
		if calculated != value.ArtifactFingerprint {
			return value, fmt.Errorf(
				"artifact Evidence fingerprintがcanonical payloadと一致しません: recorded=%s calculated=%s",
				value.ArtifactFingerprint,
				calculated,
			)
		}
		value.Payload = payload
	case evidenceArtifactKindRMTTranscription:
		if value.Schema != rmtevidence.Schema {
			return value, fmt.Errorf(
				"artifact Evidence kind %q のschema %q は未対応です",
				value.Kind,
				value.Schema,
			)
		}
		normalized, err := rmtevidence.Decode(value.Payload)
		if err != nil {
			return value, err
		}
		payload, err := rmtevidence.CanonicalBytes(normalized)
		if err != nil {
			return value, err
		}
		calculated := sourceindex.Fingerprint(payload)
		if calculated != value.ArtifactFingerprint {
			return value, fmt.Errorf(
				"artifact Evidence fingerprintがcanonical payloadと一致しません: recorded=%s calculated=%s",
				value.ArtifactFingerprint,
				calculated,
			)
		}
		value.Payload = payload
	default:
		return value, fmt.Errorf("artifact Evidence kind %q は未対応です", value.Kind)
	}
	return value, nil
}

func attachSemanticEvaluationEvidence(candidateData, evaluationData []byte) (Candidate, error) {
	candidate, err := decodeCandidate(candidateData)
	if err != nil {
		return Candidate{}, err
	}
	evaluation, err := semanticevidence.Decode(evaluationData)
	if err != nil {
		return Candidate{}, err
	}
	payload, err := semanticevidence.CanonicalBytes(evaluation)
	if err != nil {
		return Candidate{}, err
	}
	artifact := EvidenceArtifact{
		Kind:                evidenceArtifactKindSemanticEvaluation,
		Schema:              evaluation.Schema,
		ArtifactFingerprint: sourceindex.Fingerprint(payload),
		Payload:             payload,
	}
	candidate.Evidence = append(candidate.Evidence, Evidence{Artifact: &artifact})
	normalized, _, err := normalizeCandidate(candidate)
	if err != nil {
		return Candidate{}, err
	}
	return normalized, nil
}

func validateEvidenceArtifact(value *EvidenceArtifact) []Diagnostic {
	if value == nil {
		return nil
	}
	if _, err := normalizeEvidenceArtifact(*value); err != nil {
		return []Diagnostic{{
			Code:    "EVIDENCE_ARTIFACT_INVALID",
			Message: err.Error(),
		}}
	}
	return nil
}

func semanticEvaluationFromEvidence(e Evidence) (semanticEvaluationArtifact, bool) {
	if e.Artifact == nil || e.Artifact.Kind != evidenceArtifactKindSemanticEvaluation {
		return semanticEvaluationArtifact{}, false
	}
	value, err := semanticevidence.Decode(e.Artifact.Payload)
	if err != nil {
		return semanticEvaluationArtifact{}, false
	}
	return value, true
}
