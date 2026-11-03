package artifact

import (
	"fmt"
	"sort"
	"strings"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/semantic"
	"tei/internal/sourceindex"
)

const (
	// FunctionDefinitionRuleID and FunctionDefinitionRuleVersion identify the
	// rules this producer applies.
	FunctionDefinitionRuleID      = "tei.artifact.function-definition"
	FunctionDefinitionRuleVersion = "v1"

	// FunctionDefinitionGenerator identifies the implementation.
	FunctionDefinitionGenerator = "tei-artifact-functiondef/v1"
)

const functionDefinitionRulePolicy = `function-definition/v1
lines: application, capability (<id>), input, outcome, output, effect, validation, period, effect-rule, pattern
order: capabilities by id, inputs and outcomes in the authored order, patterns by id
unresolved: capability without input, capability without output or outcomes, capability without effect`

// FunctionDefinitionInput is everything the producer reads.
type FunctionDefinitionInput struct {
	InputRevision string
	Application   application.Application
	Capabilities  map[core.CanonicalID]semantic.Capability
	Validations   map[core.CanonicalID]semantic.Validation
	Periods       map[core.CanonicalID]semantic.Period
	EffectRules   []semantic.EffectRule
}

// RenderFunctionDefinition renders the functions an application declares. It
// reads only accepted meaning: capabilities, their inputs and results, effects,
// the validations the application uses and the patterns it binds.
func RenderFunctionDefinition(input FunctionDefinitionInput) (Artifact, error) {
	content, unresolved := renderFunctionDefinition(input)
	return New(
		KindFunctionDefinition,
		Rule{
			ID:          FunctionDefinitionRuleID,
			Version:     FunctionDefinitionRuleVersion,
			Fingerprint: sourceindex.Fingerprint([]byte(functionDefinitionRulePolicy)),
		},
		FunctionDefinitionGenerator,
		input.InputRevision,
		content,
		unresolved,
	)
}

func renderFunctionDefinition(input FunctionDefinitionInput) (string, []Unresolved) {
	var b strings.Builder
	fmt.Fprintf(&b, "application %s\n", input.Application.ID)

	capabilityIDs := make([]core.CanonicalID, 0, len(input.Capabilities))
	for id := range input.Capabilities {
		capabilityIDs = append(capabilityIDs, id)
	}
	sort.Slice(capabilityIDs, func(i, j int) bool { return capabilityIDs[i] < capabilityIDs[j] })

	var unresolved []Unresolved
	for _, id := range capabilityIDs {
		capability := input.Capabilities[id]
		fmt.Fprintf(&b, "capability %s\n", capability.ID)
		for _, parameter := range capability.Inputs {
			fmt.Fprintf(&b, "input %s %s\n", parameter.Name, parameter.Type)
		}
		if capability.Output != "" {
			fmt.Fprintf(&b, "output %s\n", capability.Output)
		}
		outcomes := append([]semantic.Outcome(nil), capability.Outcomes...)
		sort.Slice(outcomes, func(i, j int) bool { return outcomes[i].Name < outcomes[j].Name })
		for _, outcome := range outcomes {
			fmt.Fprintf(&b, "outcome %s %s\n", outcome.Name, outcome.Payload)
		}
		if capability.Effect != nil {
			fmt.Fprintf(&b, "effect %s %s\n", capability.Effect.Kind, capability.Effect.Data)
		}
		for _, validation := range sortedValidations(input.Application.Validations, input.Validations) {
			fmt.Fprintf(&b, "validation %s %s\n", validation.ID, validation.Kind)
		}

		if len(capability.Inputs) == 0 {
			unresolved = append(unresolved, Unresolved{
				Subject: string(capability.ID),
				Reason:  "missing-input",
				Detail:  "capability declares no input",
			})
		}
		if capability.Output == "" && len(capability.Outcomes) == 0 {
			unresolved = append(unresolved, Unresolved{
				Subject: string(capability.ID),
				Reason:  "missing-semantics",
				Detail:  "capability declares neither an output nor outcomes",
			})
		}
		if capability.Effect == nil {
			unresolved = append(unresolved, Unresolved{
				Subject: string(capability.ID),
				Reason:  "missing-semantics",
				Detail:  "capability declares no effect",
			})
		}
	}

	periodIDs := make([]core.CanonicalID, 0, len(input.Periods))
	for id := range input.Periods {
		periodIDs = append(periodIDs, id)
	}
	sort.Slice(periodIDs, func(i, j int) bool { return periodIDs[i] < periodIDs[j] })
	for _, id := range periodIDs {
		period := input.Periods[id]
		fmt.Fprintf(&b, "period %s data %s states %s\n", period.ID, period.Data, strings.Join(period.States, ","))
	}

	rules := append([]semantic.EffectRule(nil), input.EffectRules...)
	sort.Slice(rules, func(i, j int) bool { return rules[i].ID < rules[j].ID })
	for _, rule := range rules {
		fmt.Fprintf(&b, "effect-rule %s on %s stage %s effect %s %s", rule.ID, rule.OnCapability, rule.Stage, rule.EffectKind, rule.EffectData)
		if rule.ViaCapability != "" {
			fmt.Fprintf(&b, " via %s", rule.ViaCapability)
		}
		if rule.OnFailure != "" {
			fmt.Fprintf(&b, " on-failure %s", rule.OnFailure)
		}
		b.WriteString("\n")
	}

	patterns := append([]application.Pattern(nil), input.Application.Patterns...)
	sort.Slice(patterns, func(i, j int) bool { return patterns[i].ID < patterns[j].ID })
	for _, pattern := range patterns {
		fmt.Fprintf(&b, "pattern %s %s\n", pattern.ID, pattern.Kind)
	}
	if len(patterns) == 0 {
		unresolved = append(unresolved, Unresolved{
			Subject: string(input.Application.ID),
			Reason:  "missing-semantics",
			Detail:  "application declares no pattern, so no function has an entry",
		})
	}
	return b.String(), unresolved
}

func sortedValidations(used []core.CanonicalID, definitions map[core.CanonicalID]semantic.Validation) []semantic.Validation {
	out := make([]semantic.Validation, 0, len(used))
	for _, id := range used {
		if validation, ok := definitions[id]; ok {
			out = append(out, validation)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}
