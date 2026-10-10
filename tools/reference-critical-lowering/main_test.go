package main

import (
	"encoding/json"
	"strings"
	"testing"
)

func ref(id string) Ref { return Ref{ID: id, Version: "1.0.0", Fingerprint: "sha256:" + id} }

type stubTrust struct {
	denyAcceptance, denyPublisher, denyConformance bool
	denyResolution, denyRuntimeBinding, denyData, denyContext, denyInvocation bool
	expectedPlan string
}

func (s stubTrust) Accepted(d string, receipt, snapshot Ref) bool {
	return !s.denyAcceptance && d != "" && receipt.valid() && snapshot.valid()
}
func (s stubTrust) Published(r Ref) bool { return !s.denyPublisher && r.valid() }
func (s stubTrust) Conformant(p Provider) bool {
	return !s.denyConformance && p.Conformance.valid()
}
func (s stubTrust) ResolutionCurrent(p Plan) bool {
	return !s.denyResolution && (s.expectedPlan == "" || Digest(p) == s.expectedPlan)
}
func (s stubTrust) RuntimeBindingCurrent(step Step) bool { return !s.denyRuntimeBinding }
func (s stubTrust) DataCompatible(step Step) bool { return !s.denyData }
func (s stubTrust) ContextBound(step Step) bool { return !s.denyContext }
func (s stubTrust) Authorized(step Step) bool {
	return !s.denyInvocation && step.Contract.valid()
}

type fixture struct {
	c         Candidate
	a         Acceptance
	wiring    []Wiring
	providers []Provider
	target    Target
}

func orderFixture(language string) fixture {
	// Subject IDs are references to the existing order_entry scenario; contract and
	// provider refs below are deliberately synthetic until #951 production wiring exists.
	slots := []Ref{ref("tei.context.verified-principal")}
	need := []Requirement{
		{Contract: ref("tei.test.customer-query"), Scope: "application-runtime", Effect: "read",
			DataContracts: []DataNeed{{Datasource: "master", Database: "master_data", Contract: "master_data.customer", VersionConstraint: "^2"}}},
		{Contract: ref("tei.test.order-submit"), Scope: "application-runtime", Effect: "write", RequiredSlots: slots,
			DataContracts: []DataNeed{
				{Datasource: "sales", Database: "sales_core", Contract: "sales_core.order", VersionConstraint: "^4"},
				{Datasource: "sales", Database: "sales_core", Contract: "sales_core.order_line", VersionConstraint: "^1"},
			}},
	}
	c := Candidate{
		Schema: CandidateSchema,
		Source: ref("reference/workspace/applications/order-entry/application.tei"),
		SourceInputs: []Ref{
			ref("reference/workspace/applications/order-entry/patterns/submit-order.tei"),
			ref("reference/workspace/semantic/sales/order.tei"),
			ref("reference/workspace/semantic/sales/validation.tei"),
		},
		Language: ref(language),
		Subjects: []Subject{
			{Kind: "business", Ref: ref("order_entry.submit_order")},
			{Kind: "data", Ref: ref("sales.order")},
			{Kind: "function", Ref: ref("sales.order.submit")},
		},
		Requirements: need,
	}
	wiring := []Wiring{
		{Scope: need[0].Scope, Contract: need[0].Contract, Provider: ref("tei.test.customer-reader.web")},
		{Scope: need[1].Scope, Contract: need[1].Contract, Provider: ref("tei.test.order-writer.web")},
	}
	providers := []Provider{
		{
			Identity: wiring[0].Provider, Contract: need[0].Contract, Scope: need[0].Scope,
			Target: "web", Conformance: ref("test.customer.conformance"),
			ResolutionLock: ref("existing.capability-lock.customer"),
			Artifact: ref("existing.artifact.customer"),
			RuntimeDeps: []Ref{ref("runtime.customer-service")},
			BuildOnlyDeps: []Ref{ref("build.codegen-tool")},
		},
		{
			Identity: wiring[1].Provider, Contract: need[1].Contract, Scope: need[1].Scope,
			Target: "web", Conformance: ref("test.submit.conformance"),
			ResolutionLock: ref("existing.capability-lock.submit"),
			Artifact: ref("existing.artifact.submit"), Slots: append([]Ref(nil), slots...),
			RuntimeDeps: []Ref{ref("runtime.order-service")},
			BuildOnlyDeps: []Ref{ref("build.compiler")},
		},
	}
	return fixture{
		c: c, a: Acceptance{Schema: AcceptanceSchema, CandidateDigest: Digest(c), Snapshot: ref("accepted.order_entry.snapshot"), Receipt: ref("test.trusted.receipt")},
		wiring: wiring, providers: providers, target: Target{ID: "web", Effects: []string{"read", "write"}},
	}
}

