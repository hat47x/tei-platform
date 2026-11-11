package partition

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strconv"
	"strings"
	"time"
)

// ProjectedFunction is a function assigned to a subsystem with its operations.
type ProjectedFunction struct {
	Function string   `json:"function"`
	Writes   []string `json:"writes,omitempty"`
	Reads    []string `json:"reads,omitempty"`
}

// ProjectedDependency is one delegation edge visible from a subsystem.
// An invocation is not a direct write; it is recorded as a projection only.
type ProjectedDependency struct {
	Caller          string `json:"caller"`
	Callee          string `json:"callee"`
	CalleeSubsystem string `json:"calleeSubsystem,omitempty"`
	Internal        bool   `json:"internal"`
	Pattern         string `json:"pattern,omitempty"`
	SourceLocator   string `json:"sourceLocator,omitempty"`
}

// ProjectedSubsystem is the ER/CRUD/dependency projection for one subsystem.
type ProjectedSubsystem struct {
	Subsystem    string                `json:"subsystem"`
	Functions    []ProjectedFunction   `json:"functions"`
	DataIDs      []string              `json:"dataIds,omitempty"`
	Dependencies []ProjectedDependency `json:"dependencies,omitempty"`
}

// ProjectionReport is a deterministic per-subsystem projection of write/read/dependency edges.
type ProjectionReport struct {
	SchemaVersion  string               `json:"schemaVersion"`
	GeneratedAt    string               `json:"generatedAt"`
	BaseCommit     string               `json:"baseCommit"`
	AuthorityLimit string               `json:"authorityLimit"`
	Subsystems     []ProjectedSubsystem `json:"subsystems"`
}

// ProjectionInput carries one candidate and its supplied graph for all S7 projections.
// A nil PublicSlice means every supplied subject is in scope; callers serving
// low-authority consumers must set it explicitly.
type ProjectionInput struct {
	CandidateID   string
	Assignments   []Assignment
	WriteEdges    []AccessOccurrence
	ReadEdges     []ReadEdge
	Invocations   []InvocationEdge
	Functions     []string
	DataIDs       []string
	ExternalOwned map[string]bool
	PublicSlice   map[string]bool
	BaseCommit    string
	GeneratedAt   string
}

func (in ProjectionInput) assignmentMap() map[string]string {
	out := make(map[string]string, len(in.Assignments))
	for _, a := range in.Assignments {
		out[a.Function] = a.Subsystem
	}
	return out
}

func (in ProjectionInput) stamp() (string, string) {
	generatedAt := in.GeneratedAt
	if generatedAt == "" {
		generatedAt = time.Now().UTC().Format(time.RFC3339)
	}
	return generatedAt, in.BaseCommit
}

