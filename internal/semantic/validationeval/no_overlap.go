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
// Two rows that certainly state a period, are certainly in the same group
// and certainly share a day settle the rule as not satisfied: nothing unread
// can undo an overlap that was seen. Without such a pair the rule is
// satisfied only when every row was read, every row certainly states a
// period, and no two rows might break the rule once the unread parts are
// known.
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

	// A row states a period when it has a first day and a last day, or no
	// last day and the rule calls that open. silent[i]: row i certainly
	// states none. unsure[i]: whether it states one depends on a date that
	// was not read.
	silent := make([]bool, len(reading.Rows))
	unsure := make([]bool, len(reading.Rows))
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
		switch row.Start.State {
		case ValueNull:
			silent[i] = true
			note(&dateNull, rule.Attribute)
		case ValueNotFetched:
			unsure[i] = true
			note(&dateNotFetched, rule.Attribute)
		}
		if !rule.OpenEnded {
			switch row.End.State {
			case ValueNull:
				silent[i] = true
				note(&dateNull, rule.Compared)
			case ValueNotFetched:
				unsure[i] = true
				note(&dateNotFetched, rule.Compared)
			}
		}
	}

	// startsByEnd: does a start on or before the last day of b? A period
	// without a last day is reached by every first day.
	startsByEnd := func(a, b PeriodRow) Tri {
		switch {
		case b.End.State == ValueNull:
			return TriYes
		case b.End.State == ValueNotFetched:
			note(&dateNotFetched, rule.Compared)
			return TriMaybe
		case a.Start.State == ValueNotFetched:
			note(&dateNotFetched, rule.Attribute)
			return TriMaybe
		case a.Start.Value.Compare(b.End.Value) <= 0:
			return TriYes
		}
		return TriNo
	}
	overlapSeen, dateMaybe, groupNotFetched, groupNull := false, false, false, false
	for i := range reading.Rows {
		for j := i + 1; j < len(reading.Rows); j++ {
			if silent[i] || silent[j] {
				continue
			}
			a, b := reading.Rows[i], reading.Rows[j]
			// Whether the rows are in one group. A row with a value and a
			// row without one are not.
			sameGroup, unstated := TriYes, false
			if rule.GroupBy != "" {
				switch {
				case a.Group.State == ValueNotFetched || b.Group.State == ValueNotFetched:
					sameGroup = TriMaybe
				case a.Group.State == ValueNull && b.Group.State == ValueNull:
					// Whether two rows without a value are one group is not
					// stated by the rule.
					sameGroup, unstated = TriMaybe, true
				case a.Group.State == ValueNull || b.Group.State == ValueNull || a.Group.Key != b.Group.Key:
					sameGroup = TriNo
				}
			}
			if sameGroup == TriNo {
				continue
			}
			var shareADay Tri
			// The first clause decides alone when it is no; its notes about
			// unread dates then do not matter, so they are taken afterwards.
			noted := dateNotFetched
			first := startsByEnd(a, b)
			second := startsByEnd(b, a)
			shareADay = TriAnd(first, second)
			if shareADay != TriMaybe {
				dateNotFetched = noted
			}
			switch {
			case shareADay == TriNo:
			case shareADay == TriYes && sameGroup == TriYes && !unsure[i] && !unsure[j]:
				overlapSeen = true
			case unstated:
				groupNull = true
			case sameGroup == TriMaybe:
				groupNotFetched = true
				dateMaybe = dateMaybe || shareADay == TriMaybe
			default:
				dateMaybe = dateMaybe || shareADay == TriMaybe
			}
		}
	}
	anyUnsure := false
	for _, u := range unsure {
		anyUnsure = anyUnsure || u
	}
	anySilent := false
	for _, s := range silent {
		anySilent = anySilent || s
	}

	switch {
	case overlapSeen:
		return decided(ResultNotSatisfied)
	// What reading more could settle is reported before what it could not.
	case anyUnsure || dateMaybe:
		return undecided(UndecidedValueNotFetched, rule.Relation, dateNotFetched)
	case groupNotFetched:
		return undecided(UndecidedRelationKeyNotFetched, rule.Relation, rule.GroupBy)
	case reading.Coverage == CoveragePartial:
		return undecided(UndecidedRelationPartiallyRead, rule.Relation, rule.Attribute)
	case anySilent:
		return undecided(UndecidedValueNull, rule.Relation, dateNull)
	case groupNull:
		return undecided(UndecidedRelationKeyNull, rule.Relation, rule.GroupBy)
	default:
		return decided(ResultSatisfied)
	}
}