func (f fixture) lower(trust Trust) (Plan, error) {
	return Lower(f.c, f.a, f.wiring, f.providers, f.target, trust)
}

func code(err error) string {
	if err == nil {
		return ""
	}
	f, ok := err.(Failure)
	if !ok {
		return "unexpected-error-type"
	}
	return f.Code
}

func TestWebOrderScenarioWithTwoLanguageProducers(t *testing.T) {
	for _, language := range []string{"tei.language.dsl", "tei.language.sql-projection"} {
		t.Run(language, func(t *testing.T) {
			f := orderFixture(language)
			neutral, err := Neutralize(f.c, f.a, stubTrust{})
			if err != nil || neutral.Schema != NeutralSchema || len(neutral.Intents) != 2 {
				t.Fatalf("neutral boundary: %v", err)
			}
			got, err := f.lower(stubTrust{})
			if err != nil {
				t.Fatal(err)
			}
			if got.Schema != Schema || got.Snapshot != f.a.Snapshot || got.Origin.AcceptanceReceipt != f.a.Receipt ||
				got.Origin.CandidateDigest != Digest(f.c) || got.Origin.Source != f.c.Source || len(got.Steps) != 2 {
				t.Fatalf("bad plan: %+v", got)
			}
			if got.Steps[1].Effect != "write" ||
				!equalSlots(got.Steps[1].RequiredSlots, f.c.Requirements[1].RequiredSlots) ||
				len(got.Steps[1].DataContracts) != 2 {
				t.Fatalf("lost effect or context declaration: %+v", got.Steps)
			}
			b, _ := json.Marshal(got)
			if strings.Contains(string(b), "build.compiler") || strings.Contains(string(b), "build.codegen-tool") ||
				strings.Contains(string(b), "session") || strings.Contains(string(b), "credential") {
				t.Fatalf("lowering leaked runtime/build-only information: %s", b)
			}
			if err := AuthorizeInvocation(got, stubTrust{}); err != nil {
				t.Fatal(err)
			}
			reordered := f
			reordered.providers = []Provider{f.providers[1], f.providers[0]}
			again, err := reordered.lower(stubTrust{})
			if err != nil || Digest(again) != Digest(got) {
				t.Fatalf("provider registration order changed resolution: %v", err)
			}
		})
	}
}

