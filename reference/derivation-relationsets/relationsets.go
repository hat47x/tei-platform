// Package derivationrelationsets is a D9 read-only reference boundary.
// It fixes the meaning of a Relation set bound as a derivation input: shape,
// completeness, business identity, fan-out and aggregation policy.
//
// It does not implement rule definitions (D1 #1637), SELECT planning or
// Filter/Sort/Aggregate/Limit order (D2 #1640), disclosure (D5 #1693), the
// Semantic Relation / CandidateKey authority (#178/#1126) or any query
// planner, ORM or second key master.
package derivationrelationsets

import (
	"errors"
	"fmt"
	"math/big"
	"strings"

	"tei/internal/semantic/derivationref"
)

var (
	ErrInvalidSet  = errors.New("derivation relation set is invalid")
	ErrShape       = errors.New("derivation relation shape mismatch")
	ErrCardinality = errors.New("derivation relation cardinality violated")
	ErrIncomplete  = errors.New("derivation relation set is incomplete")
	ErrFanOut      = errors.New("derivation relation join would fan out")
	ErrNoPolicy    = errors.New("derivation aggregation policy is undeclared")
)

// Shape is the input form a Relation reference is bound as. A single-row
// attribute, an optional reference, a 1:N collection and an already
// aggregated result are different inputs, not the same column.
type Shape string

const (
	Scalar      Shape = "scalar"       // exactly one row attribute
	OptionalOne Shape = "optional-one" // 0..1 reference
	Many        Shape = "many"         // 1:N collection
	Aggregate   Shape = "aggregate"    // an already reduced result
)

// Completeness records whether the fetched rows are the whole set for the
// anchor. A display page or an aborted read is not the complete collection.
type Completeness string

const (
	Complete  Completeness = "complete"
	Partial   Completeness = "partial"   // page / top-N / truncated read
	Unfetched Completeness = "unfetched" // never read
)

// Row is one row occurrence. Identity is the existing business identity
// (CandidateKey or explicit collection identity) and may be empty only for
// Scalar/OptionalOne; aggregation requires it so that a duplicate is
// detectable instead of silently deduplicated.
type Row struct {
	Identity string
	Amount   derivationref.Input
}

// SetRef is one bound Relation input for one anchor entity instance. Anchor,
// RelationID and Key are explicit; the reference never infers meaning from
// a column name or a result row order.
type SetRef struct {
	Anchor       string // parent entity instance the set belongs to
	RelationID   string // existing Semantic Relation identity (#178/#1126)
	Key          []string
	Shape        Shape
	Completeness Completeness
	Rows         []Row
}

// EmptySetPolicy must be declared by the rule; there is no global default that
// turns an empty SUM into 0 or NULL.
type EmptySetPolicy string

const (
	EmptySetUndeclared EmptySetPolicy = ""
	EmptySetNull       EmptySetPolicy = "null"
	EmptySetZero       EmptySetPolicy = "zero"
)

// NullPolicy declares how a NULL element is treated. It is required only when
// a NULL is actually present.
type NullPolicy string

const (
	NullsUndeclared NullPolicy = ""
	NullsPropagate  NullPolicy = "propagate"
	NullsSkip       NullPolicy = "skip"
)

// AggregatePolicy is the declared reduction policy for one aggregation.
type AggregatePolicy struct {
	EmptySet EmptySetPolicy
	Nulls    NullPolicy
}

// Bind validates the shape and completeness of one bound set without reading
// any data. A Many set must carry business identities, and an explicit Key is
// mandatory: no silent key inference.
func Bind(ref SetRef) error {
	if ref.Anchor == "" || ref.Anchor != strings.TrimSpace(ref.Anchor) ||
		ref.RelationID == "" || ref.RelationID != strings.TrimSpace(ref.RelationID) ||
		len(ref.Key) == 0 {
		return fmt.Errorf("%w: anchor, relation or key", ErrInvalidSet)
	}
	for _, key := range ref.Key {
		if key == "" || key != strings.TrimSpace(key) {
			return fmt.Errorf("%w: key", ErrInvalidSet)
		}
	}
	switch ref.Completeness {
	case Complete, Partial, Unfetched:
	default:
		return fmt.Errorf("%w: completeness", ErrInvalidSet)
	}
	switch ref.Shape {
	case Scalar:
		if len(ref.Rows) != 1 {
			return fmt.Errorf("%w: scalar needs exactly one row", ErrCardinality)
		}
	case OptionalOne:
		if len(ref.Rows) > 1 {
			return fmt.Errorf("%w: optional-one matched %d rows", ErrCardinality, len(ref.Rows))
		}
	case Many, Aggregate:
		for _, row := range ref.Rows {
			if row.Identity == "" || row.Identity != strings.TrimSpace(row.Identity) {
				return fmt.Errorf("%w: collection row without business identity", ErrInvalidSet)
			}
		}
	default:
		return fmt.Errorf("%w: shape", ErrShape)
	}
	return nil
}

// AsScalar binds a set to a scalar argument. A Many or Aggregate set is
// rejected; there is no implicit first-row pick or implicit reduction.
func AsScalar(ref SetRef) (derivationref.Input, error) {
	if err := Bind(ref); err != nil {
		return derivationref.Input{}, err
	}
	switch ref.Shape {
	case Scalar:
		return ref.Rows[0].Amount, nil
	case OptionalOne:
		if len(ref.Rows) == 0 {
			return derivationref.Input{State: derivationref.Null}, nil
		}
		return ref.Rows[0].Amount, nil
	default:
		return derivationref.Input{}, fmt.Errorf("%w: many relation bound to a scalar argument", ErrShape)
	}
}

