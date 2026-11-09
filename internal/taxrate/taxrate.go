// Package taxrate computes a tax amount from declared rate rows. It is a
// reference implementation: the rate rows are supplied by the caller, and the
// package does not read a database or a clock.
//
// X-TEA's Session read a tax table ordered by start date and used the newest row
// whose start date was not later than the target date (D:Session.java L2315–2334).
// It carried no end date, and it rounded by the sign of the amount: floor for a
// positive amount and ceiling for a negative one. Here the applicable span has
// both ends and the rounding is a declared rule, so a boundary and a correction
// are matters the declaration decides.
package taxrate

import (
	"errors"
	"fmt"
	"math"
	"sort"
	"time"
)

// Rounding names how a fractional tax amount becomes a whole amount. X-TEA's
// choice (floor for a positive amount, ceiling for a negative one) became two
// different rules for one declaration; here the declaration names one rule and
// it applies to both signs, so a negative correction cannot silently round the
// other way.
type Rounding string

const (
	// RoundingFloor rounds toward negative infinity.
	RoundingFloor Rounding = "floor"
	// RoundingCeiling rounds toward positive infinity.
	RoundingCeiling Rounding = "ceiling"
	// RoundingHalfUp rounds to the nearest, away from zero on a tie.
	RoundingHalfUp Rounding = "half-up"
	// RoundingHalfEven rounds to the nearest, to the even neighbour on a tie.
	RoundingHalfEven Rounding = "half-even"
)

// ErrInvalidRate reports a rate set that cannot answer. It is a configuration
// failure, not a business result.
var ErrInvalidRate = errors.New("tax rate cannot answer")

// Rate is one declared rate row. The span is inclusive of both ends, so a rate
// that applies on a single day names that day twice.
type Rate struct {
	// Category is the tax category the row applies to. An empty category is
	// the default row.
	Category string
	// AppliesFrom is the first date the rate applies to.
	AppliesFrom time.Time
	// AppliesUntil is the last date the rate applies to. A zero value means
	// the row has no declared end, and a row that is not the last one cannot
	// then be told from the next row; the lookup refuses that overlap.
	AppliesUntil time.Time
	// Value is the rate as a fraction: 0.10 is ten percent.
	Value float64
	// Rounding is the rounding the row itself declares. An empty value means
	// the row follows the set default, so one period can be corrected without
	// splitting the declaration.
	Rounding Rounding
}

// Set is the rate rows and the rounding rule a caller resolved from a
// declaration.
type Set struct {
	Category string
	Rounding Rounding
	Rates    []Rate
	Locale   *time.Location
}

func (s Set) validate() error {
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

func (s Set) normalize(date time.Time) time.Time {
	year, month, day := date.In(s.Locale).Date()
	return time.Date(year, month, day, 0, 0, 0, 0, s.Locale)
}

// RateFor returns the rate row that applies to the date.
//
// The lookup refuses two cases instead of picking a row: no row covers the date,
// and more than one row covers it. X-TEA picked the newest row by start date and
// never noticed an overlap, so a mistake in the rate table became a plausible
// number instead of a report.
func (s Set) RateFor(date time.Time) (Rate, error) {
	if err := s.validate(); err != nil {
		return Rate{}, err
	}
	target := s.normalize(date)
	var matches []Rate
	for _, rate := range s.Rates {
		if rate.Category != "" && rate.Category != s.Category {
			continue
		}
		if target.Before(s.normalize(rate.AppliesFrom)) {
			continue
		}
		if !rate.AppliesUntil.IsZero() && !target.Before(s.normalize(rate.AppliesUntil).AddDate(0, 0, 1)) {
			continue
		}
		matches = append(matches, rate)
	}
	switch len(matches) {
	case 0:
		return Rate{}, fmt.Errorf("%w: no rate applies on %s", ErrInvalidRate, target.Format("2006-01-02"))
	case 1:
		return matches[0], nil
	default:
		sort.Slice(matches, func(i, j int) bool { return matches[i].AppliesFrom.Before(matches[j].AppliesFrom) })
		return Rate{}, fmt.Errorf("%w: %d rates apply on %s (%s and %s)",
			ErrInvalidRate, len(matches), target.Format("2006-01-02"),
			matches[0].AppliesFrom.Format("2006-01-02"), matches[1].AppliesFrom.Format("2006-01-02"))
	}
}

// Amount returns the tax amount for the date and the amount. The amount is in
// the smallest unit the declaration uses; the fractional result is rounded by
// the declared rule.
func (s Set) Amount(date time.Time, amount float64) (int64, error) {
	rate, err := s.RateFor(date)
	if err != nil {
		return 0, err
	}
	return s.Round(amount * rate.Value)
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

// Overlaps reports the rate pairs whose spans both cover a date. It is the
// check a declaration needs before its rows can be used: the lookup refuses an
// overlap, and this check says which rows to correct.
func (s Set) Overlaps() [][2]Rate {
	if s.Locale == nil {
		return nil
	}
	var out [][2]Rate
	for i := 0; i < len(s.Rates); i++ {
		for j := i + 1; j < len(s.Rates); j++ {
			first, second := s.Rates[i], s.Rates[j]
			if first.Category != "" && second.Category != "" && first.Category != second.Category {
				continue
			}
			firstFrom, firstUntil := s.normalize(first.AppliesFrom), s.end(first)
			secondFrom, secondUntil := s.normalize(second.AppliesFrom), s.end(second)
			if firstUntil.Before(secondFrom) || secondUntil.Before(firstFrom) {
				continue
			}
			out = append(out, [2]Rate{first, second})
		}
	}
	return out
}

// end returns the last date a row covers. A row without a declared end covers
// every later date, which is why two open-ended rows always overlap.
func (s Set) end(rate Rate) time.Time {
	if rate.AppliesUntil.IsZero() {
		return time.Date(9999, 12, 31, 0, 0, 0, 0, s.Locale)
	}
	return s.normalize(rate.AppliesUntil)
}
