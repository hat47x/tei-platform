package generatedcode

import "testing"

func baseInput() Input {
	return Input{
		Artifacts: []Artifact{
			{ID: "a-customer", Language: "go", Interface: &Interface{Inputs: []string{"id"}, Outputs: []string{"json"}}, Contract: "contract-v2"},
			{ID: "a-report", Language: "go", Interface: &Interface{Inputs: []string{"from", "to"}, Outputs: []string{"csv"}}, Contract: "contract-v2"},
			{ID: "a-bad-auth", Language: "typescript", Interface: &Interface{Inputs: []string{"x"}, Outputs: []string{"y"}}, Authority: []string{"perm:missing"}, Contract: "contract-v2"},
			{ID: "a-noiface", Language: "go", Contract: "contract-v2"},
			{ID: "a-stale", Language: "go", Interface: &Interface{Inputs: []string{"x"}, Outputs: []string{"y"}}, Contract: "contract-v1"},
			{ID: "a-external", Language: "go", Interface: &Interface{Inputs: []string{"x"}, Outputs: []string{"y"}}, External: []string{"https://api.example/unknown"}, Contract: "contract-v2"},
			{ID: "a-provider", Language: "go", Interface: &Interface{Inputs: []string{"x"}, Outputs: []string{"y"}}, Providers: []string{"oracle"}, Contract: "contract-v2"},
		},
		KnownAuthority: []string{"perm:read"},
		KnownContracts: []string{"contract-v2"},
		KnownProviders: []string{"postgres"},
		KnownExternal:  []string{"https://api.example/known"},
	}
}

func promoted() Input {
	in := baseInput()
	in.Promotions = []Promotion{{Artifact: "a-customer", Kind: "function", Model: "Fn.Customer"}}
	return in
}

func connMode(r Report, id string) string {
	for _, c := range r.Connections {
		if c.Artifact == id {
			return c.Mode
		}
	}
	return ""
}

func connCallable(r Report, id string) bool {
	for _, c := range r.Connections {
		if c.Artifact == id {
			return c.Callable
		}
	}
	return false
}

func hasFinding(findings []Finding, code string) bool {
	for _, f := range findings {
		if f.Code == code {
			return true
		}
	}
	return false
}

func TestPlanDetectsBlockingReasons(t *testing.T) {
	r := Plan(baseInput())
	cases := map[string]string{
		"a-bad-auth": "unknown-authority",
		"a-noiface":  "missing-interface",
		"a-stale":    "stale-contract",
		"a-external": "external-call-outside-contract",
		"a-provider": "unsupported-provider",
	}
	for id, code := range cases {
		if connMode(r, id) != "unsupported" {
			t.Errorf("%s mode = %q, want unsupported", id, connMode(r, id))
		}
		if !hasFinding(r.Findings, code) {
			t.Errorf("missing finding %q for %s in %+v", code, id, r.Findings)
		}
	}
}

func TestPlanMarksCallableAsNative(t *testing.T) {
	r := Plan(baseInput())
	for _, id := range []string{"a-customer", "a-report"} {
		if !connCallable(r, id) || connMode(r, id) != "native" {
			t.Errorf("%s = callable:%v mode:%q, want callable native", id, connCallable(r, id), connMode(r, id))
		}
	}
	if r.Summary.Connectable != 2 || r.Summary.Unsupported != 5 {
		t.Errorf("summary = %+v", r.Summary)
	}
	if !hasFinding(r.Findings, "native-code-retained") {
		t.Errorf("missing native-code-retained in %+v", r.Findings)
	}
}

func TestPromotionKeepsMeaningUnaccepted(t *testing.T) {
	r := Plan(promoted())
	if len(r.Promotions) != 1 {
		t.Fatalf("promotions = %+v", r.Promotions)
	}
	p := r.Promotions[0]
	if !p.Accepted {
		t.Fatalf("promotion not accepted: %+v", p)
	}
	if p.MeaningAccepted {
		t.Fatal("connecting the code accepted its execution meaning")
	}
	if !hasFinding(r.Findings, "external-code-not-accepted") {
		t.Errorf("missing external-code-not-accepted in %+v", r.Findings)
	}
	if connMode(r, "a-customer") != "capability" {
		t.Errorf("promoted artifact mode = %q", connMode(r, "a-customer"))
	}
}

func TestPromotionOfNonCallableRefused(t *testing.T) {
	in := baseInput()
	in.Promotions = []Promotion{{Artifact: "a-noiface", Kind: "function", Model: "Fn.X"}}
	r := Plan(in)
	if r.Promotions[0].Accepted {
		t.Fatal("a non-callable artifact was promoted")
	}
	if !hasFinding(r.Findings, "capability-not-callable") {
		t.Errorf("missing capability-not-callable in %+v", r.Findings)
	}
}

func TestPromotionOfUnknownArtifact(t *testing.T) {
	in := baseInput()
	in.Promotions = []Promotion{{Artifact: "ghost", Kind: "function", Model: "Fn.Ghost"}}
	r := Plan(in)
	if r.Promotions[0].Accepted {
		t.Fatal("an unknown artifact was promoted")
	}
	if !hasFinding(r.Findings, "promotion-of-unknown-artifact") {
		t.Errorf("missing promotion-of-unknown-artifact in %+v", r.Findings)
	}
}

func TestPartialModelingRetainsRest(t *testing.T) {
	r := Plan(promoted())
	if r.Summary.Promoted != 1 || r.Summary.Native != 1 {
		t.Fatalf("summary = %+v", r.Summary)
	}
	found := false
	for _, id := range r.RetainedNative {
		if id == "a-report" {
			found = true
		}
	}
	if !found {
		t.Fatalf("the unmodelled artifact was not retained: %+v", r.RetainedNative)
	}
}

func TestCompareMeasuresChange(t *testing.T) {
	cmp := Compare(baseInput(), promoted())
	if cmp.PromotedDelta != 1 {
		t.Errorf("promotedDelta = %d, want 1", cmp.PromotedDelta)
	}
	if cmp.NativeDelta != -1 {
		t.Errorf("nativeDelta = %d, want -1", cmp.NativeDelta)
	}
	if cmp.ConnectableDelta != 0 {
		t.Errorf("connectableDelta = %d, want 0", cmp.ConnectableDelta)
	}
	if len(cmp.ModeChanges) != 1 || cmp.ModeChanges[0].Artifact != "a-customer" ||
		cmp.ModeChanges[0].From != "native" || cmp.ModeChanges[0].To != "capability" {
		t.Errorf("modeChanges = %+v", cmp.ModeChanges)
	}
}
