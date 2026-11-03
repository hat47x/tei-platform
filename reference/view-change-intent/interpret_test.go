package viewintent

import "testing"

func fixture() (ReadEvidence, Change, Action) {
	anchor := Ref{Scope: "tenant-A", Kind: "order", ID: "H1"}
	read := ReadEvidence{
		MeaningRevision: "model-A", ContextID: "order-lines", ContextInstanceID: "form-A", ContextEpoch: "s1", Anchor: anchor, Completeness: "complete",
		Rows: []Row{
			{OccurrenceID: "join-L1-D1", Grain: "entity", Entity: Ref{Scope: "tenant-A", Kind: "line", ID: "L1"}, FieldRef: "line.amount"},
			{OccurrenceID: "join-L1-D2", Grain: "entity", Entity: Ref{Scope: "tenant-A", Kind: "line", ID: "L1"}, FieldRef: "line.amount"},
			{OccurrenceID: "join-L1-D3", Grain: "entity", Entity: Ref{Scope: "tenant-A", Kind: "line", ID: "L1"}, FieldRef: "line.amount"},
			{OccurrenceID: "line-L2", Grain: "entity", Entity: Ref{Scope: "tenant-A", Kind: "line", ID: "L2"}, FieldRef: "line.amount"},
		},
	}
	change := Change{MeaningRevision: "model-A", ActionRevision: "action-v1", ContextID: "order-lines", ContextInstanceID: "form-A", ContextEpoch: "s1", Anchor: anchor,
		OccurrenceIDs: []string{"join-L1-D1", "join-L1-D2", "join-L1-D3"}, Operation: "update-field", FieldRef: "line.amount", Value: "500"}
	action := Action{MeaningRevision: "model-A", Revision: "action-v1", ContextID: "order-lines", Operation: "update-field", SubjectKind: "line", TargetScope: "tenant-A", FieldRef: "line.amount", CapabilityID: "line.edit"}
	return read, change, action
}

func TestFanoutMapsToOneTypedSubject(t *testing.T) {
	read, change, action := fixture()
	got := Interpret(read, change, action)
	if got.Status != "candidate" || got.Intent == nil || got.Intent.Target.ID != "L1" {
		t.Fatalf("unexpected: %#v", got)
	}
	change.OccurrenceIDs = []string{"join-L1-D3", "join-L1-D1", "join-L1-D2"}
	got2 := Interpret(read, change, action)
	if got2 != got { // Go structs with pointer fields may not compare equal by address.
		if got2.Status != got.Status || got2.Intent == nil || *got2.Intent != *got.Intent {
			t.Fatalf("order changed semantic intent: %#v %#v", got, got2)
		}
	}
}

func TestMultipleTargetsStayAmbiguous(t *testing.T) {
	read, change, action := fixture()
	change.OccurrenceIDs = append(change.OccurrenceIDs, "line-L2")
	if got := Interpret(read, change, action); got.Status != "ambiguous" {
		t.Fatalf("unexpected: %#v", got)
	}
}

func TestRevisedMeaningAndActionCannotReuseOldBinding(t *testing.T) {
	read, change, action := fixture()
	read.MeaningRevision = "model-B"
	if got := Interpret(read, change, action); got.Status != "stale" {
		t.Fatalf("model mismatch: %#v", got)
	}
	read, change, action = fixture()
	action.Revision = "action-v2"
	if got := Interpret(read, change, action); got.Status != "stale" {
		t.Fatalf("action mismatch: %#v", got)
	}
}

func TestSelectionEpochAndScopeCannotBeReused(t *testing.T) {
	read, change, action := fixture()
	read.ContextEpoch = "s2"
	if got := Interpret(read, change, action); got.Status != "stale" {
		t.Fatalf("epoch: %#v", got)
	}
	read, change, action = fixture()
	change.Anchor.Scope = "tenant-B"
	if got := Interpret(read, change, action); got.Status != "stale" {
		t.Fatalf("scope: %#v", got)
	}
	read, change, action = fixture()
	change.OccurrenceIDs = []string{"row-index-0"}
	if got := Interpret(read, change, action); got.Status != "unresolved" {
		t.Fatalf("row index: %#v", got)
	}
}

