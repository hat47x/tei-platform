package access

import (
	"fmt"

	"tei/internal/core"
	"tei/internal/permission"
)

// Resolution maps a declared scope onto the concrete place one run uses.
//
// A declaration is written in the vocabulary of meaning: `migration-workdir`,
// not `/tmp/abc`. A run happens somewhere concrete. The mapping between the two
// is a decision about this run, so it says who made it and is recorded with the
// issuance. Without that record, a run that writes to `/tmp/abc` cannot be traced
// back to the declaration that allowed it.
type Resolution struct {
	ResolvedBy string `json:"resolvedBy"`
	Declared   string `json:"declared"`
	Root       string `json:"root"`
}

// Ground reports whether an issuance stays inside what the declarations allow.
//
// The declarations say what the environment permits at all; the issuance says
// what this one run may touch. The second must stay inside the first, otherwise
// the declaration is decoration: whoever can issue can widen it. Both are
// compared after the declaration is resolved, because a declaration names a place
// in the vocabulary of meaning and an issuance names a concrete path.
//
// Nothing here proves a third party approved the issuance. A resolution and an
// issuance both carry a name, and a name can be written by anyone able to write
// the file.
func Ground(issued, declared Document, resolutions []Resolution, capabilityID core.CanonicalID) error {
	if err := AcceptIssuedBy(issued, capabilityID); err != nil {
		return err
	}
	if len(issued.Grants) == 0 {
		return nil
	}
	if len(declared.Grants) == 0 {
		return fmt.Errorf("%d grant(s) were issued but no declaration allows any of them: an issuance needs something to stay inside of", len(issued.Grants))
	}
	if err := checkResolutions(declared, resolutions, capabilityID); err != nil {
		return err
	}
	if !Declares(declared, capabilityID) {
		return fmt.Errorf("no declaration names %s, so nothing it issues can be grounded", capabilityID)
	}
	for index, entry := range issued.Grants {
		if !groundsIn(declared, resolutions, entry) {
			return fmt.Errorf("grant %d (%s %s %s) is not inside anything the declaration allows", index+1, entry.Kind, entry.Access, entry.Scope)
		}
	}
	return nil
}

// checkResolutions refuses a resolution that does not name a declared scope, that
// names no resolver, or that the capability made for itself.
func checkResolutions(declared Document, resolutions []Resolution, capabilityID core.CanonicalID) error {
	declaredScopes := map[string]bool{}
	for _, entry := range declared.Grants {
		declaredScopes[entry.Scope] = true
	}
	for index, resolution := range resolutions {
		if resolution.Declared == "" || resolution.Root == "" {
			return fmt.Errorf("resolution %d does not name both the declared scope and its root", index+1)
		}
		if resolution.ResolvedBy == "" {
			return fmt.Errorf("resolution %d does not name who resolved it", index+1)
		}
		if resolution.ResolvedBy == string(capabilityID) {
			return fmt.Errorf("resolution %d was made by %s, which is the capability that wants to use it", index+1, capabilityID)
		}
		if !declaredScopes[resolution.Declared] {
			return fmt.Errorf("resolution %d maps %s, which no declaration names", index+1, resolution.Declared)
		}
	}
	return nil
}

// groundsIn reports whether one issued grant stays inside a declared grant for
// the same capability, either as written or through a resolution of the declared
// scope. A declaration for another capability grounds nothing: the environment
// said what that capability may touch, not what this one may.
func groundsIn(declared Document, resolutions []Resolution, entry Entry) bool {
	required := permission.Spec{Kind: permission.Kind(entry.Kind), Access: permission.Access(entry.Access), Scope: entry.Scope}
	for _, candidate := range declared.Grants {
		if candidate.CapabilityID != entry.CapabilityID {
			continue
		}
		// Kind must match exactly, but access must not: a declaration of
		// read-write allows an issuance that asks only for write. Comparing access
		// for equality here would refuse exactly the case that makes the
		// declaration useful.
		if candidate.Kind != entry.Kind {
			continue
		}
		granted := permission.Spec{Kind: permission.Kind(candidate.Kind), Access: permission.Access(candidate.Access), Scope: candidate.Scope}
		if permission.Covers(granted, required) {
			return true
		}
		for _, resolution := range resolutions {
			if resolution.Declared != candidate.Scope {
				continue
			}
			resolved := permission.Spec{Kind: permission.Kind(candidate.Kind), Access: permission.Access(candidate.Access), Scope: resolution.Root}
			if permission.Covers(resolved, required) {
				return true
			}
		}
	}
	return false
}

// Declares reports which capabilities the document has declarations for. It lets
// a caller say "no declaration names this capability" instead of "the grant is
// outside the declaration", which are different problems.
func Declares(declared Document, capabilityID core.CanonicalID) bool {
	for _, entry := range declared.Grants {
		if entry.CapabilityID == capabilityID {
			return true
		}
	}
	return false
}
