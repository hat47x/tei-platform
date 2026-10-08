package knowledgecontext

import (
	"sort"
	"strings"

	"tei/internal/core"
	"tei/internal/index"
	"tei/internal/ir"
	"tei/internal/sourceindex"
	"tei/internal/subject"
)

type Edge struct {
	From     subject.Subject `json:"from"`
	Relation string          `json:"relation"`
	To       subject.Subject `json:"to"`
	Alias    core.LocalName  `json:"alias,omitempty"`
}

type Context struct {
	Subject            subject.Subject                     `json:"subject"`
	Artifacts          []sourceindex.Artifact              `json:"artifacts,omitempty"`
	References         []sourceindex.Reference             `json:"references,omitempty"`
	SemanticElement    *index.SemanticElementEntry         `json:"semanticElement,omitempty"`
	ApplicationPattern *index.ApplicationPatternIndexEntry `json:"applicationPattern,omitempty"`
	DataDesign         *DataDesignContext                  `json:"dataDesign,omitempty"`
	Outgoing           []Edge                              `json:"outgoing,omitempty"`
	Incoming           []Edge                              `json:"incoming,omitempty"`
	Unresolved         []string                            `json:"unresolved,omitempty"`
	AuthorityLimits    []string                            `json:"authorityLimits"`
}

func Build(w ir.World, sources sourceindex.Index, target subject.Subject) Context {
	derived := index.Build(w)
	ctx := Context{
		Subject: target,
		AuthorityLimits: []string{
			"derived-context-is-not-canonical-authority",
			"dependency-edge-is-not-semantic-applicability",
			"dependency-edge-is-not-write-or-runtime-authority",
			"projection-or-runtime-identity-is-not-business-data-identity",
		},
	}

	ctx.References = append(ctx.References, sources.ReferencesFor(target)...)
	artifactPaths := make(map[string]struct{}, len(ctx.References))
	for _, ref := range ctx.References {
		artifactPaths[ref.ArtifactPath] = struct{}{}
	}
	for _, artifact := range sources.Artifacts {
		if _, ok := artifactPaths[artifact.Path]; ok {
			ctx.Artifacts = append(ctx.Artifacts, artifact)
		}
	}
	if len(ctx.References) == 0 {
		ctx.Unresolved = append(ctx.Unresolved, "source-reference-unavailable")
	}

	if kind, ok := semanticElementKind(target.Kind); ok {
		for _, entry := range derived.SemanticElements {
			if entry.Kind == kind && string(entry.ID) == target.ID {
				copy := entry
				ctx.SemanticElement = &copy
				break
			}
		}
		if ctx.SemanticElement == nil {
			ctx.Unresolved = append(ctx.Unresolved, "semantic-element-not-found-in-derived-index")
		}
	}

	if target.Kind == "application-pattern" {
		for _, entry := range derived.ApplicationPatterns {
			if string(entry.Pattern) == target.ID {
				copy := entry
				ctx.ApplicationPattern = &copy
				break
			}
		}
		if ctx.ApplicationPattern == nil {
			ctx.Unresolved = append(ctx.Unresolved, "application-pattern-not-found-in-derived-index")
		}
	}

	ctx.DataDesign = buildDataDesignContext(w, sources, target, ctx.SemanticElement, ctx.ApplicationPattern)
	if ctx.DataDesign == nil {
		switch target.Kind {
		case "semantic-validation":
			if ctx.SemanticElement != nil {
				ctx.Unresolved = append(ctx.Unresolved, "data-design-validation-scope-not-uniquely-resolvable")
			}
		case "persistence":
			if persistenceSubjectCount(w, core.CanonicalID(target.ID)) != 1 {
				ctx.Unresolved = append(ctx.Unresolved, "data-design-persistence-subject-not-uniquely-resolvable")
			}
		}
	}

	for _, dep := range derived.SemanticDependencies {
		edge := Edge{
			From:     subject.Subject{Kind: dep.FromKind, ID: string(dep.From)},
			Relation: dep.Relation,
			To:       subject.Subject{Kind: dep.ToKind, ID: string(dep.To)},
		}
		appendMatchingEdge(&ctx, target, edge)
	}
	for _, dep := range derived.DeclaredDependencies {
		edge := Edge{
			From:     subject.Subject{Kind: dep.FromKind, ID: string(dep.From)},
			Relation: dep.Relation,
			To:       subject.Subject{Kind: dep.ToKind, ID: string(dep.To)},
			Alias:    dep.Alias,
		}
		appendMatchingEdge(&ctx, target, edge)
	}

	sortArtifacts(ctx.Artifacts)
	sortReferences(ctx.References)
	sortEdges(ctx.Outgoing)
	sortEdges(ctx.Incoming)
	sort.Strings(ctx.Unresolved)
	return ctx
}

func appendMatchingEdge(ctx *Context, target subject.Subject, edge Edge) {
	if subjectMatches(edge.From, target) {
		ctx.Outgoing = append(ctx.Outgoing, edge)
	}
	if subjectMatches(edge.To, target) {
		ctx.Incoming = append(ctx.Incoming, edge)
	}
}

func subjectMatches(candidate, target subject.Subject) bool {
	if candidate.ID != target.ID {
		return false
	}
	if candidate.Kind == target.Kind {
		return true
	}
	if candidate.Kind == "semantic-element" && strings.HasPrefix(target.Kind, "semantic-") {
		return true
	}
	if candidate.Kind == "semantic" && target.Kind == "semantic-namespace" {
		return true
	}
	return false
}

func semanticElementKind(subjectKind string) (string, bool) {
	switch subjectKind {
	case "semantic-namespace":
		return "namespace", true
	case "semantic-type":
		return "type", true
	case "semantic-data":
		return "data", true
	case "semantic-attribute":
		return "attribute", true
	case "semantic-relation":
		return "relation", true
	case "semantic-validation":
		return "validation", true
	case "semantic-capability":
		return "capability", true
	default:
		return "", false
	}
}

func sortArtifacts(values []sourceindex.Artifact) {
	sort.Slice(values, func(i, j int) bool {
		return values[i].Path < values[j].Path
	})
}

func sortReferences(values []sourceindex.Reference) {
	sort.Slice(values, func(i, j int) bool {
		if values[i].Role != values[j].Role {
			return values[i].Role < values[j].Role
		}
		if values[i].ArtifactPath != values[j].ArtifactPath {
			return values[i].ArtifactPath < values[j].ArtifactPath
		}
		return values[i].Range.Start.Offset < values[j].Range.Start.Offset
	})
}

func sortEdges(values []Edge) {
	sort.Slice(values, func(i, j int) bool {
		a, b := values[i], values[j]
		if a.Relation != b.Relation {
			return a.Relation < b.Relation
		}
		if a.From.Kind != b.From.Kind {
			return a.From.Kind < b.From.Kind
		}
		if a.From.ID != b.From.ID {
			return a.From.ID < b.From.ID
		}
		if a.To.Kind != b.To.Kind {
			return a.To.Kind < b.To.Kind
		}
		if a.To.ID != b.To.ID {
			return a.To.ID < b.To.ID
		}
		return a.Alias < b.Alias
	})
}
