package hook

import (
	"context"
	"fmt"

	"tei/internal/application"
	"tei/internal/core"
)

// Caller is the boundary a hook uses to call a capability. The hook does not
// receive the world directly, so the only capabilities it can reach are the ones
// the caller permits.
type Caller interface {
	Call(ctx context.Context, capability core.CanonicalID, inputs map[string]string) (any, error)
}

// Declared is the caller a hook is given. It permits only the capabilities the
// hook's declaration lists, so a hook that reaches an undeclared capability is
// refused instead of being trusted to behave.
type Declared struct {
	Inner Caller
	Hook  *application.Hook
}

// Call refuses a capability the hook did not declare, then delegates.
func (d Declared) Call(ctx context.Context, capability core.CanonicalID, inputs map[string]string) (any, error) {
	if d.Hook == nil {
		return nil, &Error{Code: "HOOK_UNDECLARED_CALL", Message: fmt.Sprintf("capability %s was called without a hook declaration", capability)}
	}
	for _, declared := range d.Hook.Uses {
		if declared == capability {
			if d.Inner == nil {
				return nil, &Error{Code: "HOOK_IMPLEMENTATION_UNAVAILABLE", Message: "no caller is configured for this hook boundary"}
			}
			return d.Inner.Call(ctx, capability, inputs)
		}
	}
	return nil, &Error{
		Code:    "HOOK_UNDECLARED_CALL",
		Message: fmt.Sprintf("hook %s does not declare capability %s, so the call is refused", d.Hook.Phase, capability),
	}
}
