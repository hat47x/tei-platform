package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	"tei/internal/sourceindex"
	"tei/internal/workspace"
)

const actorContextR6BReviewTask = "R1-customer-edit-review"

type actorContextR6BReviewEvidenceFixture struct {
	Procedure          string `json:"procedure"`
	Step               string `json:"step"`
	ApplicationPattern string `json:"applicationPattern"`
	RequestedAccess    struct {
		Required string `json:"required"`
	} `json:"requestedAccess"`
}

type actorContextR6BHeterogeneousCandidate struct {
	Identity            string   `json:"identity"`
	KnowledgeKind       string   `json:"knowledgeKind"`
	TaskHints           []string `json:"taskHints"`
	Artifact            string   `json:"artifact,omitempty"`
	SupportsSubjectKind string   `json:"supportsSubjectKind,omitempty"`
	SupportsSubjectID   string   `json:"supportsSubjectId,omitempty"`
}

type actorContextR6BHeterogeneousQualified struct {
	Identity                  string                           `json:"identity"`
	KnowledgeKind             string                           `json:"knowledgeKind"`
	RequiredInputFingerprints []actorContextR0InputFingerprint `json:"requiredInputFingerprints"`
	SourceRefs                []string                         `json:"sourceRefs"`
}

type actorContextR6BHeterogeneousRejection struct {
	Identity      string `json:"identity"`
	KnowledgeKind string `json:"knowledgeKind"`
	Reason        string `json:"reason"`
}

type actorContextR6BHeterogeneousDisclosure struct {
	Identity      string   `json:"identity"`
	KnowledgeKind string   `json:"knowledgeKind"`
	SourceRefs    []string `json:"sourceRefs"`
}

type actorContextR6BHeterogeneousProbe struct {
	Task             string                                   `json:"task"`
	RoutingMechanism string                                   `json:"routingMechanism"`
	Available        []actorContextR6BHeterogeneousCandidate  `json:"available"`
	Selected         []actorContextR6BHeterogeneousCandidate  `json:"selected"`
	Qualified        []actorContextR6BHeterogeneousQualified  `json:"qualified"`
	Rejected         []actorContextR6BHeterogeneousRejection  `json:"rejected,omitempty"`
	Disclosed        []actorContextR6BHeterogeneousDisclosure `json:"disclosed"`
	AuthorityLimits  []string                                 `json:"authorityLimits"`
}

