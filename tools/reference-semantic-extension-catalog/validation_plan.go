package main

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"tei/internal/sourceindex"
)

const semanticValidationPlanSchema = "tei.reference.semantic-validation-plan/v0"

type SemanticValidationPlan struct {
	Schema string `json:"schema"`
	SemanticSetFingerprint string `json:"semanticSetFingerprint"`
	Bindings []SemanticEvaluatorBinding `json:"bindings"`
	PlanFingerprint string `json:"planFingerprint"`
}

func buildSemanticValidationPlan(
	catalog SemanticCatalog,
	bindings []SemanticEvaluatorBinding,
) (SemanticValidationPlan, error) {
	if len(catalog.Unresolved) != 0 {
		return SemanticValidationPlan{}, fmt.Errorf("semantic validation plan requires a fully resolved Semantic Set")
	}
	if !validSHA256Fingerprint(catalog.ResolutionFingerprint) {
		return SemanticValidationPlan{}, fmt.Errorf("semantic validation plan requires exact Semantic Set fingerprint")
	}
	normalized, err := normalizeSemanticEvaluatorBindings(catalog, bindings)
	if err != nil {
		return SemanticValidationPlan{}, err
	}
	plan := SemanticValidationPlan{
		Schema: semanticValidationPlanSchema,
		SemanticSetFingerprint: catalog.ResolutionFingerprint,
		Bindings: normalized,
	}
	plan.PlanFingerprint = semanticValidationPlanFingerprint(plan)
	return plan, nil
}

func verifySemanticValidationPlan(plan SemanticValidationPlan, catalog SemanticCatalog) error {
	plan.Schema = strings.TrimSpace(plan.Schema)
	plan.SemanticSetFingerprint = strings.TrimSpace(plan.SemanticSetFingerprint)
	plan.PlanFingerprint = strings.TrimSpace(plan.PlanFingerprint)
	if plan.Schema != semanticValidationPlanSchema {
		return fmt.Errorf("semantic validation plan schema %q is not supported", plan.Schema)
	}
	if plan.SemanticSetFingerprint != catalog.ResolutionFingerprint {
		return fmt.Errorf(
			"semantic validation plan is bound to Semantic Set %s, current set is %s",
			plan.SemanticSetFingerprint, catalog.ResolutionFingerprint,
		)
	}
	if !validSHA256Fingerprint(plan.PlanFingerprint) ||
		plan.PlanFingerprint != semanticValidationPlanFingerprint(plan) {
		return fmt.Errorf("semantic validation plan fingerprint is stale")
	}
	normalized, err := normalizeSemanticEvaluatorBindings(catalog, plan.Bindings)
	if err != nil {
		return err
	}
	if !sameSemanticEvaluatorBindings(normalized, plan.Bindings) {
		return fmt.Errorf("semantic validation plan bindings are not canonical")
	}
	return nil
}

func semanticValidationPlanBinding(
	plan SemanticValidationPlan,
	extensionID, constraintID string,
) (SemanticEvaluatorBinding, bool) {
	for _, binding := range plan.Bindings {
		if binding.ConstraintExtension == extensionID && binding.ConstraintID == constraintID {
			return binding, true
		}
	}
	return SemanticEvaluatorBinding{}, false
}

func normalizeSemanticEvaluatorBindings(
	catalog SemanticCatalog,
	bindings []SemanticEvaluatorBinding,
) ([]SemanticEvaluatorBinding, error) {
	out := make([]SemanticEvaluatorBinding, 0, len(bindings))
	seen := map[string]struct{}{}
	for _, binding := range bindings {
		binding.ConstraintExtension = strings.TrimSpace(binding.ConstraintExtension)
		binding.ConstraintID = strings.TrimSpace(binding.ConstraintID)
		binding.ImplementationID = strings.TrimSpace(binding.ImplementationID)
		binding.ArtifactDigest = strings.TrimSpace(binding.ArtifactDigest)
		if binding.ConstraintExtension == "" || binding.ConstraintID == "" ||
			binding.ImplementationID == "" || !validSHA256Fingerprint(binding.ArtifactDigest) {
			return nil, fmt.Errorf("semantic validation binding requires exact constraint, implementation, and artifact digest")
		}
		constraint, ok := findCatalogConstraint(catalog, binding.ConstraintExtension, binding.ConstraintID)
		if !ok {
			return nil, fmt.Errorf(
				"semantic validation binding references unavailable constraint %s/%s",
				binding.ConstraintExtension, binding.ConstraintID,
			)
		}
		if constraint.EvaluatorInterface == "" ||
			constraint.EvaluatorProviderExtension == "" ||
			!validSHA256Fingerprint(constraint.EvaluatorContractFingerprint) ||
			constraint.EvaluatorConformanceProfile == "" {
			return nil, fmt.Errorf(
				"semantic validation binding references constraint %s/%s without resolved evaluator",
				binding.ConstraintExtension, binding.ConstraintID,
			)
		}
		key := semanticEvaluatorBindingKey(binding.ConstraintExtension, binding.ConstraintID)
		if _, ok := seen[key]; ok {
			return nil, fmt.Errorf("semantic validation plan duplicates binding %s", key)
		}
		seen[key] = struct{}{}
		out = append(out, binding)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].ConstraintExtension != out[j].ConstraintExtension {
			return out[i].ConstraintExtension < out[j].ConstraintExtension
		}
		return out[i].ConstraintID < out[j].ConstraintID
	})
	return out, nil
}

func semanticEvaluatorBindingKey(extensionID, constraintID string) string {
	return fmt.Sprintf("%q|%q", extensionID, constraintID)
}

func sameSemanticEvaluatorBindings(left, right []SemanticEvaluatorBinding) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

func semanticValidationPlanFingerprint(plan SemanticValidationPlan) string {
	plan.PlanFingerprint = ""
	encoded, _ := json.Marshal(plan)
	return sourceindex.Fingerprint(encoded)
}
