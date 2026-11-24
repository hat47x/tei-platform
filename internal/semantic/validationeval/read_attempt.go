package validationeval

import (
	"fmt"

	"tei/internal/core"
	"tei/internal/semantic"
)

// This file carries the read-attempt identity of the values one multi-input
// operator decision uses.
//
// A rule such as `quantity is within low: ... high: ...` reads more than one
// attribute. Having three present values is not enough to decide it: the
// decision document (section 14.2 B-10) requires the subject and every input
// read from an attribute to belong to the same Read Attempt or Snapshot, or to
// have had their consistency established explicitly. A constant input needs no
// snapshot; an attribute reference does.
//
// The reader states which attempt it read. Two reads that name different
// attempts are refused rather than decided: assuming a common snapshot is
// exactly what B-10 forbids. A reader that states no attempt cannot qualify an
// attribute input either, so such a read stays undecided instead of being
// treated as consistent.

// AttributedSource is a reader that can say which Read Attempt each attribute
// value came from. The identity is opaque to this package: it is compared for
// equality only, never parsed, ordered, or used as a business key.
type AttributedSource interface {
	ReadAttempt(attribute core.CanonicalID) (attempt string, stated bool)
}

// inputOrigin records where one operator input's value came from.
type inputOrigin struct {
	input    string
	attempt  string
	fromData bool
	stated   bool
}

// requireSameAttempt refuses a decision whose attribute inputs came from
// different Read Attempts, or from a source that can state none.
//
// The rule applies only when more than one input was read from data. A single
// attribute read needs no cross-read consistency: the subject is one value from
// one read, and constants are the same whatever was read (section 14.2 states
// this explicitly). The returned Undecided is what a read that cannot establish
// consistency must report; the error is for a stated mismatch, which is a wrong
// read rather than an unknown one.
func requireSameAttempt(rule semantic.Validation, origins []inputOrigin) (*Undecided, error) {
	fromData := 0
	for _, origin := range origins {
		if origin.fromData {
			fromData++
		}
	}
	if fromData < 2 {
		return nil, nil
	}
	var first *inputOrigin
	for i := range origins {
		origin := &origins[i]
		if !origin.fromData {
			// A constant input needs no snapshot: it is the same value
			// whatever was read. Section 14.2 states this explicitly.
			continue
		}
		if !origin.stated {
			// The reader cannot say which attempt this value belongs to, so
			// consistency with the subject cannot be established. It is not
			// assumed.
			return &Undecided{Reason: UndecidedReadAttemptUnknown, Attribute: core.CanonicalID(origin.input)}, nil
		}
		if first == nil {
			first = origin
			continue
		}
		if origin.attempt != first.attempt {
			return nil, fmt.Errorf(
				"semantic validation %s reads operator %s input %q from read attempt %q but input %q from %q; one decision cannot mix read attempts",
				rule.ID, rule.Operator, origin.input, origin.attempt, first.input, first.attempt,
			)
		}
	}
	return nil, nil
}

// constantOrigin is the origin of an input the rule assigned a literal.
func constantOrigin(name string) inputOrigin {
	return inputOrigin{input: name}
}

// dataOrigin is the origin of an input read from an attribute of the subject's
// row, in the attempt the reader states.
func dataOrigin(name, attribute string, source any) inputOrigin {
	reader, ok := source.(AttributedSource)
	if !ok {
		return inputOrigin{input: name, fromData: true}
	}
	attempt, stated := reader.ReadAttempt(core.CanonicalID(attribute))
	return inputOrigin{input: name, fromData: true, attempt: attempt, stated: stated}
}
