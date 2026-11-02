package studioplugindatalifecycle

import (
	"reflect"
	"testing"
)

func dataFixture() DataState {
	return DataState{
		PluginID: "p",
		Areas: []Area{
			{Key: "a", Mapping: MappingConfirmed, Owners: []string{"p"}},
			{Key: "b", Mapping: MappingConfirmed, Owners: []string{"p"}, ReferencedBy: []string{"q"}},
			{Key: "c", Mapping: MappingShared, Owners: []string{"p", "r"}},
			{Key: "d", Mapping: MappingUnknown},
		},
		Files: []SourceFile{
			{Path: "src/order.tei", Segments: []Segment{
				{Key: "s1", Area: "a", Kind: SegmentOwned, Bytes: "A1"},
				{Key: "s2", Kind: SegmentUnknownAttribute, Bytes: `<x-data q="z"/>`},
				{Key: "s3", Area: "a", Kind: SegmentOwned, Bytes: "A2"},
				{Key: "s4", Kind: SegmentExtension, Bytes: "<ext:note/>"},
				{Key: "s5", Area: "b", Kind: SegmentOwned, Bytes: "B1"},
			}},
			{Path: "src/shared.tei", Segments: []Segment{
				{Key: "s6", Area: "c", Kind: SegmentShared, Bytes: "C1"},
				{Key: "s7", Kind: SegmentUnknownElement, Bytes: "<mystery/>"},
			}},
		},
	}
}

func TestPluginRemovalKeepsAllData(t *testing.T) {
	state := dataFixture()
	plan, err := PlanDeletion(state, DeleteRequest{Action: ActionRemovePlugin})
	if err != nil {
		t.Fatal(err)
	}
	if !plan.OperationAllowed || plan.DataDeleted {
		t.Fatalf("plan=%+v", plan)
	}
	if !reflect.DeepEqual(plan.FilesAfter, state.Files) {
		t.Fatal("plugin removal changed the source bytes")
	}
	if len(plan.Retained) != 4 {
		t.Fatalf("retained=%v", plan.Retained)
	}
	for _, retained := range plan.Retained {
		if retained.Reason != "plugin-removal-keeps-data" {
			t.Errorf("retained %q reason=%q", retained.Key, retained.Reason)
		}
	}
}

func TestDeleteConfirmedAreaKeepsUnknownAndExtensionBytes(t *testing.T) {
	state := dataFixture()
	plan, err := PlanDeletion(state, DeleteRequest{Action: ActionDeleteData, SelectedAreas: []string{"a"}})
	if err != nil {
		t.Fatal(err)
	}
	if !plan.OperationAllowed || !plan.DataDeleted {
		t.Fatalf("plan=%+v", plan)
	}
	if plan.Diff.Areas != 1 || plan.Diff.Files != 1 || plan.Diff.RemovedBytes != 4 {
		t.Fatalf("diff=%+v", plan.Diff)
	}
	if len(plan.FilesAfter) != 2 {
		t.Fatalf("files after=%d", len(plan.FilesAfter))
	}
	order := plan.FilesAfter[0]
	for _, segment := range order.Segments {
		if segment.Area == "a" {
			t.Fatalf("selected area segment survived: %+v", segment)
		}
	}
	if !segmentPresent(order.Segments, "s2") || !segmentPresent(order.Segments, "s4") || !segmentPresent(order.Segments, "s5") {
		t.Fatalf("out-of-scope bytes were dropped: %+v", order.Segments)
	}
	if !plan.ProtectedPreserved || plan.SourceUnchanged {
		t.Fatalf("plan flags=%v/%v", plan.ProtectedPreserved, plan.SourceUnchanged)
	}
	if !reflect.DeepEqual(state, dataFixture()) {
		t.Fatal("PlanDeletion mutated the input")
	}
}

func TestSharedAreaNeedsExplicitSelectionAndConfirmation(t *testing.T) {
	state := dataFixture()
	without := DeleteRequest{Action: ActionDeleteData, SelectedAreas: []string{"c"}}
	plan, err := PlanDeletion(state, without)
	if err != nil {
		t.Fatal(err)
	}
	if plan.DataDeleted {
		t.Fatal("a shared area was deleted without confirmation")
	}
	if !hasRetained(plan.Retained, "c", "shared-requires-explicit-selection") {
		t.Fatalf("retained=%v", plan.Retained)
	}
	with := without
	with.ConfirmShared = true
	plan, err = PlanDeletion(state, with)
	if err != nil {
		t.Fatal(err)
	}
	if !plan.DataDeleted {
		t.Fatalf("plan=%+v", plan)
	}
	if plan.Diff.Areas != 1 {
		t.Fatalf("diff=%+v", plan.Diff)
	}
}

