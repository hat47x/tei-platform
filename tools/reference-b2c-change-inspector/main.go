// Command reference-b2c-change-inspector is a deliberately limited, read-only
// reference for #3509. It does not analyze arbitrary frontend source or grant
// release permission. Its source inventory is caller-declared, not proven.
package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
)

const inputSchema = "tei.reference.b2c-change-input/v0"
const reportSchema = "tei.reference.b2c-change-report/v0"

type node struct {
	ID    string `json:"id"`
	Layer string `json:"layer"`
}

type edge struct {
	From string `json:"from"`
	To   string `json:"to"`
}

type input struct {
	Schema              string   `json:"schema"`
	Scope               string   `json:"scope"`
	BaseRevision        string   `json:"baseRevision"`
	CandidateRevision   string   `json:"candidateRevision"`
	DeclaredScopeClosed bool     `json:"declaredScopeClosed"`
	Nodes               []node   `json:"nodes"`
	Edges               []edge   `json:"edges"`
	BaseOnlyEdges       []edge   `json:"baseOnlyEdges,omitempty"`
	Changed             []string `json:"changed"`
	Required            []string `json:"required"`
	Unobserved          []string `json:"unobserved"`
}

type report struct {
	Schema            string   `json:"schema"`
	Scope             string   `json:"scope"`
	BaseRevision      string   `json:"baseRevision"`
	CandidateRevision string   `json:"candidateRevision"`
	Impacted          []string `json:"impacted"`
	Lanes             []string `json:"lanes"`
	Unresolved        []string `json:"unresolved"`
	Coverage          string   `json:"coverage"`
	Disposition       string   `json:"disposition"`
	AuthorityLimits   []string `json:"authorityLimits"`
}

var laneForLayer = map[string]string{
	"content": "C0", "presentation": "C1", "interaction": "C2",
	"business": "C3", "consent": "C3", "release": "C4",
	"measurement": "C4", "unknown": "",
}

