package operations

import (
	"crypto/sha256"
	"fmt"
	"path/filepath"
	"reflect"
	"sort"
	"testing"

	"tei/internal/application"
	"tei/internal/core"
	"tei/internal/driver"
	"tei/internal/driver/queryresult"
	"tei/internal/environment"
	"tei/internal/functioncontext"
	"tei/internal/query"
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
	persons   []string
	paths     []string
	claim     string
	notUnique bool
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
		name               string
		roles              []orgPositiveRole
		duties             []orgPositiveDuty
		roleRead, dutyRead orgPositiveRead
		want               orgPositiveEvidence // Hand-authored literals, not copied from the probe.
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

// This uses the actual Product Function Context projector, but the query result
// is deliberately synthesized in process. It does not perform a Provider read.
func orgPositiveSyntheticNativeContext(t *testing.T, values [][]any) functioncontext.Context {
	return orgPositiveSyntheticContext(t, values, "organization.role_assignment", []string{
		"role_assignment_id", "person_id", "role_id",
	})
}

// Both Source-backed Data shapes use the real Function Context projection.
// Only the input BrowseQueryEnvironmentExecution is synthetic; no DB is read.
func orgPositiveSyntheticResponsibilityContext(t *testing.T, values [][]any) functioncontext.Context {
	return orgPositiveSyntheticContext(t, values, "organization.responsibility", []string{
		"responsibility_id", "subject_person_id", "object_org_unit_id",
	})
}

func orgPositiveSyntheticContext(t *testing.T, values [][]any, dataID string, columns []string) functioncontext.Context {
	t.Helper()
	if len(columns) != 3 {
		t.Fatal("this synthetic fixture expects three columns")
	}
	cols := []queryresult.Column{
		{Raw: columns[0], Attribute: core.CanonicalID(dataID + "." + columns[0])},
		{Raw: columns[1], Attribute: core.CanonicalID(dataID + "." + columns[1])},
		{Raw: columns[2], Attribute: core.CanonicalID(dataID + "." + columns[2])},
	}
	// These are distinct mock Query sources, not evidence of a shared read
	// snapshot. Their digests are local test inputs, not accepted Source pins.
	syntheticQueryID := core.CanonicalID("org_axes_view.synthetic_" + dataID)
	sourceDigest := fmt.Sprintf("%x", sha256.Sum256([]byte("synthetic-query-source:"+dataID)))
	execution := driver.BrowseQueryEnvironmentExecution{
		Environment: "development",
		Instance: environment.DatabaseInstance{
			ID: "org-synthetic.dev", Definition: "org_store", Provider: environment.ProviderPostgres,
		},
		QuerySourceSHA256: sourceDigest,
		Execution: driver.BrowseQueryExecution{
			Resolution: driver.PatternResolution{
				Application: application.Application{ID: "org_axes_view"},
				Pattern:     application.Pattern{ID: syntheticQueryID},
				Query: &query.EmbeddedSQL{
					ID: syntheticQueryID, Dialect: environment.ProviderPostgres,
				},
				Datasource: &application.DatabaseRequirement{
					Alias: "org_read", TargetDatabase: "org_store",
				},
				Data: &semantic.Data{ID: core.CanonicalID(dataID)},
			},
			Result: driver.BrowseQueryResult{
				Raw: queryresult.RawResult{
					Columns: columns, Rows: values,
				},
				Mapped: queryresult.MappedResult{
					Columns: cols, Rows: values,
				},
			},
		},
	}
	ctx, err := functioncontext.From(functioncontext.Input{Execution: execution})
	if err != nil {
		t.Fatal(err)
	}
	return ctx
}

// This is only a *test consumer adapter*: it reads the Context's actual
// mapped row values and verifies the selected columns/identities. It is not
// TEI's accepted Organization Rule, its Person resolver, or a real Query.
func orgPositiveRolesFromNativeContext(ctx functioncontext.Context) ([]orgPositiveRole, error) {
	if string(ctx.Function.Data) != "organization.role_assignment" || len(ctx.Columns) != 3 {
		return nil, fmt.Errorf("unbound RoleAssignment rowset")
	}
	for i, want := range []string{
		"organization.role_assignment.role_assignment_id",
		"organization.role_assignment.person_id",
		"organization.role_assignment.role_id",
	} {
		if string(ctx.Columns[i].Attribute) != want {
			return nil, fmt.Errorf("unbound RoleAssignment column %d", i)
		}
	}
	out := make([]orgPositiveRole, 0, len(ctx.Rows))
	for _, row := range ctx.Rows {
		if len(row) != 3 {
			return nil, fmt.Errorf("incomplete RoleAssignment row")
		}
		fields := [3]string{}
		for i := range fields {
			value, ok := row[i].Value.(string)
			if row[i].State != functioncontext.StateValue || !ok || value == "" {
				return nil, fmt.Errorf("RoleAssignment row has missing or null identity at column %d", i)
			}
			fields[i] = value
		}
		out = append(out, orgPositiveRole{id: fields[0], person: fields[1], role: fields[2]})
	}
	return out, nil
}

// This test-only consumer takes Responsibility values from a *second*
// Function Context. Same-name labels and a cursor EOF are not authority to
// join; the synthetic cohort assertion remains separate from these values.
func orgPositiveDutiesFromNativeContext(ctx functioncontext.Context) ([]orgPositiveDuty, error) {
	if string(ctx.Function.Data) != "organization.responsibility" || len(ctx.Columns) != 3 {
		return nil, fmt.Errorf("unbound Responsibility rowset")
	}
	for i, want := range []string{
		"organization.responsibility.responsibility_id",
		"organization.responsibility.subject_person_id",
		"organization.responsibility.object_org_unit_id",
	} {
		if string(ctx.Columns[i].Attribute) != want {
			return nil, fmt.Errorf("unbound Responsibility column %d", i)
		}
	}
	out := make([]orgPositiveDuty, 0, len(ctx.Rows))
	for _, row := range ctx.Rows {
		if len(row) != 3 {
			return nil, fmt.Errorf("incomplete Responsibility row")
		}
		fields := [3]string{}
		for i := range fields {
			v, ok := row[i].Value.(string)
			if row[i].State != functioncontext.StateValue || !ok || v == "" {
				return nil, fmt.Errorf("Responsibility has missing or null identity at column %d", i)
			}
			fields[i] = v
		}
		out = append(out, orgPositiveDuty{id: fields[0], person: fields[1], unit: fields[2]})
	}
	return out, nil
}

// The native Context from a completed cursor cannot independently establish
// that the RoleAssignment and Responsibility universes are complete. A Product
// Consumer must not copy the fixture's closed-world flag into its own claim.
func TestOrgStagedPositiveCandidateActualFunctionContextCoverage(t *testing.T) {
	// Role and Responsibility are separate Product Context projections.
	// Their observed values are read by the test Consumer, not passed as
	// synthetic role/duty structs straight into the candidate probe.
	for _, tc := range []struct {
		name              string
		roleRows          [][]any
		dutyRows          [][]any
		expectedClaim     string
		expectedNotUnique bool
	}{
		{
			name:          "zero native rows",
			roleRows:      [][]any{},
			dutyRows:      [][]any{},
			expectedClaim: "partial-none",
		},
		{
			name:          "one native row each",
			roleRows:      [][]any{{"RA1", "P1", "R1"}},
			dutyRows:      [][]any{{"RS1", "P1", "D1"}},
			expectedClaim: "partial-one",
		},
		{
			name:          "two native rows each",
			roleRows:      [][]any{{"RA1", "P1", "R1"}, {"RA2", "P2", "R1"}},
			dutyRows:      [][]any{{"RS1", "P1", "D1"}, {"RS2", "P2", "D1"}},
			expectedClaim: "partial-many", expectedNotUnique: true,
		},
		{
			name:          "one role, no responsibility",
			roleRows:      [][]any{{"RA1", "P1", "R1"}},
			dutyRows:      [][]any{},
			expectedClaim: "partial-none",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			roleContext := orgPositiveSyntheticNativeContext(t, tc.roleRows)
			dutyContext := orgPositiveSyntheticResponsibilityContext(t, tc.dutyRows)
			if roleContext.Source.QuerySourceSHA256 == dutyContext.Source.QuerySourceSHA256 ||
				roleContext.Function.Query == dutyContext.Function.Query {
				t.Fatal("independent synthetic Query sources were wrongly conflated")
			}
			for _, read := range []struct {
				ctx functioncontext.Context
				n   int
			}{
				{ctx: roleContext, n: len(tc.roleRows)},
				{ctx: dutyContext, n: len(tc.dutyRows)},
			} {
				if read.ctx.Rowset.RowCount != read.n || !read.ctx.Rowset.FullyRead {
					t.Fatalf("actual Function Context rowset: %+v", read.ctx.Rowset)
				}
				if read.ctx.Coverage.Status != semantic.CoverageUnestablished ||
					read.ctx.Coverage.EmptyJustified {
					t.Fatalf("cursor EOF incorrectly established required cohort: %+v", read.ctx.Coverage)
				}
				if read.ctx.Binding.Assertion != functioncontext.BindingHostAsserted ||
					read.ctx.Source.SourceRevision != functioncontext.SourceUnknown {
					t.Fatalf("native Context claimed unverified Source or connection: %+v %+v",
						read.ctx.Binding, read.ctx.Source)
				}
			}
			// These two Consumers preserve the distinct Context outputs.
			observedRoles, err := orgPositiveRolesFromNativeContext(roleContext)
			if err != nil {
				t.Fatal(err)
			}
			observedDuties, err := orgPositiveDutiesFromNativeContext(dutyContext)
			if err != nil {
				t.Fatal(err)
			}
			// This cohort identity is test-only; the Product Context has
			// not verified a same-snapshot read across the two Queries.
			roleRead := orgPositiveRead{
				cohort:          "test-fixture-cohort",
				closed:          roleContext.Coverage.Status == semantic.CoverageComplete,
				cursorFullyRead: roleContext.Rowset.FullyRead,
			}
			dutyRead := orgPositiveRead{
				cohort:          "test-fixture-cohort",
				closed:          dutyContext.Coverage.Status == semantic.CoverageComplete,
				cursorFullyRead: dutyContext.Rowset.FullyRead,
			}
			got := orgPositiveWitnessProbe(observedRoles, observedDuties, roleRead, dutyRead)
			if got.claim != tc.expectedClaim || got.notUnique != tc.expectedNotUnique {
				t.Fatalf("native Context overqualified observed candidate: %+v", got)
			}
		})
	}
}

// A database-mapped NULL is not proof of an absent Person or an empty Role.
func TestOrgStagedPositiveCandidateNativeNullIdentityIsUnresolved(t *testing.T) {
	ctx := orgPositiveSyntheticNativeContext(t, [][]any{{"RA1", nil, "R1"}})
	if ctx.Rows[0][1].State != functioncontext.StateNull {
		t.Fatal("NULL lost in native Function Context projection")
	}
	if _, err := orgPositiveRolesFromNativeContext(ctx); err == nil {
		t.Fatal("a NULL Person ID was silently joined or converted to no candidate")
	}
	dutyCtx := orgPositiveSyntheticResponsibilityContext(t, [][]any{{"RS1", nil, "D1"}})
	if dutyCtx.Rows[0][1].State != functioncontext.StateNull {
		t.Fatal("Responsibility NULL lost in native Function Context")
	}
	if _, err := orgPositiveDutiesFromNativeContext(dutyCtx); err == nil {
		t.Fatal("a NULL Responsibility subject Person ID was treated as no candidate")
	}
}
