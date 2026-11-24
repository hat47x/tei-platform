package main

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

// `-data-design-candidate-kind` is a closed value the catalogue publishes. A
// value outside that set used to be answered with KNOWLEDGE_BUILD_FAILED, which
// names a problem with the Workspace or the build: a caller would look at their
// model instead of at the flag they typed. A flag the command does not have is
// the same failure class as a missing flag (#5797).
//
// The expected set below is read from the catalogue, not copied, so an
// advertised kind and an accepted kind cannot drift apart.
func TestCandidateReviewRefusesAnUnpublishedKindAsAnArgument(t *testing.T) {
	reference := filepath.Join(testRepositoryRoot(t), "reference", "workspace")
	for _, kind := range []string{"bogus", "Business-Identity", "functional_dependency", "business-identity "} {
		t.Run(kind, func(t *testing.T) {
			var out bytes.Buffer
			code, _ := knowledgeDataDesignCandidateReviewJSON(&out, reference,
				"semantic-data", "sales.order", kind, "sha256:00", "", "", "", 0)
			body := out.String()
			if code != 2 {
				t.Fatalf("exit=%d, want 2 for an argument failure: %s", code, body)
			}
			if !strings.Contains(body, "INVALID_ARGUMENT") {
				t.Fatalf("the failure is not an argument failure: %s", body)
			}
			// The message names the value and the set, so the caller learns
			// what to change without reading the source.
			if !strings.Contains(body, kind) {
				t.Errorf("the refusal does not name %q: %s", kind, body)
			}
			for _, published := range candidateKinds {
				if !strings.Contains(body, published) {
					t.Errorf("the refusal does not offer %q: %s", published, body)
				}
			}
		})
	}
}

// Every published kind still reaches the builder, so the guard narrows what is
// rejected and does not change what the command does with a real value.
func TestPublishedCandidateKindsStillReachTheBuilder(t *testing.T) {
	reference := filepath.Join(testRepositoryRoot(t), "reference", "workspace")
	if len(candidateKinds) == 0 {
		t.Fatal("no candidate kind is published")
	}
	for _, kind := range candidateKinds {
		t.Run(kind, func(t *testing.T) {
			var out bytes.Buffer
			code, _ := knowledgeDataDesignCandidateReviewJSON(&out, reference,
				"semantic-data", "sales.order", kind, "sha256:00", "", "", "", 0)
			body := out.String()
			// The fingerprint is deliberately wrong, so the run stops at the
			// builder rather than at the flag. What matters here is that the
			// kind itself was not refused as an argument error.
			if strings.Contains(body, "invalid -data-design-candidate-kind") {
				t.Fatalf("the published kind %q was refused as an argument error: %s", kind, body)
			}
			if code == 0 {
				t.Fatalf("a wrong fingerprint was accepted: %s", body)
			}
		})
	}
}
