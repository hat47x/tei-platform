package main

import "fmt"

// This file adds the opt-in flag SB-C0 §4.25 left unnamed (P-4).
//
// §4.25 measured three ways to strengthen a default attempt and found they pull
// against each other, so it proposed one flag rather than three defaults. This is
// that flag, named for what it buys rather than for the mechanism: -strong asks
// for an attempt that can support the L2-b claim.
//
// It is deliberately thin, and its thinness is the design. It does not change any
// default: an attempt that passes nothing still derives no plan and makes no
// claim. It does not invent inputs. It turns on reproduction, and it refuses when
// the route that could make the claim is absent, because a flag named "strong"
// that silently returns a weak attempt is the exact failure §4.23's unmet list
// exists to prevent.

const strongFlagBoundary = "-strong turns on the reproduction check and requires the in-process route; it changes nothing else, grants no authority, and does not make the claim hold by itself"

// selfBuildStrong records what the flag changed, so a reader can tell why the
// attempt ran a second build.
type selfBuildStrong struct {
	// Requested is true when the caller passed -strong.
	Requested bool `json:"requested"`
	// Implied lists the observations this flag turned on that the caller did not
	// ask for explicitly. Naming them keeps the record honest about why they ran.
	Implied []string `json:"implied,omitempty"`
	// Route names the derivation route the flag requires.
	Route    string `json:"routeRequired,omitempty"`
	Boundary string `json:"boundary"`
}

const strongImpliedRebuildCheck = "rebuild-check"

// applyStrong turns -strong into the observations it implies.
//
// It is a pure function of the flags so the implication is testable without
// running an attempt.
func applyStrong(veryStrong bool, rebuildCheck bool, inProcessRequested bool) (bool, selfBuildStrong, error) {
	record := selfBuildStrong{Requested: veryStrong, Boundary: strongFlagBoundary}
	if !veryStrong {
		return rebuildCheck, record, nil
	}
	record.Route = derivedByEntryInProcess

	// Without the in-process route the claim cannot be made: the entry did not
	// interpret the declaration. Refusing is better than returning a strong name
	// over a weak attempt.
	if !inProcessRequested {
		return rebuildCheck, record, fmt.Errorf(
			"-strong needs the in-process route: pass -build-catalog, -build-catalog-digest and -build-source-root, " +
				"because a plan derived by a resolver child cannot support the claim this flag asks for",
		)
	}

	if !rebuildCheck {
		record.Implied = append(record.Implied, strongImpliedRebuildCheck)
		rebuildCheck = true
	}
	return rebuildCheck, record, nil
}
