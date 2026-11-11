// Command reference-sui-record-projection は、SUI が返す
// `tai.reference.sui-document-reference/v0` から、開示判断つきの
// record 投影を作る read-only の Reference 実装である（パケット M3・M4・M5）。
//
// この tool は Product API・共通 Operation・第二の正本ではない。
//
//   - 入力は SUI の reference envelope だけであり、DocumentV1 も artifact も読まない。
//   - 出力は `tai.reference.sui-document-record/v0` で、本文・題名・要約を
//     含まない。項目ごとの開示状態（present/withheld/unknown/not-applicable）だけを持つ。
//   - Source 状態は envelope の宣言をそのまま使い、欠落を active に読み替えない。
//   - 内容指紋（documentDigest）と Source revision を別欄に保つ。
//   - 文書単位の開示制限は、項目ごとの withheld ではなく投影全体の拒否として扱う。
//
// 失敗メッセージに、入力の ID・本文・件数を含めない。
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"
)

const (
	recordSchema     = "tai.reference.sui-document-record/v0"
	inputSchema      = "tai.reference.sui-document-reference/v0"
	generatorVersion = "sui-record-projection/v0"
)

// DisclosureState は項目ごとの開示判断である。boolean にしない。
type DisclosureState string

const (
	Present       DisclosureState = "present"
	Withheld      DisclosureState = "withheld"
	Unknown       DisclosureState = "unknown"
	NotApplicable DisclosureState = "not-applicable"
)

// Target は開示先である。判断を開示先ごとに分ける。
type Target string

const (
	TargetSUIUI        Target = "sui-ui"
	TargetTEIReference Target = "tei-reference"
	TargetSHODraft     Target = "sho-draft"
	TargetExternalAI   Target = "external-agent"
)

var targets = []Target{TargetSUIUI, TargetTEIReference, TargetSHODraft, TargetExternalAI}

// FailureCode は入力に依存しない安定した失敗の名前である。
type FailureCode string

const (
	FailureUnsupportedSchema  FailureCode = "unsupported-schema"
	FailureInvalidReference   FailureCode = "invalid-reference"
	FailureAcceptedMeaning    FailureCode = "accepted-meaning-claimed"
	FailureContentLeak        FailureCode = "content-in-reference"
	FailureSourceUnavailable  FailureCode = "source-unavailable"
	FailureDocumentDisclosure FailureCode = "document-disclosure-restricted"
	FailureStaleReference     FailureCode = "stale-reference"
	FailureRevisionUnknown    FailureCode = "revision-unresolved"
)

// Report は投影の結果である。ID・本文・件数を持たない失敗だけを返せる。
type Report struct {
	OK       bool        `json:"ok"`
	Schema   string      `json:"schema,omitempty"`
	Failure  FailureCode `json:"failure,omitempty"`
	Problems []string    `json:"problems,omitempty"`

	Record *Record `json:"record,omitempty"`
}

type Generator struct {
	Version             string `json:"version"`
	InputSchema         string `json:"inputSchema"`
	InputDocumentDigest string `json:"inputDocumentDigest"`
	InputSourceRevision string `json:"inputSourceRevision,omitempty"`
	CarriesContent      bool   `json:"carriesContent"`
}

type RecordDocument struct {
	Ref map[string]string `json:"ref"`
}

// SourceObservation は、Source 状態の権威を consumer が推定しないための欄である。
type SourceObservation struct {
	Status      string `json:"status"`
	Authority   string `json:"authority"`
	Revision    string `json:"revision,omitempty"`
	ObservedVia string `json:"observedVia"`
}

type RecordRevision struct {
	ContentDigest       string `json:"contentDigest"`
	SourceRevision      string `json:"sourceRevision,omitempty"`
	RevisionStatus      string `json:"revisionStatus"`
	Change              string `json:"change"`
	PreviousDigest      string `json:"previousDigest,omitempty"`
	PreviousInvalidated bool   `json:"previousReferenceInvalidated"`
}

