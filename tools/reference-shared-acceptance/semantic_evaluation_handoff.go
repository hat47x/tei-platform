package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"sort"
	"strings"

	"tei/internal/sourceindex"
	"tei/internal/subject"
)

const (
	evidenceArtifactKindSemanticEvaluation = "semantic-evaluation"
	semanticEvaluationEvidenceSchema       = "tei.reference.semantic-evaluation-evidence/v0"
)

type EvidenceArtifact struct {
	Kind                string          `json:"kind"`
	Schema              string          `json:"schema"`
	ArtifactFingerprint string          `json:"artifactFingerprint"`
	Payload             json.RawMessage `json:"payload"`
}

type semanticEvaluationArtifact struct {
	Schema                         string                         `json:"schema"`
	SemanticSetFingerprint         string                         `json:"semanticSetFingerprint"`
	ConstraintExtension            string                         `json:"constraintExtension"`
	ConstraintID                   string                         `json:"constraintId"`
	Subject                        subject.Subject                `json:"subject"`
	ValidatorRef                   string                         `json:"validatorRef"`
	EvaluatorInterface             string                         `json:"evaluatorInterface"`
	EvaluatorContractFingerprint   string                         `json:"evaluatorContractFingerprint"`
	EvaluatorConformanceProfile    string                         `json:"evaluatorConformanceProfile"`
	EvaluatorProviderExtension     string                         `json:"evaluatorProviderExtension"`
	EvaluatorImplementationID      string                         `json:"evaluatorImplementationId"`
	EvaluatorArtifactDigest        string                         `json:"evaluatorArtifactDigest"`
	ConformanceEvidenceFingerprint string                         `json:"conformanceEvidenceFingerprint"`
	InputFingerprint               string                         `json:"inputFingerprint"`
	Status                         string                         `json:"status"`
	Diagnostics                    []semanticEvaluationDiagnostic `json:"diagnostics,omitempty"`
	EvidenceFingerprint            string                         `json:"evidenceFingerprint"`
}

type semanticEvaluationDiagnostic struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func decodeSemanticEvaluationArtifact(data []byte) (semanticEvaluationArtifact, error) {
	var value semanticEvaluationArtifact
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&value); err != nil {
		return value, fmt.Errorf("semantic evaluation EvidenceをJSONとして読めません: %w", err)
	}
	if err := requireSemanticEvaluationEOF(decoder); err != nil {
		return value, err
	}
	return normalizeSemanticEvaluationArtifact(value)
}

func requireSemanticEvaluationEOF(decoder *json.Decoder) error {
	var trailing any
	if err := decoder.Decode(&trailing); err == io.EOF {
		return nil
	} else if err != nil {
		return err
	}
	return fmt.Errorf("semantic evaluation EvidenceのJSONの後ろに余分な内容があります")
}

