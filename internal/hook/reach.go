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

// ReasonCallerUnavailable は、フックが能力を宣言したのに呼出し元が無いため、
// その位相を評価しなかった理由である。
//
// 宣言に`uses`があり呼出し元が無い場合、その能力は一度も呼ばれません。それでも
// `executed`と報告すると、「宣言した能力を呼んだ」と読めてしまいます。呼んで
// いないものを呼んだと書かないために、この理由で未評価にします。
const ReasonCallerUnavailable = "the hook declares a capability to call but no caller is configured, so this phase was not evaluated"

// ReasonDeclaredCallNotMade は、実装が呼出し元を使わない経路で実行されたため、
// 宣言した能力が呼ばれなかったことを示す理由である。
const ReasonDeclaredCallNotMade = "the hook declares a capability to call but the implementation does not call it, so this phase was not evaluated"

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
	// Produced は、実行した位相が提案した値である。位相ごとに、宣言した
	// `produces`に載る属性だけが入る。提案は値を確定しない。呼出し側が採用を
	// 決めるまでの候補として、ここに残す。
	//
	// この欄が無かったころは、`produces`を宣言したフックが値を返しても、
	// `RunReachable`は`RunWithDeclaredReads`を呼ぶため提案が捨てられ、
	// `hook.ProducesFor`の宣言照合も走らなかった。宣言と実行が食い違う状態を
	// 黙って通さないために欄を足す。
	Produced map[string]Produced `json:"produced,omitempty"`
	// Uncalled names, by phase, the capabilities a hook declared in `uses` that
	// were not called. A phase with an uncalled declaration is reported as not
	// evaluated, so `executed` never claims a call that did not happen.
	Uncalled map[string][]core.CanonicalID `json:"uncalled,omitempty"`
	// Calls records, by phase, the calls the declaration made and how each
	// ended. A phase listed here did reach the boundary.
	Calls map[string][]CallOutcome `json:"calls,omitempty"`
}

// Ran は、実行した位相があるかを返す。すべての位相を飛ばした実行と、位相を
// 一つも宣言していない Pattern を区別するために使う。
func (r PhaseReport) Ran() bool {
	return len(r.Executed) > 0 || len(r.Warned) > 0 || r.Refused != ""
}

// Proposed は、位相が提案した値を返す。提案が無ければ空を返す。
func (r PhaseReport) Proposed(phase string) Produced {
	return r.Produced[phase]
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
		// A scope this runner cannot apply is reported as skipped rather than run
		// as if the scope did not matter: this runner runs one hook per record, so
		// only the record scope and the pattern's own unit are applied. Reporting a
		// reason keeps the phase visible instead of silently passing it.
		if scope := declaration.Scope; scope != "" && scope != string(application.PatternRecord) {
			// A row-scoped hook is applied per row when the runner was given a row
			// set. Without one it is reported as needing the rows, never run once
			// for the record as if the scope did not matter.
			if scope == "row" && len(r.Rows) > 0 {
				if err := r.applyReachableRows(ctx, report, applicationID, patternID, phase, declaration); err != nil {
					return report, err
				}
				continue
			}
			report.Skipped = append(report.Skipped, phase)
			if scope == "row" {
				report.Reasons[phase] = "the hook declares scope row, so it needs the row set; pass the rows to apply it"
			} else {
				report.Reasons[phase] = fmt.Sprintf("the hook declares scope %s but this runner applies one hook per record", scope)
			}
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
		// A hook that declares `uses` promises to call those capabilities. Call
		// them here, through the restricted caller the Runner holds, so the
		// promise is kept or reported as a refusal rather than assumed.
		if len(declaration.Uses) > 0 {
			// A boundary with no caller behind it cannot keep the promise. Record
			// the phases as not evaluated, with the capabilities named, instead of
			// refusing a run the host legitimately started.
			if r.Caller == nil {
				report.Skipped = append(report.Skipped, phase)
				report.NotEvaluated = append(report.NotEvaluated, phase)
				report.Uncalled = appendUncalled(report.Uncalled, phase, declaration.Uses)
				report.Reasons[phase] = ReasonCallerUnavailable
				continue
			}
			// The inputs a capability receives come from this phase's own reads,
			// through the binding the command pattern already declares. A host
			// does not supply a mapping, and none is invented here.
			callInputs, bindErr := DeclaredCalls(r.World, applicationID, patternID, phase, reads)
			if bindErr != nil {
				report.Skipped = append(report.Skipped, phase)
				report.NotEvaluated = append(report.NotEvaluated, phase)
				report.Uncalled = appendUncalled(report.Uncalled, phase, declaration.Uses)
				report.Reasons[phase] = bindErr.Error()
				continue
			}
			calls, callErr := r.RunDeclaredCalls(ctx, applicationID, patternID, phase, callInputs, r.Caller)
			if len(calls) > 0 {
				if report.Calls == nil {
					report.Calls = map[string][]CallOutcome{}
				}
				report.Calls[phase] = calls
			}
			var uncalled []core.CanonicalID
			for _, call := range calls {
				if !call.Called {
					uncalled = append(uncalled, call.Capability)
				}
			}
			if len(uncalled) > 0 {
				report.Skipped = append(report.Skipped, phase)
				report.NotEvaluated = append(report.NotEvaluated, phase)
				report.Uncalled = appendUncalled(report.Uncalled, phase, uncalled)
				if r.Caller == nil {
					report.Reasons[phase] = ReasonCallerUnavailable
				} else {
					report.Reasons[phase] = ReasonDeclaredCallNotMade
				}
				continue
			}
			if callErr != nil {
				report.Refused = phase
				return report, callErr
			}
			// Every declared call was made; the phase still runs its rule below.
		}
		invocation := Invocation{
			Application: applicationID,
			Pattern:     patternID,
			Phase:       phase,
			Reads:       reads,
		}
		// RunWithDeclaredValues applies ReadsFor and, when the implementation
		// proposes values, ProducesFor. Calling the read-only path here would
		// drop a declared proposal and skip the declaration check with it.
		produced, err := r.RunWithDeclaredValues(ctx, invocation)
		if err == nil {
			report.Executed = append(report.Executed, phase)
			if len(produced) > 0 {
				if report.Produced == nil {
					report.Produced = map[string]Produced{}
				}
				report.Produced[phase] = produced
			}
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

// appendUncalled は、位相ごとの未実行の呼出しを記録する。同じ位相の項目を
// 二度足さない。
func appendUncalled(uncalled map[string][]core.CanonicalID, phase string, capabilities []core.CanonicalID) map[string][]core.CanonicalID {
	if uncalled == nil {
		uncalled = map[string][]core.CanonicalID{}
	}
	if _, exists := uncalled[phase]; exists {
		return uncalled
	}
	declared := make([]core.CanonicalID, len(capabilities))
	copy(declared, capabilities)
	uncalled[phase] = declared
	return uncalled
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