// Sum reduces a Many/Aggregate set to one value. An incomplete or unfetched
// set is rejected rather than summed as if it were the whole collection, and a
// duplicate business identity is treated as a join fan-out, not a genuine
// second row.
func Sum(ref SetRef, policy AggregatePolicy) (derivationref.Input, error) {
	if err := Bind(ref); err != nil {
		return derivationref.Input{}, err
	}
	if ref.Shape != Many && ref.Shape != Aggregate {
		return derivationref.Input{}, fmt.Errorf("%w: sum needs a collection", ErrShape)
	}
	if ref.Completeness != Complete {
		return derivationref.Input{}, fmt.Errorf("%w: %s is %s", ErrIncomplete, ref.RelationID, ref.Completeness)
	}
	seen := make(map[string]bool, len(ref.Rows))
	total := new(big.Rat)
	nulls, present := 0, 0
	for _, row := range ref.Rows {
		if seen[row.Identity] {
			return derivationref.Input{}, fmt.Errorf("%w: duplicate business identity %s", ErrFanOut, row.Identity)
		}
		seen[row.Identity] = true
		switch row.Amount.State {
		case derivationref.Present:
			value, ok := new(big.Rat).SetString(row.Amount.Decimal)
			if !ok {
				return derivationref.Input{}, fmt.Errorf("%w: value is not a decimal", ErrInvalidSet)
			}
			total.Add(total, value)
			present++
		case derivationref.Null:
			nulls++
		case derivationref.Missing, derivationref.AbsentRelation, derivationref.MultipleRelations:
			return derivationref.Input{}, fmt.Errorf("%w: element %s is %s", ErrIncomplete, row.Identity, row.Amount.State)
		default:
			return derivationref.Input{}, fmt.Errorf("%w: element %s has state %q", ErrInvalidSet, row.Identity, row.Amount.State)
		}
	}
	if nulls > 0 {
		switch policy.Nulls {
		case NullsUndeclared:
			return derivationref.Input{}, fmt.Errorf("%w: NULL elements present", ErrNoPolicy)
		case NullsPropagate:
			return derivationref.Input{State: derivationref.Null}, nil
		case NullsSkip:
			// NULLs are excluded; an all-NULL set is still empty.
		default:
			return derivationref.Input{}, fmt.Errorf("%w: null policy", ErrNoPolicy)
		}
	}
	if present == 0 {
		switch policy.EmptySet {
		case EmptySetUndeclared:
			return derivationref.Input{}, fmt.Errorf("%w: empty set", ErrNoPolicy)
		case EmptySetZero:
			return derivationref.Input{State: derivationref.Present, Decimal: "0"}, nil
		case EmptySetNull:
			return derivationref.Input{State: derivationref.Null}, nil
		default:
			return derivationref.Input{}, fmt.Errorf("%w: empty set policy", ErrNoPolicy)
		}
	}
	return derivationref.Input{State: derivationref.Present, Decimal: total.FloatString(resultScale(ref.Rows))}, nil
}

// FanOut is the diagnostic of a naive column-equality JOIN between two
// independent collections. NaiveRows is what a simple cross product would
// count; DistinctRows is the number of real rows in one collection.
type FanOut struct {
	RelationA    string
	RelationB    string
	NaiveRows    int
	DistinctRows int
}

func (f FanOut) Error() string {
	return fmt.Sprintf("join fan-out: %s x %s would count %d rows instead of %d",
		f.RelationA, f.RelationB, f.NaiveRows, f.DistinctRows)
}

// JoinFanOut guards a scalar aggregate that would otherwise receive two
// independent collections. When the right collection has more than one row,
// joining it multiplies each left row, so it returns ErrFanOut together with
// the naive and distinct counts. It never returns a fanned-out set for Sum.
func JoinFanOut(left, right SetRef) (FanOut, error) {
	if err := Bind(left); err != nil {
		return FanOut{}, err
	}
	if err := Bind(right); err != nil {
		return FanOut{}, err
	}
	if left.Shape != Many || right.Shape != Many {
		return FanOut{}, fmt.Errorf("%w: join needs two collections", ErrShape)
	}
	if left.Completeness != Complete || right.Completeness != Complete {
		return FanOut{}, ErrIncomplete
	}
	diagnostic := FanOut{
		RelationA:    left.RelationID,
		RelationB:    right.RelationID,
		NaiveRows:    len(left.Rows) * len(right.Rows),
		DistinctRows: len(left.Rows),
	}
	if len(left.Rows) > 0 && len(right.Rows) > 1 {
		return diagnostic, fmt.Errorf("%w: %v", ErrFanOut, diagnostic)
	}
	return diagnostic, nil
}

func resultScale(rows []Row) int {
	scale := 0
	for _, row := range rows {
		if row.Amount.State != derivationref.Present {
			continue
		}
		if dot := strings.IndexByte(row.Amount.Decimal, '.'); dot >= 0 {
			if s := len(row.Amount.Decimal) - dot - 1; s > scale {
				scale = s
			}
		}
	}
	return scale
}
