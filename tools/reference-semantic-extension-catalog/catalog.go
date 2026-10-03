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
	ID             string                 `json:"id"`
	Namespace      string                 `json:"namespace"`
	Version        string                 `json:"version"`
	AppliesToKinds []string               `json:"appliesToKinds,omitempty"`
	Requires       []ExtensionRequirement `json:"requires,omitempty"`
}

type CatalogSubject struct {
	Subject   subject.Subject    `json:"subject"`
	Origin    string             `json:"origin"`
	Extension *ExtensionIdentity `json:"extension,omitempty"`
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

type SemanticCatalog struct {
	Schema          string              `json:"schema"`
	Extensions      []CatalogExtension  `json:"extensions,omitempty"`
	Subjects        []CatalogSubject    `json:"subjects"`
	Relations       []CatalogRelation   `json:"relations,omitempty"`
	Constraints     []CatalogConstraint `json:"constraints,omitempty"`
	Connections     []CatalogConnection `json:"connections,omitempty"`
	Unresolved      []CatalogUnresolved `json:"unresolved,omitempty"`
	AuthorityLimits []string            `json:"authorityLimits"`
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

func resolveSemanticCatalog(
	coreSubjects []subject.Subject,
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

	selected := map[string]ExtensionDescriptor{}
	visiting := map[string]bool{}
	var visit func(ExtensionRequirement) error
	visit = func(requirement ExtensionRequirement) error {
		requirement.Extension = strings.TrimSpace(requirement.Extension)
		if requirement.Extension == "" || requirement.CompatibleMajor < 0 {
			return fmt.Errorf("invalid extension requirement")
		}
		if existing, ok := selected[requirement.Extension]; ok {
			if !compatibleMajor(existing.Version, requirement.CompatibleMajor) {
				return fmt.Errorf(
					"extension %s already resolved to %s, incompatible with required major %d",
					requirement.Extension,
					existing.Version,
					requirement.CompatibleMajor,
				)
			}
			return nil
		}
		if visiting[requirement.Extension] {
			return fmt.Errorf("extension dependency cycle at %s", requirement.Extension)
		}
		var match *ExtensionDescriptor
		for i := range byID[requirement.Extension] {
			if compatibleMajor(byID[requirement.Extension][i].Version, requirement.CompatibleMajor) {
				copy := byID[requirement.Extension][i]
				match = &copy
				break
			}
		}
		if match == nil {
			return fmt.Errorf(
				"no semantic extension %s compatible with major %d",
				requirement.Extension,
				requirement.CompatibleMajor,
			)
		}
		visiting[requirement.Extension] = true
		for _, child := range match.Requires {
			if err := visit(child); err != nil {
				return fmt.Errorf("extension %s requires: %w", match.ID, err)
			}
		}
		visiting[requirement.Extension] = false
		selected[match.ID] = *match
		return nil
	}

	for _, root := range roots {
		if err := visit(root); err != nil {
			return SemanticCatalog{}, err
		}
	}

	catalog := SemanticCatalog{
		Schema: catalogSchema,
		Subjects: []CatalogSubject{},
		AuthorityLimits: []string{
			"catalog-is-not-canonical-authority",
			"extension-selection-is-not-acceptance",
			"connection-reference-is-not-runtime-authority",
			"reference-descriptor-is-not-production-dsl",
		},
	}

	subjectOwners := map[string][]CatalogSubject{}
	for _, coreSubject := range coreSubjects {
		normalizedSubject, err := normalizeSubject(coreSubject)
		if err != nil {
			return SemanticCatalog{}, fmt.Errorf("core subject: %w", err)
		}
		entry := CatalogSubject{Subject: normalizedSubject, Origin: "core"}
		subjectOwners[subjectKey(normalizedSubject)] = append(subjectOwners[subjectKey(normalizedSubject)], entry)
	}

	var ids []string
	for id := range selected {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		descriptor := selected[id]
		identity := ExtensionIdentity{ID: descriptor.ID, Version: descriptor.Version}
		catalog.Extensions = append(catalog.Extensions, CatalogExtension{
			ID:             descriptor.ID,
			Namespace:      descriptor.Namespace,
			Version:        descriptor.Version,
			AppliesToKinds: append([]string(nil), descriptor.AppliesToKinds...),
			Requires:       append([]ExtensionRequirement(nil), descriptor.Requires...),
		})
		for _, concept := range descriptor.Concepts {
			identityCopy := identity
			entry := CatalogSubject{
				Subject:   concept.Subject,
				Origin:    "extension",
				Extension: &identityCopy,
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
