// E24 distinguishes an accepted-looking typed source from a validated wire
// command. Only temporary candidate Sources are changed; SUI remains the owner
// of the real card-move input, resource revision, authority and persistence.
package main

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/driver"
	"tei/internal/validation"
	"tei/internal/workspace"
	"tei/tools/reference-sui-action-boundary/actionboundary"
)

const e24TypedNativeSource = `namespace sui

data CardMoveInput {
    attribute cardId type std.text
    attribute x type std.decimal
    attribute y type std.decimal
}

capability sui.move {
    input move sui.card_move_input
}
`

func e24TypedWorkspace(t *testing.T, capability string) workspace.Result {
	t.Helper()
	root := e22Workspace(t, true, false)
	path := filepath.Join(root, "semantic", "sui", "native-action.tei")
	if err := os.WriteFile(path, []byte(capability), 0600); err != nil {
		t.Fatal(err)
	}
	loaded, err := workspace.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	return loaded
}

func TestE24TypedNativeCandidateRetainsSourceAndShape(t *testing.T) {
	loaded := e24TypedWorkspace(t, e24TypedNativeSource)
	r, err := driver.NewResolver(loaded.World).ResolvePattern("sui", "sui.move")
	if err != nil {
		t.Fatal(err)
	}
	if r.Capability == nil || len(r.Capability.Inputs) != 1 {
		t.Fatalf("typed capability did not resolve: %+v", r.Capability)
	}
	if got := string(r.Capability.Inputs[0].Name); got != "move" {
		t.Fatalf("unexpected parameter name: %q", got)
	}
	if got := string(r.Capability.Inputs[0].Type); got != "sui.card_move_input" {
		t.Fatalf("unexpected parameter type: %q", got)
	}

	found := false
	for _, namespace := range loaded.World.Semantics {
		for _, data := range namespace.Data {
			if data.ID != "sui.card_move_input" {
				continue
			}
			found = true
			want := map[string]string{
				"cardId": "std.text",
				"x":      "std.decimal",
				"y":      "std.decimal",
			}
			for _, attribute := range data.Attributes {
				key := attribute.Symbol
				if string(attribute.TypeRef) != want[key] {
					t.Fatalf("unexpected attribute %s: %s", key, attribute.TypeRef)
				}
				delete(want, key)
			}
			if len(want) != 0 {
				t.Fatalf("candidate does not retain the three input attributes: %v", want)
			}
		}
	}
	if !found {
		t.Fatal("typed input data declaration missing")
	}
	if _, ok := loaded.Sources["semantic-capability:sui.move"]; !ok {
		t.Fatal("capability source not indexed")
	}
	if _, ok := loaded.Sources["semantic-data:sui.card_move_input"]; !ok {
		t.Fatal("typed input data source not indexed")
	}
	for _, diagnostic := range validation.Validate(loaded.World) {
		if diagnostic.Code == "UNKNOWN_SEMANTIC_CAPABILITY_INPUT" ||
			diagnostic.Code == "UNKNOWN_SEMANTIC_ATTRIBUTE_TYPE" {
			t.Fatalf("known structural input types unexpectedly failed: %+v", diagnostic)
		}
	}
}

func TestE24MissingDeclaredInputTypeNeedsExplicitValidation(t *testing.T) {
	// The Driver performs a semantic lookup, not the complete acceptance gate.
	// A dangling input type cannot become accepted merely because the action
	// and Capability identifiers resolve.
	badSource := strings.Replace(e24TypedNativeSource,
		"input move sui.card_move_input", "input move sui.absent_input", 1)
	loaded := e24TypedWorkspace(t, badSource)
	if _, err := driver.NewResolver(loaded.World).ResolvePattern("sui", "sui.move"); err != nil {
		t.Fatalf("Driver lookup must be assessed separately from validation: %v", err)
	}
	found := false
	for _, diagnostic := range validation.Validate(loaded.World) {
		if diagnostic.Code == "UNKNOWN_SEMANTIC_CAPABILITY_INPUT" {
			found = true
		}
	}
	if !found {
		t.Fatal("missing input type escaped explicit semantic validation")
	}
}

type e24AllowReferenceOnly struct{}

func (e24AllowReferenceOnly) Authorize(context.Context, string, string, string) error {
	return nil
}

func TestE24TypedDeclarationDoesNotReplaceApplicationPayloadDecoder(t *testing.T) {
	loaded := e24TypedWorkspace(t, e24TypedNativeSource)
	store := &e22CountingStore{}
	gate := &actionboundary.CanonicalCommandGate{Resolver: driver.NewResolver(loaded.World)}
	// A command lookup intentionally does not interpret its opaque payload.
	// This is NOT permission to send malformed input to an actual Store.
	illTyped := actionboundary.Intent{
		ApplicationID: "sui",
		ResourceID:    "doc-1",
		ActionID:      "sui.move",
		ExpectedRevision: "v1",
		Payload:       map[string]any{"cardId": 42, "x": "invalid", "y": nil},
	}
	if err := gate.Check(context.Background(), illTyped); err != nil {
		t.Fatalf("CommandGate performs identity lookup, not input decoding: %v", err)
	}
	handler := actionboundary.JSONCommitHandler{
		Host: actionboundary.Host{
			CommandGate: gate,
			Authorizer:  e24AllowReferenceOnly{},
			Bindings: []actionboundary.Binding{{
				ApplicationID: "sui", ActionID: "sui.move", Store: store,
			}},
		},
		// No SUI-owned decoder is registered, despite the typed declaration.
	}
	if got := e20HTTPCommit(t, handler, "sui", "doc-1", "sui.move", "v1",
		json.RawMessage(`{"cardId":42,"x":"invalid","y":null}`)); got != http.StatusServiceUnavailable {
		t.Fatalf("missing application decoder must fail closed (HTTP %d)", got)
	}
	if store.writes != 0 {
		t.Fatalf("declaration was mistaken for an authorized decoded action (%d writes)", store.writes)
	}
}
