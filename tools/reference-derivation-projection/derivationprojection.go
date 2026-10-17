// Package derivationprojection is a read-only, synthetic reference for Data
// Derivation D10 (#1707). It checks the smallest projection contract for a
// derived field shown by a modeler or a business screen: the meaning
// declaration, the unsaved preview, the runtime derived value, and the
// confirmed saved value are four different states, and a direct write never
// overwrites a derived or confirmed value.
//
// It is not an Editor, a CRUD generator, a persistence executor, a renderer, or
// an accepted API. It does not write a Source. A stale value is never shown as a
// new, confirmed value, and a saved business fact is never overwritten by a
// re-render, an input change, or a formula change.
package derivationprojection

import "fmt"

// Meaning states.
const (
	StateDeclared       = "declared"        // the rule and binding in the Source (not a runtime result)
	StatePreview        = "preview"         // a temporary evaluation of an unsaved input
	StateRuntimeDerived = "runtime-derived" // a derived display value under the current authorization
	StateConfirmedSaved = "confirmed-saved" // a business fact saved by an explicit action
)

// Events.
const (
	EventEditInput    = "edit-input"
	EventMasterChange = "master-change"
	EventConfirm      = "confirm"
	EventWriteValue   = "write-value"
	EventPropose      = "propose"
	EventAccept       = "accept"
	EventReevaluate   = "reevaluate"
	EventRender       = "render"
)

// Recomputation availability.
const (
	RecomputeAvailable           = "available"
	RecomputeInputUnavailable    = "input-unavailable"
	RecomputeTypeMismatch        = "type-mismatch"
	RecomputeComputeError        = "compute-error"
	RecomputeAuthorizationDenied = "authorization-denied"
)

// Result codes.
const (
	CodePreviewUpdated            = "PREVIEW_UPDATED"
	CodeProjectionUpdated         = "PROJECTION_UPDATED"
	CodeEditForbidden             = "EDIT_FORBIDDEN"
	CodeRecomputeUnavailable      = "RECOMPUTE_UNAVAILABLE"
	CodeAuthorizationDenied       = "AUTHORIZATION_DENIED"
	CodeConfirmedSaved            = "CONFIRMED_SAVED"
	CodeSavedFactImmutable        = "SAVED_FACT_IMMUTABLE"
	CodeProposalNotApplied        = "PROPOSAL_NOT_APPLIED"
	CodeProposalApplied           = "PROPOSAL_APPLIED"
	CodeRendererAgnostic          = "RENDERER_AGNOSTIC"
	CodeRendererContractViolation = "RENDERER_CONTRACT_VIOLATION"
	CodeInvalidRequest            = "INVALID_REQUEST"
)

// Input is the whole synthetic request.
type Input struct {
	// State is the current meaning state of the projected field.
	State string `json:"state"`
	// Value is the current value in that state, if any.
	Value string `json:"value,omitempty"`
	// Event is applied to the state.
	Event string `json:"event"`
	// Recompute is the availability of re-evaluating the derived value. The
	// default is available.
	Recompute string `json:"recompute,omitempty"`
	// NewValue is the value produced by a recomputation.
	NewValue string `json:"newValue,omitempty"`
	// Renderer is the target renderer for the render event.
	Renderer string `json:"renderer,omitempty"`
	// RendererValue is the value the target renderer would show. It must equal
	// the canonical meaning value.
	RendererValue string `json:"rendererValue,omitempty"`
}

// Report is the read-only result.
type Report struct {
	Code      string `json:"code"`
	Accepted  bool   `json:"accepted"`
	State     string `json:"state"`
	Value     string `json:"value,omitempty"`
	ValueNull bool   `json:"valueNull,omitempty"`
	Message   string `json:"message,omitempty"`
}

func reject(code, state, message string) Report {
	return Report{Code: code, State: state, Message: message}
}

func knownState(state string) bool {
	switch state {
	case StateDeclared, StatePreview, StateRuntimeDerived, StateConfirmedSaved:
		return true
	default:
		return false
	}
}

// Apply checks the projection contract for one event. It never writes anything.
func Apply(in Input) Report {
	if !knownState(in.State) {
		return reject(CodeInvalidRequest, in.State, fmt.Sprintf("unknown state %q", in.State))
	}

	switch in.Event {
	case EventWriteValue:
		return reject(CodeEditForbidden, in.State,
			"仮想項目の値セルは編集不可。規則の編集と表示結果の書換えを区別し、書込みはAction/保存境界でも拒否する。")

	case EventPropose:
		return Report{Code: CodeProposalNotApplied, Accepted: true, State: in.State, Value: in.Value,
			Message: "受理前の提案はCanonicalや保存値を変えない。"}

	case EventAccept:
		return Report{Code: CodeProposalApplied, Accepted: true, State: StateRuntimeDerived, Value: in.NewValue,
			Message: "受理後にだけ対応Projectionへ反映する。"}

	case EventConfirm:
		if !recomputeAvailable(in.Recompute) {
			return unavailable(in)
		}
		return Report{Code: CodeConfirmedSaved, Accepted: true, State: StateConfirmedSaved, Value: in.NewValue,
			Message: "業務Actionで確定保存した事実。以後の再描画・入力変更・計算式更新で勝手に上書きしない。"}

	case EventRender:
		if in.Renderer == "" {
			return reject(CodeInvalidRequest, in.State, "renderer が未指定。")
		}
		if in.RendererValue != "" && in.RendererValue != in.Value {
			return reject(CodeRendererContractViolation, in.State,
				fmt.Sprintf("renderer %q が意味値 %q と異なる %q を表示しようとした。二重正本を禁止する。", in.Renderer, in.Value, in.RendererValue))
		}
		return Report{Code: CodeRendererAgnostic, Accepted: true, State: in.State, Value: in.Value,
			Message: "実行先固有表示へ渡す意味状態は同一。"}

	case EventEditInput, EventMasterChange, EventReevaluate:
		if in.State == StateConfirmedSaved {
			return Report{Code: CodeSavedFactImmutable, Accepted: true, State: StateConfirmedSaved, Value: in.Value,
				Message: fmt.Sprintf("確定保存値 %s は不変。新しい未確定の計算は %s。", in.Value, in.NewValue)}
		}
		if !recomputeAvailable(in.Recompute) {
			return unavailable(in)
		}
		next := StateRuntimeDerived
		code := CodeProjectionUpdated
		if in.Event == EventEditInput {
			next = StatePreview
			code = CodePreviewUpdated
		}
		return Report{Code: code, Accepted: true, State: next, Value: in.NewValue,
			Message: "未保存入力の評価は保存・確定とはみなさない。"}
	default:
		return reject(CodeInvalidRequest, in.State, fmt.Sprintf("unknown event %q", in.Event))
	}
}

func recomputeAvailable(recompute string) bool {
	return recompute == "" || recompute == RecomputeAvailable
}

func unavailable(in Input) Report {
	if in.Recompute == RecomputeAuthorizationDenied {
		return Report{Code: CodeAuthorizationDenied, State: in.State, ValueNull: true,
			Message: "権限不足。古い表示値を確定値として見せない。"}
	}
	return Report{Code: CodeRecomputeUnavailable, State: in.State, ValueNull: true,
		Message: "入力未取得または計算不能。古い値を新しい結果として継続表示しない。"}
}
