package workbenchfixture

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"
	"time"
)

// TestJSONLIndependentExpected guards the authored synthetic fixture, not any
// TEI Provider, Plugin, CLI, Studio, or runtime behavior. A green test must not
// be promoted into provider conformance or actual read-completeness evidence.
func TestJSONLIndependentExpected(t *testing.T) {
	raw, err := os.ReadFile("events.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	lines := bytes.Split(bytes.TrimSpace(raw), []byte("\n"))
	if len(lines) != 5 {
		t.Fatalf("fixture must have 5 physical log records, got %d", len(lines))
	}

	wantIDs := []string{"evt-A", "evt-B", "evt-B", "evt-C", "evt-D"}
	counts := map[string]int{}
	var nullIngest, missingIngest, missingMessage int
	times := make([]time.Time, len(lines))
	for i, line := range lines {
		var doc map[string]json.RawMessage
		if err := json.Unmarshal(line, &doc); err != nil {
			t.Fatalf("record %d: %v", i+1, err)
		}
		var id, eventTime string
		if err := json.Unmarshal(doc["event_id"], &id); err != nil {
			t.Fatalf("event id in record %d: %v", i+1, err)
		}
		if id != wantIDs[i] {
			t.Fatalf("record %d: event_id=%q, want %q", i+1, id, wantIDs[i])
		}
		counts[id]++
		if err := json.Unmarshal(doc["event_time"], &eventTime); err != nil {
			t.Fatalf("event time in record %d: %v", i+1, err)
		}
		times[i], err = time.Parse(time.RFC3339, eventTime)
		if err != nil {
			t.Fatalf("event time format in record %d: %v", i+1, err)
		}
		if value, present := doc["ingest_time"]; !present {
			missingIngest++
		} else if bytes.Equal(bytes.TrimSpace(value), []byte("null")) {
			nullIngest++
		}
		if _, present := doc["message"]; !present {
			missingMessage++
		}
	}
	if len(counts) != 4 || counts["evt-B"] != 2 {
		t.Fatalf("event identity/duplicates changed: %#v", counts)
	}
	if !times[1].Before(times[0]) {
		t.Fatal("second record must have an event time earlier than the first")
	}
	if nullIngest != 1 || missingIngest != 1 || missingMessage != 1 {
		t.Fatalf("presence/null expected 1/1/1, got %d/%d/%d", nullIngest, missingIngest, missingMessage)
	}
}
