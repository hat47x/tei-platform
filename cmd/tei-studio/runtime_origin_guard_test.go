//go:build server

package main

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/wailsapp/wails/v3/pkg/application"
)

func TestStudioRuntimeOriginMiddlewareAllowsSameOriginBridgeCall(t *testing.T) {
	const port = 43123
	bridge := newRuntimeGuardTestBridge(t)
	calls := 0
	handler := newRuntimeGuardTestHandler(port, bridge, &calls)
	origin, host := studioRuntimeAuthority(port)

	req := httptest.NewRequest(http.MethodPost, studioRuntimePath+"?call=1", strings.NewReader("{}"))
	req.Host = host
	req.Header.Set("Origin", origin)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if calls != 1 {
		t.Fatalf("Bridge calls = %d, want 1", calls)
	}
}

func TestStudioRuntimeOriginMiddlewareRejectsBeforeBridgeCall(t *testing.T) {
	const port = 43123
	bridge := newRuntimeGuardTestBridge(t)
	origin, host := studioRuntimeAuthority(port)

	tests := []struct {
		name    string
		method  string
		origins []string
		host    string
		tls     bool
		headers http.Header
	}{
		{name: "missing origin", method: http.MethodPost, host: host},
		{name: "null origin", method: http.MethodPost, origins: []string{"null"}, host: host},
		{name: "wrong scheme", method: http.MethodPost, origins: []string{"https://127.0.0.1:43123"}, host: host},
		{name: "wrong origin host", method: http.MethodPost, origins: []string{"http://attacker.example:43123"}, host: host},
		{name: "dns rebinding authority", method: http.MethodPost, origins: []string{"http://attacker.example:43123"}, host: "attacker.example:43123"},
		{name: "wrong origin port", method: http.MethodPost, origins: []string{"http://127.0.0.1:43124"}, host: host},
		{name: "origin path", method: http.MethodPost, origins: []string{origin + "/path"}, host: host},
		{name: "origin query", method: http.MethodPost, origins: []string{origin + "?x=1"}, host: host},
		{name: "origin user info", method: http.MethodPost, origins: []string{"http://user@127.0.0.1:43123"}, host: host},
		{name: "malformed origin", method: http.MethodPost, origins: []string{"http://127.0.0.1:43123, http://attacker.example"}, host: host},
		{name: "duplicate origins", method: http.MethodPost, origins: []string{origin, origin}, host: host},
		{name: "origin hostname alias", method: http.MethodPost, origins: []string{"http://localhost:43123"}, host: host},
		{name: "origin alternate ipv4", method: http.MethodPost, origins: []string{"http://127.1:43123"}, host: host},
		{name: "noncanonical origin port", method: http.MethodPost, origins: []string{"http://127.0.0.1:043123"}, host: host},
		{name: "missing host", method: http.MethodPost, origins: []string{origin}},
		{name: "wrong host", method: http.MethodPost, origins: []string{origin}, host: "attacker.example:43123"},
		{name: "host alias", method: http.MethodPost, origins: []string{origin}, host: "localhost:43123"},
		{name: "host port mismatch", method: http.MethodPost, origins: []string{origin}, host: "127.0.0.1:43124"},
		{name: "malformed host port", method: http.MethodPost, origins: []string{origin}, host: "127.0.0.1:bad"},
		{name: "https transport", method: http.MethodPost, origins: []string{origin}, host: host, tls: true},
		{name: "wrong origin preflight", method: http.MethodOptions, origins: []string{"http://attacker.example:43123"}, host: host},
		{
			name:    "wrong origin simple form post",
			method:  http.MethodPost,
			origins: []string{"http://attacker.example:43123"},
			host:    host,
			headers: http.Header{"Content-Type": []string{"application/x-www-form-urlencoded"}},
		},
		{
			name:    "wrong origin chunk post",
			method:  http.MethodPost,
			origins: []string{"http://attacker.example:43123"},
			host:    host,
			headers: http.Header{
				"X-Wails-Chunk-Id":    []string{"chunk-1"},
				"X-Wails-Chunk-Index": []string{"0"},
				"X-Wails-Chunk-Total": []string{"1"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			calls := 0
			handler := newRuntimeGuardTestHandler(port, bridge, &calls)
			req := httptest.NewRequest(tt.method, studioRuntimePath, strings.NewReader("{}"))
			req.Host = tt.host
			for _, value := range tt.origins {
				req.Header.Add("Origin", value)
			}
			for name, values := range tt.headers {
				for _, value := range values {
					req.Header.Add(name, value)
				}
			}
			if tt.tls {
				req.TLS = &tls.ConnectionState{}
			}
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, req)

			if recorder.Code != http.StatusForbidden {
				t.Fatalf("status = %d, want %d; body: %s", recorder.Code, http.StatusForbidden, recorder.Body.String())
			}
			if calls != 0 {
				t.Fatalf("Bridge calls = %d, want 0", calls)
			}
		})
	}
}

