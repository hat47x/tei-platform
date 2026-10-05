package semanticevidence

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strings"

	"tei/internal/sourceindex"
	"tei/internal/subject"
)

const Schema = "tei.reference.semantic-evaluation-evidence/v0"

type Status string

const (
	Pass    Status = "pass"
	Fail    Status = "fail"
	Unknown Status = "unknown"
)

type Diagnostic struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type Evidence struct {
	Schema                         string       `json:"schema"`
	SemanticSetFingerprint         string       `json:"semanticSetFingerprint"`
	ConstraintExtension            string       `json:"constraintExtension"`
	ConstraintID                   string       `json:"constraintId"`
	Subject                        subject.Subject `json:"subject"`
	ValidatorRef                   string       `json:"validatorRef"`
	EvaluatorInterface             string       `json:"evaluatorInterface"`
	EvaluatorContractFingerprint   string       `json:"evaluatorContractFingerprint"`
	EvaluatorConformanceProfile    string       `json:"evaluatorConformanceProfile"`
	EvaluatorProviderExtension     string       `json:"evaluatorProviderExtension"`
	EvaluatorImplementationID      string       `json:"evaluatorImplementationId"`
	EvaluatorArtifactDigest        string       `json:"evaluatorArtifactDigest"`
	ConformanceEvidenceFingerprint string       `json:"conformanceEvidenceFingerprint"`
	InputFingerprint               string       `json:"inputFingerprint"`
	Status                         Status       `json:"status"`
	Diagnostics                    []Diagnostic `json:"diagnostics,omitempty"`
	EvidenceFingerprint            string       `json:"evidenceFingerprint"`
}

var fingerprintPattern = regexp.MustCompile(`^sha256:[0-9a-f]{64}$`)

func Decode(data []byte) (Evidence, error) {
	var value Evidence
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&value); err != nil {
		return value, fmt.Errorf("semantic evaluation EvidenceをJSONとして読めません: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err == io.EOF {
		// expected
	} else if err != nil {
		return value, err
	} else {
		return value, fmt.Errorf("semantic evaluation EvidenceのJSONの後ろに余分な内容があります")
	}
	return Verify(value)
}

func Prepare(value Evidence) (Evidence, error) {
	normalized, err := normalize(value)
	if err != nil {
		return normalized, err
	}
	normalized.EvidenceFingerprint = Fingerprint(normalized)
	return normalized, nil
}

func Verify(value Evidence) (Evidence, error) {
	recorded := strings.TrimSpace(value.EvidenceFingerprint)
	normalized, err := normalize(value)
	if err != nil {
		return normalized, err
	}
	if !fingerprintPattern.MatchString(recorded) {
		return normalized, fmt.Errorf("semantic evaluation Evidence.evidenceFingerprint %q は exact sha256 fingerprintではありません", recorded)
	}
	calculated := Fingerprint(normalized)
	if recorded != calculated {
		return normalized, fmt.Errorf(
			"semantic evaluation Evidence fingerprintが内容と一致しません: recorded=%s calculated=%s",
			recorded,
			calculated,
		)
	}
	normalized.EvidenceFingerprint = recorded
	return normalized, nil
}

func CanonicalBytes(value Evidence) ([]byte, error) {
	verified, err := Verify(value)
	if err != nil {
		return nil, err
	}
	return json.Marshal(verified)
}

func Fingerprint(value Evidence) string {
	value.EvidenceFingerprint = ""
	encoded, _ := json.Marshal(value)
	return sourceindex.Fingerprint(encoded)
}

func normalize(value Evidence) (Evidence, error) {
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
	value.EvidenceFingerprint = strings.TrimSpace(value.EvidenceFingerprint)

	if value.Schema != Schema {
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
	}
	for _, item := range fingerprints {
		if !fingerprintPattern.MatchString(item.value) {
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
	case Pass, Fail, Unknown:
	default:
		return value, fmt.Errorf("semantic evaluation Evidence.status %q は pass / fail / unknown ではありません", value.Status)
	}

	diagnostics := make([]Diagnostic, 0, len(value.Diagnostics))
	for i, diagnostic := range value.Diagnostics {
		diagnostic.Code = strings.TrimSpace(diagnostic.Code)
		diagnostic.Message = strings.TrimSpace(diagnostic.Message)
		if diagnostic.Code == "" || diagnostic.Message == "" {
			return value, fmt.Errorf("semantic evaluation Evidence.diagnostics[%d]に空のcode / messageがあります", i)
		}
		diagnostics = append(diagnostics, diagnostic)
	}
	sort.Slice(diagnostics, func(i, j int) bool {
		if diagnostics[i].Code != diagnostics[j].Code {
			return diagnostics[i].Code < diagnostics[j].Code
		}
		return diagnostics[i].Message < diagnostics[j].Message
	})
	value.Diagnostics = diagnostics
	return value, nil
}
