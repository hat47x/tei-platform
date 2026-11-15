package functioncontext_test

import (
	"reflect"
	"testing"

	"tei/internal/core"
	"tei/internal/driver"
	"tei/internal/functioncontext"
	"tei/internal/semantic"
)

var declaredOrders = driver.DeclaredBrowseRelations{
	Data: "sales.order", Database: "sales_core", Status: driver.DeclaredRelationsResolved,
	Tables: []string{"public.orders"}, PrimaryKey: []core.CanonicalID{"sales.order.order_id"},
}

// cleanObservation is the best a native SQL read can show: every observation is favourable.
func cleanObservation() *functioncontext.AcquisitionObservation {
	return &functioncontext.AcquisitionObservation{
		SystemIdentifier: "7707437377464762413",
		LockedTables:     []string{"public.orders"},
		PlanRelations:    []string{"public.orders"},
		PlanNodes:        []string{"Seq Scan", "Sort"},
		Relations:        []functioncontext.RelationObservation{{Name: "public.orders", Found: true, RelKind: "r"}},
	}
}

func assessed(t *testing.T, declared driver.DeclaredBrowseRelations, obs *functioncontext.AcquisitionObservation, rows [][]any) functioncontext.Context {
	t.Helper()
	c, err := functioncontext.From(functioncontext.Input{Execution: execution(rows, nil), Declared: declared, Acquisition: obs})
	if err != nil {
		t.Fatal(err)
	}
	return c
}

// Pinned: every observation can be favourable and the rows can still have been restricted by
// something the plan does not show (a function that inlines into the plan). Coverage must
// stay unestablished, never complete, and the open list must say why.
func TestAllFavourableObservationsStillNeverMakeCoverageComplete(t *testing.T) {
	for name, rows := range map[string][][]any{"zero": {}, "some": {{"O-1", "1.00"}}} {
		c := assessed(t, declaredOrders, cleanObservation(), rows)
		if c.Coverage.Status != semantic.CoverageUnestablished || len(c.Coverage.Reasons) != 0 || c.Coverage.EmptyJustified {
			t.Fatalf("%s: %+v", name, c.Coverage)
		}
		if !contains(c.Coverage.Open, functioncontext.OpenNativePredicate) {
			t.Fatalf("%s: the open native predicate must be listed: %v", name, c.Coverage.Open)
		}
		if c.Coverage.Visibility != functioncontext.VisibilityRLSInactive || c.Grain.Scan != functioncontext.ScanSingle {
			t.Fatalf("%s: visibility/scan %s %s", name, c.Coverage.Visibility, c.Grain.Scan)
		}
	}
}

func TestSeenRestrictionsLowerCoverageToPartialWithTheirReasons(t *testing.T) {
	type mutate func(*functioncontext.AcquisitionObservation)
	cases := map[string]struct {
		change mutate
		reason string
	}{
		"row security": {func(o *functioncontext.AcquisitionObservation) {
			o.Relations[0].RowSecurityActive = true
		}, "row-security-active:public.orders"},
		"limit": {func(o *functioncontext.AcquisitionObservation) {
			o.PlanNodes = append(o.PlanNodes, "Limit")
		}, "limit-in-plan"},
		"function scan": {func(o *functioncontext.AcquisitionObservation) {
			o.PlanNodes = append(o.PlanNodes, "Function Scan")
		}, "function-scan"},
		"foreign scan": {func(o *functioncontext.AcquisitionObservation) {
			o.PlanNodes = append(o.PlanNodes, "Foreign Scan")
		}, "foreign-scan"},
		"partitioned": {func(o *functioncontext.AcquisitionObservation) {
			o.PlanNodes = append(o.PlanNodes, "Append")
		}, "append-or-partitioned"},
		"a view under the declared name": {func(o *functioncontext.AcquisitionObservation) {
			o.PlanRelations = []string{"public.orders_base"}
			o.Relations[0].RelKind = "v"
		}, "relation-kind:public.orders=v"},
		"another relation is read": {func(o *functioncontext.AcquisitionObservation) {
			o.PlanRelations = []string{"public.order_lines", "public.orders"}
		}, "relation-not-declared:public.order_lines"},
		"the declared relation is not read": {func(o *functioncontext.AcquisitionObservation) {
			o.PlanRelations = []string{"public.orders_archive"}
		}, "declared-relation-not-read:public.orders"},
		"no relation at all": {func(o *functioncontext.AcquisitionObservation) {
			o.PlanRelations = nil
		}, "no-relation-in-plan"},
	}
	for name, tc := range cases {
		obs := cleanObservation()
		tc.change(obs)
		c := assessed(t, declaredOrders, obs, [][]any{{"O-1", "1.00"}})
		if c.Coverage.Status != semantic.CoveragePartial || !contains(c.Coverage.Reasons, tc.reason) {
			t.Errorf("%s: status=%s reasons=%v want %s", name, c.Coverage.Status, c.Coverage.Reasons, tc.reason)
		}
		if c.Coverage.EmptyJustified {
			t.Errorf("%s: EmptyJustified raised", name)
		}
	}
	// The visibility word follows what the Provider reported for the session.
	obs := cleanObservation()
	obs.Relations[0].RowSecurityActive = true
	if c := assessed(t, declaredOrders, obs, nil); c.Coverage.Visibility != functioncontext.VisibilityRLSActive {
		t.Errorf("visibility=%s", c.Coverage.Visibility)
	}
}

