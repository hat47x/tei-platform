package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fixture returns a minimal valid reference envelope. Values are synthetic;
// no real SUI document is used.
func fixture() map[string]any {
	return map[string]any{
		"schema":          "tai.reference.sui-document-reference/v0",
		"authority":       "sui-document",
		"acceptedMeaning": false,
		"semanticStatus":  "reference-only",
		"document": map[string]any{
			"ref": map[string]any{"kind": "sui.document", "id": "doc-1"},
		},
		"revision": map[string]any{
			"sourceRevision":               "rev-1",
			"documentDigest":               strings.Repeat("a", 64),
			"revisionStatus":               "observed",
			"change":                       "initial",
			"previousDigest":               nil,
			"previousSourceRevision":       nil,
			"previousReferenceInvalidated": false,
		},
		"cards": []any{
			map[string]any{
				"ref":     map[string]any{"kind": "sui.card", "id": "card-1"},
				"review":  map[string]any{"text": "reviewed"},
				"content": map[string]any{"status": "not-carried"},
				"canvas":  map[string]any{"x": 1, "y": 2, "coordinateSource": "artifact"},
				"memberships": []any{
					map[string]any{"kind": "containment", "islandRef": map[string]any{"kind": "sui.island", "id": "island-1"}},
					map[string]any{"kind": "affiliation", "islandRef": map[string]any{"kind": "sui.island", "id": "island-2"}},
				},
			},
			map[string]any{
				"ref":         map[string]any{"kind": "sui.card", "id": "card-2"},
				"review":      map[string]any{"text": "unreviewed"},
				"content":     map[string]any{"status": "withheld-unreviewed"},
				"canvas":      map[string]any{"x": 3, "y": 4, "coordinateSource": "artifact"},
				"memberships": []any{},
			},
		},
		"islands": []any{
			map[string]any{
				"ref":                 map[string]any{"kind": "sui.island", "id": "island-1"},
				"review":              map[string]any{"title": "reviewed"},
				"containmentCardRefs": []any{map[string]any{"kind": "sui.card", "id": "card-1"}},
				"parentIslandRef":     nil,
				"affiliationRefCount": 1,
			},
		},
		"relations": []any{
			map[string]any{
				"ref":          map[string]any{"kind": "sui.edge", "id": "edge-1"},
				"relationKind": "directed",
				"fromRef":      map[string]any{"kind": "sui.card", "id": "card-1"},
				"toRef":        map[string]any{"kind": "sui.card", "id": "card-2"},
				"relationType": "related",
				"parallel":     map[string]any{"groupRef": "card-1|card-2|related", "index": 1, "count": 2},
			},
			map[string]any{
				"ref":          map[string]any{"kind": "sui.edge", "id": "edge-2"},
				"relationKind": "directed",
				"fromRef":      map[string]any{"kind": "sui.card", "id": "card-1"},
				"toRef":        map[string]any{"kind": "sui.card", "id": "card-2"},
				"relationType": "related",
				"parallel":     map[string]any{"groupRef": "card-1|card-2|related", "index": 2, "count": 2},
			},
			map[string]any{
				"ref":          map[string]any{"kind": "sui.containment", "id": "island-1:card-1"},
				"relationKind": "containment",
				"fromRef":      map[string]any{"kind": "sui.island", "id": "island-1"},
				"toRef":        map[string]any{"kind": "sui.card", "id": "card-1"},
				"relationType": "contains",
			},
			map[string]any{
				"ref":          map[string]any{"kind": "sui.affiliation", "id": "aff-1"},
				"relationKind": "affiliation",
				"fromRef":      map[string]any{"kind": "sui.card", "id": "card-1"},
				"toRef":        map[string]any{"kind": "sui.island", "id": "island-2"},
				"relationType": "affiliated-with",
			},
		},
		"completeness": map[string]any{"status": "complete", "reasons": []any{}},
		"disclosure": map[string]any{
			"mode":                 "references-only",
			"textIncluded":         false,
			"unreviewedContent":    "withheld",
			"privateEntities":      "omitted",
			"restrictedKinds":      []any{},
			"hasRestrictedContent": false,
		},
		"losses": []any{
			map[string]any{"feature": "tree-forest", "status": "unsupported", "detail": "not carried"},
		},
	}
}

func encode(t *testing.T, value any) []byte {
	t.Helper()
	payload, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return payload
}

