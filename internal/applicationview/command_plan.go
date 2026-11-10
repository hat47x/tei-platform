package applicationview

import (
	"fmt"
	"sort"
	"strconv"
	"time"

	"tei/internal/application"
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
	Period        *CommandPeriodRef  `json:"period,omitempty"`
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

// CommandPeriodRefは、Commandの効果が及ぶDataを統制する期間である。
type CommandPeriodRef struct {
	ID            core.CanonicalID `json:"id"`
	Data          core.CanonicalID `json:"data"`
	BusinessDate  core.CanonicalID `json:"businessDate"`
	States        []string         `json:"states,omitempty"`
	ClosedRejects []string         `json:"closedRejects,omitempty"`
	// ClosingDay splits the month into two halves. Zero means the period is the
	// whole month, so a reader sees "no closing" as an absence, not a default.
	ClosingDay int `json:"closingDay,omitempty"`
	// CloseRequires names the capability a period close must run. The declaration
	// was validated (the capability must exist) but never reached the plan, so a
	// period could declare a close capability and close without it. Carrying it
	// here lets the invoker refuse that.
	CloseRequires core.CanonicalID `json:"closeRequires,omitempty"`
	// Correction names the capability a correction after closing must run, for the
	// same reason.
	Correction core.CanonicalID `json:"correction,omitempty"`
}

// CommandPlanIssueは、入力の不足以外の不備を表す。
type CommandPlanIssue struct {
	Input   string `json:"input"`
	Message string `json:"message"`
}

// BuildCommandPlanは、Patternの対話契約にあるCommandへ、呼出し側の値を割り当てる。
// 不足と値の形の不備はMissing・Invalidに残し、Ready=falseにする。
func BuildCommandPlan(world ir.World, applicationID, patternID, capabilityID core.CanonicalID, values map[string]string) (CommandPlan, error) {
	kind, err := PatternKind(world, applicationID, patternID)
	if err != nil {
		return CommandPlan{}, err
	}
	if kind == application.PatternCommand {
		return buildCommandPatternPlan(world, applicationID, patternID, capabilityID, values)
	}
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
	if plan.Effect != nil {
		for _, namespace := range world.Semantics {
			for _, period := range namespace.Periods {
				if period.Data == plan.Effect.Data {
					plan.Period = &CommandPeriodRef{
						ID:            period.ID,
						Data:          period.Data,
						BusinessDate:  period.BusinessDate,
						States:        period.States,
						ClosedRejects: period.ClosedRejects,
						ClosingDay:    period.ClosingDay,
						CloseRequires: period.CloseRequires,
						Correction:    period.Correction,
					}
					break
				}
			}
			if plan.Period != nil {
				break
			}
		}
	}
	sort.Strings(plan.Missing)
	plan.Ready = len(plan.Missing) == 0 && len(plan.Invalid) == 0
	return plan, nil
}

func buildCommandPatternPlan(world ir.World, applicationID, patternID, capabilityID core.CanonicalID, values map[string]string) (CommandPlan, error) {
	var capability *semantic.Capability
	for i := range world.Semantics {
		for j := range world.Semantics[i].Capabilities {
			if world.Semantics[i].Capabilities[j].ID == capabilityID {
				capability = &world.Semantics[i].Capabilities[j]
				break
			}
		}
		if capability != nil {
			break
		}
	}
	if capability == nil {
		return CommandPlan{}, fmt.Errorf("capability %sがありません", capabilityID)
	}
	types := map[core.CanonicalID]semantic.Type{}
	for _, namespace := range world.Semantics {
		for _, typ := range namespace.Types {
			types[typ.ID] = typ
		}
	}
	plan := CommandPlan{
		SchemaVersion: SchemaVersion,
		Application:   applicationID,
		Pattern:       patternID,
		Capability:    capability.ID,
		Label:         capability.Symbol,
	}
	for _, input := range capability.Inputs {
		name := string(input.Name)
		value, provided := values[name]
		plan.Inputs = append(plan.Inputs, CommandPlanInput{Name: name, Type: input.Type, Value: value, Provided: provided})
		if !provided {
			plan.Missing = append(plan.Missing, name)
			continue
		}
		if err := checkCommandValue(input.Type, value, types); err != nil {
			plan.Invalid = append(plan.Invalid, CommandPlanIssue{Input: name, Message: err.Error()})
		}
	}
	for _, outcome := range capability.Outcomes {
		plan.Outcomes = append(plan.Outcomes, CommandOutcome{Name: string(outcome.Name), Payload: outcome.Payload})
	}
	if capability.Effect != nil {
		plan.Effect = &CommandEffect{Kind: string(capability.Effect.Kind), Data: capability.Effect.Data}
		plan.Period = periodForEffect(world, capability.Effect.Data)
	}
	sort.Strings(plan.Missing)
	plan.Ready = len(plan.Missing) == 0 && len(plan.Invalid) == 0
	return plan, nil
}

func periodForEffect(world ir.World, data core.CanonicalID) *CommandPeriodRef {
	for _, namespace := range world.Semantics {
		for _, period := range namespace.Periods {
			if period.Data == data {
				return &CommandPeriodRef{
					ID:            period.ID,
					Data:          period.Data,
					BusinessDate:  period.BusinessDate,
					States:        period.States,
					ClosedRejects: period.ClosedRejects,
					ClosingDay:    period.ClosingDay,
					CloseRequires: period.CloseRequires,
					Correction:    period.Correction,
				}
			}
		}
	}
	return nil
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
