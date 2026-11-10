package currencyrate

import (
	"fmt"
	"time"

	"tei/internal/core"
	"tei/internal/ir"
)

// Table is one resolved rate Data with the rows it holds, so two Data can be
// joined without the caller passing rows and declarations separately.
//
// X-TEA kept a monthly table and an annual table as two tables named by system
// variables, and read whichever one a branch chose (D:Session.java L2253–2313).
// Nothing said the two tables were the same currency's rates at different
// granularity, so a table with the wrong granularity was read as if it were the
// right one. Here each table names its granularity, and the join says which
// periods the pair covers.
type Table struct {
	Declaration Declaration
	// Granularity says whether the rows are monthly (a month sequence of 1..12)
	// or annual (the annual month sequence). It is read from the rows, because a
	// Data's granularity is a property of what it stores.
	Granularity Granularity
	Rows        []Rate
}

// Granularity is how finely a rate table is kept.
type Granularity string

const (
	// GranularityMonthly keeps one row per month sequence.
	GranularityMonthly Granularity = "monthly"
	// GranularityAnnual keeps one row per fiscal year.
	GranularityAnnual Granularity = "annual"
)

// SortGranularity reads the granularity from the bound rows.
//
// A table whose rows mix monthly and annual sequences is refused: the two are
// different periods, and one table cannot answer for both without the caller
// guessing which row a year refers to.
func (d Declaration) Granularity(rows []Rate) (Granularity, error) {
	if len(rows) == 0 {
		return "", fmt.Errorf("%w: %s has no rows, so its granularity is unknown", ErrInvalidRate, d.DataID)
	}
	annual := 0
	monthly := 0
	for _, row := range rows {
		if row.MonthSequence == AnnualMonthSequence {
			annual++
			continue
		}
		monthly++
	}
	switch {
	case annual > 0 && monthly > 0:
		return "", fmt.Errorf("%w: %s mixes annual and monthly rows", ErrInvalidRate, d.DataID)
	case annual > 0:
		return GranularityAnnual, nil
	default:
		return GranularityMonthly, nil
	}
}

// ResolveTable reads a rate Data and binds its rows in one call.
// ResolveTable reads a rate Data and binds its rows in one call, and reconciles the
// granularity the rows imply with the one the Data declares.
//
// The declaration is checked here rather than left to the caller because the two
// can disagree in a way neither alone shows: a table declared annual whose rows use
// month sequences 1..12 would be read as monthly by a caller that trusts the rows
// and as annual by a caller that trusts the declaration. A table with no rows has no
// implied granularity, so it keeps the declared one without a comparison.
func ResolveTable(world ir.World, dataID core.CanonicalID, rows []map[core.CanonicalID]string) (Table, error) {
	declaration, err := Resolve(world, dataID)
	if err != nil {
		return Table{}, err
	}
	bound, err := declaration.Bind(rows)
	if err != nil {
		return Table{}, err
	}
	declared, err := ResolveGranularity(world, dataID)
	if err != nil {
		return Table{}, err
	}
	declaredValue, hasDeclared, err := declared.GranularityRows(rows)
	if err != nil {
		return Table{}, err
	}
	if len(bound) == 0 {
		// The granularity value lives in the rows, so an empty table has none: the
		// role says which column would carry it, not what it holds. An empty table
		// is therefore unknown and is refused rather than guessed.
		return Table{}, fmt.Errorf("%w: %s has no rows, so its granularity is unknown", ErrInvalidRate, dataID)
	}
	implied, err := declaration.Granularity(bound)
	if err != nil {
		return Table{}, err
	}
	if hasDeclared {
		if err := CheckDeclared(declaredValue, implied); err != nil {
			return Table{}, err
		}
	}
	return Table{Declaration: declaration, Granularity: implied, Rows: bound}, nil
}

