package operations

import (
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	"tei/internal/semantic"
	"tei/internal/workspace"
)

// All helpers below are test-only. This synthetic positive conjunction is not
// an accepted Case C1 approval Rule, an actual Provider read or a Host Grant.
type orgPositiveRole struct{ id, person, role string }
type orgPositiveDuty struct{ id, person, unit string }
type orgPositiveRead struct {
	cohort          string
	closed          bool // Independently asserted only by this closed-world test input.
	cursorFullyRead bool // Intentionally insufficient to assert closed.
}
type orgPositiveEvidence struct {
	persons       []string
	paths         []string
	claim         string
	notUnique     bool
}

// orgPositiveWitnessProbe computes only observed positive paths. It does not
// perform negative/exception guards, historical resolution or authorization.
func orgPositiveWitnessProbe(roles []orgPositiveRole, duties []orgPositiveDuty, roleRead, dutyRead orgPositiveRead) orgPositiveEvidence {
	out := orgPositiveEvidence{persons: []string{}, paths: []string{}}
	if roleRead.cohort == "" || dutyRead.cohort == "" || roleRead.cohort != dutyRead.cohort {
		out.claim = "unresolved-cohort"
		return out
	}
	seenRole, seenDuty := map[string]bool{}, map[string]bool{}
	for _, r := range roles {
		if r.id == "" || r.person == "" || r.role == "" || seenRole[r.id] {
			out.claim = "unresolved-identity"
			return out
		}
		seenRole[r.id] = true
	}
	for _, d := range duties {
		if d.id == "" || d.person == "" || d.unit == "" || seenDuty[d.id] {
			out.claim = "unresolved-identity"
			return out
		}
		seenDuty[d.id] = true
	}
	people := map[string]bool{}
	for _, r := range roles {
		if r.role != "R1" {
			continue
		}
		for _, d := range duties {
			if d.unit == "D1" && r.person == d.person {
				people[r.person] = true
				out.paths = append(out.paths, r.person+"/"+r.id+"/"+d.id)
			}
		}
	}
	for person := range people {
		out.persons = append(out.persons, person)
	}
	sort.Strings(out.persons)
	sort.Strings(out.paths)
	out.notUnique = len(out.persons) >= 2
	// This fixture's closed is not inferred from cursorFullyRead. Current
	// functioncontext.From preserves CoverageUnestablished after cursor EOF.
	closed := roleRead.closed && dutyRead.closed
	switch {
	case closed && len(out.persons) == 0:
		out.claim = "complete-none"
	case closed && len(out.persons) == 1:
		out.claim = "complete-one"
	case closed:
		out.claim = "complete-many"
	case len(out.persons) == 0:
		out.claim = "partial-none"
	case len(out.persons) == 1:
		out.claim = "partial-one"
	default:
		out.claim = "partial-many"
	}
	return out
}

func orgPositiveHasKey(data semantic.Data, target, sourceAttr, targetAttr string) bool {
	for _, relation := range data.Relations {
		if string(relation.Target) != target || len(relation.Keys) != 1 {
			continue
		}
		key := relation.Keys[0]
		if string(key.SourceAttribute) == sourceAttr && string(key.TargetAttribute) == targetAttr {
			return true
		}
	}
	return false
}

func TestOrgStagedPositiveCandidateRelationKeysFromRealWorkspace(t *testing.T) {
	root := filepath.Join("..", "..", "reference", "org-staged-axis")
	loaded, err := workspace.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	data := map[string]semantic.Data{}
	for _, ns := range loaded.World.Semantics {
		for _, d := range ns.Data {
			data[string(d.ID)] = d
		}
	}
	role, ok := data["organization.role_assignment"]
	if !ok || !orgPositiveHasKey(role, "organization.person", "organization.role_assignment.person_id", "organization.person.person_id") ||
		!orgPositiveHasKey(role, "organization.role", "organization.role_assignment.role_id", "organization.role.role_id") {
		t.Fatalf("actual RoleAssignment source relation/key changed: %+v", role)
	}
	duty, ok := data["organization.responsibility"]
	if !ok || !orgPositiveHasKey(duty, "organization.person", "organization.responsibility.subject_person_id", "organization.person.person_id") ||
		!orgPositiveHasKey(duty, "organization.org_unit", "organization.responsibility.object_org_unit_id", "organization.org_unit.org_unit_id") {
		t.Fatalf("actual Responsibility source relation/key changed: %+v", duty)
	}
	if _, exists := data["organization.case_record"]; exists {
		t.Fatal("the staged axis now has Case data: reconsider this test's no-Case boundary")
	}
}

