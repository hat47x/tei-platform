// Package taxrate binds a declared tax rate Data to the rate rows the lookup
// uses. The declaration part lives in Resolve: it reads which attributes carry
// the applicable span, the category, the rate, and the rounding. The runtime
// part lives in Bind: it turns stored rows into Rate values.
//
// X-TEA read a tax table by column name (DTSTART, VLTAXRATE) from a table whose
// name came from a system variable (D:Session.java L2315–2334). The columns were
// a storage detail and not a declaration, so a renamed column, a missing span,
// or a rounding that changed with the sign of the amount was invisible. Here the
// attribute roles are declared, and a Data that lacks one of them cannot be used
// as a rate table.
package taxrate

import (
	"fmt"
	"strconv"
	"time"

	"tei/internal/core"
	"tei/internal/ir"
)

// Attribute roles a tax rate Data must declare. They are the declaration's side
// of the contract: the binder refuses a Data that does not name them.
const (
	RoleAppliesFrom  = "tax-applies-from"
	RoleAppliesUntil = "tax-applies-until"
	RoleCategory     = "tax-category"
	RoleValue        = "tax-value"
	RoleRounding     = "tax-rounding"
)

// Declaration is the attributes a tax rate Data declares, resolved from the
// world. It does not hold rows: those are business records.
type Declaration struct {
	DataID       core.CanonicalID
	AppliesFrom  core.CanonicalID
	AppliesUntil core.CanonicalID
	Category     core.CanonicalID
	Value        core.CanonicalID
	Rounding     core.CanonicalID
}

// Resolve reads the tax rate declaration from the world.
//
// A Data used as a rate table must declare the span's two ends, the value, and
// the rounding. The category is optional, because a table of one category does
// not have to name it. A missing required role is reported with the role name,
// so the fix is the declaration and not a guess at a column.
func Resolve(world ir.World, dataID core.CanonicalID) (Declaration, error) {
	for _, namespace := range world.Semantics {
		for _, data := range namespace.Data {
			if data.ID != dataID {
				continue
			}
			declaration := Declaration{DataID: data.ID}
			for _, attribute := range data.Attributes {
				switch attribute.Role {
				case RoleAppliesFrom:
					declaration.AppliesFrom = attribute.ID
				case RoleAppliesUntil:
					declaration.AppliesUntil = attribute.ID
				case RoleCategory:
					declaration.Category = attribute.ID
				case RoleValue:
					declaration.Value = attribute.ID
				case RoleRounding:
					declaration.Rounding = attribute.ID
				}
			}
			var missing []string
			if declaration.AppliesFrom == "" {
				missing = append(missing, RoleAppliesFrom)
			}
			if declaration.AppliesUntil == "" {
				missing = append(missing, RoleAppliesUntil)
			}
			if declaration.Value == "" {
				missing = append(missing, RoleValue)
			}
			if declaration.Rounding == "" {
				missing = append(missing, RoleRounding)
			}
			if len(missing) > 0 {
				return Declaration{}, fmt.Errorf("tax rate data %s does not declare %v", dataID, missing)
			}
			return declaration, nil
		}
	}
	return Declaration{}, fmt.Errorf("tax rate data %s is not declared", dataID)
}

// Bind turns stored rows into Rates. Each row is keyed by attribute ID, as the
// reference store keeps it, so the binder does not depend on column names.
//
// A row whose value cannot be parsed is reported with its date, not skipped:
// skipping would leave a gap the lookup then reports somewhere else.
func (d Declaration) Bind(rows []map[core.CanonicalID]string, locale *time.Location) ([]Rate, error) {
	if locale == nil {
		return nil, fmt.Errorf("%w: the time zone is not declared", ErrInvalidRate)
	}
	out := make([]Rate, 0, len(rows))
	for index, row := range rows {
		from, err := parseDate(row[d.AppliesFrom], locale)
		if err != nil {
			return nil, fmt.Errorf("tax rate row %d: %s: %w", index+1, d.AppliesFrom, err)
		}
		rate := Rate{AppliesFrom: from}
		if raw, ok := row[d.AppliesUntil]; ok && raw != "" {
			until, err := parseDate(raw, locale)
			if err != nil {
				return nil, fmt.Errorf("tax rate row %d: %s: %w", index+1, d.AppliesUntil, err)
			}
			rate.AppliesUntil = until
		}
		if d.Category != "" {
			rate.Category = row[d.Category]
		}
		value, err := strconv.ParseFloat(row[d.Value], 64)
		if err != nil {
			return nil, fmt.Errorf("tax rate row %d: %s: %q is not a number", index+1, d.Value, row[d.Value])
		}
		rate.Value = value
		if raw := row[d.Rounding]; raw != "" {
			switch Rounding(raw) {
			case RoundingFloor, RoundingCeiling, RoundingHalfUp, RoundingHalfEven:
				rate.Rounding = Rounding(raw)
			default:
				return nil, fmt.Errorf("tax rate row %d: %s: unsupported rounding %q", index+1, d.Rounding, raw)
			}
		}
		out = append(out, rate)
	}
	return out, nil
}

// Set builds a Set from the bound rows and the rounding the caller declared.
// A row that carries its own rounding wins over the declared default, so a
// table can correct one period without splitting the declaration.
func (d Declaration) Set(rows []Rate, rounding Rounding, category string, locale *time.Location) Set {
	routes := make([]Rate, len(rows))
	copy(routes, rows)
	for index := range routes {
		if routes[index].Rounding == "" {
			routes[index].Rounding = rounding
		}
	}
	return Set{Category: category, Rounding: rounding, Rates: routes, Locale: locale}
}

func parseDate(value string, locale *time.Location) (time.Time, error) {
	if value == "" {
		return time.Time{}, fmt.Errorf("the date is empty")
	}
	parsed, err := time.ParseInLocation("2006-01-02", value, locale)
	if err != nil {
		return time.Time{}, fmt.Errorf("%q is not a date", value)
	}
	return parsed, nil
}

// RateRounding is the rounding a row carries, or the declared default when the
// row is silent. It exists so a reader can see which rule a row followed.
func (r Rate) RateRounding(defaultRounding Rounding) Rounding {
	if r.Rounding != "" {
		return r.Rounding
	}
	return defaultRounding
}
