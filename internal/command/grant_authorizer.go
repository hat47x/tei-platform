package command

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"tei/internal/core"
	"tei/internal/permission"
)

// ErrPermissionNotGranted marks a denial that the permission rules decided, so a
// caller can tell "the grant does not cover the declaration" from "the grant
// source could not be read".
var ErrPermissionNotGranted = errors.New("permission is not granted")

// GrantAuthorizer checks a capability's declared permissions against the grants
// issued for this run. It is a reference authorizer: the grants are supplied by
// the host, and this package neither stores nor issues them.
//
// A capability that declares no permissions is refused. Silence is not a grant:
// a capability must say what it needs before it can be called, so a host cannot
// widen access by omitting a declaration.
type GrantAuthorizer struct {
	// Declared lists the permissions each capability needs. A capability that is
	// absent declares nothing and is refused.
	Declared map[core.CanonicalID][]permission.Spec
	// Grants are the permissions issued to this run, independent of the menu or
	// the Pattern that advertises the command.
	Grants []permission.Spec
}

// Authorize refuses the call unless every declared permission of the capability
// is covered by a grant.
func (a GrantAuthorizer) Authorize(_ context.Context, request Request) error {
	required, declared := a.Declared[request.Capability]
	if !declared {
		return fmt.Errorf("capability %s declares no permissions: %w", request.Capability, ErrPermissionNotGranted)
	}
	for _, spec := range required {
		if err := spec.Validate(); err != nil {
			return fmt.Errorf("capability %s declares an invalid permission: %w", request.Capability, err)
		}
	}
	missing := permission.Missing(a.Grants, required)
	if len(missing) == 0 {
		return nil
	}
	return fmt.Errorf("capability %s is not granted %s: %w", request.Capability, describePermissions(missing), ErrPermissionNotGranted)
}

// describePermissions renders permissions in a stable order so a denial reads
// the same on every run and can be compared in tests.
func describePermissions(specs []permission.Spec) string {
	keys := make([]string, 0, len(specs))
	for _, spec := range specs {
		keys = append(keys, spec.Key())
	}
	sort.Strings(keys)
	return strings.Join(keys, ", ")
}