// BuildProjectionReport deterministically constructs per-subsystem projections from an assignment.
//
// The projection is derived-only and read-only:
//   - DataIDs lists the caller-supplied data identities touched by this subsystem's functions.
//   - Dependencies lists invocation edges whose caller belongs to this subsystem. It is a
//     projection of delegation, not evidence of a direct write and not accepted ownership.
//
// Callers must still apply a disclosure-safe Public Slice before returning this to any consumer.
func BuildProjectionReport(assign map[string]string, writes []Write, readEdges []ReadEdge, invocations []InvocationEdge, functions []string, dataIDs []string) ProjectionReport {
	subsystemMap := map[string]*ProjectedSubsystem{}
	for _, fn := range functions {
		sub, ok := assign[fn]
		if !ok {
			continue
		}
		if _, ok := subsystemMap[sub]; !ok {
			subsystemMap[sub] = &ProjectedSubsystem{Subsystem: sub}
		}
		subsystemMap[sub].Functions = append(subsystemMap[sub].Functions, ProjectedFunction{Function: fn})
	}
	touched := map[string]map[string]bool{}
	for _, w := range writes {
		sub, ok := assign[w.Function]
		if !ok || subsystemMap[sub] == nil {
			continue
		}
		if touched[sub] == nil {
			touched[sub] = map[string]bool{}
		}
		touched[sub][w.Data] = true
		fnIdx := indexOfFunction(subsystemMap[sub].Functions, w.Function)
		if fnIdx < 0 {
			continue
		}
		subsystemMap[sub].Functions[fnIdx].Writes = append(subsystemMap[sub].Functions[fnIdx].Writes, w.Data)
	}
	for _, r := range readEdges {
		sub, ok := assign[r.Function]
		if !ok || subsystemMap[sub] == nil {
			continue
		}
		if touched[sub] == nil {
			touched[sub] = map[string]bool{}
		}
		touched[sub][r.Data] = true
		fnIdx := indexOfFunction(subsystemMap[sub].Functions, r.Function)
		if fnIdx < 0 {
			continue
		}
		subsystemMap[sub].Functions[fnIdx].Reads = append(subsystemMap[sub].Functions[fnIdx].Reads, r.Data)
	}
	for _, inv := range invocations {
		sub, ok := assign[inv.Caller]
		if !ok || subsystemMap[sub] == nil {
			continue
		}
		calleeSub, assigned := assign[inv.Callee]
		if !assigned {
			calleeSub = ""
		}
		subsystemMap[sub].Dependencies = append(subsystemMap[sub].Dependencies, ProjectedDependency{
			Caller: inv.Caller, Callee: inv.Callee, CalleeSubsystem: calleeSub,
			Internal: assigned && calleeSub == sub, Pattern: inv.Pattern, SourceLocator: inv.SourceLocator,
		})
	}
	subsystems := make([]ProjectedSubsystem, 0, len(subsystemMap))
	for _, sub := range subsystemMap {
		sort.Slice(sub.Functions, func(i, j int) bool { return sub.Functions[i].Function < sub.Functions[j].Function })
		for _, fn := range sub.Functions {
			sort.Strings(fn.Writes)
			sort.Strings(fn.Reads)
		}
		dataSet := touched[sub.Subsystem]
		if dataSet != nil && len(dataIDs) > 0 {
			ids := make([]string, 0, len(dataSet))
			for _, d := range dataIDs {
				if dataSet[d] {
					ids = append(ids, d)
				}
			}
			sub.DataIDs = ids
		}
		sort.Strings(sub.DataIDs)
		sort.Slice(sub.Dependencies, func(i, j int) bool {
			if sub.Dependencies[i].Caller != sub.Dependencies[j].Caller {
				return sub.Dependencies[i].Caller < sub.Dependencies[j].Caller
			}
			return sub.Dependencies[i].Callee < sub.Dependencies[j].Callee
		})
		subsystems = append(subsystems, *sub)
	}
	sort.Slice(subsystems, func(i, j int) bool { return subsystems[i].Subsystem < subsystems[j].Subsystem })
	return ProjectionReport{
		SchemaVersion:  "tei.subsystem-partition.projection/v0",
		GeneratedAt:    "",
		BaseCommit:     "",
		AuthorityLimit: AuthorityLimit,
		Subsystems:     subsystems,
	}
}

func indexOfFunction(fns []ProjectedFunction, id string) int {
	for i, f := range fns {
		if f.Function == id {
			return i
		}
	}
	return -1
}

// ERNode is one node in a subsystem ER view. External nodes are preserved, never deleted.
type ERNode struct {
	DataID       string   `json:"dataId"`
	Ownership    string   `json:"ownership"` // internal | external
	ReferencedBy []string `json:"referencedBy,omitempty"`
	Disclosure   string   `json:"disclosure,omitempty"` // disclosed | non-disclosed
	ReasonCode   string   `json:"reasonCode,omitempty"`
}

// SubsystemERView is the ER projection for one subsystem.
type SubsystemERView struct {
	Subsystem string   `json:"subsystem"`
	Nodes     []ERNode `json:"nodes"`
}

// NonDisclosedEntry records a subject withheld from a low-authority consumer.
// The subject is marked, not removed, so the count stays auditable.
type NonDisclosedEntry struct {
	SubjectID  string `json:"subjectId"`
	Kind       string `json:"kind"` // data | function
	ReasonCode string `json:"reasonCode"`
}

// ERProjection is the per-subsystem ER view with external nodes preserved.
type ERProjection struct {
	SchemaVersion  string              `json:"schemaVersion"`
	GeneratedAt    string              `json:"generatedAt"`
	BaseCommit     string              `json:"baseCommit"`
	AuthorityLimit string              `json:"authorityLimit"`
	CandidateID    string              `json:"candidateId"`
	Subsystems     []SubsystemERView   `json:"subsystems"`
	NonDisclosed   []NonDisclosedEntry `json:"nonDisclosed,omitempty"`
	Digest         string              `json:"digest"`
}

