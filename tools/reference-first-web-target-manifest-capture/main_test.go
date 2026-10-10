package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const webManifest = `{"handoverBaselineId":"B","baselineRepositoryRevision":"R","targetBuildRevision":"T","targetAdapter":"reference-first-web-target","applicationId":"order_entry","patterns":[],"unresolved":["not-accepted"]}`

func captureArguments(baseURL string) []string {
	return []string{
		"-url", baseURL + "/manifest.json",
		"-application", "order_entry", "-baseline-id", "B",
		"-baseline-revision", "R", "-target-revision", "T",
	}
}

func TestCaptureFirstWebTargetBytesWithoutApprovingProvenance(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/manifest.json" || r.Method != http.MethodGet {
			t.Errorf("unexpected request %s %s", r.Method, r.URL)
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		fmt.Fprint(w, webManifest)
	}))
	defer server.Close()

	var stdout, stderr bytes.Buffer
	if err := runCapture(captureArguments(server.URL), &stdout, &stderr); err != nil {
		t.Fatal(err)
	}
	if stdout.String() != webManifest {
		t.Fatalf("source bytes were rewritten: %q", stdout.String())
	}
	sum := sha256.Sum256([]byte(webManifest))
	want := hex.EncodeToString(sum[:])
	if !strings.Contains(stderr.String(), want) || !strings.Contains(stderr.String(), "not an approved") {
		t.Fatalf("observed hash was incorrectly labeled: %q", stderr.String())
	}
}

func TestCaptureRejectsOffLoopbackOrNonLiteralURL(t *testing.T) {
	for _, address := range []string{
		"", "http://localhost:8080/manifest.json",
		"http://127.0.0.2:8080/manifest.json",
		"https://127.0.0.1:8080/manifest.json",
		"http://127.0.0.1:8080/manifest.json?debug=1",
		"http://user@127.0.0.1:8080/manifest.json",
		"http://127.0.0.1:8080/other",
		"http://127.0.0.1:0/manifest.json",
		"http://127.0.0.1:99999/manifest.json",
		"http://127.0.0.1:08080/manifest.json",
		"http://127.0.0.1:8080/%6danifest.json",
		"http://127.0.0.1:8080/manifest.json#fragment",
	} {
		if err := validateCaptureURL(address); err == nil {
			t.Errorf("invalid URL accepted: %q", address)
		}
	}
	for _, args := range [][]string{
		nil, {"-url", "http://127.0.0.1:8080/manifest.json"},
		append(captureArguments("http://127.0.0.1:8080"), "unexpected"),
	} {
		if _, err := parseCaptureOptions(args); err == nil {
			t.Errorf("incomplete or ambiguous CLI invocation accepted: %q", args)
		}
	}
}

func TestCaptureFailsBeforePrintingBytesOnWrongIdentityOrProtocol(t *testing.T) {
	for _, tc := range []struct {
		name, status, contentType, body string
		code                          int
	}{
		{name: "wrong-app", code: 200, contentType: "application/json", body: strings.Replace(webManifest, "order_entry", "repair_intake", 1)},
		{name: "wrong-build", code: 200, contentType: "application/json", body: strings.Replace(webManifest, `"T"`, `"T2"`, 1)},
		{name: "duplicate-root", code: 200, contentType: "application/json", body: `{"applicationId":"repair_intake",` + webManifest[1:]},
		{name: "malformed-json", code: 200, contentType: "application/json", body: "{broken"},
		{name: "html", code: 200, contentType: "text/html", body: webManifest},
		{name: "redirect", code: 302, contentType: "text/html", body: "redirect"},
		{name: "not-found", code: 404, contentType: "text/plain", body: "not found"},
		{name: "oversized", code: 200, contentType: "application/json", body: strings.Repeat("a", maxCaptureBytes+1)},
		{name: "empty", code: 200, contentType: "application/json", body: ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", tc.contentType)
				w.WriteHeader(tc.code)
				fmt.Fprint(w, tc.body)
			}))
			defer srv.Close()
			var stdout, stderr bytes.Buffer
			if err := runCapture(captureArguments(srv.URL), &stdout, &stderr); err == nil {
				t.Fatal("invalid manifest response was accepted")
			}
			if stdout.Len() != 0 || stderr.Len() != 0 {
				t.Fatalf("failed capture emitted unverified output: stdout=%d stderr=%d", stdout.Len(), stderr.Len())
			}
		})
	}
}

func TestCaptureRejectsRedirectToExternalHost(t *testing.T) {
	remoteCalled := false
	remote := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { remoteCalled = true }))
	defer remote.Close()
	local := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, remote.URL+"/manifest.json", http.StatusFound)
	}))
	defer local.Close()
	opts, err := parseCaptureOptions(captureArguments(local.URL))
	if err != nil {
		t.Fatal(err)
	}
	client := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
	if _, _, err := capture(context.Background(), client, opts); err == nil {
		t.Fatal("redirect was accepted")
	}
	if remoteCalled {
		t.Fatal("off-origin URL contacted")
	}
}
