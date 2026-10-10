package runtimesnapshot

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const validManifest = `{"applicationId":"order_entry","handoverBaselineId":"B","baselineRepositoryRevision":"R1","targetBuildRevision":"T1","targetAdapter":"reference-first-web-target","unresolved":["not-production"],"patterns":[{"id":"order.list"}]}`

func fixture(raw string) Identity {
	digest := sha256.Sum256([]byte(raw))
	return Identity{
		ApplicationID: "order_entry", HandoverBaselineID: "B",
		BaselineRepositoryRevision: "R1", TargetBuildRevision: "T1",
		ManifestSHA256: hex.EncodeToString(digest[:]),
	}
}

func TestAcceptPinnedProjectionWithoutReauthoringMeaning(t *testing.T) {
	raw := []byte(validManifest)
	original := string(raw)
	s, err := Accept(raw, fixture(original))
	if err != nil {
		t.Fatal(err)
	}
	for i := range raw {
		raw[i] = 'x'
	}
	req := httptest.NewRequest(http.MethodGet, "/manifest.json", nil)
	res := httptest.NewRecorder()
	s.Handler().ServeHTTP(res, req)
	if res.Code != http.StatusOK || res.Body.String() != original {
		t.Fatalf("manifest bytes were not preserved: %d %q", res.Code, res.Body.String())
	}
	if res.Header().Get("Cache-Control") != "no-store" || res.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("unsafe manifest headers: %v", res.Header())
	}
	if !strings.Contains(res.Body.String(), `"unresolved":["not-production"]`) {
		t.Fatal("unresolved status was lost")
	}
}

func TestAcceptRejectsInconsistentProjection(t *testing.T) {
	scenarios := []struct {
		name, raw string
		identity  func(*Identity)
	}{
		{"incorrect-app", validManifest, func(v *Identity) { v.ApplicationID = "repair_intake" }},
		{"incorrect-baseline", validManifest, func(v *Identity) { v.HandoverBaselineID = "B2" }},
		{"incorrect-revision", validManifest, func(v *Identity) { v.BaselineRepositoryRevision = "R2" }},
		{"incorrect-build", validManifest, func(v *Identity) { v.TargetBuildRevision = "T2" }},
		{"incorrect-digest", validManifest, func(v *Identity) { v.ManifestSHA256 = strings.Repeat("0", 64) }},
		{"empty-expected-app", validManifest, func(v *Identity) { v.ApplicationID = "" }},
		{"invalid-digest", validManifest, func(v *Identity) { v.ManifestSHA256 = "invalid" }},
		{"duplicate-app", `{"applicationId":"evil","applicationId":"order_entry","handoverBaselineId":"B","baselineRepositoryRevision":"R1","targetBuildRevision":"T1"}`, nil},
		{"missing-app", `{"handoverBaselineId":"B","baselineRepositoryRevision":"R1","targetBuildRevision":"T1"}`, nil},
		{"trailing-json", validManifest + " {}", nil},
		{"unknown-top-level", `{"applicationId":"order_entry","handoverBaselineId":"B","baselineRepositoryRevision":"R1","targetBuildRevision":"T1","customRef":true}`, nil},
	}
	for _, tc := range scenarios {
		t.Run(tc.name, func(t *testing.T) {
			id := fixture(tc.raw)
			if tc.identity != nil {
				tc.identity(&id)
			}
			_, err := Accept([]byte(tc.raw), id)
			if tc.name == "unknown-top-level" {
				if err != nil {
					t.Fatalf("opaque derived manifest extension rejected: %v", err)
				}
			} else if err == nil {
				t.Fatal("invalid projected manifest was accepted")
			}
		})
	}
}

// A pinned digest authenticates bytes, not the interpretation of ambiguous
// nested JSON objects by every downstream consumer.
func TestAcceptRejectsNestedDuplicateKeysAndDeepInputs(t *testing.T) {
	bad := []string{
		`{ "applicationId":"order_entry", "handoverBaselineId":"B", "baselineRepositoryRevision":"R1", "targetBuildRevision":"T1", "patterns":[{"id":"first", "id":"second"}] }`,
		`{ "applicationId":"order_entry", "handoverBaselineId":"B", "baselineRepositoryRevision":"R1", "targetBuildRevision":"T1", "patterns":[{"nested":{"x":1,"x":2}}] }`,
	}
	bad = append(bad, validManifest[:len(validManifest)-1]+`,"nested":`+strings.Repeat("[", 129)+`null`+strings.Repeat("]", 129)+`}`)
	for _, raw := range bad {
		if _, err := Accept([]byte(raw), fixture(raw)); err == nil {
			t.Fatalf("ambiguous or excessively nested JSON accepted: %s", raw[:min(len(raw), 160)])
		}
	}
	good := validManifest[:len(validManifest)-1] + `,"patternsV2":[{"id":"a"},{"id":"b"}],"metadata":{"x":1,"y":2}}`
	if _, err := Accept([]byte(good), fixture(good)); err != nil {
		t.Fatalf("valid extensible nested manifest rejected: %v", err)
	}
}

func TestHandlerDoesNotExposeBusinessRoutes(t *testing.T) {
	s, err := Accept([]byte(validManifest), fixture(validManifest))
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		method, path string
		want         int
	}{
		{http.MethodGet, "/healthz", http.StatusNoContent},
		{http.MethodPost, "/healthz", http.StatusMethodNotAllowed},
		{http.MethodPost, "/manifest.json", http.StatusMethodNotAllowed},
		{http.MethodGet, "/scenario/read", http.StatusNotFound},
		{http.MethodPost, "/scenario/submit", http.StatusNotFound},
		{http.MethodGet, "/scenario/customer-lookup", http.StatusNotFound},
		{http.MethodGet, "/", http.StatusNotFound},
	} {
		res := httptest.NewRecorder()
		s.Handler().ServeHTTP(res, httptest.NewRequest(tc.method, tc.path, nil))
		if res.Code != tc.want {
			t.Fatalf("%s %s: got %d, want %d", tc.method, tc.path, res.Code, tc.want)
		}
	}
}

func TestZeroSnapshotFailsClosed(t *testing.T) {
	res := httptest.NewRecorder()
	var snapshot Snapshot
	snapshot.Handler().ServeHTTP(res, httptest.NewRequest(http.MethodGet, "/manifest.json", nil))
	if res.Code != http.StatusServiceUnavailable {
		t.Fatalf("unaccepted snapshot got %d, want unavailable", res.Code)
	}
}