func TestFailClosedNegativeControls(t *testing.T) {
	tests := []struct {
		name string
		edit func(*fixture)
		trust stubTrust
		want string
	}{
		{"missing-capability", func(f *fixture) { f.providers = f.providers[1:] }, stubTrust{}, "missing-capability"},
		{"provider-version", func(f *fixture) { f.providers[0].Identity.Version = "2.0.0" }, stubTrust{}, "incompatible-version"},
		{"contract-version", func(f *fixture) { f.providers[0].Contract.Version = "2.0.0" }, stubTrust{}, "incompatible-version"},
		{"contract-fingerprint", func(f *fixture) { f.providers[0].Contract.Fingerprint = "drift" }, stubTrust{}, "contract-drift"},
		{"provider-scope", func(f *fixture) { f.providers[0].Scope = "authoring-tool" }, stubTrust{}, "conformance-or-context-drift"},
		{"required-context", func(f *fixture) { f.providers[1].Slots = nil }, stubTrust{}, "conformance-or-context-drift"},
		{"context-version", func(f *fixture) { f.providers[1].Slots[0].Version = "2.0.0" }, stubTrust{}, "conformance-or-context-drift"},
		{"conformance", func(f *fixture) { f.providers[0].Conformance = Ref{} }, stubTrust{}, "conformance-or-context-drift"},
		{"missing-wiring", func(f *fixture) { f.wiring = f.wiring[1:] }, stubTrust{}, "missing-wiring"},
		{"ambiguous-wiring", func(f *fixture) { f.wiring = append(f.wiring, f.wiring[0]) }, stubTrust{}, "ambiguous-wiring"},
		{"ambiguous-provider", func(f *fixture) { f.providers = append(f.providers, f.providers[0]) }, stubTrust{}, "ambiguous-provider"},
		{"unsupported-effect", func(f *fixture) { f.target.Effects = []string{"read"} }, stubTrust{}, "unsupported-target"},
		{"wrong-target", func(f *fixture) { f.target.ID = "desktop" }, stubTrust{}, "unsupported-target"},
		{"wrong-candidate-schema", func(f *fixture) { f.c.Schema = "tei.lowering-candidate/v99" }, stubTrust{}, "invalid-boundary"},
		{"unaccepted-candidate", func(f *fixture) { f.c.Source = ref("changed.file") }, stubTrust{}, "unaccepted-meaning"},
		{"acceptance-authority", func(f *fixture) {}, stubTrust{denyAcceptance: true}, "unaccepted-meaning"},
		{"publisher-authority", func(f *fixture) {}, stubTrust{denyPublisher: true}, "unverified-provider"},
		{"provider-conformance", func(f *fixture) {}, stubTrust{denyConformance: true}, "unverified-provider"},
		{"invalid-runtime-dependency", func(f *fixture) { f.providers[0].RuntimeDeps = []Ref{{}} }, stubTrust{}, "invalid-provider-dependency"},
		{"duplicate-runtime-dependency", func(f *fixture) {
			f.providers[0].RuntimeDeps = append(f.providers[0].RuntimeDeps, f.providers[0].RuntimeDeps[0])
		}, stubTrust{}, "invalid-provider-dependency"},
		{"invalid-build-only-dependency", func(f *fixture) { f.providers[0].BuildOnlyDeps = []Ref{{}} }, stubTrust{}, "invalid-provider-dependency"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			f := orderFixture("tei.language.dsl")
			tc.edit(&f)
			_, err := f.lower(tc.trust)
			if got := code(err); got != tc.want {
				t.Fatalf("got %q; want %q", got, tc.want)
			}
		})
	}
}

func TestCurrentAuthorizationIsNotCachedIntoPlan(t *testing.T) {
	f := orderFixture("tei.language.dsl")
	plan, err := f.lower(stubTrust{denyInvocation: true})
	if err != nil {
		t.Fatal(err)
	}
	if got := code(AuthorizeInvocation(plan, stubTrust{denyResolution: true})); got != "stale-or-untrusted-resolution" {
		t.Fatalf("stale plan was accepted: %s", got)
	}
	if got := code(AuthorizeInvocation(plan, stubTrust{denyRuntimeBinding: true})); got != "runtime-binding-unavailable" {
		t.Fatalf("current provider/target binding was not checked: %s", got)
	}
	if got := code(AuthorizeInvocation(plan, stubTrust{denyData: true})); got != "data-contract-incompatible" {
		t.Fatalf("current database schema incompatibility was not checked: %s", got)
	}
	if got := code(AuthorizeInvocation(plan, stubTrust{denyContext: true})); got != "missing-execution-context" {
		t.Fatalf("missing invocation context was not fail-closed: %s", got)
	}
	if got := code(AuthorizeInvocation(plan, stubTrust{denyInvocation: true})); got != "authorization-denied" {
		t.Fatalf("authorization was not fail-closed: %s", got)
	}
	if err := AuthorizeInvocation(plan, stubTrust{}); err != nil {
		t.Fatal(err)
	}
}

func TestSecondTargetIsConceptOnly(t *testing.T) {
	f := orderFixture("tei.language.dsl")
	neutralBefore, err := Neutralize(f.c, f.a, stubTrust{})
	if err != nil { t.Fatal(err) }
	web, err := f.lower(stubTrust{})
	if err != nil {
		t.Fatal(err)
	}
	f.target.ID = "desktop"
	for i := range f.providers {
		f.providers[i].Target = "desktop"
		f.providers[i].Identity.ID += ".desktop"
		f.wiring[i].Provider = f.providers[i].Identity
	}
	neutralAfter, err := Neutralize(f.c, f.a, stubTrust{})
	if err != nil || Digest(neutralBefore) != Digest(neutralAfter) { t.Fatal("Target changed neutral intent", err) }
	desktop, err := f.lower(stubTrust{})
	if err != nil || desktop.Snapshot != web.Snapshot {
		t.Fatalf("meaning changed with Target: %v", err)
	}
	if Digest(desktop) == Digest(web) {
		t.Fatal("target difference lost")
	}
	// No desktop binary, desktop UI, or actual desktop behavior was executed.
}