// CRUDCell is one Function×Data entry with origins and undetermined marking.
type CRUDCell struct {
	Function     string   `json:"function"`
	DataID       string   `json:"dataId"`
	Operations   []string `json:"operations"` // subset of C,R,U,D
	Provenance   []string `json:"provenance"`
	Undetermined bool     `json:"undetermined"`
}

// CRUDMatrix is the whole-world Function×Data CRUD table.
type CRUDMatrix struct {
	SchemaVersion  string     `json:"schemaVersion"`
	GeneratedAt    string     `json:"generatedAt"`
	BaseCommit     string     `json:"baseCommit"`
	AuthorityLimit string     `json:"authorityLimit"`
	CandidateID    string     `json:"candidateId"`
	Cells          []CRUDCell `json:"cells"`
	Digest         string     `json:"digest"`
}

// DependencyEdge is one typed edge in the dependency graph. Kinds are never merged.
type DependencyEdge struct {
	Kind            string `json:"kind"` // invocation | function-data-write | data-data-relation | external-boundary
	From            string `json:"from"`
	To              string `json:"to"`
	Subsystem       string `json:"subsystem,omitempty"`
	TargetSubsystem string `json:"targetSubsystem,omitempty"`
	Operation       string `json:"operation,omitempty"`
	Provenance      string `json:"provenance,omitempty"`
	OccurrenceID    string `json:"occurrenceId,omitempty"`
	External        bool   `json:"external,omitempty"`
}

// DependencyGraph is the typed dependency projection for one candidate.
type DependencyGraph struct {
	SchemaVersion  string           `json:"schemaVersion"`
	GeneratedAt    string           `json:"generatedAt"`
	BaseCommit     string           `json:"baseCommit"`
	AuthorityLimit string           `json:"authorityLimit"`
	CandidateID    string           `json:"candidateId"`
	Edges          []DependencyEdge `json:"edges"`
	Notes          []string         `json:"notes,omitempty"`
	Digest         string           `json:"digest"`
}

