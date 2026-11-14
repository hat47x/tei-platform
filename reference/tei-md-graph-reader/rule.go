package teimdreader

import (
	"fmt"
	"go/format"
	"sort"
	"strconv"
	"strings"
)

// GraphRule is a *candidate* read from an explicitly headed Markdown table.
// It has no Workspace Selection, Meaning Acceptance or Host Authority.
type GraphRule struct {
	ID              string
	Endpoint        string // "from" or "to"; selected by the authored rule's condition.
	IDSource        Span
	ConditionSource Span
}

type RuleCandidate struct {
	Graph Candidate
	Rule  GraphRule
}

// ParseRuleCandidate is deliberately restricted to the fixed authoring
// pressure fixture. An arbitrary Markdown table or prose is never a Rule.
func ParseRuleCandidate(source []byte) (RuleCandidate, error) {
	graph, err := ParseCandidate(source)
	if err != nil {
		return RuleCandidate{}, err
	}
	const section = "## 算定規則の定義"
	selected, fence := false, false
	count, rows := 0, 0
	var rule GraphRule
	for _, ln := range sourceLines(source) {
		line := strings.TrimSpace(ln.text)
		if ln.number == 1 {
			line = strings.TrimPrefix(line, "\ufeff")
		}
		if strings.HasPrefix(line, ">") {
			continue
		}
		if strings.HasPrefix(line, "```") {
			fence = !fence
			selected = false
			continue
		}
		if fence {
			continue
		}
		if strings.HasPrefix(line, "#") {
			selected = line == section
			if selected {
				count++
				if count > 1 {
					return RuleCandidate{}, &Diagnostic{Code: "DUPLICATE_RULE_SECTION", At: Span{Start: ln.start, End: ln.start + len(ln.text), Line: ln.number}, Detail: section}
				}
			}
			continue
		}
		if !selected || !strings.HasPrefix(line, "|") {
			continue
		}
		cells, e := tableCells(ln)
		if e != nil {
			return RuleCandidate{}, e
		}
		switch rows {
		case 0:
			if !headerMatches(cells, "Rule ID", "対象", "条件", "出力", "順序") {
				return RuleCandidate{}, &Diagnostic{Code: "UNSUPPORTED_RULE_HEADER", At: cells[0].span, Detail: "expected Rule ID, 対象, 条件, 出力, 順序"}
			}
		case 1:
			if !isSeparator(cells) {
				return RuleCandidate{}, &Diagnostic{Code: "INVALID_RULE_SEPARATOR", At: cells[0].span, Detail: "expected Markdown table separator"}
			}
		case 2:
			if len(cells) != 5 {
				return RuleCandidate{}, &Diagnostic{Code: "INVALID_RULE_ROW", At: cells[0].span, Detail: "five cells required"}
			}
			if cells[0].value != "node.edge_ids" || cells[1].value != "対象ノード" || cells[3].value != "Edge ID一覧" || cells[4].value != "Edge ID昇順" {
				return RuleCandidate{}, &Diagnostic{Code: "UNSUPPORTED_RULE", At: cells[0].span, Detail: "unbound rule ID, subject, output or order"}
			}
			endpoint := ""
			switch cells[2].value {
			case "起点 = 対象ノード":
				endpoint = "from"
			case "終点 = 対象ノード":
				endpoint = "to"
			default:
				return RuleCandidate{}, &Diagnostic{Code: "UNSUPPORTED_RULE_CONDITION", At: cells[2].span, Detail: "only the explicit source/target condition is supported"}
			}
			rule = GraphRule{ID: cells[0].value, Endpoint: endpoint, IDSource: cells[0].span, ConditionSource: cells[2].span}
		default:
			return RuleCandidate{}, &Diagnostic{Code: "MULTIPLE_RULE_ROWS", At: cells[0].span, Detail: "one explicitly selected rule only"}
		}
		rows++
	}
	if count != 1 || rows != 3 {
		return RuleCandidate{}, &Diagnostic{Code: "RULE_NOT_BOUND", Detail: "explicit rule table with one row required"}
	}
	return RuleCandidate{Graph: graph, Rule: rule}, nil
}

// GenerateGo emits actual Go control logic from a *parsed* rule condition,
// rather than emitting a precomputed JSON result. The generated code still
// requires an independently qualified Build/Provider before product use.
func GenerateGo(c RuleCandidate) ([]byte, error) {
	if c.Rule.ID != "node.edge_ids" || (c.Rule.Endpoint != "from" && c.Rule.Endpoint != "to") {
		return nil, fmt.Errorf("unsupported rule candidate")
	}
	nodes := make([]string, 0, len(c.Graph.Nodes))
	nodeSet := map[string]bool{}
	for _, n := range c.Graph.Nodes {
		if n.ID == "" || nodeSet[n.ID] {
			return nil, fmt.Errorf("invalid node ID")
		}
		nodeSet[n.ID] = true
		nodes = append(nodes, n.ID)
	}
	if len(nodes) == 0 {
		return nil, fmt.Errorf("no nodes")
	}
	edges := append([]Edge(nil), c.Graph.Edges...)
	edgeSet := map[string]bool{}
	for _, e := range edges {
		if e.ID == "" || edgeSet[e.ID] || !nodeSet[e.From] || !nodeSet[e.To] {
			return nil, fmt.Errorf("invalid edge graph")
		}
		edgeSet[e.ID] = true
	}
	sort.Strings(nodes)
	sort.Slice(edges, func(i, j int) bool { return edges[i].ID < edges[j].ID })
	var b strings.Builder
	b.WriteString("// Code generated from a restricted Markdown Graph/Rule *candidate*. DO NOT EDIT.\n")
	b.WriteString("package main\n\nimport (\"encoding/json\"; \"os\"; \"sort\")\n")
	b.WriteString("type edge struct { id, from, to string }\n")
	b.WriteString("var nodes = []string{")
	for _, n := range nodes {
		b.WriteString(strconv.Quote(n))
		b.WriteByte(',')
	}
	b.WriteString("}\nvar edges = []edge{\n")
	for _, e := range edges {
		b.WriteString("{id:")
		b.WriteString(strconv.Quote(e.ID))
		b.WriteString(",from:")
		b.WriteString(strconv.Quote(e.From))
		b.WriteString(",to:")
		b.WriteString(strconv.Quote(e.To))
		b.WriteString("},\n")
	}
	b.WriteString("}\nfunc NodeEdgeIDs(nodeID string) []string {\n ids := []string{}\n for _,e := range edges {\n if ")
	if c.Rule.Endpoint == "from" {
		b.WriteString("e.from == nodeID")
	} else {
		b.WriteString("e.to == nodeID")
	}
	b.WriteString(" { ids = append(ids,e.id) }\n }\n sort.Strings(ids)\n return ids\n}\n")
	b.WriteString("func main(){ answer:=map[string][]string{};for _,node:=range nodes { answer[node]=NodeEdgeIDs(node) };if err:=json.NewEncoder(os.Stdout).Encode(answer);err!=nil { panic(err) } }\n")
	generated, err := format.Source([]byte(b.String()))
	if err != nil {
		return nil, fmt.Errorf("generated Go syntax: %w", err)
	}
	return generated, nil
}