func TestReferencedAreaNeedsConfirmation(t *testing.T) {
	state := dataFixture()
	req := DeleteRequest{Action: ActionDeleteData, SelectedAreas: []string{"b"}}
	plan, err := PlanDeletion(state, req)
	if err != nil {
		t.Fatal(err)
	}
	if plan.DataDeleted || !hasRetained(plan.Retained, "b", "referenced-by-others:q") {
		t.Fatalf("plan=%+v", plan)
	}
	req.ConfirmReferenced = true
	plan, err = PlanDeletion(state, req)
	if err != nil {
		t.Fatal(err)
	}
	if !plan.DataDeleted {
		t.Fatalf("plan=%+v", plan)
	}
	if !contains(plan.ReferenceImpact, "area:b:referenced-by:q") {
		t.Fatalf("reference impact=%v", plan.ReferenceImpact)
	}
}

func TestUnknownAreaIsNeverDeletedByGuess(t *testing.T) {
	state := dataFixture()
	plan, err := PlanDeletion(state, DeleteRequest{Action: ActionDeleteData, SelectedAreas: []string{"d"}})
	if err != nil {
		t.Fatal(err)
	}
	if plan.DataDeleted || !hasRetained(plan.Retained, "d", "unknown-provider") {
		t.Fatalf("plan=%+v", plan)
	}
	if !contains(plan.Reasons, "unknown-area-not-deletable:d") {
		t.Fatalf("reasons=%v", plan.Reasons)
	}

	all := DeleteRequest{Action: ActionDeleteData, SelectedAreas: []string{"a"}, DeleteAllUnconfirmed: true}
	plan, err = PlanDeletion(state, all)
	if err != nil {
		t.Fatal(err)
	}
	if plan.OperationAllowed || plan.DataDeleted {
		t.Fatalf("delete-all-unconfirmed was allowed: %+v", plan)
	}
	if !contains(plan.Reasons, "unknown-area-not-guessable") {
		t.Fatalf("reasons=%v", plan.Reasons)
	}
	if plan.FilesAfter != nil {
		t.Fatal("a refused deletion produced files")
	}
}

func TestPreservationConflictDoesNotPartiallySave(t *testing.T) {
	state := dataFixture()
	state.Files[0].Segments = append(state.Files[0].Segments,
		Segment{Key: "s8", Area: "a", Kind: SegmentExtension, Bytes: "<ext:inside/>"})
	plan, err := PlanDeletion(state, DeleteRequest{Action: ActionDeleteData, SelectedAreas: []string{"a"}})
	if err != nil {
		t.Fatal(err)
	}
	if plan.OperationAllowed || plan.DataDeleted {
		t.Fatalf("a preservation conflict was allowed: %+v", plan)
	}
	if plan.FilesAfter != nil {
		t.Fatal("a partial save was produced")
	}
	if !plan.SourceUnchanged {
		t.Fatal("the source was not reported as unchanged")
	}
	if !contains(plan.Reasons, "preservation-conflict:s8") {
		t.Fatalf("reasons=%v", plan.Reasons)
	}
}

func TestNoSelectionIsANoOp(t *testing.T) {
	plan, err := PlanDeletion(dataFixture(), DeleteRequest{Action: ActionDeleteData, SelectedAreas: []string{"a"}})
	if err != nil {
		t.Fatal(err)
	}
	if !plan.DataDeleted {
		t.Fatalf("plan=%+v", plan)
	}
	empty, err := PlanDeletion(dataFixture(), DeleteRequest{Action: ActionDeleteData})
	if err != nil {
		t.Fatal(err)
	}
	if empty.DataDeleted || !contains(empty.Reasons, "no-selected-confirmed-area") {
		t.Fatalf("empty=%+v", empty)
	}
}

