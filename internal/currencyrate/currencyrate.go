// Package currencyrate converts amounts between a system currency and another
// currency using declared rate rows. It is a reference implementation: the rate
// rows and the system currency come from the caller, so the package does not read
// a database or a clock.
//
// X-TEA's Session chose the rate by currency, fiscal year, and month, selected
// buying, selling, or middle by a string argument ("TTB", "TTS", anything else),
// returned exactly 1 for the system currency, and fell back from the monthly
// table to the annual table when the monthly rate read as zero
// (D:Session.java L2253–2313). A zero rate is also a legitimate value in some
// data, so the fallback could not tell "no row" from "a zero rate". Here a
// missing row is an error, the rate kind is a declared name, and the fallback is
// an explicit choice of the caller.
package currencyrate

import (
	"errors"
	"fmt"
	"math"
	"sort"
	"time"
)

// Kind names which rate of a row is used. X-TEA passed "TTB" or "TTS" as a string
// and treated anything else as the middle rate (D:Session.java L2265–2277), so a
// typo silently selected the middle rate.
type Kind string

const (
	// KindMiddle is the middle rate.
	KindMiddle Kind = "middle"
	// KindBuying is the rate at which the system buys the other currency.
	KindBuying Kind = "buying"
	// KindSelling is the rate at which the system sells the other currency.
	KindSelling Kind = "selling"
)

// Direction names which way the conversion goes. X-TEA had no argument for it:
// the caller decided by how it used the number.
type Direction string

const (
	// DirectionForeignToSystem multiplies by the rate.
	DirectionForeignToSystem Direction = "foreign-to-system"
	// DirectionSystemToForeign divides by the rate.
	DirectionSystemToForeign Direction = "system-to-foreign"
)

// Rounding names how a fractional converted amount becomes a whole amount.
type Rounding string

const (
	RoundingFloor    Rounding = "floor"
	RoundingCeiling  Rounding = "ceiling"
	RoundingHalfUp   Rounding = "half-up"
	RoundingHalfEven Rounding = "half-even"
)

// ErrInvalidRate reports a rate set that cannot answer. It is a configuration
// failure, not a business result.
var ErrInvalidRate = errors.New("currency rate cannot answer")

// Rate is one declared rate row. The period is a fiscal year and a month
// sequence, like X-TEA's DTNEND and DTMSEQ columns; the row's three values are
// the middle, buying, and selling rates.
type Rate struct {
	Currency      string
	FiscalYear    int
	MonthSequence int
	Middle        float64
	Buying        float64
	Selling       float64
}

// Set is the rate rows, the system currency, and the rules a caller resolved
// from a declaration.
type Set struct {
	// SystemCurrency is the currency the amounts are kept in. It is declared,
	// not read from a system variable (X-TEA used "SYSTEM_CURRENCY", L2284).
	SystemCurrency string
	Kind           Kind
	Direction      Direction
	Rounding       Rounding
	Locale         *time.Location
	Rates          []Rate
	// Fallback names what happens when a monthly rate is missing. An empty
	// value means FallbackNone, so a caller does not get a fallback it did not
	// ask for.
	Fallback Fallback
}

func (s Set) validate() error {
	if s.SystemCurrency == "" {
		return fmt.Errorf("%w: the system currency is not declared", ErrInvalidRate)
	}
	switch s.Kind {
	case KindMiddle, KindBuying, KindSelling:
	case "":
		return fmt.Errorf("%w: the rate kind is not declared", ErrInvalidRate)
	default:
		return fmt.Errorf("%w: unsupported rate kind %q", ErrInvalidRate, s.Kind)
	}
	switch s.Direction {
	case DirectionForeignToSystem, DirectionSystemToForeign:
	case "":
		return fmt.Errorf("%w: the conversion direction is not declared", ErrInvalidRate)
	default:
		return fmt.Errorf("%w: unsupported conversion direction %q", ErrInvalidRate, s.Direction)
	}
	switch s.Rounding {
	case RoundingFloor, RoundingCeiling, RoundingHalfUp, RoundingHalfEven:
	case "":
		return fmt.Errorf("%w: the rounding rule is not declared", ErrInvalidRate)
	default:
		return fmt.Errorf("%w: unsupported rounding rule %q", ErrInvalidRate, s.Rounding)
	}
	if s.Locale == nil {
		return fmt.Errorf("%w: the time zone is not declared", ErrInvalidRate)
	}
	return nil
}

