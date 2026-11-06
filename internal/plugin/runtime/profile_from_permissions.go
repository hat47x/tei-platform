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
