package main

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"sort"
	"strings"
)

// Navigation is a scoped read-only reference lookup, never a write location
// for Canonical Source Acceptance. Source pointers belong to fixture snapshots.
type navigationSubject struct {
	Kind      string `json:"kind"`
	ID        string `json:"id"`
	Scope     string `json:"scope"`
	Authority string `json:"authority"`
}

type navigationLink struct {
	Relation string            `json:"relation"`
	Subject  navigationSubject `json:"subject"`
	Status   string            `json:"status"`
}

type navigationSnapshot struct {
	File   string `json:"file"`
	SHA256 string `json:"sha256"`
}

type navigationResult struct {
	SchemaVersion          string               `json:"schemaVersion"`
	Authority              string               `json:"authority"`
	Status                 string               `json:"status"`
	Selector               string               `json:"selector"`
	Subject                navigationSubject    `json:"subject"`
	Source                 *location            `json:"source,omitempty"`
	SourceSnapshots        []navigationSnapshot `json:"sourceSnapshots"`
	WorldRevisionStatus    string               `json:"worldRevisionStatus"`
	Links                  []navigationLink     `json:"links"`
	Editable               bool                 `json:"editable"`
	AcceptanceTargetStatus string               `json:"acceptanceTargetStatus"`
	Reason                 string               `json:"reason,omitempty"`
}

func snapshot(path string) (navigationSnapshot, error) {
	bytes, err := os.ReadFile(path)
	if err != nil {
		return navigationSnapshot{}, err
	}
	sum := sha256.Sum256(bytes)
	return navigationSnapshot{File: path, SHA256: "sha256:" + hex.EncodeToString(sum[:])}, nil
}

