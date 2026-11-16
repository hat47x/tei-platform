package validationeval

import (
	"fmt"

	"tei/internal/core"
	"tei/internal/semantic"
)

// This file decides "the periods of the rows of a relation do not overlap".
//
// A period is the days from its first day to its last day, both included.
// Two periods overlap when they share a day: each starts on or before the
// last day of the other. A row without a last day is a period that never
// ends only when the rule says so (OpenEnded). With GroupBy, two rows are
// compared only when they hold the same value for it.

// PeriodRow is what a reader knows about one row: the two dates of its
// period and, when the rule groups rows, the value it is grouped by.
type PeriodRow struct {
	Start DateReading
	End   DateReading
	// Group is read only when the rule has a GroupBy attribute.
	Group KeyReading
}

// PeriodRowsReading is the rows of a relation that were read and whether
// those are all the rows.
type PeriodRowsReading struct {
	Coverage Coverage
	Rows     []PeriodRow
}

// StatefulPeriodSource is implemented by a reader that can read the periods
// of the rows of a relation. group is empty when the rule does not group.
type StatefulPeriodSource interface {
	ReadRelationPeriods(relation, start, end, group core.CanonicalID) (PeriodRowsReading, error)
}

func checkDateReading(rule semantic.Validation, attribute core.CanonicalID, reading DateReading) error {
	switch reading.State {
	case ValuePresent:
		if reading.Value.IsZero() {
			return fmt.Errorf("semantic validation %s: the reader reports attribute %s as present without a value", rule.ID, attribute)
		}
	case ValueNull, ValueNotFetched:
		if !reading.Value.IsZero() {
			return fmt.Errorf("semantic validation %s: the reader reports attribute %s as %s together with a value", rule.ID, attribute, reading.State)
		}
	default:
		return fmt.Errorf("semantic validation %s: the reader reports attribute %s in the unknown state %q", rule.ID, attribute, reading.State)
	}
	return nil
}

// decideNoOverlap decides the rule from what was read.
//
// Two rows whose periods are fully known, that are known to be in the same
// group, and that share a day settle the rule as not satisfied: nothing
// unread can undo an overlap that was seen. Without such a pair the rule is
// satisfied only when every row was read, every period is fully known, and
// no two periods that share a day might be in the same group.
func decideNoOverlap(rule semantic.Validation, source StatefulSource) (Decision, error) {
	periods, readsPeriods := source.(StatefulPeriodSource)
	if !readsPeriods {
		return Decision{}, fmt.Errorf("semantic validation %s compares the periods of rows, which this reader cannot read", rule.ID)
	}
	if rule.Compared == "" {
		return Decision{}, fmt.Errorf("semantic validation %s names no attribute that ends the period", rule.ID)
	}
	reading, err := periods.ReadRelationPeriods(rule.Relation, rule.Attribute, rule.Compared, rule.GroupBy)
	if err != nil {
		return Decision{}, fmt.Errorf("semantic validation %s cannot evaluate relation %s: %w", rule.ID, rule.Relation, err)
	}
	if reading.Coverage != CoverageComplete && reading.Coverage != CoveragePartial {
		return Decision{}, fmt.Errorf("semantic validation %s: the reader reports relation %s with the unknown coverage %q", rule.ID, rule.Relation, reading.Coverage)
	}

	// known[i] is true when the period of row i is fully known: it has a
	// first day, and a last day or, by the rule's own words, none.
	known := make([]bool, len(reading.Rows))
	dateNotFetched, dateNull := core.CanonicalID(""), core.CanonicalID("")
	note := func(target *core.CanonicalID, attribute core.CanonicalID) {
		if *target == "" {
			*target = attribute
		}
	}
	for i, row := range reading.Rows {
		if err := checkDateReading(rule, rule.Attribute, row.Start); err != nil {
			return Decision{}, err
		}
		if err := checkDateReading(rule, rule.Compared, row.End); err != nil {
			return Decision{}, err
		}
		if rule.GroupBy != "" {
			switch row.Group.State {
			case ValuePresent:
			case ValueNull, ValueNotFetched:
				if row.Group.Key != "" {
					return Decision{}, fmt.Errorf("semantic validation %s: the reader reports a value of %s as %s together with a value", rule.ID, rule.GroupBy, row.Group.State)
				}
			default:
				return Decision{}, fmt.Errorf("semantic validation %s: the reader reports a value of %s in the unknown state %q", rule.ID, rule.GroupBy, row.Group.State)
			}
		}
		known[i] = true
		switch row.Start.State {
		case ValueNotFetched:
			known[i] = false
			note(&dateNotFetched, rule.Attribute)
		case ValueNull:
			known[i] = false
			note(&dateNull, rule.Attribute)
		}
		switch {
		case row.End.State == ValueNotFetched:
			known[i] = false
			note(&dateNotFetched, rule.Compared)
		case row.End.State == ValueNull && !rule.OpenEnded:
			known[i] = false
			note(&dateNull, rule.Compared)
		}
	}

	// startsByEnd reports whether a starts on or before the last day of b.
	startsByEnd := func(a, b PeriodRow) bool {
		return b.End.State == ValueNull || a.Start.Value.Compare(b.End.Value) <= 0
	}
	overlapSeen, groupNotFetched, groupNull := false, false, false
	for i := range reading.Rows {
		for j := i + 1; j < len(reading.Rows); j++ {
			if !known[i] || !known[j] {
				continue
			}
			a, b := reading.Rows[i], reading.Rows[j]
			if !startsByEnd(a, b) || !startsByEnd(b, a) {
				continue
			}
			// The two periods share a day. Whether that breaks the rule
			// depends on whether the rows are in the same group.
			switch {
			case rule.GroupBy == "":
				overlapSeen = true
			case a.Group.State == ValuePresent && b.Group.State == ValuePresent:
				overlapSeen = overlapSeen || a.Group.Key == b.Group.Key
			case a.Group.State == ValueNotFetched || b.Group.State == ValueNotFetched:
				groupNotFetched = true
			case a.Group.State == ValueNull && b.Group.State == ValueNull:
				// Whether two rows without a value are in one group is not
				// stated by the rule.
				groupNull = true
			}
			// A row with a value and a row without one are not in one group.
		}
	}

	switch {
	case overlapSeen:
		return decided(ResultNotSatisfied)
	// What reading more could settle is reported before what it could not.
	case dateNotFetched != "":
		return undecided(UndecidedValueNotFetched, rule.Relation, dateNotFetched)
	case groupNotFetched:
		return undecided(UndecidedRelationKeyNotFetched, rule.Relation, rule.GroupBy)
	case reading.Coverage == CoveragePartial:
		return undecided(UndecidedRelationPartiallyRead, rule.Relation, rule.Attribute)
	case dateNull != "":
		return undecided(UndecidedValueNull, rule.Relation, dateNull)
	case groupNull:
		return undecided(UndecidedRelationKeyNull, rule.Relation, rule.GroupBy)
	default:
		return decided(ResultSatisfied)
	}
}
