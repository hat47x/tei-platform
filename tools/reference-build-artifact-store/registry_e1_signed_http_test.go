package main

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// E1-C is an integration test only. Publisher P exposes the existing signed
// test-only fixture and the existing CAS bytes. Consumer C (a NEW Node OS
// process) must get and verify both subjects independently over loopback HTTP.
// Neither a real Trust issuer nor a new Registry HTTP API is introduced.
func registryE1SignedEnvelopeHandler(t *testing.T, store localArtifactStore, members []artifactMemberRequest, fault string) http.Handler {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "reference-plugin-registry", "fixture.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(members) != 2 {
		t.Fatal("the pinned reference requires exactly two members")
	}

	switch fault {
	case "bad-signature":
		raw = bytes.Replace(raw, []byte("publisherSignatureBase64\": \"uiwQ"), []byte("publisherSignatureBase64\": \"AiwQ"), 1)
	case "wrong-version":
		raw = bytes.Replace(raw, []byte("version=1.0.0"), []byte("version=2.0.0"), 1)
	case "wrong-target":
		raw = bytes.Replace(raw, []byte("\"target\": \"linux-amd64\""), []byte("\"target\": \"wasm32-wasi\""), 1)
	case "duplicate-key":
		raw = bytes.Replace(raw, []byte("\"releaseText\": "), []byte("\"releaseText\": \"poison\", \"releaseText\": "), 1)
	case "oversize-envelope":
		raw = bytes.Repeat([]byte("x"), registryE1HTTPMaxBytes+1)
	}
	artifactFault := ""
	if fault == "mixed-member" {
		artifactFault = "mixed-generation"
	}
	artifactHandler := registryE1LoopbackHandler(store, artifactFault, members[0].Digest, members[1].Digest)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/reference/signed-release.json" {
			if r.Method != http.MethodGet || r.URL.RawQuery != "" {
				http.NotFound(w, r)
				return
			}
			switch fault {
			case "redirect-envelope":
				http.Redirect(w, r, "https://example.invalid/unknown-release", http.StatusFound)
				return
			case "missing-envelope":
				http.NotFound(w, r)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Cache-Control", "no-store")
			_, _ = w.Write(raw)
			return
		}
		artifactHandler.ServeHTTP(w, r)
	})
}

func registryE1RunSignedHTTPNodeConsumer(t *testing.T, origin, expectedStatus string) {
	t.Helper()
	script := filepath.Join("..", "reference-plugin-registry", "http_consumer.mjs")
	child := exec.Command("node", script, origin)
	output, err := child.CombinedOutput()
	if !bytes.Contains(output, []byte("E1-SIGNED-C status="+expectedStatus+"\n")) {
		t.Fatalf("expected %s, got child output %s (err=%v)", expectedStatus, output, err)
	}
	if (err == nil) != (expectedStatus == "complete") {
		t.Fatalf("Node Consumer exit status does not match %s: %s (err=%v)", expectedStatus, output, err)
	}
	if expectedStatus == "complete" {
		if !bytes.Contains(output, []byte("count=2")) {
			t.Fatal("Node Consumer did not GET both signed members")
		}
		if bytes.Contains(output, []byte(fmt.Sprintf("pid=%d ", os.Getpid()))) {
			t.Fatal("Node Consumer did not run in a separate OS process")
		}
	}
}

func TestRegistryE1SignedHTTPNodeConsumer(t *testing.T) {
	cases := []struct {
		name       string
		fault      string
		expect     string
		deleteRole bool
	}{
		{name: "valid signed release and both actual HTTP members", expect: "complete"},
		{name: "invalid publisher signature", fault: "bad-signature", expect: "invalid-signed-subject"},
		{name: "changed release version", fault: "wrong-version", expect: "invalid-signed-subject"},
		{name: "wrong verifier target", fault: "wrong-target", expect: "invalid-signed-subject"},
		{name: "ambiguous duplicate raw JSON key", fault: "duplicate-key", expect: "invalid-wire"},
		{name: "signed envelope redirect blocked", fault: "redirect-envelope", expect: "redirect-blocked"},
		{name: "oversize signed envelope blocked", fault: "oversize-envelope", expect: "too-large"},
		{name: "missing signed envelope blocked", fault: "missing-envelope", expect: "missing"},
		{name: "signed R1 with a different member response", fault: "mixed-member", expect: "digest-mismatch"},
		{name: "member missing after publisher readback", expect: "missing", deleteRole: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			store, members := registryE1PublishedStore(t)
			if tc.deleteRole {
				path, err := store.pathForDigest(members[1].Digest)
				if err != nil {
					t.Fatal(err)
				}
				if err := os.Remove(path); err != nil {
					t.Fatal(err)
				}
			}
			server := httptest.NewServer(registryE1SignedEnvelopeHandler(t, store, members, tc.fault))
			defer server.Close()
			if !strings.HasPrefix(server.URL, "http://127.0.0.1:") {
				t.Fatalf("E1-C test server must be literal loopback, got %s", server.URL)
			}
			registryE1RunSignedHTTPNodeConsumer(t, server.URL, tc.expect)
		})
	}
}
