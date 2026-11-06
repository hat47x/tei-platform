package runtime

import (
	"sort"

	"tei/internal/permission"
)

// ProfileFromPermissions derives the sandbox profile from the declared
// permissions. The derivation covers what a declaration can express: network
// destinations and secret handles. It does not invent mounts or secret
// injections, which a declaration does not describe; those are reported as
// underived so a caller can supply them explicitly instead of assuming.
func ProfileFromPermissions(perms []permission.Spec) (SandboxProfile, []string) {
	profile := SandboxProfile{
		DenyNetworkByDefault:    true,
		DenyFilesystemByDefault: true,
	}
	notes := make([]string, 0, 2)
	for _, perm := range perms {
		switch perm.Kind {
		case permission.KindNetwork:
			if perm.Access == permission.AccessConnect {
				profile.NetworkAllowlist = append(profile.NetworkAllowlist, perm.Scope)
			}
		case permission.KindSecret:
			if perm.Access == permission.AccessRead {
				profile.SecretHandles = append(profile.SecretHandles, perm.Scope)
			}
		case permission.KindFilesystem:
			// A permission names a scope, not a host path and access mode, so the
			// mount cannot be derived from it.
			notes = append(notes, "filesystem: "+perm.Scope+" is declared, but mounts are not derived from a permission")
		}
	}
	sort.Strings(profile.NetworkAllowlist)
	sort.Strings(profile.SecretHandles)
	if len(profile.SecretHandles) > 0 {
		notes = append(notes, "secret injections are not derived from a permission; supply them explicitly")
	}
	return profile, notes
}

// ProfileExceedsDeclarations reports the ways a profile allows more than the
// declared permissions permit. Only the network allowlist is compared: it names
// destinations in the same terms as a declaration. Secret handles and mounts are
// physical references produced by resource resolution (a handle such as
// "vault://v1" against a declared scope such as "database-credential"), so
// comparing them with a scope would report a difference that is not one.
func ProfileExceedsDeclarations(profile SandboxProfile, perms []permission.Spec) []string {
	violations := ProfilePolicyViolations(profile)
	allowed := map[string]bool{}
	for _, perm := range perms {
		if perm.Kind == permission.KindNetwork && perm.Access == permission.AccessConnect {
			allowed[perm.Scope] = true
		}
	}
	for _, scope := range profile.NetworkAllowlist {
		if !allowed[scope] {
			violations = append(violations, "the network allowlist names "+scope+", which no declaration permits")
		}
	}
	return violations
}

// ProfilePolicyViolations reports the policy itself: both defaults must deny.
// It is used when nothing is declared, so there is no allowlist to compare.
func ProfilePolicyViolations(profile SandboxProfile) []string {
	var violations []string
	if !profile.DenyNetworkByDefault {
		violations = append(violations, "the network policy does not deny by default")
	}
	if !profile.DenyFilesystemByDefault {
		violations = append(violations, "the filesystem policy does not deny by default")
	}
	return violations
}