func TestVersionOnlyUpdateIsRefused(t *testing.T) {
	plan, err := PlanConversion(dataFixture(), ConvertRequest{
		FromVersion: "1", ToVersion: "2", Convert: false,
		Changes: []ConvertChange{{Area: "a", Kind: SegmentOwned, Before: "A1", After: "A1x", Preservable: true}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if plan.Allowed || plan.Converted || !contains(plan.Reasons, "conversion-required") {
		t.Fatalf("plan=%+v", plan)
	}
}

func TestConversionRequiresRules(t *testing.T) {
	plan, err := PlanConversion(dataFixture(), ConvertRequest{FromVersion: "1", ToVersion: "2", Convert: true})
	if err != nil {
		t.Fatal(err)
	}
	if plan.Allowed || !contains(plan.Reasons, "no-conversion-rules") {
		t.Fatalf("plan=%+v", plan)
	}
}

func TestUnpreservableChangeIsRefusedWithoutPartialSave(t *testing.T) {
	plan, err := PlanConversion(dataFixture(), ConvertRequest{
		FromVersion: "1", ToVersion: "2", Convert: true,
		Changes: []ConvertChange{{Area: "a", Kind: SegmentOwned, Before: "A1", After: "A1x", Preservable: false}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if plan.Allowed || plan.Converted || !contains(plan.Reasons, "unpreservable-change") {
		t.Fatalf("plan=%+v", plan)
	}
	if plan.FilesAfter != nil || !plan.SourceUnchanged {
		t.Fatalf("plan=%+v", plan)
	}
	if !contains(plan.Unpreservable, "a:owned") {
		t.Fatalf("unpreservable=%v", plan.Unpreservable)
	}
}

func TestConversionAppliesChangesAndKeepsUnknownBytes(t *testing.T) {
	plan, err := PlanConversion(dataFixture(), ConvertRequest{
		FromVersion: "1", ToVersion: "2", Convert: true,
		Changes: []ConvertChange{{Area: "a", Kind: SegmentOwned, Before: "A1/A2", After: "A-new", Preservable: true}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !plan.Allowed || !plan.Converted || plan.SourceUnchanged {
		t.Fatalf("plan=%+v", plan)
	}
	order := plan.FilesAfter[0]
	if !segmentBytes(order.Segments, "s1", "A-new") || !segmentBytes(order.Segments, "s3", "A-new") {
		t.Fatalf("converted segments=%+v", order.Segments)
	}
	if !segmentBytes(order.Segments, "s2", `<x-data q="z"/>`) || !segmentBytes(order.Segments, "s4", "<ext:note/>") {
		t.Fatalf("unknown/extension bytes changed: %+v", order.Segments)
	}
	if !plan.PreservedUnknown {
		t.Fatal("unknown bytes were not reported as preserved")
	}
}

func TestPartialConversionKeepsTheUnpreservableArea(t *testing.T) {
	plan, err := PlanConversion(dataFixture(), ConvertRequest{
		FromVersion: "1", ToVersion: "2", Convert: true, AllowPartial: true,
		Changes: []ConvertChange{
			{Area: "a", Kind: SegmentOwned, Before: "A1", After: "A-new", Preservable: true},
			{Area: "b", Kind: SegmentOwned, Before: "B1", After: "B-new", Preservable: false},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !plan.Allowed || !plan.Partial || !plan.Converted {
		t.Fatalf("plan=%+v", plan)
	}
	order := plan.FilesAfter[0]
	if !segmentBytes(order.Segments, "s5", "B1") {
		t.Fatalf("the unpreservable area was changed: %+v", order.Segments)
	}
	if !segmentBytes(order.Segments, "s1", "A-new") {
		t.Fatalf("the preservable area was not changed: %+v", order.Segments)
	}
}

func TestSameVersionIsNoChange(t *testing.T) {
	plan, err := PlanConversion(dataFixture(), ConvertRequest{FromVersion: "1", ToVersion: "1", Convert: true})
	if err != nil {
		t.Fatal(err)
	}
	if plan.Allowed || !contains(plan.Reasons, "no-change") {
		t.Fatalf("plan=%+v", plan)
	}
}

func segmentPresent(segments []Segment, key string) bool {
	for _, segment := range segments {
		if segment.Key == key {
			return true
		}
	}
	return false
}

func segmentBytes(segments []Segment, key, want string) bool {
	for _, segment := range segments {
		if segment.Key == key {
			return segment.Bytes == want
		}
	}
	return false
}

func hasRetained(values []RetainedArea, key, reason string) bool {
	for _, value := range values {
		if value.Key == key && value.Reason == reason {
			return true
		}
	}
	return false
}
