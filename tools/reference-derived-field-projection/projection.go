// Package derivedfieldprojection is a read-only, synthetic reference for Data
// Derivation D10 (#1707). It keeps four states apart: the declared meaning, an
// unsaved edit preview, a runtime derived value, and a business-confirmed saved
// value. A derived field is never directly writable, including at the
// Action/save boundary, and an old displayed value is never presented as a
// confirmed one when the current evaluation cannot run.
//
// It is not a UI framework, an ER editor, a rule registry, a transaction
// executor, or an accepted API. It does not write a Source. Widget selection is
// delegated to the UI issues; this reference only carries the semantic state.
package derivedfieldprojection

import (
	"fmt"
	"strings"
)

// Field kinds.
const (
	KindPhysical  = "physical"
	KindDerived   = "derived"
	KindConfirmed = "confirmed"
)

// Phases.
const (
	PhaseDeclaration = "declaration"
	PhasePreview     = "preview"
	PhaseRuntime     = "runtime"
)

// Operations.
const (
	OpDisplay  = "display"
	OpWrite    = "write"
	OpEditRule = "edit-rule"
)

// Derivation states supplied by the D1-style evaluator.
const (
	DerivedPresent          = "present"
	DerivedInputUnavailable = "input-unavailable"
	DerivedAuthorization    = "authorization-denied"
	DerivedComputationError = "computation-failed"
)

// Result codes. Preview and runtime carry the same state names with a distinct
// prefix, so a caller can never confuse one with the other.
const (
	CodeProjectionDeclared           = "PROJECTION_DECLARED"
	CodePhysicalWritable             = "PHYSICAL_FIELD_WRITABLE"
	CodePhysicalRead                 = "PHYSICAL_FIELD_READ"
	CodeDerivedReadOnly              = "DERIVED_FIELD_READ_ONLY"
	CodeRuleEditRequiresAcceptance   = "RULE_EDIT_REQUIRES_ACCEPTANCE"
	CodePreviewPresent               = "PREVIEW_PRESENT"
	CodePreviewInputUnavailable      = "PREVIEW_INPUT_UNAVAILABLE"
	CodePreviewAuthorizationDenied   = "PREVIEW_AUTHORIZATION_DENIED"
	CodePreviewComputationFailed     = "PREVIEW_COMPUTATION_FAILED"
	CodeRuntimePresent               = "RUNTIME_PRESENT"
	CodeRuntimeInputUnavailable      = "RUNTIME_INPUT_UNAVAILABLE"
	CodeRuntimeAuthorizationDenied   = "RUNTIME_AUTHORIZATION_DENIED"
	CodeRuntimeComputationFailed     = "RUNTIME_COMPUTATION_FAILED"
	CodeConfirmedSaved               = "CONFIRMED_SAVED"
	CodeConfirmedWriteRequiresAction = "CONFIRMED_WRITE_REQUIRES_ACTION"
	CodeInvalidRequest               = "INVALID_REQUEST"
)

// Field is one projected field.
type Field struct {
	ID   string `json:"id"`
	Kind string `json:"kind"`
}

// Input is the whole synthetic request.
type Input struct {
	Field Field `json:"field"`
	// Phase is declaration, preview, or runtime.
	Phase string `json:"phase"`
	// Operation is display, write, or edit-rule.
	Operation string `json:"operation"`
	// AttemptedValue is the value a caller wants to write.
	AttemptedValue string `json:"attemptedValue,omitempty"`
	// DerivedState and DerivedValue are the current evaluator result.
	DerivedState string `json:"derivedState,omitempty"`
	DerivedValue string `json:"derivedValue,omitempty"`
	// SavedValue is the business-confirmed saved fact, if any.
	SavedValue string `json:"savedValue,omitempty"`
	// InputsUnsaved is true when the current inputs are unsaved edits.
	InputsUnsaved bool `json:"inputsUnsaved,omitempty"`
	// Renderers lists display targets that consume the same semantic state.
	Renderers []string `json:"renderers,omitempty"`
}

// Report is the read-only result.
type Report struct {
	Field               string            `json:"field"`
	Kind                string            `json:"kind"`
	Editable            bool              `json:"editable"`
	Code                string            `json:"code"`
	Value               string            `json:"value,omitempty"`
	AttemptedValue      string            `json:"attemptedValue,omitempty"`
	SavedValue          string            `json:"savedValue,omitempty"`
	SavedValueUnchanged bool              `json:"savedValueUnchanged"`
	ProjectionState     string            `json:"projectionState"`
	Renderers           map[string]string `json:"renderers,omitempty"`
	Message             string            `json:"message,omitempty"`
}

func reject(in Input, code, message string) Report {
	return Report{
		Field: in.Field.ID, Kind: in.Field.Kind,
		Editable: in.Field.Kind == KindPhysical,
		Code:     code, ProjectionState: "rejected", Message: message,
	}
}

// Evaluate applies the projection contract. It never writes anything.
func Evaluate(in Input) Report {
	if in.Field.ID == "" {
		return reject(in, CodeInvalidRequest, "field id is required")
	}
	switch in.Field.Kind {
	case KindPhysical, KindDerived, KindConfirmed:
	default:
		return reject(in, CodeInvalidRequest, fmt.Sprintf("unknown field kind %q", in.Field.Kind))
	}
	if in.Operation == OpEditRule {
		return reject(in, CodeRuleEditRequiresAcceptance,
			"規則の編集は値の編集ではない。既存のProposal→Acceptance→Sourceの経路で扱う。")
	}
	switch in.Operation {
	case OpDisplay, OpWrite:
	default:
		return reject(in, CodeInvalidRequest, fmt.Sprintf("unknown operation %q", in.Operation))
	}

	switch in.Field.Kind {
	case KindPhysical:
		return evaluatePhysical(in)
	case KindConfirmed:
		return evaluateConfirmed(in)
	case KindDerived:
		return evaluateDerived(in)
	}
	return reject(in, CodeInvalidRequest, "unreachable")
}

