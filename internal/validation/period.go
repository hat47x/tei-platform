package validation

import (
	"tei/internal/core"
	"tei/internal/ir"
	"tei/internal/subject"
)

// validateSemanticPeriods validates the declared meaning of a period: the data
// it belongs to, the business date attribute, the three states and the
// capabilities it names. It does not check whether a close has run.
func validateSemanticPeriods(w ir.World) []Diagnostic {
	data := map[core.CanonicalID]struct {
		attributes map[core.CanonicalID]bool
	}{}
	for _, namespace := range w.Semantics {
		for _, element := range namespace.Data {
			attributes := map[core.CanonicalID]bool{}
			for _, attribute := range element.Attributes {
				attributes[attribute.ID] = true
			}
			data[element.ID] = struct {
				attributes map[core.CanonicalID]bool
			}{attributes: attributes}
		}
	}
	capabilities := map[core.CanonicalID]bool{}
	for _, namespace := range w.Semantics {
		for _, capability := range namespace.Capabilities {
			capabilities[capability.ID] = true
		}
	}

	var out []Diagnostic
	for _, namespace := range w.Semantics {
		for _, period := range namespace.Periods {
			subj := subject.Subject{Kind: "semantic-period", ID: string(period.ID)}

			element, ok := data[period.Data]
			if !ok || period.Data == "" {
				out = append(out, errFor(subj, "PERIOD_UNKNOWN_DATA", "period %s refers to unknown semantic data %s", period.ID, period.Data))
			} else if !element.attributes[period.BusinessDate] {
				out = append(out, errFor(subj, "PERIOD_UNKNOWN_BUSINESS_DATE", "period %s business date %s is not an attribute of %s", period.ID, period.BusinessDate, period.Data))
			}

			states := map[string]bool{}
			for _, state := range period.States {
				states[state] = true
			}
			for _, required := range []string{"open", "closing", "closed"} {
				if !states[required] {
					out = append(out, errFor(subj, "PERIOD_STATES_INCOMPLETE", "period %s must declare the %q state", period.ID, required))
				}
			}

			for _, reference := range []core.CanonicalID{period.CloseRequires, period.Correction} {
				if reference == "" {
					continue
				}
				if !capabilities[reference] {
					out = append(out, errFor(subj, "PERIOD_UNKNOWN_CAPABILITY", "period %s refers to unknown capability %s", period.ID, reference))
				}
			}

			for _, rejected := range period.ClosedRejects {
				switch rejected {
				case "create", "update", "delete":
				default:
					out = append(out, errFor(subj, "PERIOD_UNKNOWN_CLOSED_REJECT", "period %s rejects unknown operation %q", period.ID, rejected))
				}
			}
		}
	}
	return out
}
