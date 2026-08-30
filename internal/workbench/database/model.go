package databaseworkbench

import (
	"sort"

	"tei/internal/core"
	"tei/internal/database/plan"
	"tei/internal/index"
	"tei/internal/query"
)

// Model is a UI-oriented projection. It is derived from the cross-cutting TEI
// index and must never become a semantic or query source of truth.
type Model struct {
	Schemas []SchemaView `json:"schemas,omitempty"`
	Queries []QueryView  `json:"queries,omitempty"`
}

type SchemaView struct {
	Database core.CanonicalID `json:"database"`
	Tables   []TableView      `json:"tables,omitempty"`
}

type TableView struct {
	Name       string       `json:"name"`
	Kind       string       `json:"kind"`
	Columns    []ColumnView `json:"columns,omitempty"`
	PrimaryKey []string     `json:"primaryKey,omitempty"`
}

type ColumnView struct {
	Name         string `json:"name"`
	PhysicalType string `json:"physicalType,omitempty"`
	Nullable     bool   `json:"nullable"`
}

type QueryView struct {
	Query       core.CanonicalID       `json:"query"`
	Application core.CanonicalID       `json:"application"`
	Datasource  core.LocalName         `json:"datasource"`
	Database    core.CanonicalID       `json:"database"`
	Coverage    query.AnalysisCoverage `json:"coverage"`
	Scopes      []ScopeView            `json:"scopes,omitempty"`
	Graph       []GraphNode            `json:"graph,omitempty"`
	Fields      []FieldView            `json:"fields,omitempty"`
	Outputs     []OutputFieldView      `json:"outputs,omitempty"`
	Lineage     []LineageView          `json:"lineage,omitempty"`
	Plans       []ExecutionPlanView    `json:"plans,omitempty"`
}

type ExecutionPlanView struct {
	Mode            plan.CaptureMode         `json:"mode"`
	Provider        string                   `json:"provider"`
	Root            string                   `json:"root"`
	PlanningTimeMs  *float64                 `json:"planningTimeMs,omitempty"`
	ExecutionTimeMs *float64                 `json:"executionTimeMs,omitempty"`
	Nodes           []PlanNodeView           `json:"nodes,omitempty"`
	LogicalMappings []LogicalPlanMappingView `json:"logicalMappings,omitempty"`
}

type PlanNodeView struct {
	ID                string        `json:"id"`
	Kind              plan.NodeKind `json:"kind"`
	ProviderKind      string        `json:"providerKind"`
	Children          []string      `json:"children,omitempty"`
	PhysicalObject    string        `json:"physicalObject,omitempty"`
	Alias             string        `json:"alias,omitempty"`
	IndexName         string        `json:"indexName,omitempty"`
	EstimatedRows     *float64      `json:"estimatedRows,omitempty"`
	EstimatedCost     *float64      `json:"estimatedTotalCost,omitempty"`
	ActualRowsPerLoop *float64      `json:"actualRowsPerLoop,omitempty"`
	ActualLoops       *float64      `json:"actualLoops,omitempty"`
	ActualTimeMs      *float64      `json:"actualTotalTimeMs,omitempty"`
	RowEstimateRatio  *float64      `json:"rowEstimateRatio,omitempty"`
}

type LogicalPlanMappingView struct {
	RelNodeID  string                 `json:"relNodeId"`
	PlanNodeID string                 `json:"planNodeId"`
	Confidence plan.MappingConfidence `json:"confidence"`
	Reason     string                 `json:"reason,omitempty"`
}

type ScopeView struct {
	ID       string `json:"id"`
	ParentID string `json:"parentId,omitempty"`
	Kind     string `json:"kind"`
	Name     string `json:"name,omitempty"`
}

type GraphNode struct {
	ID         string   `json:"id"`
	Kind       string   `json:"kind"`
	ScopeID    string   `json:"scopeId,omitempty"`
	Inputs     []string `json:"inputs,omitempty"`
	Label      string   `json:"label,omitempty"`
	Resolution string   `json:"resolution,omitempty"`
}

type FieldView struct {
	ID                 query.FieldID `json:"id"`
	ScopeID            string        `json:"scopeId,omitempty"`
	Name               string        `json:"name"`
	Source             string        `json:"source,omitempty"`
	PhysicalSource     string        `json:"physicalSource,omitempty"`
	LogicalSourceField query.FieldID `json:"logicalSourceField,omitempty"`
	Resolution         string        `json:"resolution"`
	Correlated         bool          `json:"correlated,omitempty"`
	OuterScopeID       string        `json:"outerScopeId,omitempty"`
}