func TestStudioRuntimeOriginMiddlewareAllowsBrowserSerializedPort80(t *testing.T) {
	bridge := newRuntimeGuardTestBridge(t)
	calls := 0
	handler := newRuntimeGuardTestHandler(80, bridge, &calls)

	req := httptest.NewRequest(http.MethodPost, studioRuntimePath, strings.NewReader("{}"))
	req.Host = "127.0.0.1"
	req.Header.Set("Origin", "http://127.0.0.1")
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body: %s", recorder.Code, http.StatusOK, recorder.Body.String())
	}
	if calls != 1 {
		t.Fatalf("Bridge calls = %d, want 1", calls)
	}
}

func TestStudioRuntimeOriginMiddlewareLeavesOriginlessAssetsAvailable(t *testing.T) {
	bridge := newRuntimeGuardTestBridge(t)
	calls := 0
	handler := newRuntimeGuardTestHandler(43123, bridge, &calls)

	for _, path := range []string{"/", "/wails/runtime.js"} {
		t.Run(path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, path, nil)
			req.Host = "127.0.0.1:43123"
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, req)
			if recorder.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
			}
		})
	}
	if calls != 0 {
		t.Fatalf("Bridge calls = %d, want 0 for static assets", calls)
	}
}

func TestRequireWailsServerPortUnset(t *testing.T) {
	tests := []struct {
		name    string
		lookup  func(string) (string, bool)
		wantErr bool
	}{
		{name: "unset", lookup: func(string) (string, bool) { return "", false }},
		{name: "empty but present", lookup: func(string) (string, bool) { return "", true }, wantErr: true},
		{name: "same value", lookup: func(string) (string, bool) { return "43123", true }, wantErr: true},
		{name: "different value", lookup: func(string) (string, bool) { return "43124", true }, wantErr: true},
		{name: "invalid value", lookup: func(string) (string, bool) { return "invalid", true }, wantErr: true},
		{name: "missing environment lookup", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := requireWailsServerPortUnset(tt.lookup)
			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && !strings.Contains(err.Error(), "WAILS_SERVER_PORT") {
				t.Fatalf("error = %q, want WAILS_SERVER_PORT context", err)
			}
		})
	}
}

func TestValidateBuildConfigurationRejectsPresentWailsServerPort(t *testing.T) {
	t.Setenv("WAILS_SERVER_PORT", "")
	if err := validateBuildConfiguration(); err == nil || !strings.Contains(err.Error(), "WAILS_SERVER_PORT") {
		t.Fatalf("validateBuildConfiguration() = %v, want WAILS_SERVER_PORT error", err)
	}
}

func TestStudioRuntimeAuthorityRejectsInvalidPorts(t *testing.T) {
	for _, port := range []int{-1, 0, 65536} {
		if origin, host := studioRuntimeAuthority(port); origin != "" || host != "" {
			t.Errorf("studioRuntimeAuthority(%d) = (%q, %q), want empty authority", port, origin, host)
		}
	}
}

func newRuntimeGuardTestBridge(t *testing.T) *StudioBridge {
	t.Helper()
	root := filepath.Join("..", "..", "reference", "workspace")
	host, err := newStudioHost([]string{root})
	if err != nil {
		t.Fatalf("newStudioHost(%q): %v", root, err)
	}
	return &StudioBridge{host: host}
}

func newRuntimeGuardTestHandler(port int, bridge *StudioBridge, bridgeCalls *int) http.Handler {
	transport := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == studioRuntimePath {
			*bridgeCalls = *bridgeCalls + 1
			if response := bridge.ListWorkspaces(); !response.OK {
				http.Error(w, "Bridge operation failed", http.StatusInternalServerError)
				return
			}
		}
		w.WriteHeader(http.StatusOK)
	})
	assets := application.AssetOptions{Middleware: studioAssetMiddleware(port)}
	return assets.Middleware(transport)
}
