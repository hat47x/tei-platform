package hook

import (
	"context"

	"tei/internal/application"
	"tei/internal/command"
	"tei/internal/core"
)

// CommandCaller runs a declared capability through the Command boundary, so a
// hook's call inherits the invoker's authorization, transaction, effect check,
// and operation record instead of reaching the data directly.
type CommandCaller struct {
	Invoker     command.Invoker
	Application core.CanonicalID
	// Pattern maps a capability to the Pattern the hook calls it through. The
	// Command boundary resolves inputs against a Pattern, so the hook cannot
	// invent a call target the Application does not declare.
	Pattern func(capability core.CanonicalID) (core.CanonicalID, bool)
}

// Call runs the capability through the Command boundary and returns its result.
func (c CommandCaller) Call(ctx context.Context, capability core.CanonicalID, inputs map[string]string) (any, error) {
	if c.Application == "" {
		return nil, &Error{Code: "HOOK_APPLICATION_REQUIRED", Message: "the command caller needs the application id"}
	}
	if c.Pattern == nil {
		return nil, &Error{
			Code:    "HOOK_PATTERN_NOT_DECLARED",
			Message: "no pattern is configured for capability " + string(capability),
		}
	}
	pattern, ok := c.Pattern(capability)
	if !ok || pattern == "" {
		return nil, &Error{
			Code:    "HOOK_PATTERN_NOT_DECLARED",
			Message: "no pattern declares capability " + string(capability),
		}
	}
	request := command.Request{
		Application: c.Application,
		Pattern:     pattern,
		Capability:  capability,
		Inputs:      inputs,
	}
	result, err := c.Invoker.Invoke(ctx, request)
	if err != nil {
		return nil, err
	}
	return result, nil
}

// DeclaredCommand calls a declared hook's capability through the Command
// boundary, refusing a capability the hook did not declare.
type DeclaredCommand struct {
	CommandCaller
	Hook *application.Hook
}

// Call refuses an undeclared capability, then runs it through the boundary.
func (d DeclaredCommand) Call(ctx context.Context, capability core.CanonicalID, inputs map[string]string) (any, error) {
	return Declared{Inner: d.CommandCaller, Hook: d.Hook}.Call(ctx, capability, inputs)
}
