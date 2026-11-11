package traitfulfillment

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"tei/internal/composition"
	"tei/internal/inspectoradapter"
	"tei/internal/operations"
	"tei/internal/sourceindex"
)

// The expectations in this file are written by hand. None is derived from the
// function under test, the Registry, or an inventory.

// inspectTrait is the one atomic Trait of this reference: the existing authoring
// Inspector Contract. Its literal value is repeated here on purpose, as the
// independent expectation of what inspectoradapter composes.
var inspectTrait = composition.Contract{ID: "tei.operations.inspect", Version: "0", Shape: "bounded-reference-v0"}

const (
	scope          = "authoring-tool"
	subjectOrder   = "sales.order"
	standardID     = "tei.reference.inspector.local"
	manualID       = "sales.inspector.manual"
	wantManual     = "tei/tools/reference-trait-fulfillment.manualInspect"
	wantWrong      = "tei/tools/reference-trait-fulfillment.signatureOnlyInspect"
	wantStandard   = "tei/internal/operations.InspectSubject"
	reasonNoKnown  = "no-known-provider"
	reasonDeclared = "declaration-only"
)

var orderSubject = sourceindex.Subject{Kind: "semantic-data", ID: subjectOrder}

// manualInspect is a hand-written implementation that keeps the invariant.
func manualInspect(root string, subject sourceindex.Subject) (operations.SubjectInspection, error) {
	return operations.InspectSubject(root, subject)
}

// signatureOnlyInspect has the same Go signature and returns a well-formed
// answer, but for another Subject. Only the invariant tells it apart.
func signatureOnlyInspect(root string, _ sourceindex.Subject) (operations.SubjectInspection, error) {
	return operations.InspectSubject(root, sourceindex.Subject{Kind: "semantic-data", ID: "sales.customer"})
}

func manualRequired() Obligation {
	return Obligation{Trait: inspectTrait, Scope: scope, Subject: subjectOrder,
		Level: LevelRequired, Method: MethodManualRequired, ImplementedBy: manualID}
}

// registryWith declares the given Provider IDs as offering the Trait. The
// declaration is all a Registry holds: no code, no artifact.
func registryWith(t *testing.T, ids ...string) (*composition.Registry, uint64) {
	t.Helper()
	r := composition.NewRegistry()
	var providers []composition.Provider
	for _, id := range ids {
		providers = append(providers, composition.Provider{ID: id, Scope: scope, Offers: inspectTrait})
	}
	snapshot, err := r.Apply(0, providers, nil)
	if err != nil {
		t.Fatal(err)
	}
	return r, snapshot.Revision
}

func complete(revision uint64) Closure {
	return Closure{SourceComplete: true, RequiredComplete: true, CatalogComplete: true, CatalogRevision: revision}
}

func workspaceRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "semantic"), 0755); err != nil {
		t.Fatal(err)
	}
	manifest := "workspace: alpha\nformatVersion: 1\nsources:\n  - semantic\n"
	source := "namespace sales\n\ndata Order {\n    attribute orderId type std.identifier\n}\n\n" +
		"data Customer {\n    attribute customerId type std.identifier\n}\n"
	if err := os.WriteFile(filepath.Join(root, "tei.workspace.yaml"), []byte(manifest), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "semantic", "sales.tei"), []byte(source), 0644); err != nil {
		t.Fatal(err)
	}
	return root
}