// BuildERProjection builds the per-subsystem ER view. External nodes are preserved with
// ownership "external" and the referencing occurrence ids; subjects outside the public
// slice are marked non-disclosed rather than removed.
func BuildERProjection(in ProjectionInput) ERProjection {
	generatedAt, baseCommit := in.stamp()
	assign := in.assignmentMap()
	ownerByData := map[string]string{}
	for _, w := range in.WriteEdges {
		if strings.TrimSpace(w.Owner) != "" {
			ownerByData[w.Data] = strings.TrimSpace(w.Owner)
		}
	}
	type nodeAcc struct {
		dataID   string
		refs     map[string]bool
		external bool
	}
	perSub := map[string]map[string]*nodeAcc{}
	ensure := func(sub, dataID string) *nodeAcc {
		if dataID == "" {
			return nil
		}
		if perSub[sub] == nil {
			perSub[sub] = map[string]*nodeAcc{}
		}
		if perSub[sub][dataID] == nil {
			perSub[sub][dataID] = &nodeAcc{dataID: dataID, refs: map[string]bool{}}
		}
		return perSub[sub][dataID]
	}
	for _, fn := range in.Functions {
		if sub, ok := assign[fn]; ok {
			if perSub[sub] == nil {
				perSub[sub] = map[string]*nodeAcc{}
			}
		}
	}
	for _, w := range in.WriteEdges {
		sub, ok := assign[w.Function]
		if !ok {
			continue
		}
		node := ensure(sub, w.Data)
		if node == nil {
			continue
		}
		node.refs[w.ID] = true
		owner := strings.TrimSpace(w.Owner)
		if owner != "" && owner != sub {
			node.external = true
		}
		if in.ExternalOwned[w.Data] {
			node.external = true
		}
	}
	for _, r := range in.ReadEdges {
		sub, ok := assign[r.Function]
		if !ok {
			continue
		}
		node := ensure(sub, r.Data)
		if node == nil {
			continue
		}
		node.refs[r.ID] = true
		if owner, ok := ownerByData[r.Data]; ok && owner != sub {
			node.external = true
		}
		if in.ExternalOwned[r.Data] {
			node.external = true
		}
	}
	allowed := func(id string) bool {
		if in.PublicSlice == nil {
			return true
		}
		return in.PublicSlice[id]
	}
	nonDisclosed := make([]NonDisclosedEntry, 0)
	subsystems := make([]SubsystemERView, 0, len(perSub))
	for sub, nodes := range perSub {
		view := SubsystemERView{Subsystem: sub}
		ids := make([]string, 0, len(nodes))
		for id := range nodes {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		for _, id := range ids {
			n := nodes[id]
			refs := make([]string, 0, len(n.refs))
			for r := range n.refs {
				refs = append(refs, r)
			}
			sort.Strings(refs)
			ownership := "internal"
			if n.external {
				ownership = "external"
			}
			node := ERNode{DataID: id, Ownership: ownership, ReferencedBy: refs, Disclosure: "disclosed"}
			if !allowed(id) {
				node.Disclosure = "non-disclosed"
				node.ReasonCode = "not-in-public-slice"
				nonDisclosed = append(nonDisclosed, NonDisclosedEntry{SubjectID: id, Kind: "data", ReasonCode: "not-in-public-slice"})
			}
			view.Nodes = append(view.Nodes, node)
		}
		subsystems = append(subsystems, view)
	}
	sort.Slice(subsystems, func(i, j int) bool { return subsystems[i].Subsystem < subsystems[j].Subsystem })
	sort.Slice(nonDisclosed, func(i, j int) bool {
		if nonDisclosed[i].SubjectID != nonDisclosed[j].SubjectID {
			return nonDisclosed[i].SubjectID < nonDisclosed[j].SubjectID
		}
		return nonDisclosed[i].ReasonCode < nonDisclosed[j].ReasonCode
	})
	return ERProjection{
		SchemaVersion:  "tei.subsystem-partition.er-projection/v0",
		GeneratedAt:    generatedAt,
		BaseCommit:     baseCommit,
		AuthorityLimit: AuthorityLimit,
		CandidateID:    in.CandidateID,
		Subsystems:     subsystems,
		NonDisclosed:   nonDisclosed,
		Digest:         in.digest(),
	}
}

// BuildCRUDMatrix builds the whole-world Function×Data CRUD table with origins.
// Undetermined provenance is marked, never rendered as "no operation".
func BuildCRUDMatrix(in ProjectionInput) CRUDMatrix {
	generatedAt, baseCommit := in.stamp()
	type key struct{ fn, data string }
	cells := map[key]*CRUDCell{}
	add := func(fn, data, op, prov string) {
		if fn == "" || data == "" {
			return
		}
		k := key{fn, data}
		c, ok := cells[k]
		if !ok {
			c = &CRUDCell{Function: fn, DataID: data}
			cells[k] = c
		}
		if op != "" && !containsString(c.Operations, op) {
			c.Operations = append(c.Operations, op)
		}
		if prov != "" && !containsString(c.Provenance, prov) {
			c.Provenance = append(c.Provenance, prov)
		}
		if prov == string(ProvUnresolved) || prov == string(ProvNotExamined) {
			c.Undetermined = true
		}
	}
	for _, w := range in.WriteEdges {
		add(w.Function, w.Data, strings.TrimSpace(string(w.Operation)), string(w.Provenance))
	}
	for _, r := range in.ReadEdges {
		add(r.Function, r.Data, "R", string(r.Provenance))
	}
	out := make([]CRUDCell, 0, len(cells))
	for _, c := range cells {
		sort.Strings(c.Operations)
		sort.Strings(c.Provenance)
		out = append(out, *c)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Function != out[j].Function {
			return out[i].Function < out[j].Function
		}
		return out[i].DataID < out[j].DataID
	})
	return CRUDMatrix{
		SchemaVersion:  "tei.subsystem-partition.crud-matrix/v0",
		GeneratedAt:    generatedAt,
		BaseCommit:     baseCommit,
		AuthorityLimit: AuthorityLimit,
		CandidateID:    in.CandidateID,
		Cells:          out,
		Digest:         in.digest(),
	}
}

