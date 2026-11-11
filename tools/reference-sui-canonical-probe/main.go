// Command reference-sui-canonical-probe は、SUI の意味構造を `.tei` で
// 宣言したとき、どの区別が定義から観測できるかを read-only で報告する。
//
// この tool は Product runtime でも Canonical source でもない。instance を
// TEI Canonical object へ昇格させず、定義の解決結果と「宣言していないもの」
// だけを JSON で返す。判定は probe の観測に限られ、TEI Product Requirement
// の成立や Owner の意味受理を意味しない。
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"sort"

	"tei/internal/ir"
	"tei/internal/semantic"
	"tei/internal/workspace"
)

const (
	reportSchemaVersion = "sui-canonical-m2-probe/v2"
	// TEI authority is definition-only: this probe reads the World and never
	// writes it, and instance data stays outside the Canonical World.
	teiAuthority = "definition-only"
)

// undeclared lists the SUI structures this definition intentionally does not
// carry. Keeping them out is a decision, not an omission to be filled in later
// by inference.
var undeclared = []string{
	"tree.viewState (fold/focus/selection)",
	"canvas.transform (z-order/zoom/gesture)",
	"island.geometry (polygon/rect/placard)",
	"review.attribution (who/when)",
	"relation.rationale",
	"narrative.summary.critique",
	// The reference envelope never carries card text, island titles, or the
	// document title, so they are not declared as attributes. Whether text
	// may ever be declared is M4 / SUI #3369, not this definition.
	"card.text / island.title / document.title (content)",
}

type relationSummary struct {
	ID          string `json:"id"`
	Target      string `json:"target"`
	Cardinality string `json:"cardinality"`
}

type dataSummary struct {
	ID         string            `json:"id"`
	Attributes []string          `json:"attributes"`
	Relations  []relationSummary `json:"relations"`
}

type report struct {
	SchemaVersion string        `json:"schemaVersion"`
	TEIAuthority  string        `json:"teiAuthority"`
	Canonicalized bool          `json:"canonicalizedInstanceData"`
	InstanceCount int           `json:"instanceCount"`
	Data          []dataSummary `json:"data"`
	// ContainmentAndAffiliationSeparate is true only when both memberships
	// resolve as their own Data with their own occurrence identity.
	ContainmentAndAffiliationSeparate bool `json:"containmentAndAffiliationSeparate"`
	// ParallelEdgesDistinguishable is true when an edge carries its own
	// identity and its endpoints are kind+id attributes, so parallel edges are
	// not folded by their endpoint pair and island->island stays expressible.
	ParallelEdgesDistinguishable bool `json:"parallelEdgesDistinguishable"`
	// ReviewAndRevisionDeclared is true when the states the reference actually
	// carries (document revision / disclosure, card text review, island title
	// review) are declared. Card text itself is deliberately not declared: the
	// reference envelope never carries it.
	ReviewAndRevisionDeclared bool `json:"reviewAndRevisionDeclared"`
	// Undeclared lists what the definition does not represent. It is never
	// empty for this Stage 0 probe: the losses are part of the report.
	Undeclared []string `json:"undeclared"`
	// ParentPresenceSemantics stays unresolved: current Relation cardinality
	// is one|many only, so a nullable root parent is not proven lossless.
	ParentPresenceSemantics string `json:"parentPresenceSemantics"`
}

func main() {
	workspacePath := flag.String("workspace", "reference/sui-adoption/workspace-m2", "TEI M2 Reference Workspace")
	flag.Parse()

	result, err := run(*workspacePath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(result); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(workspacePath string) (report, error) {
	loaded, err := workspace.Load(workspacePath)
	if err != nil {
		return report{}, fmt.Errorf("TEI M2 Reference Workspace を読み込めません: %w", err)
	}
	return probe(loaded.World)
}

func probe(world ir.World) (report, error) {
	data := map[string]semantic.Data{}
	for _, ns := range world.Semantics {
		for _, d := range ns.Data {
			id := string(d.ID)
			if _, dup := data[id]; dup {
				return report{}, fmt.Errorf("semantic element %s is declared more than once", id)
			}
			data[id] = d
		}
	}

	required := []string{
		"sui.document",
		"sui.card",
		"sui.island",
		"sui.containment",
		"sui.affiliation",
		"sui.edge",
		"sui.island_parent",
	}
	for _, id := range required {
		if _, ok := data[id]; !ok {
			return report{}, fmt.Errorf("required definition %s is missing", id)
		}
	}

	// containment と affiliation が別の Data として存在し、それぞれが
	// 自分の occurrence ID 属性を持つことを確かめる。
	separate := hasAttribute(data["sui.containment"], "containment_id") &&
		hasAttribute(data["sui.affiliation"], "affiliation_id") &&
		string(data["sui.containment"].ID) != string(data["sui.affiliation"].ID)

	edge := data["sui.edge"]
	// 並行Edgeは端点の組で畳まない。endpointは card|island の多相なので、
	// 型付きrelationではなく kind+id 属性で宣言する（island->island も
	// 表現できるようにするため）。この制約を定義から隠さない。
	parallel := hasAttribute(edge, "edge_id") &&
		hasAttribute(edge, "edge_type") &&
		hasAttribute(edge, "from_kind") &&
		hasAttribute(edge, "from_id") &&
		hasAttribute(edge, "to_kind") &&
		hasAttribute(edge, "to_id") &&
		countRelations(edge, "from_card") == 0 &&
		countRelations(edge, "to_card") == 0

	summaries := make([]dataSummary, 0, len(data))
	ids := make([]string, 0, len(data))
	for id := range data {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		summary := dataSummary{ID: id, Attributes: []string{}, Relations: []relationSummary{}}
		for _, a := range data[id].Attributes {
			summary.Attributes = append(summary.Attributes, string(a.ID))
		}
		for _, r := range data[id].Relations {
			summary.Relations = append(summary.Relations, relationSummary{
				ID:          string(r.ID),
				Target:      string(r.Target),
				Cardinality: string(r.Cardinality),
			})
		}
		// Attributes and relations come from different collections; sort each
		// so the report is stable across runs without implying an order.
		sort.Strings(summary.Attributes)
		sort.Slice(summary.Relations, func(i, j int) bool {
			return summary.Relations[i].ID < summary.Relations[j].ID
		})
		summaries = append(summaries, summary)
	}

	reviewAndRevision := hasAttribute(data["sui.document"], "revision_state") &&
		hasAttribute(data["sui.document"], "disclosure_state") &&
		hasAttribute(data["sui.card"], "text_review_state") &&
		hasAttribute(data["sui.island"], "title_review_state")

	return report{
		SchemaVersion:                     reportSchemaVersion,
		TEIAuthority:                      teiAuthority,
		Canonicalized:                     false,
		InstanceCount:                     0,
		Data:                              summaries,
		ContainmentAndAffiliationSeparate: separate,
		ParallelEdgesDistinguishable:      parallel,
		ReviewAndRevisionDeclared:         reviewAndRevision,
		Undeclared:                        append([]string{}, undeclared...),
		ParentPresenceSemantics:           "unresolved",
	}, nil
}

func hasAttribute(d semantic.Data, symbol string) bool {
	for _, a := range d.Attributes {
		if string(a.ID) == string(d.ID)+"."+symbol {
			return true
		}
	}
	return false
}

func countRelations(d semantic.Data, symbol string) int {
	count := 0
	for _, r := range d.Relations {
		if string(r.ID) == string(d.ID)+"."+symbol {
			count++
		}
	}
	return count
}
