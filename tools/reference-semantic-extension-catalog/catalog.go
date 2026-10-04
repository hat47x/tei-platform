package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"tei/internal/sourceindex"
	"tei/internal/subject"
)

type ExtensionIdentity struct {
	ID      string `json:"id"`
	Version string `json:"version"`
}

type CatalogExtension struct {
	ID                    string                 `json:"id"`
	Namespace             string                 `json:"namespace"`
	Version               string                 `json:"version"`
	DescriptorFingerprint string                 `json:"descriptorFingerprint"`
	Source                string                 `json:"source,omitempty"`
	SourceFingerprint     string                 `json:"sourceFingerprint,omitempty"`
	AppliesToKinds        []string               `json:"appliesToKinds,omitempty"`
	Requires              []ExtensionRequirement `json:"requires,omitempty"`
}

type CatalogSource struct {
	Path        string           `json:"path"`
	Fingerprint string           `json:"fingerprint"`
	Role        sourceindex.Role `json:"role"`
}

type CatalogSubject struct {
	Subject   subject.Subject    `json:"subject"`
	Origin    string             `json:"origin"`
	Extension *ExtensionIdentity `json:"extension,omitempty"`
	Sources   []CatalogSource    `json:"sources,omitempty"`
}

type CatalogRelation struct {
	From      subject.Subject   `json:"from"`
	Relation  string            `json:"relation"`
	To        subject.Subject   `json:"to"`
	Extension ExtensionIdentity `json:"extension"`
}

type CatalogConstraint struct {
	ID           string            `json:"id"`
	Subject      subject.Subject   `json:"subject"`
	ValidatorRef string            `json:"validatorRef"`
	Extension    ExtensionIdentity `json:"extension"`
}

type CatalogConnection struct {
	Kind      string            `json:"kind"`
	Ref       string            `json:"ref"`
	Extension ExtensionIdentity `json:"extension"`
}

type CatalogUnresolved struct {
	Code       string            `json:"code"`
	Subject    *subject.Subject  `json:"subject,omitempty"`
	Extension  *ExtensionIdentity `json:"extension,omitempty"`
	Detail     string            `json:"detail"`
}

type CoreCatalogInput struct {
	Subjects               []subject.Subject
	Sources                map[string][]CatalogSource
	SubjectFingerprint     string
	SourceIndexFingerprint string
}

type SemanticCatalog struct {
	Schema                     string                 `json:"schema"`
	ResolutionFingerprint      string                 `json:"resolutionFingerprint"`
	EvidenceFingerprint        string                 `json:"evidenceFingerprint"`
	CoreSubjectFingerprint     string                 `json:"coreSubjectFingerprint,omitempty"`
	CoreSourceIndexFingerprint string                 `json:"coreSourceIndexFingerprint,omitempty"`
	Requirements               []ExtensionRequirement `json:"requirements"`
	Extensions             []CatalogExtension     `json:"extensions,omitempty"`
	Subjects        []CatalogSubject    `json:"subjects"`
	Relations       []CatalogRelation   `json:"relations,omitempty"`
	Constraints     []CatalogConstraint `json:"constraints,omitempty"`
	Connections     []CatalogConnection `json:"connections,omitempty"`
	Unresolved      []CatalogUnresolved `json:"unresolved,omitempty"`
	AuthorityLimits        []string               `json:"authorityLimits"`
}

const catalogSchema = "tei.reference.semantic-catalog/v0"

func loadExtensionDescriptors(root string) ([]ExtensionDescriptor, error) {
	var out []ExtensionDescriptor
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".json") {
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		decoder := json.NewDecoder(bytes.NewReader(content))
		decoder.DisallowUnknownFields()
		var descriptor ExtensionDescriptor
		if err := decoder.Decode(&descriptor); err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		if _, err := decoder.Token(); err != io.EOF {
			return fmt.Errorf("%s: JSONの後ろに余分な内容があります", path)
		}
		descriptor, err = normalizeDescriptor(descriptor)
		if err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		descriptor.sourcePath = filepath.ToSlash(rel)
		descriptor.sourceFingerprint = sourceindex.Fingerprint(content)
		out = append(out, descriptor)
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].ID != out[j].ID {
			return out[i].ID < out[j].ID
		}
		return compareVersion(out[i].Version, out[j].Version) < 0
	})
	return out, nil
}

