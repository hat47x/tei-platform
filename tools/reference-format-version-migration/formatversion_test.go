package formatversion

import (
	"strings"
	"testing"
)

func strptr(s string) *string { return &s }

func base() Input {
	return Input{
		Current: "3",
		Rules: []Rule{
			{
				From:       "1",
				To:         "2",
				Renames:    []Rename{{From: "oldName", To: "name"}},
				SetDefault: []SetDefault{{Name: "strict", Value: "false"}},
				Removed:    []string{"legacyFlag"},
			},
			{From: "2", To: "3", SetDefault: []SetDefault{{Name: "kind", Value: "record"}}},
		},
		Documents: []Definition{
			{ID: "d-old", Version: "1", Kind: "format", Attrs: []Attr{
				{Name: "oldName", Value: strptr("orders")},
				{Name: "legacyFlag", Value: strptr("on")},
			}},
			{ID: "d-current", Version: "3", Kind: "format", Attrs: []Attr{
				{Name: "name", Value: strptr("orders")},
				{Name: "empty", Value: strptr("")},
			}},
		},
	}
}

func outcomeByID(r Report, id string) (Outcome, bool) {
	for _, o := range r.Outcomes {
		if o.Document == id {
			return o, true
		}
	}
	return Outcome{}, false
}

func attrOf(attrs []Attr, name string) (*string, bool) {
	for _, a := range attrs {
		if a.Name == name {
			return a.Value, true
		}
	}
	return nil, false
}

func hasItem(items []Item, code string) bool {
	for _, it := range items {
		if it.Code == code {
			return true
		}
	}
	return false
}

func TestMissingVersionIsNotDefaulted(t *testing.T) {
	in := base()
	in.Documents = append(in.Documents, Definition{ID: "d-noversion", Kind: "format"})
	r := Plan(in)
	o, _ := outcomeByID(r, "d-noversion")
	if o.Status != "failure" {
		t.Fatalf("status = %q, want failure", o.Status)
	}
	if !hasItem(o.Failures, "missing-format-version") {
		t.Errorf("failures = %+v", o.Failures)
	}
	if o.VersionRaised {
		t.Error("the version was raised for a definition without a recorded version")
	}
}

func TestContentVersionIsNotFormatVersion(t *testing.T) {
	in := Input{Current: "2", Documents: []Definition{{ID: "d", Version: "1", Kind: "content"}}}
	r := Plan(in)
	o, _ := outcomeByID(r, "d")
	if !hasItem(o.Failures, "content-version-not-format") {
		t.Errorf("failures = %+v", o.Failures)
	}
}

func TestNewerVersionRejectedWithRequiredTool(t *testing.T) {
	in := Input{Current: "3", Documents: []Definition{{ID: "d", Version: "4", Kind: "format"}}}
	r := Plan(in)
	o, _ := outcomeByID(r, "d")
	if !hasItem(o.Failures, "unsupported-newer-version") {
		t.Fatalf("failures = %+v", o.Failures)
	}
	found := false
	for _, f := range o.Failures {
		if strings.Contains(f.Detail, "4") {
			found = true
		}
	}
	if !found {
		t.Errorf("required tool version not shown: %+v", o.Failures)
	}
}

func TestCurrentVersionIsIdempotent(t *testing.T) {
	r := Plan(base())
	o, _ := outcomeByID(r, "d-current")
	if o.Status != "success" {
		t.Fatalf("status = %q, want success", o.Status)
	}
	if o.VersionRaised || len(o.Warnings) != 0 {
		t.Errorf("current version changed: %+v", o)
	}
	v, ok := attrOf(o.Attrs, "empty")
	if !ok || v == nil || *v != "" {
		t.Errorf("empty attribute not preserved: %+v", o.Attrs)
	}
}

func TestChainAppliesDefaultsAndWarnsRemoved(t *testing.T) {
	r := Plan(base())
	o, _ := outcomeByID(r, "d-old")
	if o.Status != "success-with-warning" {
		t.Fatalf("status = %q", o.Status)
	}
	if o.To != "3" || !o.VersionRaised {
		t.Errorf("version not raised correctly: %+v", o)
	}
	if len(o.Path) != 3 || o.Path[0] != "1" || o.Path[2] != "3" {
		t.Errorf("path = %+v", o.Path)
	}
	if !hasItem(o.Warnings, "attribute-removed") {
		t.Errorf("removed attribute not reported: %+v", o.Warnings)
	}
	name, ok := attrOf(o.Attrs, "name")
	if !ok || name == nil || *name != "orders" {
		t.Errorf("rename lost the value: %+v", o.Attrs)
	}
	if _, leaked := attrOf(o.Attrs, "oldName"); leaked {
		t.Errorf("old name retained: %+v", o.Attrs)
	}
	for _, want := range []string{"strict", "kind"} {
		v, ok := attrOf(o.Attrs, want)
		if !ok || v == nil || *v == "" {
			t.Errorf("default %q not written explicitly: %+v", want, o.Attrs)
		}
	}
}

func TestSilentDropIsFailureAndRetainsOriginal(t *testing.T) {
	in := Input{
		Current: "2",
		Rules:   []Rule{{From: "1", To: "2", DropBySilently: []string{"gone"}}},
		Documents: []Definition{{ID: "d", Version: "1", Kind: "format", Attrs: []Attr{
			{Name: "gone", Value: strptr("x")},
			{Name: "kept", Value: strptr("y")},
		}}},
	}
	r := Plan(in)
	o, _ := outcomeByID(r, "d")
	if o.Status != "failure" {
		t.Fatalf("status = %q, want failure", o.Status)
	}
	if !hasItem(o.Failures, "attribute-silently-dropped") {
		t.Errorf("failures = %+v", o.Failures)
	}
	if _, ok := attrOf(o.Attrs, "gone"); !ok {
		t.Error("a failed conversion changed the original definition")
	}
	if o.VersionRaised {
		t.Error("the version was raised on a failed conversion")
	}
}

func TestVersionContentMismatch(t *testing.T) {
	in := Input{Current: "2", Documents: []Definition{{ID: "d", Version: "2", Kind: "format", ContentVersion: "1"}}}
	r := Plan(in)
	o, _ := outcomeByID(r, "d")
	if !hasItem(o.Failures, "version-content-mismatch") {
		t.Errorf("failures = %+v", o.Failures)
	}
}

func TestStrategyRecommendation(t *testing.T) {
	if s := PlanStrategy(2, false); s.Mode != "chain" || s.Reason == "" {
		t.Errorf("strategy = %+v", s)
	}
	if s := PlanStrategy(5, false); s.Mode != "hub" || s.Reason == "" {
		t.Errorf("strategy = %+v", s)
	}
	if s := PlanStrategy(2, true); s.Mode != "hub" || s.Reason == "" {
		t.Errorf("strategy = %+v", s)
	}
}
