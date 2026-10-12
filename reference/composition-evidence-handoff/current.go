package handoff

import "tei/internal/composition"

// ReconcileCurrent checks the current registry before linking any external
// evidence references. It deliberately does NOT authorize execution: a
// concurrent registry update may occur immediately after this observation,
// and a trusted Host must revalidate on admission and invocation.
//
// Using Reconcile without this wrapper remains useful for historical,
// explicitly revision-scoped analysis, but MUST NOT be presented as current.
func ReconcileCurrent(
	registry *composition.Registry,
	set composition.WiringSetFinding,
	refs []EvidenceReference,
) HandoffReport {
	if registry == nil || !registry.ProposalCurrent(set) {
		return HandoffReport{
			Revision: set.Revision,
			Scope:    set.Scope,
			Blocked:  true,
		}
	}
	return Reconcile(set, refs)
}
