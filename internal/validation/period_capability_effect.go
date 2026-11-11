package validation

import (
	"tei/internal/core"
	"tei/internal/ir"
	"tei/internal/subject"
)

// declaredPeriodCapability is one capability a period names, with the role that
// declaration plays. The role is kept so a diagnostic can say which declaration was
// wrong instead of leaving the reader to guess.
type declaredPeriodCapability struct {
	Role       string
	Capability core.CanonicalID
}

// periodEffectEntry is one period's declared capabilities beside the data the period
// governs.
type periodEffectEntry struct {
	ID            core.CanonicalID
	Data          core.CanonicalID
	CloseRequires core.CanonicalID
	Correction    core.CanonicalID
}

func (p periodEffectEntry) declarations() []declaredPeriodCapability {
	var out []declaredPeriodCapability
	if p.CloseRequires != "" {
		out = append(out, declaredPeriodCapability{Role: "close-requires", Capability: p.CloseRequires})
	}
	if p.Correction != "" {
		out = append(out, declaredPeriodCapability{Role: "correction", Capability: p.Correction})
	}
	return out
}

// capabilityEffectEntry is one capability's effect: which Data it changes.
type capabilityEffectEntry struct {
	ID        core.CanonicalID
	Namespace core.CanonicalID
	Data      core.CanonicalID
	Kind      string
}

// validatePeriodCapabilityEffects checks that the capabilities a period names can
// act on the period's data.
//
// A period declares `correction via <capability>`, meaning a change to this period
// goes through that capability. Validation checked that the capability exists but
// not that it effects the period's data, so a period could name a capability for
// another Data: the correction rule would then let through a change that cannot
// touch the period, and refuse the one that can.
//
// A capability that declares no effect is left alone: the declaration says nothing
// about which Data it changes, so there is nothing to contradict. Only a declared
// effect on a different Data is reported.
func validatePeriodCapabilityEffectsOf(w ir.World) []Diagnostic {
	var periods []periodEffectEntry
	effects := map[core.CanonicalID]capabilityEffectEntry{}
	known := map[core.CanonicalID]bool{}
	for _, namespace := range w.Semantics {
		for _, capability := range namespace.Capabilities {
			known[capability.ID] = true
			if capability.Effect == nil {
				continue
			}
			effects[capability.ID] = capabilityEffectEntry{
				ID:        capability.ID,
				Namespace: namespace.ID,
				Data:      capability.Effect.Data,
				Kind:      string(capability.Effect.Kind),
			}
		}
		for _, period := range namespace.Periods {
			periods = append(periods, periodEffectEntry{
				ID:            period.ID,
				Data:          period.Data,
				CloseRequires: period.CloseRequires,
				Correction:    period.Correction,
			})
		}
	}

	var out []Diagnostic
	for _, period := range periods {
		if period.Data == "" {
			// A period with no data has no effect to compare against; the missing
			// data is its own diagnostic.
			continue
		}
		subj := subject.Subject{Kind: "semantic-period", ID: string(period.ID)}
		for _, declaration := range period.declarations() {
			if !known[declaration.Capability] {
				// The unknown-capability check reports this; reporting it twice
				// would be the same cause in two places.
				continue
			}
			effect, hasEffect := effects[declaration.Capability]
			if !hasEffect {
				// A capability that declares no effect is not refused here: the
				// declaration says nothing about which Data it changes, so there is
				// nothing to contradict. Reading "no effect" as "wrong Data" would
				// invent a rule the model does not state.
				continue
			}
			if effect.Data != period.Data {
				out = append(out, errFor(subj, "PERIOD_CAPABILITY_EFFECTS_OTHER_DATA",
					"period %s names %s capability %s that effects %s, not the period's data %s",
					period.ID, declaration.Role, declaration.Capability, effect.Data, period.Data))
			}
		}
	}
	return out
}
