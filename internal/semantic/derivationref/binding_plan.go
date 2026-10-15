package derivationref

import (
	"fmt"
	"sort"
	"strings"

	"tei/internal/core"
	"tei/internal/modelref"
	"tei/internal/semantic"
	"tei/internal/subject"
)

// The D4 binding planner is read-only. Its accepted inputs MUST be supplied
// from the existing acceptance boundary; constructing this value does not
// itself grant acceptance or runtime execution authority.
const (
	SemanticAttributeScope = "semantic-attribute"
	QueryLocalScope        = "query-local"
)

// RuleContract is an accepted view of a D1 rule, not a second definition.
// Its version and meaning must come from the canonical acceptance snapshot.
type RuleContract struct {
	Ref          modelref.Ref
	DefinitionID string
	OutputType   core.CanonicalID
	NullPolicy   string
}

// TypedTarget is a read-only target projection. For actual Semantic Data
// attributes, use SemanticAttributeTarget rather than guessing a symbol.
type TypedTarget struct {
	Ref        modelref.Ref
	OutputType core.CanonicalID
}

type AcceptedBinding struct {
	ID     string
	Rule   modelref.Ref
	Target modelref.Ref
	Scope  string
}

// AcceptedChoice records an explicit, externally accepted decision for a
// conflicting target. There is no registration-order or priority fallback.
type AcceptedChoice struct {
	Target    modelref.Ref
	BindingID string
}

// SelectedRealization is a selection observed from the external composition
// and Plugin Lock mechanism. This planner checks its semantic compatibility,
// but NEVER authorizes or invokes the selected plugin.
type SelectedRealization struct {
	Rule           modelref.Ref
	PluginID       string
	Implementation string
	OutputType     core.CanonicalID
	NullPolicy     string
}

type AcceptedBindingSnapshot struct {
	Rules        []RuleContract
	Bindings     []AcceptedBinding
	Choices      []AcceptedChoice
	Realizations []SelectedRealization
}

type BindingRequest struct {
	Graph            modelref.Graph
	Definitions      []Definition
	Accepted         AcceptedBindingSnapshot
	ProposedBindings []AcceptedBinding
	Targets          []TypedTarget
}

type ResolvedBinding struct {
	BindingID      string
	Rule           modelref.Key
	Target         modelref.Key
	Scope          string
	OutputType     core.CanonicalID
	PluginID       string
	Implementation string
}

type BindingDiagnostic struct {
	Code string
	At   string
}

type BindingPlan struct {
	Bindings    []ResolvedBinding
	Diagnostics []BindingDiagnostic
}

// SemanticAttributeTarget connects D4 to the existing Semantic Data Attribute
// type, without introducing a duplicate type registry or a physical-column
// lookup. The modelref graph must separately contain the same logical node.
func SemanticAttributeTarget(scope string, namespace semantic.Namespace, dataID, attributeID core.CanonicalID, version string) (TypedTarget, error) {
	if scope == "" || version == "" || dataID == "" || attributeID == "" {
		return TypedTarget{}, fmt.Errorf("semantic attribute target requires scope, version, data id and attribute id")
	}
	matches := 0
	var attr semantic.Attribute
	for _, data := range namespace.Data {
		if data.ID != dataID {
			continue
		}
		for _, candidate := range data.Attributes {
			if candidate.ID == attributeID {
				attr = candidate
				matches++
			}
		}
	}
	if matches != 1 || attr.TypeRef == "" {
		return TypedTarget{}, fmt.Errorf("semantic attribute %q of data %q is missing, ambiguous or untyped", attributeID, dataID)
	}
	return TypedTarget{
		Ref: modelref.Ref{Scope: scope, Subject: subject.Subject{Kind: "attribute", ID: string(attr.ID)}, Version: version},
		OutputType: attr.TypeRef,
	}, nil
}

