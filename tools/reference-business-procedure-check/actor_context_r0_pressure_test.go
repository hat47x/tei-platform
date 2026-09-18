package main

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	teiindex "tei/internal/index"
	"tei/internal/ir"
	"tei/internal/sourceindex"
	"tei/internal/workspace"
)

const (
	actorContextR0ProcedureArtifact = "reference/business-system-design/order-entry-business-procedure.json"
	actorContextR0ConflictArtifact  = "reference/business-system-design/order-entry-customer-edit-conflict.json"
)

type actorContextR0InputFingerprint struct {
	Scope       string `json:"scope"`
	Artifact    string `json:"artifact"`
	Fingerprint string `json:"fingerprint"`
}

type actorContextR0BuilderProbe struct {
	Purpose                   string                           `json:"purpose"`
	ReferenceFingerprint      string                           `json:"referenceFingerprint"`
	RequiredInputFingerprints []actorContextR0InputFingerprint `json:"requiredInputFingerprints"`
	SourceArtifact            string                           `json:"sourceArtifact"`
	Subjects            []string `json:"subjects,omitempty"`
	Dependencies        []string `json:"dependencies,omitempty"`
	EvidencePaths       []string `json:"evidencePaths,omitempty"`
	AuthorityLimits     []string `json:"authorityLimits"`
}

type actorContextR0HumanReviewProbe struct {
	Purpose                   string                           `json:"purpose"`
	ReferenceFingerprint      string                           `json:"referenceFingerprint"`
	RequiredInputFingerprints []actorContextR0InputFingerprint `json:"requiredInputFingerprints"`
	DiagnosticCodes     []string `json:"diagnosticCodes,omitempty"`
	ImpactSubjects      []string `json:"impactSubjects,omitempty"`
	EvidencePaths       []string `json:"evidencePaths,omitempty"`
	DrillDownRefs       []string `json:"drillDownRefs,omitempty"`
	AuthorityLimits     []string `json:"authorityLimits"`
}

type actorContextR0OperationalProbe struct {
	Purpose                   string                           `json:"purpose"`
	ReferenceFingerprint      string                           `json:"referenceFingerprint"`
	RequiredInputFingerprints []actorContextR0InputFingerprint `json:"requiredInputFingerprints"`
	Procedure           string   `json:"procedure"`
	Step                string   `json:"step"`
	BusinessAction      string   `json:"businessAction"`
	Realization         string   `json:"realization"`
	PatternRefs         []string `json:"patternRefs,omitempty"`
	ValidationRefs      []string `json:"validationRefs,omitempty"`
	CapabilityRefs      []string `json:"capabilityRefs,omitempty"`
	DataRefs            []string `json:"dataRefs,omitempty"`
	RelationRefs        []string `json:"relationRefs,omitempty"`
	Datasource          string   `json:"datasource,omitempty"`
	AcceptedAccess      string   `json:"acceptedAccess,omitempty"`
	EvidencePaths       []string `json:"evidencePaths,omitempty"`
	Unresolved          []string `json:"unresolved,omitempty"`
	AuthorityLimits     []string `json:"authorityLimits"`
}

