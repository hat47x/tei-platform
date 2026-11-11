// Command reference-sui-reference-consumer は、SUI が返す
// `tai.reference.sui-document-reference/v0` の envelope を読み、
// consumer が緩めてはいけない主張を検査する。
//
// この tool は read-only の Reference consumer である。Product API でも
// 共通 Operation でも第二の正本でもない。検査するのは次の点に限る。
//
//   - Card が読めることは TEI の Accepted Meaning ではない
//     (`acceptedMeaning` は false、`semanticStatus` は `reference-only`)
//   - 伏せた本文・未レビュー・非開示の内容が envelope に混じっていない
//   - occurrence identity が保たれている（同名 Card、並行 Edge、
//     containment と affiliation を織り込まない）
//
// 失敗メッセージに、入力の ID や本文を含めない。
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
	referenceSchema    = "tai.reference.sui-document-reference/v0"
	referenceAuthority = "sui-document"
	referenceStatus    = "reference-only"
)

// FailureCode は入力に依存しない安定した失敗の名前である。
type FailureCode string

const (
	FailureUnsupportedSchema FailureCode = "unsupported-schema"
	FailureInvalidReference  FailureCode = "invalid-reference"
	FailureAcceptedMeaning   FailureCode = "accepted-meaning-claimed"
	FailureContentLeak       FailureCode = "content-in-reference"
)

// Report は consumer の結果である。ID・本文・非公開 entity 由来の件数を持たない。
type Report struct {
	OK       bool        `json:"ok"`
	Schema   string      `json:"schema,omitempty"`
	Failure  FailureCode `json:"failure,omitempty"`
	Problems []string    `json:"problems,omitempty"`
}

// contentKeys は、reference-only の envelope に現れてはいけない内容の名前である。
// 既知の参照フィールド（`content` は status だけ、`canvas` は座標、
// `review.text` は review 状態の名前）は名前が重なるため、走査から除く。
var contentKeys = []string{"text", "title", "body", "summary", "narrative", "attribution", "reviewerRef"}

// excludedSubtrees are reference field names whose nested keys are fixed
// vocabulary, not content. `review.text` is a review state name, not a body.
var excludedSubtrees = []string{"content", "review", "canvas", "losses", "disclosure", "completeness", "revision"}

func main() {
	input := flag.String("input", "", "SUI reference envelope JSON file (required)")
	flag.Parse()

	if *input == "" {
		fmt.Fprintln(os.Stderr, "reference-sui-reference-consumer: -input is required")
		os.Exit(2)
	}
	payload, err := os.ReadFile(*input)
	if err != nil {
		fmt.Fprintln(os.Stderr, "reference-sui-reference-consumer: input cannot be read")
		os.Exit(1)
	}

	report := Check(payload)
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(report); err != nil {
		os.Exit(1)
	}
	if !report.OK {
		os.Exit(1)
	}
}