// ResolveBindings produces a deterministic, all-or-nothing plan. No selected
// Binding is returned if any binding, selection, type or realization is invalid.
// modelref.Resolve remains the sole typed logical reference resolver.
func ResolveBindings(request BindingRequest) BindingPlan {
	var plan BindingPlan
	diag := func(code, at string) {
		plan.Diagnostics = append(plan.Diagnostics, BindingDiagnostic{Code: code, At: at})
	}
	if request.Graph.Schema != "tei.modelref.v0" {
		diag("invalid-graph-schema", "graph/schema")
	}
	if err := ValidateGraph(request.Definitions, nil); err != nil {
		diag("invalid-rule-definition", "definitions")
	}
	definitions := make(map[string]Definition, len(request.Definitions))
	for _, d := range request.Definitions {
		definitions[d.ID] = d
	}

	resolve := func(ref modelref.Ref, at string) (modelref.Key, bool) {
		if strings.TrimSpace(ref.Scope) == "" || ref.Scope != strings.TrimSpace(ref.Scope) || strings.TrimSpace(ref.Version) == "" {
			diag("unqualified-reference", at)
			return modelref.Key{}, false
		}
		key, issue := modelref.Resolve(request.Graph.Nodes, ref)
		if issue != "" {
			diag(issue, at)
			return modelref.Key{}, false
		}
		return key, true
	}

	// Accepted views are input claims, not a newly minted acceptance registry.
	// Every reference is still resolved against the supplied modelref snapshot.
	rules := make(map[modelref.Key]RuleContract, len(request.Accepted.Rules))
	for i, rule := range request.Accepted.Rules {
		at := fmt.Sprintf("accepted/rules/%d", i)
		key, ok := resolve(rule.Ref, at)
		if !ok {
			continue
		}
		if key.Subject.Kind != "derivation-rule" || rule.DefinitionID != key.Subject.ID {
			diag("rule-identity-mismatch", at)
			continue
		}
		if _, ok := definitions[rule.DefinitionID]; !ok {
			diag("rule-definition-missing", at)
			continue
		}
		if rule.OutputType == "" || rule.NullPolicy == "" {
			diag("untyped-rule-contract", at)
			continue
		}
		if _, exists := rules[key]; exists {
			diag("duplicate-rule-contract", at)
			continue
		}
		rules[key] = rule
	}

	targets := make(map[modelref.Key]TypedTarget, len(request.Targets))
	for i, target := range request.Targets {
		at := fmt.Sprintf("targets/%d", i)
		key, ok := resolve(target.Ref, at)
		if !ok {
			continue
		}
		if key.Subject.Kind != "attribute" && key.Subject.Kind != "query-local-result" {
			diag("invalid-target-kind", at)
			continue
		}
		if target.OutputType == "" {
			diag("untyped-target", at)
			continue
		}
		if _, exists := targets[key]; exists {
			diag("duplicate-target-contract", at)
			continue
		}
		targets[key] = target
	}

	// A proposal must be admitted by the existing Acceptance flow first.
	// Its presence in the planning request never creates an accepted binding.
	for i := range request.ProposedBindings {
		diag("unaccepted-binding", fmt.Sprintf("proposed/%d", i))
	}

	type candidate struct {
		entry ResolvedBinding
	}
	groups := make(map[modelref.Key][]candidate)
	seenIDs := make(map[string]bool)
	for i, binding := range request.Accepted.Bindings {
		at := fmt.Sprintf("accepted/bindings/%d", i)
		if strings.TrimSpace(binding.ID) == "" || seenIDs[binding.ID] {
			diag("duplicate-or-empty-binding-id", at)
			continue
		}
		seenIDs[binding.ID] = true
		ruleKey, ruleOK := resolve(binding.Rule, at+"/rule")
		targetKey, targetOK := resolve(binding.Target, at+"/target")
		if !ruleOK || !targetOK {
			continue
		}
		if binding.Rule.Subject.Kind != "derivation-rule" {
			diag("wrong-rule-kind", at)
			continue
		}
		rule, ruleAccepted := rules[ruleKey]
		if !ruleAccepted {
			diag("unaccepted-rule", at)
			continue
		}
		target, targetExists := targets[targetKey]
		if !targetExists {
			diag("target-contract-missing", at)
			continue
		}
		switch binding.Scope {
		case SemanticAttributeScope:
			if targetKey.Subject.Kind != "attribute" {
				diag("binding-scope-mismatch", at)
				continue
			}
		case QueryLocalScope:
			if targetKey.Subject.Kind != "query-local-result" {
				diag("binding-scope-mismatch", at)
				continue
			}
		default:
			diag("unknown-binding-scope", at)
			continue
		}
		if target.OutputType != rule.OutputType {
			diag("target-type-mismatch", at)
			continue
		}
		groups[targetKey] = append(groups[targetKey], candidate{entry: ResolvedBinding{
			BindingID: binding.ID, Rule: ruleKey, Target: targetKey,
			Scope: binding.Scope, OutputType: rule.OutputType,
		}})
	}

	choices := make(map[modelref.Key]string)
	for i, choice := range request.Accepted.Choices {
		at := fmt.Sprintf("accepted/choices/%d", i)
		key, ok := resolve(choice.Target, at)
		if !ok {
			continue
		}
		if choice.BindingID == "" {
			diag("invalid-choice", at)
			continue
		}
		if _, already := choices[key]; already {
			diag("conflicting-choice", at)
			continue
		}
		choices[key] = choice.BindingID
	}
	realizations := make(map[modelref.Key]SelectedRealization)
	for i, selected := range request.Accepted.Realizations {
		at := fmt.Sprintf("accepted/realizations/%d", i)
		key, ok := resolve(selected.Rule, at)
		if !ok {
			continue
		}
		rule, exists := rules[key]
		if !exists {
			diag("unaccepted-realization-rule", at)
			continue
		}
		if selected.PluginID == "" || selected.Implementation == "" {
			diag("invalid-realization", at)
			continue
		}
		if selected.OutputType != rule.OutputType || selected.NullPolicy != rule.NullPolicy {
			diag("realization-contract-change-reaccept", at)
			continue
		}
		if _, duplicate := realizations[key]; duplicate {
			diag("conflicting-realization", at)
			continue
		}
		realizations[key] = selected
	}

	for key, bindingID := range choices {
		group := groups[key]
		found := false
		for _, candidate := range group {
			if candidate.entry.BindingID == bindingID {
				found = true
			}
		}
		if !found {
			diag("choice-not-applicable", "accepted/choices")
		}
	}

	for key, group := range groups {
		selected := -1
		if len(group) == 1 {
			selected = 0
		} else {
			if preferred, ok := choices[key]; ok {
				for i, candidate := range group {
					if candidate.entry.BindingID == preferred {
						selected = i
						break
					}
				}
			}
			if selected < 0 {
				diag("ambiguous-target", "target/"+key.Scope+"/"+key.Subject.Kind+"/"+key.Subject.ID)
			}
		}
		if selected < 0 {
			continue
		}
		entry := group[selected].entry
		if implementation, ok := realizations[entry.Rule]; ok {
			entry.PluginID = implementation.PluginID
			entry.Implementation = implementation.Implementation
		}
		plan.Bindings = append(plan.Bindings, entry)
	}
	sort.Slice(plan.Bindings, func(i, j int) bool {
		a, b := plan.Bindings[i], plan.Bindings[j]
		if a.Target.Scope != b.Target.Scope {
			return a.Target.Scope < b.Target.Scope
		}
		if a.Target.Subject.Kind != b.Target.Subject.Kind {
			return a.Target.Subject.Kind < b.Target.Subject.Kind
		}
		if a.Target.Subject.ID != b.Target.Subject.ID {
			return a.Target.Subject.ID < b.Target.Subject.ID
		}
		return a.BindingID < b.BindingID
	})
	sort.Slice(plan.Diagnostics, func(i, j int) bool {
		a, b := plan.Diagnostics[i], plan.Diagnostics[j]
		if a.At != b.At {
			return a.At < b.At
		}
		return a.Code < b.Code
	})
	if len(plan.Diagnostics) != 0 {
		plan.Bindings = nil
	}
	return plan
}