// navigate deliberately accepts a single producer Projection, not a fabricated
// accepted World. A Step is identified only inside the reference Procedure ID
// and cannot collide with an unrelated accepted business/task Subject.
func navigate(p projection, selector string, sourcePaths ...string) (navigationResult, error) {
	parts := strings.SplitN(selector, ":", 2)
	if len(parts) != 2 || strings.TrimSpace(parts[0]) != parts[0] || strings.TrimSpace(parts[1]) != parts[1] || parts[1] == "" {
		return navigationResult{}, fmt.Errorf("selection requires exact kind:id")
	}
	kind, id := parts[0], parts[1]
	const local = "reference-procedure:"
	scope := local + p.ProcedureID
	result := navigationResult{
		SchemaVersion:          "business-dfd-navigation-v0",
		Authority:              "derived-read-only",
		Status:                 "unresolved",
		Selector:               selector,
		Subject:                navigationSubject{Kind: kind, ID: id, Scope: scope, Authority: "reference-only"},
		SourceSnapshots:        []navigationSnapshot{},
		WorldRevisionStatus:    "unverified-no-model-revision-binding",
		Links:                  []navigationLink{},
		Editable:               false,
		AcceptanceTargetStatus: "unresolved-use-shared-acceptance",
	}
	for _, path := range sourcePaths {
		s, err := snapshot(path)
		if err != nil {
			return navigationResult{}, fmt.Errorf("snapshot %s: %w", path, err)
		}
		result.SourceSnapshots = append(result.SourceSnapshots, s)
	}
	for _, d := range p.Diagnostics {
		if d.Severity == "error" {
			result.Reason = "DFD producer has unresolved diagnostics; navigation must be revalidated"
			return result, nil
		}
	}
	localLink := func(relation, kind, id string) {
		result.Links = append(result.Links, navigationLink{
			Relation: relation, Subject: navigationSubject{Kind: kind, ID: id, Scope: scope, Authority: "reference-only"},
			Status: "reference-only-no-accepted-source",
		})
	}
	worldLink := func(relation, kind, id, status string) {
		result.Links = append(result.Links, navigationLink{
			Relation: relation, Subject: navigationSubject{Kind: kind, ID: id, Scope: "unverified-current-world", Authority: "identity-match-only"},
			Status: status,
		})
	}
	setSource := func(src location) { copied := src; result.Source = &copied; result.Status = "reference-only" }

	switch kind {
	case "step":
		count := 0
		for _, s := range p.Steps {
			if s.ID != id {
				continue
			}
			count++
			setSource(s.Source)
			for _, explicit := range s.ExplicitMeaning {
				worldLink("explicit-business-semantic-reference",
					explicit.Kind, explicit.ID, explicit.Status)
			}
			for _, f := range s.FunctionRefs {
				worldLink("explicit-application-pattern", "application-pattern", f.ID,
					map[bool]string{true: "resolved-id-not-accepted", false: "unresolved"}[f.Status == "resolved"])
				if f.Status == "resolved" {
					if f.Data != "" {
						worldLink("pattern-derived-data", "semantic-data", f.Data, "derived-not-authority")
					}
					if f.Relation != "" {
						worldLink("pattern-derived-relation", "semantic-relation", f.Relation, "derived-not-authority")
					}
				}
			}
			for _, e := range p.Flows {
				if e.From == id {
					localLink("proposed-control-next", "step", e.To)
				}
				if e.To == id {
					localLink("proposed-control-previous", "step", e.From)
				}
			}
			for _, a := range p.Information.Responsibilities {
				if a.Step == id {
					localLink("proposed-business-role", "business-role", a.Role)
				}
			}
			for _, e := range p.Information.InformationFlows {
				if e.From.Kind == "step" && e.From.ID == id {
					localLink("proposed-information-to", e.To.Kind, e.To.ID)
				}
				if e.To.Kind == "step" && e.To.ID == id {
					localLink("proposed-information-from", e.From.Kind, e.From.ID)
				}
			}
			for _, h := range p.Handoffs {
				if h.From.Procedure == p.ProcedureID && h.From.Step == id {
					result.Links = append(result.Links, navigationLink{
						Relation: "conditional-handoff-to",
						Subject: navigationSubject{Kind: "step", ID: h.To.Step,
							Scope: local + h.To.Procedure, Authority: "reference-only"},
						Status: "reference-set-member-only-remote-step-unverified",
					})
				}
			}
		}
		if count > 1 {
			result.Status = "unresolved"
			result.Source = nil
			result.Links = nil
			result.Reason = "ambiguous Step identity"
		}
	case "pattern":
		count := 0
		result.Subject = navigationSubject{Kind: "application-pattern", ID: id, Scope: "unverified-current-world", Authority: "identity-match-only"}
		for _, s := range p.Steps {
			for _, f := range s.FunctionRefs {
				if f.ID == id {
					count++
					localLink("explicitly-used-by-step", "step", s.ID)
					if f.Status != "resolved" {
						result.Reason = "Pattern reference unresolved in producer"
					}
				}
			}
		}
		if count > 0 && result.Reason == "" {
			result.Status = "reference-only"
		}
	case "semantic-data", "semantic-relation", "semantic-validation", "semantic-capability":
		result.Subject = navigationSubject{
			Kind: kind, ID: id, Scope: "unverified-current-world", Authority: "identity-match-only"}
		matchCount := 0
		for _, node := range p.Steps {
			for _, explicit := range node.ExplicitMeaning {
				if explicit.Kind == kind && explicit.ID == id && explicit.Status == "resolved-id-not-accepted" {
					localLink("explicitly-referenced-by-step", "step", node.ID)
					matchCount++
				}
			}
			for _, f := range node.FunctionRefs {
				if f.Status != "resolved" {
					continue
				}
				var referenced string
				switch kind {
				case "semantic-data":
					referenced = f.Data
				case "semantic-relation":
					referenced = f.Relation
				case "semantic-capability":
					referenced = f.Capability
				}
				if referenced == id && referenced != "" {
					localLink("pattern-derived-step-reference", "step", node.ID)
					matchCount++
				}
			}
		}
		if kind == "semantic-data" {
			for _, store := range p.Information.Stores {
				if store.SemanticData == id && store.DataStatus == "resolved" {
					localLink("proposed-conceptual-store", "store", store.ID)
					matchCount++
				}
			}
		}
		if matchCount > 0 {
			result.Status = "reference-only"
		}
	case "trigger":
		for _, t := range p.Triggers {
			if t.ID == id {
				setSource(t.Source)
				localLink("proposed-trigger-target", "step", t.TargetStep)
			}
		}
	case "business-role":
		for _, role := range p.Information.Roles {
			if role.ID == id {
				setSource(role.Source)
				for _, a := range p.Information.Responsibilities {
					if a.Role == id {
						localLink("proposed-responsibility", "step", a.Step)
					}
				}
			}
		}
	case "party":
		for _, party := range p.Information.ExternalParties {
			if party.ID == id {
				setSource(party.Source)
			}
		}
	case "store":
		for _, store := range p.Information.Stores {
			if store.ID == id {
				setSource(store.Source)
				if store.SemanticData != "" {
					worldLink("explicit-semantic-data", "semantic-data", store.SemanticData,
						map[bool]string{true: "resolved-id-not-accepted", false: "unresolved"}[store.DataStatus == "resolved"])
				}
			}
		}
	case "information-flow":
		for _, flow := range p.Information.InformationFlows {
			if flow.ID == id {
				setSource(flow.Source)
				localLink("proposed-information-from", flow.From.Kind, flow.From.ID)
				localLink("proposed-information-to", flow.To.Kind, flow.To.ID)
			}
		}
	default:
		return navigationResult{}, fmt.Errorf("unsupported selector kind %q", kind)
	}
	if result.Status == "unresolved" && result.Reason == "" {
		result.Reason = "exact reference element not found; no inferred alternative"
	}
	sort.Slice(result.Links, func(i, j int) bool {
		a, b := result.Links[i], result.Links[j]
		if a.Relation != b.Relation {
			return a.Relation < b.Relation
		}
		if a.Subject.Kind != b.Subject.Kind {
			return a.Subject.Kind < b.Subject.Kind
		}
		if a.Subject.Scope != b.Subject.Scope {
			return a.Subject.Scope < b.Subject.Scope
		}
		return a.Subject.ID < b.Subject.ID
	})
	return result, nil
}
