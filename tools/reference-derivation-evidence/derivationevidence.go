// Package derivationevidence is a read-only, synthetic reference for Data
// Derivation D7 (#1696). For one derived value it builds the smallest
// explanation projection: which accepted rule, revision, target, realization
// and input produced it, what the result state is, and how much of that may be
// disclosed to the requester.
//
// It is not a distributed tracing system, a second Source index, an
// authorization decision, a UI, or an accepted API. It does not write a Source.
// It never collapses a missing input or an undeclared NULL into zero, never
// claims a full reproduction when the input version is unknown, and never leaks
// a hidden input value, unit price or row existence to an unauthorized
// requester.
package derivationevidence

import "fmt"

// Result codes.
const (
	CodePresent               = "RESULT_PRESENT"
	CodeInputMissing          = "INPUT_MISSING"
	CodeNullDeclared          = "NULL_DECLARED"
	CodeNullUndeclared        = "NULL_UNDECLARED"
	CodeUnauthorized          = "UNAUTHORIZED"
	CodeComputeError          = "COMPUTE_ERROR"
	CodeMultiplicityViolation = "MULTIPLICITY_VIOLATION"
	CodeTypeMismatch          = "TYPE_MISMATCH"
	CodeUnsupported           = "UNSUPPORTED_OPERATION"
	CodeInvalidRequest        = "INVALID_REQUEST"
)

// Result states of the evaluated value.
const (
	StatePresent               = "present"
	StateInputMissing          = "input-missing"
	StateNullDeclared          = "null-declared"
	StateNullUndeclared        = "null-undeclared"
	StateUnauthorized          = "unauthorized"
	StateComputeError          = "compute-error"
	StateMultiplicityViolation = "multiplicity-violation"
	StateTypeMismatch          = "type-mismatch"
	StateUnsupported           = "unsupported"
)

// Requesters.
const (
	RequesterAdmin   = "admin"
	RequesterGeneral = "general"
)

// Input is the whole synthetic request.
type Input struct {
	// Design provenance (where the rule is written), not proof of a runtime
	// value.
	RuleID           string `json:"ruleId"`
	AcceptedRevision string `json:"acceptedRevision"`
	TargetID         string `json:"targetId"`
	Scope            string `json:"scope,omitempty"`
	// Evaluation evidence.
	Realization        string `json:"realization,omitempty"`
	RealizationVersion string `json:"realizationVersion,omitempty"`
	InputID            string `json:"inputId,omitempty"`
	InputVersion       string `json:"inputVersion,omitempty"`
	InputVersionKnown  bool   `json:"inputVersionKnown"`
	ContextClass       string `json:"contextClass,omitempty"`
	// Evaluation result state.
	ResultState string `json:"resultState"`
	// Sensitive values, disclosed only to an authorized requester.
	InputValue string `json:"inputValue,omitempty"`
	UnitPrice  string `json:"unitPrice,omitempty"`
	// Requester is admin or general.
	Requester string `json:"requester"`
	// SourceMoved is true when the rule file moved without a change of identity or
	// meaning.
	SourceMoved bool `json:"sourceMoved,omitempty"`
}

// Report is the read-only explanation projection.
type Report struct {
	Code               string   `json:"code"`
	Accepted           bool     `json:"accepted"`
	RuleID             string   `json:"ruleId,omitempty"`
	AcceptedRevision   string   `json:"acceptedRevision,omitempty"`
	TargetID           string   `json:"targetId,omitempty"`
	Realization        string   `json:"realization,omitempty"`
	RealizationVersion string   `json:"realizationVersion,omitempty"`
	ResultState        string   `json:"resultState"`
	Traceable          bool     `json:"traceable"`
	Unknown            []string `json:"unknown,omitempty"`
	Disclosed          []string `json:"disclosed"`
	Redacted           []string `json:"redacted,omitempty"`
	InputValue         string   `json:"inputValue,omitempty"`
	UnitPrice          string   `json:"unitPrice,omitempty"`
	Message            string   `json:"message,omitempty"`
}

