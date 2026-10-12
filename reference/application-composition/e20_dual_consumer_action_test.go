// E20 compares the same declared Action boundary for real order_entry and a
// candidate SUI native consumer. It does not execute a browser or SUI service.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/driver"
	"tei/internal/semantic"
	"tei/internal/workspace"
	"tei/tools/reference-sui-action-boundary/actionboundary"
)

// The actual order_entry command resolves from reference/workspace.
// SUI Stage 0 has no accepted TEI Application yet, so only that one candidate
// is added to the test resolver. It is not registered in the real Workspace.
type e20Resolver struct{ order driver.Resolver }

func (r e20Resolver) ResolvePattern(app, pattern core.CanonicalID) (driver.PatternResolution, error) {
	if app == "sui" && pattern == "sui.move" {
		return driver.PatternResolution{
			Application: application.Application{ID: "sui"},
			Pattern:     application.Pattern{ID: "sui.move", Kind: application.PatternCommand},
			Capability:  &semantic.Capability{ID: "sui.move_candidate"},
		}, nil
	}
	return r.order.ResolvePattern(app, pattern)
}

type e20Authorizer struct{ deny bool }

func (a e20Authorizer) Authorize(_ context.Context, app, resource, action string) error {
	if a.deny {
		return actionboundary.ErrDenied
	}
	if (app == "order_entry" && resource == "order-1" && action == "order_entry.submit_order") ||
		(app == "sui" && resource == "doc-1" && action == "sui.move") {
		return nil
	}
	return actionboundary.ErrDenied
}

// Order persistence remains application-owned; it is a recording fixture,
// not a second definition of the order domain or a production provider.
type e20OrderStore struct {
	revision string
	writes   int
}

type e20OrderPayload struct {
	OrderID string `json:"orderId"`
}

func (s *e20OrderStore) Commit(_ context.Context, intent actionboundary.Intent) (string, error) {
	if intent.ResourceID != "order-1" || intent.ActionID != "order_entry.submit_order" {
		return "", actionboundary.ErrDenied
	}
	if intent.ExpectedRevision != s.revision {
		return "", actionboundary.ErrConflict
	}
	p, ok := intent.Payload.(e20OrderPayload)
	if !ok || p.OrderID != "order-1" {
		return "", actionboundary.ErrInvalid
	}
	s.writes++
	s.revision = "order-v2"
	return s.revision, nil
}

const e20SUIDocument = `{"version":1,"id":"doc-1","cards":[{"id":"card-1","x":1,"y":2,"meta":{"source":"sui-original"},"holdState":"held"}],"edges":[{"id":"edge-1","type":"unknown-extension"}],"reviewAttribution":{"reviewerRef":"human-1"}}`

func e20Fixture(t *testing.T, deny bool) (actionboundary.JSONCommitHandler, *e20OrderStore, *actionboundary.JSONDocumentStore) {
	t.Helper()
	loaded, err := workspace.Load(filepath.Join("..", "..", "reference", "workspace"))
	if err != nil {
		t.Fatalf("real order-entry Workspace cannot be loaded: %v", err)
	}
	// The real driver must resolve the command and its capability. A declared
	// Pattern with no backing capability is not sufficient for this test.
	resolved, err := driver.NewResolver(loaded.World).ResolvePattern("order_entry", "order_entry.submit_order")
	if err != nil || resolved.Pattern.Kind != application.PatternCommand || resolved.Capability == nil {
		t.Fatalf("real order-entry Command is not resolvable: %v", err)
	}
	order := &e20OrderStore{revision: "order-v1"}
	sui := actionboundary.NewScopedJSONDocumentStore("doc-1", "sui-v1", []byte(e20SUIDocument))
	host := actionboundary.Host{
		CommandGate: &actionboundary.CanonicalCommandGate{Resolver: e20Resolver{order: driver.NewResolver(loaded.World)}},
		Authorizer:  e20Authorizer{deny: deny},
		Bindings: []actionboundary.Binding{
			{ApplicationID: "order_entry", ActionID: "order_entry.submit_order", Store: order},
			{ApplicationID: "sui", ActionID: "sui.move", Store: sui},
		},
	}
	decode := func(app, action string, raw json.RawMessage) (any, error) {
		switch {
		case app == "order_entry" && action == "order_entry.submit_order":
			var p e20OrderPayload
			if err := json.Unmarshal(raw, &p); err != nil || p.OrderID == "" {
				return nil, actionboundary.ErrInvalid
			}
			return p, nil
		case app == "sui" && action == "sui.move":
			var p actionboundary.MovePayload
			if err := json.Unmarshal(raw, &p); err != nil || p.CardID == "" {
				return nil, actionboundary.ErrInvalid
			}
			return p, nil
		default:
			return nil, actionboundary.ErrDenied
		}
	}
	return actionboundary.JSONCommitHandler{Host: host, DecodePayload: decode}, order, sui
}

func e20HTTPCommit(t *testing.T, handler actionboundary.JSONCommitHandler,
	app, resource, action, revision string, payload any) int {
	t.Helper()
	request := map[string]any{
		"protocolVersion": "1", "applicationID": app, "resourceID": resource,
		"actionID": action, "expectedRevision": revision, "payload": payload,
	}
	raw, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "http://tei.example/action", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-TEI-Action", "commit")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)
	return recorder.Code
}

