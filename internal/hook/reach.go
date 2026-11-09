package hook

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"tei/internal/application"
	"tei/internal/core"
)

// Reach は、ある実行境界が位相へ到達できるかを表す。
//
// 参照実行（CLI・Studio）は何も書かないため after-* の時点が無い。実行境界が
// どこまで進めるかを型で言わせることで、「実行しなかった位相」を「合格した位相」
// と取り違えないようにする。
type Reach interface {
	// UnreachableReason は、位相を実行できない理由を返す。到達できる位相には
	// 空文字を返す。
	UnreachableReason(phase string) string
}

// ReferenceReach は、参照実行（データを書かない）の到達範囲である。
// before-* へは到達でき、after-* へは到達できない。
type ReferenceReach struct{}

// UnreachableReason は、after-* の位相に到達しない理由を返す。
func (ReferenceReach) UnreachableReason(phase string) string {
	if isBeforeChange(phase) {
		return ""
	}
	return ReasonAfterChangeUnreachable
}

// ReasonAfterChangeUnreachable は、参照実行が after-* に到達しない理由である。
const ReasonAfterChangeUnreachable = "the reference run writes nothing, so there is no after-change point to run this phase at"

// ReasonReadNotSupplied は、宣言した読み取りに値が無いために位相を評価しなかった
// 理由の前置きである。実際の理由は属性名を続けて返す。
const ReasonReadNotSupplied = "no value was supplied for the declared read, so this phase was not evaluated"

// ReasonPhaseUnsupported は、位相順序が位相を名指さないために評価しなかった理由である。
const ReasonPhaseUnsupported = "the runner order does not name this phase, so it was not evaluated"

// PhaseReport は、位相をどこまで実行したかと、実行しなかった位相の理由である。
//
// Skip と NotEvaluated は別である。Skip はその実行境界が原理的に到達しない位相
// （参照実行の after-* など）、NotEvaluated は到達しうるが前提が欠けた位相である。
// どちらも「合格」ではないことを読み手が区別できるようにする。
type PhaseReport struct {
	// Executed は、成功した位相である。実行順に並ぶ。
	Executed []string `json:"executed,omitempty"`
	// Warned は、on-failure warn で失敗したが続行した位相である。
	Warned []string `json:"warned,omitempty"`
	// Skipped は、この実行境界が到達しない位相である。
	Skipped []string `json:"skipped,omitempty"`
	// NotEvaluated は、到達しうるが前提が欠けて評価しなかった位相である。
	NotEvaluated []string `json:"notEvaluated,omitempty"`
	// Reasons は、実行しなかった位相の理由である。
	Reasons map[string]string `json:"reasons,omitempty"`
	// Refused は、不成立または実行の失敗で止まった位相である。
	Refused string `json:"refused,omitempty"`
	// Outcome は、不成立に対応する業務上の結末である。実行の失敗では空になる。
	Outcome string `json:"outcome,omitempty"`
}

// Ran は、実行した位相があるかを返す。すべての位相を飛ばした実行と、位相を
// 一つも宣言していない Pattern を区別するために使う。
func (r PhaseReport) Ran() bool {
	return len(r.Executed) > 0 || len(r.Warned) > 0 || r.Refused != ""
}

