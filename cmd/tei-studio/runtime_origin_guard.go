package main

import (
	"net/http"
	"strconv"

	"github.com/wailsapp/wails/v3/pkg/application"
)

const studioRuntimePath = "/wails/runtime"

// studioRuntimeOriginMiddleware protects only Wails' HTTP RPC dispatch route.
// Static assets remain available without Origin so the initial page can load.
func studioRuntimeOriginMiddleware(port int) application.Middleware {
	expectedOrigin, expectedHost := studioRuntimeAuthority(port)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL == nil || r.URL.Path != studioRuntimePath {
				next.ServeHTTP(w, r)
				return
			}
			origins := r.Header.Values("Origin")
			if r.TLS != nil || expectedOrigin == "" || expectedHost == "" ||
				r.Host != expectedHost || len(origins) != 1 || origins[0] != expectedOrigin {
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// studioRuntimeAuthority returns the browser-serialized HTTP Origin and Host.
// Browsers omit the default :80 port from both values.
func studioRuntimeAuthority(port int) (origin, host string) {
	if port < 1 || port > 65535 {
		return "", ""
	}
	if port == 80 {
		return "http://127.0.0.1", "127.0.0.1"
	}
	portText := strconv.Itoa(port)
	return "http://127.0.0.1:" + portText, "127.0.0.1:" + portText
}
