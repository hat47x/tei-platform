package query

import (
	"fmt"
	"sort"
	"strings"

	"tei/internal/core"
	"tei/internal/database/physical"
	"tei/internal/query/analyzer"
)

type CoverageState string

const (
	CoverageNone    CoverageState = "none"
	CoveragePartial CoverageState = "partial"
	CoverageFull    CoverageState = "full"
)

type AnalysisCoverage struct {
	Parse      CoverageState `json:"parse"`
	Resolution CoverageState `json:"resolution"`
	Lowering   CoverageState `json:"lowering"`
}

// QueryScope preserves lexical query boundaries rather than optimizer output.
type QueryScope struct {
	ID       string `json:"id"`
	ParentID string `json:"parentId,omitempty"`
	Kind     string `json:"kind"`
	Name     string `json:"name,omitempty"`
}

type SourceRef struct {
	ID         string           `json:"id,omitempty"`
	ScopeID    string           `json:"scopeId,omitempty"`
	Database   core.CanonicalID `json:"database"`
	Name       string           `json:"name"`
	SourceKind string           `json:"sourceKind,omitempty"`
	Binding    string           `json:"binding,omitempty"`
	Alias      string           `json:"alias,omitempty"`
	Resolution string           `json:"resolution"`
}

// FieldID is stable inside one derived query revision. It is deliberately not
// a CanonicalID because SQL-authored fields are derived, not canonical meaning.
type FieldID string

type FieldRef struct {
	ID                 FieldID `json:"id,omitempty"`
	OccurrenceID       string  `json:"occurrenceId,omitempty"`
	ScopeID            string  `json:"scopeId,omitempty"`
	Name               string  `json:"name"`
	Source             string  `json:"source,omitempty"`
	PhysicalSource     string  `json:"physicalSource,omitempty"`
	LogicalSourceField FieldID `json:"logicalSourceField,omitempty"`
	Resolution         string  `json:"resolution"`
	Correlated         bool    `json:"correlated,omitempty"`
	OuterScopeID       string  `json:"outerScopeId,omitempty"`
}

type OutputField struct {
	ID         FieldID   `json:"id"`
	ScopeID    string    `json:"scopeId"`
	Name       string    `json:"name,omitempty"`
	Expression string    `json:"expression,omitempty"`
	Inputs     []FieldID `json:"inputs,omitempty"`
}

type LineageEdge struct {
	From FieldID `json:"from"`
	To   FieldID `json:"to"`
}

// DerivedIR is a query semantic artifact. SQL remains the authoring source of
// truth; scopes, field identities and lineage are reproducible observations.
type DerivedIR struct {
	Query        core.CanonicalID `json:"query"`
	Datasource   core.LocalName   `json:"datasource"`
	Database     core.CanonicalID `json:"database"`
	Scopes       []QueryScope     `json:"scopes,omitempty"`
	Sources      []SourceRef      `json:"sources,omitempty"`
	Fields       []FieldRef       `json:"fields,omitempty"`
	OutputFields []OutputField    `json:"outputFields,omitempty"`
	Lineage      []LineageEdge    `json:"lineage,omitempty"`
	Coverage     AnalysisCoverage `json:"coverage"`
	Relational   RelationalIR     `json:"relational,omitempty"`
}