func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// inspect computes only the reachability of edges that the input explicitly
// declares. A claim that the inventory is closed is never independently proven.
func inspect(in input) (report, error) {
	if in.Schema != inputSchema {
		return report{}, fmt.Errorf("unsupported input schema %q", in.Schema)
	}
	if in.Scope == "" || in.BaseRevision == "" || in.CandidateRevision == "" {
		return report{}, errors.New("scope and both revisions are required")
	}
	if len(in.Changed) == 0 && len(in.BaseOnlyEdges) == 0 {
		return report{}, errors.New("explicit changed subjects or base-only edges are required")
	}
	byID := make(map[string]node, len(in.Nodes))
	for _, n := range in.Nodes {
		if n.ID == "" {
			return report{}, errors.New("node id is empty")
		}
		if _, exists := byID[n.ID]; exists {
			return report{}, fmt.Errorf("duplicate node id %q", n.ID)
		}
		if _, valid := laneForLayer[n.Layer]; !valid {
			return report{}, fmt.Errorf("unsupported layer %q", n.Layer)
		}
		byID[n.ID] = n
	}
	outgoing := map[string][]string{}
	incoming := map[string][]string{}
	missing := map[string]bool{}
	// Required is the caller's question-bound universe, not every node
	// available in a Workspace. A complete universe is only asserted by
	// the caller; this reference cannot prove it independently.
	required := map[string]bool{}
	for _, id := range in.Required {
		if strings.TrimSpace(id) == "" {
			return report{}, errors.New("required subject must not be empty")
		}
		required[id] = true
		if n, exists := byID[id]; !exists || n.Layer == "unknown" {
			missing[id] = true
		}
	}
	candidateEdges := map[edge]bool{}
	for _, e := range in.Edges {
		if e.From == "" || e.To == "" {
			return report{}, errors.New("edge endpoints must not be empty")
		}
		candidateEdges[e] = true
		outgoing[e.From] = append(outgoing[e.From], e.To)
		incoming[e.To] = append(incoming[e.To], e.From)
		// Unknown edge endpoints matter if they touch the explicitly
		// required universe. Unrelated malformed source inventories
		// must not force a full requalification of this question.
		if required[e.From] {
			if n, exists := byID[e.To]; !exists || n.Layer == "unknown" {
				missing[e.To] = true
			}
		}
		if required[e.To] {
			if n, exists := byID[e.From]; !exists || n.Layer == "unknown" {
				missing[e.From] = true
			}
		}
	}
	// The caller-declared R0-only edges are not present in candidate
	// reachability. Keep their reverse links for the question's Required
	// universe, not as a request to invalidate all unrelated consumers.
	baseOnlyIncoming := map[string][]string{}
	seenBaseOnly := map[edge]bool{}
	for _, e := range in.BaseOnlyEdges {
		if strings.TrimSpace(e.From) == "" || strings.TrimSpace(e.To) == "" {
			return report{}, errors.New("base-only edge endpoints must not be empty")
		}
		if candidateEdges[e] {
			return report{}, fmt.Errorf("base-only edge %q -> %q also occurs in candidate", e.From, e.To)
		}
		if seenBaseOnly[e] {
			return report{}, fmt.Errorf("duplicate base-only edge %q -> %q", e.From, e.To)
		}
		seenBaseOnly[e] = true
		baseOnlyIncoming[e.To] = append(baseOnlyIncoming[e.To], e.From)
	}
	for _, id := range in.Unobserved {
		if strings.TrimSpace(id) == "" {
			return report{}, errors.New("unobserved subject must not be empty")
		}
		missing[id] = true
	}

	// Edges point from a source toward its dependents. When a Required
	// consumer depends on a known intermediate node, unknown inputs further
	// upstream are still gaps in that requested consumer's declared closure.
	// The previous direct-neighbor check alone silently missed such gaps.
	// Traverse the union of R0-only and R1 candidate upstream edges.
	// This retains an entire deleted chain toward Required while excluding
	// unrelated deleted branches and downstream sibling consumers.
	seenUpstream := map[string]bool{}
	upstreamQueue := append([]string(nil), in.Required...)
	for len(upstreamQueue) > 0 {
		id := upstreamQueue[0]
		upstreamQueue = upstreamQueue[1:]
		if seenUpstream[id] {
			continue
		}
		seenUpstream[id] = true
		n, exists := byID[id]
		if !exists || n.Layer == "unknown" {
			missing[id] = true
		}
		upstreamQueue = append(upstreamQueue, incoming[id]...)
		upstreamQueue = append(upstreamQueue, baseOnlyIncoming[id]...)
	}

	// A deleted edge changes its old dependent only within the selected
	// consumer's declared dependency universe (R0 union R1).
	baseOnlyDependents := []string{}
	for _, e := range in.BaseOnlyEdges {
		if seenUpstream[e.To] {
			baseOnlyDependents = append(baseOnlyDependents, e.To)
		}
	}

	visited := map[string]bool{}
	lanes := map[string]bool{}
	walk := func(queue []string, requiredOnly bool) {
		for len(queue) > 0 {
			id := queue[0]
			queue = queue[1:]
			if requiredOnly && !seenUpstream[id] {
				continue
			}
			if visited[id] {
				continue
			}
			visited[id] = true
			n, exists := byID[id]
			if !exists {
				missing[id] = true
				continue
			}
			if lane := laneForLayer[n.Layer]; lane != "" {
				lanes[lane] = true
			} else {
				missing[id] = true
			}
			for _, dependent := range outgoing[id] {
				if !requiredOnly || seenUpstream[dependent] {
					queue = append(queue, dependent)
				}
			}
		}
	}
	// Explicit changes retain their ordinary forward-reachability meaning.
	walk(append([]string(nil), in.Changed...), false)
	// An R0-only change, unlike an explicit Changed node, is relevant only
	// to the selected consumer's input closure. Do not chase R1 siblings.
	walk(baseOnlyDependents, true)

	impacted := map[string]bool{}
	for id := range visited {
		if _, ok := byID[id]; ok {
			impacted[id] = true
		}
	}
	complete := in.DeclaredScopeClosed && len(in.Required) > 0 && len(missing) == 0
	coverage := "partial-or-unverified"
	if complete {
		coverage = "declared-closed-only"
	}
	disposition := "incomplete-scope-needs-review"
	if lanes["C2"] || lanes["C3"] || lanes["C4"] {
		disposition = "known-interaction-business-or-release-impact"
	} else if complete {
		disposition = "no-known-critical-impact-within-declared-scope"
	}

	return report{
		Schema: reportSchema, Scope: in.Scope,
		BaseRevision: in.BaseRevision, CandidateRevision: in.CandidateRevision,
		Impacted: sortedKeys(impacted), Lanes: sortedKeys(lanes),
		Unresolved: sortedKeys(missing), Coverage: coverage, Disposition: disposition,
		AuthorityLimits: []string{
			"input-inventory-and-closure-are-unverified-claims",
			"native-frontend-source-and-external-services-not-inspected",
			"no-release-grant-effect-or-conversion-claim",
		},
	}, nil
}

func run(r io.Reader, w io.Writer) error {
	// A bounded, pre-decoding key check prevents later JSON keys from
	// silently overriding scope/closure/changed/required declarations.
	const maxInputBytes = 1 << 20
	data, err := io.ReadAll(io.LimitReader(r, maxInputBytes+1))
	if err != nil {
		return err
	}
	if len(data) > maxInputBytes {
		return errors.New("B2C reference input exceeds the 1 MiB limit")
	}
	if err := inspectJSONKeys(data); err != nil {
		return err
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	var in input
	if err := dec.Decode(&in); err != nil {
		return err
	}
	var trailing any
	if err := dec.Decode(&trailing); err != io.EOF {
		if err == nil {
			return errors.New("trailing JSON value")
		}
		return fmt.Errorf("trailing input: %w", err)
	}
	out, err := inspect(in)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

func main() {
	if err := run(os.Stdin, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, "b2c reference inspection rejected:", err)
		os.Exit(1)
	}
}