func coreCatalogFromSourceIndex(index sourceindex.Index) CoreCatalogInput {
	normalized := sourceindex.Index{
		Artifacts:  append([]sourceindex.Artifact(nil), index.Artifacts...),
		References: append([]sourceindex.Reference(nil), index.References...),
	}
	normalized.Normalize()
	encoded, _ := json.Marshal(normalized)
	subjects := sourceIndexSubjects(normalized)
	subjectsEncoded, _ := json.Marshal(subjects)
	artifacts := make(map[string]sourceindex.Artifact, len(normalized.Artifacts))
	for _, artifact := range normalized.Artifacts {
		artifacts[artifact.Path] = artifact
	}
	sources := map[string][]CatalogSource{}
	for _, ref := range normalized.References {
		artifact, ok := artifacts[ref.ArtifactPath]
		if !ok {
			continue
		}
		key := subjectKey(ref.Subject)
		item := CatalogSource{
			Path:        ref.ArtifactPath,
			Fingerprint: artifact.Fingerprint,
			Role:        ref.Role,
		}
		sources[key] = append(sources[key], item)
	}
	for key := range sources {
		sources[key] = normalizeCatalogSources(sources[key])
	}
	return CoreCatalogInput{
		Subjects:               subjects,
		Sources:                sources,
		SubjectFingerprint:     sourceindex.Fingerprint(subjectsEncoded),
		SourceIndexFingerprint: sourceindex.Fingerprint(encoded),
	}
}

func sourceIndexSubjects(index sourceindex.Index) []subject.Subject {
	seen := map[string]subject.Subject{}
	for _, ref := range index.References {
		if ref.Subject.IsZero() {
			continue
		}
		seen[subjectKey(ref.Subject)] = ref.Subject
	}
	out := make([]subject.Subject, 0, len(seen))
	for _, item := range seen {
		out = append(out, item)
	}
	sort.Slice(out, func(i, j int) bool {
		return subjectKey(out[i]) < subjectKey(out[j])
	})
	return out
}

func resolveExtensionSelection(
	byID map[string][]ExtensionDescriptor,
	roots []ExtensionRequirement,
) (map[string]ExtensionDescriptor, error) {
	constraints := map[string]map[int]struct{}{}
	for _, root := range roots {
		addExtensionConstraint(constraints, root)
	}
	return searchExtensionSelection(byID, map[string]ExtensionDescriptor{}, constraints)
}

