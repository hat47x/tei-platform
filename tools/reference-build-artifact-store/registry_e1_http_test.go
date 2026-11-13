package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"
)

const registryE1HTTPMaxBytes = 1 << 20

// This HTTP distribution surface is an in-process loopback fixture around the
// EXISTING localArtifactStore. It is not a Registry service or a GitHub Release.
func registryE1LoopbackHandler(store localArtifactStore, fault, coreDigest, uiDigest string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.RawQuery != "" ||
			!strings.HasPrefix(r.URL.Path, "/sha256/") {
			http.Error(w, "unsupported request", http.StatusNotFound)
			return
		}
		hexDigest := strings.TrimPrefix(r.URL.Path, "/sha256/")
		digest := "sha256:" + hexDigest
		normalized, err := normalizeDigest(digest)
		if err != nil || normalized != digest {
			http.Error(w, "invalid digest locator", http.StatusNotFound)
			return
		}
		if fault == "redirect" && digest == coreDigest {
			// An untrusted distribution service may send a Location to another
			// host; Consumer C must not silently change retrieval authority.
			http.Redirect(w, r, "https://example.invalid/foreign", http.StatusFound)
			return
		}
		if fault == "oversize" && digest == coreDigest {
			w.Header().Set("Content-Type", "application/octet-stream")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(bytes.Repeat([]byte("x"), registryE1HTTPMaxBytes+1))
			return
		}
		if fault == "mixed-generation" && digest == coreDigest {
			// The path still advertises R1 core's digest, but its body is R2/
			// another role's valid bytes. Content digest must reject it.
			value, _, err := store.GetBytes(uiDigest)
			if err != nil {
				http.Error(w, "member unavailable", http.StatusServiceUnavailable)
				return
			}
			w.Header().Set("Content-Type", "application/octet-stream")
			_, _ = w.Write(value)
			return
		}
		value, _, err := store.GetBytes(digest)
		if err != nil {
			if errors.Is(err, errArtifactNotFound) {
				http.NotFound(w, r)
				return
			}
			// A remote 503 cannot tell the Consumer whether the backend
			// object was corrupt or the source was unavailable.
			http.Error(w, "member unavailable", http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("ETag", fmt.Sprintf("%q", digest))
		_, _ = w.Write(value)
	})
}

func registryE1HTTPFail(t *testing.T, status string) {
	t.Helper()
	fmt.Printf("E1-HTTP-C status=%s\n", status)
	t.Fatalf("HTTP Consumer C refused: %s", status)
}