func TestSelectiveCacheKeys(t *testing.T) {
	f := orderFixture("tei.language.dsl")
	plan, err := f.lower(stubTrust{})
	if err != nil {
		t.Fatal(err)
	}
	k := Keys(f.c, f.a, plan, f.target, ref("builder-v1"), []Ref{ref("build.codegen-tool"), ref("build.compiler")})
	unrelated := f
	unrelated.providers = append(unrelated.providers, Provider{Identity: ref("unrelated")})
	again, err := unrelated.lower(stubTrust{})
	if err != nil || Keys(unrelated.c, unrelated.a, again, unrelated.target, ref("builder-v1"), []Ref{ref("build.compiler"), ref("build.codegen-tool")}) != k {
		t.Fatal("unrelated Provider invalidated accepted resolution")
	}

	reorderedEffects := Keys(f.c, f.a, plan,
		Target{ID: f.target.ID, Effects: []string{"write", "read"}},
		ref("builder-v1"), []Ref{ref("build.compiler"), ref("build.codegen-tool")})
	if reorderedEffects != k {
		t.Fatal("Target effect set ordering caused needless cache invalidation")
	}
	changedTarget := Keys(f.c, f.a, plan, Target{ID: "alternate", Effects: f.target.Effects}, ref("builder-v1"), []Ref{ref("build.compiler"), ref("build.codegen-tool")})
	if changedTarget.Interpret != k.Interpret || changedTarget.Accept != k.Accept ||
		changedTarget.Resolve != k.Resolve || changedTarget.Target == k.Target || changedTarget.Build == k.Build {
		t.Fatal("Target changes must not invalidate prior stages")
	}
	newBuilder := Keys(f.c, f.a, plan, f.target, ref("builder-v2"), []Ref{ref("build.compiler"), ref("build.codegen-tool")})
	if newBuilder.Target != k.Target || newBuilder.Build == k.Build {
		t.Fatal("builder change must only invalidate artifact stage")
	}
	// Build-only tool changes do not belong to resolution or runtime plan,
	// but must invalidate the artifact build stage even with an unchanged builder.
	otherBuildInput := Keys(f.c, f.a, plan, f.target, ref("builder-v1"), []Ref{ref("build.compiler-v2"), ref("build.codegen-tool")})
	if otherBuildInput.Interpret != k.Interpret || otherBuildInput.Accept != k.Accept ||
		otherBuildInput.Resolve != k.Resolve || otherBuildInput.Target != k.Target ||
		otherBuildInput.Build == k.Build {
		t.Fatal("build-only dependency change failed selective artifact invalidation")
	}
	changedSource := f.c
	changedSource.Source = ref("changed-source")
	recomputed := Keys(changedSource, f.a, plan, f.target, ref("builder-v1"), []Ref{ref("build.compiler"), ref("build.codegen-tool")})
	if recomputed.Interpret == k.Interpret || recomputed.Accept == k.Accept {
		t.Fatal("source change must invalidate interpretation and acceptance")
	}
}

func TestTargetSpecificProviderSelectionInvalidatesResolveButNotAcceptedMeaning(t *testing.T) {
	web := orderFixture("tei.language.dsl")
	webPlan, err := web.lower(stubTrust{})
	if err != nil { t.Fatal(err) }
	webKeys := Keys(web.c, web.a, webPlan, web.target, ref("builder-v1"), []Ref{ref("build.compiler")})

	desktop := orderFixture("tei.language.dsl")
	desktop.target.ID = "desktop"
	for i := range desktop.providers {
		desktop.providers[i].Target = "desktop"
		desktop.providers[i].Identity.ID += ".desktop"
		desktop.wiring[i].Provider = desktop.providers[i].Identity
	}
	desktopPlan, err := desktop.lower(stubTrust{})
	if err != nil { t.Fatal(err) }
	desktopKeys := Keys(desktop.c, desktop.a, desktopPlan, desktop.target, ref("builder-v1"), []Ref{ref("build.compiler")})
	if desktopKeys.Interpret != webKeys.Interpret || desktopKeys.Accept != webKeys.Accept {
		t.Fatal("Target-local Provider selection changed accepted semantics")
	}
	if desktopKeys.Resolve == webKeys.Resolve || desktopKeys.Target == webKeys.Target {
		t.Fatal("Target-specific Provider selection must invalidate exact Capability resolution")
	}
	// Composition resolution is intentionally target-dependent when Provider
	// choice changes. Reuse of Accepted meaning does not mean reuse of Lock.
}

