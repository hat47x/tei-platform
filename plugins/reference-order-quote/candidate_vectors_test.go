package main

import (
	"encoding/json"
	"os"
	"testing"
)

// Direct invocation intentionally excludes the process carrier, plugin resolution,
// and Host authorization. It is only the non-Plugin calculation baseline (D0).
// Share the unaccepted candidate vectors with the existing Go/JS process test
// rather than creating a second business oracle.
func TestQuoteDirectCalculationMatchesSharedCandidateVectors(t *testing.T) {
	raw, err := os.ReadFile("../../reference/candidates/order-quote/semantic-vectors-v0.json")
	if err != nil {
		t.Fatal(err)
	}
	var corpus struct {
		Schema   string `json:"schema"`
		Status   string `json:"status"`
		Contract string `json:"contract"`
		Vectors  []struct {
			Name    string       `json:"name"`
			Request string       `json:"request"`
			Expect  *quoteResult `json:"expect"`
			Reject  bool         `json:"reject"`
		} `json:"vectors"`
	}
	if err := json.Unmarshal(raw, &corpus); err != nil {
		t.Fatal(err)
	}
	if corpus.Schema != "tei.reference.quote-semantic-vectors/candidate-v0" ||
		corpus.Status != "unaccepted" ||
		corpus.Contract != "reference.order.quote-lines/v1" ||
		len(corpus.Vectors) != 23 {
		t.Fatalf("unexpected candidate corpus: schema=%q status=%q contract=%q count=%d",
			corpus.Schema, corpus.Status, corpus.Contract, len(corpus.Vectors))
	}
	seen := make(map[string]bool)
	accepted, rejected := 0, 0
	for _, vector := range corpus.Vectors {
		if vector.Name == "" || seen[vector.Name] ||
			vector.Reject == (vector.Expect != nil) ||
			!json.Valid([]byte(vector.Request)) {
			t.Fatalf("invalid candidate vector %q", vector.Name)
		}
		seen[vector.Name] = true
		t.Run(vector.Name, func(t *testing.T) {
			// Pass the original JSON lexeme without normalizing through a map:
			// duplicate keys and exact number spellings are meaningful negatives.
			result, err := calculate([]byte(vector.Request))
			if vector.Reject {
				if err == nil {
					t.Fatalf("candidate should be rejected; got %+v", result)
				}
				return
			}
			if err != nil {
				t.Fatalf("candidate should be accepted: %v", err)
			}
			if result != *vector.Expect {
				t.Fatalf("direct calculation=%+v want=%+v", result, *vector.Expect)
			}
		})
		if vector.Reject {
			rejected++
		} else {
			accepted++
		}
	}
	if accepted != 8 || rejected != 15 {
		t.Fatalf("candidate coverage: accepted=%d rejected=%d", accepted, rejected)
	}
}
