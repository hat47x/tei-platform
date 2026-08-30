package index

import (
	"sort"

	"tei/internal/core"
	"tei/internal/database/physical"
	"tei/internal/database/plan"
	"tei/internal/ir"
	"tei/internal/query"
	"tei/internal/query/analyzer"
)

type ObservedPhysicalDependency struct {
	Query       core.CanonicalID `json:"query"`
	Application core.CanonicalID `json:"application"`
	Datasource  core.LocalName   `json:"datasource"`
	Database    core.CanonicalID `json:"database"`
	Kind        string           `json:"kind"`
	Name        string           `json:"name"`
	Alias       string           `json:"alias,omitempty"`
	Resolution  string           `json:"resolution"`
}

// BuildWithQueryAnalyses overlays non-canonical observations onto the normal
// derived index. It is the compatibility wrapper for callers without a
// Physical Schema Snapshot.
func BuildWithQueryAnalyses(w ir.World, analyses []analyzer.Result) Derived {
	return BuildWithObservations(w, analyses, nil)
}

// BuildWithObservations combines SQL analysis with optional observed physical
// schema metadata. Neither analysis nor snapshots mutate ir.World.
func BuildWithObservations(w ir.World, analyses []analyzer.Result, snapshots []physical.SchemaSnapshot) Derived {
	return BuildWithAllObservations(w, analyses, snapshots, nil)
}

// BuildWithAllObservations overlays query analyses, physical schema snapshots
// and execution plans without mutating canonical ir.World.
func BuildWithAllObservations(w ir.World, analyses []analyzer.Result, snapshots []physical.SchemaSnapshot, plans []plan.IR) Derived {
	d := Build(w)
	queries := map[core.CanonicalID]query.EmbeddedSQL{}
	queryDB := map[core.CanonicalID]core.CanonicalID{}
	apps := map[core.CanonicalID]map[core.LocalName]core.CanonicalID{}
	for _, app := range w.Applications {
		m := map[core.LocalName]core.CanonicalID{}
		for _, req := range app.Databases {
			m[req.Alias] = req.TargetDatabase
		}
		apps[app.ID] = m
	}
	for _, q := range w.Queries {
		queries[q.ID] = q
		queryDB[q.ID] = apps[q.Application][q.Datasource]
	}
	snapshotByDB := map[core.CanonicalID]physical.SchemaSnapshot{}
	for _, s := range snapshots {
		n := s.Normalize()
		snapshotByDB[n.Database] = n
		d.PhysicalSchemas = append(d.PhysicalSchemas, n)
	}
	sort.Slice(d.PhysicalSchemas, func(i, j int) bool { return d.PhysicalSchemas[i].Database < d.PhysicalSchemas[j].Database })

	for _, a := range analyses {
		q, ok := queries[a.Query]
		if !ok {
			continue
		}
		db := queryDB[a.Query]
		var snap *physical.SchemaSnapshot
		if s, ok := snapshotByDB[db]; ok {
			s := s
			snap = &s
		}
		qir := query.DeriveFromAnalysis(q, db, a, snap)
		d.AnalyzedQueries = append(d.AnalyzedQueries, AnalyzedQueryEntry{Application: q.Application, IR: qir})
		if !a.OK() {
			continue
		}
		for _, src := range qir.Sources {
			if src.SourceKind != "table" {
				continue
			}
			d.ObservedPhysicalDependencies = append(d.ObservedPhysicalDependencies, ObservedPhysicalDependency{
				Query: a.Query, Application: q.Application, Datasource: q.Datasource, Database: db,
				Kind: "table", Name: src.Name, Alias: src.Alias, Resolution: src.Resolution,
			})
		}
		for _, field := range qir.Fields {
			if field.Resolution == "logical-resolved" {
				continue
			}
			name := field.Name
			if field.PhysicalSource != "" {
				name = field.PhysicalSource + "." + field.Name
			} else if field.Source != "" {
				name = field.Source + "." + field.Name
			}
			d.ObservedPhysicalDependencies = append(d.ObservedPhysicalDependencies, ObservedPhysicalDependency{
				Query: a.Query, Application: q.Application, Datasource: q.Datasource, Database: db,
				Kind: "column", Name: name, Alias: field.Source, Resolution: field.Resolution,
			})
		}
	}
	d.ObservedPhysicalDependencies = dedupeObservedPhysicalDependencies(d.ObservedPhysicalDependencies)
	sort.Slice(d.ObservedPhysicalDependencies, func(i, j int) bool {
		a, b := d.ObservedPhysicalDependencies[i], d.ObservedPhysicalDependencies[j]
		if a.Query != b.Query {
			return a.Query < b.Query
		}
		if a.Kind != b.Kind {
			return a.Kind < b.Kind
		}
		return a.Name < b.Name
	})
	sort.Slice(d.AnalyzedQueries, func(i, j int) bool { return d.AnalyzedQueries[i].IR.Query < d.AnalyzedQueries[j].IR.Query })

	analyzedByQuery := map[core.CanonicalID]AnalyzedQueryEntry{}
	for _, a := range d.AnalyzedQueries {
		analyzedByQuery[a.IR.Query] = a
	}
	for _, p := range plans {
		q, ok := queries[p.Query]
		if !ok || p.Database != queryDB[p.Query] {
			continue
		}
		entry := ExecutionPlanEntry{Application: q.Application, Plan: p.Normalize()}
		if analyzed, ok := analyzedByQuery[p.Query]; ok {
			entry.Mappings = plan.MapLogical(analyzed.IR, entry.Plan)
		}
		d.ExecutionPlans = append(d.ExecutionPlans, entry)
	}
	sort.Slice(d.ExecutionPlans, func(i, j int) bool {
		if d.ExecutionPlans[i].Plan.Query != d.ExecutionPlans[j].Plan.Query {
			return d.ExecutionPlans[i].Plan.Query < d.ExecutionPlans[j].Plan.Query
		}
		return d.ExecutionPlans[i].Plan.Mode < d.ExecutionPlans[j].Plan.Mode
	})
	return d
}

func dedupeObservedPhysicalDependencies(in []ObservedPhysicalDependency) []ObservedPhysicalDependency {
	seen := map[string]bool{}
	out := make([]ObservedPhysicalDependency, 0, len(in))
	for _, dep := range in {
		// The observed dependency index describes coupling to physical objects,
		// not every syntactic occurrence. Detailed occurrences remain in Query IR.
		key := string(dep.Query) + "\x00" + dep.Kind + "\x00" + string(dep.Database) + "\x00" + dep.Name
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, dep)
	}
	return out
}
