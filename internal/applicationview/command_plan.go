package applicationview

import (
	"fmt"
	"sort"

	"tei/internal/core"
	"tei/internal/ir"
)

// CommandPlanは、対話からCommandを起動する前の入力を解決した結果である。
// 実行は行わず、宣言された入力・結末・効果と、値の過不足だけを示す。
type CommandPlan struct {
	SchemaVersion string             `json:"schemaVersion"`
	Application   core.CanonicalID   `json:"application"`
	Pattern       core.CanonicalID   `json:"pattern"`
	Capability    core.CanonicalID   `json:"capability"`
	Label         string             `json:"label,omitempty"`
	Inputs        []CommandPlanInput `json:"inputs,omitempty"`
	Missing       []string           `json:"missing,omitempty"`
	Outcomes      []CommandOutcome   `json:"outcomes,omitempty"`
	Effect        *CommandEffect     `json:"effect,omitempty"`
	Ready         bool               `json:"ready"`
}

// CommandPlanInputは、Commandの入力と、呼出し側が渡した値である。
type CommandPlanInput struct {
	Name      string           `json:"name"`
	Type      core.CanonicalID `json:"type"`
	Attribute core.CanonicalID `json:"attribute,omitempty"`
	Label     string           `json:"label,omitempty"`
	Value     string           `json:"value,omitempty"`
	Provided  bool             `json:"provided"`
}

// BuildCommandPlanは、Patternの対話契約にあるCommandへ、呼出し側の値を割り当てる。
// 不足はMissingに残し、Ready=falseにする。型の解釈は行わない。
func BuildCommandPlan(world ir.World, applicationID, patternID, capabilityID core.CanonicalID, values map[string]string) (CommandPlan, error) {
	view, err := Build(world, applicationID, patternID)
	if err != nil {
		return CommandPlan{}, err
	}
	var command *Command
	for i := range view.Commands {
		if view.Commands[i].Capability == capabilityID {
			command = &view.Commands[i]
			break
		}
	}
	if command == nil {
		return CommandPlan{}, fmt.Errorf("pattern %sはcommand %sを宣言していません", patternID, capabilityID)
	}
	plan := CommandPlan{
		SchemaVersion: SchemaVersion,
		Application:   view.Application,
		Pattern:       view.Pattern,
		Capability:    command.Capability,
		Label:         command.Label,
		Outcomes:      command.Outcomes,
		Effect:        command.Effect,
	}
	for _, input := range command.Inputs {
		value, provided := values[input.Name]
		plan.Inputs = append(plan.Inputs, CommandPlanInput{
			Name:      input.Name,
			Type:      input.Type,
			Attribute: input.Attribute,
			Label:     input.Label,
			Value:     value,
			Provided:  provided,
		})
		if !provided {
			plan.Missing = append(plan.Missing, input.Name)
		}
	}
	sort.Strings(plan.Missing)
	plan.Ready = len(plan.Missing) == 0
	return plan, nil
}
