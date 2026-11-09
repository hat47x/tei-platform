package studiohost

import (
	"context"
	"errors"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/hook"
	"tei/internal/ir"
)

// HookRunReport records the pattern hooks the Studio reference execution
// consulted, and which of them it could not evaluate.
//
// It mirrors the CLI report (cmd/tei/cli_view_hooks.go): the Studio reference
// host also has no data source, so a hook whose declared read the caller did
// not supply is "not evaluated" with a reason, not a pass.
type HookRunReport struct {
	Pattern      core.CanonicalID  `json:"pattern"`
	Executed     []string          `json:"executed,omitempty"`
	Warned       []string          `json:"warned,omitempty"`
	Skipped      []string          `json:"skipped,omitempty"`
	NotEvaluated []string          `json:"notEvaluated,omitempty"`
	Reasons      map[string]string `json:"reasons,omitempty"`
	Refused      string            `json:"refused,omitempty"`
	Outcome      string            `json:"outcome,omitempty"`
	Note         string            `json:"note"`
}

const hookRunNote = "the reference rule evaluates the declared reads only; running it does not accept the rule, observe the command effect, or grant authority"

const (
	hookSkipAfterChange = "the reference execution writes nothing, so there is no after-change point to run this phase at"
	hookSkipNoSource    = "the reference execution has no data source for the declared read, so this phase was not evaluated"
	hookSkipUnsupported = "the runner order does not name this phase, so it was not evaluated"
)

// runHooks runs the hooks a pattern declares that the reference execution can
// reach, before the reference command runs.
func (h *ReadOnly) runHooks(world ir.World, applicationID, patternID core.CanonicalID, inputs map[string]string) (*HookRunReport, error) {
	declarations := studiohookDeclarations(world, applicationID, patternID)
	if len(declarations) == 0 {
		return nil, nil
	}
	report := &HookRunReport{
		Pattern: patternID,
		Note:    hookRunNote,
		Reasons: map[string]string{},
	}
	runner := hook.Runner{World: world, Implementation: hook.ReferenceRule{}}
	for _, phase := range hook.PhaseOrder {
		declaration, declared := declarations[phase]
		if !declared {
			continue
		}
		if !studiohookBeforeChange(phase) {
			report.Skipped = append(report.Skipped, phase)
			report.Reasons[phase] = hookSkipAfterChange
			continue
		}
		reads, missing := studiohookReads(declaration.Reads, inputs)
		if missing != "" {
			report.Skipped = append(report.Skipped, phase)
			report.NotEvaluated = append(report.NotEvaluated, phase)
			report.Reasons[phase] = hookSkipNoSource + ": " + missing
			continue
		}
		err := runner.RunWithDeclaredReads(context.Background(), hook.Invocation{
			Application: applicationID,
			Pattern:     patternID,
			Phase:       phase,
			Reads:       reads,
		})
		if err == nil {
			report.Executed = append(report.Executed, phase)
			continue
		}
		var warning *hook.Warning
		if errors.As(err, &warning) {
			report.Warned = append(report.Warned, phase)
			continue
		}
		report.Refused = phase
		var notSatisfied *hook.NotSatisfied
		if errors.As(err, &notSatisfied) {
			report.Outcome = notSatisfied.Outcome
		}
		return report, err
	}
	for phase := range declarations {
		if studiohookKnownPhase(phase) {
			continue
		}
		report.Skipped = append(report.Skipped, phase)
		report.Reasons[phase] = hookSkipUnsupported
	}
	return report, nil
}

// studiohookDeclarations returns a pattern's hook declarations by phase.
func studiohookDeclarations(world ir.World, applicationID, patternID core.CanonicalID) map[string]*application.Hook {
	for i := range world.Applications {
		if world.Applications[i].ID != applicationID {
			continue
		}
		for _, pattern := range world.Applications[i].Patterns {
			if pattern.ID != patternID || pattern.Interaction == nil {
				continue
			}
			out := make(map[string]*application.Hook, len(pattern.Interaction.Hooks))
			for j := range pattern.Interaction.Hooks {
				phase := pattern.Interaction.Hooks[j].Phase
				if _, exists := out[phase]; exists {
					continue
				}
				out[phase] = &pattern.Interaction.Hooks[j]
			}
			return out
		}
	}
	return nil
}

func studiohookReads(declared []core.CanonicalID, inputs map[string]string) (map[core.CanonicalID]string, string) {
	values := make(map[core.CanonicalID]string, len(declared))
	for _, attribute := range declared {
		value, ok := inputs[string(attribute)]
		if !ok {
			return nil, string(attribute)
		}
		values[attribute] = value
	}
	return values, ""
}

func studiohookBeforeChange(phase string) bool {
	switch phase {
	case "before-read", "before-create", "before-update", "before-delete":
		return true
	}
	return false
}

func studiohookKnownPhase(phase string) bool {
	for _, known := range hook.PhaseOrder {
		if known == phase {
			return true
		}
	}
	return false
}
