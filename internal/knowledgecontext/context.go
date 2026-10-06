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

type DataDesignKnowledgeState struct {
	Nature string `json:"nature"`
	Reason string `json:"reason"`
}

type DataDesignAttribute struct {
	ID   core.CanonicalID `json:"id"`
	Type core.CanonicalID `json:"type"`
}

type DataDesignRelationKey struct {
	SourceAttribute core.CanonicalID `json:"sourceAttribute"`
	TargetAttribute core.CanonicalID `json:"targetAttribute"`
}

type DataDesignRelation struct {
	ID          core.CanonicalID        `json:"id"`
	Source      core.CanonicalID        `json:"source"`
	Target      core.CanonicalID        `json:"target"`
	Cardinality string                  `json:"cardinality"`
	Keys        []DataDesignRelationKey `json:"keys,omitempty"`
}

type DataDesignConstraint struct {
	ID        core.CanonicalID `json:"id"`
	Kind      string           `json:"kind"`
	Attribute core.CanonicalID `json:"attribute,omitempty"`
	Relation  core.CanonicalID `json:"relation,omitempty"`
}

type DataDesignContext struct {
	Data                                  core.CanonicalID         `json:"data"`
	Focus                                 subject.Subject          `json:"focus"`
	Attributes                            []DataDesignAttribute    `json:"attributes,omitempty"`
	Relations                             []DataDesignRelation     `json:"relations,omitempty"`
	Constraints                           []DataDesignConstraint   `json:"constraints,omitempty"`
	BusinessOccurrenceIdentifier          DataDesignKnowledgeState `json:"businessOccurrenceIdentifier"`
	CandidateKeys                         DataDesignKnowledgeState `json:"candidateKeys"`
	FunctionalDependencies                DataDesignKnowledgeState `json:"functionalDependencies"`
	BusinessIdentityEvidenceQuestions     []string                 `json:"businessIdentityEvidenceQuestions"`
	FunctionalDependencyEvidenceQuestions []string                 `json:"functionalDependencyEvidenceQuestions"`
	AuthorityLimits                       []string                 `json:"authorityLimits"`
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

	ctx.DataDesign = buildDataDesignContext(w, target, ctx.SemanticElement)

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

func buildDataDesignContext(w ir.World, target subject.Subject, element *index.SemanticElementEntry) *DataDesignContext {
	var dataID core.CanonicalID
	switch target.Kind {
	case "semantic-data":
		if element == nil {
			return nil
		}
		dataID = element.ID
	case "semantic-attribute", "semantic-relation":
		if element == nil || element.Parent == "" {
			return nil
		}
		dataID = element.Parent
	default:
		return nil
	}

	ctx := &DataDesignContext{
		Data:  dataID,
		Focus: target,
		BusinessOccurrenceIdentifier: DataDesignKnowledgeState{
			Nature: "unresolved",
			Reason: "business-occurrence-identifier-not-first-class-in-current-canonical-model",
		},
		CandidateKeys: DataDesignKnowledgeState{
			Nature: "unresolved",
			Reason: "candidate-key-not-first-class-in-current-canonical-model",
		},
		FunctionalDependencies: DataDesignKnowledgeState{
			Nature: "unresolved",
			Reason: "functional-dependency-not-first-class-in-current-canonical-model",
		},
		BusinessIdentityEvidenceQuestions: []string{
			"which-fields-identify-the-business-occurrence",
			"what-is-the-uniqueness-scope",
			"when-is-the-identifier-valid",
			"which-relations-or-constraints-depend-on-the-identity",
			"which-source-or-contract-owns-the-uniqueness-claim",
			"which-normalization-or-equivalence-policy-applies",
		},
		FunctionalDependencyEvidenceQuestions: []string{
			"which-determinants-determine-which-dependents",
			"under-what-scope-and-time-does-the-dependency-hold",
			"which-source-or-evidence-establishes-the-dependency",
			"is-this-a-functional-dependency-or-a-domain-constraint",
		},
		AuthorityLimits: []string{
			"data-design-context-is-derived-not-authority",
			"absence-of-authored-business-identifier-is-not-no-identifier",
			"absence-of-authored-functional-dependency-is-not-no-dependency",
			"relation-key-is-not-business-identifier",
			"attribute-name-or-id-shape-is-not-key-evidence",
			"identifier-like-attribute-is-not-business-identity",
			"identifier-value-presence-is-not-uniqueness-evidence",
			"normalized-string-equality-is-not-business-identity-evidence",
			"application-pattern-ui-or-runtime-identity-is-not-data-identity",
		},
	}

	attributeOwner := map[core.CanonicalID]core.CanonicalID{}
	relations := map[core.CanonicalID]DataDesignRelation{}
	for _, ns := range w.Semantics {
		for _, data := range ns.Data {
			for _, attribute := range data.Attributes {
				attributeOwner[attribute.ID] = data.ID
				if data.ID == dataID {
					ctx.Attributes = append(ctx.Attributes, DataDesignAttribute{ID: attribute.ID, Type: attribute.TypeRef})
				}
			}
			for _, relation := range data.Relations {
				item := DataDesignRelation{
					ID:          relation.ID,
					Source:      data.ID,
					Target:      relation.Target,
					Cardinality: string(relation.Cardinality),
				}
				for _, key := range relation.Keys {
					item.Keys = append(item.Keys, DataDesignRelationKey{
						SourceAttribute: key.SourceAttribute,
						TargetAttribute: key.TargetAttribute,
					})
				}
				relations[relation.ID] = item
				if data.ID == dataID || relation.Target == dataID {
					ctx.Relations = append(ctx.Relations, item)
				}
			}
		}
	}

	for _, ns := range w.Semantics {
		for _, validation := range ns.Validations {
			relation, hasRelation := relations[validation.Relation]
			attributeRelevant := validation.Attribute != "" && attributeOwner[validation.Attribute] == dataID
			relationRelevant := hasRelation && (relation.Source == dataID || relation.Target == dataID)
			if !attributeRelevant && !relationRelevant {
				continue
			}
			ctx.Constraints = append(ctx.Constraints, DataDesignConstraint{
				ID:        validation.ID,
				Kind:      string(validation.Kind),
				Attribute: validation.Attribute,
				Relation:  validation.Relation,
			})
		}
	}

	sort.Slice(ctx.Attributes, func(i, j int) bool {
		return ctx.Attributes[i].ID < ctx.Attributes[j].ID
	})
	sort.Slice(ctx.Relations, func(i, j int) bool {
		if ctx.Relations[i].ID != ctx.Relations[j].ID {
			return ctx.Relations[i].ID < ctx.Relations[j].ID
		}
		if ctx.Relations[i].Source != ctx.Relations[j].Source {
			return ctx.Relations[i].Source < ctx.Relations[j].Source
		}
		return ctx.Relations[i].Target < ctx.Relations[j].Target
	})
	sort.Slice(ctx.Constraints, func(i, j int) bool {
		return ctx.Constraints[i].ID < ctx.Constraints[j].ID
	})
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