func TestActorContextR6BHeterogeneousRoutingKeepsKnowledgeKindsTaskScoped(t *testing.T) {
	root := repoPath()
	workspacePath := filepath.Join(root, "reference", "workspace")
	procedurePath := filepath.Join(root, "reference", "business-system-design", "order-entry-business-procedure.json")
	reviewEvidencePath := filepath.Join(root, "reference", "business-system-design", "order-entry-customer-edit-conflict.json")

	loaded, err := workspace.Load(workspacePath)
	if err != nil {
		t.Fatal(err)
	}
	f, err := readFixture(procedurePath)
	if err != nil {
		t.Fatal(err)
	}
	sourceIdx, err := workspace.BuildSourceIndex(workspacePath, loaded)
	if err != nil {
		t.Fatal(err)
	}
	reviewEvidence, err := actorContextR6BReadReviewEvidence(reviewEvidencePath)
	if err != nil {
		t.Fatal(err)
	}

	available, err := actorContextR6BBuildAvailableCandidates(f, reviewEvidence)
	if err != nil {
		t.Fatal(err)
	}
	for _, kind := range []string{
		"business-procedure-reference",
		"application-pattern",
		"semantic-validation",
		"semantic-capability",
		"probe-evidence",
	} {
		if !actorContextR6BHasAvailableKind(available, kind) {
			t.Fatalf("R6B candidate pool lost heterogeneous knowledge kind %q: %+v", kind, available)
		}
	}

	tests := []struct {
		task     string
		expected []string
	}{
		{
			task: "BP-02",
			expected: []string{
				"application-pattern|order_entry.customer_reference",
				"business-procedure-reference|order-entry.order-handling",
			},
		},
		{
			task: "BP-05",
			expected: []string{
				"business-procedure-reference|order-entry.order-handling",
				"semantic-validation|sales.order.lines.line_number_unique",
				"semantic-validation|sales.order_line.quantity_positive",
			},
		},
		{
			task: "BP-06",
			expected: []string{
				"business-procedure-reference|order-entry.order-handling",
				"semantic-capability|security.authorize",
			},
		},
		{
			task: actorContextR6BReviewTask,
			expected: []string{
				"application-pattern|order_entry.customer_reference",
				"business-procedure-reference|order-entry.order-handling",
				"probe-evidence|" + actorContextR0ConflictArtifact,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.task, func(t *testing.T) {
			exact, err := deriveActorContextR6BHeterogeneousProbe(
				sourceIdx,
				f,
				reviewEvidence,
				procedurePath,
				reviewEvidencePath,
				tc.task,
				"exact-task-router",
				available,
				actorContextR6BSelectTaskHints,
			)
			if err != nil {
				t.Fatal(err)
			}
			broad, err := deriveActorContextR6BHeterogeneousProbe(
				sourceIdx,
				f,
				reviewEvidence,
				procedurePath,
				reviewEvidencePath,
				tc.task,
				"broad-heterogeneous-router",
				available,
				actorContextR6BSelectAll,
			)
			if err != nil {
				t.Fatal(err)
			}

			exactQualified := actorContextR6BQualifiedKeys(exact.Qualified)
			broadQualified := actorContextR6BQualifiedKeys(broad.Qualified)
			sort.Strings(tc.expected)
			if !reflect.DeepEqual(exactQualified, tc.expected) {
				t.Fatalf("exact router qualified unexpected task knowledge: got=%v want=%v", exactQualified, tc.expected)
			}
			if !reflect.DeepEqual(broadQualified, tc.expected) {
				t.Fatalf("broad router changed deterministic qualification result: got=%v want=%v", broadQualified, tc.expected)
			}
			if len(broad.Rejected) == 0 {
				t.Fatalf("broad router did not expose any over-selection for task %s", tc.task)
			}
			if !reflect.DeepEqual(
				actorContextR6BDisclosureKeys(exact.Disclosed),
				actorContextR6BDisclosureKeys(broad.Disclosed),
			) {
				t.Fatalf("routing mechanism changed disclosed task knowledge: exact=%v broad=%v", exact.Disclosed, broad.Disclosed)
			}
			if len(exact.Disclosed) >= len(exact.Available) {
				t.Fatalf("task %s disclosed the full heterogeneous candidate pool", tc.task)
			}

			for _, item := range exact.Qualified {
				if len(item.RequiredInputFingerprints) == 0 || len(item.SourceRefs) == 0 {
					t.Fatalf("qualified heterogeneous knowledge lost provenance: %+v", item)
				}
			}
		})
	}

	readOnly, err := deriveActorContextR6BHeterogeneousProbe(
		sourceIdx,
		f,
		reviewEvidence,
		procedurePath,
		reviewEvidencePath,
		"BP-02",
		"exact-task-router",
		available,
		actorContextR6BSelectTaskHints,
	)
	if err != nil {
		t.Fatal(err)
	}
	if actorContextR6BHasQualifiedKind(readOnly.Qualified, "probe-evidence") {
		t.Fatal("BP-02 read-only task injected customer-edit conflict Evidence that belongs to the R1 review task")
	}
	if actorContextR6BHasQualifiedKind(readOnly.Qualified, "semantic-capability") {
		t.Fatal("BP-02 read-only task injected BP-06 Authorization Capability knowledge")
	}

	review, err := deriveActorContextR6BHeterogeneousProbe(
		sourceIdx,
		f,
		reviewEvidence,
		procedurePath,
		reviewEvidencePath,
		actorContextR6BReviewTask,
		"exact-task-router",
		available,
		actorContextR6BSelectTaskHints,
	)
	if err != nil {
		t.Fatal(err)
	}
	if !actorContextR6BHasQualifiedKind(review.Qualified, "probe-evidence") {
		t.Fatal("R1 customer-edit review lost its explicit conflict probe Evidence")
	}

	actorContextR6BAssertNoRoutingAuthorityShortcut(t, reflect.TypeOf(actorContextR6BHeterogeneousProbe{}))
}

func actorContextR6BReadReviewEvidence(path string) (actorContextR6BReviewEvidenceFixture, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return actorContextR6BReviewEvidenceFixture{}, err
	}
	var out actorContextR6BReviewEvidenceFixture
	if err := json.Unmarshal(raw, &out); err != nil {
		return actorContextR6BReviewEvidenceFixture{}, err
	}
	return out, nil
}