func TestActorContextR0DerivesPurposeSpecificProjectionsFromSameReference(t *testing.T) {
	root := repoPath()
	workspacePath := filepath.Join(root, "reference", "workspace")
	fixturePath := filepath.Join(root, "reference", "business-system-design", "order-entry-business-procedure.json")
	conflictPath := filepath.Join(root, "reference", "business-system-design", "order-entry-customer-edit-conflict.json")

	loaded, err := workspace.Load(workspacePath)
	if err != nil {
		t.Fatal(err)
	}
	f, err := readFixture(fixturePath)
	if err != nil {
		t.Fatal(err)
	}
	if f.Canonical {
		t.Fatal("R0 must not promote the Business Procedure Reference to Canonical authority")
	}

	referenceFingerprint, err := actorContextR0Fingerprint(fixturePath)
	if err != nil {
		t.Fatal(err)
	}
	sourceIdx, err := workspace.BuildSourceIndex(workspacePath, loaded)
	if err != nil {
		t.Fatal(err)
	}

	builder, err := deriveActorContextR0Builder(loaded.World, sourceIdx, f, "BP-02", fixturePath, referenceFingerprint)
	if err != nil {
		t.Fatal(err)
	}
	human, err := deriveActorContextR0HumanReview(loaded.World, sourceIdx, f, workspacePath, fixturePath, conflictPath, referenceFingerprint)
	if err != nil {
		t.Fatal(err)
	}
	operational, err := deriveActorContextR0Operational(loaded.World, sourceIdx, f, "BP-02", fixturePath, referenceFingerprint)
	if err != nil {
		t.Fatal(err)
	}

	if builder.ReferenceFingerprint != human.ReferenceFingerprint || builder.ReferenceFingerprint != operational.ReferenceFingerprint {
		t.Fatalf("three R0 projections did not preserve the same Business Reference baseline: builder=%s human=%s operational=%s",
			builder.ReferenceFingerprint, human.ReferenceFingerprint, operational.ReferenceFingerprint)
	}
	for name, inputs := range map[string][]actorContextR0InputFingerprint{
		"builder":     builder.RequiredInputFingerprints,
		"human":       human.RequiredInputFingerprints,
		"operational": operational.RequiredInputFingerprints,
	} {
		if !actorContextR0HasInputScope(inputs, "business-procedure-reference") {
			t.Fatalf("%s projection lost the shared Business Procedure input fingerprint: %+v", name, inputs)
		}
		if !actorContextR0HasInputScope(inputs, "source-definition") {
			t.Fatalf("%s projection did not bind freshness to selected workspace source definitions: %+v", name, inputs)
		}
	}
	if !actorContextR0HasInputScope(human.RequiredInputFingerprints, "review-evidence") {
		t.Fatalf("Human Review projection did not bind freshness to the conflict Evidence input: %+v", human.RequiredInputFingerprints)
	}

	if operational.AcceptedAccess != "read" {
		t.Fatalf("BP-02 operational projection changed accepted datasource access: %+v", operational)
	}
	if !actorContextR0Contains(human.DiagnosticCodes, "BUSINESS_FUNCTION_DATASOURCE_ACCESS_CONFLICT") {
		t.Fatalf("Human Review projection lost the intentional read/write conflict: %+v", human)
	}
	if !actorContextR0Contains(operational.RelationRefs, "sales.order.customer") {
		t.Fatalf("Operational projection lost the exact BP-02 relation: %+v", operational)
	}
	if !actorContextR0Contains(builder.Subjects, "order_entry.customer_reference") ||
		!actorContextR0Contains(builder.Subjects, "sales.order.customer") {
		t.Fatalf("Builder context lost the change-local Pattern / Relation subjects: %+v", builder.Subjects)
	}
	if len(builder.Dependencies) == 0 {
		t.Fatalf("Builder context did not derive any exact dependency from the loaded World: %+v", builder)
	}

	actorContextR0AssertNoAuthorityShortcutFields(t, reflect.TypeOf(actorContextR0BuilderProbe{}))
	actorContextR0AssertNoAuthorityShortcutFields(t, reflect.TypeOf(actorContextR0HumanReviewProbe{}))
	actorContextR0AssertNoAuthorityShortcutFields(t, reflect.TypeOf(actorContextR0OperationalProbe{}))
}

func TestActorContextR0OperationalKeepsManualAndObservationStepsUnresolved(t *testing.T) {
	root := repoPath()
	loaded, err := workspace.Load(filepath.Join(root, "reference", "workspace"))
	if err != nil {
		t.Fatal(err)
	}
	fixturePath := filepath.Join(root, "reference", "business-system-design", "order-entry-business-procedure.json")
	f, err := readFixture(fixturePath)
	if err != nil {
		t.Fatal(err)
	}
	referenceFingerprint, err := actorContextR0Fingerprint(fixturePath)
	if err != nil {
		t.Fatal(err)
	}
	sourceIdx, err := workspace.BuildSourceIndex(filepath.Join(root, "reference", "workspace"), loaded)
	if err != nil {
		t.Fatal(err)
	}

	manual, err := deriveActorContextR0Operational(loaded.World, sourceIdx, f, "BP-04", fixturePath, referenceFingerprint)
	if err != nil {
		t.Fatal(err)
	}
	if !actorContextR0Contains(manual.Unresolved, "manual-or-external-realization") {
		t.Fatalf("BP-04 manual judgement was silently promoted to machine execution: %+v", manual)
	}
	if !actorContextR0Contains(manual.Unresolved, "no-data-trace-from-current-refs") {
		t.Fatalf("BP-04 missing current Data trace was not preserved as unresolved: %+v", manual)
	}

	observation, err := deriveActorContextR0Operational(loaded.World, sourceIdx, f, "BP-08", fixturePath, referenceFingerprint)
	if err != nil {
		t.Fatal(err)
	}
	if !actorContextR0Contains(observation.Unresolved, "observation-realization-not-semantic-capability") {
		t.Fatalf("BP-08 runtime Observation boundary was silently promoted to Semantic Capability: %+v", observation)
	}
	if len(observation.CapabilityRefs) != 0 {
		t.Fatalf("BP-08 gained an unaccepted Semantic Capability: %+v", observation.CapabilityRefs)
	}
}