func DeriveFromAnalysis(q EmbeddedSQL, databaseID core.CanonicalID, result analyzer.Result, snapshot *physical.SchemaSnapshot) DerivedIR {
	ir := DerivedIR{Query: q.ID, Datasource: q.Datasource, Database: databaseID}
	if !result.OK() {
		ir.Coverage = AnalysisCoverage{Parse: CoverageNone, Resolution: CoverageNone, Lowering: CoverageNone}
		return ir
	}
	ir.Coverage.Parse = CoverageFull
	for _, s := range result.Scopes {
		ir.Scopes = append(ir.Scopes, QueryScope{ID: s.ID, ParentID: s.ParentID, Kind: s.Kind, Name: s.Name})
	}
	if len(ir.Scopes) == 0 {
		ir.Scopes = []QueryScope{{ID: "scope:root", Kind: "root"}}
	}

	resolved, total := 0, 0
	aliases := result.TableAliases

	// Prefer source occurrences from relational lowering because a set of table
	// names cannot preserve self-joins, CTE bindings or nested scopes.
	for _, op := range result.Operations {
		if op.Kind != string(RelSource) {
			continue
		}
		total++
		resolution := "logical"
		if op.SourceKind == "" || op.SourceKind == "table" {
			resolution = resolveTable(snapshot, databaseID, op.Source)
			if resolution == "schema-resolved" {
				resolved++
			}
		} else {
			// A CTE/subquery source is lexically resolved by the analyzer, not by
			// the physical schema snapshot.
			resolved++
		}
		ir.Sources = append(ir.Sources, SourceRef{
			ID: op.ID, ScopeID: op.ScopeID, Database: databaseID, Name: op.Source,
			SourceKind: defaultSourceKind(op.SourceKind), Binding: op.Binding,
			Alias: op.Alias, Resolution: resolution,
		})
	}
	if len(ir.Sources) == 0 {
		for i, name := range result.Tables {
			total++
			resolution := resolveTable(snapshot, databaseID, name)
			if resolution == "schema-resolved" {
				resolved++
			}
			ir.Sources = append(ir.Sources, SourceRef{ID: fmt.Sprintf("source:%d", i), ScopeID: "scope:root", Database: databaseID, Name: name, SourceKind: "table", Alias: firstAliasFor(aliases, name), Resolution: resolution})
		}
	}

	// Pre-assign output IDs so CTE/subquery references can resolve to an
	// upstream logical field before physical field occurrences are built.
	outputIDs := map[string]FieldID{}
	outputByScopeName := map[string]map[string]FieldID{}
	outputOrdinal := map[string]int{}
	for _, f := range result.OutputFields {
		ordinal := outputOrdinal[f.ScopeID]
		outputOrdinal[f.ScopeID] = ordinal + 1
		outID := FieldID(fmt.Sprintf("%s#output:%s:%d", q.ID, safeIDPart(f.ScopeID), ordinal))
		outputIDs[f.ID] = outID
		if f.Name != "" {
			if outputByScopeName[f.ScopeID] == nil {
				outputByScopeName[f.ScopeID] = map[string]FieldID{}
			}
			outputByScopeName[f.ScopeID][strings.ToLower(f.Name)] = outID
		}
	}

	occurrenceFields := map[string]FieldID{}
	if len(result.ColumnRefs) > 0 {
		for i, c := range result.ColumnRefs {
			total++
			physicalSource, resolution := resolveColumn(snapshot, databaseID, ir.Sources, aliases, c.Qualifier, c.Name)
			logicalSource := FieldID("")
			if resolution != "schema-resolved" {
				logicalSource = resolveLogicalField(ir.Sources, c.ScopeID, c.Qualifier, c.Name, outputByScopeName)
				if logicalSource != "" {
					resolution = "logical-resolved"
				}
			}
			if resolution == "schema-resolved" || resolution == "logical-resolved" {
				resolved++
			}
			fid := FieldID(fmt.Sprintf("%s#fieldref:%d", q.ID, i))
			occurrenceFields[c.ID] = fid
			ir.Fields = append(ir.Fields, FieldRef{ID: fid, OccurrenceID: c.ID, ScopeID: c.ScopeID, Name: c.Name, Source: c.Qualifier, PhysicalSource: physicalSource, LogicalSourceField: logicalSource, Resolution: resolution, Correlated: c.Correlated, OuterScopeID: c.OuterScopeID})
			if logicalSource != "" {
				ir.Lineage = append(ir.Lineage, LineageEdge{From: logicalSource, To: fid})
			}
		}
	} else {
		for i, name := range result.Columns {
			total++
			source := ""
			col := name
			if dot := strings.LastIndex(name, "."); dot >= 0 {
				source, col = name[:dot], name[dot+1:]
			}
			physicalSource, resolution := resolveColumn(snapshot, databaseID, ir.Sources, aliases, source, col)
			if resolution == "schema-resolved" {
				resolved++
			}
			ir.Fields = append(ir.Fields, FieldRef{ID: FieldID(fmt.Sprintf("%s#fieldref:%d", q.ID, i)), ScopeID: "scope:root", Name: col, Source: source, PhysicalSource: physicalSource, Resolution: resolution})
		}
	}

	for _, f := range result.OutputFields {
		outID := outputIDs[f.ID]
		var inputs []FieldID
		for _, input := range f.Inputs {
			if id, ok := occurrenceFields[input]; ok {
				inputs = append(inputs, id)
				ir.Lineage = append(ir.Lineage, LineageEdge{From: id, To: outID})
			}
		}
		ir.OutputFields = append(ir.OutputFields, OutputField{ID: outID, ScopeID: f.ScopeID, Name: f.Name, Expression: f.Expression, Inputs: inputs})
	}

	for _, op := range result.Operations {
		n := RelNode{ID: op.ID, Kind: RelNodeKind(op.Kind), ScopeID: op.ScopeID, Inputs: append([]string(nil), op.Inputs...), Source: op.Source, SourceKind: defaultSourceKind(op.SourceKind), Binding: op.Binding, Alias: op.Alias, JoinKind: op.JoinKind, Expressions: append([]string(nil), op.Expressions...)}
		if n.Kind == RelSource {
			if n.SourceKind == "table" {
				n.Resolution = resolveTable(snapshot, databaseID, op.Source)
			} else {
				n.Resolution = "logical"
			}
		}
		ir.Relational.Nodes = append(ir.Relational.Nodes, n)
		// The root is the last operation in the root scope. Nested CTE/subquery
		// operations do not displace the top-level query root.
		if op.ScopeID == "" || op.ScopeID == "scope:root" {
			ir.Relational.Root = n.ID
		}
	}
	if ir.Relational.Root == "" && len(ir.Relational.Nodes) > 0 {
		ir.Relational.Root = ir.Relational.Nodes[len(ir.Relational.Nodes)-1].ID
	}
	switch result.Lowering {
	case "full":
		ir.Coverage.Lowering = CoverageFull
	case "partial":
		ir.Coverage.Lowering = CoveragePartial
	default:
		ir.Coverage.Lowering = CoverageNone
	}
	switch {
	case total == 0:
		ir.Coverage.Resolution = CoverageNone
	case resolved == total:
		ir.Coverage.Resolution = CoverageFull
	case resolved > 0:
		ir.Coverage.Resolution = CoveragePartial
	default:
		ir.Coverage.Resolution = CoverageNone
	}
	return ir
}