func actorContextR6BBuildAvailableCandidates(
	f fixture,
	review actorContextR6BReviewEvidenceFixture,
) ([]actorContextR6BHeterogeneousCandidate, error) {
	type key struct {
		kind string
		id   string
	}
	candidates := map[key]actorContextR6BHeterogeneousCandidate{}

	add := func(kind, id, task string) {
		k := key{kind: kind, id: id}
		item := candidates[k]
		item.Identity = id
		item.KnowledgeKind = kind
		item.TaskHints = actorContextR0SortedUnique(append(item.TaskHints, task))
		candidates[k] = item
	}

	for _, step := range f.Steps {
		add("business-procedure-reference", f.Procedure.ID, step.ID)
		for _, id := range step.ApplicationPatterns {
			add("application-pattern", id, step.ID)
		}
		for _, id := range step.Validations {
			add("semantic-validation", id, step.ID)
		}
		for _, id := range step.Capabilities {
			add("semantic-capability", id, step.ID)
		}
	}

	if review.Procedure != f.Procedure.ID {
		return nil, errProcedureMeaning("R6B review Evidence points to a different Procedure")
	}
	add("business-procedure-reference", f.Procedure.ID, actorContextR6BReviewTask)
	add("application-pattern", review.ApplicationPattern, actorContextR6BReviewTask)

	evidenceKey := key{kind: "probe-evidence", id: actorContextR0ConflictArtifact}
	candidates[evidenceKey] = actorContextR6BHeterogeneousCandidate{
		Identity:            actorContextR0ConflictArtifact,
		KnowledgeKind:       "probe-evidence",
		TaskHints:           []string{actorContextR6BReviewTask},
		Artifact:            actorContextR0ConflictArtifact,
		SupportsSubjectKind: "application-pattern",
		SupportsSubjectID:   review.ApplicationPattern,
	}

	out := make([]actorContextR6BHeterogeneousCandidate, 0, len(candidates))
	for _, item := range candidates {
		out = append(out, item)
	}
	sort.Slice(out, func(i, j int) bool {
		return actorContextR6BCandidateKey(out[i]) < actorContextR6BCandidateKey(out[j])
	})
	return out, nil
}

func deriveActorContextR6BHeterogeneousProbe(
	sourceIdx sourceindex.Index,
	f fixture,
	review actorContextR6BReviewEvidenceFixture,
	procedurePath,
	reviewEvidencePath,
	task,
	mechanism string,
	available []actorContextR6BHeterogeneousCandidate,
	selector func([]actorContextR6BHeterogeneousCandidate, string) []actorContextR6BHeterogeneousCandidate,
) (actorContextR6BHeterogeneousProbe, error) {
	selected := selector(available, task)
	qualified, rejected, err := actorContextR6BQualify(
		sourceIdx,
		f,
		review,
		procedurePath,
		reviewEvidencePath,
		task,
		selected,
	)
	if err != nil {
		return actorContextR6BHeterogeneousProbe{}, err
	}

	disclosed := make([]actorContextR6BHeterogeneousDisclosure, 0, len(qualified))
	for _, item := range qualified {
		disclosed = append(disclosed, actorContextR6BHeterogeneousDisclosure{
			Identity:      item.Identity,
			KnowledgeKind: item.KnowledgeKind,
			SourceRefs:    append([]string(nil), item.SourceRefs...),
		})
	}

	return actorContextR6BHeterogeneousProbe{
		Task:             task,
		RoutingMechanism: mechanism,
		Available:        append([]actorContextR6BHeterogeneousCandidate(nil), available...),
		Selected:         selected,
		Qualified:        qualified,
		Rejected:         rejected,
		Disclosed:        disclosed,
		AuthorityLimits: []string{
			"heterogeneous-routing-does-not-flatten-authority-kinds",
			"selected-knowledge-does-not-equal-semantic-applicability",
			"qualified-knowledge-does-not-equal-execution-authority",
			"review-evidence-is-not-operational-authority",
			"procedure-reference-remains-non-canonical",
		},
	}, nil
}

