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

const (
	subjectContextResolved   = "resolved"
	subjectContextUnresolved = "unresolved"

	subjectExtensionActive     = "active"
	subjectExtensionCandidate  = "candidate"
	subjectExtensionSupporting = "supporting"
	subjectExtensionUnresolved = "unresolved"
)

type SubjectExtensionUse struct {
	Extension                  CatalogExtension        `json:"extension"`
	State                      string                  `json:"state"`
	Roles                      []string                `json:"roles"`
	RootPaths                  []CatalogDependencyPath `json:"rootPaths,omitempty"`
	PackageRequiredBy          []ExtensionIdentity     `json:"packageRequiredBy,omitempty"`
	InterfaceConsumers         []ExtensionIdentity     `json:"interfaceConsumers,omitempty"`
	PotentiallyAffectedExtensions []ExtensionIdentity  `json:"potentiallyAffectedExtensions,omitempty"`
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
	CoreContextStatus          string                       `json:"coreContextStatus"`
	SemanticSetStatus          string                       `json:"semanticSetStatus"`
	SubjectSemanticStatus      string                       `json:"subjectSemanticStatus"`
	Core                       knowledgecontext.Context     `json:"core"`
	Extensions                 []SubjectExtensionUse        `json:"extensions,omitempty"`
	Interfaces                 []CatalogSemanticInterface   `json:"interfaces,omitempty"`
	InterfaceWirings           []CatalogInterfaceWiring     `json:"interfaceWirings,omitempty"`
	Constraints                []CatalogConstraint          `json:"constraints,omitempty"`
	Relations                  []CatalogRelation            `json:"relations,omitempty"`
	Connections                []CatalogConnection          `json:"connections,omitempty"`
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

	for _, unresolved := range catalog.Unresolved {
		if unresolved.Subject == nil || *unresolved.Subject != target ||
			unresolved.Extension == nil {
			continue
		}
		addRole(unresolved.Extension.ID, "unresolved")
	}

	extensionsByID := map[string]CatalogExtension{}
	for _, extension := range catalog.Extensions {
		extensionsByID[extension.ID] = extension
	}
	explanationsByID := map[string]CatalogDependencyExplanation{}
	for _, explanation := range catalog.DependencyExplanations {
		explanationsByID[explanation.Extension.ID] = explanation
	}

	impactAdjacency := subjectExtensionImpactAdjacency(catalog)
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
		state := subjectExtensionState(roles)
		if state == "" {
			return SubjectSemanticContext{}, fmt.Errorf(
				"subject semantic context cannot classify extension %s roles %v",
				id,
				roles,
			)
		}
		use := SubjectExtensionUse{
			Extension: extension,
			State:     state,
			Roles:     roles,
		}
		if explanation, ok := explanationsByID[id]; ok {
			use.RootPaths = append([]CatalogDependencyPath(nil), explanation.RootPaths...)
			use.PackageRequiredBy = append([]ExtensionIdentity(nil), explanation.RequiredBy...)
		}
		use.InterfaceConsumers = subjectInterfaceConsumers(catalog, id)
		use.PotentiallyAffectedExtensions = transitiveSubjectExtensionImpact(
			id,
			impactAdjacency,
		)
		out.Extensions = append(out.Extensions, use)
	}

	for _, connection := range catalog.Connections {
		if _, relevant := rolesByExtension[connection.Extension.ID]; !relevant {
			continue
		}
		out.Connections = append(out.Connections, connection)
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

	out.CoreContextStatus = subjectContextResolved
	if len(core.Unresolved) > 0 {
		out.CoreContextStatus = subjectContextUnresolved
	}
	out.SemanticSetStatus = subjectContextResolved
	if len(out.SemanticSetUnresolved) > 0 {
		out.SemanticSetStatus = subjectContextUnresolved
	}
	out.SubjectSemanticStatus = subjectContextResolved
	if len(out.SubjectUnresolved) > 0 {
		out.SubjectSemanticStatus = subjectContextUnresolved
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
	sort.Slice(out.Connections, func(i, j int) bool {
		if out.Connections[i].Extension.ID != out.Connections[j].Extension.ID {
			return out.Connections[i].Extension.ID < out.Connections[j].Extension.ID
		}
		if out.Connections[i].Kind != out.Connections[j].Kind {
			return out.Connections[i].Kind < out.Connections[j].Kind
		}
		return out.Connections[i].Ref < out.Connections[j].Ref
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

func subjectExtensionImpactAdjacency(
	catalog SemanticCatalog,
) map[string][]ExtensionIdentity {
	adjacency := map[string][]ExtensionIdentity{}
	add := func(from string, to ExtensionIdentity) {
		if from == "" || to.ID == "" || from == to.ID {
			return
		}
		for _, existing := range adjacency[from] {
			if existing == to {
				return
			}
		}
		adjacency[from] = append(adjacency[from], to)
	}

	for _, explanation := range catalog.DependencyExplanations {
		for _, dependent := range explanation.RequiredBy {
			add(explanation.Extension.ID, dependent)
		}
	}

	interfaceOwner := map[string]ExtensionIdentity{}
	for _, contract := range catalog.Interfaces {
		interfaceOwner[contract.ID] = contract.Extension
	}
	for _, wiring := range catalog.InterfaceWirings {
		consumer := extensionIdentityFromCatalog(catalog, wiring.ConsumerExtension)
		provider := extensionIdentityFromCatalog(catalog, wiring.ProviderExtension)
		add(wiring.ProviderExtension, consumer)
		if owner, ok := interfaceOwner[wiring.Interface]; ok {
			add(owner.ID, consumer)
			add(owner.ID, provider)
		}
	}
	for id := range adjacency {
		sort.Slice(adjacency[id], func(i, j int) bool {
			return compareExtensionIdentity(adjacency[id][i], adjacency[id][j]) < 0
		})
	}
	return adjacency
}

func extensionIdentityFromCatalog(
	catalog SemanticCatalog,
	id string,
) ExtensionIdentity {
	for _, extension := range catalog.Extensions {
		if extension.ID == id {
			return ExtensionIdentity{ID: extension.ID, Version: extension.Version}
		}
	}
	return ExtensionIdentity{ID: id}
}

func subjectInterfaceConsumers(
	catalog SemanticCatalog,
	providerID string,
) []ExtensionIdentity {
	seen := map[string]ExtensionIdentity{}
	for _, wiring := range catalog.InterfaceWirings {
		if wiring.ProviderExtension != providerID {
			continue
		}
		identity := extensionIdentityFromCatalog(catalog, wiring.ConsumerExtension)
		seen[identity.ID+"@"+identity.Version] = identity
	}
	out := make([]ExtensionIdentity, 0, len(seen))
	for _, identity := range seen {
		out = append(out, identity)
	}
	sort.Slice(out, func(i, j int) bool {
		return compareExtensionIdentity(out[i], out[j]) < 0
	})
	return out
}

func transitiveSubjectExtensionImpact(
	start string,
	adjacency map[string][]ExtensionIdentity,
) []ExtensionIdentity {
	seen := map[string]ExtensionIdentity{}
	queue := append([]ExtensionIdentity(nil), adjacency[start]...)
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		if current.ID == "" || current.ID == start {
			continue
		}
		key := current.ID + "@" + current.Version
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = current
		queue = append(queue, adjacency[current.ID]...)
	}
	out := make([]ExtensionIdentity, 0, len(seen))
	for _, identity := range seen {
		out = append(out, identity)
	}
	sort.Slice(out, func(i, j int) bool {
		return compareExtensionIdentity(out[i], out[j]) < 0
	})
	return out
}

func subjectExtensionState(roles []string) string {
	roleSet := map[string]struct{}{}
	for _, role := range roles {
		roleSet[role] = struct{}{}
	}
	if _, ok := roleSet["unresolved"]; ok {
		return subjectExtensionUnresolved
	}
	for _, role := range []string{"constraint", "relation", "subject-definition"} {
		if _, ok := roleSet[role]; ok {
			return subjectExtensionActive
		}
	}
	if _, ok := roleSet["evaluator-provider"]; ok {
		return subjectExtensionSupporting
	}
	if _, ok := roleSet["kind-applicable"]; ok {
		return subjectExtensionCandidate
	}
	return ""
}

func subjectSemanticContextFingerprint(value SubjectSemanticContext) string {
	value.ContextFingerprint = ""
	encoded, _ := json.Marshal(value)
	return sourceindex.Fingerprint(encoded)
}
