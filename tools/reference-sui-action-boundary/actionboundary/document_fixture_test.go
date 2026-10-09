package actionboundary

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"reflect"
	"sync"
	"testing"
)

// This fixture uses the actual SUI DocumentV1 field names (Card.x/Card.y,
// meta.source, holdState, reviewAttribution), but is not parsed by SUI's
// native TypeScript validator.
const documentFixture = `{"version":1,"id":"doc","createdAt":"2026-10-09T00:00:00Z","updatedAt":"2026-10-09T00:00:00Z","transform":{"panX":0,"panY":0,"zoom":1},"cards":[{"id":"a","text":"one","x":0,"y":1,"meta":{"source":"interview:1"},"holdState":"held","textReviewed":true},{"id":"b","text":"two","x":8,"y":9}],"reviewAttribution":{"schemaVersion":"1.0.0","reviewState":"human_reviewed","reviewedAt":"2026-10-09T00:00:00Z","reviewerRef":"reviewer:1","auditRecordedAt":"2026-10-09T00:00:00Z","overridePolicy":"human_dual_control_only"},"edges":[{"id":"edge-1","fromId":"a","toId":"b","type":"future-edge"}],"islands":[{"id":"island-1","cardIds":["a"]}],"affiliations":[{"id":"f1","cardId":"a","islandId":"island-1"}]}`

func documentIntent(revision string, payload any) Intent {
	return Intent{
		ApplicationID: "sui", DocumentID: "doc", ActionID: "sui.move",
		ExpectedRevision: revision, Payload: payload,
	}
}

func TestJSONDocumentStorePreservesUnrelatedFields(t *testing.T) {
	s := NewScopedJSONDocumentStore("doc", "r1", []byte(documentFixture))
	h := Host{Authorizer: permit(true), Bindings: []Binding{{ApplicationID: "sui", ActionID: "sui.move", Store: s}}}
	rev, err := h.Commit(context.Background(), documentIntent("r1", MovePayload{CardID: "a", X: 5, Y: 6}))
	if err != nil || rev != "r1:1" {
		t.Fatalf("rev=%s err=%v", rev, err)
	}
	_, got, writes := s.Snapshot()
	if writes != 1 {
		t.Fatalf("writes=%d", writes)
	}
	var before, after map[string]any
	if err := json.Unmarshal([]byte(documentFixture), &before); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(got, &after); err != nil {
		t.Fatal(err)
	}
	originalCards := before["cards"].([]any)
	nextCards := after["cards"].([]any)
	if len(nextCards) != len(originalCards) {
		t.Fatal("card count changed")
	}
	original := originalCards[0].(map[string]any)
	moved := nextCards[0].(map[string]any)
	if moved["x"] != float64(5) || moved["y"] != float64(6) {
		t.Fatalf("unexpected position: %v", moved)
	}
	original["x"], original["y"] = float64(5), float64(6)
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("fields other than target Card.x/y changed:\nbefore=%v\nafter=%v", before, after)
	}
}

func TestJSONDocumentStoreRejectsWithoutWrites(t *testing.T) {
	cases := []struct {
		name    string
		rev     string
		payload any
		want    error
	}{
		{"stale", "old", MovePayload{CardID: "a", X: 1, Y: 2}, ErrConflict},
		{"unknown card", "r1", MovePayload{CardID: "missing", X: 1, Y: 2}, ErrInvalid},
		{"missing card", "r1", MovePayload{X: 1, Y: 2}, ErrInvalid},
		{"wrong payload", "r1", "untyped", ErrInvalid},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := NewScopedJSONDocumentStore("doc", "r1", []byte(documentFixture))
			_, err := s.Commit(context.Background(), documentIntent(tc.rev, tc.payload))
			if !errors.Is(err, tc.want) {
				t.Fatalf("err=%v want=%v", err, tc.want)
			}
			rev, got, writes := s.Snapshot()
			if rev != "r1" || writes != 0 || string(got) != documentFixture {
				t.Fatalf("mutated on rejection: %s %d", rev, writes)
			}
		})
	}
}

func TestJSONDocumentStoreConcurrentRevision(t *testing.T) {
	s := NewScopedJSONDocumentStore("doc", "r1", []byte(documentFixture))
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := s.Commit(context.Background(), documentIntent("r1", MovePayload{CardID: "a", X: 1, Y: 2}))
			results <- err
		}()
	}
	wg.Wait()
	close(results)
	successes, conflicts := 0, 0
	for err := range results {
		if err == nil {
			successes++
		} else if errors.Is(err, ErrConflict) {
			conflicts++
		} else {
			t.Fatalf("unexpected %v", err)
		}
	}
	_, _, writes := s.Snapshot()
	if successes != 1 || conflicts != 1 || writes != 1 {
		t.Fatalf("success=%d conflict=%d writes=%d", successes, conflicts, writes)
	}
}