type RecordCard struct {
	Ref         map[string]string `json:"ref"`
	ClaimType   string            `json:"claimType,omitempty"`
	HoldState   string            `json:"holdState,omitempty"`
	TextReview  string            `json:"textReview"`
	Canvas      map[string]any    `json:"canvas"`
	Memberships []map[string]any  `json:"memberships"`
}

type RecordIsland struct {
	Ref         map[string]string   `json:"ref"`
	TitleReview string              `json:"titleReview"`
	Containment []map[string]string `json:"containment"`
	Parent      map[string]string   `json:"parentIslandRef,omitempty"`
}

type RecordRelation struct {
	Ref          map[string]string `json:"ref"`
	RelationKind string            `json:"relationKind"`
	FromRef      map[string]string `json:"fromRef"`
	ToRef        map[string]string `json:"toRef"`
	RelationType string            `json:"relationType,omitempty"`
	Parallel     map[string]any    `json:"parallel,omitempty"`
}

type RecordMembership struct {
	Ref       map[string]string `json:"ref"`
	Kind      string            `json:"kind"`
	CardRef   map[string]string `json:"cardRef"`
	IslandRef map[string]string `json:"islandRef"`
}

// DisclosureEntry は「項目 × 開示先」の判断である。unknown と withheld を区別する。
type DisclosureEntry struct {
	Item   string `json:"item"`
	Target string `json:"target"`
	State  string `json:"state"`
	Reason string `json:"reason,omitempty"`
}

type Record struct {
	Schema          string              `json:"schema"`
	Generator       Generator           `json:"generator"`
	Authority       string              `json:"authority"`
	AcceptedMeaning bool                `json:"acceptedMeaning"`
	Document        RecordDocument      `json:"document"`
	Source          SourceObservation   `json:"source"`
	Revision        RecordRevision      `json:"revision"`
	Cards           []RecordCard        `json:"cards"`
	Islands         []RecordIsland      `json:"islands"`
	Relations       []RecordRelation    `json:"relations"`
	Memberships     []RecordMembership  `json:"memberships"`
	Disclosure      []DisclosureEntry   `json:"disclosure"`
	Completeness    map[string]any      `json:"completeness"`
	Losses          []map[string]string `json:"losses"`
}

func main() {
	input := flag.String("input", "", "SUI reference envelope JSON file (required)")
	expectedRevision := flag.String("expected-source-revision", "", "Source revision the caller last saw")
	flag.Parse()

	if *input == "" {
		fmt.Fprintln(os.Stderr, "reference-sui-record-projection: -input is required")
		os.Exit(2)
	}
	payload, err := os.ReadFile(*input)
	if err != nil {
		fmt.Fprintln(os.Stderr, "reference-sui-record-projection: input cannot be read")
		os.Exit(1)
	}

	report := Project(payload, *expectedRevision)
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(report); err != nil {
		os.Exit(1)
	}
	if !report.OK {
		os.Exit(1)
	}
}

