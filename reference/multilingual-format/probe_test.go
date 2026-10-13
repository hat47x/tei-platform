package multilingualformat

import (
	"encoding/json"
	"os"
	"testing"
)

type testCase struct {
	ID       string `json:"id"`
	Locale   string `json:"locale"`
	Count    int64  `json:"count"`
	Category string `json:"category"`
	Number   string `json:"number"`
}

func TestSharedIntegerForms(t *testing.T) {
	data, err := os.ReadFile("testdata/integer-forms.json")
	if err != nil {
		t.Fatal(err)
	}
	var corpus struct {
		Schema string     `json:"schema"`
		Cases  []testCase `json:"cases"`
	}
	if err := json.Unmarshal(data, &corpus); err != nil {
		t.Fatal(err)
	}
	if corpus.Schema != "tei.multilingual.integer-forms.e2a" || len(corpus.Cases) == 0 {
		t.Fatalf("unexpected corpus: %q; %d cases", corpus.Schema, len(corpus.Cases))
	}
	seen := map[string]bool{}
	for _, tc := range corpus.Cases {
		t.Run(tc.ID, func(t *testing.T) {
			if tc.ID == "" || seen[tc.ID] {
				t.Fatal("missing or duplicated case ID")
			}
			seen[tc.ID] = true
			got, err := Probe(tc.Locale, tc.Count)
			if err != nil {
				t.Fatal(err)
			}
			if got.Category != tc.Category || got.Number != tc.Number {
				t.Fatalf("Go CLDR difference in %s: got %+v; expected %s / %s", tc.ID, got, tc.Category, tc.Number)
			}
		})
	}
}

func TestUnsupportedLocaleAndInvalidCounts(t *testing.T) {
	for _, tc := range []struct {
		name   string
		locale string
		count  int64
	}{
		{"unlisted locale", "fr-FR", 1},
		{"invalid tag", "EN_us", 1},
		{"negative", "en-US", -1},
		{"overflow of cross-target range", "en-US", MaxCount + 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := Probe(tc.locale, tc.count)
			if err == nil || got != (Result{}) {
				t.Fatalf("unsafe input accepted: got %+v err %v", got, err)
			}
		})
	}
}

func TestProbeDeterministic(t *testing.T) {
	initial, err := Probe("ja-JP", 12345)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 50; i++ {
		got, err := Probe("ja-JP", 12345)
		if err != nil || got != initial {
			t.Fatalf("inconsistent result: %+v; %v", got, err)
		}
	}
}