// Pair is a monthly table and an annual table joined by declaration.
//
// The monthly table answers for a month sequence, and the annual table answers for
// a fiscal year. Which one a lookup uses is the declared fallback's business, so a
// caller does not choose a table by which one happened to have a row.
type Pair struct {
	Monthly Table
	Annual  Table
	// Fallback is the declared rule, read from the monthly table's rows when the
	// Data declares the fallback role. It is kept here so a lookup can be asked
	// once and used for many dates.
	Fallback Fallback
	Set      Set
}

// NewPair joins two tables. The monthly table must be monthly and the annual table
// annual; a pair the other way round is refused, because the fallback's meaning is
// "a missing month uses the year".
func NewPair(monthly, annual Table, fallback Fallback, systemCurrency string, kind Kind, direction Direction, rounding Rounding, locale *time.Location) (Pair, error) {
	if monthly.Granularity != GranularityMonthly {
		return Pair{}, fmt.Errorf("%w: %s is not a monthly table", ErrInvalidRate, monthly.Declaration.DataID)
	}
	if annual.Granularity != GranularityAnnual {
		return Pair{}, fmt.Errorf("%w: %s is not an annual table", ErrInvalidRate, annual.Declaration.DataID)
	}
	set := Set{
		SystemCurrency: systemCurrency,
		Kind:           kind,
		Direction:      direction,
		Rounding:       rounding,
		Locale:         locale,
		Fallback:       fallback,
		Rates:          make([]Rate, 0, len(monthly.Rows)+len(annual.Rows)),
	}
	set.Rates = append(set.Rates, monthly.Rows...)
	set.Rates = append(set.Rates, annual.Rows...)
	return Pair{Monthly: monthly, Annual: annual, Fallback: fallback, Set: set}, nil
}

// RateFor returns the rate for a period, using the declared fallback across the
// two tables.
//
// A monthly row wins. A missing monthly row uses the annual row only when the
// fallback is annual; otherwise the miss is reported, so "no row" is not read as a
// rate of zero the way X-TEA's zero comparison did.
func (p Pair) RateFor(currency string, fiscalYear, monthSequence int) (float64, error) {
	if err := p.Set.validateFallback(); err != nil {
		return 0, err
	}
	monthlyErr := error(nil)
	if monthSequence != AnnualMonthSequence {
		rate, err := p.Set.rateInTable(p.Monthly, currency, fiscalYear, monthSequence)
		if err == nil {
			return rate, nil
		}
		monthlyErr = err
	}
	if p.Set.fallback() != FallbackAnnual {
		// Without the declared fallback the miss is the answer: reading the annual
		// row here would be the zero-comparison fallback X-TEA had.
		if monthlyErr != nil {
			return 0, monthlyErr
		}
		return 0, fmt.Errorf("%w: no annual rate for %s in fiscal year %d", ErrInvalidRate, currency, fiscalYear)
	}
	annual, err := p.Set.rateInTable(p.Annual, currency, fiscalYear, AnnualMonthSequence)
	if err != nil {
		if monthlyErr != nil {
			return 0, fmt.Errorf("%w: no monthly or annual rate for %s in fiscal year %d", ErrInvalidRate, currency, fiscalYear)
		}
		return 0, err
	}
	return annual, nil
}

// rateInTable looks one rate up in one table, so a pair lookup does not have to
// filter the joined list and risk taking a row from the other table.
func (s Set) rateInTable(table Table, currency string, fiscalYear, monthSequence int) (float64, error) {
	if currency == s.SystemCurrency {
		return 1, nil
	}
	var found *Rate
	for index := range table.Rows {
		row := table.Rows[index]
		if row.Currency != currency || row.FiscalYear != fiscalYear || row.MonthSequence != monthSequence {
			continue
		}
		if found != nil {
			return 0, fmt.Errorf("%w: %d rows for %s in %s", ErrInvalidRate, 2, currency, table.Declaration.DataID)
		}
		found = &table.Rows[index]
	}
	if found == nil {
		return 0, fmt.Errorf("%w: no rate for %s in fiscal year %d month %d in %s", ErrInvalidRate, currency, fiscalYear, monthSequence, table.Declaration.DataID)
	}
	return s.value(*found)
}
