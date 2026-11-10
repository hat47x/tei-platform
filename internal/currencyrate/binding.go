// Package currencyrate binds a declared exchange rate Data to the rate rows the
// conversion uses.
//
// X-TEA read a monthly table by currency, fiscal year, and month sequence, and
// fell back to an annual table when the monthly rate read as zero
// (D:Session.java L2253–2313). Both tables were named by system variables and
// read by column name, so a renamed column or a missing fallback row was
// invisible. Here the attributes are declared by role, and the fallback is a
// decision the caller makes rather than a value the code compares against zero.
package currencyrate

import (
	"fmt"
	"strconv"

	"tei/internal/core"
	"tei/internal/ir"
)

// Attribute roles an exchange rate Data must declare.
const (
	RoleCurrency      = "currency-rate-currency"
	RoleFiscalYear    = "currency-rate-fiscal-year"
	RoleMonthSequence = "currency-rate-month-sequence"
	RoleMiddle        = "currency-rate-middle"
	RoleBuying        = "currency-rate-buying"
	RoleSelling       = "currency-rate-selling"
)

// Declaration is the attributes an exchange rate Data declares.
type Declaration struct {
	DataID        core.CanonicalID
	Currency      core.CanonicalID
	FiscalYear    core.CanonicalID
	MonthSequence core.CanonicalID
	Middle        core.CanonicalID
	Buying        core.CanonicalID
	Selling       core.CanonicalID
}

// Resolve reads the exchange rate declaration from the world.
//
// The currency, the fiscal year, the month sequence, and the middle rate are
// required. The buying and selling rates are optional, because a table that
// only keeps one rate is a legitimate declaration; asking for the missing kind
// then fails at lookup instead of returning zero.
func Resolve(world ir.World, dataID core.CanonicalID) (Declaration, error) {
	for _, namespace := range world.Semantics {
		for _, data := range namespace.Data {
			if data.ID != dataID {
				continue
			}
			declaration := Declaration{DataID: data.ID}
			for _, attribute := range data.Attributes {
				switch attribute.Role {
				case RoleCurrency:
					declaration.Currency = attribute.ID
				case RoleFiscalYear:
					declaration.FiscalYear = attribute.ID
				case RoleMonthSequence:
					declaration.MonthSequence = attribute.ID
				case RoleMiddle:
					declaration.Middle = attribute.ID
				case RoleBuying:
					declaration.Buying = attribute.ID
				case RoleSelling:
					declaration.Selling = attribute.ID
				}
			}
			var missing []string
			if declaration.Currency == "" {
				missing = append(missing, RoleCurrency)
			}
			if declaration.FiscalYear == "" {
				missing = append(missing, RoleFiscalYear)
			}
			if declaration.MonthSequence == "" {
				missing = append(missing, RoleMonthSequence)
			}
			if declaration.Middle == "" {
				missing = append(missing, RoleMiddle)
			}
			if len(missing) > 0 {
				return Declaration{}, fmt.Errorf("exchange rate data %s does not declare %v", dataID, missing)
			}
			return declaration, nil
		}
	}
	return Declaration{}, fmt.Errorf("exchange rate data %s is not declared", dataID)
}

// Bind turns stored rows into Rates. A row that does not parse names its
// position and the attribute, so a bad row is reported instead of skipped.
func (d Declaration) Bind(rows []map[core.CanonicalID]string) ([]Rate, error) {
	out := make([]Rate, 0, len(rows))
	for index, row := range rows {
		currency := row[d.Currency]
		if currency == "" {
			return nil, fmt.Errorf("exchange rate row %d: %s is empty", index+1, d.Currency)
		}
		fiscalYear, err := parseInt(row[d.FiscalYear], "fiscal year")
		if err != nil {
			return nil, fmt.Errorf("exchange rate row %d: %s: %w", index+1, d.FiscalYear, err)
		}
		monthSequence, err := parseInt(row[d.MonthSequence], "month sequence")
		if err != nil {
			return nil, fmt.Errorf("exchange rate row %d: %s: %w", index+1, d.MonthSequence, err)
		}
		// AnnualMonthSequence is the annual marker, so it is accepted beside the
		// twelve months. Refusing it would make every annual row unbindable, and
		// the declared fallback could never read one.
		if monthSequence != AnnualMonthSequence && (monthSequence < 1 || monthSequence > 12) {
			return nil, fmt.Errorf("exchange rate row %d: month sequence %d is outside 1..12 and is not the annual marker %d", index+1, monthSequence, AnnualMonthSequence)
		}
		middle, err := parseRate(row[d.Middle], "middle rate")
		if err != nil {
			return nil, fmt.Errorf("exchange rate row %d: %s: %w", index+1, d.Middle, err)
		}
		rate := Rate{Currency: currency, FiscalYear: fiscalYear, MonthSequence: monthSequence, Middle: middle}
		if d.Buying != "" {
			if value, ok := row[d.Buying]; ok && value != "" {
				buying, err := parseRate(value, "buying rate")
				if err != nil {
					return nil, fmt.Errorf("exchange rate row %d: %s: %w", index+1, d.Buying, err)
				}
				rate.Buying = buying
			}
		}
		if d.Selling != "" {
			if value, ok := row[d.Selling]; ok && value != "" {
				selling, err := parseRate(value, "selling rate")
				if err != nil {
					return nil, fmt.Errorf("exchange rate row %d: %s: %w", index+1, d.Selling, err)
				}
				rate.Selling = selling
			}
		}
		out = append(out, rate)
	}
	return out, nil
}

func parseInt(value, label string) (int, error) {
	if value == "" {
		return 0, fmt.Errorf("%s is empty", label)
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("%s %q is not a whole number", label, value)
	}
	return parsed, nil
}

func parseRate(value, label string) (float64, error) {
	if value == "" {
		return 0, fmt.Errorf("%s is empty", label)
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, fmt.Errorf("%s %q is not a number", label, value)
	}
	if parsed <= 0 {
		// A rate must be positive: zero cannot convert, and a negative rate is
		// a sign error. X-TEA compared the monthly rate against zero to decide
		// a fallback (同 L2296–2298), which made zero a control value.
		return 0, fmt.Errorf("%s %q is not positive", label, value)
	}
	return parsed, nil
}