func searchExtensionSelection(
	byID map[string][]ExtensionDescriptor,
	selected map[string]ExtensionDescriptor,
	constraints map[string]map[int]struct{},
) (map[string]ExtensionDescriptor, error) {
	var ids []string
	for id := range constraints {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	var unresolvedID string
	var unresolvedMajor int
	for _, id := range ids {
		majors := constraints[id]
		if len(majors) != 1 {
			values := make([]int, 0, len(majors))
			for major := range majors {
				values = append(values, major)
			}
			sort.Ints(values)
			return nil, fmt.Errorf("extension %s requires incompatible majors %v", id, values)
		}
		var major int
		for value := range majors {
			major = value
		}
		if existing, ok := selected[id]; ok {
			if !compatibleMajor(existing.Version, major) {
				return nil, fmt.Errorf(
					"extension %s already resolved to %s, incompatible with required major %d",
					id,
					existing.Version,
					major,
				)
			}
			continue
		}
		unresolvedID = id
		unresolvedMajor = major
		break
	}

	if unresolvedID == "" {
		if err := validateExtensionDependencyAcyclic(selected); err != nil {
			return nil, err
		}
		return selected, nil
	}

	var compatible []ExtensionDescriptor
	for _, descriptor := range byID[unresolvedID] {
		if compatibleMajor(descriptor.Version, unresolvedMajor) {
			compatible = append(compatible, descriptor)
		}
	}
	if len(compatible) == 0 {
		return nil, fmt.Errorf(
			"no semantic extension %s compatible with major %d",
			unresolvedID,
			unresolvedMajor,
		)
	}

	var lastErr error
	for _, candidate := range compatible {
		nextSelected := cloneExtensionSelection(selected)
		nextSelected[unresolvedID] = candidate
		nextConstraints := cloneExtensionConstraints(constraints)
		for _, child := range candidate.Requires {
			addExtensionConstraint(nextConstraints, child)
		}

		resolved, err := searchExtensionSelection(byID, nextSelected, nextConstraints)
		if err == nil {
			return resolved, nil
		}
		lastErr = err
	}

	return nil, fmt.Errorf(
		"no globally compatible semantic extension %s for major %d: %w",
		unresolvedID,
		unresolvedMajor,
		lastErr,
	)
}

func addExtensionConstraint(
	constraints map[string]map[int]struct{},
	requirement ExtensionRequirement,
) {
	if constraints[requirement.Extension] == nil {
		constraints[requirement.Extension] = map[int]struct{}{}
	}
	constraints[requirement.Extension][requirement.CompatibleMajor] = struct{}{}
}

func cloneExtensionSelection(
	values map[string]ExtensionDescriptor,
) map[string]ExtensionDescriptor {
	out := make(map[string]ExtensionDescriptor, len(values))
	for key, value := range values {
		out[key] = value
	}
	return out
}

func cloneExtensionConstraints(
	values map[string]map[int]struct{},
) map[string]map[int]struct{} {
	out := make(map[string]map[int]struct{}, len(values))
	for id, majors := range values {
		copyMajors := make(map[int]struct{}, len(majors))
		for major := range majors {
			copyMajors[major] = struct{}{}
		}
		out[id] = copyMajors
	}
	return out
}

func validateExtensionDependencyAcyclic(
	selected map[string]ExtensionDescriptor,
) error {
	const (
		visiting = iota + 1
		visited
	)
	state := map[string]int{}
	var visit func(string) error
	visit = func(id string) error {
		switch state[id] {
		case visiting:
			return fmt.Errorf("extension dependency cycle at %s", id)
		case visited:
			return nil
		}

		state[id] = visiting
		descriptor, ok := selected[id]
		if ok {
			for _, requirement := range descriptor.Requires {
				if _, selectedDependency := selected[requirement.Extension]; !selectedDependency {
					continue
				}
				if err := visit(requirement.Extension); err != nil {
					return err
				}
			}
		}
		state[id] = visited
		return nil
	}

	var ids []string
	for id := range selected {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		if err := visit(id); err != nil {
			return err
		}
	}
	return nil
}

func resolveSemanticCatalog(
	core CoreCatalogInput,
	descriptors []ExtensionDescriptor,
	roots []ExtensionRequirement,
) (SemanticCatalog, error) {
	normalized := make([]ExtensionDescriptor, 0, len(descriptors))
	for _, descriptor := range descriptors {
		value, err := normalizeDescriptor(descriptor)
		if err != nil {
			return SemanticCatalog{}, err
		}
		normalized = append(normalized, value)
	}

	byID := map[string][]ExtensionDescriptor{}
	for _, descriptor := range normalized {
		byID[descriptor.ID] = append(byID[descriptor.ID], descriptor)
	}
	for id := range byID {
		sort.Slice(byID[id], func(i, j int) bool {
			return compareVersion(byID[id][i].Version, byID[id][j].Version) > 0
		})
		for i := 1; i < len(byID[id]); i++ {
			if byID[id][i].Version == byID[id][i-1].Version {
				return SemanticCatalog{}, fmt.Errorf("extension %s version %s is declared more than once", id, byID[id][i].Version)
			}
		}
	}

	normalizedRoots, err := normalizeRootRequirements(roots)
	if err != nil {
		return SemanticCatalog{}, err
	}
	selected, err := resolveExtensionSelection(byID, normalizedRoots)
	if err != nil {
		return SemanticCatalog{}, err
	}

	catalog := SemanticCatalog{
		Schema:                     catalogSchema,
		CoreSubjectFingerprint:     core.SubjectFingerprint,
		CoreSourceIndexFingerprint: core.SourceIndexFingerprint,
		Requirements:               normalizedRoots,
		Subjects:                    []CatalogSubject{},
		AuthorityLimits: []string{
			"catalog-is-not-canonical-authority",
			"extension-selection-is-not-acceptance",
			"connection-reference-is-not-runtime-authority",
			"reference-descriptor-is-not-production-dsl",
		},
	}

	subjectOwners := map[string][]CatalogSubject{}
	for _, coreSubject := range core.Subjects {
		normalizedSubject, err := normalizeSubject(coreSubject)
		if err != nil {
			return SemanticCatalog{}, fmt.Errorf("core subject: %w", err)
		}
		entry := CatalogSubject{
			Subject: normalizedSubject,
			Origin:  "core",
			Sources: append([]CatalogSource(nil), core.Sources[subjectKey(normalizedSubject)]...),
		}
		subjectOwners[subjectKey(normalizedSubject)] = append(subjectOwners[subjectKey(normalizedSubject)], entry)
	}

	var ids []string
	for id := range selected {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	namespaceOwners := map[string][]string{}
	for _, coreSubject := range core.Subjects {
		if coreSubject.Kind == "semantic" {
			namespaceOwners[coreSubject.ID] = append(namespaceOwners[coreSubject.ID], "core")
		}
	}
	for _, id := range ids {
		namespaceOwners[selected[id].Namespace] = append(namespaceOwners[selected[id].Namespace], id)
	}
	blockedExtensions := map[string]bool{}
	for namespace, owners := range namespaceOwners {
		if len(owners) < 2 {
			continue
		}
		sort.Strings(owners)
		for _, owner := range owners {
			if owner != "core" {
				blockedExtensions[owner] = true
			}
		}
		catalog.Unresolved = append(catalog.Unresolved, CatalogUnresolved{
			Code:   "namespace-ownership-conflict",
			Detail: fmt.Sprintf("namespace %q has multiple owners: %s", namespace, strings.Join(owners, ", ")),
		})
	}

	for _, id := range ids {
		descriptor := selected[id]
		identity := ExtensionIdentity{ID: descriptor.ID, Version: descriptor.Version}
		catalog.Extensions = append(catalog.Extensions, CatalogExtension{
			ID:                    descriptor.ID,
			Namespace:             descriptor.Namespace,
			Version:               descriptor.Version,
			DescriptorFingerprint: descriptorFingerprint(descriptor),
			Source:                descriptor.sourcePath,
			SourceFingerprint:     descriptor.sourceFingerprint,
			AppliesToKinds:        append([]string(nil), descriptor.AppliesToKinds...),
			Requires:              append([]ExtensionRequirement(nil), descriptor.Requires...),
		})
		if blockedExtensions[id] {
			continue
		}
		for _, concept := range descriptor.Concepts {
			identityCopy := identity
			var sources []CatalogSource
			if descriptor.sourcePath != "" && descriptor.sourceFingerprint != "" {
				sources = []CatalogSource{{
					Path:        descriptor.sourcePath,
					Fingerprint: descriptor.sourceFingerprint,
					Role:        sourceindex.RoleDefinition,
				}}
			}
			entry := CatalogSubject{
				Subject:   concept.Subject,
				Origin:    "extension",
				Extension: &identityCopy,
				Sources:   sources,
			}
			key := subjectKey(concept.Subject)
			subjectOwners[key] = append(subjectOwners[key], entry)
		}
	}

	for key, owners := range subjectOwners {
		if len(owners) == 1 {
			catalog.Subjects = append(catalog.Subjects, owners[0])
			continue
		}
		subjectCopy := owners[0].Subject
		var ownerLabels []string
		for _, owner := range owners {
			if owner.Extension == nil {
				ownerLabels = append(ownerLabels, "core")
			} else {
				ownerLabels = append(ownerLabels, owner.Extension.ID+"@"+owner.Extension.Version)
			}
		}
		sort.Strings(ownerLabels)
		catalog.Unresolved = append(catalog.Unresolved, CatalogUnresolved{
			Code:    "subject-conflict",
			Subject: &subjectCopy,
			Detail:  fmt.Sprintf("%s is declared by %s", key, strings.Join(ownerLabels, ", ")),
		})
	}
	sort.Slice(catalog.Subjects, func(i, j int) bool {
		return subjectKey(catalog.Subjects[i].Subject) < subjectKey(catalog.Subjects[j].Subject)
	})

	available := map[string]struct{}{}
	for _, entry := range catalog.Subjects {
		available[subjectKey(entry.Subject)] = struct{}{}
	}
	for _, id := range ids {
		if blockedExtensions[id] {
			continue
		}
		descriptor := selected[id]
		identity := ExtensionIdentity{ID: descriptor.ID, Version: descriptor.Version}
		for _, relation := range descriptor.Relations {
			if _, ok := available[subjectKey(relation.From)]; !ok {
				extensionCopy := identity
				subjectCopy := relation.From
				catalog.Unresolved = append(catalog.Unresolved, CatalogUnresolved{
					Code:      "relation-source-unavailable",
					Subject:   &subjectCopy,
					Extension: &extensionCopy,
					Detail:    relationKey(relation),
				})
				continue
			}
			if _, ok := available[subjectKey(relation.To)]; !ok {
				extensionCopy := identity
				subjectCopy := relation.To
				catalog.Unresolved = append(catalog.Unresolved, CatalogUnresolved{
					Code:      "relation-target-unavailable",
					Subject:   &subjectCopy,
					Extension: &extensionCopy,
					Detail:    relationKey(relation),
				})
				continue
			}
			catalog.Relations = append(catalog.Relations, CatalogRelation{
				From: relation.From, Relation: relation.Relation, To: relation.To, Extension: identity,
			})
		}
		for _, constraint := range descriptor.Constraints {
			if _, ok := available[subjectKey(constraint.Subject)]; !ok {
				extensionCopy := identity
				subjectCopy := constraint.Subject
				catalog.Unresolved = append(catalog.Unresolved, CatalogUnresolved{
					Code:      "constraint-subject-unavailable",
					Subject:   &subjectCopy,
					Extension: &extensionCopy,
					Detail:    constraint.ID,
				})
				continue
			}
			catalog.Constraints = append(catalog.Constraints, CatalogConstraint{
				ID: constraint.ID, Subject: constraint.Subject, ValidatorRef: constraint.ValidatorRef, Extension: identity,
			})
		}
		for _, connection := range descriptor.Connections {
			catalog.Connections = append(catalog.Connections, CatalogConnection{
				Kind: connection.Kind, Ref: connection.Ref, Extension: identity,
			})
		}
	}

	sort.Slice(catalog.Relations, func(i, j int) bool {
		left := ExtensionRelation{From: catalog.Relations[i].From, Relation: catalog.Relations[i].Relation, To: catalog.Relations[i].To}
		right := ExtensionRelation{From: catalog.Relations[j].From, Relation: catalog.Relations[j].Relation, To: catalog.Relations[j].To}
		if relationKey(left) != relationKey(right) {
			return relationKey(left) < relationKey(right)
		}
		return catalog.Relations[i].Extension.ID < catalog.Relations[j].Extension.ID
	})
	sort.Slice(catalog.Constraints, func(i, j int) bool {
		if catalog.Constraints[i].ID != catalog.Constraints[j].ID {
			return catalog.Constraints[i].ID < catalog.Constraints[j].ID
		}
		return catalog.Constraints[i].Extension.ID < catalog.Constraints[j].Extension.ID
	})
	sort.Slice(catalog.Connections, func(i, j int) bool {
		if catalog.Connections[i].Kind != catalog.Connections[j].Kind {
			return catalog.Connections[i].Kind < catalog.Connections[j].Kind
		}
		if catalog.Connections[i].Ref != catalog.Connections[j].Ref {
			return catalog.Connections[i].Ref < catalog.Connections[j].Ref
		}
		return catalog.Connections[i].Extension.ID < catalog.Connections[j].Extension.ID
	})
	sort.Slice(catalog.Unresolved, func(i, j int) bool {
		if catalog.Unresolved[i].Code != catalog.Unresolved[j].Code {
			return catalog.Unresolved[i].Code < catalog.Unresolved[j].Code
		}
		return catalog.Unresolved[i].Detail < catalog.Unresolved[j].Detail
	})
	catalog.ResolutionFingerprint = semanticCatalogFingerprint(catalog)
	catalog.EvidenceFingerprint = semanticCatalogEvidenceFingerprint(catalog)
	return catalog, nil
}

func filterCatalog(catalog SemanticCatalog, kind, id string) SemanticCatalog {
	kind = strings.TrimSpace(kind)
	id = strings.TrimSpace(id)
	if kind == "" && id == "" {
		return catalog
	}
	filtered := catalog
	filtered.Subjects = nil
	for _, item := range catalog.Subjects {
		if kind != "" && item.Subject.Kind != kind {
			continue
		}
		if id != "" && item.Subject.ID != id {
			continue
		}
		filtered.Subjects = append(filtered.Subjects, item)
	}
	filtered.Relations = nil
	for _, relation := range catalog.Relations {
		if (kind == "" || relation.From.Kind == kind || relation.To.Kind == kind) &&
			(id == "" || relation.From.ID == id || relation.To.ID == id) {
			filtered.Relations = append(filtered.Relations, relation)
		}
	}
	filtered.Constraints = nil
	for _, constraint := range catalog.Constraints {
		if (kind == "" || constraint.Subject.Kind == kind) &&
			(id == "" || constraint.Subject.ID == id) {
			filtered.Constraints = append(filtered.Constraints, constraint)
		}
	}
	return filtered
}


func normalizeRootRequirements(values []ExtensionRequirement) ([]ExtensionRequirement, error) {
	if len(values) == 0 {
		return nil, fmt.Errorf("at least one root extension requirement is required")
	}
	normalized := make([]ExtensionRequirement, 0, len(values))
	seen := map[string]struct{}{}
	majors := map[string]int{}
	for _, value := range values {
		value.Extension = strings.TrimSpace(value.Extension)
		if value.Extension == "" || value.CompatibleMajor < 0 {
			return nil, fmt.Errorf("invalid extension requirement")
		}
		if major, ok := majors[value.Extension]; ok && major != value.CompatibleMajor {
			return nil, fmt.Errorf(
				"root extension %s requires incompatible majors %d and %d",
				value.Extension,
				major,
				value.CompatibleMajor,
			)
		}
		majors[value.Extension] = value.CompatibleMajor
		key := fmt.Sprintf("%s@%d", value.Extension, value.CompatibleMajor)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		normalized = append(normalized, value)
	}
	sort.Slice(normalized, func(i, j int) bool {
		if normalized[i].Extension != normalized[j].Extension {
			return normalized[i].Extension < normalized[j].Extension
		}
		return normalized[i].CompatibleMajor < normalized[j].CompatibleMajor
	})
	return normalized, nil
}

func semanticCatalogFingerprint(catalog SemanticCatalog) string {
	catalog.ResolutionFingerprint = ""
	catalog.EvidenceFingerprint = ""
	catalog.CoreSourceIndexFingerprint = ""
	for i := range catalog.Extensions {
		catalog.Extensions[i].Source = ""
		catalog.Extensions[i].SourceFingerprint = ""
	}
	for i := range catalog.Subjects {
		catalog.Subjects[i].Sources = nil
	}
	encoded, _ := json.Marshal(catalog)
	return sourceindex.Fingerprint(encoded)
}

func semanticCatalogEvidenceFingerprint(catalog SemanticCatalog) string {
	catalog.ResolutionFingerprint = ""
	catalog.EvidenceFingerprint = ""
	encoded, _ := json.Marshal(catalog)
	return sourceindex.Fingerprint(encoded)
}


func normalizeCatalogSources(values []CatalogSource) []CatalogSource {
	seen := map[string]struct{}{}
	out := make([]CatalogSource, 0, len(values))
	for _, value := range values {
		key := fmt.Sprintf("%q|%q|%q", value.Path, value.Fingerprint, value.Role)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, value)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Role != out[j].Role {
			return out[i].Role < out[j].Role
		}
		if out[i].Path != out[j].Path {
			return out[i].Path < out[j].Path
		}
		return out[i].Fingerprint < out[j].Fingerprint
	})
	return out
}
