// Package viewintent is a synthetic, read-only reference for mapping view
// changes to candidate business operation targets. It does not execute writes.
package viewintent

import "sort"

// Ref identifies a semantic entity or relation occurrence. A display row index,
// SQL alias, physical primary key, or human-readable label is not a Ref.
type Ref struct {
	Scope string
	Kind  string
	ID    string
}

func (r Ref) valid() bool { return r.Scope != "" && r.Kind != "" && r.ID != "" }

// Row is an occurrence in a projected view, not necessarily a business row.
// Entity and Edge are alternative write targets; Grain determines which one
// may be used. An aggregate carries no implicit write target.
type Row struct {
	OccurrenceID string
	Grain        string // entity, relation, aggregate
	Entity       Ref
	Edge         Ref
	RelationRole string
	FieldRef     string // explicit semantic field binding, if any
}

type ReadEvidence struct {
	MeaningRevision   string
	ContextID         string
	ContextInstanceID string
	ContextEpoch      string
	Anchor            Ref
	Completeness      string // complete, partial, unfetched, unknown
	Rows              []Row
}

// Change is a recorded editing intention from a specific view/selection epoch.
// Value is intentionally opaque: typed conversion and business validation
// belong to the accepted operation and its owner.
type Change struct {
	MeaningRevision   string
	ActionRevision    string
	ContextID         string
	ContextInstanceID string
	ContextEpoch      string
	Anchor            Ref
	OccurrenceIDs     []string
	Operation         string // update-field or unlink
	FieldRef          string
	Value             string
}

// Action is a synthetic operation contract supplied for a fixed fixture.
// This package does not decide whether an owner accepted it or authorized it.
type Action struct {
	MeaningRevision      string
	Revision             string
	ContextID            string
	Operation            string
	SubjectKind          string
	TargetScope          string // identity scope bound by the synthetic action, not a grant
	RelationRole         string
	FieldRef             string
	RequiresCompleteRead bool
	CapabilityID         string
}

type Intent struct {
	CapabilityID      string
	Operation         string
	Target            Ref
	FieldRef          string
	Value             string
	ContextID         string
	ContextInstanceID string
	ActionRev         string
}

// Decision is an interpretation result, never a runtime authorization,
// write receipt, validation result, or independent effect observation.
type Decision struct {
	Status string // candidate, ambiguous, unsupported, stale, unresolved
	Reason string
	Intent *Intent
}

func reject(status, reason string) Decision { return Decision{Status: status, Reason: reason} }

// Interpret recognizes a narrow set of reversible changes in one Context.
// A projection can contain several occurrences of one entity (for example
// after a JOIN); deduplication is by *typed semantic ref*, never by row index.
func Interpret(read ReadEvidence, change Change, action Action) Decision {
	if read.ContextID == "" || read.ContextInstanceID == "" || change.ContextInstanceID == "" || read.MeaningRevision == "" || read.ContextEpoch == "" || action.Revision == "" || action.MeaningRevision == "" {
		return reject("unresolved", "missing-binding-evidence")
	}
	if change.MeaningRevision != read.MeaningRevision || action.MeaningRevision != read.MeaningRevision || change.ActionRevision != action.Revision {
		return reject("stale", "meaning-or-action-revision-changed")
	}
	if change.ContextID != read.ContextID || action.ContextID != read.ContextID || change.ContextInstanceID != read.ContextInstanceID || change.ContextEpoch != read.ContextEpoch || change.Anchor != read.Anchor {
		return reject("stale", "context-or-selection-changed")
	}
	if !read.Anchor.valid() {
		return reject("unresolved", "anchor-not-typed")
	}
	if change.Operation != action.Operation || action.CapabilityID == "" {
		return reject("unsupported", "operation-not-declared")
	}
	if action.RequiresCompleteRead && read.Completeness != "complete" {
		return reject("unresolved", "collection-not-proven-complete")
	}
	if read.Completeness != "complete" && read.Completeness != "partial" {
		return reject("unresolved", "result-not-qualified")
	}
	if len(change.OccurrenceIDs) == 0 {
		return reject("unresolved", "no-typed-selection")
	}

	lookup := make(map[string]Row, len(read.Rows))
	for _, row := range read.Rows {
		if row.OccurrenceID == "" {
			return reject("unresolved", "missing-occurrence-id")
		}
		if _, duplicate := lookup[row.OccurrenceID]; duplicate {
			return reject("unresolved", "duplicate-occurrence-id")
		}
		lookup[row.OccurrenceID] = row
	}

	byKey := map[Ref]bool{}
	for _, id := range change.OccurrenceIDs {
		row, found := lookup[id]
		if !found {
			return reject("unresolved", "selected-occurrence-not-in-read-result")
		}
		var target Ref
		switch row.Grain {
		case "entity":
			if change.Operation != "update-field" || change.FieldRef == "" || change.FieldRef != action.FieldRef || row.FieldRef != change.FieldRef || (action.RelationRole != "" && row.RelationRole != action.RelationRole) {
				return reject("unsupported", "entity-field-binding-unsupported")
			}
			target = row.Entity
		case "relation":
			if change.Operation != "unlink" || action.RelationRole == "" || row.RelationRole != action.RelationRole {
				return reject("unsupported", "relation-operation-or-role-unsupported")
			}
			target = row.Edge
		case "aggregate":
			return reject("unsupported", "aggregate-has-no-canonical-inverse")
		default:
			return reject("unresolved", "unknown-result-grain")
		}
		if !target.valid() || action.TargetScope == "" || target.Scope != action.TargetScope || target.Kind != action.SubjectKind {
			return reject("unresolved", "semantic-target-not-qualified")
		}
		byKey[target] = true
	}
	if len(byKey) != 1 {
		return reject("ambiguous", "selected-view-has-multiple-semantic-targets")
	}
	var targets []Ref
	for key := range byKey {
		targets = append(targets, key)
	}
	// The sorting is deliberately independent of projected row order. With one
	// target this is redundant, but keeps the intent deterministic if extended.
	sort.Slice(targets, func(i, j int) bool {
		if targets[i].Scope != targets[j].Scope {
			return targets[i].Scope < targets[j].Scope
		}
		if targets[i].Kind != targets[j].Kind {
			return targets[i].Kind < targets[j].Kind
		}
		return targets[i].ID < targets[j].ID
	})
	return Decision{Status: "candidate", Intent: &Intent{
		CapabilityID:      action.CapabilityID,
		Operation:         change.Operation,
		Target:            targets[0],
		FieldRef:          change.FieldRef,
		Value:             change.Value,
		ContextID:         read.ContextID,
		ContextInstanceID: read.ContextInstanceID,
		ActionRev:         action.Revision,
	}}
}