func evaluatePhysical(in Input) Report {
	if in.Operation == OpWrite {
		return Report{
			Field: in.Field.ID, Kind: in.Field.Kind, Editable: true,
			Code: CodePhysicalWritable, Value: in.AttemptedValue,
			SavedValueUnchanged: false, ProjectionState: "physical-writable",
			Renderers: renderers(in, "physical-writable"),
			Message:   "物理保存値は通常の属性として書ける。",
		}
	}
	return Report{
		Field: in.Field.ID, Kind: in.Field.Kind, Editable: true,
		Code: CodePhysicalRead, Value: in.AttemptedValue,
		SavedValueUnchanged: false, ProjectionState: "physical-read",
		Renderers: renderers(in, "physical-read"),
	}
}

func evaluateConfirmed(in Input) Report {
	if in.Operation == OpWrite {
		// A confirmed value is written by a business Action, not by a field edit.
		return reject(in, CodeConfirmedWriteRequiresAction,
			"確定保存値は業務Actionの結果として書く。フィールドの直接編集では上書きしない。")
	}
	return Report{
		Field: in.Field.ID, Kind: in.Field.Kind, Editable: false,
		Code: CodeConfirmedSaved, Value: in.SavedValue, SavedValue: in.SavedValue,
		SavedValueUnchanged: true, ProjectionState: "confirmed",
		Renderers: renderers(in, "confirmed"),
		Message:   "保存済みの業務確定事実を返す。再計算しない。",
	}
}

func evaluateDerived(in Input) Report {
	editable := false
	// The write boundary is independent of phase and of whether a UI happens to
	// disable the cell: an Action or save path that writes a derived field is
	// rejected here too.
	if in.Operation == OpWrite {
		return Report{
			Field: in.Field.ID, Kind: in.Field.Kind, Editable: editable,
			Code: CodeDerivedReadOnly, AttemptedValue: in.AttemptedValue,
			SavedValue: in.SavedValue, SavedValueUnchanged: true, ProjectionState: "derived-read-only",
			Renderers: renderers(in, "derived-read-only"),
			Message:   fmt.Sprintf("導出値 %q への直接書込を拒否する。案件で金額調整が必要なら別の業務Actionとして定義する。", in.Field.ID),
		}
	}

	if in.Phase == PhaseDeclaration {
		return Report{
			Field: in.Field.ID, Kind: in.Field.Kind, Editable: editable,
			Code: CodeProjectionDeclared, SavedValue: in.SavedValue, SavedValueUnchanged: true,
			ProjectionState: "derived-declared",
			Renderers:       renderers(in, "derived-declared"),
			Message:         "導出値として宣言する。表示は読み取り専用で、値の正本は規則と入力。",
		}
	}

	prefix := "PREVIEW"
	if in.Phase == PhaseRuntime {
		prefix = "RUNTIME"
	}
	if in.Phase != PhasePreview && in.Phase != PhaseRuntime {
		return reject(in, CodeInvalidRequest, fmt.Sprintf("unknown phase %q", in.Phase))
	}

	switch strings.TrimSpace(in.DerivedState) {
	case DerivedPresent:
		return Report{
			Field: in.Field.ID, Kind: in.Field.Kind, Editable: editable,
			Code: prefixCode(prefix, "PRESENT"), Value: in.DerivedValue,
			SavedValue: in.SavedValue, SavedValueUnchanged: true, ProjectionState: strings.ToLower(prefix) + "-present",
			Renderers: renderers(in, strings.ToLower(prefix)+"-present"),
			Message:   "現在の入力で評価できた。確定保存値ではない。",
		}
	case DerivedInputUnavailable:
		return unavailable(in, prefixCode(prefix, "INPUT_UNAVAILABLE"), "入力未取得。古い表示値を新しい結果として返さない。")
	case DerivedAuthorization:
		return unavailable(in, prefixCode(prefix, "AUTHORIZATION_DENIED"), "認可拒否。値も存在も開示しない。")
	case DerivedComputationError:
		return unavailable(in, prefixCode(prefix, "COMPUTATION_FAILED"), "計算不能。0や古い値で代用しない。")
	default:
		return reject(in, CodeInvalidRequest, fmt.Sprintf("unknown derived state %q", in.DerivedState))
	}
}

func unavailable(in Input, code, message string) Report {
	return Report{
		Field: in.Field.ID, Kind: in.Field.Kind, Editable: false,
		Code: code, Value: "", SavedValue: in.SavedValue, SavedValueUnchanged: true,
		ProjectionState: "unavailable",
		Renderers:       renderers(in, "unavailable"),
		Message:         message,
	}
}

func prefixCode(prefix, suffix string) string {
	return prefix + "_" + suffix
}

func renderers(in Input, state string) map[string]string {
	if len(in.Renderers) == 0 {
		return nil
	}
	out := make(map[string]string, len(in.Renderers))
	for _, renderer := range in.Renderers {
		out[renderer] = state
	}
	return out
}