func TestMissingEvidenceStaysUnestablishedAndNamesWhatIsMissing(t *testing.T) {
	// Not observed at all.
	c := assessed(t, declaredOrders, nil, nil)
	if c.Coverage.Status != semantic.CoverageUnestablished || c.Coverage.Visibility != functioncontext.NotObserved ||
		c.Grain.Scan != functioncontext.ScanUnobserved || !contains(c.Coverage.Open, "acquisition-not-observed") || c.Acquisition.Observed {
		t.Fatalf("%+v %+v", c.Coverage, c.Grain)
	}
	// The plan could not be observed: nothing is inferred from its absence.
	obs := cleanObservation()
	obs.PlanRelations, obs.PlanNodes, obs.PlanNote = nil, nil, "cannot explain the statement"
	obs.LockNote, obs.LockedTables, obs.SystemIdentifier = "cannot lock", nil, ""
	c = assessed(t, declaredOrders, obs, nil)
	if c.Coverage.Status != semantic.CoverageUnestablished || c.Grain.Scan != functioncontext.ScanUnobserved {
		t.Fatalf("%+v %+v", c.Coverage, c.Grain)
	}
	for _, want := range []string{"plan-unobserved", "not-locked-before-observation", "system-identifier-unobserved"} {
		if !contains(c.Coverage.Open, want) {
			t.Errorf("open list lacks %s: %v", want, c.Coverage.Open)
		}
	}
	// No declaration to compare against: relation mismatch cannot be claimed, and is listed open.
	none := driver.DeclaredBrowseRelations{Status: driver.DeclaredRelationsNone}
	obs = cleanObservation()
	obs.PlanRelations = []string{"public.something_else"}
	c = assessed(t, none, obs, nil)
	if c.Coverage.Status != semantic.CoverageUnestablished || !contains(c.Coverage.Open, "declared-relations-none") {
		t.Fatalf("%+v", c.Coverage)
	}
}

func TestJoinedPlanShapeIsReported(t *testing.T) {
	obs := cleanObservation()
	obs.PlanNodes = append(obs.PlanNodes, "Hash Join")
	if c := assessed(t, declaredOrders, obs, nil); c.Grain.Scan != functioncontext.ScanMulti {
		t.Fatalf("scan=%s", c.Grain.Scan)
	}
}

func TestDeclaredKeyIsCheckedOnlyWhenPresentAndOnlyAmongTheRowsRead(t *testing.T) {
	unique := assessed(t, declaredOrders, cleanObservation(), [][]any{{"O-1", "1.00"}, {"O-2", "2.00"}})
	if unique.Grain.DeclaredKey != functioncontext.KeyUnique {
		t.Fatalf("%s", unique.Grain.DeclaredKey)
	}
	dup := assessed(t, declaredOrders, cleanObservation(), [][]any{{"O-1", "1.00"}, {"O-1", "9.99"}})
	if dup.Grain.DeclaredKey != functioncontext.KeyDuplicate {
		t.Fatalf("%s", dup.Grain.DeclaredKey)
	}
	absent := declaredOrders
	absent.PrimaryKey = []core.CanonicalID{"sales.order.not_in_result"}
	if c := assessed(t, absent, cleanObservation(), [][]any{{"O-1", "1.00"}}); c.Grain.DeclaredKey != functioncontext.KeyNotCheckable {
		t.Fatalf("%s", c.Grain.DeclaredKey)
	}
	if c := assessed(t, driver.DeclaredBrowseRelations{}, cleanObservation(), [][]any{{"O-1", "1.00"}}); c.Grain.DeclaredKey != functioncontext.KeyNotCheckable {
		t.Fatalf("no declaration must not be checkable: %s", c.Grain.DeclaredKey)
	}
}

func TestObservationsAreCarriedForTheConsumerAndDoNotChangeNullOrigin(t *testing.T) {
	c := assessed(t, declaredOrders, cleanObservation(), [][]any{{"O-1", nil}})
	if !c.Acquisition.Observed || c.Acquisition.Observation == nil || c.Acquisition.Declared.Status != driver.DeclaredRelationsResolved {
		t.Fatalf("%+v", c.Acquisition)
	}
	if !reflect.DeepEqual(c.Acquisition.Observation.PlanRelations, []string{"public.orders"}) {
		t.Fatalf("%+v", c.Acquisition.Observation)
	}
	if c.NullOrigin != "unestablished" {
		t.Fatalf("nullOrigin=%s: column provenance is not observed, so a NULL's origin stays open", c.NullOrigin)
	}
}

func contains(list []string, want string) bool {
	for _, s := range list {
		if s == want {
			return true
		}
	}
	return false
}
