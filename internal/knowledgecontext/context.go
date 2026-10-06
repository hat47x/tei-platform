package knowledgecontext

import (
	"fmt"
	"sort"
	"strings"

	"tei/internal/core"
	"tei/internal/index"
	"tei/internal/ir"
	"tei/internal/semantic"
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

type DataDesignAmbiguity struct {
	Kind   string           `json:"kind"`
	ID     core.CanonicalID `json:"id"`
	Count  int              `json:"count"`
	Effect string           `json:"effect"`
}

type DataDesignScopedUniqueness struct {
	Constraint core.CanonicalID `json:"constraint"`
	Relation   core.CanonicalID `json:"relation"`
	Source     core.CanonicalID `json:"source"`
	Target     core.CanonicalID `json:"target"`
	Attribute  core.CanonicalID `json:"attribute"`
	Scope      string           `json:"scope"`
	Nature     string           `json:"nature"`
}

type DataDesignEvidence struct {
	Subject           subject.Subject   `json:"subject"`
	Source            string            `json:"source"`
	SourceFingerprint string            `json:"sourceFingerprint"`
	Range             sourceindex.Range `json:"range"`
	Role              sourceindex.Role  `json:"role"`
}

type DataDesignDeterminantTerm struct {
	Kind      string           `json:"kind"`
	Data      core.CanonicalID `json:"data,omitempty"`
	Attribute core.CanonicalID `json:"attribute,omitempty"`
}

type DataDesignBusinessIdentityCandidate struct {
	Relation         core.CanonicalID            `json:"relation"`
	Components       []DataDesignDeterminantTerm `json:"components"`
	TargetOccurrence core.CanonicalID            `json:"targetOccurrence"`
	BasisConstraint  core.CanonicalID            `json:"basisConstraint"`
	Scope            string                      `json:"scope"`
	Nature           string                      `json:"nature"`
	EvidenceNeeded   []string                    `json:"evidenceNeeded"`
}

type DataDesignFunctionalDependencyCandidate struct {
	Relation            core.CanonicalID            `json:"relation"`
	Determinants        []DataDesignDeterminantTerm `json:"determinants"`
	DependentOccurrence core.CanonicalID            `json:"dependentOccurrence"`
	BasisConstraint     core.CanonicalID            `json:"basisConstraint"`
	Scope               string                      `json:"scope"`
	Nature              string                      `json:"nature"`
	EvidenceNeeded      []string                    `json:"evidenceNeeded"`
}

type DataDesignPersistenceColumn struct {
	Attribute   core.CanonicalID `json:"attribute"`
	Column      string           `json:"column"`
	Storage     string           `json:"storage"`
	Nullability string           `json:"nullability"`
}

type DataDesignPersistenceUniqueness struct {
	Kind       string             `json:"kind"`
	Name       string             `json:"name,omitempty"`
	Attributes []core.CanonicalID `json:"attributes"`
	Nature     string             `json:"nature"`
}

type DataDesignPersistenceRealization struct {
	Mapping      core.CanonicalID                  `json:"mapping"`
	Database     core.CanonicalID                  `json:"database"`
	Table        string                            `json:"table"`
	Columns      []DataDesignPersistenceColumn     `json:"columns,omitempty"`
	Uniqueness   []DataDesignPersistenceUniqueness `json:"uniqueness,omitempty"`
	Authority    string                            `json:"authority"`
}

const dataDesignProjectionRevision = "tei.knowledge.data-design/v1"

type DataDesignContext struct {
	ProjectionRevision                    string                                    `json:"projectionRevision"`
	Fingerprint                           string                                    `json:"fingerprint"`
	Data                                  core.CanonicalID                          `json:"data"`
	Focus                                 subject.Subject                           `json:"focus"`
	Attributes                            []DataDesignAttribute                     `json:"attributes,omitempty"`
	Relations                             []DataDesignRelation                      `json:"relations,omitempty"`
	Constraints                           []DataDesignConstraint                    `json:"constraints,omitempty"`
	Ambiguities                           []DataDesignAmbiguity                     `json:"ambiguities,omitempty"`
	ScopedUniqueness                      []DataDesignScopedUniqueness              `json:"scopedUniqueness,omitempty"`
	BusinessIdentityCandidates            []DataDesignBusinessIdentityCandidate     `json:"businessIdentityCandidates,omitempty"`
	FunctionalDependencyCandidates        []DataDesignFunctionalDependencyCandidate `json:"functionalDependencyCandidates,omitempty"`
	PersistenceRealizations               []DataDesignPersistenceRealization        `json:"persistenceRealizations,omitempty"`
	Evidence                              []DataDesignEvidence                      `json:"evidence,omitempty"`
	EvidenceMissing                       []subject.Subject                         `json:"evidenceMissing,omitempty"`
	BusinessOccurrenceIdentifier          DataDesignKnowledgeState                  `json:"businessOccurrenceIdentifier"`
	CandidateKeys                         DataDesignKnowledgeState                  `json:"candidateKeys"`
	FunctionalDependencies                DataDesignKnowledgeState                  `json:"functionalDependencies"`
	BusinessIdentityEvidenceQuestions     []string                                  `json:"businessIdentityEvidenceQuestions"`
	FunctionalDependencyEvidenceQuestions []string                                  `json:"functionalDependencyEvidenceQuestions"`
	RelationEvidenceQuestions             []string                                  `json:"relationEvidenceQuestions"`
	AuthorityLimits                       []string                                  `json:"authorityLimits"`
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

func validationDataID(w ir.World, validationID core.CanonicalID) (core.CanonicalID, bool) {
	dataCounts := map[core.CanonicalID]int{}
	attributeOwners := map[core.CanonicalID]core.CanonicalID{}
	attributeCounts := map[core.CanonicalID]int{}
	relationTargets := map[core.CanonicalID]core.CanonicalID{}
	relationCounts := map[core.CanonicalID]int{}
	for _, ns := range w.Semantics {
		for _, data := range ns.Data {
			dataCounts[data.ID]++
			for _, attribute := range data.Attributes {
				attributeCounts[attribute.ID]++
				if attributeCounts[attribute.ID] == 1 {
					attributeOwners[attribute.ID] = data.ID
				}
			}
			for _, relation := range data.Relations {
				relationCounts[relation.ID]++
				if relationCounts[relation.ID] == 1 {
					relationTargets[relation.ID] = relation.Target
				}
			}
		}
	}

	var resolved core.CanonicalID
	matches := 0
	for _, ns := range w.Semantics {
		for _, validation := range ns.Validations {
			if validation.ID != validationID {
				continue
			}
			matches++
			var candidate core.CanonicalID
			if validation.Attribute != "" {
				owner, ok := attributeOwners[validation.Attribute]
				if !ok || attributeCounts[validation.Attribute] != 1 {
					return "", false
				}
				candidate = owner
			}
			if validation.Relation != "" {
				target, ok := relationTargets[validation.Relation]
				if !ok || relationCounts[validation.Relation] != 1 {
					return "", false
				}
				if candidate != "" && candidate != target {
					return "", false
				}
				candidate = target
			}
			if candidate == "" || dataCounts[candidate] != 1 {
				return "", false
			}
			resolved = candidate
		}
	}
	return resolved, matches == 1 && resolved != ""
}

func buildDataDesignContext(
	w ir.World,
	sources sourceindex.Index,
	target subject.Subject,
	element *index.SemanticElementEntry,
	pattern *index.ApplicationPatternIndexEntry,
) *DataDesignContext {
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
	case "semantic-validation":
		resolved, ok := validationDataID(w, core.CanonicalID(target.ID))
		if !ok {
			return nil
		}
		dataID = resolved
	case "application-pattern":
		if pattern == nil || pattern.Data == "" {
			return nil
		}
		dataID = pattern.Data
	case "persistence":
		var matched core.CanonicalID
		count := 0
		for _, mapping := range w.Persistence {
			if string(mapping.ID) != target.ID {
				continue
			}
			count++
			matched = mapping.SemanticData
		}
		if count != 1 || matched == "" {
			return nil
		}
		dataID = matched
	default:
		return nil
	}

	ctx := &DataDesignContext{
		ProjectionRevision: dataDesignProjectionRevision,
		Data:               dataID,
		Focus:              target,
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
			"does-persistence-key-represent-business-identity-or-only-realization",
			"if-surrogate-key-is-used-which-accepted-uniqueness-preserves-business-identity",
		},
		FunctionalDependencyEvidenceQuestions: []string{
			"which-determinants-determine-which-dependents",
			"under-what-scope-and-time-does-the-dependency-hold",
			"which-source-or-evidence-establishes-the-dependency",
			"is-this-a-functional-dependency-or-a-domain-constraint",
			"does-scoped-uniqueness-imply-business-functional-dependency-or-only-local-constraint",
		},
		RelationEvidenceQuestions: []string{
			"what-business-meaning-does-the-relation-express",
			"is-the-relation-a-changeable-reference-parent-contained-repeat-or-independent-association",
			"what-cardinality-is-required",
			"is-the-relation-current-only-or-temporal",
			"what-history-or-validity-period-must-be-preserved",
			"which-source-or-contract-owns-the-relation",
		},
		AuthorityLimits: []string{
			"data-design-context-is-derived-not-authority",
			"data-design-fingerprint-is-cache-invalidation-not-semantic-proof",
			"absence-of-authored-business-identifier-is-not-no-identifier",
			"business-identity-candidate-is-not-business-identifier-or-candidate-key",
			"absence-of-authored-functional-dependency-is-not-no-dependency",
			"functional-dependency-candidate-is-not-accepted-functional-dependency",
			"ambiguous-semantic-identity-does-not-produce-design-candidates",
			"relation-key-is-not-business-identifier",
			"scoped-uniqueness-is-not-business-identifier-or-candidate-key",
			"persistence-primary-key-is-not-business-identifier-or-candidate-key",
			"persistence-unique-constraint-is-not-business-identifier-or-candidate-key",
			"persistence-semantic-data-reference-is-reverse-trace-not-business-identity",
			"application-pattern-data-reference-is-context-not-business-identity",
			"attribute-name-or-id-shape-is-not-key-evidence",
			"identifier-like-attribute-is-not-business-identity",
			"identifier-value-presence-is-not-uniqueness-evidence",
			"normalized-string-equality-is-not-business-identity-evidence",
			"application-pattern-ui-or-runtime-identity-is-not-data-identity",
		},
	}

	dataCount := map[core.CanonicalID]int{}
	attributeOwner := map[core.CanonicalID]core.CanonicalID{}
	attributeCount := map[core.CanonicalID]int{}
	relations := map[core.CanonicalID]DataDesignRelation{}
	relationCount := map[core.CanonicalID]int{}
	validationCount := map[core.CanonicalID]int{}
	for _, ns := range w.Semantics {
		for _, validation := range ns.Validations {
			validationCount[validation.ID]++
		}
	}
	evidenceSubjects := map[subject.Subject]struct{}{
		{Kind: "semantic-data", ID: string(dataID)}: {},
	}
	for _, ns := range w.Semantics {
		for _, data := range ns.Data {
			dataCount[data.ID]++
			for _, attribute := range data.Attributes {
				attributeCount[attribute.ID]++
				if attributeCount[attribute.ID] == 1 {
					attributeOwner[attribute.ID] = data.ID
				}
				if data.ID == dataID {
					ctx.Attributes = append(ctx.Attributes, DataDesignAttribute{ID: attribute.ID, Type: attribute.TypeRef})
					evidenceSubjects[subject.Subject{Kind: "semantic-attribute", ID: string(attribute.ID)}] = struct{}{}
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
				relationCount[relation.ID]++
				if relationCount[relation.ID] == 1 {
					relations[relation.ID] = item
				}
				if data.ID == dataID || relation.Target == dataID {
					ctx.Relations = append(ctx.Relations, item)
					evidenceSubjects[subject.Subject{Kind: "semantic-relation", ID: string(relation.ID)}] = struct{}{}
					evidenceSubjects[subject.Subject{Kind: "semantic-data", ID: string(data.ID)}] = struct{}{}
					evidenceSubjects[subject.Subject{Kind: "semantic-data", ID: string(relation.Target)}] = struct{}{}
					for _, key := range relation.Keys {
						evidenceSubjects[subject.Subject{Kind: "semantic-attribute", ID: string(key.SourceAttribute)}] = struct{}{}
						evidenceSubjects[subject.Subject{Kind: "semantic-attribute", ID: string(key.TargetAttribute)}] = struct{}{}
					}
				}
			}
		}
	}

	for _, mapping := range w.Persistence {
		if mapping.SemanticData != dataID {
			continue
		}
		realization := DataDesignPersistenceRealization{
			Mapping:   mapping.ID,
			Database:  mapping.Database,
			Table:     mapping.Table,
			Authority: "database-owned-realization",
		}
		for _, column := range mapping.Columns {
			realization.Columns = append(realization.Columns, DataDesignPersistenceColumn{
				Attribute:   column.Attribute,
				Column:      column.Column,
				Storage:     string(column.Storage),
				Nullability: string(column.Nullability),
			})
			evidenceSubjects[subject.Subject{Kind: "semantic-attribute", ID: string(column.Attribute)}] = struct{}{}
		}
		if len(mapping.PrimaryKey) != 0 {
			realization.Uniqueness = append(realization.Uniqueness, DataDesignPersistenceUniqueness{
				Kind:       "primary-key",
				Attributes: append([]core.CanonicalID(nil), mapping.PrimaryKey...),
				Nature:     "accepted-realization",
			})
		}
		for _, unique := range mapping.UniqueConstraints {
			realization.Uniqueness = append(realization.Uniqueness, DataDesignPersistenceUniqueness{
				Kind:       "unique-constraint",
				Name:       unique.Name,
				Attributes: append([]core.CanonicalID(nil), unique.Attributes...),
				Nature:     "accepted-realization",
			})
		}
		sort.Slice(realization.Columns, func(i, j int) bool {
			return realization.Columns[i].Attribute < realization.Columns[j].Attribute
		})
		sort.Slice(realization.Uniqueness, func(i, j int) bool {
			if realization.Uniqueness[i].Kind != realization.Uniqueness[j].Kind {
				return realization.Uniqueness[i].Kind < realization.Uniqueness[j].Kind
			}
			if realization.Uniqueness[i].Name != realization.Uniqueness[j].Name {
				return realization.Uniqueness[i].Name < realization.Uniqueness[j].Name
			}
			return persistenceUniquenessKey(realization.Uniqueness[i]) < persistenceUniquenessKey(realization.Uniqueness[j])
		})
		ctx.PersistenceRealizations = append(ctx.PersistenceRealizations, realization)
		evidenceSubjects[subject.Subject{Kind: "persistence", ID: string(mapping.ID)}] = struct{}{}
	}

	for _, ns := range w.Semantics {
		for _, validation := range ns.Validations {
			relation, hasRelation := relations[validation.Relation]
			hasRelation = hasRelation && relationCount[validation.Relation] == 1
			attributeRelevant := validation.Attribute != "" &&
				attributeCount[validation.Attribute] == 1 &&
				attributeOwner[validation.Attribute] == dataID
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
			evidenceSubjects[subject.Subject{Kind: "semantic-validation", ID: string(validation.ID)}] = struct{}{}
			if validation.Attribute != "" {
				evidenceSubjects[subject.Subject{Kind: "semantic-attribute", ID: string(validation.Attribute)}] = struct{}{}
			}
			if validation.Relation != "" {
				evidenceSubjects[subject.Subject{Kind: "semantic-relation", ID: string(validation.Relation)}] = struct{}{}
			}
			if validation.Kind == semantic.ValidationRelationUnique &&
				validationCount[validation.ID] == 1 &&
				hasRelation &&
				dataCount[relation.Source] == 1 &&
				dataCount[relation.Target] == 1 &&
				relation.Cardinality == "many" &&
				validation.Attribute != "" &&
				attributeCount[validation.Attribute] == 1 &&
				attributeOwner[validation.Attribute] == relation.Target {
				ctx.ScopedUniqueness = append(ctx.ScopedUniqueness, DataDesignScopedUniqueness{
					Constraint: validation.ID,
					Relation:   validation.Relation,
					Source:     relation.Source,
					Target:     relation.Target,
					Attribute:  validation.Attribute,
					Scope:      "per-relation-source-occurrence",
					Nature:     "accepted",
				})
				identityTerms := []DataDesignDeterminantTerm{
					{
						Kind: "relation-source-occurrence",
						Data: relation.Source,
					},
					{
						Kind:      "attribute",
						Data:      relation.Target,
						Attribute: validation.Attribute,
					},
				}
				ctx.BusinessIdentityCandidates = append(
					ctx.BusinessIdentityCandidates,
					DataDesignBusinessIdentityCandidate{
						Relation:         validation.Relation,
						Components:       append([]DataDesignDeterminantTerm(nil), identityTerms...),
						TargetOccurrence: relation.Target,
						BasisConstraint:  validation.ID,
						Scope:            "per-relation-source-occurrence",
						Nature:           "candidate-from-accepted-scoped-uniqueness",
						EvidenceNeeded: []string{
							"source-occurrence-business-identifier",
							"relation-business-meaning",
							"target-membership-exclusivity",
							"identifier-stability",
							"temporal-validity",
						},
					},
				)
				ctx.FunctionalDependencyCandidates = append(
					ctx.FunctionalDependencyCandidates,
					DataDesignFunctionalDependencyCandidate{
						Relation:            validation.Relation,
						Determinants:        append([]DataDesignDeterminantTerm(nil), identityTerms...),
						DependentOccurrence: relation.Target,
						BasisConstraint:     validation.ID,
						Scope:               "per-relation-source-occurrence",
						Nature:              "candidate-from-accepted-scoped-uniqueness",
						EvidenceNeeded: []string{
							"source-occurrence-business-identifier",
							"relation-business-meaning",
							"target-membership-exclusivity",
							"temporal-validity",
						},
					},
				)
			}
		}
	}

	if dataCount[dataID] > 1 {
		ctx.Ambiguities = append(ctx.Ambiguities, DataDesignAmbiguity{
			Kind:   "duplicate-semantic-data",
			ID:     dataID,
			Count:  dataCount[dataID],
			Effect: "design-candidates-suppressed",
		})
	}
	for id, count := range attributeCount {
		if count <= 1 {
			continue
		}
		if _, relevant := evidenceSubjects[subject.Subject{Kind: "semantic-attribute", ID: string(id)}]; !relevant {
			continue
		}
		ctx.Ambiguities = append(ctx.Ambiguities, DataDesignAmbiguity{
			Kind:   "duplicate-semantic-attribute",
			ID:     id,
			Count:  count,
			Effect: "design-candidates-suppressed",
		})
	}
	for id, count := range relationCount {
		if count <= 1 {
			continue
		}
		if _, relevant := evidenceSubjects[subject.Subject{Kind: "semantic-relation", ID: string(id)}]; !relevant {
			continue
		}
		ctx.Ambiguities = append(ctx.Ambiguities, DataDesignAmbiguity{
			Kind:   "duplicate-semantic-relation",
			ID:     id,
			Count:  count,
			Effect: "design-candidates-suppressed",
		})
	}
	for id, count := range validationCount {
		if count <= 1 {
			continue
		}
		if _, relevant := evidenceSubjects[subject.Subject{Kind: "semantic-validation", ID: string(id)}]; !relevant {
			continue
		}
		ctx.Ambiguities = append(ctx.Ambiguities, DataDesignAmbiguity{
			Kind:   "duplicate-semantic-validation",
			ID:     id,
			Count:  count,
			Effect: "design-candidates-suppressed",
		})
	}

	ctx.Evidence, ctx.EvidenceMissing = collectDataDesignEvidence(sources, evidenceSubjects)

	sort.Slice(ctx.Ambiguities, func(i, j int) bool {
		if ctx.Ambiguities[i].Kind != ctx.Ambiguities[j].Kind {
			return ctx.Ambiguities[i].Kind < ctx.Ambiguities[j].Kind
		}
		return ctx.Ambiguities[i].ID < ctx.Ambiguities[j].ID
	})
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
	sort.Slice(ctx.ScopedUniqueness, func(i, j int) bool {
		return ctx.ScopedUniqueness[i].Constraint < ctx.ScopedUniqueness[j].Constraint
	})
	sort.Slice(ctx.BusinessIdentityCandidates, func(i, j int) bool {
		if ctx.BusinessIdentityCandidates[i].BasisConstraint != ctx.BusinessIdentityCandidates[j].BasisConstraint {
			return ctx.BusinessIdentityCandidates[i].BasisConstraint < ctx.BusinessIdentityCandidates[j].BasisConstraint
		}
		return ctx.BusinessIdentityCandidates[i].TargetOccurrence < ctx.BusinessIdentityCandidates[j].TargetOccurrence
	})
	sort.Slice(ctx.FunctionalDependencyCandidates, func(i, j int) bool {
		if ctx.FunctionalDependencyCandidates[i].BasisConstraint != ctx.FunctionalDependencyCandidates[j].BasisConstraint {
			return ctx.FunctionalDependencyCandidates[i].BasisConstraint < ctx.FunctionalDependencyCandidates[j].BasisConstraint
		}
		return ctx.FunctionalDependencyCandidates[i].DependentOccurrence < ctx.FunctionalDependencyCandidates[j].DependentOccurrence
	})
	sort.Slice(ctx.PersistenceRealizations, func(i, j int) bool {
		return ctx.PersistenceRealizations[i].Mapping < ctx.PersistenceRealizations[j].Mapping
	})
	ctx.Fingerprint = dataDesignFingerprint(ctx)
	return ctx
}

func dataDesignFingerprint(ctx *DataDesignContext) string {
	var builder strings.Builder
	builder.WriteString(ctx.ProjectionRevision)
	builder.WriteByte('\n')
	builder.WriteString(string(ctx.Data))
	builder.WriteByte('\n')
	builder.WriteString(ctx.Focus.Kind)
	builder.WriteByte(':')
	builder.WriteString(ctx.Focus.ID)
	builder.WriteByte('\n')
	for _, ambiguity := range ctx.Ambiguities {
		builder.WriteString("ambiguity:")
		builder.WriteString(ambiguity.Kind)
		builder.WriteByte(':')
		builder.WriteString(string(ambiguity.ID))
		builder.WriteByte(':')
		builder.WriteString(fmt.Sprintf("%d", ambiguity.Count))
		builder.WriteByte('|')
		builder.WriteString(ambiguity.Effect)
		builder.WriteByte('\n')
	}
	for _, evidence := range ctx.Evidence {
		builder.WriteString(evidence.Subject.Kind)
		builder.WriteByte(':')
		builder.WriteString(evidence.Subject.ID)
		builder.WriteByte('|')
		builder.WriteString(evidence.Source)
		builder.WriteByte('|')
		builder.WriteString(evidence.SourceFingerprint)
		builder.WriteByte('|')
		builder.WriteString(string(evidence.Role))
		builder.WriteByte('\n')
	}
	for _, missing := range ctx.EvidenceMissing {
		builder.WriteString("missing:")
		builder.WriteString(missing.Kind)
		builder.WriteByte(':')
		builder.WriteString(missing.ID)
		builder.WriteByte('\n')
	}
	return sourceindex.Fingerprint([]byte(builder.String()))
}

func persistenceUniquenessKey(value DataDesignPersistenceUniqueness) string {
	parts := make([]string, len(value.Attributes))
	for i, attribute := range value.Attributes {
		parts[i] = string(attribute)
	}
	return strings.Join(parts, "\x1f")
}

func collectDataDesignEvidence(
	sources sourceindex.Index,
	subjects map[subject.Subject]struct{},
) ([]DataDesignEvidence, []subject.Subject) {
	artifactFingerprints := make(map[string]string, len(sources.Artifacts))
	for _, artifact := range sources.Artifacts {
		artifactFingerprints[artifact.Path] = artifact.Fingerprint
	}
	var evidence []DataDesignEvidence
	var missing []subject.Subject
	for item := range subjects {
		found := false
		for _, ref := range sources.ReferencesFor(item) {
			if ref.Role != sourceindex.RoleDefinition && ref.Role != sourceindex.RoleAuthoring {
				continue
			}
			fingerprint := artifactFingerprints[ref.ArtifactPath]
			if fingerprint == "" {
				continue
			}
			found = true
			evidence = append(evidence, DataDesignEvidence{
				Subject:           item,
				Source:            ref.ArtifactPath,
				SourceFingerprint: fingerprint,
				Range:             ref.Range,
				Role:              ref.Role,
			})
		}
		if !found {
			missing = append(missing, item)
		}
	}
	sort.Slice(evidence, func(i, j int) bool {
		if evidence[i].Subject.Kind != evidence[j].Subject.Kind {
			return evidence[i].Subject.Kind < evidence[j].Subject.Kind
		}
		if evidence[i].Subject.ID != evidence[j].Subject.ID {
			return evidence[i].Subject.ID < evidence[j].Subject.ID
		}
		if evidence[i].Source != evidence[j].Source {
			return evidence[i].Source < evidence[j].Source
		}
		return evidence[i].Range.Start.Offset < evidence[j].Range.Start.Offset
	})
	sort.Slice(missing, func(i, j int) bool {
		if missing[i].Kind != missing[j].Kind {
			return missing[i].Kind < missing[j].Kind
		}
		return missing[i].ID < missing[j].ID
	})
	return evidence, missing
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