// BuildDependencyGraph builds the typed dependency graph. Delegation is never drawn as a
// cross-boundary direct CUD edge.
func BuildDependencyGraph(in ProjectionInput) DependencyGraph {
	generatedAt, baseCommit := in.stamp()
	assign := in.assignmentMap()
	edges := make([]DependencyEdge, 0)
	for _, inv := range in.Invocations {
		callerSub := assign[inv.Caller]
		calleeSub, assigned := assign[inv.Callee]
		if !assigned {
			calleeSub = ""
		}
		edges = append(edges, DependencyEdge{
			Kind: "invocation", From: inv.Caller, To: inv.Callee,
			Subsystem: callerSub, TargetSubsystem: calleeSub,
			Provenance: string(inv.Provenance), OccurrenceID: inv.SourceLocator,
		})
	}
	for _, w := range in.WriteEdges {
		sub := assign[w.Function]
		owner := strings.TrimSpace(w.Owner)
		edges = append(edges, DependencyEdge{
			Kind: "function-data-write", From: w.Function, To: w.Data,
			Subsystem: sub, TargetSubsystem: owner,
			Operation: strings.TrimSpace(string(w.Operation)), Provenance: string(w.Provenance),
			OccurrenceID: w.ID, External: owner != "" && owner != sub,
		})
	}
	for _, r := range in.ReadEdges {
		sub := assign[r.Function]
		if in.ExternalOwned[r.Data] {
			edges = append(edges, DependencyEdge{
				Kind: "external-boundary", From: r.Function, To: r.Data,
				Subsystem: sub, Provenance: string(r.Provenance),
				OccurrenceID: r.ID, External: true,
			})
		}
	}
	sort.Slice(edges, func(i, j int) bool {
		if edges[i].Kind != edges[j].Kind {
			return edges[i].Kind < edges[j].Kind
		}
		if edges[i].From != edges[j].From {
			return edges[i].From < edges[j].From
		}
		if edges[i].To != edges[j].To {
			return edges[i].To < edges[j].To
		}
		return edges[i].OccurrenceID < edges[j].OccurrenceID
	})
	notes := []string{
		"data-data-relation-edges-require-caller-supplied-relations",
		"delegation-is-not-a-cross-boundary-direct-cud-edge",
	}
	return DependencyGraph{
		SchemaVersion:  "tei.subsystem-partition.dependency-graph/v0",
		GeneratedAt:    generatedAt,
		BaseCommit:     baseCommit,
		AuthorityLimit: AuthorityLimit,
		CandidateID:    in.CandidateID,
		Edges:          edges,
		Notes:          notes,
		Digest:         in.digest(),
	}
}

func containsString(list []string, v string) bool {
	for _, s := range list {
		if s == v {
			return true
		}
	}
	return false
}

// digest derives a deterministic digest from the candidate id and the exact supplied graph,
// so ER/CRUD/dependency outputs can be shown to share one candidate.
func (in ProjectionInput) digest() string {
	var sb strings.Builder
	sb.WriteString("candidate=" + in.CandidateID + "\n")
	fns := make([]string, len(in.Functions))
	copy(fns, in.Functions)
	sort.Strings(fns)
	assign := in.assignmentMap()
	for _, f := range fns {
		sb.WriteString("fn=" + f + "->" + assign[f] + "\n")
	}
	ids := make([]string, len(in.DataIDs))
	copy(ids, in.DataIDs)
	sort.Strings(ids)
	for _, d := range ids {
		sb.WriteString("data=" + d + "\n")
	}
	ws := make([]AccessOccurrence, len(in.WriteEdges))
	copy(ws, in.WriteEdges)
	sort.Slice(ws, func(i, j int) bool { return ws[i].ID < ws[j].ID })
	for _, w := range ws {
		sb.WriteString("w=" + w.ID + "|" + w.Function + "|" + w.Data + "|" + string(w.Operation) + "|" + w.Owner + "|" + string(w.Provenance) + "|" + strconv.Itoa(w.Multiplicity) + "\n")
	}
	rs := make([]ReadEdge, len(in.ReadEdges))
	copy(rs, in.ReadEdges)
	sort.Slice(rs, func(i, j int) bool { return rs[i].ID < rs[j].ID })
	for _, r := range rs {
		sb.WriteString("r=" + r.ID + "|" + r.Function + "|" + r.Data + "|" + r.Provenance + "\n")
	}
	sum := sha256.Sum256([]byte(sb.String()))
	return "sha256:" + hex.EncodeToString(sum[:])
}
