package main

import (
	"fmt"
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	"tei/internal/ir"
	"tei/internal/sourceindex"
	"tei/internal/workspace"
)

type actorContextR6KnowledgeCandidate struct {
	Identity     string   `json:"identity"`
	SubjectKind  string   `json:"subjectKind"`
	ReferencedBy []string `json:"referencedBy"`
}

type actorContextR6QualifiedKnowledge struct {
	Identity                  string                           `json:"identity"`
	SubjectKind               string                           `json:"subjectKind"`
	OwnerApplication          string                           `json:"ownerApplication"`
	RequiredInputFingerprints []actorContextR0InputFingerprint `json:"requiredInputFingerprints"`
	SourceRefs                []string                         `json:"sourceRefs"`
}

type actorContextR6RoutingRejection struct {
	Identity string `json:"identity"`
	Reason   string `json:"reason"`
}

type actorContextR6KnowledgeDisclosure struct {
	Identity         string   `json:"identity"`
	OwnerApplication string   `json:"ownerApplication"`
	SourceRefs       []string `json:"sourceRefs"`
}

type actorContextR6RoutingProbe struct {
	TaskStep         string                              `json:"taskStep"`
	RoutingMechanism string                              `json:"routingMechanism"`
	Available        []actorContextR6KnowledgeCandidate  `json:"available"`
	Selected         []actorContextR6KnowledgeCandidate  `json:"selected"`
	Qualified        []actorContextR6QualifiedKnowledge  `json:"qualified"`
	Rejected         []actorContextR6RoutingRejection    `json:"rejected,omitempty"`
	Disclosed        []actorContextR6KnowledgeDisclosure `json:"disclosed"`
	AuthorityLimits  []string                            `json:"authorityLimits"`
}

func TestActorContextR6SeparatesAvailableSelectedQualifiedAndDisclosedKnowledge(t *testing.T) {
	root := repoPath()
	workspacePath := filepath.Join(root, "reference", "workspace")
	fixturePath := filepath.Join(root, "reference", "business-system-design", "order-entry-business-procedure.json")

	loaded, err := workspace.Load(workspacePath)
	if err != nil {
		t.Fatal(err)
	}
	f, err := readFixture(fixturePath)
	if err != nil {
		t.Fatal(err)
	}
	sourceIdx, err := workspace.BuildSourceIndex(workspacePath, loaded)
	if err != nil {
		t.Fatal(err)
	}

	exact, err := deriveActorContextR6RoutingProbe(
		loaded.World,
		sourceIdx,
		f,
		"BP-02",
		"exact-deterministic",
		actorContextR6SelectExact,
	)
	if err != nil {
		t.Fatal(err)
	}
	broad, err := deriveActorContextR6RoutingProbe(
		loaded.World,
		sourceIdx,
		f,
		"BP-02",
		"broad-test-router",
		actorContextR6SelectBroad,
	)
	if err != nil {
		t.Fatal(err)
	}

	if len(exact.Available) < 2 {
		t.Fatalf("R6 pressure requires more than one available Application Pattern candidate, got %+v", exact.Available)
	}
	if len(exact.Selected) != 1 || exact.Selected[0].Identity != "order_entry.customer_reference" {
		t.Fatalf("exact router did not select the BP-02 Pattern only: %+v", exact.Selected)
	}
	if len(broad.Selected) <= len(exact.Selected) {
		t.Fatalf("broad test router did not over-select candidates for qualification pressure: exact=%d broad=%d", len(exact.Selected), len(broad.Selected))
	}
	if len(broad.Rejected) == 0 {
		t.Fatal("broad test router over-selection was not rejected by deterministic qualification")
	}

	exactQualified := actorContextR6QualifiedIdentities(exact.Qualified)
	broadQualified := actorContextR6QualifiedIdentities(broad.Qualified)
	if !reflect.DeepEqual(exactQualified, broadQualified) {
		t.Fatalf("routing mechanism changed qualified semantic applicability: exact=%v broad=%v", exactQualified, broadQualified)
	}
	if !reflect.DeepEqual(exactQualified, []string{"order_entry.customer_reference"}) {
		t.Fatalf("R6 qualification lost the exact BP-02 Pattern: %v", exactQualified)
	}

	exactDisclosed := actorContextR6DisclosedIdentities(exact.Disclosed)
	broadDisclosed := actorContextR6DisclosedIdentities(broad.Disclosed)
	if !reflect.DeepEqual(exactDisclosed, broadDisclosed) {
		t.Fatalf("routing mechanism changed disclosed qualified knowledge: exact=%v broad=%v", exactDisclosed, broadDisclosed)
	}
	if len(exact.Disclosed) >= len(exact.Available) {
		t.Fatalf("R6 disclosed the whole available knowledge set instead of the qualified subset: available=%d disclosed=%d", len(exact.Available), len(exact.Disclosed))
	}

	for _, qualified := range exact.Qualified {
		if len(qualified.RequiredInputFingerprints) == 0 || len(qualified.SourceRefs) == 0 {
			t.Fatalf("qualified knowledge lost current source provenance: %+v", qualified)
		}
		if !actorContextR0HasInputSubject(qualified.RequiredInputFingerprints, "application-pattern", qualified.Identity) {
			t.Fatalf("qualified knowledge is not freshness-bound to its Application Pattern source: %+v", qualified)
		}
	}

	actorContextR6AssertNoRoutingAuthorityShortcut(t, reflect.TypeOf(actorContextR6RoutingProbe{}))
}

