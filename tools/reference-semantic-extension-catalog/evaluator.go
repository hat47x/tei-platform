package main

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"tei/internal/sourceindex"
	"tei/internal/subject"
)

const semanticEvaluationEvidenceSchema = "tei.reference.semantic-evaluation-evidence/v0"

type SemanticEvaluationStatus string

const (
	SemanticEvaluationPass SemanticEvaluationStatus = "pass"
	SemanticEvaluationFail SemanticEvaluationStatus = "fail"
	SemanticEvaluationUnknown SemanticEvaluationStatus = "unknown"
)

func (s SemanticEvaluationStatus) valid() bool {
	return s == SemanticEvaluationPass || s == SemanticEvaluationFail || s == SemanticEvaluationUnknown
}

type SemanticEvaluationDiagnostic struct {
	Code string `json:"code"`
	Message string `json:"message"`
}

type SemanticEvaluationOutcome struct {
	Status SemanticEvaluationStatus `json:"status"`
	Diagnostics []SemanticEvaluationDiagnostic `json:"diagnostics,omitempty"`
}

type SemanticEvaluatorFunc func(context.Context, []byte) (SemanticEvaluationOutcome, error)

type SemanticEvaluatorImplementation struct {
	ID string
	ProviderExtension string
	Interface string
	ContractFingerprint string
	ArtifactDigest string
	Evaluate SemanticEvaluatorFunc
}

type SemanticEvaluatorBinding struct {
	ConstraintExtension string `json:"constraintExtension"`
	ConstraintID string `json:"constraintId"`
	ImplementationID string `json:"implementationId"`
	ArtifactDigest string `json:"artifactDigest"`
}

type SemanticEvaluationEvidence struct {
	Schema string `json:"schema"`
	SemanticSetFingerprint string `json:"semanticSetFingerprint"`
	ConstraintExtension string `json:"constraintExtension"`
	ConstraintID string `json:"constraintId"`
	Subject subject.Subject `json:"subject"`
	ValidatorRef string `json:"validatorRef"`
	EvaluatorInterface string `json:"evaluatorInterface"`
	EvaluatorContractFingerprint string `json:"evaluatorContractFingerprint"`
	EvaluatorConformanceProfile string `json:"evaluatorConformanceProfile"`
	EvaluatorProviderExtension string `json:"evaluatorProviderExtension"`
	EvaluatorImplementationID string `json:"evaluatorImplementationId"`
	EvaluatorArtifactDigest string `json:"evaluatorArtifactDigest"`
	ConformanceEvidenceFingerprint string `json:"conformanceEvidenceFingerprint"`
	InputFingerprint string `json:"inputFingerprint"`
	Status SemanticEvaluationStatus `json:"status"`
	Diagnostics []SemanticEvaluationDiagnostic `json:"diagnostics,omitempty"`
	EvidenceFingerprint string `json:"evidenceFingerprint"`
}

