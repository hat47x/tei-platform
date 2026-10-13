package webapicontract

import "tei/internal/core"

// ProviderReadSupport is an input supplied after the existing trusted
// Provider/Query binding is resolved. It is NOT an authorization grant or
// proof that a Provider enforces its own declarations.
type ProviderReadSupport struct {
	QueryID           core.CanonicalID
	PolicyRevision    string
	EnforcesScope     bool
	ProjectsFields    map[core.CanonicalID]bool
	Filters           map[core.CanonicalID]map[string]bool
	Sorts             map[core.CanonicalID]bool
	SupportsLimit     bool
}

// CheckReadDispatch is the last synthetic preflight before handing off a
// plan. Unsupported dynamic operations must not silently run as static SQL,
// or be emulated after applying pagination to an unscoped result.
func CheckReadDispatch(publication PublishedBrowse, plan ReadPlan, support ProviderReadSupport) error {
	if support.QueryID == "" || support.QueryID != publication.QueryID ||
		plan.PolicyRevision == "" || support.PolicyRevision != plan.PolicyRevision ||
		plan.ScopeID == "" || !support.EnforcesScope || !support.SupportsLimit ||
		plan.Limit < 1 || plan.Limit > 100 {
		return ErrRejected
	}
	fields := map[string]core.CanonicalID{}
	for _, f := range publication.Fields {
		if f.Name == "" || f.AttributeID == "" {
			return ErrRejected
		}
		if _, seen := fields[f.Name]; seen {
			return ErrRejected
		}
		fields[f.Name] = f.AttributeID
	}
	if len(plan.Selected) == 0 {
		return ErrRejected
	}
	for _, name := range plan.Selected {
		id, exists := fields[name]
		if !exists || !support.ProjectsFields[id] {
			return ErrRejected
		}
	}
	if plan.Filter != nil {
		id, exists := fields[plan.Filter.FieldID]
		if !exists || !support.Filters[id][plan.Filter.Operator] {
			return ErrRejected
		}
	}
	if plan.Order != nil {
		id, exists := fields[plan.Order.FieldID]
		if !exists || !support.Sorts[id] {
			return ErrRejected
		}
	}
	return nil
}