func actorContextR6BSelectTaskHints(
	available []actorContextR6BHeterogeneousCandidate,
	task string,
) []actorContextR6BHeterogeneousCandidate {
	var out []actorContextR6BHeterogeneousCandidate
	for _, candidate := range available {
		if actorContextR0Contains(candidate.TaskHints, task) {
			out = append(out, candidate)
		}
	}
	return out
}

func actorContextR6BSelectAll(
	available []actorContextR6BHeterogeneousCandidate,
	_ string,
) []actorContextR6BHeterogeneousCandidate {
	return append([]actorContextR6BHeterogeneousCandidate(nil), available...)
}

func actorContextR6BQualify(
	sourceIdx sourceindex.Index,
	f fixture,
	review actorContextR6BReviewEvidenceFixture,
	procedurePath,
	reviewEvidencePath,
	task string,
	selected []actorContextR6BHeterogeneousCandidate,
) ([]actorContextR6BHeterogeneousQualified, []actorContextR6BHeterogeneousRejection, error) {
	allowed, err := actorContextR6BAllowedKeys(f, review, task)
	if err != nil {
		return nil, nil, err
	}

	var qualified []actorContextR6BHeterogeneousQualified
	var rejected []actorContextR6BHeterogeneousRejection
	for _, candidate := range selected {
		key := actorContextR6BCandidateKey(candidate)
		if _, ok := allowed[key]; !ok {
			rejected = append(rejected, actorContextR6BHeterogeneousRejection{
				Identity:      candidate.Identity,
				KnowledgeKind: candidate.KnowledgeKind,
				Reason:        "not-in-deterministic-task-scope",
			})
			continue
		}

		var inputs []actorContextR0InputFingerprint
		switch candidate.KnowledgeKind {
		case "business-procedure-reference":
			input, err := actorContextR0FileInputFingerprint(
				"business-procedure-reference",
				actorContextR0ProcedureArtifact,
				procedurePath,
			)
			if err != nil {
				return nil, nil, err
			}
			inputs = []actorContextR0InputFingerprint{input}
		case "application-pattern", "semantic-validation", "semantic-capability":
			input, err := actorContextR1SourceIndexInputFingerprint(
				sourceIdx,
				candidate.KnowledgeKind,
				candidate.Identity,
			)
			if err != nil {
				rejected = append(rejected, actorContextR6BHeterogeneousRejection{
					Identity:      candidate.Identity,
					KnowledgeKind: candidate.KnowledgeKind,
					Reason:        "current-source-unresolved",
				})
				continue
			}
			inputs = []actorContextR0InputFingerprint{input}
		case "probe-evidence":
			if task != actorContextR6BReviewTask ||
				candidate.Artifact != actorContextR0ConflictArtifact ||
				candidate.SupportsSubjectKind != "application-pattern" ||
				candidate.SupportsSubjectID != review.ApplicationPattern {
				rejected = append(rejected, actorContextR6BHeterogeneousRejection{
					Identity:      candidate.Identity,
					KnowledgeKind: candidate.KnowledgeKind,
					Reason:        "evidence-scope-mismatch",
				})
				continue
			}
			input, err := actorContextR0FileInputFingerprint(
				"review-evidence",
				actorContextR0ConflictArtifact,
				reviewEvidencePath,
			)
			if err != nil {
				return nil, nil, err
			}
			inputs = []actorContextR0InputFingerprint{input}
		default:
			rejected = append(rejected, actorContextR6BHeterogeneousRejection{
				Identity:      candidate.Identity,
				KnowledgeKind: candidate.KnowledgeKind,
				Reason:        "unsupported-knowledge-kind",
			})
			continue
		}

		inputs = actorContextR0NormalizeInputFingerprints(inputs)
		qualified = append(qualified, actorContextR6BHeterogeneousQualified{
			Identity:                  candidate.Identity,
			KnowledgeKind:             candidate.KnowledgeKind,
			RequiredInputFingerprints: inputs,
			SourceRefs:                actorContextR3InputArtifacts(inputs),
		})
	}

	sort.Slice(qualified, func(i, j int) bool {
		return actorContextR6BQualifiedKey(qualified[i]) < actorContextR6BQualifiedKey(qualified[j])
	})
	sort.Slice(rejected, func(i, j int) bool {
		left := rejected[i].KnowledgeKind + "|" + rejected[i].Identity + "|" + rejected[i].Reason
		right := rejected[j].KnowledgeKind + "|" + rejected[j].Identity + "|" + rejected[j].Reason
		return left < right
	})
	return qualified, rejected, nil
}

