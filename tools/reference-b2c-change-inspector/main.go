// Command reference-b2c-change-inspector is a deliberately limited, read-only
// reference for #3509. It does not analyze arbitrary frontend source or grant
// release permission. Its source inventory is caller-declared, not proven.
package main

import (
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
	if len(in.Changed) == 0 {
		return report{}, errors.New("explicit changed subjects are required")
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
	for _, e := range in.Edges {
		if e.From == "" || e.To == "" {
			return report{}, errors.New("edge endpoints must not be empty")
		}
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
	// Traverse only upstream from Required: following downstream siblings
	// would unnecessarily invalidate independent consumer questions.
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
			continue
		}
		upstreamQueue = append(upstreamQueue, incoming[id]...)
	}

	visited := map[string]bool{}
	lanes := map[string]bool{}
	queue := append([]string(nil), in.Changed...)
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
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
		queue = append(queue, outgoing[id]...)
	}

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
	dec := json.NewDecoder(io.LimitReader(r, 1<<20))
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
