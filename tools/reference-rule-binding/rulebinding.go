// Package rulebinding is a read-only, synthetic reference for Data Derivation
// D4 (#1692). It checks the smallest contract for resolving an accepted
// derivation rule to its application target: reuse of one rule across several
// targets, conflicting bindings, case-specific overrides that only replace an
// implementation, and the negative cases (same name in another namespace, type
// mismatch, unaccepted rule, unregistered plugin).
//
// It is not a general expression language, an automatic priority guesser, a
// plugin compatibility registry, a second Resolver, or an accepted API. It does
// not write a Source. It never guesses a binding from a physical column name,
// a same-named symbol, a file path, or a registration order.
package rulebinding

import (
	"fmt"
	"strings"
)

// Result codes.
const (
	CodeResolved                     = "RESOLVED"
	CodeUnbound                      = "UNBOUND"
	CodeAmbiguous                    = "AMBIGUOUS"
	CodeTargetUnknown                = "TARGET_UNKNOWN"
	CodeRuleUnknownBinding           = "RULE_UNKNOWN_BINDING"
	CodeSameNameDifferentID          = "SAME_NAME_DIFFERENT_ID"
	CodeRuleNotAccepted              = "RULE_NOT_ACCEPTED"
	CodeTypeMismatch                 = "TYPE_MISMATCH"
	CodePluginNotRegistered          = "PLUGIN_NOT_REGISTERED"
	CodeImplementationReplaced       = "IMPLEMENTATION_REPLACED"
	CodeContractChangeRequiresAccept = "CONTRACT_CHANGE_REQUIRES_ACCEPTANCE"
	CodeInvalidRequest               = "INVALID_REQUEST"
)

// Override kinds.
const (
	OverrideImplementation = "implementation"
	OverrideContractChange = "contract-change"
)

// Rule is an accepted derivation rule with a typed semantic identity.
type Rule struct {
	ID         string `json:"id"`
	Version    string `json:"version"`
	Accepted   bool   `json:"accepted"`
	Type       string `json:"type"`
	NullPolicy string `json:"nullPolicy,omitempty"`
}

// Target is a typed application target.
type Target struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}

// Binding connects a rule to a target. Priority and Replace are explicit
// declarations; a zero priority and false Replace mean none.
type Binding struct {
	RuleID   string `json:"ruleId"`
	TargetID string `json:"targetId"`
	Priority int    `json:"priority,omitempty"`
	Replace  bool   `json:"replace,omitempty"`
	Plugin   string `json:"plugin,omitempty"`
}

// Override is a case-specific implementation replacement for a target.
type Override struct {
	TargetID string `json:"targetId"`
	RuleID   string `json:"ruleId"`
	Kind     string `json:"kind"`
	FromImpl string `json:"fromImpl,omitempty"`
	ToImpl   string `json:"toImpl,omitempty"`
}

// Input is the whole synthetic request.
type Input struct {
	Rules             []Rule     `json:"rules"`
	Targets           []Target   `json:"targets"`
	Bindings          []Binding  `json:"bindings"`
	Overrides         []Override `json:"overrides,omitempty"`
	RegisteredPlugins []string   `json:"registeredPlugins,omitempty"`
	ResolveTarget     string     `json:"resolveTarget"`
}

// Report is the read-only result.
type Report struct {
	Code            string   `json:"code"`
	Accepted        bool     `json:"accepted"`
	TargetID        string   `json:"targetId,omitempty"`
	RuleID          string   `json:"ruleId,omitempty"`
	RuleVersion     string   `json:"ruleVersion,omitempty"`
	Implementations []string `json:"implementations,omitempty"`
	Message         string   `json:"message,omitempty"`
}

func reject(code string, message string) Report {
	return Report{Code: code, Message: message}
}

func findTarget(targets []Target, id string) (Target, bool) {
	for _, target := range targets {
		if target.ID == id {
			return target, true
		}
	}
	return Target{}, false
}

func findRule(rules []Rule, id string) (Rule, bool) {
	for _, rule := range rules {
		if rule.ID == id {
			return rule, true
		}
	}
	return Rule{}, false
}

func symbol(id string) string {
	if index := strings.LastIndex(id, "."); index >= 0 {
		return id[index+1:]
	}
	return id
}

func pluginRegistered(plugins []string, name string) bool {
	for _, plugin := range plugins {
		if plugin == name {
			return true
		}
	}
	return false
}