// Consumer C is a distinct OS process and the request uses a real HTTP
// transport. It does not reuse Publisher P's in-memory bytes, Store handle,
// receipts or HTTP handler implementation. The same test-only pinned fixture
// is read again in C; this is not a real issuer/trust/profile lookup.
func TestRegistryE1HTTPConsumerChild(t *testing.T) {
	if os.Getenv("TEI_REGISTRY_E1_HTTP_CHILD") != "1" {
		return
	}
	rawOrigin := os.Getenv("TEI_REGISTRY_E1_HTTP_ORIGIN")
	origin, err := url.Parse(rawOrigin)
	if err != nil || origin.Scheme != "http" || origin.User != nil ||
		origin.RawQuery != "" || origin.Fragment != "" || origin.Path != "" {
		t.Fatal("test-only HTTP origin must be an exact loopback base URL")
	}
	loopbackIP := net.ParseIP(origin.Hostname())
	if loopbackIP == nil || !loopbackIP.IsLoopback() || origin.Port() == "" {
		t.Fatal("test-only HTTP Consumer refuses non-loopback origins")
	}
	requests, expectedBytes := registryE1Expected(t)
	if os.Getenv("TEI_REGISTRY_E1_HTTP_MUTATION") == "swap-role" {
		requests[0].Digest, requests[1].Digest = requests[1].Digest, requests[0].Digest
	}
	client := &http.Client{
		Timeout: 5 * time.Second,
		Transport: &http.Transport{
			Proxy: nil, // no environment proxy or accidental egress
		},
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	for _, member := range requests {
		digest, err := normalizeDigest(member.Digest)
		if err != nil {
			registryE1HTTPFail(t, "invalid-digest")
		}
		address := rawOrigin + "/sha256/" + strings.TrimPrefix(digest, "sha256:")
		response, err := client.Get(address)
		if err != nil {
			registryE1HTTPFail(t, "unavailable")
		}
		if response.StatusCode >= 300 && response.StatusCode < 400 {
			response.Body.Close()
			registryE1HTTPFail(t, "redirect-blocked")
		}
		if response.StatusCode == http.StatusNotFound {
			response.Body.Close()
			registryE1HTTPFail(t, "missing")
		}
		if response.StatusCode != http.StatusOK {
			response.Body.Close()
			registryE1HTTPFail(t, "unavailable")
		}
		value, readErr := io.ReadAll(io.LimitReader(response.Body, registryE1HTTPMaxBytes+1))
		closeErr := response.Body.Close()
		if readErr != nil || closeErr != nil {
			registryE1HTTPFail(t, "read-failed")
		}
		if len(value) > registryE1HTTPMaxBytes {
			registryE1HTTPFail(t, "too-large")
		}
		if digestBytes(value) != digest {
			registryE1HTTPFail(t, "digest-mismatch")
		}
		if !bytes.Equal(value, expectedBytes[member.Role]) {
			registryE1HTTPFail(t, "role-mismatch")
		}
	}
	fmt.Printf("E1-HTTP-C status=complete pid=%d count=%d subject=%s\n",
		os.Getpid(), len(requests), registryE1PinnedReleaseDigest)
}

func registryE1RunHTTPConsumer(t *testing.T, origin, mutation, expectedStatus string) {
	t.Helper()
	child := exec.Command(os.Args[0], "-test.run=^TestRegistryE1HTTPConsumerChild$", "-test.v")
	child.Env = append(os.Environ(),
		"TEI_REGISTRY_E1_HTTP_CHILD=1",
		"TEI_REGISTRY_E1_HTTP_ORIGIN="+origin,
		"TEI_REGISTRY_E1_HTTP_MUTATION="+mutation)
	output, err := child.CombinedOutput()
	if !bytes.Contains(output, []byte("E1-HTTP-C status="+expectedStatus)) {
		t.Fatalf("expected %s, actual child: %s (error=%v)", expectedStatus, output, err)
	}
	if (err == nil) != (expectedStatus == "complete") {
		t.Fatalf("child exit status inconsistent with %s: %s (error=%v)", expectedStatus, output, err)
	}
	if expectedStatus == "complete" {
		if bytes.Contains(output, []byte(fmt.Sprintf("pid=%d ", os.Getpid()))) {
			t.Fatal("Publisher and Consumer were not separate processes")
		}
		if !bytes.Contains(output, []byte("count=2")) {
			t.Fatal("Consumer did not retrieve the entire pinned member set")
		}
	}
}

func TestRegistryE1HTTPTransportToSeparateProcess(t *testing.T) {
	t.Run("HTTP complete member set", func(t *testing.T) {
		store, _ := registryE1PublishedStore(t)
		server := httptest.NewServer(registryE1LoopbackHandler(store, "", "", ""))
		defer server.Close()
		registryE1RunHTTPConsumer(t, server.URL, "", "complete")
	})
	t.Run("missing member after Publisher readback", func(t *testing.T) {
		store, members := registryE1PublishedStore(t)
		path, err := store.pathForDigest(members[1].Digest)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.Remove(path); err != nil {
			t.Fatal(err)
		}
		server := httptest.NewServer(registryE1LoopbackHandler(store, "", "", ""))
		defer server.Close()
		registryE1RunHTTPConsumer(t, server.URL, "", "missing")
	})
	t.Run("remote backend corrupt becomes unavailable rather than accepted", func(t *testing.T) {
		store, members := registryE1PublishedStore(t)
		path, err := store.pathForDigest(members[1].Digest)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.Chmod(path, 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("corrupted-after-publication"), 0o600); err != nil {
			t.Fatal(err)
		}
		server := httptest.NewServer(registryE1LoopbackHandler(store, "", "", ""))
		defer server.Close()
		registryE1RunHTTPConsumer(t, server.URL, "", "unavailable")
	})
	t.Run("R1 locator serving another generation is rejected", func(t *testing.T) {
		store, members := registryE1PublishedStore(t)
		server := httptest.NewServer(registryE1LoopbackHandler(store, "mixed-generation", members[0].Digest, members[1].Digest))
		defer server.Close()
		registryE1RunHTTPConsumer(t, server.URL, "", "digest-mismatch")
	})
	t.Run("redirect does not change retrieval origin implicitly", func(t *testing.T) {
		store, members := registryE1PublishedStore(t)
		server := httptest.NewServer(registryE1LoopbackHandler(store, "redirect", members[0].Digest, ""))
		defer server.Close()
		registryE1RunHTTPConsumer(t, server.URL, "", "redirect-blocked")
	})
	t.Run("bounded read rejects oversized HTTP body", func(t *testing.T) {
		store, members := registryE1PublishedStore(t)
		server := httptest.NewServer(registryE1LoopbackHandler(store, "oversize", members[0].Digest, ""))
		defer server.Close()
		registryE1RunHTTPConsumer(t, server.URL, "", "too-large")
	})
	t.Run("HTTP valid bytes with swapped Role bindings are rejected", func(t *testing.T) {
		store, _ := registryE1PublishedStore(t)
		server := httptest.NewServer(registryE1LoopbackHandler(store, "", "", ""))
		defer server.Close()
		registryE1RunHTTPConsumer(t, server.URL, "swap-role", "role-mismatch")
	})
}