func evaluateSemanticConstraint(
	ctx context.Context,
	catalog SemanticCatalog,
	binding SemanticEvaluatorBinding,
	implementation SemanticEvaluatorImplementation,
	acceptedConformance []SemanticInterfaceConformance,
	input []byte,
) (SemanticEvaluationEvidence, error) {
	if ctx == nil {
		return SemanticEvaluationEvidence{}, fmt.Errorf("semantic evaluator requires context")
	}
	if len(catalog.Unresolved) != 0 {
		return SemanticEvaluationEvidence{}, fmt.Errorf("semantic evaluator requires a fully resolved Semantic Set")
	}
	if !validSHA256Fingerprint(catalog.ResolutionFingerprint) {
		return SemanticEvaluationEvidence{}, fmt.Errorf("semantic evaluator requires exact Semantic Set fingerprint")
	}

	binding.ConstraintExtension = strings.TrimSpace(binding.ConstraintExtension)
	binding.ConstraintID = strings.TrimSpace(binding.ConstraintID)
	binding.ImplementationID = strings.TrimSpace(binding.ImplementationID)
	binding.ArtifactDigest = strings.TrimSpace(binding.ArtifactDigest)
	if binding.ConstraintExtension == "" || binding.ConstraintID == "" ||
		binding.ImplementationID == "" || !validSHA256Fingerprint(binding.ArtifactDigest) {
		return SemanticEvaluationEvidence{}, fmt.Errorf("semantic evaluator binding requires exact constraint, implementation, and artifact digest")
	}

	constraint, ok := findCatalogConstraint(catalog, binding.ConstraintExtension, binding.ConstraintID)
	if !ok {
		return SemanticEvaluationEvidence{}, fmt.Errorf(
			"semantic evaluator binding references unavailable constraint %s/%s",
			binding.ConstraintExtension, binding.ConstraintID,
		)
	}
	if constraint.EvaluatorInterface == "" ||
		constraint.EvaluatorProviderExtension == "" ||
		!validSHA256Fingerprint(constraint.EvaluatorContractFingerprint) ||
		constraint.EvaluatorConformanceProfile == "" {
		return SemanticEvaluationEvidence{}, fmt.Errorf(
			"constraint %s/%s has no resolved semantic evaluator binding",
			binding.ConstraintExtension, binding.ConstraintID,
		)
	}

	implementation.ID = strings.TrimSpace(implementation.ID)
	implementation.ProviderExtension = strings.TrimSpace(implementation.ProviderExtension)
	implementation.Interface = strings.TrimSpace(implementation.Interface)
	implementation.ContractFingerprint = strings.TrimSpace(implementation.ContractFingerprint)
	implementation.ArtifactDigest = strings.TrimSpace(implementation.ArtifactDigest)
	if implementation.Evaluate == nil {
		return SemanticEvaluationEvidence{}, fmt.Errorf("semantic evaluator implementation %s has no evaluator", implementation.ID)
	}
	if implementation.ID != binding.ImplementationID || implementation.ArtifactDigest != binding.ArtifactDigest {
		return SemanticEvaluationEvidence{}, fmt.Errorf("semantic evaluator implementation does not match explicit Validation binding")
	}
	if implementation.ProviderExtension != constraint.EvaluatorProviderExtension ||
		implementation.Interface != constraint.EvaluatorInterface ||
		implementation.ContractFingerprint != constraint.EvaluatorContractFingerprint {
		return SemanticEvaluationEvidence{}, fmt.Errorf(
			"semantic evaluator implementation %s is not bound to constraint contract %s / %s",
			implementation.ID, constraint.EvaluatorInterface, constraint.EvaluatorContractFingerprint,
		)
	}
	if !validSHA256Fingerprint(implementation.ArtifactDigest) {
		return SemanticEvaluationEvidence{}, fmt.Errorf("semantic evaluator implementation %s has invalid artifact digest", implementation.ID)
	}

	conformances, err := normalizeSemanticInterfaceConformance(acceptedConformance)
	if err != nil {
		return SemanticEvaluationEvidence{}, err
	}
	conformanceKey := semanticInterfaceConformanceKey(
		constraint.EvaluatorProviderExtension,
		constraint.EvaluatorInterface,
		constraint.EvaluatorContractFingerprint,
		constraint.EvaluatorConformanceProfile,
	)
	conformance, ok := conformances[conformanceKey]
	if !ok {
		return SemanticEvaluationEvidence{}, fmt.Errorf(
			"constraint %s/%s has no currently accepted evaluator Conformance Evidence",
			binding.ConstraintExtension, binding.ConstraintID,
		)
	}

	if err := ctx.Err(); err != nil {
		return SemanticEvaluationEvidence{}, fmt.Errorf("semantic evaluator did not start: %w", err)
	}
	outcome, err := implementation.Evaluate(ctx, input)
	if err != nil {
		return SemanticEvaluationEvidence{}, fmt.Errorf("semantic evaluator %s failed: %w", implementation.ID, err)
	}
	if err := ctx.Err(); err != nil {
		return SemanticEvaluationEvidence{}, fmt.Errorf("semantic evaluator did not complete: %w", err)
	}
	if !outcome.Status.valid() {
		return SemanticEvaluationEvidence{}, fmt.Errorf("semantic evaluator %s returned invalid status %q", implementation.ID, outcome.Status)
	}
	outcome.Diagnostics = normalizeSemanticEvaluationDiagnostics(outcome.Diagnostics)

	evidence := SemanticEvaluationEvidence{
		Schema: semanticEvaluationEvidenceSchema,
		SemanticSetFingerprint: catalog.ResolutionFingerprint,
		ConstraintExtension: binding.ConstraintExtension,
		ConstraintID: binding.ConstraintID,
		Subject: constraint.Subject,
		ValidatorRef: constraint.ValidatorRef,
		EvaluatorInterface: constraint.EvaluatorInterface,
		EvaluatorContractFingerprint: constraint.EvaluatorContractFingerprint,
		EvaluatorConformanceProfile: constraint.EvaluatorConformanceProfile,
		EvaluatorProviderExtension: constraint.EvaluatorProviderExtension,
		EvaluatorImplementationID: implementation.ID,
		EvaluatorArtifactDigest: implementation.ArtifactDigest,
		ConformanceEvidenceFingerprint: conformance.EvidenceFingerprint,
		InputFingerprint: sourceindex.Fingerprint(input),
		Status: outcome.Status,
		Diagnostics: outcome.Diagnostics,
	}
	evidence.EvidenceFingerprint = semanticEvaluationEvidenceFingerprint(evidence)
	return evidence, nil
}

func findCatalogConstraint(catalog SemanticCatalog, extensionID, constraintID string) (CatalogConstraint, bool) {
	for _, constraint := range catalog.Constraints {
		if constraint.Extension.ID == extensionID && constraint.ID == constraintID {
			return constraint, true
		}
	}
	return CatalogConstraint{}, false
}

func normalizeSemanticEvaluationDiagnostics(values []SemanticEvaluationDiagnostic) []SemanticEvaluationDiagnostic {
	out := make([]SemanticEvaluationDiagnostic, 0, len(values))
	for _, value := range values {
		value.Code = strings.TrimSpace(value.Code)
		value.Message = strings.TrimSpace(value.Message)
		if value.Code == "" || value.Message == "" {
			continue
		}
		out = append(out, value)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Code != out[j].Code {
			return out[i].Code < out[j].Code
		}
		return out[i].Message < out[j].Message
	})
	return out
}

func semanticEvaluationEvidenceFingerprint(value SemanticEvaluationEvidence) string {
	value.EvidenceFingerprint = ""
	encoded, _ := json.Marshal(value)
	return sourceindex.Fingerprint(encoded)
}
