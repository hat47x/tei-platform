// Package derivationcontext is a D12 read-only reference boundary.
// It fixes how a derived value may depend on evaluation context: an explicit,
// Host-supplied fixed clock and timezone instead of a hidden time.Now or a
// machine locale, so that a pure rule is deterministically re-evaluable.
//
// It does not define the trusted Invocation Context (#1118), operators
// (#988), lowering (#1121), cache versions (#1641) or implementations (#1694).
package derivationcontext

import (
	"errors"
	"fmt"
	"time"
)

var (
	ErrContextMissing = errors.New("derivation evaluation context is missing")
	ErrClockUnfixed   = errors.New("derivation evaluation clock is not fixed")
)

// Clock is the request-fixed evaluation clock supplied by the trusted Host for
// one request. Implementations must not consult the machine clock, the Process
// Environment or a session default, and every row in one request must observe
// the same instant.
type Clock interface {
	Now() (time.Time, error)
}

// FixedClock is a clock fixed to one instant.
type FixedClock struct {
	Instant time.Time
}

func (c FixedClock) Now() (time.Time, error) {
	if c.Instant.IsZero() {
		return time.Time{}, ErrClockUnfixed
	}
	return c.Instant, nil
}

// UnfixedClock is a Host clock that was not supplied. It fails closed rather
// than falling back to the local OS time.
type UnfixedClock struct{}

func (UnfixedClock) Now() (time.Time, error) { return time.Time{}, ErrClockUnfixed }

// Requirement is the non-business context a rule evaluation depends on. It is
// read from the accepted rule; it must not force every environment value onto
// rules that do not depend on it.
type Requirement struct {
	NeedsClock    bool
	NeedsTimezone bool
	NeedsLocale   bool
}

// Context carries the Host-supplied context values for one request. Business
// inputs (quantity, unit price) are not part of it.
type Context struct {
	Clock    Clock
	Location *time.Location
	Locale   string
}

// Check rejects a request whose declared context is absent. Presence only is
// validated here; the clock is read once by the operation that uses it.
func Check(req Requirement, ctx Context) error {
	if req.NeedsClock && ctx.Clock == nil {
		return fmt.Errorf("%w: clock", ErrContextMissing)
	}
	if req.NeedsTimezone && ctx.Location == nil {
		return fmt.Errorf("%w: timezone", ErrContextMissing)
	}
	if req.NeedsLocale && ctx.Locale == "" {
		return fmt.Errorf("%w: locale", ErrContextMissing)
	}
	return nil
}

// CalendarDate is the calendar date of an instant in an explicit location. A
// nil location fails closed; it never falls back to the machine locale.
func CalendarDate(instant time.Time, location *time.Location) (time.Time, error) {
	if instant.IsZero() {
		return time.Time{}, fmt.Errorf("%w: instant", ErrContextMissing)
	}
	if location == nil {
		return time.Time{}, fmt.Errorf("%w: timezone", ErrContextMissing)
	}
	local := instant.In(location)
	return time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, time.UTC), nil
}

// ElapsedDays returns the whole calendar days from start to the request-fixed
// evaluation date, both interpreted in the explicitly declared location. It
// reads the Host clock exactly once, so every row in one request agrees even
// when the instant crosses a date boundary in another timezone.
func ElapsedDays(start time.Time, ctx Context) (int, error) {
	if err := Check(Requirement{NeedsClock: true, NeedsTimezone: true}, ctx); err != nil {
		return 0, err
	}
	instant, err := ctx.Clock.Now()
	if err != nil {
		return 0, err
	}
	evaluationDate, err := CalendarDate(instant, ctx.Location)
	if err != nil {
		return 0, err
	}
	startDate, err := CalendarDate(start, ctx.Location)
	if err != nil {
		return 0, err
	}
	days := int(evaluationDate.Sub(startDate) / (24 * time.Hour))
	if days < 0 {
		return 0, fmt.Errorf("%w: evaluation date precedes start", ErrContextMissing)
	}
	return days, nil
}
