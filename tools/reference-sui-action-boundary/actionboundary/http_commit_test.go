package actionboundary

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"
)

type httpTestStore struct {
	calls int
	err   error
}

func (s *httpTestStore) Commit(_ context.Context, _ Intent) (string, error) {
	s.calls++
	return "r2", s.err
}
func TestJSONCommitHandler(t *testing.T) {
	good := `{"protocolVersion":"1","applicationID":"sui","resourceID":"doc","actionID":"move","expectedRevision":"r1","payload":{"x":1}}`
	for _, tt := range []struct {
		name      string
		method    string
		body      string
		origin    string
		header    bool
		commitErr error
		expected  int
		calls     int
		noAuth    bool
		noDecoder bool
	}{
		{name: "commit once", method: "POST", body: good, header: true, expected: 200, calls: 1},
		{name: "denied", method: "POST", body: good, header: true, commitErr: ErrDenied, expected: 403, calls: 1},
		{name: "conflict", method: "POST", body: good, header: true, commitErr: ErrConflict, expected: 409, calls: 1},
		{name: "invalid", method: "POST", body: good, header: true, commitErr: ErrInvalid, expected: 400, calls: 1},
		{name: "unavailable authorization", method: "POST", body: good, header: true, noAuth: true, expected: 403},
		{name: "unavailable decoder", method: "POST", body: good, header: true, noDecoder: true, expected: 503},
		{name: "unknown field", method: "POST", body: `{"protocolVersion":"1","applicationID":"sui","resourceID":"doc","actionID":"move","expectedRevision":"r1","payload":{"x":1},"subject":"admin"}`, header: true, expected: 400},
		{name: "wrong origin", method: "POST", body: good, origin: "https://evil.example", header: true, expected: 403},
		{name: "same origin", method: "POST", body: good, origin: "http://example.test", header: true, expected: 200, calls: 1},
		{name: "scheme mismatch", method: "POST", body: good, origin: "https://example.test", header: true, expected: 403},
		{name: "protocol missing", method: "POST", body: strings.Replace(good, `"protocolVersion":"1",`, "", 1), header: true, expected: 400},
		{name: "null payload", method: "POST", body: strings.Replace(good, `"payload":{"x":1}`, `"payload":null`, 1), header: true, expected: 400},
		{name: "whitespace null payload", method: "POST", body: strings.Replace(good, `"payload":{"x":1}`, `"payload":  null  `, 1), header: true, expected: 400},
		{name: "protocol unsupported", method: "POST", body: strings.Replace(good, `"protocolVersion":"1"`, `"protocolVersion":"2"`, 1), header: true, expected: 400},
		{name: "missing resource", method: "POST", body: strings.Replace(good, `"resourceID":"doc"`, `"resourceID":""`, 1), header: true, expected: 400},
		{name: "legacy documentID rejected", method: "POST", body: strings.Replace(good, `"resourceID"`, `"documentID"`, 1), header: true, expected: 400},
		{name: "missing csrf header", method: "POST", body: good, expected: 403},
		{name: "method", method: "GET", body: good, expected: 405},
		{name: "trailing json", method: "POST", body: good + ` {}`, header: true, expected: 400},
		{name: "unknown action", method: "POST", body: strings.Replace(good, `"move"`, `"delete"`, 1), header: true, expected: 400},
		{name: "oversize", method: "POST", body: good + strings.Repeat(" ", 65536), header: true, expected: 400},
	} {
		t.Run(tt.name, func(t *testing.T) {
			store := &httpTestStore{err: tt.commitErr}
			host := Host{Bindings: []Binding{{ApplicationID: "sui", ActionID: "move", Store: store}}}
			if !tt.noAuth {
				host.Authorizer = permit(true)
			}
			handler := JSONCommitHandler{Host: host, DecodePayload: func(application, action string, raw json.RawMessage) (any, error) {
				if application != "sui" || action != "move" {
					return nil, errors.New("unknown")
				}
				return string(raw), nil
			}}
			if tt.noDecoder {
				handler.DecodePayload = nil
			}
			r := httptest.NewRequest(tt.method, "http://example.test/commit", bytes.NewBufferString(tt.body))
			r.Header.Set("Content-Type", "application/json")
			if tt.header {
				r.Header.Set("X-TEI-Action", "commit")
			}
			if tt.origin != "" {
				r.Header.Set("Origin", tt.origin)
			}
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, r)
			if w.Code != tt.expected || store.calls != tt.calls {
				t.Fatalf("status=%d want=%d calls=%d want=%d body=%s", w.Code, tt.expected, store.calls, tt.calls, w.Body.String())
			}
		})
	}
}

func TestJSONCommitHandlerSUIRoundTripAndConflict(t *testing.T) {
	store := NewScopedJSONDocumentStore("doc", "r1", []byte(documentFixture))
	host := Host{
		Authorizer: permit(true),
		Bindings:   []Binding{{ApplicationID: "sui", ActionID: "sui.move", Store: store}},
	}
	handler := JSONCommitHandler{
		Host: host,
		DecodePayload: func(application, action string, raw json.RawMessage) (any, error) {
			if application != "sui" || action != "sui.move" {
				return nil, ErrInvalid
			}
			var payload MovePayload
			if err := json.Unmarshal(raw, &payload); err != nil {
				return nil, err
			}
			return payload, nil
		},
	}
	requestBody := `{"protocolVersion":"1","applicationID":"sui","resourceID":"doc","actionID":"sui.move","expectedRevision":"r1","payload":{"cardId":"a","x":5,"y":6}}`
	send := func() *httptest.ResponseRecorder {
		r := httptest.NewRequest("POST", "http://example.test/commit", bytes.NewBufferString(requestBody))
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("X-TEI-Action", "commit")
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		return w
	}
	first := send()
	if first.Code != 200 {
		t.Fatalf("first commit returned %d: %s", first.Code, first.Body.String())
	}
	var result struct {
		Revision string `json:"revision"`
	}
	if err := json.Unmarshal(first.Body.Bytes(), &result); err != nil || result.Revision != "r1:1" {
		t.Fatalf("bad revision: %+v err=%v", result, err)
	}
	second := send()
	if second.Code != 409 {
		t.Fatalf("expected conflict, got %d: %s", second.Code, second.Body.String())
	}
	revision, document, writes := store.Snapshot()
	if revision != "r1:1" || writes != 1 {
		t.Fatalf("revision=%q writes=%d", revision, writes)
	}
	var updated map[string]json.RawMessage
	if err := json.Unmarshal(document, &updated); err != nil {
		t.Fatal(err)
	}
	var cards []map[string]json.RawMessage
	if err := json.Unmarshal(updated["cards"], &cards); err != nil {
		t.Fatal(err)
	}
	if string(cards[0]["x"]) != "5" || string(cards[0]["y"]) != "6" {
		t.Fatalf("card did not move: %s", cards[0])
	}
	if string(updated["edges"]) == "" || string(updated["reviewAttribution"]) == "" {
		t.Fatal("unrelated SUI fields lost")
	}
}
