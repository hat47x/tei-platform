package currencyrate

import "fmt"

// Fallback names what happens when a monthly rate is missing. X-TEA fell back
// from the monthly table to the annual table whenever the monthly rate read as
// zero (D:Session.java L2296–2298), which could not tell "no row" from "a zero
// rate". Here the fallback is a declaration: with FallbackNone a missing row is
// reported, and with FallbackAnnual the caller asked for the annual rate.
type Fallback string

const (
	// FallbackNone reports a missing monthly rate. It is the default.
	FallbackNone Fallback = "none"
	// FallbackAnnual uses the annual row of the same currency and fiscal year
	// when the monthly row is missing. The annual row is the row whose month
	// sequence is AnnualMonthSequence.
	FallbackAnnual Fallback = "annual"
)

// AnnualMonthSequence marks the annual row in a shared rate Data. X-TEA kept the
// annual table separate; here the row is told apart by its month sequence, so one
// Data can hold both and the fallback reads the same rows.
const AnnualMonthSequence = 0

func (s Set) validateFallback() error {
	switch s.Fallback {
	case "", FallbackNone, FallbackAnnual:
		return nil
	default:
		return fmt.Errorf("%w: unsupported fallback %q", ErrInvalidRate, s.Fallback)
	}
}

func (s Set) fallback() Fallback {
	if s.Fallback == "" {
		return FallbackNone
	}
	return s.Fallback
}

// RateForWithFallback returns the rate for the period, using the declared
// fallback when the monthly row is missing.
//
// The annual row is read only when the declaration asks for it. A missing annual
// row is still reported, so a fallback that cannot answer is not silently zero.
func (s Set) RateForWithFallback(currency string, fiscalYear, monthSequence int) (float64, error) {
	if err := s.validateFallback(); err != nil {
		return 0, err
	}
	if s.fallback() == FallbackAnnual && monthSequence != AnnualMonthSequence {
		if _, monthlyErr := s.rateForNoFallback(currency, fiscalYear, monthSequence); monthlyErr != nil {
			annual, err := s.rateForNoFallback(currency, fiscalYear, AnnualMonthSequence)
			if err != nil {
				return 0, fmt.Errorf("%w: no monthly or annual rate for %s in fiscal year %d", ErrInvalidRate, currency, fiscalYear)
			}
			return annual, nil
		}
	}
	return s.rateForNoFallback(currency, fiscalYear, monthSequence)
}

// rateForNoFallback is the plain lookup: it never substitutes another period.
func (s Set) rateForNoFallback(currency string, fiscalYear, monthSequence int) (float64, error) {
	if err := s.validate(); err != nil {
		return 0, err
	}
	if currency == s.SystemCurrency {
		return 1, nil
	}
	if currency == "" {
		return 0, fmt.Errorf("%w: the currency is not named", ErrInvalidRate)
	}
	matches := s.rows(currency, fiscalYear, monthSequence)
	switch len(matches) {
	case 0:
		return 0, fmt.Errorf("%w: no rate for %s in fiscal year %d month %d", ErrInvalidRate, currency, fiscalYear, monthSequence)
	case 1:
		return s.value(matches[0])
	default:
		return 0, fmt.Errorf("%w: %d rates for %s in fiscal year %d month %d",
			ErrInvalidRate, len(matches), currency, fiscalYear, monthSequence)
	}
}
