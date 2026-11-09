package operations_test

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"tei/internal/operations"
	"tei/internal/sourceindex"
)

// TestInspectStatesWhatItDoesNotGrant pins #1178's disclosure boundary at the
// answer shape. The inspection carries Source paths and reference counts, so a
// reader must not read a located Source as editability, acceptance, or a
// permission to pass the answer to another party. trace and catalog already
// stated this; inspect did not.
func TestInspectStatesWhatItDoesNotGrant(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "workspace")
	result, err := operations.InspectSubject(root, sourceindex.Subject{
		Kind: "semantic-data", ID: "sales.order",
	})
	if err != nil {
		t.Fatalf("inspect: %v", err)
	}
	want := []string{
		"inspection-is-a-derived-read-only-projection",
		"source-location-is-not-semantic-acceptance",
		"source-location-is-not-write-authority",
		"local-read-is-not-disclosure-permission",
	}
	got := result.AuthorityLimits
	if len(got) != len(want) {
		t.Fatalf("authorityLimits = %v, want %v", got, want)
	}
	for i, value := range want {
		if got[i] != value {
			t.Fatalf("authorityLimits[%d] = %q, want %q", i, got[i], value)
		}
	}
}

// The field must be on the wire with that name: it is what a caller reads.
func TestInspectAuthorityLimitsWireName(t *testing.T) {
	encoded, err := json.Marshal(operations.SubjectInspection{
		SchemaVersion:   "tei.operations.inspect/v0",
		AuthorityLimits: operations.InspectionAuthorityLimits(),
	})
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &fields); err != nil {
		t.Fatal(err)
	}
	raw, present := fields["authorityLimits"]
	if !present {
		t.Fatalf("authorityLimits absent from %s", encoded)
	}
	var limits []string
	if err := json.Unmarshal(raw, &limits); err != nil {
		t.Fatal(err)
	}
	if len(limits) != len(operations.InspectionAuthorityLimits()) {
		t.Fatalf("authorityLimits = %v", limits)
	}
}

// A subject with no indexed Source still states the limits: the statement is
// about what the answer does not grant, not about whether a location was found.
func TestInspectStatesLimitsWithoutSourceReferences(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "workspace")
	result, err := operations.InspectSubject(root, sourceindex.Subject{
		Kind: "semantic-type", ID: "std.text",
	})
	if err != nil {
		t.Fatalf("inspect: %v", err)
	}
	if result.SourceStatus != "not-indexed" {
		t.Fatalf("fixture should have no Source, status = %q", result.SourceStatus)
	}
	if len(result.AuthorityLimits) == 0 {
		t.Fatal("an unindexed subject lost the authority limits")
	}
}

// The statement must not claim an authorization decision: it says a local read
// is not a disclosure permission, and nothing more.
func TestInspectAuthorityLimitsDoNotClaimAuthorization(t *testing.T) {
	for _, limit := range operations.InspectionAuthorityLimits() {
		switch limit {
		case "inspection-is-a-derived-read-only-projection",
			"source-location-is-not-semantic-acceptance",
			"source-location-is-not-write-authority",
			"local-read-is-not-disclosure-permission":
		default:
			t.Fatalf("unexpected limit %q: the list is a fixed statement, not a decision", limit)
		}
	}
}