func TestE20TwoConsumerHostUsesApplicationOwnedActions(t *testing.T) {
	handler, order, sui := e20Fixture(t, false)
	orderPayload := e20OrderPayload{OrderID: "order-1"}
	suiPayload := actionboundary.MovePayload{CardID: "card-1", X: 12, Y: 34}

	// Native previews and cancellation remain in the SUI Canvas; there is
	// deliberately no TEI request before the final, explicit commit.
	if order.writes != 0 {
		t.Fatal("the Host wrote before either consumer committed")
	}
	_, _, writes := sui.Snapshot()
	if writes != 0 {
		t.Fatal("native preview unexpectedly wrote the SUI Document")
	}
	if got := e20HTTPCommit(t, handler, "order_entry", "order-1",
		"order_entry.submit_order", "order-v1", orderPayload); got != http.StatusOK {
		t.Fatalf("actual order_entry Command was rejected: HTTP %d", got)
	}
	if order.writes != 1 {
		t.Fatalf("order Action was not committed exactly once: %d", order.writes)
	}
	if _, _, n := sui.Snapshot(); n != 0 {
		t.Fatal("order Action mutated the SUI Store")
	}
	if got := e20HTTPCommit(t, handler, "sui", "doc-1", "sui.move", "sui-v1", suiPayload); got != http.StatusOK {
		t.Fatalf("SUI candidate Action was rejected: HTTP %d", got)
	}
	if order.writes != 1 {
		t.Fatal("SUI Action mutated the order Store")
	}
	rev, content, n := sui.Snapshot()
	if rev != "sui-v1:1" || n != 1 {
		t.Fatalf("unexpected SUI revision/writes: %s / %d", rev, n)
	}
	var doc map[string]any
	if err := json.Unmarshal(content, &doc); err != nil {
		t.Fatal(err)
	}
	card := doc["cards"].([]any)[0].(map[string]any)
	if card["x"] != float64(12) || card["y"] != float64(34) ||
		card["holdState"] != "held" || card["meta"].(map[string]any)["source"] != "sui-original" ||
		doc["reviewAttribution"].(map[string]any)["reviewerRef"] != "human-1" ||
		doc["edges"].([]any)[0].(map[string]any)["type"] != "unknown-extension" {
		t.Fatal("SUI candidate Action corrupted fields outside the selected Card coordinates")
	}
}

func TestE20CompositionCannotCrossBindOrBypassAuthority(t *testing.T) {
	tests := []struct {
		name, app, resource, action, revision string
		payload                               any
		deny                                  bool
		want                                  int
	}{
		{"SUI action claimed by order-entry", "order_entry", "order-1", "sui.move", "order-v1", actionboundary.MovePayload{CardID: "card-1", X: 2, Y: 3}, false, http.StatusBadRequest},
		{"order action claimed by SUI", "sui", "doc-1", "order_entry.submit_order", "sui-v1", e20OrderPayload{OrderID: "order-1"}, false, http.StatusBadRequest},
		{"foreign SUI resource", "sui", "other-doc", "sui.move", "sui-v1", actionboundary.MovePayload{CardID: "card-1", X: 2, Y: 3}, false, http.StatusForbidden},
		{"foreign order resource", "order_entry", "other-order", "order_entry.submit_order", "order-v1", e20OrderPayload{OrderID: "order-1"}, false, http.StatusForbidden},
		{"stale SUI revision", "sui", "doc-1", "sui.move", "obsolete", actionboundary.MovePayload{CardID: "card-1", X: 2, Y: 3}, false, http.StatusConflict},
		{"stale order revision", "order_entry", "order-1", "order_entry.submit_order", "obsolete", e20OrderPayload{OrderID: "order-1"}, false, http.StatusConflict},
		{"denied native Action", "sui", "doc-1", "sui.move", "sui-v1", actionboundary.MovePayload{CardID: "card-1", X: 2, Y: 3}, true, http.StatusForbidden},
		{"denied order Action", "order_entry", "order-1", "order_entry.submit_order", "order-v1", e20OrderPayload{OrderID: "order-1"}, true, http.StatusForbidden},
		{"invalid native payload", "sui", "doc-1", "sui.move", "sui-v1", map[string]any{"x": 1}, false, http.StatusBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler, order, sui := e20Fixture(t, tt.deny)
			if got := e20HTTPCommit(t, handler, tt.app, tt.resource,
				tt.action, tt.revision, tt.payload); got != tt.want {
				t.Fatalf("HTTP %d, want %d", got, tt.want)
			}
			if order.writes != 0 {
				t.Fatalf("rejected Action mutated order Store: %d", order.writes)
			}
			rev, doc, writes := sui.Snapshot()
			if rev != "sui-v1" || writes != 0 || string(doc) != e20SUIDocument {
				t.Fatalf("rejected Action mutated SUI: rev=%s writes=%d", rev, writes)
			}
		})
	}

	// Removing the real runtime authorizer must never turn an otherwise
	// valid binding and Canonical Pattern into an authorized write.
	handler, order, sui := e20Fixture(t, false)
	handler.Host.Authorizer = nil
	got := e20HTTPCommit(t, handler, "sui", "doc-1", "sui.move", "sui-v1",
		actionboundary.MovePayload{CardID: "card-1", X: 2, Y: 3})
	if got != http.StatusForbidden {
		t.Fatalf("missing authorization returned %d", got)
	}
	if order.writes != 0 {
		t.Fatal("missing authorizer mutated order")
	}
	if _, _, writes := sui.Snapshot(); writes != 0 {
		t.Fatal("missing authorizer mutated SUI")
	}
}

var _ actionboundary.Store = (*e20OrderStore)(nil)
var _ actionboundary.Authorizer = e20Authorizer{}
var _ actionboundary.PatternResolver = e20Resolver{}