// Project は 1 つの envelope を record 投影へ変換する。
func Project(payload []byte, expectedSourceRevision string) Report {
	if len(bytes.TrimSpace(payload)) == 0 {
		return Report{Failure: FailureInvalidReference}
	}
	var root any
	if err := json.Unmarshal(payload, &root); err != nil {
		return Report{Failure: FailureInvalidReference}
	}
	envelope, ok := root.(map[string]any)
	if !ok {
		return Report{Failure: FailureInvalidReference}
	}

	if len(unknownTopLevel(envelope)) > 0 {
		return Report{Failure: FailureUnsupportedSchema}
	}
	if schema, _ := envelope["schema"].(string); schema != inputSchema {
		return Report{Failure: FailureUnsupportedSchema}
	}
	if claimed, _ := envelope["acceptedMeaning"].(bool); claimed {
		return Report{Failure: FailureAcceptedMeaning}
	}
	if status, _ := envelope["semanticStatus"].(string); status != "reference-only" {
		return Report{Failure: FailureAcceptedMeaning}
	}
	if authority, _ := envelope["authority"].(string); authority != "sui-document" {
		return Report{Failure: FailureAcceptedMeaning}
	}

	disclosure, _ := envelope["disclosure"].(map[string]any)
	if disclosure == nil {
		return Report{Failure: FailureInvalidReference}
	}
	if included, _ := disclosure["textIncluded"].(bool); included {
		return Report{Failure: FailureContentLeak}
	}

	// 文書単位の制限は、項目ごとの withheld ではなく投影全体の拒否として扱う。
	if restricted, _ := disclosure["documentRestricted"].(bool); restricted {
		return Report{Failure: FailureDocumentDisclosure}
	}

	// M5: Source 状態は宣言をそのまま使う。欠落を active にしない。
	document, _ := envelope["document"].(map[string]any)
	documentStatus, _ := document["status"].(string)
	switch documentStatus {
	case "active", "unknown":
	case "revoked", "expired":
		return Report{Failure: FailureSourceUnavailable}
	case "":
		documentStatus = "unknown"
	default:
		// 未知の値は active でも unknown でもない。宣言を理解できないので拒否する。
		return Report{Failure: FailureSourceUnavailable}
	}

	revision, _ := envelope["revision"].(map[string]any)
	digest, _ := revision["documentDigest"].(string)
	if digest == "" {
		return Report{Failure: FailureInvalidReference}
	}
	sourceRevision, _ := revision["sourceRevision"].(string)
	revisionStatus, _ := revision["revisionStatus"].(string)
	switch revisionStatus {
	case "observed", "unknown":
	default:
		return Report{Failure: FailureRevisionUnknown}
	}
	if expectedSourceRevision != "" {
		if sourceRevision == "" || revisionStatus != "observed" {
			return Report{Failure: FailureRevisionUnknown}
		}
		if sourceRevision != expectedSourceRevision {
			// 期待した revision と違う参照を、現在として受理しない。
			return Report{Failure: FailureStaleReference}
		}
	}

	record := buildRecord(envelope, document, revision, digest, sourceRevision, revisionStatus, documentStatus)
	if problems := validateRecord(record); len(problems) > 0 {
		sort.Strings(problems)
		return Report{Failure: FailureInvalidReference, Problems: problems}
	}
	return Report{OK: true, Schema: recordSchema, Record: record}
}

func buildRecord(
	envelope, document, revision map[string]any,
	digest, sourceRevision, revisionStatus, documentStatus string,
) *Record {
	ref, _ := document["ref"].(map[string]any)

	record := &Record{
		Schema: recordSchema,
		Generator: Generator{
			Version:             generatorVersion,
			InputSchema:         inputSchema,
			InputDocumentDigest: digest,
			InputSourceRevision: sourceRevision,
			// この投影は本文を運ばない。値を上げる変更は生成物の意味を変える。
			CarriesContent: false,
		},
		Authority:       "sui-document",
		AcceptedMeaning: false,
		Document:        RecordDocument{Ref: cloneRef(ref)},
		Source: SourceObservation{
			Status:      documentStatus,
			Authority:   "artifact-declared",
			Revision:    sourceRevision,
			ObservedVia: "sui-reference-envelope",
		},
		Revision: RecordRevision{
			ContentDigest:       digest,
			SourceRevision:      sourceRevision,
			RevisionStatus:      revisionStatus,
			Change:              stringField(revision, "change"),
			PreviousDigest:      stringField(revision, "previousDigest"),
			PreviousInvalidated: boolField(revision, "previousReferenceInvalidated"),
		},
		Completeness: map[string]any{},
	}

	if completeness, ok := envelope["completeness"].(map[string]any); ok {
		record.Completeness = completeness
	}
	if items, ok := envelope["losses"].([]any); ok {
		record.Losses = stringMapItems(items)
	}

	if cards, ok := envelope["cards"].([]any); ok {
		for _, item := range cards {
			card, ok := item.(map[string]any)
			if !ok {
				continue
			}
			record.Cards = append(record.Cards, RecordCard{
				Ref:         cloneRef(mapValue(card, "ref")),
				ClaimType:   stringField(card, "claimType"),
				HoldState:   stringField(card, "holdState"),
				TextReview:  nestedString(card, "review", "text"),
				Canvas:      mapValue(card, "canvas"),
				Memberships: mapList(card, "memberships"),
			})
		}
	}

	if islands, ok := envelope["islands"].([]any); ok {
		for _, item := range islands {
			island, ok := item.(map[string]any)
			if !ok {
				continue
			}
			entry := RecordIsland{
				Ref:         cloneRef(mapValue(island, "ref")),
				TitleReview: nestedString(island, "review", "title"),
				Containment: refList(island, "containmentCardRefs"),
			}
			if parent, ok := island["parentIslandRef"].(map[string]any); ok {
				entry.Parent = cloneRef(parent)
			}
			record.Islands = append(record.Islands, entry)
		}
	}

	if relations, ok := envelope["relations"].([]any); ok {
		for _, item := range relations {
			relation, ok := item.(map[string]any)
			if !ok {
				continue
			}
			entry := RecordRelation{
				Ref:          cloneRef(mapValue(relation, "ref")),
				RelationKind: stringField(relation, "relationKind"),
				FromRef:      cloneRef(mapValue(relation, "fromRef")),
				ToRef:        cloneRef(mapValue(relation, "toRef")),
				RelationType: stringField(relation, "relationType"),
			}
			if parallel, ok := relation["parallel"].(map[string]any); ok {
				entry.Parallel = parallel
			}
			record.Relations = append(record.Relations, entry)

			// membership は containment と affiliation を別の occurrence として数える。
			switch entry.RelationKind {
			case "containment":
				record.Memberships = append(record.Memberships, RecordMembership{
					Ref: entry.Ref, Kind: entry.RelationKind,
					CardRef: entry.ToRef, IslandRef: entry.FromRef,
				})
			case "affiliation":
				record.Memberships = append(record.Memberships, RecordMembership{
					Ref: entry.Ref, Kind: entry.RelationKind,
					CardRef: entry.FromRef, IslandRef: entry.ToRef,
				})
			}
		}
	}

	record.Disclosure = disclosureMatrix(envelope)
	return record
}