// observe makes one real call through the existing Inspector Host and returns
// what the Host and the consumer saw. No part of it is a fake of that route.
func observe(t *testing.T, provider inspectoradapter.LocalProvider, expectedEntry string) Observation {
	t.Helper()
	host, err := inspectoradapter.NewLocalInspectorHostSelecting(provider.ID, []inspectoradapter.LocalProvider{provider})
	if err != nil {
		t.Fatal(err)
	}
	var seen inspectoradapter.Invocation
	host.Observe(func(i inspectoradapter.Invocation) { seen = i })
	selected, err := host.Bind(workspaceRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	_, consumerErr := selected.Inspect(orderSubject)
	return Observation{Invocation: seen, ExpectedEntry: expectedEntry, ConsumerChecked: true, ConsumerErr: consumerErr}
}

func only(t *testing.T, report Report) Finding {
	t.Helper()
	if len(report.Findings) != 1 {
		t.Fatalf("want one finding, got %+v", report.Findings)
	}
	return report.Findings[0]
}

// 1. Everything needed was confirmed, and only the manual implementation is
// missing. The standard Provider of the same Contract is not used instead.
func TestManualRequiredMissingWithCompleteClosureIsUnfulfilled(t *testing.T) {
	registry, revision := registryWith(t, standardID)
	report := Assess(registry, []Obligation{manualRequired()}, complete(revision), nil)
	got := only(t, report)
	if report.Summary != "unfulfilled" || got.State != "unfulfilled" || got.Reason != reasonNoKnown {
		t.Fatalf("missing manual implementation was not reported: %+v", report)
	}
	if got.ProviderID != "" || !reflect.DeepEqual(got.NotChosen, []string{standardID}) {
		t.Fatalf("the standard Provider stood in for the manual one: %+v", got)
	}
	if got.Subject != subjectOrder || got.Trait != inspectTrait || got.Scope != scope {
		t.Fatalf("the finding does not name what is missing where: %+v", got)
	}
}

// 2. The same absence, with something not loaded, is not a confirmed gap.
func TestMissingImplementationIsUnknownUntilTheClosureIsConfirmed(t *testing.T) {
	registry, revision := registryWith(t, standardID)
	cases := map[string]Closure{
		"nothing stated":     {},
		"source partial":     {RequiredComplete: true, CatalogComplete: true, CatalogRevision: revision},
		"catalog not loaded": {SourceComplete: true, RequiredComplete: true, CatalogRevision: revision},
		"stale catalog":      {SourceComplete: true, RequiredComplete: true, CatalogComplete: true, CatalogRevision: revision + 1},
	}
	for name, closure := range cases {
		report := Assess(registry, []Obligation{manualRequired()}, closure, nil)
		if got := only(t, report); report.Summary != "unknown" || got.State != "unknown" {
			t.Fatalf("%s: absence was concluded from incomplete inputs: %+v", name, report)
		}
	}
	if report := Assess(nil, []Obligation{manualRequired()}, complete(0), nil); report.Summary != "unknown" {
		t.Fatalf("a missing Registry was read as an empty one: %+v", report)
	}
}

// 2b. A Required set that was only partly obtained: a known gap is still
// reported, but declared findings do not add up to an overall result.
func TestPartialRequiredSetReportsKnownGapsButNoOverallFulfilment(t *testing.T) {
	registry, revision := registryWith(t, standardID)
	closure := complete(revision)
	closure.RequiredComplete = false
	if report := Assess(registry, []Obligation{manualRequired()}, closure, nil); report.Summary != "unfulfilled" {
		t.Fatalf("a confirmed gap was hidden by a partial Required set: %+v", report)
	}
	declared, declaredRevision := registryWith(t, manualID)
	closure = complete(declaredRevision)
	closure.RequiredComplete = false
	report := Assess(declared, []Obligation{manualRequired()}, closure, nil)
	if got := only(t, report); report.Summary != "partial" || got.State != "declared-unverified" {
		t.Fatalf("a partial Required set was summarised as fulfilled: %+v", report)
	}
	if report := Assess(declared, nil, closure, nil); report.Summary != "unknown" {
		t.Fatalf("no obtained requirement was read as nothing required: %+v", report)
	}
	if report := Assess(declared, nil, complete(declaredRevision), nil); report.Summary != "" {
		t.Fatalf("a confirmed empty Required set must stay empty: %+v", report)
	}
}

// 3. A Provider that declares the Contract, or whose code merely has the right
// signature, has not been shown to conform.
func TestDeclarationOrSignatureAloneIsNotConformance(t *testing.T) {
	registry, revision := registryWith(t, manualID)
	closure := complete(revision)

	selfDeclared := only(t, Assess(registry, []Obligation{manualRequired()}, closure, nil))
	if selfDeclared.State != "declared-unverified" || selfDeclared.Reason != reasonDeclared ||
		selfDeclared.ProviderID != manualID || selfDeclared.Observed != nil {
		t.Fatalf("a self-declaration was raised: %+v", selfDeclared)
	}

	// Same signature, real call, answer for another Subject.
	wrong := observe(t, inspectoradapter.LocalProvider{ID: manualID, Entry: wantWrong, Inspect: signatureOnlyInspect}, wantWrong)
	if wrong.Invocation.ObservedEntry != wantWrong || wrong.ConsumerErr == nil {
		t.Fatalf("the signature-only entry was not entered and rejected: %+v", wrong)
	}
	got := only(t, Assess(registry, []Obligation{manualRequired()}, closure, []Observation{wrong}))
	if got.State != "declared-unverified" || got.Reason != "invariant-rejected-the-answer" || got.Observed != nil {
		t.Fatalf("a matching signature was accepted as conformance: %+v", got)
	}

	// The right label on other code: the standard function under the manual ID.
	relabelled := observe(t, inspectoradapter.LocalProvider{ID: manualID, Entry: wantStandard, Inspect: operations.InspectSubject}, wantManual)
	got = only(t, Assess(registry, []Obligation{manualRequired()}, closure, []Observation{relabelled}))
	if got.State != "declared-unverified" || got.Reason != "entered-code-is-not-the-expected-implementation" {
		t.Fatalf("another implementation under the manual ID was accepted: %+v", got)
	}

	// A call whose answer nobody checked against the invariant.
	unchecked := observe(t, inspectoradapter.LocalProvider{ID: manualID, Entry: wantManual, Inspect: manualInspect}, wantManual)
	unchecked.ConsumerChecked = false
	got = only(t, Assess(registry, []Obligation{manualRequired()}, closure, []Observation{unchecked}))
	if got.State != "declared-unverified" || got.Reason != "invariant-not-checked" {
		t.Fatalf("an unchecked call was accepted: %+v", got)
	}
}

// 3b. The manual implementation, once really supplied and observed.
func TestManualImplementationObservedKeepingTheInvariant(t *testing.T) {
	registry, revision := registryWith(t, standardID, manualID)
	seen := observe(t, inspectoradapter.LocalProvider{ID: manualID, Entry: wantManual, Inspect: manualInspect}, wantManual)
	report := Assess(registry, []Obligation{manualRequired()}, complete(revision), []Observation{seen})
	got := only(t, report)
	want := []string{"selected:" + manualID, "entered:" + wantManual, "origin:host-manifest", "invariant:tei.operations.inspect"}
	if report.Summary != "invariant-observed" || got.ProviderID != manualID || !reflect.DeepEqual(got.Observed, want) {
		t.Fatalf("the supplied manual implementation was not recognised: %+v", report)
	}
}

// 4. `provided`: an eligible existing Provider. The result states only the
// facts that were seen, and names what it does not establish.
func TestProvidedShowsOnlyTheVerifiedRange(t *testing.T) {
	registry, revision := registryWith(t, standardID)
	provided := Obligation{Trait: inspectTrait, Scope: scope, Subject: subjectOrder, Level: LevelRequired, Method: MethodProvided}

	declared := Assess(registry, []Obligation{provided}, complete(revision), nil)
	if got := only(t, declared); declared.Summary != "declared-unverified" || got.ProviderID != standardID {
		t.Fatalf("an unobserved Provider was reported beyond its declaration: %+v", declared)
	}

	seen := observe(t, inspectoradapter.ReferenceInventory()[0], wantStandard)
	report := Assess(registry, []Obligation{provided}, complete(revision), []Observation{seen})
	got := only(t, report)
	want := []string{"selected:" + standardID, "entered:" + wantStandard, "origin:host-manifest", "invariant:tei.operations.inspect"}
	if report.Summary != "invariant-observed" || got.Reason != "one-observed-call" || !reflect.DeepEqual(got.Observed, want) {
		t.Fatalf("the observed range is not exactly what was seen: %+v", report)
	}
	wantNot := []string{
		"formal Provider artifact entry invoked by a product Consumer (#1251)",
		"verified BOM for an exact Provider/Consumer/Target lock (#3267)",
		"current Host Grant, disclosure and Effect (#1174)",
		"Owner acceptance of meaning",
	}
	if !reflect.DeepEqual(report.NotQualified, wantNot) {
		t.Fatalf("the report does not withhold the separate qualifications: %+v", report.NotQualified)
	}

	// Two Providers of one Contract: `provided` does not pick one.
	ambiguous, ambiguousRevision := registryWith(t, standardID, manualID)
	if got := only(t, Assess(ambiguous, []Obligation{provided}, complete(ambiguousRevision), []Observation{seen})); got.State != "conflict" || got.ProviderID != "" {
		t.Fatalf("an ambiguous Provider was chosen: %+v", got)
	}
}

// 5. An advisory Trait is never an unmet obligation, and never changes the
// result of a required one.
func TestAdvisoryIsNotAnObligation(t *testing.T) {
	registry, revision := registryWith(t, standardID)
	advisory := manualRequired()
	advisory.Level = LevelAdvisory

	report := Assess(registry, []Obligation{advisory}, complete(revision), nil)
	if got := only(t, report); report.Summary != "" || got.State != "not-provided" || got.Level != "advisory" {
		t.Fatalf("an advisory Trait was treated as required: %+v", report)
	}

	other := Obligation{Trait: inspectTrait, Scope: scope, Subject: "sales.customer", Level: LevelRequired, Method: MethodProvided}
	report = Assess(registry, []Obligation{advisory, other}, complete(revision), nil)
	if report.Summary != "declared-unverified" || len(report.Findings) != 2 {
		t.Fatalf("an unmet advisory Trait changed the required result: %+v", report)
	}

	// A declared advisory implementation does not lift the summary either.
	both, bothRevision := registryWith(t, manualID)
	report = Assess(both, []Obligation{advisory}, complete(bothRevision), nil)
	if got := only(t, report); report.Summary != "" || got.State != "declared-unverified" {
		t.Fatalf("an advisory finding leaked into the required summary: %+v", report)
	}
}

// 6. The same Trait repeated, or composed in the other order, gives the same
// answer.
func TestDuplicateAndReversedCompositionDoNotChangeTheResult(t *testing.T) {
	registry, revision := registryWith(t, standardID)
	closure := complete(revision)
	manual := manualRequired()
	advisory := manualRequired()
	advisory.Level = LevelAdvisory
	other := Obligation{Trait: inspectTrait, Scope: scope, Subject: "sales.customer", Level: LevelRequired, Method: MethodProvided}

	base := Assess(registry, []Obligation{manual, other}, closure, nil)
	if len(base.Findings) != 2 || base.Summary != "unfulfilled" {
		t.Fatalf("unexpected base result: %+v", base)
	}
	for name, set := range map[string][]Obligation{
		"reversed":                 {other, manual},
		"duplicated":               {manual, other, manual, other},
		"advisory before required": {advisory, manual, other},
		"advisory after required":  {other, manual, advisory},
	} {
		if got := Assess(registry, set, closure, nil); !reflect.DeepEqual(got, base) {
			t.Fatalf("%s changed the result:\n got %+v\nwant %+v", name, got, base)
		}
	}

	// Two methods for one Trait on one Subject contradict, in either order.
	providedSame := Obligation{Trait: inspectTrait, Scope: scope, Subject: subjectOrder, Level: LevelRequired, Method: MethodProvided}
	forward := Assess(registry, []Obligation{manual, providedSame}, closure, nil)
	backward := Assess(registry, []Obligation{providedSame, manual}, closure, nil)
	if forward.Summary != "conflict" || backward.Summary != "conflict" ||
		only(t, forward).State != only(t, backward).State || only(t, forward).Reason != only(t, backward).Reason {
		t.Fatalf("contradictory methods were resolved by order: %+v / %+v", forward, backward)
	}
}

// A malformed Obligation is not a gap and not a pass.
func TestInvalidObligationIsUnknown(t *testing.T) {
	registry, revision := registryWith(t, standardID)
	for name, o := range map[string]Obligation{
		"manual without implementer": {Trait: inspectTrait, Scope: scope, Subject: subjectOrder, Level: LevelRequired, Method: MethodManualRequired},
		"unknown level":              {Trait: inspectTrait, Scope: scope, Subject: subjectOrder, Level: "must", Method: MethodProvided},
		"unknown method":             {Trait: inspectTrait, Scope: scope, Subject: subjectOrder, Level: LevelRequired, Method: "generated"},
		"no subject":                 {Trait: inspectTrait, Scope: scope, Level: LevelRequired, Method: MethodProvided},
	} {
		if got := only(t, Assess(registry, []Obligation{o}, complete(revision), nil)); got.State != "unknown" || got.Reason != "invalid-obligation" {
			t.Fatalf("%s: %+v", name, got)
		}
	}
}
