//go:build tei_api_only && tei_order_only && !tei_repair_only

package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestDiagnosticExecutableObservationMatchesRunningLinuxImage(t *testing.T) {
	f, err := os.Open("/proc/self/exe")
	if err != nil {
		t.Skipf("Linux process image is unavailable: %v", err)
	}
	defer f.Close()
	hash := sha256.New()
	size, err := io.Copy(hash, f)
	if err != nil {
		t.Fatal(err)
	}
	server := &webTargetServer{}
	r := httptest.NewRecorder()
	server.Handler().ServeHTTP(r, httptest.NewRequest(http.MethodGet, "/diagnostic-executable.json", nil))
	if r.Code != http.StatusOK {
		t.Fatalf("live process image status: %d: %s", r.Code, r.Body.String())
	}
	var got struct {
		SHA256 string `json:"sha256"`
		Bytes  int64  `json:"bytes"`
	}
	if err := json.Unmarshal(r.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got.SHA256 != hex.EncodeToString(hash.Sum(nil)) || got.Bytes != size {
		t.Fatalf("observation differs from actual process image: %+v", got)
	}
	if r.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("diagnostic process image response may not be cached")
	}
	post := httptest.NewRecorder()
	server.Handler().ServeHTTP(post, httptest.NewRequest(http.MethodPost, "/diagnostic-executable.json", nil))
	if post.Code != http.StatusMethodNotAllowed {
		t.Fatalf("POST diagnostic observation got %d, want 405", post.Code)
	}
}
