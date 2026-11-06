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
// declared permissions permit: a policy that does not deny by default, a network
// destination or secret handle that no declaration names, or a mount whose scope
// is not declared. An empty result means the profile stays within the
// declarations.
func ProfileExceedsDeclarations(profile SandboxProfile, perms []permission.Spec) []string {
	derived, _ := ProfileFromPermissions(perms)
	allowed := map[string]bool{}
	for _, scope := range derived.NetworkAllowlist {
		allowed["network\x00"+scope] = true
	}
	secrets := map[string]bool{}
	for _, scope := range derived.SecretHandles {
		secrets["secret\x00"+scope] = true
	}
	files := map[string]bool{}
	for _, perm := range perms {
		if perm.Kind == permission.KindFilesystem {
			files[perm.Scope] = true
		}
	}

	var violations []string
	if !profile.DenyNetworkByDefault {
		violations = append(violations, "the network policy does not deny by default")
	}
	if !profile.DenyFilesystemByDefault {
		violations = append(violations, "the filesystem policy does not deny by default")
	}
	for _, scope := range profile.NetworkAllowlist {
		if !allowed["network\x00"+scope] {
			violations = append(violations, "the network allowlist names "+scope+", which no declaration permits")
		}
	}
	for _, handle := range profile.SecretHandles {
		if !secrets["secret\x00"+handle] {
			violations = append(violations, "the secret handle "+handle+" is not declared")
		}
	}
	for _, mount := range profile.Mounts {
		if !files[mount.Scope] {
			violations = append(violations, "the mount "+mount.Scope+" is not declared")
		}
	}
	for _, injection := range profile.SecretInjections {
		if !secrets["secret\x00"+injection.Scope] {
			violations = append(violations, "the secret injection "+injection.Scope+" is not declared")
		}
	}
	return violations
}