func defaultSourceKind(kind string) string {
	if strings.TrimSpace(kind) == "" {
		return "table"
	}
	return kind
}

func safeIDPart(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return "root"
	}
	v = strings.ReplaceAll(v, ":", "_")
	v = strings.ReplaceAll(v, ".", "_")
	return v
}

func resolveTable(snapshot *physical.SchemaSnapshot, databaseID core.CanonicalID, name string) string {
	if snapshot != nil && snapshot.Database == databaseID {
		if _, ok := snapshot.Table(name); ok {
			return "schema-resolved"
		}
	}
	return "name-only"
}

func firstAliasFor(aliases map[string]string, target string) string {
	var keys []string
	for alias, v := range aliases {
		if strings.EqualFold(v, target) && !strings.EqualFold(alias, target) {
			keys = append(keys, alias)
		}
	}
	sort.Strings(keys)
	if len(keys) == 0 {
		return ""
	}
	return keys[0]
}

func resolveLogicalField(sources []SourceRef, scopeID, qualifier, column string, outputs map[string]map[string]FieldID) FieldID {
	var matches []FieldID
	for _, src := range sources {
		if src.ScopeID != scopeID || (src.SourceKind != "cte" && src.SourceKind != "subquery") || src.Binding == "" {
			continue
		}
		if qualifier != "" && !strings.EqualFold(src.Alias, qualifier) && !strings.EqualFold(src.Name, qualifier) {
			continue
		}
		if id := outputs[src.Binding][strings.ToLower(column)]; id != "" {
			matches = append(matches, id)
		}
	}
	if len(matches) == 1 {
		return matches[0]
	}
	return ""
}

func resolveColumn(snapshot *physical.SchemaSnapshot, databaseID core.CanonicalID, sources []SourceRef, aliases map[string]string, source, column string) (string, string) {
	if snapshot == nil || snapshot.Database != databaseID {
		return "", "name-only"
	}
	if source != "" {
		physicalSource := source
		if target, ok := aliases[source]; ok {
			physicalSource = target
		}
		// A qualifier can be an alias in the specific source occurrence even if
		// it was not present in the legacy flat alias map.
		for _, src := range sources {
			if strings.EqualFold(src.Alias, source) && src.SourceKind == "table" {
				physicalSource = src.Name
				break
			}
		}
		if table, ok := snapshot.Table(physicalSource); ok {
			if _, ok := table.Column(column); ok {
				return table.QualifiedName(), "schema-resolved"
			}
		}
		return physicalSource, "name-only"
	}

	var matches []string
	for _, src := range sources {
		if src.SourceKind != "table" {
			continue
		}
		table, ok := snapshot.Table(src.Name)
		if !ok {
			continue
		}
		if _, ok := table.Column(column); ok {
			matches = append(matches, table.QualifiedName())
		}
	}
	matches = uniqueStrings(matches)
	if len(matches) == 1 {
		return matches[0], "schema-resolved"
	}
	return "", "name-only"
}

func uniqueStrings(in []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(in))
	for _, v := range in {
		k := strings.ToLower(v)
		if !seen[k] {
			seen[k] = true
			out = append(out, v)
		}
	}
	return out
}