func deriveActorContextR6RoutingProbe(
	world ir.World,
	sourceIdx sourceindex.Index,
	f fixture,
	stepID,
	mechanism string,
	selector func([]actorContextR6KnowledgeCandidate, string) []actorContextR6KnowledgeCandidate,
) (actorContextR6RoutingProbe, error) {
	available := actorContextR6AvailablePatternCandidates(f)
	selected := selector(available, stepID)
	qualified, rejected, err := actorContextR6QualifySelection(world, sourceIdx, f, stepID, selected)
	if err != nil {
		return actorContextR6RoutingProbe{}, err
	}

	disclosed := make([]actorContextR6KnowledgeDisclosure, 0, len(qualified))
	for _, item := range qualified {
		disclosed = append(disclosed, actorContextR6KnowledgeDisclosure{
			Identity:         item.Identity,
			OwnerApplication: item.OwnerApplication,
			SourceRefs:       append([]string(nil), item.SourceRefs...),
		})
	}

	return actorContextR6RoutingProbe{
		TaskStep:         stepID,
		RoutingMechanism: mechanism,
		Available:        available,
		Selected:         selected,
		Qualified:        qualified,
		Rejected:         rejected,
		Disclosed:        disclosed,
		AuthorityLimits: []string{
			"routing-selection-is-not-semantic-applicability",
			"qualified-knowledge-is-not-execution-authority",
			"routing-mechanism-does-not-own-canonical-meaning",
			"disclosure-is-derived-and-task-scoped",
		},
	}, nil
}

