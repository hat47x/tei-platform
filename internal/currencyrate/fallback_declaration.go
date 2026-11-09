package currencyrate

import (
	"fmt"

	"tei/internal/core"
	"tei/internal/ir"
)

// RoleFallback is the optional role that names the fallback rule in the
// declaration. X-TEA decided the fallback in code by comparing the monthly rate
// against zero (D:Session.java L2296–2298), so the rule was not visible in the
// model. Here the rule is a declared value, and a misspelt value is refused.
const RoleFallback = "currency-rate-fallback"

// FallbackDeclaration is the fallback rule an exchange rate Data declares.
type FallbackDeclaration struct {
	DataID    core.CanonicalID
	Attribute core.CanonicalID
}

// ResolveFallback reads the fallback role from the Data declaration.
//
// A Data that does not declare the role has no rule, and that is not an error:
// a rate table that keeps only the rates it needs does not have to say "no
// fallback". The rule itself is a value in the rows, read by FallbackRows.
func ResolveFallback(world ir.World, dataID core.CanonicalID) (FallbackDeclaration, error) {
	for _, namespace := range world.Semantics {
		for _, data := range namespace.Data {
			if data.ID != dataID {
				continue
			}
			for _, attribute := range data.Attributes {
				if attribute.Role == RoleFallback {
					return FallbackDeclaration{DataID: data.ID, Attribute: attribute.ID}, nil
				}
			}
			return FallbackDeclaration{DataID: data.ID}, nil
		}
	}
	return FallbackDeclaration{}, fmt.Errorf("exchange rate data %s is not declared", dataID)
}

// FallbackRows reads the rule from the rows. Every row must carry the same rule,
// because one Data describes one fallback: two rules in one table would mean the
// answer depends on row order.
func (d FallbackDeclaration) FallbackRows(rows []map[core.CanonicalID]string) (Fallback, error) {
	if d.Attribute == "" {
		return FallbackNone, nil
	}
	rule := Fallback("")
	for index, row := range rows {
		value := row[d.Attribute]
		switch Fallback(value) {
		case "":
			return "", fmt.Errorf("exchange rate row %d: %s is empty", index+1, d.Attribute)
		case FallbackNone, FallbackAnnual:
		default:
			return "", fmt.Errorf("exchange rate row %d: %s: %q is not one of %q or %q", index+1, d.Attribute, value, FallbackNone, FallbackAnnual)
		}
		if rule != "" && rule != Fallback(value) {
			return "", fmt.Errorf("exchange rate rows disagree on the fallback: %q and %q", rule, value)
		}
		rule = Fallback(value)
	}
	if rule == "" {
		return FallbackNone, nil
	}
	return rule, nil
}
