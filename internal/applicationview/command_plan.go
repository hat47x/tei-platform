package applicationview

import (
	"fmt"
	"sort"
	"strconv"
	"time"

	"tei/internal/core"
	"tei/internal/ir"
	"tei/internal/semantic"
)

// CommandPlanは、対話からCommandを起動する前の入力を解決した結果である。
// 実行は行わず、宣言された入力・結末・効果と、値の過不足・形だけを示す。
type CommandPlan struct {
	SchemaVersion string             `json:"schemaVersion"`
	Application   core.CanonicalID   `json:"application"`
	Pattern       core.CanonicalID   `json:"pattern"`
	Capability    core.CanonicalID   `json:"capability"`
	Label         string             `json:"label,omitempty"`
	Inputs        []CommandPlanInput `json:"inputs,omitempty"`
	Missing       []string           `json:"missing,omitempty"`
	Invalid       []CommandPlanIssue `json:"invalid,omitempty"`
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

// CommandPlanIssueは、入力の不足以外の不備を表す。
type CommandPlanIssue struct {
	Input   string `json:"input"`
	Message string `json:"message"`
}

// BuildCommandPlanは、Patternの対話契約にあるCommandへ、呼出し側の値を割り当てる。
// 不足と値の形の不備はMissing・Invalidに残し、Ready=falseにする。
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
	types := map[core.CanonicalID]semantic.Type{}
	for _, namespace := range world.Semantics {
		for _, typ := range namespace.Types {
			types[typ.ID] = typ
		}
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
			continue
		}
		if err := checkCommandValue(input.Type, value, types); err != nil {
			plan.Invalid = append(plan.Invalid, CommandPlanIssue{Input: input.Name, Message: err.Error()})
		}
	}
	sort.Strings(plan.Missing)
	plan.Ready = len(plan.Missing) == 0 && len(plan.Invalid) == 0
	return plan, nil
}

func checkCommandValue(typeRef core.CanonicalID, value string, types map[core.CanonicalID]semantic.Type) error {
	primitive := typeRef
	for depth := 0; depth < 16; depth++ {
		typ, ok := types[primitive]
		if !ok || typ.Kind != semantic.TypeAlias || typ.Underlying == "" {
			break
		}
		primitive = typ.Underlying
	}
	switch primitive {
	case "std.integer":
		if _, err := strconv.ParseInt(value, 10, 64); err != nil {
			return fmt.Errorf("%sは整数ではありません", value)
		}
	case "std.decimal", "std.money":
		if _, err := strconv.ParseFloat(value, 64); err != nil {
			return fmt.Errorf("%sは数値ではありません", value)
		}
	case "std.boolean":
		if value != "true" && value != "false" {
			return fmt.Errorf("%sはtrue/falseではありません", value)
		}
	case "std.date":
		if _, err := time.Parse("2006-01-02", value); err != nil {
			return fmt.Errorf("%sはYYYY-MM-DDではありません", value)
		}
	case "std.datetime":
		if _, err := time.Parse(time.RFC3339, value); err != nil {
			if _, err := time.Parse("2006-01-02T15:04", value); err != nil {
				return fmt.Errorf("%sは日時ではありません", value)
			}
		}
	case "std.time":
		if _, err := time.Parse("15:04", value); err != nil {
			return fmt.Errorf("%sはHH:MMではありません", value)
		}
	}
	return nil
}