func TestOrgStagedPositiveCandidatePartialIsNotComplete(t *testing.T) {
	closed := orgPositiveRead{cohort: "synthetic-s1", closed: true, cursorFullyRead: true}
	partial := orgPositiveRead{cohort: "synthetic-s1", closed: false, cursorFullyRead: true}
	r1 := orgPositiveRole{id: "RA1", person: "P1", role: "R1"}
	r2 := orgPositiveRole{id: "RA2", person: "P2", role: "R1"}
	d1 := orgPositiveDuty{id: "RS1", person: "P1", unit: "D1"}
	d2 := orgPositiveDuty{id: "RS2", person: "P2", unit: "D1"}
	cases := []struct {
		name              string
		roles             []orgPositiveRole
		duties            []orgPositiveDuty
		roleRead, dutyRead orgPositiveRead
		want              orgPositiveEvidence // Hand-authored literals, not copied from the probe.
	}{
		{"closed zero", []orgPositiveRole{r1}, []orgPositiveDuty{{id: "RS0", person: "P1", unit: "D2"}}, closed, closed,
			orgPositiveEvidence{persons: []string{}, paths: []string{}, claim: "complete-none"}},
		{"closed one", []orgPositiveRole{r1}, []orgPositiveDuty{d1}, closed, closed,
			orgPositiveEvidence{persons: []string{"P1"}, paths: []string{"P1/RA1/RS1"}, claim: "complete-one"}},
		{"closed many", []orgPositiveRole{r1, r2}, []orgPositiveDuty{d1, d2}, closed, closed,
			orgPositiveEvidence{persons: []string{"P1", "P2"}, paths: []string{"P1/RA1/RS1", "P2/RA2/RS2"}, claim: "complete-many", notUnique: true}},
		{"partial zero, cursors fully read", []orgPositiveRole{}, []orgPositiveDuty{}, partial, partial,
			orgPositiveEvidence{persons: []string{}, paths: []string{}, claim: "partial-none"}},
		{"same visible P1 but partial", []orgPositiveRole{r1}, []orgPositiveDuty{d1}, closed, partial,
			orgPositiveEvidence{persons: []string{"P1"}, paths: []string{"P1/RA1/RS1"}, claim: "partial-one"}},
		{"partial two disproves unique but not exhaustion", []orgPositiveRole{r1, r2}, []orgPositiveDuty{d1, d2}, closed, partial,
			orgPositiveEvidence{persons: []string{"P1", "P2"}, paths: []string{"P1/RA1/RS1", "P2/RA2/RS2"}, claim: "partial-many", notUnique: true}},
		{"two paths one person", []orgPositiveRole{r1, {id: "RA3", person: "P1", role: "R1"}}, []orgPositiveDuty{d1}, closed, closed,
			orgPositiveEvidence{persons: []string{"P1"}, paths: []string{"P1/RA1/RS1", "P1/RA3/RS1"}, claim: "complete-one"}},
		{"different person ID is not same-name join", []orgPositiveRole{r1}, []orgPositiveDuty{{id: "RS3", person: "P1-other", unit: "D1"}}, closed, closed,
			orgPositiveEvidence{persons: []string{}, paths: []string{}, claim: "complete-none"}},
		{"missing person is unresolved", []orgPositiveRole{{id: "RA1", role: "R1"}}, []orgPositiveDuty{d1}, closed, closed,
			orgPositiveEvidence{persons: []string{}, paths: []string{}, claim: "unresolved-identity"}},
		{"different read cohorts are unresolved", []orgPositiveRole{r1}, []orgPositiveDuty{d1}, closed, orgPositiveRead{cohort: "synthetic-s2", closed: true, cursorFullyRead: true},
			orgPositiveEvidence{persons: []string{}, paths: []string{}, claim: "unresolved-cohort"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := orgPositiveWitnessProbe(tc.roles, tc.duties, tc.roleRead, tc.dutyRead)
			if !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("observed: %#v; independent expected: %#v", got, tc.want)
			}
		})
	}
}