func TestNeutralPlanDoesNotAliasMutableCandidate(t *testing.T) {
	f := orderFixture("tei.language.dsl")
	neutral, err := Neutralize(f.c, f.a, stubTrust{})
	if err != nil {
		t.Fatal(err)
	}
	before := Digest(neutral)
	f.c.Requirements[1].RequiredSlots[0].ID = "mutated.slot"
	if Digest(neutral) != before {
		t.Fatal("neutral plan shared mutable context slot backing array")
	}
}

func TestMalformedContextDeclarationRejected(t *testing.T) {
	f := orderFixture("tei.language.dsl")
	f.c.Requirements[1].RequiredSlots = append(f.c.Requirements[1].RequiredSlots, f.c.Requirements[1].RequiredSlots[0])
	f.a.CandidateDigest = Digest(f.c)
	_, err := Neutralize(f.c, f.a, stubTrust{})
	if got := code(err); got != "invalid-context-declaration" {
		t.Fatalf("got %s, want invalid-context-declaration", got)
	}
}

func TestS05PreviewCannotGrantExecutionAuthority(t *testing.T) {
	f := orderFixture("tei.language.dsl")
	plan, err := f.lower(stubTrust{})
	if err != nil {
		t.Fatal(err)
	}
	// Synthetic rehearsal: a preview is resolved, then one current condition
	// changes before invocation. None of the following reuses that preview.
	tests := []struct {
		name string
		trust stubTrust
		code string
	}{
		{"locked-resolution-revoked", stubTrust{denyResolution: true}, "stale-or-untrusted-resolution"},
		{"provider-unbound", stubTrust{denyRuntimeBinding: true}, "runtime-binding-unavailable"},
		{"schema-incompatible", stubTrust{denyData: true}, "data-contract-incompatible"},
		{"required-context-lost", stubTrust{denyContext: true}, "missing-execution-context"},
		{"grant-revoked", stubTrust{denyInvocation: true}, "authorization-denied"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := code(AuthorizeInvocation(plan, tc.trust)); got != tc.code {
				t.Fatalf("preview was used as authority: got %q, expected %q", got, tc.code)
			}
		})
	}
	// The plan is an immutable witness. Re-authorization does not edit it.
	pinned := Digest(plan)
	_ = AuthorizeInvocation(plan, stubTrust{denyInvocation: true})
	if Digest(plan) != pinned {
		t.Fatal("invocation changed resolved witness")
	}
}

func TestDataRequirementRetainsSourceMeaningAndIsImmutable(t *testing.T) {
	f := orderFixture("tei.language.dsl")
	neutral, err := Neutralize(f.c, f.a, stubTrust{})
	if err != nil {
		t.Fatal(err)
	}
	before := Digest(neutral)
	f.c.Requirements[1].DataContracts[0].VersionConstraint = "^99"
	if Digest(neutral) != before {
		t.Fatal("neutral data contract aliases authoring input")
	}
}

