package management

import (
	"errors"

	"tei/internal/plugin"
)

// resolverReasons maps the resolver's own refusal codes to the management
// vocabulary. The mapping lives here, once, so the two vocabularies cannot
// both claim to be the reason for the same fact. The resolver's code is
// reported beside the management code and is never rewritten.
func resolverReasons(err error) []ResolverReason {
	// A lock problem is an input problem (#3184): it is reported in the
	// management vocabulary that names the lock, not as a resolver judgement.
	if IsLockError(err) {
		return []ResolverReason{{
			ManagementCode: lockReasonCode(err),
			Message:        err.Error(),
		}}
	}
	diagnostics := plugin.ResolveDiagnostics(err)
	if len(diagnostics) == 0 {
		return []ResolverReason{{
			ManagementCode: ReasonResolverRefused,
			Message:        err.Error(),
		}}
	}
	out := make([]ResolverReason, 0, len(diagnostics))
	for _, d := range diagnostics {
		out = append(out, ResolverReason{
			ManagementCode: managementCodeFor(d.Code),
			ResolverCode:   d.Code,
			Subject:        string(d.Subject),
			Message:        d.Message,
		})
	}
	return out
}

// managementCodeFor is the single mapping from a resolver refusal code to a
// management reason code. A resolver code with no specific meaning maps to
// the generic resolver-refused, so a new resolver code never silently becomes
// a specific management claim.
func managementCodeFor(resolverCode string) string {
	switch resolverCode {
	case plugin.DiagnosticNoCompatiblePlugin, plugin.DiagnosticNoCompatibleBundle:
		return ReasonVersionUnsupported
	case plugin.DiagnosticDependencyCycle:
		return ReasonDependencyCycle
	case plugin.DiagnosticDuplicateRequirement, plugin.DiagnosticInvalidRequirement,
		plugin.DiagnosticDuplicatePluginVersion, plugin.DiagnosticDuplicateBundleVersion:
		return ReasonDependencyUnresolved
	case plugin.DiagnosticIncompatibleTransitive:
		return ReasonContractIncompatible
	case plugin.DiagnosticEnvironmentSelection:
		return ReasonResolverRefused
	default:
		return ReasonResolverRefused
	}
}

// IsLockError reports whether a resolution error came from the lock bytes or a
// lock that describes another workspace. Both are input problems: the caller
// can only fix them by supplying another lock (#3184), so the CLI keeps
// INVALID_PLUGIN_LOCK rather than inventing a management failure code.
func IsLockError(err error) bool {
	return plugin.IsInvalidLock(err) || plugin.IsLockMismatch(err)
}

// lockReasonCode names the lock problem in the management vocabulary without
// adding a second failure-code system. It is empty when err is not a lock
// problem.
func lockReasonCode(err error) string {
	switch {
	case err == nil:
		return ""
	case plugin.IsInvalidLock(err):
		return ReasonLockNotProvided
	case plugin.IsLockMismatch(err):
		return ReasonSourceFingerprintMismatch
	default:
		return ""
	}
}

var errUnknownPlugin = errors.New("unknown plugin")
