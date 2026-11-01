package rulebinding

import (
	"fmt"
	"sort"
)

// Impact schema and binding statuses after a rule change. A binding that still
// resolves to the same rule but whose rule changed is never reported as
// unchanged: it is either unresolved or a candidate for re-acceptance or
// re-validation.
const (
	ImpactSchema = "tei.reference.rule-binding-impact/v0"

	BindingUnresolved    = "BINDING_UNRESOLVED"
	BindingReacceptance  = "REACCEPTANCE_REQUIRED"
	BindingRevalidation  = "REVALIDATION_CANDIDATE"
	BindingUnchanged     = "BINDING_UNCHANGED"
	BindingTargetMissing = "TARGET_MISSING"
)

// Kind of change reported against a rule.
const (
	ChangeRemoved        = "removed"
	ChangeTypeChanged    = "type-changed"
	ChangeContractChange = "contract-changed"
	ChangeVersionChanged = "version-changed"
)

// RuleChange is one fact about a rule that may affect its bindings. NewType and
// NewVersion are optional details; an unknown change kind is treated as a
// contract change so it is never silently ignored.
type RuleChange struct {
	RuleID     string `json:"ruleId"`
	Change     string `json:"change"`
	NewType    string `json:"newType,omitempty"`
	NewVersion string `json:"newVersion,omitempty"`
}

// ImpactInput is the read-only input for PlanImpact. It reuses Target and
// Binding from the resolution reference and adds the rule changes.
type ImpactInput struct {
	Targets  []Target     `json:"targets"`
	Bindings []Binding    `json:"bindings"`
	Changes  []RuleChange `json:"changes"`
}

// BindingImpact is what one rule change does to one binding.
type BindingImpact struct {
	RuleID   string `json:"ruleId"`
	TargetID string `json:"targetId"`
	Status   string `json:"status"`
	Reason   string `json:"reason,omitempty"`
}

// ImpactSummary counts each binding status.
type ImpactSummary struct {
	Unresolved    int `json:"unresolved"`
	Reacceptance  int `json:"reacceptance"`
	Revalidation  int `json:"revalidation"`
	Unchanged     int `json:"unchanged"`
	TargetMissing int `json:"targetMissing"`
}

// ImpactReport is the plan. Accepted is true only when every binding is
// unchanged; a rule change is never treated as an automatic re-binding.
type ImpactReport struct {
	Schema   string          `json:"schema"`
	Impacts  []BindingImpact `json:"impacts"`
	Summary  ImpactSummary   `json:"summary"`
	Accepted bool            `json:"accepted"`
	Applied  bool            `json:"applied"`
	Note     string          `json:"note"`
}

// PlanImpact lists the bindings affected by rule removal, type change, contract
// change, or version change. It never rewrites a binding and never creates a
// second impact graph; the caller decides which candidate to re-accept.
func PlanImpact(in ImpactInput) ImpactReport {
	report := ImpactReport{
		Schema:  ImpactSchema,
		Impacts: []BindingImpact{},
		Applied: false,
		Note:    "影響の一覧です。Bindingを書き換えません。再受理・再検証の判断は既存の受理境界で行います。",
	}

	changes := make(map[string]RuleChange, len(in.Changes))
	for _, change := range in.Changes {
		changes[change.RuleID] = change
	}

	for _, binding := range in.Bindings {
		impact := BindingImpact{RuleID: binding.RuleID, TargetID: binding.TargetID}
		target, targetKnown := findTarget(in.Targets, binding.TargetID)
		change, ruleChanged := changes[binding.RuleID]

		switch {
		case !targetKnown:
			impact.Status = BindingTargetMissing
			impact.Reason = fmt.Sprintf("適用先 %q が未登録。", binding.TargetID)
		case !ruleChanged:
			impact.Status = BindingUnchanged
		default:
			switch change.Change {
			case ChangeRemoved:
				impact.Status = BindingUnresolved
				impact.Reason = fmt.Sprintf("Rule %q が削除された。Bindingは未解決になる。", binding.RuleID)
			case ChangeTypeChanged:
				impact.Status = BindingReacceptance
				impact.Reason = fmt.Sprintf("Rule %q の型が変わった。適用先 %q の型 %q で再受理が必要。", binding.RuleID, target.ID, target.Type)
				if change.NewType != "" && change.NewType != target.Type {
					impact.Reason = fmt.Sprintf("Rule %q の新しい型 %q が適用先 %q の型 %q と不一致。再受理なしに解決しない。",
						binding.RuleID, change.NewType, target.ID, target.Type)
				}
			case ChangeContractChange:
				impact.Status = BindingReacceptance
				impact.Reason = fmt.Sprintf("Rule %q の意味契約が変わった。実装差替えではなく再受理が必要。", binding.RuleID)
			case ChangeVersionChanged:
				impact.Status = BindingRevalidation
				impact.Reason = fmt.Sprintf("Rule %q の版が %q へ変わった。Bindingは解決するが再検証の候補。", binding.RuleID, change.NewVersion)
			default:
				impact.Status = BindingReacceptance
				impact.Reason = fmt.Sprintf("Rule %q の未知の変更種別 %q。黙って無視せず再受理を要求する。", binding.RuleID, change.Change)
			}
		}

		report.Impacts = append(report.Impacts, impact)
		switch impact.Status {
		case BindingUnresolved:
			report.Summary.Unresolved++
		case BindingReacceptance:
			report.Summary.Reacceptance++
		case BindingRevalidation:
			report.Summary.Revalidation++
		case BindingTargetMissing:
			report.Summary.TargetMissing++
		default:
			report.Summary.Unchanged++
		}
	}

	sort.Slice(report.Impacts, func(i, j int) bool {
		if report.Impacts[i].RuleID == report.Impacts[j].RuleID {
			return report.Impacts[i].TargetID < report.Impacts[j].TargetID
		}
		return report.Impacts[i].RuleID < report.Impacts[j].RuleID
	})

	report.Accepted = report.Summary.Unresolved == 0 &&
		report.Summary.Reacceptance == 0 &&
		report.Summary.Revalidation == 0 &&
		report.Summary.TargetMissing == 0
	return report
}
