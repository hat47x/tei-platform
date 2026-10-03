package main

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"tei/internal/subject"
)

const extensionSchema = "tei.reference.semantic-extension/v0"

type ExtensionRequirement struct {
	Extension       string `json:"extension"`
	CompatibleMajor int    `json:"compatibleMajor"`
}

type ExtensionConcept struct {
	Subject subject.Subject `json:"subject"`
}

type ExtensionRelation struct {
	From     subject.Subject `json:"from"`
	Relation string          `json:"relation"`
	To       subject.Subject `json:"to"`
}

type ExtensionConstraint struct {
	ID           string          `json:"id"`
	Subject      subject.Subject `json:"subject"`
	ValidatorRef string          `json:"validatorRef"`
}

type ExtensionConnection struct {
	Kind string `json:"kind"`
	Ref  string `json:"ref"`
}

type ExtensionDescriptor struct {
	Schema         string                `json:"schema"`
	ID             string                `json:"id"`
	Namespace      string                `json:"namespace"`
	Version        string                `json:"version"`
	AppliesToKinds []string              `json:"appliesToKinds,omitempty"`
	Requires       []ExtensionRequirement `json:"requires,omitempty"`
	Concepts       []ExtensionConcept     `json:"concepts,omitempty"`
	Relations      []ExtensionRelation    `json:"relations,omitempty"`
	Constraints    []ExtensionConstraint  `json:"constraints,omitempty"`
	Connections    []ExtensionConnection  `json:"connections,omitempty"`
}

func normalizeDescriptor(d ExtensionDescriptor) (ExtensionDescriptor, error) {
	d.Schema = strings.TrimSpace(d.Schema)
	d.ID = strings.TrimSpace(d.ID)
	d.Namespace = strings.TrimSpace(d.Namespace)
	d.Version = strings.TrimSpace(d.Version)
	if d.Schema != extensionSchema {
		return d, fmt.Errorf("extension %q uses unsupported schema %q", d.ID, d.Schema)
	}
	if d.ID == "" || d.Namespace == "" || d.Version == "" {
		return d, fmt.Errorf("extension requires id, namespace, and version")
	}
	if _, err := parseVersion(d.Version); err != nil {
		return d, fmt.Errorf("extension %s: %w", d.ID, err)
	}

	for i := range d.AppliesToKinds {
		d.AppliesToKinds[i] = strings.TrimSpace(d.AppliesToKinds[i])
		if d.AppliesToKinds[i] == "" {
			return d, fmt.Errorf("extension %s has empty appliesToKinds entry", d.ID)
		}
	}
	d.AppliesToKinds = uniqueSortedStrings(d.AppliesToKinds)

	seenRequirements := map[string]struct{}{}
	for i := range d.Requires {
		req := &d.Requires[i]
		req.Extension = strings.TrimSpace(req.Extension)
		if req.Extension == "" || req.CompatibleMajor < 0 {
			return d, fmt.Errorf("extension %s has invalid dependency", d.ID)
		}
		key := fmt.Sprintf("%s@%d", req.Extension, req.CompatibleMajor)
		if _, ok := seenRequirements[key]; ok {
			return d, fmt.Errorf("extension %s duplicates dependency %s", d.ID, key)
		}
		seenRequirements[key] = struct{}{}
	}
	sort.Slice(d.Requires, func(i, j int) bool {
		if d.Requires[i].Extension != d.Requires[j].Extension {
			return d.Requires[i].Extension < d.Requires[j].Extension
		}
		return d.Requires[i].CompatibleMajor < d.Requires[j].CompatibleMajor
	})

	seenConcepts := map[string]struct{}{}
	for i := range d.Concepts {
		s, err := normalizeSubject(d.Concepts[i].Subject)
		if err != nil {
			return d, fmt.Errorf("extension %s concept[%d]: %w", d.ID, i, err)
		}
		d.Concepts[i].Subject = s
		key := subjectKey(s)
		if _, ok := seenConcepts[key]; ok {
			return d, fmt.Errorf("extension %s duplicates concept %s", d.ID, key)
		}
		seenConcepts[key] = struct{}{}
	}
	sort.Slice(d.Concepts, func(i, j int) bool {
		return subjectKey(d.Concepts[i].Subject) < subjectKey(d.Concepts[j].Subject)
	})

	for i := range d.Relations {
		r := &d.Relations[i]
		var err error
		if r.From, err = normalizeSubject(r.From); err != nil {
			return d, fmt.Errorf("extension %s relation[%d].from: %w", d.ID, i, err)
		}
		if r.To, err = normalizeSubject(r.To); err != nil {
			return d, fmt.Errorf("extension %s relation[%d].to: %w", d.ID, i, err)
		}
		r.Relation = strings.TrimSpace(r.Relation)
		if r.Relation == "" {
			return d, fmt.Errorf("extension %s relation[%d] has empty relation", d.ID, i)
		}
	}
	sort.Slice(d.Relations, func(i, j int) bool {
		return relationKey(d.Relations[i]) < relationKey(d.Relations[j])
	})

	for i := range d.Constraints {
		c := &d.Constraints[i]
		c.ID = strings.TrimSpace(c.ID)
		c.ValidatorRef = strings.TrimSpace(c.ValidatorRef)
		var err error
		if c.Subject, err = normalizeSubject(c.Subject); err != nil {
			return d, fmt.Errorf("extension %s constraint[%d].subject: %w", d.ID, i, err)
		}
		if c.ID == "" || c.ValidatorRef == "" {
			return d, fmt.Errorf("extension %s constraint[%d] requires id and validatorRef", d.ID, i)
		}
	}
	sort.Slice(d.Constraints, func(i, j int) bool {
		if d.Constraints[i].ID != d.Constraints[j].ID {
			return d.Constraints[i].ID < d.Constraints[j].ID
		}
		return subjectKey(d.Constraints[i].Subject) < subjectKey(d.Constraints[j].Subject)
	})

	for i := range d.Connections {
		d.Connections[i].Kind = strings.TrimSpace(d.Connections[i].Kind)
		d.Connections[i].Ref = strings.TrimSpace(d.Connections[i].Ref)
		if d.Connections[i].Kind == "" || d.Connections[i].Ref == "" {
			return d, fmt.Errorf("extension %s connection[%d] requires kind and ref", d.ID, i)
		}
	}
	sort.Slice(d.Connections, func(i, j int) bool {
		if d.Connections[i].Kind != d.Connections[j].Kind {
			return d.Connections[i].Kind < d.Connections[j].Kind
		}
		return d.Connections[i].Ref < d.Connections[j].Ref
	})

	return d, nil
}

