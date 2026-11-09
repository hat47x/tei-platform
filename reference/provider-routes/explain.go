// Package providerroutes provides a small, synthetic read-only conformance
// reference for the TEI multiple-Data-Provider discussion. It is not a
// production resolver, does not grant authority, and never invokes Providers.
package providerroutes

import (
	"sort"
)

// Route is a manually specified fixture fact. Physical endpoints, scopes and
// mapping revisions are intentionally distinct from the semantic data kind.
// The fields do not represent a proposed TEI Core ABI.
type Route struct {
	Alias           string
	SemanticData    string
	Scope           string
	Provider        string
	Endpoint        string
	Mapping         string
	MappingRevision string
	Purpose         string // current, archive, replica, projection, ... (fixture labels only)
	CanRead         bool
	CanWrite        bool
}

// Path groups aliases pointing at one explicitly identical realization.
// It is not a statement that two physical realizations share the same data.
type Path struct {
	SemanticData    string
	Scope           string
	Provider        string
	Endpoint        string
	Mapping         string
	MappingRevision string
	Purpose         string
	Aliases         []string
}

// Report is an explanation of declarative candidates, NOT an execution plan.
// Absent is a missing qualified declaration, not evidence of nonexistent data.
type Report struct {
	Operation      string
	Subject        string
	Scope          string
	Paths          []Path
	ExcludedAlias  []string
	InvalidAliases []string
	CandidateCount int
}

func equalPath(a, b Route) bool {
	return a.SemanticData == b.SemanticData && a.Scope == b.Scope &&
		a.Provider == b.Provider && a.Endpoint == b.Endpoint &&
		a.Mapping == b.Mapping && a.MappingRevision == b.MappingRevision &&
		a.Purpose == b.Purpose
}

// Explain enumerates all declared candidates for exactly one Subject+Scope,
// without choosing one. Every exclusion is local to a route; having a readable
// route does not erase another route's declared read restriction.
func Explain(routes []Route, subject, scope, operation string) Report {
	out := Report{Operation: operation, Subject: subject, Scope: scope, Paths: []Path{}, ExcludedAlias: []string{}, InvalidAliases: []string{}}
	unique := make([]Route, 0)
	for _, r := range routes {
		if r.SemanticData != subject || r.Scope != scope {
			continue
		}
		if r.Alias == "" || r.Provider == "" || r.Endpoint == "" || r.Mapping == "" || r.MappingRevision == "" || r.Purpose == "" {
			out.InvalidAliases = append(out.InvalidAliases, r.Alias)
			continue
		}
		eligible := (operation == "read" && r.CanRead) || (operation == "write" && r.CanWrite)
		if !eligible {
			out.ExcludedAlias = append(out.ExcludedAlias, r.Alias)
			continue
		}
		idx := -1
		for i, candidate := range unique {
			if equalPath(candidate, r) {
				idx = i
				break
			}
		}
		if idx < 0 {
			unique = append(unique, r)
			out.Paths = append(out.Paths, Path{SemanticData: r.SemanticData, Scope: r.Scope, Provider: r.Provider, Endpoint: r.Endpoint,
				Mapping: r.Mapping, MappingRevision: r.MappingRevision, Purpose: r.Purpose, Aliases: []string{r.Alias}})
		} else {
			out.Paths[idx].Aliases = append(out.Paths[idx].Aliases, r.Alias)
		}
	}
	for i := range out.Paths {
		sort.Strings(out.Paths[i].Aliases)
	}
	sort.Slice(out.Paths, func(i, j int) bool {
		a, b := out.Paths[i], out.Paths[j]
		if a.Provider != b.Provider {
			return a.Provider < b.Provider
		}
		if a.Endpoint != b.Endpoint {
			return a.Endpoint < b.Endpoint
		}
		if a.Mapping != b.Mapping {
			return a.Mapping < b.Mapping
		}
		if a.MappingRevision != b.MappingRevision {
			return a.MappingRevision < b.MappingRevision
		}
		if a.Purpose != b.Purpose {
			return a.Purpose < b.Purpose
		}
		return a.Scope < b.Scope
	})
	sort.Strings(out.ExcludedAlias)
	sort.Strings(out.InvalidAliases)
	out.CandidateCount = len(out.Paths)
	return out
}
