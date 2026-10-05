package main

import (
	"encoding/json"
	"fmt"
	"sort"

	"tei/internal/knowledgecontext"
	"tei/internal/sourceindex"
	"tei/internal/subject"
)

const subjectSemanticContextSchema = "tei.reference.subject-semantic-context/v0"

type SubjectExtensionUse struct {
	Extension CatalogExtension        `json:"extension"`
	Roles     []string                `json:"roles"`
	RootPaths []CatalogDependencyPath `json:"rootPaths,omitempty"`
}

type SubjectValidationPlanContext struct {
	SemanticSetFingerprint string                     `json:"semanticSetFingerprint"`
	PlanFingerprint        string                     `json:"planFingerprint"`
	Bindings               []SemanticEvaluatorBinding `json:"bindings,omitempty"`
}

type SubjectSemanticContext struct {
	Schema                     string                       `json:"schema"`
	Subject                    subject.Subject              `json:"subject"`
	ContextFingerprint         string                       `json:"contextFingerprint"`
	SemanticSetFingerprint     string                       `json:"semanticSetFingerprint"`
	SemanticEvidenceFingerprint string                      `json:"semanticEvidenceFingerprint"`
	CoreSourceIndexFingerprint string                       `json:"coreSourceIndexFingerprint,omitempty"`
	ResolutionPolicy           string                       `json:"resolutionPolicy"`
	Core                       knowledgecontext.Context     `json:"core"`
	Extensions                 []SubjectExtensionUse        `json:"extensions,omitempty"`
	Interfaces                 []CatalogSemanticInterface   `json:"interfaces,omitempty"`
	InterfaceWirings           []CatalogInterfaceWiring     `json:"interfaceWirings,omitempty"`
	Constraints                []CatalogConstraint          `json:"constraints,omitempty"`
	Relations                  []CatalogRelation            `json:"relations,omitempty"`
	ValidationPlan             *SubjectValidationPlanContext `json:"validationPlan,omitempty"`
	SubjectUnresolved          []CatalogUnresolved          `json:"subjectUnresolved,omitempty"`
	SemanticSetUnresolved      []CatalogUnresolved          `json:"semanticSetUnresolved,omitempty"`
	AuthorityLimits            []string                     `json:"authorityLimits"`
}

