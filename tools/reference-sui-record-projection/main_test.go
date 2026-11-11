package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// envelope returns a synthetic SUI reference envelope. Values are synthetic;
// no real document is used.
func envelope() map[string]any {
	return map[string]any{
		"schema":          "tai.reference.sui-document-reference/v0",
		"authority":       "sui-document",
		"acceptedMeaning": false,
		"semanticStatus":  "reference-only",
		"document": map[string]any{
			"ref":              map[string]any{"kind": "sui.document", "id": "doc-1"},
			"status":           "active",
			"projectionSchema": "tai.reference.tei-documentv1-projection/v0",
			"referenceOf":      "SUI DocumentV1",
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
				"ref":       map[string]any{"kind": "sui.card", "id": "card-1"},
				"claimType": "claim",
				"review":    map[string]any{"text": "reviewed"},
				"content":   map[string]any{"status": "not-carried"},
				"canvas":    map[string]any{"x": 1, "y": 2, "coordinateSource": "artifact"},
				"memberships": []any{
					map[string]any{"kind": "containment", "islandRef": map[string]any{"kind": "sui.island", "id": "island-1"}},
					map[string]any{"kind": "affiliation", "islandRef": map[string]any{"kind": "sui.island", "id": "island-2"}},
				},
			},
			map[string]any{
				"ref":         map[string]any{"kind": "sui.card", "id": "card-2"},
				"claimType":   "hypothesis",
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
		"completeness": map[string]any{"status": "partial", "reasons": []any{"artifact-declares-partial"}},
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

func marshal(t *testing.T, value any) []byte {
	t.Helper()
	payload, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return payload
}

func expectFailure(t *testing.T, payload []byte, want FailureCode) Report {
	t.Helper()
	report := Project(payload, "")
	if report.OK {
		t.Fatalf("projection succeeded, want failure %q", want)
	}
	if report.Failure != want {
		t.Fatalf("failure = %q, want %q (problems: %v)", report.Failure, want, report.Problems)
	}
	for _, problem := range report.Problems {
		if strings.Contains(problem, "card-1") || strings.Contains(problem, "doc-1") {
			t.Fatalf("failure leaked an id: %s", problem)
		}
	}
	return report
}

// M3-T01: 5 Card / 3 Island / 2 Relation の ID 集合が入力と一致する。
func TestM3ProjectionPreservesIdentitySet(t *testing.T) {
	payload := envelope()
	// 入力の Card と Island を 5 / 3 に増やす。
	cards := payload["cards"].([]any)
	for _, id := range []string{"card-3", "card-4", "card-5"} {
		cards = append(cards, map[string]any{
			"ref":         map[string]any{"kind": "sui.card", "id": id},
			"review":      map[string]any{"text": "reviewed"},
			"content":     map[string]any{"status": "not-carried"},
			"canvas":      map[string]any{"x": 0, "y": 0, "coordinateSource": "artifact"},
			"memberships": []any{},
		})
	}
	payload["cards"] = cards
	islands := payload["islands"].([]any)
	for _, id := range []string{"island-2", "island-3"} {
		islands = append(islands, map[string]any{
			"ref":                 map[string]any{"kind": "sui.island", "id": id},
			"review":              map[string]any{"title": "unknown"},
			"containmentCardRefs": []any{},
			"parentIslandRef":     nil,
			"affiliationRefCount": 0,
		})
	}
	payload["islands"] = islands

	report := Project(marshal(t, payload), "")
	if !report.OK {
		t.Fatalf("projection failed: %+v", report)
	}
	if len(report.Record.Cards) != 5 {
		t.Fatalf("cards = %d, want 5", len(report.Record.Cards))
	}
	if len(report.Record.Islands) != 3 {
		t.Fatalf("islands = %d, want 3", len(report.Record.Islands))
	}
	if len(report.Record.Relations) != 4 {
		t.Fatalf("relations = %d, want 4", len(report.Record.Relations))
	}
	if report.Record.Schema != recordSchema {
		t.Fatalf("schema = %q", report.Record.Schema)
	}
}

// M3-T02: 同名 Card を 2 件として保持する。
func TestM3SameNamedCardsStaySeparate(t *testing.T) {
	payload := envelope()
	cards := payload["cards"].([]any)
	cards[0].(map[string]any)["ref"] = map[string]any{"kind": "sui.card", "id": "card-same-a"}
	cards[1].(map[string]any)["ref"] = map[string]any{"kind": "sui.card", "id": "card-same-b"}
	payload["cards"] = cards

	report := Project(marshal(t, payload), "")
	if !report.OK {
		t.Fatalf("projection failed: %+v", report)
	}
	ids := map[string]bool{}
	for _, card := range report.Record.Cards {
		ids[card.Ref["id"]] = true
	}
	if len(ids) != 2 {
		t.Fatalf("distinct card ids = %d, want 2", len(ids))
	}
}

// M3-T03 / M4-T01: 未レビュー Card は参照を残し、本文は withheld。
func TestM3M4UnreviewedCardKeepsReferenceAndWithholdsText(t *testing.T) {
	report := Project(marshal(t, envelope()), "")
	if !report.OK {
		t.Fatalf("projection failed: %+v", report)
	}

	found := false
	for _, card := range report.Record.Cards {
		if card.Ref["id"] != "card-2" {
			continue
		}
		found = true
		if card.TextReview != "unreviewed" {
			t.Fatalf("textReview = %q, want unreviewed", card.TextReview)
		}
	}
	if !found {
		t.Fatal("unreviewed card was dropped from the projection")
	}

	textStates := map[string]string{}
	for _, entry := range report.Record.Disclosure {
		if entry.Item != "content.text" {
			continue
		}
		textStates[entry.Target] = entry.State
	}
	if textStates["tei-reference"] != "withheld" {
		t.Fatalf("tei-reference content.text = %q, want withheld", textStates["tei-reference"])
	}
	if textStates["sho-draft"] != "unknown" {
		t.Fatalf("sho-draft content.text = %q, want unknown", textStates["sho-draft"])
	}
}

// M3-T04 / M4-T03: 文書単位の制限は投影全体の拒否。
func TestM3M4DocumentRestrictionRejectsWholeProjection(t *testing.T) {
	payload := envelope()
	payload["disclosure"].(map[string]any)["documentRestricted"] = true
	expectFailure(t, marshal(t, payload), FailureDocumentDisclosure)
}

// M3-T05: digest と revision を別欄に保つ。
func TestM3DigestAndRevisionStaySeparate(t *testing.T) {
	report := Project(marshal(t, envelope()), "")
	if !report.OK {
		t.Fatalf("projection failed: %+v", report)
	}
	if report.Record.Revision.ContentDigest != strings.Repeat("a", 64) {
		t.Fatalf("contentDigest = %q", report.Record.Revision.ContentDigest)
	}
	if report.Record.Source.Revision != "rev-1" {
		t.Fatalf("source revision = %q", report.Record.Source.Revision)
	}
	if report.Record.Source.Revision == report.Record.Revision.ContentDigest {
		t.Fatal("source revision equals the content digest")
	}
	if report.Record.Source.Authority != "artifact-declared" {
		t.Fatalf("source authority = %q", report.Record.Source.Authority)
	}
}

// M3: 生成物は本文を運ばないことを自分で宣言する。
func TestM3RecordDeclaresNoContent(t *testing.T) {
	report := Project(marshal(t, envelope()), "")
	if !report.OK {
		t.Fatalf("projection failed: %+v", report)
	}
	if report.Record.Generator.CarriesContent {
		t.Fatal("record claims to carry content")
	}
	if report.Record.AcceptedMeaning {
		t.Fatal("record claims accepted meaning")
	}
	if report.Record.Generator.InputDocumentDigest != strings.Repeat("a", 64) {
		t.Fatalf("input digest = %q", report.Record.Generator.InputDocumentDigest)
	}
}

// M4-T02: entity 単位の省略は ID・件数を出さず、開示判断へ反映する。
func TestM4EntityOmissionIsVisibleWithoutCounts(t *testing.T) {
	payload := envelope()
	payload["disclosure"].(map[string]any)["restrictedKinds"] = []any{"card"}
	report := Project(marshal(t, payload), "")
	if !report.OK {
		t.Fatalf("projection failed: %+v", report)
	}
	found := false
	for _, entry := range report.Record.Disclosure {
		if entry.Target == "tei-reference" && entry.Item == "identity.and.relationShape" && entry.State == "withheld" {
			found = true
		}
	}
	if !found {
		t.Fatal("entity omission is not reflected in the disclosure matrix")
	}
}

// M4-T04: TEI と agent の判断が独立している。
func TestM4TargetsAreIndependent(t *testing.T) {
	report := Project(marshal(t, envelope()), "")
	if !report.OK {
		t.Fatalf("projection failed: %+v", report)
	}
	states := map[string]map[string]string{}
	for _, entry := range report.Record.Disclosure {
		if states[entry.Item] == nil {
			states[entry.Item] = map[string]string{}
		}
		states[entry.Item][entry.Target] = entry.State
	}

	canvas := states["canvas.coordinates"]
	if canvas["tei-reference"] != "present" {
		t.Fatalf("tei-reference canvas = %q", canvas["tei-reference"])
	}
	if canvas["external-agent"] != "unknown" {
		t.Fatalf("external-agent canvas = %q, want unknown", canvas["external-agent"])
	}
	if canvas["sho-draft"] != "not-applicable" {
		t.Fatalf("sho-draft canvas = %q, want not-applicable", canvas["sho-draft"])
	}

	text := states["content.text"]
	if text["sui-ui"] != "present" || text["tei-reference"] != "withheld" {
		t.Fatalf("content.text states = %v", text)
	}
}

// M4-T05: unknown を present / withheld へ読み替えない。
func TestM4UnknownIsNotCollapsed(t *testing.T) {
	report := Project(marshal(t, envelope()), "")
	if !report.OK {
		t.Fatalf("projection failed: %+v", report)
	}
	for _, entry := range report.Record.Disclosure {
		if entry.Item == "review.attribution" && entry.State != "unknown" {
			t.Fatalf("review.attribution = %q, want unknown", entry.State)
		}
	}
}

// M5-T02: expired / revoked は投影を返さない。
func TestM5SourceNotActiveBlocksProjection(t *testing.T) {
	for _, status := range []string{"revoked", "expired"} {
		payload := envelope()
		payload["document"].(map[string]any)["status"] = status
		expectFailure(t, marshal(t, payload), FailureSourceUnavailable)
	}
	// 未知の値は active でも unknown でもないので拒否する。
	payload := envelope()
	payload["document"].(map[string]any)["status"] = "suspended"
	expectFailure(t, marshal(t, payload), FailureSourceUnavailable)
}

// M5-T01 / M5-T04: status 欠落は unknown。active に読み替えない。
func TestM5MissingStatusIsUnknown(t *testing.T) {
	payload := envelope()
	delete(payload["document"].(map[string]any), "status")
	report := Project(marshal(t, payload), "")
	if !report.OK {
		t.Fatalf("projection failed: %+v", report)
	}
	if report.Record.Source.Status != "unknown" {
		t.Fatalf("source status = %q, want unknown", report.Record.Source.Status)
	}
}

// M5-T03 / M5-T05: 期待 revision と異なる参照は現在として受理しない。
func TestM5StaleReferenceIsRejected(t *testing.T) {
	payload := marshal(t, envelope())
	report := Project(payload, "rev-1")
	if !report.OK {
		t.Fatalf("matching revision rejected: %+v", report)
	}
	stale := Project(payload, "rev-0")
	if stale.OK || stale.Failure != FailureStaleReference {
		t.Fatalf("stale failure = %q, want %q", stale.Failure, FailureStaleReference)
	}
}

// M5: revisionStatus が unknown のとき、期待 revision は照合できない。
func TestM5UnknownRevisionCannotSatisfyExpectation(t *testing.T) {
	payload := envelope()
	payload["revision"].(map[string]any)["revisionStatus"] = "unknown"
	delete(payload["revision"].(map[string]any), "sourceRevision")
	if report := Project(marshal(t, payload), ""); !report.OK {
		t.Fatalf("unknown revision rejected without expectation: %+v", report)
	} else if report.Record.Revision.RevisionStatus != "unknown" {
		t.Fatalf("revisionStatus = %q", report.Record.Revision.RevisionStatus)
	}
	if report := Project(marshal(t, payload), "rev-1"); report.Failure != FailureRevisionUnknown {
		t.Fatalf("failure = %q, want %q", report.Failure, FailureRevisionUnknown)
	}
}

// 既存の consumer と同じ入力境界を守る。
func TestSchemaAndMeaningGates(t *testing.T) {
	payload := envelope()
	payload["schema"] = "other/v0"
	expectFailure(t, marshal(t, payload), FailureUnsupportedSchema)

	payload = envelope()
	payload["acceptedMeaning"] = true
	expectFailure(t, marshal(t, payload), FailureAcceptedMeaning)

	payload = envelope()
	payload["disclosure"].(map[string]any)["textIncluded"] = true
	expectFailure(t, marshal(t, payload), FailureContentLeak)

	payload = envelope()
	payload["text"] = "secret"
	expectFailure(t, marshal(t, payload), FailureUnsupportedSchema)
}

func TestFixtureFileRoundTrip(t *testing.T) {
	path := filepath.Join("testdata", "small-document.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	report := Project(raw, "")
	if !report.OK {
		t.Fatalf("checked-in fixture rejected: %+v", report)
	}
	if !strings.HasPrefix(report.Record.Generator.InputSourceRevision, "rev-") {
		t.Fatalf("fixture source revision = %q", report.Record.Generator.InputSourceRevision)
	}
}
