package applicationview

import (
	"fmt"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/ir"
	"tei/internal/semantic"
)

// CommandPatternViewは、command Patternの能力・入力・結末・効果を解決した結果である。
type CommandPatternView struct {
	SchemaVersion string           `json:"schemaVersion"`
	Application   core.CanonicalID `json:"application"`
	Pattern       core.CanonicalID `json:"pattern"`
	Command
}

// BuildCommandPatternは、command Patternが指すCapabilityを解決する。
func BuildCommandPattern(world ir.World, applicationID, patternID core.CanonicalID) (CommandPatternView, error) {
	var pattern *application.Pattern
	for i := range world.Applications {
		if world.Applications[i].ID != applicationID {
			continue
		}
		for j := range world.Applications[i].Patterns {
			if world.Applications[i].Patterns[j].ID == patternID {
				pattern = &world.Applications[i].Patterns[j]
				break
			}
		}
		break
	}
	if pattern == nil {
		return CommandPatternView{}, fmt.Errorf("pattern %sはapplication %sにありません", patternID, applicationID)
	}
	if pattern.Kind != application.PatternCommand {
		return CommandPatternView{}, fmt.Errorf("pattern %sはcommandではありません: %s", patternID, pattern.Kind)
	}
	var capability *semantic.Capability
	for i := range world.Semantics {
		for j := range world.Semantics[i].Capabilities {
			if world.Semantics[i].Capabilities[j].ID == pattern.Capability {
				capability = &world.Semantics[i].Capabilities[j]
				break
			}
		}
		if capability != nil {
			break
		}
	}
	if capability == nil {
		return CommandPatternView{}, fmt.Errorf("capability %sがありません", pattern.Capability)
	}
	view := CommandPatternView{
		SchemaVersion: SchemaVersion,
		Application:   applicationID,
		Pattern:       patternID,
		Command: Command{
			Capability: capability.ID,
			Label:      capability.Symbol,
		},
	}
	for _, input := range capability.Inputs {
		view.Inputs = append(view.Inputs, CommandInput{Name: string(input.Name), Type: input.Type})
	}
	for _, outcome := range capability.Outcomes {
		view.Outcomes = append(view.Outcomes, CommandOutcome{Name: string(outcome.Name), Payload: outcome.Payload})
	}
	if capability.Effect != nil {
		view.Effect = &CommandEffect{Kind: string(capability.Effect.Kind), Data: capability.Effect.Data}
	}
	return view, nil
}