// RunReachable は、宣言された位相のうち、この実行境界が到達でき、かつ宣言した
// 読み取りの値がすべて揃っている位相だけを `PhaseOrder` の順に実行する。
//
// inputs は、位相ごとの宣言が指す属性名から値を引くためのものである。宣言して
// いない属性が inputs にあっても、その位相へは渡さない（`ReadsFor` が宣言外を
// 拒否するため、渡せば失敗になる）。
//
// 不成立または実行の失敗に当たった位相で止まり、そこまでの記録を返す。`on-failure
// warn` の位相は記録して先へ進む。返すエラーは止まった位相の失敗そのものである。
// Pattern が位相を宣言していなければ `nil, nil` を返す（フックが無い実行を、
// 「フックを見た実行」と読ませない）。
func (r Runner) RunReachable(ctx context.Context, applicationID, patternID core.CanonicalID, inputs map[core.CanonicalID]string, reach Reach) (*PhaseReport, error) {
	pattern := r.findPattern(applicationID, patternID)
	// A pattern that is not a record pattern with an interaction (for example a
	// command pattern) cannot declare hooks. That is not a failure of the
	// caller's run: there is simply nothing to consult, so report nothing
	// rather than refuse a command the caller legitimately planned.
	if pattern == nil || pattern.Interaction == nil || len(pattern.Interaction.Hooks) == 0 {
		return nil, nil
	}
	declared := map[string]*application.Hook{}
	for i := range pattern.Interaction.Hooks {
		phase := pattern.Interaction.Hooks[i].Phase
		if _, exists := declared[phase]; exists {
			// 同位相の重複宣言は検証が拒否する。ここでは先の宣言を採り、
			// 二つの間に順序を発明しない。
			continue
		}
		declared[phase] = &pattern.Interaction.Hooks[i]
	}
	report := &PhaseReport{Reasons: map[string]string{}}
	for _, phase := range PhaseOrder {
		declaration, declaredHere := declared[phase]
		if !declaredHere {
			continue
		}
		if reason := reach.UnreachableReason(phase); reason != "" {
			report.Skipped = append(report.Skipped, phase)
			report.Reasons[phase] = reason
			continue
		}
		reads, missing := readsFromInputs(declaration.Reads, inputs)
		if missing != "" {
			report.Skipped = append(report.Skipped, phase)
			report.NotEvaluated = append(report.NotEvaluated, phase)
			report.Reasons[phase] = fmt.Sprintf("%s: %s", ReasonReadNotSupplied, missing)
			continue
		}
		invocation := Invocation{
			Application: applicationID,
			Pattern:     patternID,
			Phase:       phase,
			Reads:       reads,
		}
		err := r.RunWithDeclaredReads(ctx, invocation)
		if err == nil {
			report.Executed = append(report.Executed, phase)
			continue
		}
		var warning *Warning
		if errors.As(err, &warning) {
			report.Warned = append(report.Warned, phase)
			continue
		}
		report.Refused = phase
		var notSatisfied *NotSatisfied
		if errors.As(err, &notSatisfied) {
			report.Outcome = notSatisfied.Outcome
		}
		return report, err
	}
	// 順序が名指さない位相は、実行も記録もしないまま消えてしまう。黙って落とさず
	// 記録する。
	unknown := make([]string, 0, len(declared))
	for phase := range declared {
		if knownPhase(phase) {
			continue
		}
		unknown = append(unknown, phase)
	}
	sort.Strings(unknown)
	for _, phase := range unknown {
		report.Skipped = append(report.Skipped, phase)
		report.Reasons[phase] = ReasonPhaseUnsupported
	}
	return report, nil
}

// findPattern は、ApplicationのPatternを宣言の種類を問わず解決する。フックを
// 宣言できない Pattern は、フックの境界では「何も無い」であって失敗ではない。
func (r Runner) findPattern(applicationID, patternID core.CanonicalID) *application.Pattern {
	for i := range r.World.Applications {
		if r.World.Applications[i].ID != applicationID {
			continue
		}
		for j := range r.World.Applications[i].Patterns {
			if r.World.Applications[i].Patterns[j].ID == patternID {
				return &r.World.Applications[i].Patterns[j]
			}
		}
	}
	return nil
}

// readsFromInputs は、宣言した属性の値を inputs から引く。値の無い最初の属性を
// 返し、すべて揃っていれば空文字を返す。
func readsFromInputs(declared []core.CanonicalID, inputs map[core.CanonicalID]string) (map[core.CanonicalID]string, string) {
	reads := make(map[core.CanonicalID]string, len(declared))
	for _, attribute := range declared {
		value, ok := inputs[attribute]
		if !ok {
			return nil, string(attribute)
		}
		reads[attribute] = value
	}
	return reads, ""
}

// isBeforeChange は、その位相が変更の前かを返す。
func isBeforeChange(phase string) bool {
	switch phase {
	case "before-read", "before-create", "before-update", "before-delete":
		return true
	}
	return false
}