func TestAggregateCannotChooseInverse(t *testing.T) {
	read, change, action := fixture()
	read.Rows = append(read.Rows, Row{OccurrenceID: "sum-order", Grain: "aggregate"})
	change.OccurrenceIDs = []string{"sum-order"}
	if got := Interpret(read, change, action); got.Status != "unsupported" || got.Reason != "aggregate-has-no-canonical-inverse" {
		t.Fatalf("aggregate: %#v", got)
	}
}

func TestRelationUnlinkTargetsEdgeNotNode(t *testing.T) {
	read, change, action := fixture()
	read.Rows = []Row{{OccurrenceID: "relation-1", Grain: "relation", Entity: Ref{Scope: "tenant-A", Kind: "person", ID: "P1"},
		Edge: Ref{Scope: "tenant-A", Kind: "membership-edge", ID: "E1"}, RelationRole: "team-member"}}
	change.OccurrenceIDs = []string{"relation-1"}
	change.Operation = "unlink"
	change.FieldRef = ""
	action.Operation = "unlink"
	action.SubjectKind = "membership-edge"
	action.RelationRole = "team-member"
	action.FieldRef = ""
	action.CapabilityID = "organization.unlink-member"
	got := Interpret(read, change, action)
	if got.Status != "candidate" || got.Intent == nil || got.Intent.Target.Kind != "membership-edge" || got.Intent.Target.ID != "E1" {
		t.Fatalf("edge: %#v", got)
	}
	read.Rows[0].Edge = Ref{}
	if got := Interpret(read, change, action); got.Status != "unresolved" {
		t.Fatalf("missing edge key: %#v", got)
	}
}

func TestIncompleteCollectionRequiresExplicitActionCondition(t *testing.T) {
	read, change, action := fixture()
	read.Completeness = "partial"
	action.RequiresCompleteRead = true
	if got := Interpret(read, change, action); got.Status != "unresolved" || got.Reason != "collection-not-proven-complete" {
		t.Fatalf("partial bulk: %#v", got)
	}
	// One explicitly keyed row can be a candidate with a partial read; the
	// execution-time owner must still check current authority and concurrency.
	action.RequiresCompleteRead = false
	if got := Interpret(read, change, action); got.Status != "candidate" {
		t.Fatalf("partial record: %#v", got)
	}
}

func TestWrongFieldRoleAndOperationAreRejected(t *testing.T) {
	read, change, action := fixture()
	change.FieldRef = "delivery.amount"
	if got := Interpret(read, change, action); got.Status != "unsupported" {
		t.Fatalf("field alias: %#v", got)
	}
	read, change, action = fixture()
	change.Operation = "delete-entity"
	if got := Interpret(read, change, action); got.Status != "unsupported" {
		t.Fatalf("operation: %#v", got)
	}
}

func TestDuplicateOccurrenceIDCannotSilentlySelectFirst(t *testing.T) {
	read, change, action := fixture()
	read.Rows = append(read.Rows, read.Rows[0])
	if got := Interpret(read, change, action); got.Status != "unresolved" {
		t.Fatalf("duplicate: %#v", got)
	}
}

func TestSameKindButDifferentTargetScopeCannotBeWritten(t *testing.T) {
	read, change, action := fixture()
	read.Rows[0].Entity.Scope = "tenant-B"
	change.OccurrenceIDs = []string{"join-L1-D1"}
	if got := Interpret(read, change, action); got.Status != "unresolved" {
		t.Fatalf("scope: %#v", got)
	}
}

func TestTwoViewInstancesCannotShareOneSelection(t *testing.T) {
	read, change, action := fixture()
	read.ContextInstanceID = "form-B" // same definition, same epoch, different UI window
	if got := Interpret(read, change, action); got.Status != "stale" {
		t.Fatalf("window: %#v", got)
	}
}

func TestUnknownCompletenessIsNotAQualifiedRead(t *testing.T) {
	read, change, action := fixture()
	read.Completeness = "arbitrary-new-status"
	if got := Interpret(read, change, action); got.Status != "unresolved" {
		t.Fatalf("unknown completeness: %#v", got)
	}
}