func actorContextR6AvailablePatternCandidates(f fixture) []actorContextR6KnowledgeCandidate {
	byID := map[string]map[string]struct{}{}
	for _, step := range f.Steps {
		for _, patternID := range step.ApplicationPatterns {
			if _, ok := byID[patternID]; !ok {
				byID[patternID] = map[string]struct{}{}
			}
			byID[patternID][step.ID] = struct{}{}
		}
	}

	out := make([]actorContextR6KnowledgeCandidate, 0, len(byID))
	for id, stepSet := range byID {
		refs := make([]string, 0, len(stepSet))
		for stepID := range stepSet {
			refs = append(refs, stepID)
		}
		sort.Strings(refs)
		out = append(out, actorContextR6KnowledgeCandidate{
			Identity:     id,
			SubjectKind:  "application-pattern",
			ReferencedBy: refs,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Identity < out[j].Identity })
	return out
}

func actorContextR6SelectExact(available []actorContextR6KnowledgeCandidate, stepID string) []actorContextR6KnowledgeCandidate {
	var out []actorContextR6KnowledgeCandidate
	for _, candidate := range available {
		if actorContextR0Contains(candidate.ReferencedBy, stepID) {
			out = append(out, candidate)
		}
	}
	return out
}

func actorContextR6SelectBroad(available []actorContextR6KnowledgeCandidate, _ string) []actorContextR6KnowledgeCandidate {
	return append([]actorContextR6KnowledgeCandidate(nil), available...)
}

func actorContextR6QualifySelection(
	world ir.World,
	sourceIdx sourceindex.Index,
	f fixture,
	stepID string,
	selected []actorContextR6KnowledgeCandidate,
) ([]actorContextR6QualifiedKnowledge, []actorContextR6RoutingRejection, error) {
	var target *step
	for i := range f.Steps {
		if f.Steps[i].ID == stepID {
			target = &f.Steps[i]
			break
		}
	}
	if target == nil {
		return nil, nil, fmt.Errorf("R6 task step not found: %s", stepID)
	}

	allowed := map[string]struct{}{}
	for _, patternID := range target.ApplicationPatterns {
		allowed[patternID] = struct{}{}
	}

	var qualified []actorContextR6QualifiedKnowledge
	var rejected []actorContextR6RoutingRejection
	for _, candidate := range selected {
		if candidate.SubjectKind != "application-pattern" {
			rejected = append(rejected, actorContextR6RoutingRejection{Identity: candidate.Identity, Reason: "unsupported-subject-kind"})
			continue
		}
		if _, ok := allowed[candidate.Identity]; !ok {
			rejected = append(rejected, actorContextR6RoutingRejection{Identity: candidate.Identity, Reason: "not-referenced-by-task-step"})
			continue
		}

		app, _, ok := findPatternOwner(world, candidate.Identity)
		if !ok {
			rejected = append(rejected, actorContextR6RoutingRejection{Identity: candidate.Identity, Reason: "pattern-owner-unresolved"})
			continue
		}

		input, err := actorContextR1SourceIndexInputFingerprint(sourceIdx, "application-pattern", candidate.Identity)
		if err != nil {
			rejected = append(rejected, actorContextR6RoutingRejection{Identity: candidate.Identity, Reason: "current-source-unresolved"})
			continue
		}
		qualified = append(qualified, actorContextR6QualifiedKnowledge{
			Identity:                  candidate.Identity,
			SubjectKind:               candidate.SubjectKind,
			OwnerApplication:          string(app.ID),
			RequiredInputFingerprints: actorContextR0NormalizeInputFingerprints([]actorContextR0InputFingerprint{input}),
			SourceRefs:                []string{input.Artifact},
		})
	}

	sort.Slice(qualified, func(i, j int) bool { return qualified[i].Identity < qualified[j].Identity })
	sort.Slice(rejected, func(i, j int) bool {
		if rejected[i].Identity != rejected[j].Identity {
			return rejected[i].Identity < rejected[j].Identity
		}
		return rejected[i].Reason < rejected[j].Reason
	})
	return qualified, rejected, nil
}

func actorContextR6QualifiedIdentities(values []actorContextR6QualifiedKnowledge) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		out = append(out, value.Identity)
	}
	sort.Strings(out)
	return out
}

func actorContextR6DisclosedIdentities(values []actorContextR6KnowledgeDisclosure) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		out = append(out, value.Identity)
	}
	sort.Strings(out)
	return out
}

func actorContextR6AssertNoRoutingAuthorityShortcut(t *testing.T, typ reflect.Type) {
	t.Helper()
	for _, forbidden := range []string{
		"Score",
		"Confidence",
		"Applicable",
		"Authorized",
		"CanExecute",
		"Executable",
		"Winner",
		"AutoApprove",
	} {
		if _, found := typ.FieldByName(forbidden); found {
			t.Fatalf("R6 routing probe introduced forbidden routing/authority shortcut field %q", forbidden)
		}
	}
}