func normalizeSemanticEvaluationArtifact(value semanticEvaluationArtifact) (semanticEvaluationArtifact, error) {
	value.Schema = strings.TrimSpace(value.Schema)
	value.SemanticSetFingerprint = strings.TrimSpace(value.SemanticSetFingerprint)
	value.ConstraintExtension = strings.TrimSpace(value.ConstraintExtension)
	value.ConstraintID = strings.TrimSpace(value.ConstraintID)
	value.Subject.Kind = strings.TrimSpace(value.Subject.Kind)
	value.Subject.ID = strings.TrimSpace(value.Subject.ID)
	value.ValidatorRef = strings.TrimSpace(value.ValidatorRef)
	value.EvaluatorInterface = strings.TrimSpace(value.EvaluatorInterface)
	value.EvaluatorContractFingerprint = strings.TrimSpace(value.EvaluatorContractFingerprint)
	value.EvaluatorConformanceProfile = strings.TrimSpace(value.EvaluatorConformanceProfile)
	value.EvaluatorProviderExtension = strings.TrimSpace(value.EvaluatorProviderExtension)
	value.EvaluatorImplementationID = strings.TrimSpace(value.EvaluatorImplementationID)
	value.EvaluatorArtifactDigest = strings.TrimSpace(value.EvaluatorArtifactDigest)
	value.ConformanceEvidenceFingerprint = strings.TrimSpace(value.ConformanceEvidenceFingerprint)
	value.InputFingerprint = strings.TrimSpace(value.InputFingerprint)
	value.Status = strings.TrimSpace(value.Status)
	value.EvidenceFingerprint = strings.TrimSpace(value.EvidenceFingerprint)

	if value.Schema != semanticEvaluationEvidenceSchema {
		return value, fmt.Errorf("semantic evaluation Evidence schema %q は未対応です", value.Schema)
	}
	fingerprints := []struct {
		name  string
		value string
	}{
		{"semanticSetFingerprint", value.SemanticSetFingerprint},
		{"evaluatorContractFingerprint", value.EvaluatorContractFingerprint},
		{"evaluatorArtifactDigest", value.EvaluatorArtifactDigest},
		{"conformanceEvidenceFingerprint", value.ConformanceEvidenceFingerprint},
		{"inputFingerprint", value.InputFingerprint},
		{"evidenceFingerprint", value.EvidenceFingerprint},
	}
	for _, item := range fingerprints {
		if !revisionPattern.MatchString(item.value) {
			return value, fmt.Errorf("semantic evaluation Evidence.%s %q は exact sha256 fingerprintではありません", item.name, item.value)
		}
	}
	requiredFields := []struct {
		name  string
		value string
	}{
		{"constraintExtension", value.ConstraintExtension},
		{"constraintId", value.ConstraintID},
		{"subject.kind", value.Subject.Kind},
		{"subject.id", value.Subject.ID},
		{"validatorRef", value.ValidatorRef},
		{"evaluatorInterface", value.EvaluatorInterface},
		{"evaluatorConformanceProfile", value.EvaluatorConformanceProfile},
		{"evaluatorProviderExtension", value.EvaluatorProviderExtension},
		{"evaluatorImplementationId", value.EvaluatorImplementationID},
	}
	for _, item := range requiredFields {
		if item.value == "" {
			return value, fmt.Errorf("semantic evaluation Evidence.%s が空です", item.name)
		}
	}
	switch value.Status {
	case "pass", "fail", "unknown":
	default:
		return value, fmt.Errorf("semantic evaluation Evidence.status %q は pass / fail / unknown ではありません", value.Status)
	}

	normalizedDiagnostics := make([]semanticEvaluationDiagnostic, 0, len(value.Diagnostics))
	for i, diagnostic := range value.Diagnostics {
		diagnostic.Code = strings.TrimSpace(diagnostic.Code)
		diagnostic.Message = strings.TrimSpace(diagnostic.Message)
		if diagnostic.Code == "" || diagnostic.Message == "" {
			return value, fmt.Errorf("semantic evaluation Evidence.diagnostics[%d]に空のcode / messageがあります", i)
		}
		normalizedDiagnostics = append(normalizedDiagnostics, diagnostic)
	}
	sort.Slice(normalizedDiagnostics, func(i, j int) bool {
		if normalizedDiagnostics[i].Code != normalizedDiagnostics[j].Code {
			return normalizedDiagnostics[i].Code < normalizedDiagnostics[j].Code
		}
		return normalizedDiagnostics[i].Message < normalizedDiagnostics[j].Message
	})
	if !reflect.DeepEqual(normalizedDiagnostics, value.Diagnostics) {
		return value, fmt.Errorf("semantic evaluation Evidence.diagnosticsがcanonical orderではありません")
	}
	value.Diagnostics = normalizedDiagnostics

	if calculated := semanticEvaluationArtifactEvidenceFingerprint(value); calculated != value.EvidenceFingerprint {
		return value, fmt.Errorf(
			"semantic evaluation Evidence fingerprintが内容と一致しません: recorded=%s calculated=%s",
			value.EvidenceFingerprint,
			calculated,
		)
	}
	return value, nil
}

func semanticEvaluationArtifactEvidenceFingerprint(value semanticEvaluationArtifact) string {
	value.EvidenceFingerprint = ""
	encoded, _ := json.Marshal(value)
	return sourceindex.Fingerprint(encoded)
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
		normalized, err := decodeSemanticEvaluationArtifact(value.Payload)
		if err != nil {
			return value, err
		}
		payload, _ := json.Marshal(normalized)
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
	evaluation, err := decodeSemanticEvaluationArtifact(evaluationData)
	if err != nil {
		return Candidate{}, err
	}
	payload, _ := json.Marshal(evaluation)
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
	var value semanticEvaluationArtifact
	if err := json.Unmarshal(e.Artifact.Payload, &value); err != nil {
		return semanticEvaluationArtifact{}, false
	}
	return value, true
}
