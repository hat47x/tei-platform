package actionboundary

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http/httptest"
	"testing"
)

// The two applications deliberately use different native payload schemas.
// Neither schema belongs to the Host or HTTP protocol envelope.
type inventoryAdjust struct {
	SKU   string `json:"sku"`
	Delta int    `json:"delta"`
}

type protocolStore struct {
	resource string
	revision string
	writes   int
	last     any
}

func (s *protocolStore) Commit(_ context.Context, intent Intent) (string, error) {
	if intent.ResourceID != s.resource {
		return "", ErrDenied
	}
	if intent.ExpectedRevision != s.revision {
		return "", ErrConflict
	}
	if intent.Payload == nil {
		return "", ErrInvalid
	}
	s.last = intent.Payload
	s.writes++
	s.revision = fmt.Sprintf("v%d", s.writes+1)
	return s.revision, nil
}

func TestStandardActionProtocolSupportsIndependentApplications(t *testing.T) {
	sui := &protocolStore{resource: "canvas-1", revision: "v1"}
	inventory := &protocolStore{resource: "sku-store/42", revision: "v1"}
	decoded := []string{}
	handler := JSONCommitHandler{
		Host: Host{
			Authorizer: permit(true),
			Bindings:   []Binding{
				{ApplicationID: "sui", ActionID: "sui.move", Store: sui},
				{ApplicationID: "inventory", ActionID: "inventory.adjust", Store: inventory},
			},
		},
		DecodePayload: func(application, action string, raw json.RawMessage) (any, error) {
			decoded = append(decoded, application+"/"+action)
			switch application + ":" + action {
			case "sui:sui.move":
				var value MovePayload
				if err := json.Unmarshal(raw, &value); err != nil || value.CardID == "" {
					return nil, ErrInvalid
				}
				return value, nil
			case "inventory:inventory.adjust":
				var value inventoryAdjust
				if err := json.Unmarshal(raw, &value); err != nil || value.SKU == "" {
					return nil, ErrInvalid
				}
				return value, nil
			}
			return nil, ErrInvalid
		},
	}
	send := func(application, resource, action, revision, payload string) *httptest.ResponseRecorder {
		raw, err := json.Marshal(map[string]any{
			"protocolVersion": "1", "applicationID": application, "resourceID": resource,
			"actionID": action, "expectedRevision": revision,
			"payload": json.RawMessage(payload),
		})
		if err != nil {
			t.Fatal(err)
		}
		request := httptest.NewRequest("POST", "http://example.test/actions/commit", bytes.NewReader(raw))
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("X-TEI-Action", "commit")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, request)
		return w
	}
	for _, tc := range []struct{ app, resource, action, payload string }{
		{"sui", "canvas-1", "sui.move", `{"cardId":"a","x":10,"y":20}`},
		{"inventory", "sku-store/42", "inventory.adjust", `{"sku":"p100","delta":4}`},
	} {
		first := send(tc.app, tc.resource, tc.action, "v1", tc.payload)
		if first.Code != 200 || !bytes.Contains(first.Body.Bytes(), []byte(`"protocolVersion":"1"`)) {
			t.Fatalf("%s first commit: %d %s", tc.app, first.Code, first.Body.String())
		}
		stale := send(tc.app, tc.resource, tc.action, "v1", tc.payload)
		if stale.Code != 409 {
			t.Fatalf("%s expected revision conflict: %d %s", tc.app, stale.Code, stale.Body.String())
		}
	}
	if sui.writes != 1 || inventory.writes != 1 {
		t.Fatalf("cross-app writes: sui=%d inventory=%d", sui.writes, inventory.writes)
	}
	if _, ok := sui.last.(MovePayload); !ok {
		t.Fatalf("SUI adapter decoded wrong payload type %T", sui.last)
	}
	if got, ok := inventory.last.(inventoryAdjust); !ok || got.SKU != "p100" || got.Delta != 4 {
		t.Fatalf("inventory payload lost: %#v", inventory.last)
	}
	// Same action shape with a foreign application/resource cannot escape
	// an application's own binding and resource-scoped Store.
	foreign := send("inventory", "canvas-1", "inventory.adjust", "v2", `{"sku":"p100","delta":1}`)
	if foreign.Code != 403 || inventory.writes != 1 || sui.writes != 1 {
		t.Fatalf("cross-resource request mutated state: %d", foreign.Code)
	}
	// A tuple from a third application is not dispatched by accidental
	// equivalence to either application's payload grammar.
	unknown := send("crm", "canvas-1", "sui.move", "v1", `{"cardId":"a","x":5,"y":6}`)
	if unknown.Code != 400 {
		t.Fatalf("unbound app/action accepted: %d", unknown.Code)
	}
	if len(decoded) != 6 {
		t.Fatalf("expected 6 app-scoped decodes, got %d: %#v", len(decoded), decoded)
	}
}

func TestStandardActionProtocolRejectsMalformedAppOwnedPayload(t *testing.T) {
	store := &protocolStore{resource: "thing", revision: "v1"}
	h := JSONCommitHandler{
		Host: Host{Authorizer: permit(true), Bindings: []Binding{{ApplicationID: "inventory", ActionID: "inventory.adjust", Store: store}}},
		DecodePayload: func(appID, actionID string, raw json.RawMessage) (any, error) {
			if appID != "inventory" || actionID != "inventory.adjust" {
				return nil, errors.New("unknown tuple")
			}
			var payload inventoryAdjust
			if err := json.Unmarshal(raw, &payload); err != nil || payload.SKU == "" {
				return nil, ErrInvalid
			}
			return payload, nil
		},
	}
	body := `{"protocolVersion":"1","applicationID":"inventory","resourceID":"thing","actionID":"inventory.adjust","expectedRevision":"v1","payload":{}}`
	r := httptest.NewRequest("POST", "http://example.test/commit", bytes.NewBufferString(body))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("X-TEI-Action", "commit")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != 400 || store.writes != 0 {
		t.Fatalf("malformed payload wrote resource: %d %d", w.Code, store.writes)
	}
}
