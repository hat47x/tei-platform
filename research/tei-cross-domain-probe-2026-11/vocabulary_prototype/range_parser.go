package vocabularyprototype

import (
	"errors"
	"strconv"
	"strings"
)

// RangeConstraint represents a minimal range/finiteness prototype for
// VOC-01a. It is a research-only prototype; the product parser/semantic
// model are not changed.
type RangeConstraint struct {
	Raw          string
	Min          *float64
	Max          *float64
	MinInclusive bool
	MaxInclusive bool
}

// ParseRangeConstraint parses a range declaration like "-0.18..0.18"
// or "0.01..0.50" or "-1..1". It returns a constraint or an error.
func ParseRangeConstraint(raw string) (RangeConstraint, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return RangeConstraint{}, errors.New("range is empty")
	}
	minInclusive := !strings.HasPrefix(trimmed, "(")
	maxInclusive := !strings.HasSuffix(trimmed, ")")
	body := strings.Trim(trimmed, "()")
	parts := strings.Split(body, "..")
	if len(parts) != 2 {
		return RangeConstraint{}, errors.New("range must be <min>..<max>")
	}
	min, err := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
	if err != nil {
		return RangeConstraint{}, err
	}
	max, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
	if err != nil {
		return RangeConstraint{}, err
	}
	if min > max {
		return RangeConstraint{}, errors.New("min > max")
	}
	return RangeConstraint{Raw: raw, Min: &min, Max: &max, MinInclusive: minInclusive, MaxInclusive: maxInclusive}, nil
}

// Contains reports whether value is inside the constraint.
func (r RangeConstraint) Contains(value float64) bool {
	if r.Min != nil {
		if r.MinInclusive {
			if value < *r.Min {
				return false
			}
		} else {
			if value <= *r.Min {
				return false
			}
		}
	}
	if r.Max != nil {
		if r.MaxInclusive {
			if value > *r.Max {
				return false
			}
		} else {
			if value >= *r.Max {
				return false
			}
		}
	}
	return true
}