func TestInvalidDataRequirementsFailClosed(t *testing.T) {
	tests := []struct {
		name string
		edit func(*fixture)
		want string
	}{
		{"empty-constraint", func(f *fixture) { f.c.Requirements[0].DataContracts[0].VersionConstraint = "" }, "invalid-data-requirement"},
		{"missing-datasource", func(f *fixture) { f.c.Requirements[0].DataContracts[0].Datasource = "" }, "invalid-data-requirement"},
		{"unknown-constraint", func(f *fixture) { f.c.Requirements[0].DataContracts[0].VersionConstraint = "latest" }, "invalid-data-requirement"},
		{"zero-major", func(f *fixture) { f.c.Requirements[0].DataContracts[0].VersionConstraint = "^0" }, "invalid-data-requirement"},
		{"non-canonical-major", func(f *fixture) { f.c.Requirements[0].DataContracts[0].VersionConstraint = "^02" }, "invalid-data-requirement"},
		{"duplicate-contract", func(f *fixture) {
			f.c.Requirements[0].DataContracts = append(f.c.Requirements[0].DataContracts, f.c.Requirements[0].DataContracts[0])
		}, "duplicate-data-requirement"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			f := orderFixture("tei.language.dsl")
			tc.edit(&f)
			f.a.CandidateDigest = Digest(f.c)
			_, err := Neutralize(f.c, f.a, stubTrust{})
			if got := code(err); got != tc.want {
				t.Fatalf("got %q; expected %q", got, tc.want)
			}
		})
	}
}

func TestForgedPlanDoesNotBypassTrustedResolution(t *testing.T) {
	f := orderFixture("tei.language.dsl")
	p, err := f.lower(stubTrust{})
	if err != nil { t.Fatal(err) }
	trust := stubTrust{expectedPlan: Digest(p)}
	if err := AuthorizeInvocation(p, trust); err != nil { t.Fatal(err) }

	forged := p
	forged.Steps = append([]Step(nil), p.Steps...)
	forged.Steps[1].Effect = "write-all"
	if got := code(AuthorizeInvocation(forged, trust)); got != "stale-or-untrusted-resolution" {
		t.Fatalf("forged plan unexpectedly gained invoke authority: %s", got)
	}
}

func TestUndeclaredContextAndDataDoNotGate(t *testing.T) {
	f := orderFixture("tei.language.dsl")
	// Re-run the accepted boundary for a pure synthetic read operation that
	// declares neither a data contract nor invocation Context slots.
	f.c.Requirements = f.c.Requirements[:1]
	f.c.Requirements[0].DataContracts = nil
	f.wiring = f.wiring[:1]
	f.providers = f.providers[:1]
	f.a.CandidateDigest = Digest(f.c)
	plan, err := f.lower(stubTrust{})
	if err != nil { t.Fatal(err) }
	if len(plan.Steps[0].RequiredSlots) != 0 || len(plan.Steps[0].DataContracts) != 0 {
		t.Fatal("unexpected runtime requirements")
	}
	if err := AuthorizeInvocation(plan, stubTrust{denyContext: true, denyData: true}); err != nil {
		t.Fatalf("unrelated Host context/schema status invalidated unrelated operation: %v", err)
	}
}

func TestSourceDependenciesInvalidateInterpretation(t *testing.T) {
	f := orderFixture("tei.language.dsl")
	plan, err := f.lower(stubTrust{})
	if err != nil { t.Fatal(err) }
	before := Keys(f.c, f.a, plan, f.target, ref("builder-v1"), []Ref{ref("build.codegen-tool"), ref("build.compiler")})
	neutral, err := Neutralize(f.c, f.a, stubTrust{})
	if err != nil { t.Fatal(err) }
	origin := Digest(neutral.Origin)
	f.c.SourceInputs[1].Fingerprint = "sha256:changed-definition"
	if Digest(neutral.Origin) != origin {
		t.Fatal("accepted Origin aliases mutable source dependencies")
	}
	after := Keys(f.c, f.a, plan, f.target, ref("builder-v1"), []Ref{ref("build.codegen-tool"), ref("build.compiler")})
	if before.Interpret == after.Interpret || before.Accept == after.Accept {
		t.Fatal("dependent semantic Source did not invalidate interpretation and acceptance")
	}
	if got, err := Neutralize(f.c, f.a, stubTrust{}); code(err) != "unaccepted-meaning" {
		t.Fatalf("modified source entered an old accepted snapshot: %+v %v", got, err)
	}
}