func TestJSONDocumentStoreRequiresDocumentIdentity(t *testing.T) {
	s := NewScopedJSONDocumentStore("doc", "r1", []byte(documentFixture))
	for _, id := range []string{"", "other"} {
		in := documentIntent("r1", MovePayload{CardID: "a", X: 1, Y: 2})
		in.DocumentID = id
		_, err := s.Commit(context.Background(), in)
		if !errors.Is(err, ErrDenied) {
			t.Fatalf("document %q: expected denied, got %v", id, err)
		}
	}
	revision, document, writes := s.Snapshot()
	if revision != "r1" || writes != 0 || string(document) != documentFixture {
		t.Fatal("identity rejection mutated document")
	}
}

func TestJSONDocumentStoreMalformedAndNonFinite(t *testing.T) {
	cases := []struct {
		name     string
		document string
		payload  MovePayload
	}{
		{"null document", "null", MovePayload{CardID: "a", X: 1, Y: 2}},
		{"null cards", `{"version":1,"id":"doc","cards":null}`, MovePayload{CardID: "a", X: 1, Y: 2}},
		{"missing cards", `{"version":1,"id":"doc"}`, MovePayload{CardID: "a", X: 1, Y: 2}},
		{"wrong embedded document", `{"version":1,"id":"other","cards":[{"id":"a","x":0,"y":0}]}`, MovePayload{CardID: "a", X: 1, Y: 2}},
		{"missing embedded document", `{"version":1,"cards":[{"id":"a","x":0,"y":0}]}`, MovePayload{CardID: "a", X: 1, Y: 2}},
		{"unsupported version", `{"version":2,"id":"doc","cards":[{"id":"a","x":0,"y":0}]}`, MovePayload{CardID: "a", X: 1, Y: 2}},
		{"non finite x", documentFixture, MovePayload{CardID: "a", X: math.NaN(), Y: 2}},
		{"non finite y", documentFixture, MovePayload{CardID: "a", X: 1, Y: math.Inf(1)}},
		{"duplicate card ids", `{"version":1,"id":"doc","cards":[{"id":"a","x":0,"y":0},{"id":"a","x":0,"y":0}]}`, MovePayload{CardID: "a", X: 1, Y: 2}},
		{"duplicate unrelated ids", `{"version":1,"id":"doc","cards":[{"id":"a","x":0,"y":0},{"id":"b"},{"id":"b"}]}`, MovePayload{CardID: "a", X: 1, Y: 2}},
		{"null card", `{"version":1,"id":"doc","cards":[null,{"id":"a","x":0,"y":0}]}`, MovePayload{CardID: "a", X: 1, Y: 2}},
		{"missing unrelated card id", `{"version":1,"id":"doc","cards":[{"id":"a","x":0,"y":0},{}]}`, MovePayload{CardID: "a", X: 1, Y: 2}},
		{"missing x", `{"version":1,"id":"doc","cards":[{"id":"a","y":0}]}`, MovePayload{CardID: "a", X: 1, Y: 2}},
		{"missing y", `{"version":1,"id":"doc","cards":[{"id":"a","x":0}]}`, MovePayload{CardID: "a", X: 1, Y: 2}},
		{"null x", `{"version":1,"id":"doc","cards":[{"id":"a","x":null,"y":0}]}`, MovePayload{CardID: "a", X: 1, Y: 2}},
		{"null y", `{"version":1,"id":"doc","cards":[{"id":"a","x":0,"y":null}]}`, MovePayload{CardID: "a", X: 1, Y: 2}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			store := NewScopedJSONDocumentStore("doc", "r1", []byte(tc.document))
			_, err := store.Commit(context.Background(), documentIntent("r1", tc.payload))
			if !errors.Is(err, ErrInvalid) {
				t.Fatalf("expected invalid, got %v", err)
			}
			revision, document, writes := store.Snapshot()
			if revision != "r1" || writes != 0 || string(document) != tc.document {
				t.Fatal("invalid input mutated state")
			}
		})
	}
}

func TestJSONDocumentStoreNeverReusesInitialRevision(t *testing.T) {
	s := NewScopedJSONDocumentStore("doc", "r2", []byte(documentFixture))
	first, err := s.Commit(context.Background(), documentIntent("r2", MovePayload{CardID: "a", X: 1, Y: 2}))
	if err != nil {
		t.Fatal(err)
	}
	if first == "r2" || first == "" {
		t.Fatalf("revision did not advance: %q", first)
	}
	_, err = s.Commit(context.Background(), documentIntent("r2", MovePayload{CardID: "a", X: 3, Y: 4}))
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("old revision replay accepted: %v", err)
	}
	second, err := s.Commit(context.Background(), documentIntent(first, MovePayload{CardID: "a", X: 3, Y: 4}))
	if err != nil || second == first || second == "r2" {
		t.Fatalf("revision did not advance: %q err=%v", second, err)
	}
}