func buildSubjectSemanticContext(
	core knowledgecontext.Context,
	catalog SemanticCatalog,
	plan *SemanticValidationPlan,
	target subject.Subject,
) (SubjectSemanticContext, error) {
	if target.IsZero() {
		return SubjectSemanticContext{}, fmt.Errorf("subject semantic context requires exact subject")
	}
	if core.Subject != target {
		return SubjectSemanticContext{}, fmt.Errorf(
			"core knowledge context subject %+v differs from semantic context subject %+v",
			core.Subject,
			target,
		)
	}
	if catalog.ResolutionFingerprint == "" {
		return SubjectSemanticContext{}, fmt.Errorf("subject semantic context requires resolved Semantic Set identity")
	}

	out := SubjectSemanticContext{
		Schema:                      subjectSemanticContextSchema,
		Subject:                     target,
		SemanticSetFingerprint:      catalog.ResolutionFingerprint,
		SemanticEvidenceFingerprint: catalog.EvidenceFingerprint,
		CoreSourceIndexFingerprint:  catalog.CoreSourceIndexFingerprint,
		ResolutionPolicy:            catalog.ResolutionPolicy,
		Core:                        core,
		SemanticSetUnresolved:       append([]CatalogUnresolved(nil), catalog.Unresolved...),
	}

	rolesByExtension := map[string]map[string]struct{}{}
	addRole := func(id, role string) {
		if id == "" || role == "" {
			return
		}
		if rolesByExtension[id] == nil {
			rolesByExtension[id] = map[string]struct{}{}
		}
		rolesByExtension[id][role] = struct{}{}
	}

	for _, extension := range catalog.Extensions {
		for _, kind := range extension.AppliesToKinds {
			if kind == target.Kind {
				addRole(extension.ID, "kind-applicable")
				break
			}
		}
	}
	for _, item := range catalog.Subjects {
		if item.Subject == target && item.Extension != nil {
			addRole(item.Extension.ID, "subject-definition")
		}
	}
	for _, relation := range catalog.Relations {
		if relation.From == target || relation.To == target {
			out.Relations = append(out.Relations, relation)
			addRole(relation.Extension.ID, "relation")
		}
	}
	for _, constraint := range catalog.Constraints {
		if constraint.Subject != target {
			continue
		}
		out.Constraints = append(out.Constraints, constraint)
		addRole(constraint.Extension.ID, "constraint")
		if constraint.EvaluatorProviderExtension != "" {
			addRole(constraint.EvaluatorProviderExtension, "evaluator-provider")
		}
	}

	extensionsByID := map[string]CatalogExtension{}
	for _, extension := range catalog.Extensions {
		extensionsByID[extension.ID] = extension
	}
	explanationsByID := map[string]CatalogDependencyExplanation{}
	for _, explanation := range catalog.DependencyExplanations {
		explanationsByID[explanation.Extension.ID] = explanation
	}
	var extensionIDs []string
	for id := range rolesByExtension {
		extensionIDs = append(extensionIDs, id)
	}
	sort.Strings(extensionIDs)
	for _, id := range extensionIDs {
		extension, ok := extensionsByID[id]
		if !ok {
			return SubjectSemanticContext{}, fmt.Errorf(
				"subject semantic context references unresolved extension %s",
				id,
			)
		}
		var roles []string
		for role := range rolesByExtension[id] {
			roles = append(roles, role)
		}
		sort.Strings(roles)
		use := SubjectExtensionUse{
			Extension: extension,
			Roles:     roles,
		}
		if explanation, ok := explanationsByID[id]; ok {
			use.RootPaths = append([]CatalogDependencyPath(nil), explanation.RootPaths...)
		}
		out.Extensions = append(out.Extensions, use)
	}

	interfaceIDs := map[string]struct{}{}
	for _, constraint := range out.Constraints {
		if constraint.EvaluatorInterface != "" {
			interfaceIDs[constraint.EvaluatorInterface] = struct{}{}
		}
	}
	for _, item := range catalog.Interfaces {
		if _, ok := interfaceIDs[item.ID]; ok {
			out.Interfaces = append(out.Interfaces, item)
		}
	}
	for _, wiring := range catalog.InterfaceWirings {
		if _, ok := interfaceIDs[wiring.Interface]; !ok {
			continue
		}
		if _, consumerRelevant := rolesByExtension[wiring.ConsumerExtension]; consumerRelevant {
			out.InterfaceWirings = append(out.InterfaceWirings, wiring)
		}
	}

	relevantExtensions := map[string]struct{}{}
	for id := range rolesByExtension {
		relevantExtensions[id] = struct{}{}
	}
	for _, unresolved := range catalog.Unresolved {
		subjectRelevant := unresolved.Subject != nil && *unresolved.Subject == target
		extensionRelevant := unresolved.Extension != nil
		if extensionRelevant {
			_, extensionRelevant = relevantExtensions[unresolved.Extension.ID]
		}
		if subjectRelevant || extensionRelevant {
			out.SubjectUnresolved = append(out.SubjectUnresolved, unresolved)
		}
	}

	if plan != nil {
		if err := verifySemanticValidationPlan(*plan, catalog); err != nil {
			return SubjectSemanticContext{}, err
		}
		planContext := SubjectValidationPlanContext{
			SemanticSetFingerprint: plan.SemanticSetFingerprint,
			PlanFingerprint:        plan.PlanFingerprint,
		}
		for _, constraint := range out.Constraints {
			if constraint.EvaluatorInterface == "" {
				continue
			}
			if binding, ok := semanticValidationPlanBinding(
				*plan,
				constraint.Extension.ID,
				constraint.ID,
			); ok {
				planContext.Bindings = append(planContext.Bindings, binding)
			}
		}
		out.ValidationPlan = &planContext
	}

	out.AuthorityLimits = uniqueSortedStrings(append(
		append(
			append([]string(nil), core.AuthorityLimits...),
			catalog.AuthorityLimits...,
		),
		"subject-semantic-context-is-derived-not-canonical-authority",
		"kind-applicable-extension-is-candidate-not-acceptance",
		"validation-plan-binding-is-not-runtime-authority",
		"context-fingerprint-is-derived-not-semantic-identity",
	))

	sort.Slice(out.Interfaces, func(i, j int) bool {
		if out.Interfaces[i].ID != out.Interfaces[j].ID {
			return out.Interfaces[i].ID < out.Interfaces[j].ID
		}
		return out.Interfaces[i].Version < out.Interfaces[j].Version
	})
	sort.Slice(out.InterfaceWirings, func(i, j int) bool {
		if out.InterfaceWirings[i].ConsumerExtension != out.InterfaceWirings[j].ConsumerExtension {
			return out.InterfaceWirings[i].ConsumerExtension < out.InterfaceWirings[j].ConsumerExtension
		}
		return out.InterfaceWirings[i].Interface < out.InterfaceWirings[j].Interface
	})
	sort.Slice(out.Constraints, func(i, j int) bool {
		if out.Constraints[i].Extension.ID != out.Constraints[j].Extension.ID {
			return out.Constraints[i].Extension.ID < out.Constraints[j].Extension.ID
		}
		return out.Constraints[i].ID < out.Constraints[j].ID
	})
	sort.Slice(out.Relations, func(i, j int) bool {
		left := relationKey(ExtensionRelation{
			From: out.Relations[i].From,
			Relation: out.Relations[i].Relation,
			To: out.Relations[i].To,
		})
		right := relationKey(ExtensionRelation{
			From: out.Relations[j].From,
			Relation: out.Relations[j].Relation,
			To: out.Relations[j].To,
		})
		if left != right {
			return left < right
		}
		return out.Relations[i].Extension.ID < out.Relations[j].Extension.ID
	})
	sort.Slice(out.SubjectUnresolved, func(i, j int) bool {
		if out.SubjectUnresolved[i].Code != out.SubjectUnresolved[j].Code {
			return out.SubjectUnresolved[i].Code < out.SubjectUnresolved[j].Code
		}
		return out.SubjectUnresolved[i].Detail < out.SubjectUnresolved[j].Detail
	})

	out.ContextFingerprint = subjectSemanticContextFingerprint(out)
	return out, nil
}

func subjectSemanticContextFingerprint(value SubjectSemanticContext) string {
	value.ContextFingerprint = ""
	encoded, _ := json.Marshal(value)
	return sourceindex.Fingerprint(encoded)
}
