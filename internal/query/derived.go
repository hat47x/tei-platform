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
	scopeKinds := make(map[string]string, len(ir.Scopes))
	scopeParents := make(map[string]string, len(ir.Scopes))
	duplicateScopes := make(map[string]bool)
	for _, s := range ir.Scopes {
		if _, exists := scopeParents[s.ID]; exists {
			duplicateScopes[s.ID] = true
		}
		scopeKinds[s.ID] = s.Kind
		scopeParents[s.ID] = s.ParentID
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
		opScopeID := op.ScopeID
		if opScopeID == "" {
			opScopeID = "scope:root"
		}

		resolution := "name-only"
		if op.SourceKind == "" || op.SourceKind == "table" {
			resolution = resolveTable(snapshot, databaseID, op.Source)
			if resolution == "schema-resolved" {
				resolved++
			}
		} else if op.Binding != "" && !duplicateScopes[op.Binding] && scopeKinds[op.Binding] == op.SourceKind &&
			bindingScopeCanBeVisible(scopeParents, scopeKinds, duplicateScopes, opScopeID, op.Binding, op.SourceKind) {
			// Exclude targets outside the lexical hierarchy. Declaration order,
			// recursion and dialect-specific visibility are not proven here.
			resolution = "logical"
			resolved++
		}
		ir.Sources = append(ir.Sources, SourceRef{
			ID: op.ID, ScopeID: opScopeID, Database: databaseID, Name: op.Source,
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
	outputByScopeName := map[string]map[string][]FieldID{}
	outputOrdinal := map[string]int{}
	for _, f := range result.OutputFields {
		ordinal := outputOrdinal[f.ScopeID]
		outputOrdinal[f.ScopeID] = ordinal + 1
		outID := FieldID(fmt.Sprintf("%s#output:%s:%d", q.ID, safeIDPart(f.ScopeID), ordinal))
		outputIDs[f.ID] = outID
		if f.Name != "" {
			if outputByScopeName[f.ScopeID] == nil {
				outputByScopeName[f.ScopeID] = map[string][]FieldID{}
			}
			outputByScopeName[f.ScopeID][strings.ToLower(f.Name)] = append(outputByScopeName[f.ScopeID][strings.ToLower(f.Name)], outID)
		}
	}

	occurrenceFields := map[string]FieldID{}
	if len(result.ColumnRefs) > 0 {
		for i, c := range result.ColumnRefs {
			total++
			physicalSource, resolution := resolveColumn(snapshot, databaseID, ir.Sources, aliases, c.ScopeID, c.Correlated, c.OuterScopeID, c.Qualifier, c.Name)
			logicalScopeID := c.ScopeID
			if c.Correlated {
				logicalScopeID = c.OuterScopeID
			}
			logicalSource, logicalAmbiguous := resolveLogicalField(ir.Sources, logicalScopeID, c.Qualifier, c.Name, outputByScopeName)
			if (c.Correlated && !isStrictScopeAncestor(scopeParents, scopeKinds, duplicateScopes, c.ScopeID, c.OuterScopeID)) || hasUnknownEligibleSource(ir.Sources, logicalScopeID, c.Qualifier) || logicalAmbiguous || (logicalSource != "" && resolution == "schema-resolved") {
				// A physical table and a CTE (or two CTE outputs) can both
				// declare the same column. Never infer a preferred source.
				physicalSource, logicalSource, resolution = "", "", "name-only"
			} else if logicalSource != "" {
				resolution = "logical-resolved"
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
			physicalSource, resolution := resolveColumn(snapshot, databaseID, ir.Sources, aliases, "scope:root", false, "", source, col)
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
			for _, src := range ir.Sources {
				if src.ID == op.ID {
					n.Resolution = src.Resolution
					break
				}
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

func resolveLogicalField(sources []SourceRef, scopeID, qualifier, column string, outputs map[string]map[string][]FieldID) (FieldID, bool) {
	if scopeID == "" {
		scopeID = "scope:root"
	}
	var matches []FieldID
	ambiguous := false
	for _, src := range sources {
		if src.ScopeID != scopeID || (src.SourceKind != "cte" && src.SourceKind != "subquery") ||
			src.Resolution != "logical" || src.Binding == "" {
			continue
		}
		if qualifier != "" && !strings.EqualFold(src.Alias, qualifier) && !strings.EqualFold(src.Name, qualifier) {
			continue
		}
		ids := outputs[src.Binding][strings.ToLower(column)]
		if len(ids) > 1 {
			ambiguous = true
		}
		if len(ids) == 1 {
			matches = append(matches, ids[0])
		}
	}
	if ambiguous || len(matches) > 1 {
		return "", true
	}
	if len(matches) == 1 {
		return matches[0], false
	}
	return "", false
}

// An unresolved source may expose the requested column. It cannot be ignored
// when proving uniqueness, but an unrelated qualified source need not block a
// known explicit alias.
func bindingScopeCanBeVisible(parents, kinds map[string]string, duplicate map[string]bool, source, target, kind string) bool {
	parent, ok := parents[target]
	if !ok || parent == "" || duplicate[target] || target == source {
		return false
	}
	if kind == "subquery" {
		return parent == source && validScopeChain(parents, kinds, duplicate, source)
	}
	if kind == "cte" {
		return (parent == source && validScopeChain(parents, kinds, duplicate, source)) || isStrictScopeAncestor(parents, kinds, duplicate, source, parent)
	}
	return false
}

func validScopeChain(parents, kinds map[string]string, duplicate map[string]bool, scope string) bool {
	seen := make(map[string]bool)
	for scope != "" && !seen[scope] && !duplicate[scope] {
		seen[scope] = true
		parent, ok := parents[scope]
		if !ok {
			return false
		}
		if parent == "" {
			return kinds[scope] == "root"
		}
		scope = parent
	}
	return false
}

func isStrictScopeAncestor(parents, kinds map[string]string, duplicate map[string]bool, child, ancestor string) bool {
	if child == "" || ancestor == "" || child == ancestor || duplicate[ancestor] {
		return false
	}
	seen := make(map[string]bool)
	found := false
	for child != "" && !seen[child] && !duplicate[child] {
		seen[child] = true
		parent, ok := parents[child]
		if !ok || duplicate[parent] {
			return false
		}
		if parent == "" {
			return found && kinds[child] == "root"
		}
		if parent == ancestor {
			found = true
		}
		child = parent
	}
	return false
}

func hasUnknownEligibleSource(sources []SourceRef, scopeID, qualifier string) bool {
	if scopeID == "" {
		scopeID = "scope:root"
	}
	for _, src := range sources {
		if src.ScopeID != scopeID || src.Resolution == "schema-resolved" || src.Resolution == "logical" {
			continue
		}
		if qualifier == "" || strings.EqualFold(src.Alias, qualifier) || strings.EqualFold(src.Name, qualifier) {
			return true
		}
		if src.Alias == "" {
			if dot := strings.LastIndex(src.Name, "."); dot >= 0 && strings.EqualFold(src.Name[dot+1:], qualifier) {
				return true
			}
		}
	}
	return false
}

// resolveColumn uses the source occurrence in the column's lexical scope.
// Physical table identity alone is insufficient: a self-join has two
// occurrences even if both point to the same table.
func resolveColumn(
	snapshot *physical.SchemaSnapshot,
	databaseID core.CanonicalID,
	sources []SourceRef,
	aliases map[string]string,
	scopeID string,
	correlated bool,
	outerScopeID, source, column string,
) (string, string) {
	if snapshot == nil || snapshot.Database != databaseID {
		return "", "name-only"
	}
	if scopeID == "" {
		scopeID = "scope:root"
	}
	if correlated {
		if outerScopeID == "" {
			return "", "name-only"
		}
		scopeID = outerScopeID
	}
	if hasUnknownEligibleSource(sources, scopeID, source) {
		return "", "name-only"
	}

	var matches []string
	for _, src := range sources {
		if src.ScopeID != scopeID || src.SourceKind != "table" {
			continue
		}
		if source != "" {
			matched := strings.EqualFold(src.Alias, source)
			if src.Alias == "" {
				matched = matched || strings.EqualFold(src.Name, source)
				if dot := strings.LastIndex(src.Name, "."); dot >= 0 {
					matched = matched || strings.EqualFold(src.Name[dot+1:], source)
				}
				// The legacy flat alias map is only a fallback tied to an
				// occurrence in this lexical scope, never a global resolver.
				if target, ok := aliases[source]; ok && strings.EqualFold(target, src.Name) {
					matched = true
				}
			}
			if !matched {
				continue
			}
		}
		table, ok := snapshot.Table(src.Name)
		if !ok {
			continue
		}
		if _, ok := table.Column(column); ok {
			// Do not deduplicate by physical table: each occurrence counts.
			matches = append(matches, table.QualifiedName())
		}
	}
	if len(matches) == 1 {
		return matches[0], "schema-resolved"
	}
	// Unknown and ambiguous bindings are never represented as known sources.
	return "", "name-only"
}