// disclosureMatrix は「項目 × 開示先」の判断を作る。
// SUI UI は元の DocumentV1 を読むので present、TEI 参照は本文を運ばないので withheld、
// SHO 下書きは方針が未決定なので unknown とする。判断を相互に継承しない。
func disclosureMatrix(envelope map[string]any) []DisclosureEntry {
	entries := []DisclosureEntry{}
	add := func(item string, target Target, state DisclosureState, reason string) {
		entries = append(entries, DisclosureEntry{
			Item: item, Target: string(target), State: string(state), Reason: reason,
		})
	}

	for _, target := range targets {
		add("identity.and.relationShape", target, Present, "the reference carries ids and relation shape")
		add("review.flag", target, Present, "review flags are reference vocabulary, not content")
	}

	for _, target := range targets {
		switch target {
		case TargetSUIUI:
			add("content.text", target, Present, "the SUI UI reads its own DocumentV1")
		case TargetTEIReference, TargetExternalAI:
			add("content.text", target, Withheld, "the reference path never carries text")
		case TargetSHODraft:
			add("content.text", target, Unknown, "the SHO draft policy is not decided")
		}
	}
	for _, target := range targets {
		switch target {
		case TargetSUIUI:
			add("content.title", target, Present, "the SUI UI reads its own DocumentV1")
		default:
			add("content.title", target, Withheld, "titles are content and are not carried")
		}
	}
	for _, target := range targets {
		add("review.attribution", target, Unknown, "the artifact carries flags only")
	}
	for _, target := range targets {
		switch target {
		case TargetSUIUI, TargetTEIReference:
			add("canvas.coordinates", target, Present, "x/y only; transform and z-order are lost")
		case TargetSHODraft:
			add("canvas.coordinates", target, NotApplicable, "a Markdown draft has no canvas")
		case TargetExternalAI:
			add("canvas.coordinates", target, Unknown, "agent disclosure of coordinates is not decided")
		}
	}
	for _, target := range targets {
		add("revision.and.sourceState", target, Present, "revision and source status are reference vocabulary")
	}

	disclosure, _ := envelope["disclosure"].(map[string]any)
	if kinds, ok := disclosure["restrictedKinds"].([]any); ok && len(kinds) > 0 {
		entries = append(entries, DisclosureEntry{
			Item:   "identity.and.relationShape",
			Target: string(TargetTEIReference),
			State:  string(Withheld),
			Reason: "some entity kinds are omitted from the reference",
		})
	}
	return entries
}