type OutputFieldView struct {
	ID         query.FieldID   `json:"id"`
	ScopeID    string          `json:"scopeId"`
	Name       string          `json:"name,omitempty"`
	Expression string          `json:"expression,omitempty"`
	Inputs     []query.FieldID `json:"inputs,omitempty"`
}

type LineageView struct {
	From query.FieldID `json:"from"`
	To   query.FieldID `json:"to"`
}

func Build(d index.Derived) Model {
	var m Model
	for _, s := range d.PhysicalSchemas {
		sv := SchemaView{Database: s.Database}
		for _, t := range s.Tables {
			tv := TableView{Name: t.QualifiedName(), Kind: string(t.Kind), PrimaryKey: append([]string(nil), t.PrimaryKey...)}
			for _, c := range t.Columns {
				tv.Columns = append(tv.Columns, ColumnView{Name: c.Name, PhysicalType: c.PhysicalType, Nullable: c.Nullable})
			}
			sv.Tables = append(sv.Tables, tv)
		}
		m.Schemas = append(m.Schemas, sv)
	}
	plansByQuery := map[core.CanonicalID][]index.ExecutionPlanEntry{}
	for _, p := range d.ExecutionPlans {
		plansByQuery[p.Plan.Query] = append(plansByQuery[p.Plan.Query], p)
	}
	for _, a := range d.AnalyzedQueries {
		qv := QueryView{Query: a.IR.Query, Application: a.Application, Datasource: a.IR.Datasource, Database: a.IR.Database, Coverage: a.IR.Coverage}
		for _, s := range a.IR.Scopes {
			qv.Scopes = append(qv.Scopes, ScopeView{ID: s.ID, ParentID: s.ParentID, Kind: s.Kind, Name: s.Name})
		}
		for _, n := range a.IR.Relational.Nodes {
			label := n.Source
			if label == "" && len(n.Expressions) > 0 {
				label = n.Expressions[0]
			}
			qv.Graph = append(qv.Graph, GraphNode{ID: n.ID, Kind: string(n.Kind), ScopeID: n.ScopeID, Inputs: append([]string(nil), n.Inputs...), Label: label, Resolution: n.Resolution})
		}
		for _, f := range a.IR.Fields {
			qv.Fields = append(qv.Fields, FieldView{ID: f.ID, ScopeID: f.ScopeID, Name: f.Name, Source: f.Source, PhysicalSource: f.PhysicalSource, LogicalSourceField: f.LogicalSourceField, Resolution: f.Resolution, Correlated: f.Correlated, OuterScopeID: f.OuterScopeID})
		}
		for _, f := range a.IR.OutputFields {
			qv.Outputs = append(qv.Outputs, OutputFieldView{ID: f.ID, ScopeID: f.ScopeID, Name: f.Name, Expression: f.Expression, Inputs: append([]query.FieldID(nil), f.Inputs...)})
		}
		for _, e := range a.IR.Lineage {
			qv.Lineage = append(qv.Lineage, LineageView{From: e.From, To: e.To})
		}
		for _, pe := range plansByQuery[a.IR.Query] {
			pv := ExecutionPlanView{Mode: pe.Plan.Mode, Provider: string(pe.Plan.Provider), Root: pe.Plan.Root, PlanningTimeMs: pe.Plan.PlanningTimeMs, ExecutionTimeMs: pe.Plan.ExecutionTimeMs}
			for _, n := range pe.Plan.Nodes {
				nv := PlanNodeView{ID: n.ID, Kind: n.Kind, ProviderKind: n.ProviderKind, Children: append([]string(nil), n.Children...), PhysicalObject: n.PhysicalObject, Alias: n.Alias, IndexName: n.IndexName, EstimatedRows: n.Estimate.Rows, EstimatedCost: n.Estimate.TotalCost}
				if n.Actual != nil {
					nv.ActualRowsPerLoop = n.Actual.Rows
					nv.ActualLoops = n.Actual.Loops
					nv.ActualTimeMs = n.Actual.TotalTimeMs
				}
				if ratio, ok := plan.RowEstimateRatio(n); ok {
					r := ratio
					nv.RowEstimateRatio = &r
				}
				pv.Nodes = append(pv.Nodes, nv)
			}
			for _, m := range pe.Mappings {
				pv.LogicalMappings = append(pv.LogicalMappings, LogicalPlanMappingView{RelNodeID: m.RelNodeID, PlanNodeID: m.PlanNodeID, Confidence: m.Confidence, Reason: m.Reason})
			}
			qv.Plans = append(qv.Plans, pv)
		}
		m.Queries = append(m.Queries, qv)
	}
	sort.Slice(m.Schemas, func(i, j int) bool { return m.Schemas[i].Database < m.Schemas[j].Database })
	sort.Slice(m.Queries, func(i, j int) bool { return m.Queries[i].Query < m.Queries[j].Query })
	return m
}