// candidate is a rule bound to the resolved target.
type candidate struct {
	rule Rule
	bind Binding
}

// Resolve checks one target resolution. It never writes anything.
func Resolve(in Input) Report {
	target, ok := findTarget(in.Targets, in.ResolveTarget)
	if !ok {
		return reject(CodeTargetUnknown, fmt.Sprintf("適用先 %q が未登録。", in.ResolveTarget))
	}

	// Contract-changing case override is a re-acceptance, not a resolution.
	for _, override := range in.Overrides {
		if override.TargetID == in.ResolveTarget && override.Kind == OverrideContractChange {
			return reject(CodeContractChangeRequiresAccept,
				"NULL方針や型を変える候補は実装差替えではない。契約変更として再受理を要求する。")
		}
	}

	var candidates []candidate
	for _, bind := range in.Bindings {
		if bind.TargetID != in.ResolveTarget {
			continue
		}
		if bind.Plugin != "" && !pluginRegistered(in.RegisteredPlugins, bind.Plugin) {
			return reject(CodePluginNotRegistered, fmt.Sprintf("未登録のPlugin %q のBindingは自動受理しない。", bind.Plugin))
		}
		rule, found := findRule(in.Rules, bind.RuleID)
		if !found {
			// A same-named symbol in another namespace is not the same identity.
			for _, other := range in.Rules {
				if symbol(other.ID) == symbol(bind.RuleID) && other.ID != bind.RuleID {
					return reject(CodeSameNameDifferentID,
						fmt.Sprintf("同名Symbol %q は別ID %q。名前空間を跨いだ暗黙推定をしない。", symbol(bind.RuleID), other.ID))
				}
			}
			return reject(CodeRuleUnknownBinding, fmt.Sprintf("Binding先のRule %q が未登録。", bind.RuleID))
		}
		if !rule.Accepted {
			return reject(CodeRuleNotAccepted, fmt.Sprintf("Rule %q は未受理。", bind.RuleID))
		}
		if rule.Type != target.Type {
			return reject(CodeTypeMismatch,
				fmt.Sprintf("Rule %q の型 %q が適用先 %q の型 %q と不一致。", rule.ID, rule.Type, target.ID, target.Type))
		}
		candidates = append(candidates, candidate{rule: rule, bind: bind})
	}

	if len(candidates) == 0 {
		return reject(CodeUnbound, fmt.Sprintf("適用先 %q にBindingがない。", in.ResolveTarget))
	}

	chosen := candidates[0]
	if len(candidates) > 1 {
		chosen, ok = pickExplicit(candidates)
		if !ok {
			return reject(CodeAmbiguous,
				fmt.Sprintf("適用先 %q に複数の規則が到達した。明示の優先／置換がないため登録順で先勝ちにしない。", in.ResolveTarget))
		}
	}

	report := Report{
		Code:        CodeResolved,
		Accepted:    true,
		TargetID:    target.ID,
		RuleID:      chosen.rule.ID,
		RuleVersion: chosen.rule.Version,
	}
	for _, override := range in.Overrides {
		if override.TargetID == in.ResolveTarget && override.Kind == OverrideImplementation {
			report.Code = CodeImplementationReplaced
			if override.FromImpl != "" {
				report.Implementations = append(report.Implementations, override.FromImpl)
			}
			if override.ToImpl != "" {
				report.Implementations = append(report.Implementations, override.ToImpl)
			}
			report.Message = "実装だけを差替える。Rule ID・適用先・意味は不変。"
		}
	}
	return report
}

// pickExplicit chooses one candidate using only explicit declarations.
func pickExplicit(candidates []candidate) (candidate, bool) {
	// Unique highest explicit priority wins.
	best := -1
	count := 0
	index := -1
	for i, candidate := range candidates {
		if candidate.bind.Priority > best {
			best = candidate.bind.Priority
			count = 1
			index = i
		} else if candidate.bind.Priority == best {
			count++
		}
	}
	if best > 0 && count == 1 {
		return candidates[index], true
	}
	// Otherwise a unique explicit replacement wins.
	replaceIndex := -1
	replaceCount := 0
	for i, candidate := range candidates {
		if candidate.bind.Replace {
			replaceCount++
			replaceIndex = i
		}
	}
	if replaceCount == 1 {
		return candidates[replaceIndex], true
	}
	return candidates[0], false
}