func TestInvalidSourceDependencyWitnessFailsClosed(t *testing.T) {
	tests := []struct{
		name string
		edit func(*fixture)
		want string
	}{
		{"empty-closure", func(f *fixture) { f.c.SourceInputs = nil }, "missing-source-dependencies"},
		{"duplicate-dependency", func(f *fixture) {
			f.c.SourceInputs = append(f.c.SourceInputs, f.c.SourceInputs[0])
		}, "invalid-source-dependencies"},
		{"wrong-order", func(f *fixture) {
			f.c.SourceInputs[0], f.c.SourceInputs[1] = f.c.SourceInputs[1], f.c.SourceInputs[0]
		}, "invalid-source-dependencies"},
		{"unfingerprinted", func(f *fixture) {
			f.c.SourceInputs[0].Fingerprint = ""
		}, "invalid-source-dependencies"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			f := orderFixture("tei.language.dsl")
			tc.edit(&f)
			f.a.CandidateDigest = Digest(f.c)
			_, err := Neutralize(f.c, f.a, stubTrust{})
			if got := code(err); got != tc.want {
				t.Fatalf("got %q, expected %q", got, tc.want)
			}
		})
	}
}

func TestDeclaredLanguageWitnessSemanticEqualityIsNotAuthorityEquality(t *testing.T) {
	a := orderFixture("tei.language.dsl")
	b := orderFixture("tei.language.sql-projection")
	left, err := Neutralize(a.c, a.a, stubTrust{})
	if err != nil { t.Fatal(err) }
	right, err := Neutralize(b.c, b.a, stubTrust{})
	if err != nil { t.Fatal(err) }
	if Digest(left.Intents) != Digest(right.Intents) || left.Snapshot != right.Snapshot {
		t.Fatal("synthetic producers changed the accepted semantic intent")
	}
	if left.Origin.CandidateDigest == right.Origin.CandidateDigest ||
		left.Origin.Language == right.Origin.Language {
		t.Fatal("distinct Language provenance was collapsed into a shared Acceptance identity")
	}
	// These are declared Language identities, not two real Parser executions.
}

func TestStructurallyForgedPlansFailBeforeTrustedDispatch(t *testing.T) {
	tests := []struct {
		name string
		edit func(*Plan)
		want string
	}{
		{"unfingerprinted-contract", func(p *Plan) { p.Steps[0].Contract.Fingerprint = "" }, "invalid-plan-step"},
		{"mixed-targets", func(p *Plan) { p.Steps[1].Target = "desktop" }, "invalid-plan-step"},
		{"duplicate-capability", func(p *Plan) { p.Steps[1] = p.Steps[0] }, "duplicate-capability"},
		{"invalid-runtime-dependency", func(p *Plan) { p.Steps[0].RuntimeDeps = []Ref{{}} }, "invalid-runtime-dependency"},
		{"erased-required-context", func(p *Plan) { p.Steps[1].RequiredSlots = []Ref{{}} }, "invalid-context-declaration"},
		{"corrupt-data-contract", func(p *Plan) { p.Steps[1].DataContracts = []DataNeed{{}} }, "invalid-data-requirement"},
		{"duplicate-data-contract", func(p *Plan) {
			p.Steps[1].DataContracts = append(p.Steps[1].DataContracts, p.Steps[1].DataContracts[0])
		}, "duplicate-data-requirement"},
		{"unfingerprinted-source", func(p *Plan) { p.Origin.SourceInputs[0].Fingerprint = "" }, "invalid-source-dependencies"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			f := orderFixture("tei.language.dsl")
			plan, err := f.lower(stubTrust{})
			if err != nil {
				t.Fatal(err)
			}
			tc.edit(&plan)
			if got := code(AuthorizeInvocation(plan, stubTrust{})); got != tc.want {
				t.Fatalf("Host callback allowed a malformed witness: got %q, expected %q", got, tc.want)
			}
		})
	}
}

func TestRuntimeDependencyOrderDoesNotChangeResolvedPlan(t *testing.T) {
	f := orderFixture("tei.language.dsl")
	f.providers[0].RuntimeDeps = []Ref{ref("runtime.a"), ref("runtime.b")}
	first, err := f.lower(stubTrust{})
	if err != nil { t.Fatal(err) }

	reordered := orderFixture("tei.language.dsl")
	reordered.providers[0].RuntimeDeps = []Ref{ref("runtime.b"), ref("runtime.a")}
	second, err := reordered.lower(stubTrust{})
	if err != nil { t.Fatal(err) }
	if Digest(first) != Digest(second) {
		t.Fatal("runtime dependency registration order changed the target-bound witness")
	}
}