// options returns how many rows match one currency and one month. It is the same
// shape the tax lookup uses: the answer is one row, none, or more than one.
func (s Set) rows(currency string, fiscalYear, monthSequence int) []Rate {
	var matches []Rate
	for _, rate := range s.Rates {
		if rate.Currency != currency || rate.FiscalYear != fiscalYear || rate.MonthSequence != monthSequence {
			continue
		}
		matches = append(matches, rate)
	}
	sort.Slice(matches, func(i, j int) bool { return matches[i].Currency < matches[j].Currency })
	return matches
}

// RateFor returns the rate that applies to the currency and the date's period.
// A row is identified by the period key the calendar derives, so the caller
// supplies the fiscal year and month sequence rather than a month number.
func (s Set) RateFor(currency string, fiscalYear, monthSequence int) (float64, error) {
	if err := s.validate(); err != nil {
		return 0, err
	}
	if currency == s.SystemCurrency {
		// The system currency converts to itself at 1. X-TEA returned 1 here
		// before reading a table (L2284), which is the one shortcut that is
		// still correct: no row could say otherwise.
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

func (s Set) value(rate Rate) (float64, error) {
	switch s.Kind {
	case KindMiddle:
		return rate.Middle, nil
	case KindBuying:
		return rate.Buying, nil
	case KindSelling:
		return rate.Selling, nil
	default:
		return 0, fmt.Errorf("%w: unsupported rate kind %q", ErrInvalidRate, s.Kind)
	}
}

// Convert converts an amount. The direction is declared, so the caller does not
// decide by how it uses the number.
func (s Set) Convert(amount float64, currency string, fiscalYear, monthSequence int) (int64, error) {
	if err := s.validate(); err != nil {
		return 0, err
	}
	rate, err := s.RateFor(currency, fiscalYear, monthSequence)
	if err != nil {
		return 0, err
	}
	if rate == 0 {
		// A zero rate cannot convert. X-TEA could not tell this from "no row"
		// and fell back to the annual table (L2296–2298); here it is reported.
		return 0, fmt.Errorf("%w: the rate for %s is zero", ErrInvalidRate, currency)
	}
	converted := amount * rate
	if s.Direction == DirectionSystemToForeign {
		converted = amount / rate
	}
	return s.Round(converted)
}

// Round applies the declared rounding rule to a value.
func (s Set) Round(value float64) (int64, error) {
	if err := s.validate(); err != nil {
		return 0, err
	}
	switch s.Rounding {
	case RoundingFloor:
		return int64(math.Floor(value)), nil
	case RoundingCeiling:
		return int64(math.Ceil(value)), nil
	case RoundingHalfUp:
		return int64(math.Round(value)), nil
	case RoundingHalfEven:
		return int64(math.RoundToEven(value)), nil
	default:
		return 0, fmt.Errorf("%w: unsupported rounding rule %q", ErrInvalidRate, s.Rounding)
	}
}

// MissingPeriods lists the periods for which a currency has no row between two
// period keys, so a report can name the gaps instead of falling back silently.
func (s Set) MissingPeriods(currency string, fromFiscalYear, fromMonth, thruFiscalYear, thruMonth int) []Rate {
	if s.Locale == nil || fromFiscalYear > thruFiscalYear {
		return nil
	}
	var out []Rate
	fiscalYear, month := fromFiscalYear, fromMonth
	for {
		if len(s.rows(currency, fiscalYear, month)) == 0 {
			out = append(out, Rate{Currency: currency, FiscalYear: fiscalYear, MonthSequence: month})
		}
		if fiscalYear == thruFiscalYear && month == thruMonth {
			return out
		}
		month++
		if month > 12 {
			month = 1
			fiscalYear++
		}
		if fiscalYear > thruFiscalYear {
			return out
		}
	}
}
