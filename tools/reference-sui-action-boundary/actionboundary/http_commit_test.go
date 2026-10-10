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
	good := `{"applicationID":"sui","documentID":"doc","actionID":"move","expectedRevision":"r1","payload":{"x":1}}`
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
		{name: "unknown field", method: "POST", body: `{"applicationID":"sui","documentID":"doc","actionID":"move","expectedRevision":"r1","payload":{"x":1},"subject":"admin"}`, header: true, expected: 400},
		{name: "wrong origin", method: "POST", body: good, origin: "https://evil.example", header: true, expected: 403},
		{name: "same origin", method: "POST", body: good, origin: "http://example.test", header: true, expected: 200, calls: 1},
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
			handler := JSONCommitHandler{Host: host, DecodePayload: func(action string, raw json.RawMessage) (any, error) {
				if action != "move" {
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
