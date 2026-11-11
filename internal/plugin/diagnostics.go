package plugin

import (
	"errors"
	"fmt"

	"tei/internal/core"
)

// Resolution codes. The set is a rule vocabulary for the resolver, not a fixed
// core enum: a producer adds a code together with the case that emits it.
//
// These name *why* resolution stopped. They are not acceptance, not authority,
// and not a substitute for reading the message: two catalogs that produce the
// same code may differ in the ids involved, which is what Subject records.
const (
	// A declaration repeats a dependency identity.
	DiagnosticDuplicateRequirement = "PLUGIN_RESOLVE_DUPLICATE_REQUIREMENT"
	// A requirement has an invalid canonical id or a nonpositive major.
	DiagnosticInvalidRequirement = "PLUGIN_RESOLVE_INVALID_REQUIREMENT"
	// No catalog plugin matches the id and compatible major.
	DiagnosticNoCompatiblePlugin = "PLUGIN_RESOLVE_NO_COMPATIBLE_PLUGIN"
	// The catalog declares the same id and version more than once.
	DiagnosticDuplicatePluginVersion = "PLUGIN_RESOLVE_DUPLICATE_PLUGIN_VERSION"
	// No catalog bundle matches the id and compatible major.
	DiagnosticNoCompatibleBundle = "PLUGIN_RESOLVE_NO_COMPATIBLE_BUNDLE"
	// The catalog declares the same bundle id and version more than once.
	DiagnosticDuplicateBundleVersion = "PLUGIN_RESOLVE_DUPLICATE_BUNDLE_VERSION"
	// A transitive requirement demands a major another resolution already fixed.
	DiagnosticIncompatibleTransitive = "PLUGIN_RESOLVE_INCOMPATIBLE_TRANSITIVE"
	// The dependency graph contains a cycle.
	DiagnosticDependencyCycle = "PLUGIN_RESOLVE_DEPENDENCY_CYCLE"
	// The environment selection list was rejected as a whole. The specific
	// reason is the wrapped inner diagnostic, so a caller reports the whole
	// chain rather than only the outer code.
	DiagnosticEnvironmentSelection = "PLUGIN_RESOLVE_ENVIRONMENT_SELECTION"
)

// ResolveDiagnostic is one reason resolution stopped, in a form a caller can
// branch on without reading the message.
//
// Error() returns exactly the message the resolver produced before this type
// existed. That is deliberate: the type adds a code and a subject *alongside*
// the text, so existing readers and tests that match on the message keep
// working while new callers stop depending on it.
type ResolveDiagnostic struct {
	Code string
	// Subject is the plugin or bundle the refusal is about. Empty when the
	// refusal is about the environment's own selection list as a whole.
	Subject core.CanonicalID
	Message string
	// Err is the wrapped inner diagnostic when the refusal was reached
	// through one. Unwrap exposes it, so ResolveDiagnostics reports the whole
	// chain.
	Err error
}

func (d *ResolveDiagnostic) Error() string { return d.Message }

func (d *ResolveDiagnostic) Unwrap() error { return d.Err }

// diagnose builds a ResolveDiagnostic whose message is formatted identically to
// the fmt.Errorf call it replaces.
func diagnose(code string, subject core.CanonicalID, format string, args ...any) error {
	return &ResolveDiagnostic{
		Code:    code,
		Subject: subject,
		Message: fmt.Sprintf(format, args...),
	}
}

// wrapDiagnose builds a ResolveDiagnostic that also carries an inner error, as
// fmt.Errorf with %w did.
func wrapDiagnose(code string, subject core.CanonicalID, inner error, format string, args ...any) error {
	return &ResolveDiagnostic{
		Code:    code,
		Subject: subject,
		Message: fmt.Sprintf(format, args...),
		Err:     inner,
	}
}

// ResolveDiagnostics returns every ResolveDiagnostic in err's chain, outermost
// first. A resolver refusal reached through a wrapper yields one entry per
// level, so a caller can report the whole path rather than only the last hop.
//
// An error that carries no ResolveDiagnostic yields nil. A caller must treat
// nil as "unknown reason", not as "resolution succeeded".
func ResolveDiagnostics(err error) []ResolveDiagnostic {
	if err == nil {
		return nil
	}
	var out []ResolveDiagnostic
	for err != nil {
		var d *ResolveDiagnostic
		if errors.As(err, &d) {
			out = append(out, *d)
			err = d.Err
			continue
		}
		break
	}
	return out
}

// ResolveCode returns the outermost ResolveDiagnostic code in err, or "" when
// err carries none. A caller that switches on a single code uses this; a caller
// that reports the path uses ResolveDiagnostics.
func ResolveCode(err error) string {
	diagnostics := ResolveDiagnostics(err)
	if len(diagnostics) == 0 {
		return ""
	}
	return diagnostics[0].Code
}