func validateRecord(record *Record) []string {
	problems := []string{}
	if record.Generator.CarriesContent {
		problems = append(problems, "record claims to carry content")
	}
	if record.AcceptedMeaning {
		problems = append(problems, "record claims accepted meaning")
	}
	encoded, err := json.Marshal(record)
	if err != nil {
		return append(problems, "record cannot be encoded")
	}
	var generic any
	if err := json.Unmarshal(encoded, &generic); err == nil {
		if len(collectContentKeys(generic)) > 0 {
			problems = append(problems, "record contains a content key")
		}
	}
	if record.Source.Status == "" {
		problems = append(problems, "source status is not declared")
	}
	if record.Source.Revision != "" && record.Source.Revision == record.Revision.ContentDigest {
		problems = append(problems, "source revision and content digest are the same value")
	}
	for _, entry := range record.Disclosure {
		switch DisclosureState(entry.State) {
		case Present, Withheld, Unknown, NotApplicable:
		default:
			problems = append(problems, "disclosure state is not a known state")
		}
	}
	return problems
}

// collectContentKeys は content のキー名だけを返す。値は返さない。
func collectContentKeys(value any) []string {
	wanted := map[string]bool{"text": true, "title": true, "body": true, "summary": true, "narrative": true}
	var walk func(any) []string
	walk = func(node any) []string {
		switch typed := node.(type) {
		case map[string]any:
			found := []string{}
			for key, nested := range typed {
				if wanted[key] {
					found = append(found, key)
				}
				found = append(found, walk(nested)...)
			}
			return found
		case []any:
			found := []string{}
			for _, nested := range typed {
				found = append(found, walk(nested)...)
			}
			return found
		default:
			return nil
		}
	}
	return walk(value)
}

func unknownTopLevel(envelope map[string]any) []string {
	known := map[string]bool{
		"schema": true, "authority": true, "acceptedMeaning": true,
		"semanticStatus": true, "document": true, "revision": true,
		"cards": true, "islands": true, "relations": true,
		"completeness": true, "disclosure": true, "losses": true,
	}
	unknown := []string{}
	for key := range envelope {
		if !known[key] {
			unknown = append(unknown, key)
		}
	}
	return unknown
}

func stringField(object map[string]any, key string) string {
	value, _ := object[key].(string)
	return value
}

func boolField(object map[string]any, key string) bool {
	value, _ := object[key].(bool)
	return value
}

func nestedString(object map[string]any, outer, inner string) string {
	nested, ok := object[outer].(map[string]any)
	if !ok {
		return ""
	}
	value, _ := nested[inner].(string)
	return value
}

func mapValue(object map[string]any, key string) map[string]any {
	value, _ := object[key].(map[string]any)
	return value
}

func mapList(object map[string]any, key string) []map[string]any {
	items, ok := object[key].([]any)
	if !ok {
		return nil
	}
	result := []map[string]any{}
	for _, item := range items {
		if typed, ok := item.(map[string]any); ok {
			result = append(result, typed)
		}
	}
	return result
}

func refList(object map[string]any, key string) []map[string]string {
	items, ok := object[key].([]any)
	if !ok {
		return nil
	}
	result := []map[string]string{}
	for _, item := range items {
		if typed, ok := item.(map[string]any); ok {
			result = append(result, cloneRef(typed))
		}
	}
	return result
}

func stringMapItems(items []any) []map[string]string {
	result := []map[string]string{}
	for _, item := range items {
		typed, ok := item.(map[string]any)
		if !ok {
			continue
		}
		entry := map[string]string{}
		for key, value := range typed {
			if text, ok := value.(string); ok {
				entry[key] = text
			}
		}
		result = append(result, entry)
	}
	return result
}

func cloneRef(ref map[string]any) map[string]string {
	if ref == nil {
		return nil
	}
	kind := stringField(ref, "kind")
	id := stringField(ref, "id")
	if kind == "" || id == "" {
		return nil
	}
	return map[string]string{"kind": kind, "id": id}
}