func deriveActorContextR0Builder(world ir.World, sourceIdx sourceindex.Index, f fixture, stepID, fixturePath, referenceFingerprint string) (actorContextR0BuilderProbe, error) {
	operational, err := deriveActorContextR0Operational(world, sourceIdx, f, stepID, fixturePath, referenceFingerprint)
	if err != nil {
		return actorContextR0BuilderProbe{}, err
	}

	subjects := actorContextR0SortedUnique(append(
		append(
			append(
				append([]string(nil), operational.PatternRefs...),
				operational.DataRefs...,
			),
			operational.RelationRefs...,
		),
		append(operational.ValidationRefs, operational.CapabilityRefs...)...,
	))

	subjectSet := make(map[string]struct{}, len(subjects))
	for _, subject := range subjects {
		subjectSet[subject] = struct{}{}
	}

	derived := teiindex.Build(world)
	var dependencies []string
	for _, dep := range derived.SemanticDependencies {
		from, to := string(dep.From), string(dep.To)
		if _, ok := subjectSet[from]; !ok {
			if _, ok := subjectSet[to]; !ok {
				continue
			}
		}
		dependencies = append(dependencies, fmt.Sprintf("%s:%s --%s--> %s:%s", dep.FromKind, from, dep.Relation, dep.ToKind, to))
	}
	for _, dep := range derived.DeclaredDependencies {
		from, to := string(dep.From), string(dep.To)
		if _, ok := subjectSet[from]; !ok {
			if _, ok := subjectSet[to]; !ok {
				continue
			}
		}
		dependencies = append(dependencies, fmt.Sprintf("%s:%s --%s--> %s:%s", dep.FromKind, from, dep.Relation, dep.ToKind, to))
	}
	sort.Strings(dependencies)
	inputs := append([]actorContextR0InputFingerprint(nil), operational.RequiredInputFingerprints...)
	inputs = append(inputs, actorContextR0InputFingerprint{
		Scope:       "derived-dependency-set",
		Artifact:    "internal/index.Build:selected-dependencies",
		Fingerprint: sourceindex.Fingerprint([]byte(strings.Join(dependencies, "\n"))),
	})
	inputs = actorContextR0NormalizeInputFingerprints(inputs)

	return actorContextR0BuilderProbe{
		Purpose:                   "re-enter development around " + stepID,
		ReferenceFingerprint:      referenceFingerprint,
		RequiredInputFingerprints: inputs,
		SourceArtifact:            fixturePath,
		Subjects:            subjects,
		Dependencies:        dependencies,
		EvidencePaths:       append([]string(nil), operational.EvidencePaths...),
		AuthorityLimits: []string{
			"builder-context-is-derived",
			"business-procedure-reference-not-canonical",
			"proposal-must-pass-shared-acceptance-before-authoritative-change",
		},
	}, nil
}