func actorContextR6BAllowedKeys(
	f fixture,
	review actorContextR6BReviewEvidenceFixture,
	task string,
) (map[string]struct{}, error) {
	allowed := map[string]struct{}{
		"business-procedure-reference|" + f.Procedure.ID: {},
	}

	if task == actorContextR6BReviewTask {
		if review.Procedure != f.Procedure.ID || review.Step != "BP-02" || review.RequestedAccess.Required != "read-write" {
			return nil, errProcedureMeaning("R6B review Evidence no longer matches the BP-02 customer-edit challenge")
		}
		allowed["application-pattern|"+review.ApplicationPattern] = struct{}{}
		allowed["probe-evidence|"+actorContextR0ConflictArtifact] = struct{}{}
		return allowed, nil
	}

	var target *step
	for i := range f.Steps {
		if f.Steps[i].ID == task {
			target = &f.Steps[i]
			break
		}
	}
	if target == nil {
		return nil, errProcedureMeaning("R6B task step not found: " + task)
	}

	for _, id := range target.ApplicationPatterns {
		allowed["application-pattern|"+id] = struct{}{}
	}
	for _, id := range target.Validations {
		allowed["semantic-validation|"+id] = struct{}{}
	}
	for _, id := range target.Capabilities {
		allowed["semantic-capability|"+id] = struct{}{}
	}
	return allowed, nil
}

func actorContextR6BCandidateKey(value actorContextR6BHeterogeneousCandidate) string {
	return value.KnowledgeKind + "|" + value.Identity
}

func actorContextR6BQualifiedKey(value actorContextR6BHeterogeneousQualified) string {
	return value.KnowledgeKind + "|" + value.Identity
}

func actorContextR6BQualifiedKeys(values []actorContextR6BHeterogeneousQualified) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		out = append(out, actorContextR6BQualifiedKey(value))
	}
	sort.Strings(out)
	return out
}

func actorContextR6BDisclosureKeys(values []actorContextR6BHeterogeneousDisclosure) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		out = append(out, value.KnowledgeKind+"|"+value.Identity)
	}
	sort.Strings(out)
	return out
}

func actorContextR6BHasAvailableKind(values []actorContextR6BHeterogeneousCandidate, kind string) bool {
	for _, value := range values {
		if value.KnowledgeKind == kind {
			return true
		}
	}
	return false
}

func actorContextR6BHasQualifiedKind(values []actorContextR6BHeterogeneousQualified, kind string) bool {
	for _, value := range values {
		if value.KnowledgeKind == kind {
			return true
		}
	}
	return false
}

func actorContextR6BAssertNoRoutingAuthorityShortcut(t *testing.T, typ reflect.Type) {
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
		"Authority",
	} {
		if _, found := typ.FieldByName(forbidden); found {
			t.Fatalf("R6B heterogeneous routing probe introduced forbidden shortcut field %q", forbidden)
		}
	}
}