func stateCode(state string) (string, bool) {
	switch state {
	case StatePresent:
		return CodePresent, true
	case StateInputMissing:
		return CodeInputMissing, true
	case StateNullDeclared:
		return CodeNullDeclared, true
	case StateNullUndeclared:
		return CodeNullUndeclared, true
	case StateUnauthorized:
		return CodeUnauthorized, true
	case StateComputeError:
		return CodeComputeError, true
	case StateMultiplicityViolation:
		return CodeMultiplicityViolation, true
	case StateTypeMismatch:
		return CodeTypeMismatch, true
	case StateUnsupported:
		return CodeUnsupported, true
	default:
		return "", false
	}
}

// Explain builds the explanation projection for one derived value. It never
// writes anything.
func Explain(in Input) Report {
	code, ok := stateCode(in.ResultState)
	if !ok {
		return Report{Code: CodeInvalidRequest, ResultState: in.ResultState,
			Message: fmt.Sprintf("unknown result state %q", in.ResultState)}
	}
	if in.Requester != RequesterAdmin && in.Requester != RequesterGeneral {
		return Report{Code: CodeInvalidRequest, ResultState: in.ResultState,
			Message: fmt.Sprintf("unknown requester %q", in.Requester)}
	}

	report := Report{
		Code:               code,
		Accepted:           true,
		RuleID:             in.RuleID,
		AcceptedRevision:   in.AcceptedRevision,
		TargetID:           in.TargetID,
		Realization:        in.Realization,
		RealizationVersion: in.RealizationVersion,
		ResultState:        in.ResultState,
	}

	// The explanation identity depends on the canonical identity and meaning, not
	// on the rule file path.
	report.Disclosed = []string{"ruleId", "acceptedRevision", "targetId", "realization", "resultState"}
	if in.SourceMoved {
		report.Message = "規則のSource位置が移動してもCanonical identityと意味は不変。"
	}

	// Traceability: without the input version a full reproduction is not claimed.
	report.Traceable = in.InputVersionKnown
	if !in.InputVersionKnown {
		report.Unknown = append(report.Unknown, "inputVersion")
	}

	// Disclosure boundary: a general requester never receives hidden inputs. An
	// admin receives them only for a present result; a missing, NULL, or failed
	// result has no value to disclose.
	if in.Requester == RequesterAdmin && in.ResultState == StatePresent {
		report.InputValue = in.InputValue
		report.UnitPrice = in.UnitPrice
		report.Disclosed = append(report.Disclosed, "inputValue", "unitPrice")
	} else {
		report.Redacted = appendUnique(report.Redacted, "inputValue", "unitPrice")
		if in.Requester != RequesterAdmin {
			report.Redacted = appendUnique(report.Redacted, "rowExistence")
		}
	}

	// An unauthorized evaluation discloses no value and no inputs.
	if in.ResultState == StateUnauthorized {
		report.Redacted = appendUnique(report.Redacted, "inputValue", "unitPrice", "rowExistence")
		report.InputValue = ""
		report.UnitPrice = ""
	}

	// Each non-present state keeps its own diagnostic; they are not collapsed.
	switch in.ResultState {
	case StateInputMissing:
		report.Message = "入力不足。NULLや計算結果0へ潰さない。"
	case StateNullDeclared:
		report.Message = "宣言されたNULL規則による結果。"
	case StateNullUndeclared:
		report.Message = "NULL規則が未宣言。NULLを0として扱わない。"
	case StateUnauthorized:
		report.Message = "権限拒否。値と入力を開示しない。"
	case StateComputeError:
		report.Message = "計算失敗。原因不明をNULLへ潰さない。"
	}
	return report
}

func appendUnique(list []string, items ...string) []string {
	for _, item := range items {
		found := false
		for _, existing := range list {
			if existing == item {
				found = true
				break
			}
		}
		if !found {
			list = append(list, item)
		}
	}
	return list
}