func deriveActorContextR0HumanReview(
	world ir.World,
	sourceIdx sourceindex.Index,
	f fixture,
	workspacePath,
	fixturePath,
	conflictPath,
	referenceFingerprint string,
) (actorContextR0HumanReviewProbe, error) {
	result, err := run(workspacePath, fixturePath, conflictPath)
	if err != nil {
		return actorContextR0HumanReviewProbe{}, err
	}
	operational, err := deriveActorContextR0Operational(world, sourceIdx, f, "BP-02", fixturePath, referenceFingerprint)
	if err != nil {
		return actorContextR0HumanReviewProbe{}, err
	}

	var codes []string
	for _, item := range result.Diagnostics {
		codes = append(codes, item.Code)
	}
	sort.Strings(codes)

	impact := actorContextR0SortedUnique(append(
		append(append([]string(nil), operational.PatternRefs...), operational.DataRefs...),
		operational.RelationRefs...,
	))
	evidence := append([]string(nil), operational.EvidencePaths...)
	if operational.Datasource != "" && operational.AcceptedAccess != "" {
		evidence = append(evidence, "accepted-datasource-access:"+operational.Datasource+":"+operational.AcceptedAccess)
	}
	sort.Strings(evidence)

	conflictInput, err := actorContextR0FileInputFingerprint("review-evidence", actorContextR0ConflictArtifact, conflictPath)
	if err != nil {
		return actorContextR0HumanReviewProbe{}, err
	}
	inputs := append([]actorContextR0InputFingerprint(nil), operational.RequiredInputFingerprints...)
	inputs = actorContextR0NormalizeInputFingerprints(append(inputs, conflictInput))

	return actorContextR0HumanReviewProbe{
		Purpose:                   "review intentional customer-edit access conflict",
		ReferenceFingerprint:      referenceFingerprint,
		RequiredInputFingerprints: inputs,
		DiagnosticCodes:     codes,
		ImpactSubjects:      impact,
		EvidencePaths:       evidence,
		DrillDownRefs:       []string{fixturePath, conflictPath},
		AuthorityLimits: []string{
			"human-review-projection-is-derived",
			"diagnostic-does-not-author-business-meaning",
			"review-summary-does-not-grant-execution-authority",
		},
	}, nil
}

func deriveActorContextR0Operational(world ir.World, sourceIdx sourceindex.Index, f fixture, stepID, fixturePath, referenceFingerprint string) (actorContextR0OperationalProbe, error) {
	var selected *step
	for i := range f.Steps {
		if f.Steps[i].ID == stepID {
			selected = &f.Steps[i]
			break
		}
	}
	if selected == nil {
		return actorContextR0OperationalProbe{}, fmt.Errorf("step not found: %s", stepID)
	}

	rows, err := deriveBusinessActionDataProjectionProbe(world, f)
	if err != nil {
		return actorContextR0OperationalProbe{}, err
	}
	dataRow, ok := businessActionDataRowsByStep(rows)[stepID]
	if !ok {
		return actorContextR0OperationalProbe{}, fmt.Errorf("data projection not found for step: %s", stepID)
	}

	out := actorContextR0OperationalProbe{
		Purpose:              "read-only operational context for " + stepID,
		ReferenceFingerprint: referenceFingerprint,
		Procedure:           f.Procedure.ID,
		Step:                stepID,
		BusinessAction:      selected.BusinessAction,
		Realization:         selected.Realization,
		PatternRefs:         append([]string(nil), selected.ApplicationPatterns...),
		ValidationRefs:      append([]string(nil), selected.Validations...),
		CapabilityRefs:      append([]string(nil), selected.Capabilities...),
		DataRefs:            append([]string(nil), dataRow.Data...),
		RelationRefs:        append([]string(nil), dataRow.Relations...),
		EvidencePaths:       append([]string(nil), dataRow.EvidencePaths...),
		AuthorityLimits: []string{
			"operational-projection-is-derived",
			"business-procedure-reference-not-canonical",
			"context-does-not-grant-invocation-authority",
			"command-acknowledgement-would-not-prove-business-completion",
		},
	}

	if len(selected.ApplicationPatterns) == 1 {
		app, pattern, ok := findPatternOwner(world, selected.ApplicationPatterns[0])
		if !ok {
			return actorContextR0OperationalProbe{}, fmt.Errorf("Application Pattern %s did not resolve exactly once", selected.ApplicationPatterns[0])
		}
		if pattern.Datasource != "" {
			req, found := findDatasource(app, string(pattern.Datasource))
			if !found {
				return actorContextR0OperationalProbe{}, fmt.Errorf("datasource %s not found for Application %s", pattern.Datasource, app.ID)
			}
			out.Datasource = string(pattern.Datasource)
			out.AcceptedAccess = string(req.Access)
		}
	}

	if dataRow.NoDataTraceFromCurrentRefs {
		out.Unresolved = append(out.Unresolved, "no-data-trace-from-current-refs")
	}
	if strings.Contains(selected.Realization, "manual") || strings.Contains(selected.Realization, "external") || strings.Contains(selected.Realization, "unresolved") {
		out.Unresolved = append(out.Unresolved, "manual-or-external-realization")
	}
	if stepID == "BP-08" {
		out.Unresolved = append(out.Unresolved, "observation-realization-not-semantic-capability")
	}
	sort.Strings(out.Unresolved)

	sourceInputs, err := actorContextR0SourceInputFingerprints(sourceIdx, out)
	if err != nil {
		return actorContextR0OperationalProbe{}, err
	}
	procedureInput, err := actorContextR0FileInputFingerprint("business-procedure-reference", actorContextR0ProcedureArtifact, fixturePath)
	if err != nil {
		return actorContextR0OperationalProbe{}, err
	}
	out.RequiredInputFingerprints = actorContextR0NormalizeInputFingerprints(append(sourceInputs, procedureInput))
	return out, nil
}

