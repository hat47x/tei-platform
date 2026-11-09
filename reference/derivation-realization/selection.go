// Package derivationrealization is a narrow D6 reference for binding an
// already accepted D1 rule to one Host-selected Go realization. It adds no
// expression engine, Plugin Lock, Operator resolver, or acceptance authority.
package derivationrealization

import (
	"errors"
	"fmt"

	"tei/internal/environment"
	"tei/internal/plugin"
	"tei/internal/semantic/derivationref"
	languagereference "tei/reference/dsl-language-plugin"
)

// ErrUnproven means that the requested realization cannot be demonstrated
// with the exact Host-supplied lock, operator pin, and target. The caller must
// not silently substitute a different plugin or execution target.
var ErrUnproven = errors.New("derivation realization is unproven")

// Invocation refers to the existing rule identity without making the Plugin
// identity part of that rule. The artifact identity comes from the existing
// Plugin Lock / definition, not from a new realization registry.
type Invocation struct {
	Rule           derivationref.Definition
	OperatorPin    languagereference.OperatorContract
	PluginPin      plugin.Identity
	ArtifactDigest string
	Target         string
}

// GoRealization is an explicitly supplied reference implementation selected
// by the Host. ArtifactDigest must match the existing pinned descriptor;
// this is only a declaration check, not byte-level attestation. Run is invoked
// only after all independent pins match. It is not
// obtained from an untrusted Plugin descriptor, and this function does not
// itself authorize code execution or verify arbitrary AI-generated code.
type GoRealization struct {
	Plugin         plugin.Identity
	Operator       languagereference.OperatorContract
	ArtifactDigest string
	Target         string
	Run            func(derivationref.Input, derivationref.Input) (derivationref.Input, error)
}

// HostSelection consumes existing environment-scoped Plugin resolution and
// Operator pinning. It does not create a new source of Host/Plugin authority.
type HostSelection struct {
	Catalog         plugin.Catalog
	Environment     environment.Environment
	Lock            plugin.Lock
	OperatorCatalog []languagereference.OperatorContract
}

// Evaluate executes only the requested Go realization; no provider ranking,
// automatic version upgrade, or SQL/Go fallback is permitted.
//
// The initial D6 fixture intentionally supports exact decimal multiplication
// with nullable output only. Other targets, contracts, effects, cardinalities,
// or rounding policies are not silently treated as equivalent.
func (h HostSelection) Evaluate(
	request Invocation,
	realization GoRealization,
	left, right derivationref.Input,
) (derivationref.Input, error) {
	if request.Rule.ID == "" || !request.Rule.Pure ||
		len(request.Rule.Inputs) != 2 || request.Rule.Inputs[0] == "" ||
		request.Rule.Inputs[1] == "" || request.Rule.Inputs[0] == request.Rule.Inputs[1] {
		return derivationref.Input{}, fmt.Errorf("%w: rule is incomplete or impure", ErrUnproven)
	}
	if request.Rule.Operator != request.OperatorPin.ID {
		return derivationref.Input{}, fmt.Errorf("%w: rule operator differs from its pin", ErrUnproven)
	}
	if request.OperatorPin.InputType != "decimal,decimal" ||
		request.OperatorPin.OutputType != "decimal-or-null" {
		return derivationref.Input{}, fmt.Errorf("%w: decimal/NULL semantics not supported", ErrUnproven)
	}
	if _, err := languagereference.ResolveOperator(request.OperatorPin, h.OperatorCatalog); err != nil {
		return derivationref.Input{}, fmt.Errorf("%w: operator contract: %v", ErrUnproven, err)
	}
	if request.Target != "go/reference" || request.PluginPin.ID == "" ||
		request.PluginPin.Version == "" || request.ArtifactDigest == "" {
		return derivationref.Input{}, fmt.Errorf("%w: unrecognized target or implementation pin", ErrUnproven)
	}

	// Existing Plugin Lock checks exact descriptor fingerprint, artifact
	// digest, environment selection, and Plugin dependency closure.
	resolved, err := h.Catalog.ResolveEnvironmentWithLock(h.Environment, h.Lock)
	if err != nil {
		return derivationref.Input{}, fmt.Errorf("%w: locked Plugin resolution: %v", ErrUnproven, err)
	}
	inSet := false
	for _, id := range resolved.Plugins {
		if id == request.PluginPin {
			inSet = true
		}
	}
	if !inSet {
		return derivationref.Input{}, fmt.Errorf("%w: requested Plugin is not in the locked environment", ErrUnproven)
	}

	// A duplicate exact identity must not be accepted based on catalog order.
	// This is an ambiguity check, not a second version-selection mechanism.
	var chosen *plugin.Definition
	for i := range h.Catalog.Plugins {
		def := &h.Catalog.Plugins[i]
		if def.ID == request.PluginPin.ID && def.Version == request.PluginPin.Version {
			if chosen != nil {
				return derivationref.Input{}, fmt.Errorf("%w: duplicate Plugin identity", ErrUnproven)
			}
			chosen = def
		}
	}
	if chosen == nil || chosen.Runtime.Kind != plugin.RuntimeBuiltin ||
		chosen.Artifact.Digest != request.ArtifactDigest {
		return derivationref.Input{}, fmt.Errorf("%w: Plugin artifact or runtime does not match", ErrUnproven)
	}
	if err := chosen.Artifact.Validate(); err != nil {
		return derivationref.Input{}, fmt.Errorf("%w: invalid Plugin artifact pin", ErrUnproven)
	}
	if realization.Plugin != request.PluginPin || realization.Target != request.Target ||
		realization.Operator != request.OperatorPin || realization.ArtifactDigest != request.ArtifactDigest ||
		realization.Run == nil {
		return derivationref.Input{}, fmt.Errorf("%w: requested Go realization is unavailable", ErrUnproven)
	}
	return realization.Run(left, right)
}

// GoMultiply is the one real D6 Reference evaluation, reusing D1's exact
// decimal, NULL, and missing-input semantics without copying its arithmetic.
func GoMultiply(a, b derivationref.Input) (derivationref.Input, error) {
	return derivationref.Multiply(a, b)
}
