//go:build !tei_repair_only

package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func reviewFixture(t *testing.T) string {
	t.Helper()
	item := batchItem{OrderID: 1001, Result: commandScenarioResult{Stage: "completed"}}
	events := []batchJournalEvent{{Kind: "planned", Planned: []int64{1001, 1003}}, {Kind: "starting", OrderID: 1001},
		{Kind: "result", Item: &item}, {Kind: "starting", OrderID: 1003}}
	var raw strings.Builder
	for i, event := range events {
		event.Schema = "tei.reference.batch-progress/v1"
		event.Sequence = uint64(i + 1)
		line, err := json.Marshal(event)
		if err != nil {
			t.Fatal(err)
		}
		raw.Write(line)
		raw.WriteByte('\n')
	}
	return raw.String()
}

func TestBatchReviewPreservesIncompleteAndUntrustedMeaning(t *testing.T) {
	raw := reviewFixture(t)
	for _, suffix := range []string{"", `{"sequence":5,"kind":"result"`} {
		review, ids, err := decodeBatchReview([]byte(raw + suffix))
		if err != nil || review.Ended || review.IncompleteTail != (suffix != "") || len(ids) != 2 || review.Items[0].RecordedStage != "completed" || review.Items[1].Recorded != "started-without-result" {
			t.Fatalf("review: %+v %v", review, err)
		}
	}
}

func TestBatchReviewRejectsMalformedOrConflictingRecords(t *testing.T) {
	raw := reviewFixture(t)
	for _, invalid := range []string{
		"", `{"schema":"tei.reference.batch-progress/v1"}` + "\n",
		strings.Replace(raw, `"sequence":2`, `"sequence":4`, 1),
		strings.Replace(raw, `"orderId":1003`, `"orderId":1001`, 1),
		strings.Replace(raw, `"planned":[1001,1003]`, `"planned":[1001,1001]`, 1),
		strings.Replace(raw, `"orderId":1001`, `"orderId":1001,"orderId":1003`, 1),
		strings.Replace(raw, `"kind":"planned"`, `"unknown":true,"kind":"planned"`, 1),
		strings.Replace(raw, `"kind":"result"`, `"kind":"ended"`, 1),
	} {
		if _, _, err := decodeBatchReview([]byte(invalid)); err == nil {
			t.Fatalf("accepted invalid journal: %s", invalid)
		}
	}
}

func TestBatchReviewKeepsRecordedModelSeparateFromCurrentExecution(t *testing.T) {
	raw := strings.Replace(reviewFixture(t), `"kind":"planned"`, `"kind":"planned","composition":{"modelRevision":"R1"}`, 1)
	review, _, err := decodeBatchReview([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	if review.RecordedComposition == nil || review.RecordedComposition.ModelRevision != "R1" {
		t.Fatal("recorded model lost")
	}
	review.bindCurrent(batchObservationReceipt{Composition: &compositionTrace{ModelRevision: "R2"}})
	if review.ModelRevisionMatches == nil || *review.ModelRevisionMatches || review.RecordedComposition.ModelRevision != "R1" {
		t.Fatal("old log relabelled as current model")
	}
	review.bindCurrent(batchObservationReceipt{Composition: &compositionTrace{ModelRevision: "R1"}})
	if review.ModelRevisionMatches == nil || !*review.ModelRevisionMatches {
		t.Fatal("matching revisions not reported")
	}
	review.bindCurrent(batchObservationReceipt{})
	if review.ModelRevisionMatches != nil {
		t.Fatal("missing version treated as equality")
	}
}
