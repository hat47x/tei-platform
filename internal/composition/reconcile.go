package composition

import "errors"

// Reconciliation is a declarative lifecycle delta, not an imperative scheduler.
// The host must stage and verify new instances before switching any live route.
// StopOrder describes logical retirement, not permission to kill in-flight calls.
type Reconciliation struct {
	StopOrder  []string
	StartOrder []string
	Retained   []string
}

// PlanReconciliation expands changed service bindings to dependent consumers.
// Previous and next must be consecutive registry revisions. The plan does not
// grant authority, activate a process, or perform any irreversible effects.
func PlanReconciliation(previous, next Snapshot) (Reconciliation, error) {
	if next.Revision != previous.Revision+1 || next.Revision == 0 {
		return Reconciliation{}, errors.New("registry snapshots must be consecutive")
	}
	oldReady, newReady := map[string]Assessment{}, map[string]Assessment{}
	for _, a := range previous.Providers {
		if a.Eligible {
			oldReady[a.ProviderID] = a
		}
	}
	for _, a := range next.Providers {
		if a.Eligible {
			newReady[a.ProviderID] = a
		}
	}
	changed := map[string]bool{}
	for _, t := range next.Transitions {
		if t.From != t.To {
			changed[t.ProviderID] = true
		}
	}
	// An affected dependency invalidates an otherwise unchanged consumer scope.
	// The loop is bounded by the number of provider identities in the two graphs.
	for {
		updated := false
		for _, graph := range []map[string]Assessment{oldReady, newReady} {
			for id, a := range graph {
				if changed[id] {
					continue
				}
				for _, dependency := range a.Bindings {
					if changed[dependency.ProviderID] {
						changed[id] = true
						updated = true
						break
					}
				}
			}
		}
		if !updated {
			break
		}
	}
	var out Reconciliation
	for i := len(previous.ActivationOrder) - 1; i >= 0; i-- {
		id := previous.ActivationOrder[i]
		if _, ok := oldReady[id]; ok && (changed[id] || newReady[id].ProviderID == "") {
			out.StopOrder = append(out.StopOrder, id)
		}
	}
	for _, id := range next.ActivationOrder {
		if _, ok := newReady[id]; !ok {
			continue
		}
		if changed[id] || oldReady[id].ProviderID == "" {
			out.StartOrder = append(out.StartOrder, id)
		} else {
			out.Retained = append(out.Retained, id)
		}
	}
	return out, nil
}