func TestValidReferenceIsAccepted(t *testing.T) {
	report := Check(encode(t, fixture()))
	if !report.OK {
		t.Fatalf("valid reference rejected: %+v", report)
	}
	if report.Schema != "tai.reference.sui-document-reference/v0" {
		t.Fatalf("schema = %q", report.Schema)
	}
}

func TestAcceptedMeaningClaimIsRejected(t *testing.T) {
	for name, mutate := range map[string]func(map[string]any){
		"acceptedMeaning": func(m map[string]any) { m["acceptedMeaning"] = true },
		"semanticStatus":  func(m map[string]any) { m["semanticStatus"] = "accepted" },
		"authority":       func(m map[string]any) { m["authority"] = "tei-canonical" },
	} {
		payload := fixture()
		mutate(payload)
		report := Check(encode(t, payload))
		if report.Failure != FailureAcceptedMeaning {
			t.Fatalf("%s: failure = %q, want %q", name, report.Failure, FailureAcceptedMeaning)
		}
		if strings.Contains(strings.Join(report.Problems, " "), "doc-1") {
			t.Fatalf("%s: failure leaked an id", name)
		}
	}
}

func TestUnsupportedSchemaIsRejected(t *testing.T) {
	for name, mutate := range map[string]func(map[string]any){
		"schema":      func(m map[string]any) { m["schema"] = "other/v0" },
		"unknown-key": func(m map[string]any) { m["text"] = "secret" },
	} {
		payload := fixture()
		mutate(payload)
		report := Check(encode(t, payload))
		if report.Failure != FailureUnsupportedSchema {
			t.Fatalf("%s: failure = %q, want %q", name, report.Failure, FailureUnsupportedSchema)
		}
	}
}

func TestTextIncludedIsRejected(t *testing.T) {
	payload := fixture()
	payload["disclosure"].(map[string]any)["textIncluded"] = true
	report := Check(encode(t, payload))
	if report.Failure != FailureContentLeak {
		t.Fatalf("failure = %q, want %q", report.Failure, FailureContentLeak)
	}
}

func TestUnreviewedContentMustBeWithheld(t *testing.T) {
	payload := fixture()
	payload["disclosure"].(map[string]any)["unreviewedContent"] = "present"
	report := Check(encode(t, payload))
	if report.Failure != FailureInvalidReference {
		t.Fatalf("failure = %q, want %q", report.Failure, FailureInvalidReference)
	}
}

func TestReviewAndContentContradictionIsRejected(t *testing.T) {
	payload := fixture()
	card := payload["cards"].([]any)[0].(map[string]any)
	card["content"].(map[string]any)["status"] = "withheld-unreviewed"
	report := Check(encode(t, payload))
	if report.Failure != FailureInvalidReference {
		t.Fatalf("failure = %q, want %q", report.Failure, FailureInvalidReference)
	}
}

func TestParallelGroupCountMustMatchOccurrences(t *testing.T) {
	payload := fixture()
	relations := payload["relations"].([]any)
	relations[0].(map[string]any)["parallel"].(map[string]any)["count"] = 3
	report := Check(encode(t, payload))
	if report.Failure != FailureInvalidReference {
		t.Fatalf("failure = %q, want %q", report.Failure, FailureInvalidReference)
	}
}

func TestDirectedRelationWithoutParallelIsRejected(t *testing.T) {
	payload := fixture()
	delete(payload["relations"].([]any)[0].(map[string]any), "parallel")
	report := Check(encode(t, payload))
	if report.Failure != FailureInvalidReference {
		t.Fatalf("failure = %q, want %q", report.Failure, FailureInvalidReference)
	}
}

func TestUnknownContainmentCardIsRejected(t *testing.T) {
	payload := fixture()
	payload["islands"].([]any)[0].(map[string]any)["containmentCardRefs"] = []any{
		map[string]any{"kind": "sui.card", "id": "card-9"},
	}
	report := Check(encode(t, payload))
	if report.Failure != FailureInvalidReference {
		t.Fatalf("failure = %q, want %q", report.Failure, FailureInvalidReference)
	}
	for _, problem := range report.Problems {
		if strings.Contains(problem, "card-9") {
			t.Fatalf("problem leaked an id: %s", problem)
		}
	}
}

func TestFixtureFileRoundTrip(t *testing.T) {
	path := filepath.Join("testdata", "small-document.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	if report := Check(raw); !report.OK {
		t.Fatalf("checked-in fixture rejected: %+v", report)
	}
}