// Check は 1 つの envelope を検査する。
func Check(payload []byte) Report {
	if len(bytes.TrimSpace(payload)) == 0 {
		return Report{Failure: FailureInvalidReference}
	}

	var root any
	if err := json.Unmarshal(payload, &root); err != nil {
		return Report{Failure: FailureInvalidReference}
	}
	object, ok := root.(map[string]any)
	if !ok {
		return Report{Failure: FailureInvalidReference}
	}

	if unknown := unknownTopLevelKeys(object); len(unknown) > 0 {
		// consumer compatibility guard: an unknown top-level key fails closed
		// before any value is trusted.
		return Report{Failure: FailureUnsupportedSchema}
	}

	schema, _ := object["schema"].(string)
	if schema != referenceSchema {
		return Report{Failure: FailureUnsupportedSchema}
	}
	if claimed, _ := object["acceptedMeaning"].(bool); claimed {
		return Report{Failure: FailureAcceptedMeaning}
	}
	if status, _ := object["semanticStatus"].(string); status != referenceStatus {
		return Report{Failure: FailureAcceptedMeaning}
	}
	if authority, _ := object["authority"].(string); authority != referenceAuthority {
		return Report{Failure: FailureAcceptedMeaning}
	}

	disclosure, _ := object["disclosure"].(map[string]any)
	if disclosure == nil {
		return Report{Failure: FailureInvalidReference}
	}
	if included, _ := disclosure["textIncluded"].(bool); included {
		return Report{Failure: FailureContentLeak}
	}

	problems := []string{}
	if unreviewed, _ := disclosure["unreviewedContent"].(string); unreviewed != "withheld" {
		problems = append(problems, "unreviewed content is not withheld")
	}
	if leaked := findContentKeys(object); len(leaked) > 0 {
		problems = append(problems, "content key is present in a reference-only envelope")
	}

	cards, ok := object["cards"].([]any)
	if !ok {
		return Report{Failure: FailureInvalidReference}
	}
	cardIDs := map[string]bool{}
	for _, item := range cards {
		card, ok := item.(map[string]any)
		if !ok {
			problems = append(problems, "card is not an object")
			continue
		}
		id := refKey(card["ref"])
		if id == "" {
			problems = append(problems, "card ref is incomplete")
			continue
		}
		if cardIDs[id] {
			problems = append(problems, "card ref is repeated")
		}
		cardIDs[id] = true

		content, _ := card["content"].(map[string]any)
		status, _ := content["status"].(string)
		switch status {
		case "not-carried", "withheld-unreviewed":
		default:
			problems = append(problems, "card content status is not a reference status")
		}

		review, _ := card["review"].(map[string]any)
		textReview, _ := review["text"].(string)
		switch textReview {
		case "reviewed", "unreviewed", "unknown":
		default:
			problems = append(problems, "card review state is not a known state")
		}
		if status == "withheld-unreviewed" && textReview == "reviewed" {
			problems = append(problems, "reviewed card claims withheld text")
		}
	}

	islands, _ := object["islands"].([]any)
	for _, item := range islands {
		island, ok := item.(map[string]any)
		if !ok {
			problems = append(problems, "island is not an object")
			continue
		}
		if refKey(island["ref"]) == "" {
			problems = append(problems, "island ref is incomplete")
		}
		if refs, ok := island["containmentCardRefs"].([]any); ok {
			for _, ref := range refs {
				if key := refKey(ref); key == "" || !cardIDs[key] {
					problems = append(problems, "island containment names an unknown card")
				}
			}
		}
	}

	relations, _ := object["relations"].([]any)
	relationIDs := map[string]bool{}
	parallelCounts := map[string]int{}
	declared := map[string]int{}
	for _, item := range relations {
		relation, ok := item.(map[string]any)
		if !ok {
			problems = append(problems, "relation is not an object")
			continue
		}
		id := refKey(relation["ref"])
		if id == "" {
			problems = append(problems, "relation ref is incomplete")
			continue
		}
		if relationIDs[id] {
			problems = append(problems, "relation ref is repeated")
		}
		relationIDs[id] = true

		kind, _ := relation["relationKind"].(string)
		switch kind {
		case "directed", "containment", "affiliation":
		default:
			problems = append(problems, "relation kind is not a reference kind")
		}

		group, hasGroup := relation["parallel"].(map[string]any)
		if kind == "directed" && !hasGroup {
			problems = append(problems, "directed relation has no parallel group")
		}
		if kind != "directed" && hasGroup {
			problems = append(problems, "non-directed relation claims a parallel group")
		}
		if hasGroup {
			index, _ := group["index"].(float64)
			count, _ := group["count"].(float64)
			ref, _ := group["groupRef"].(string)
			if index < 1 || count < 1 || index > count {
				problems = append(problems, "parallel index is outside the group")
			}
			if ref == "" {
				problems = append(problems, "parallel group has no reference")
			} else {
				parallelCounts[ref]++
				// Every occurrence of one group must declare the same size.
				if declared[ref] == 0 {
					declared[ref] = int(count)
				} else if declared[ref] != int(count) {
					problems = append(problems, "parallel group declares inconsistent sizes")
				}
			}
		}
	}

	// 並行 group は、宣言した count と実 occurrence 数が一致しないと occurrence を失う。
	for ref, count := range declared {
		if parallelCounts[ref] != count {
			problems = append(problems, "parallel group count does not match occurrences")
			break
		}
	}

	if len(problems) > 0 {
		sort.Strings(problems)
		return Report{Failure: FailureInvalidReference, Problems: problems}
	}
	return Report{OK: true, Schema: schema}
}

// topLevelKeys is the closed set of keys the reference envelope may carry.
// A new key is a schema change and must be added here deliberately.
var topLevelKeys = []string{
	"schema", "authority", "acceptedMeaning", "semanticStatus", "document",
	"revision", "cards", "islands", "relations", "completeness", "disclosure",
	"losses",
}

func unknownTopLevelKeys(object map[string]any) []string {
	known := map[string]bool{}
	for _, key := range topLevelKeys {
		known[key] = true
	}
	unknown := []string{}
	for key := range object {
		if !known[key] {
			unknown = append(unknown, key)
		}
	}
	return unknown
}

func refKey(value any) string {
	ref, ok := value.(map[string]any)
	if !ok {
		return ""
	}
	kind, _ := ref["kind"].(string)
	id, _ := ref["id"].(string)
	if kind == "" || id == "" {
		return ""
	}
	return kind + "\x00" + id
}

// findContentKeys walks the envelope and returns the content key names it
// found. It never returns a value, so a failure message cannot leak content.
func findContentKeys(value any) []string {
	keys := []string{}
	for _, name := range contentKeys {
		keys = append(keys, collectKey(value, name)...)
	}
	return keys
}

func collectKey(value any, name string) []string {
	excluded := map[string]bool{}
	for _, key := range excludedSubtrees {
		excluded[key] = true
	}
	return collectKeyExcluding(value, name, excluded)
}

func collectKeyExcluding(value any, name string, excluded map[string]bool) []string {
	switch typed := value.(type) {
	case map[string]any:
		keys := []string{}
		if _, found := typed[name]; found {
			keys = append(keys, name)
		}
		for key, nested := range typed {
			if excluded[key] {
				continue
			}
			keys = append(keys, collectKeyExcluding(nested, name, excluded)...)
		}
		return keys
	case []any:
		keys := []string{}
		for _, nested := range typed {
			keys = append(keys, collectKeyExcluding(nested, name, excluded)...)
		}
		return keys
	default:
		return nil
	}
}