func actorContextR0SourceInputFingerprints(idx sourceindex.Index, probe actorContextR0OperationalProbe) ([]actorContextR0InputFingerprint, error) {
	type subjectGroup struct {
		kind string
		ids  []string
	}
	groups := []subjectGroup{
		{kind: "application-pattern", ids: probe.PatternRefs},
		{kind: "semantic-validation", ids: probe.ValidationRefs},
		{kind: "semantic-capability", ids: probe.CapabilityRefs},
		{kind: "semantic-data", ids: probe.DataRefs},
		{kind: "semantic-relation", ids: probe.RelationRefs},
	}

	var out []actorContextR0InputFingerprint
	for _, group := range groups {
		for _, id := range group.ids {
			refs := idx.ReferencesFor(sourceindex.Subject{Kind: group.kind, ID: id})
			if len(refs) == 0 {
				return nil, fmt.Errorf("R0 selected subject %s:%s has no Source Index reference", group.kind, id)
			}
			for _, ref := range refs {
				artifact, ok := actorContextR0ArtifactByPath(idx, ref.ArtifactPath)
				if !ok {
					return nil, fmt.Errorf("R0 Source Index reference %s:%s points to missing artifact %s", group.kind, id, ref.ArtifactPath)
				}
				out = append(out, actorContextR0InputFingerprint{
					Scope:       "source-definition",
					Artifact:    artifact.Path,
					Fingerprint: artifact.Fingerprint,
				})
			}
		}
	}
	return actorContextR0NormalizeInputFingerprints(out), nil
}

func actorContextR0FileInputFingerprint(scope, artifact, path string) (actorContextR0InputFingerprint, error) {
	fingerprint, err := actorContextR0Fingerprint(path)
	if err != nil {
		return actorContextR0InputFingerprint{}, err
	}
	return actorContextR0InputFingerprint{
		Scope:       scope,
		Artifact:    artifact,
		Fingerprint: fingerprint,
	}, nil
}

func actorContextR0NormalizeInputFingerprints(values []actorContextR0InputFingerprint) []actorContextR0InputFingerprint {
	seen := map[string]struct{}{}
	out := make([]actorContextR0InputFingerprint, 0, len(values))
	for _, value := range values {
		key := value.Scope + "\x00" + value.Artifact + "\x00" + value.Fingerprint
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, value)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Scope != out[j].Scope {
			return out[i].Scope < out[j].Scope
		}
		if out[i].Artifact != out[j].Artifact {
			return out[i].Artifact < out[j].Artifact
		}
		return out[i].Fingerprint < out[j].Fingerprint
	})
	return out
}

func actorContextR0ArtifactByPath(idx sourceindex.Index, path string) (sourceindex.Artifact, bool) {
	for _, artifact := range idx.Artifacts {
		if artifact.Path == path {
			return artifact, true
		}
	}
	return sourceindex.Artifact{}, false
}

func actorContextR0HasInputScope(values []actorContextR0InputFingerprint, scope string) bool {
	for _, value := range values {
		if value.Scope == scope {
			return true
		}
	}
	return false
}

func actorContextR0Fingerprint(path string) (string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return sourceindex.Fingerprint(raw), nil
}

func actorContextR0SortedUnique(values []string) []string {
	set := map[string]struct{}{}
	for _, value := range values {
		if strings.TrimSpace(value) == "" {
			continue
		}
		set[value] = struct{}{}
	}
	out := make([]string, 0, len(set))
	for value := range set {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func actorContextR0Contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func actorContextR0AssertNoAuthorityShortcutFields(t *testing.T, typ reflect.Type) {
	t.Helper()
	for _, forbidden := range []string{
		"CanExecute",
		"Executable",
		"Authorized",
		"Complete",
		"Truth",
		"Confidence",
		"AgentTrust",
		"Winner",
		"Preferred",
		"AutoApprove",
	} {
		if _, found := typ.FieldByName(forbidden); found {
			t.Fatalf("R0 projection %s introduced forbidden authority shortcut field %q", typ.Name(), forbidden)
		}
	}
}
