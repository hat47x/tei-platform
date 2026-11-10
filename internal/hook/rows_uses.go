package hook

import (
	"fmt"

	"tei/internal/application"
)

// RefuseRowScopeWithUses reports why a row-scoped hook that also declares `uses`
// is not applied.
//
// A `uses` is a promise that the capability is called. The record path keeps that
// promise through RunDeclaredCalls, once per phase. No per-row equivalent exists,
// so a row-scoped hook with `uses` would either call once for the whole page
// (keeping the promise for no particular row) or not call at all while the phase
// still reads as executed. Both are the inert declaration this boundary exists to
// avoid, so the shape is refused instead.
//
// The rule lives here so both row entry points (RunRows and applyReachableRows)
// answer the same declaration the same way. Two copies of this decision is how
// they came to disagree: Run succeeded while RunReachable refused.
func refuseRowScopeWithUses(phase string, declared *application.Hook) error {
	if declared == nil || len(declared.Uses) == 0 {
		return nil
	}
	return &Error{
		Code:    CodeRowScopeUsesUnsupported,
		Message: fmt.Sprintf("hook %s declares scope row with uses, which this boundary does not call per row", phase),
	}
}

// CodeRowScopeUsesUnsupported names the refusal of a row-scoped hook that
// declares `uses`.
const CodeRowScopeUsesUnsupported = "HOOK_SCOPE_ROW_USES_UNSUPPORTED"
