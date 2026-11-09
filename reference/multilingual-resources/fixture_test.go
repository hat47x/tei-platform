package multilingualresources

import (
	"encoding/json"
	"os"
	"testing"

	"tei/internal/modelref"
)

func TestOrderEntryFixture(t *testing.T) {
	// The graph is an isolated candidate fixture; loading it does not accept
	// any new TEI meaning or change the authoritative order model.
	var fixture struct {
		Model     modelref.Graph `json:"model"`
		Resources []Entry        `json:"resources"`
	}
	data, err := os.ReadFile("testdata/order-entry.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &fixture); err != nil {
		t.Fatal(err)
	}
	if got := Validate(fixture.Resources); len(got) != 0 {
		t.Fatalf("invalid fixture: %+v", got)
	}
	if len(fixture.Resources) != 1 {
		t.Fatalf("expected one display resource, got %d", len(fixture.Resources))
	}
	req := Request{Key: fixture.Resources[0].Key, Locale: "en-US"}
	if got := Resolve(fixture.Model, fixture.Resources, req); got.Diagnostic != "" || got.Text != "Confirm order" {
		t.Fatalf("approved translation unavailable: %+v", got)
	}
	req.Locale = "ja-JP"
	if got := Resolve(fixture.Model, fixture.Resources, req); got.Diagnostic != "" || got.Text != "受注を確定" || !got.FromSource {
		t.Fatalf("original language unavailable: %+v", got)
	}
	req.Locale = "fr-FR"
	req.Fallback = []string{"en-US"}
	if got := Resolve(fixture.Model, fixture.Resources, req); got.Diagnostic != "stale-translation" {
		t.Fatalf("stale translation bypassed with fallback: %+v", got)
	}
}
