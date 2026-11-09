package applicationview

import (
	"tei/internal/core"
	"tei/internal/semantic"
)

// StandardBusinessRef is one standard business Data the application's used
// semantics declare, with the attributes each binder reads.
//
// The projection is a reading aid: it shows which Data the standard business
// roles are on, so a reader does not have to search the semantic files for
// them. It does not validate the roles (internal/validation does) and it does
// not read rows.
type StandardBusinessRef struct {
	// Kind names the binder: "tax", "exchange", "numbering", or "holiday".
	Kind core.CanonicalID `json:"kind"`
	Data core.CanonicalID `json:"data"`
	// Attributes lists the attribute IDs the binder reads, in declaration
	// order. An attribute whose role is not in the binder's vocabulary is not
	// listed, so an incomplete declaration is visible as a short list.
	Attributes []core.CanonicalID `json:"attributes,omitempty"`
}

// standardBusinessRoles maps a binder kind to the roles it reads. The order is
// the order the list is built in, so a reader sees required roles before
// optional ones.
var standardBusinessRoles = map[core.CanonicalID][]string{
	"tax": {
		"tax-applies-from", "tax-applies-until", "tax-value", "tax-rounding",
		"tax-category",
	},
	"exchange": {
		"currency-rate-currency", "currency-rate-fiscal-year",
		"currency-rate-month-sequence", "currency-rate-middle",
		"currency-rate-buying", "currency-rate-selling",
	},
	"numbering": {
		"numbering-counter", "numbering-digits", "numbering-prefix",
		"numbering-check", "numbering-overflow",
	},
	"holiday": {"holiday-date"},
	// Settings are listed after the business binders, so a reader sees a Data
	// that holds settings as settings and not as a rate table.
	"setting": {"setting-key", "setting-value", "setting-user"},
}

// standardBusiness returns the standard business Data the namespaces declare.
// A Data is reported once, under the first kind whose vocabulary it matches, so
// a reserved name such as "tax-rate-data" does not have to be declared twice.
func standardBusiness(namespaces []semantic.Namespace) []StandardBusinessRef {
	var out []StandardBusinessRef
	for _, namespace := range namespaces {
		for _, data := range namespace.Data {
			for _, kind := range []core.CanonicalID{"tax", "exchange", "numbering", "holiday", "setting"} {
				roles := standardBusinessRoles[kind]
				attributes := attributesForRoles(data, roles)
				if len(attributes) == 0 {
					continue
				}
				out = append(out, StandardBusinessRef{Kind: kind, Data: data.ID, Attributes: attributes})
				break
			}
		}
	}
	return out
}

// attributesForRoles returns the attributes whose role is one of the roles, in
// the order the roles are listed.
func attributesForRoles(data semantic.Data, roles []string) []core.CanonicalID {
	var out []core.CanonicalID
	for _, role := range roles {
		for _, attribute := range data.Attributes {
			if attribute.Role == role {
				out = append(out, attribute.ID)
				break
			}
		}
	}
	return out
}