func normalizeSubject(s subject.Subject) (subject.Subject, error) {
	s.Kind = strings.TrimSpace(s.Kind)
	s.ID = strings.TrimSpace(s.ID)
	if s.Kind == "" || s.ID == "" {
		return s, fmt.Errorf("subject requires kind and id")
	}
	return s, nil
}

func subjectKey(s subject.Subject) string {
	return fmt.Sprintf("%q|%q", s.Kind, s.ID)
}

func relationKey(r ExtensionRelation) string {
	return subjectKey(r.From) + "|" + r.Relation + "|" + subjectKey(r.To)
}

func parseVersion(v string) ([3]int, error) {
	var out [3]int
	parts := strings.Split(v, ".")
	if len(parts) != 3 {
		return out, fmt.Errorf("version %q must be major.minor.patch", v)
	}
	for i, part := range parts {
		n, err := strconv.Atoi(part)
		if err != nil || n < 0 {
			return out, fmt.Errorf("version %q contains invalid numeric component", v)
		}
		out[i] = n
	}
	return out, nil
}

func compatibleMajor(v string, major int) bool {
	parsed, err := parseVersion(v)
	return err == nil && parsed[0] == major
}

func compareVersion(a, b string) int {
	av, _ := parseVersion(a)
	bv, _ := parseVersion(b)
	for i := range av {
		switch {
		case av[i] < bv[i]:
			return -1
		case av[i] > bv[i]:
			return 1
		}
	}
	return 0
}

func uniqueSortedStrings(values []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}
