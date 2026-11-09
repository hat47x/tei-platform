package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

const h = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
const m = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"

func fixture() Packet {
	return Packet{
		Schema: schema, Claim: "E2",
		Contract:      Contract{ProducerPath: "tools/reference-shared-acceptance/practical_validation_loop_test.go", ConsumerPath: "tools/reference-first-web-target/practical_business_rule_test.go", PositiveTest: "TestPracticalRuleChangePreviewsAndAcceptsBothMeaningAndApplicationSelection", NegativeTest: "TestReferenceCommandScenarioRejectsMissingValuesAcrossAllLines", EvidenceClass: "actual-observed"},
		Qualification: Qualification{Head: h, Main: m, FullHealth: "PASS", CleanBefore: true, CleanAfter: true, Receipt: "local/health.log"},
		Acceptance:    Acceptance{R0: "sha256:" + strings.Repeat("0", 64), R1: "sha256:" + strings.Repeat("1", 64), CandidateFingerprint: "sha256:" + strings.Repeat("2", 64), ReviewFingerprint: "sha256:" + strings.Repeat("3", 64), ReviewCandidateFingerprint: "sha256:" + strings.Repeat("2", 64), DecisionReceipt: "local/accept.json", ReloadReceipt: "local/reload.json", Delta: []string{"add-application-validation-use", "add-validation"}},
		Web:           Web{AcceptedRevision: "sha256:" + strings.Repeat("1", 64), Application: "order_entry", Target: "web", SelectedValidation: "sales.order_line.unit_price_positive", NegativeValidation: "sales.order_line.unit_price_positive", NegativeEvidenceMode: "running-server", ProviderKind: "actual-db", RunMode: "running-server", WorkspaceRevisionReceipt: "local/r1-web-workspace.json", TargetLaunchReceipt: "local/target-launch.json", PositiveReceipt: "local/positive.json", NegativeReceipt: "local/negative.json", UnobservedControlReceipt: "local/n11.json"},
		Physical:      Physical{AcceptedRevision: "sha256:" + strings.Repeat("1", 64), Datasource: "db-orders", SchemaCompatibility: "compatible", SchemaEvidence: "local/schema.json", HostAuthorityEvidence: "local/host.json", HostAuthorityProvenance: "real-pg", Guarantee: &Guarantee{Level: "atomic-with-commit", Reason: "declared by the Provider"}, CommandReceipt: "local/write.json", IndependentReadReceipt: "local/read.json", RecordID: "o-1", SchemaFailureControl: "local/s02.json", HostFailureControl: "local/s05.json"},
	}
}
func has(r Report, s string) bool {
	for _, d := range r.Diagnostics {
		if d == s {
			return true
		}
	}
	return false
}

// reviewableExceptKind reports whether an E2 packet is complete in every
// respect except that it names no acceptanceKind. No path issues a formal
// Acceptance yet, so an E2 packet can no longer reach REVIEW_REQUIRED: it is
// either missing the kind (E2_ACCEPTANCE_KIND_REQUIRED) or declares a kind that
// is provisional/unspecified/unrecognized. Tests about other diagnostics use this
// to assert that nothing else is wrong.
func reviewableExceptKind(r Report) bool {
	return r.Status == "INCOMPLETE" && r.ProductGate == "UNQUALIFIED" &&
		len(r.Diagnostics) == 1 && r.Diagnostics[0] == "E2_ACCEPTANCE_KIND_REQUIRED"
}
func TestCompletePacketNeverGrantsProductGate(t *testing.T) {
	// E1 keeps the legacy shape and still reaches review.
	p := fixture()
	p.Claim = "E1"
	if result := Audit(p, h, m); result.Status != "REVIEW_REQUIRED" || result.ProductGate != "UNQUALIFIED" || len(result.Diagnostics) > 0 {
		t.Fatalf("%+v", result)
	}
	// E2 without acceptanceKind is incomplete (D9 follow-up).
	if result := Audit(fixture(), h, m); !reviewableExceptKind(result) {
		t.Fatalf("%+v", result)
	}
}
func TestProvisionalGrantProvenanceIsNeverG2Evidence(t *testing.T) {
	for _, label := range []string{"provisional-grant", " Provisional-Grant\t"} {
		for _, claim := range []string{"E0", "E1", "E2"} {
			p := fixture()
			p.Claim = claim
			p.Physical.HostAuthorityProvenance = label
			r := Audit(p, h, m)
			if !has(r, "PROVISIONAL_GRANT_NOT_G2_EVIDENCE") || r.Status != "INCOMPLETE" || r.ProductGate != "UNQUALIFIED" {
				t.Fatalf("%s %q: %+v", claim, label, r)
			}
		}
	}
	// Other or absent labels are unchanged: the complete fixture still reaches review.
	p := fixture()
	p.Physical.HostAuthorityProvenance = "real-pg"
	if r := Audit(p, h, m); !reviewableExceptKind(r) || has(r, "PROVISIONAL_GRANT_NOT_G2_EVIDENCE") {
		t.Fatalf("%+v", r)
	}
	// The label is accepted by the strict packet decoder.
	raw, _ := json.Marshal(func() Packet { p := fixture(); p.Physical.HostAuthorityProvenance = "provisional-grant"; return p }())
	path := filepath.Join(t.TempDir(), "p.json")
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := run([]string{"-packet", path, "-head", h, "-main", m}, &out); err == nil || !strings.Contains(out.String(), "PROVISIONAL_GRANT_NOT_G2_EVIDENCE") {
		t.Fatalf("err=%v out=%s", err, out.String())
	}
}

func TestE2MustDeclareAKnownHostAuthorityProvenance(t *testing.T) {
	const wantProvisional, wantRequired = "PROVISIONAL_GRANT_NOT_G2_EVIDENCE", "E2_HOST_AUTHORITY_PROVENANCE_REQUIRED"
	zeroWidth, bidi, bom, nl := string(rune(0x200b)), string(rune(0x202e)), string(rune(0xfeff)), string(rune(10))
	for label, want := range map[string]string{
		"":                                   wantRequired, // omitted
		"provisional_grant":                  wantProvisional,
		"Provisional-Grant":                  wantProvisional,
		"PROVISIONAL GRANT":                  wantProvisional,
		"provisional-grant" + zeroWidth:      wantProvisional,
		"prov" + zeroWidth + "isional-grant": wantProvisional,
		"provisional" + bidi + "-grant":      wantProvisional,
		bom + "provisional-grant":            wantProvisional,
		"provisional-grant" + nl:             wantProvisional,
		"Real-PG":                            wantRequired, // case matters for an allowed label
		"real_pg":                            wantRequired,
		"real-pg ":                           wantRequired,
		"real" + zeroWidth + "-pg":           wantRequired,
		"unspecified":                        wantRequired,
		"host-authority":                     wantRequired,
		"some-new-provisionalish-ok":         wantRequired,
	} {
		p := fixture()
		p.Physical.HostAuthorityProvenance = label
		r := Audit(p, h, m)
		if !has(r, want) || r.Status != "INCOMPLETE" || r.ProductGate != "UNQUALIFIED" {
			t.Fatalf("%q: want %s, got %+v", label, want, r)
		}
		if want == wantProvisional && has(r, wantRequired) {
			t.Fatalf("%q: provisional must not also be reported as merely unknown: %+v", label, r)
		}
	}
	for _, label := range []string{"real-pg", "synthetic"} {
		p := fixture()
		p.Physical.HostAuthorityProvenance = label
		if r := Audit(p, h, m); !reviewableExceptKind(r) {
			t.Fatalf("%q: %+v", label, r)
		}
	}
	// The requirement is for E2 only; E1 packets do not carry Physical evidence.
	p := fixture()
	p.Claim = "E1"
	p.Web.ProviderKind = "synthetic"
	p.Physical = Physical{}
	if r := Audit(p, h, m); r.Status != "REVIEW_REQUIRED" || has(r, "E2_HOST_AUTHORITY_PROVENANCE_REQUIRED") {
		t.Fatalf("%+v", r)
	}
}

// Documents a known limit: a look-alike label that uses a different script
// (here Cyrillic o) is NOT folded to provisional-grant. E2 refuses it as an
// unknown label; E0/E1 do not read Physical evidence, so nothing is reported.
// The audit is not a defence against a packet author who disguises the label.
func TestHomoglyphProvisionalLabelIsOnlyCaughtByTheE2AllowList(t *testing.T) {
	label := "pr" + string(rune(0x043e)) + "visional-grant"
	for claim, wantRequired := range map[string]bool{"E2": true, "E1": false, "E0": false} {
		p := fixture()
		p.Claim = claim
		p.Physical.HostAuthorityProvenance = label
		r := Audit(p, h, m)
		if has(r, "PROVISIONAL_GRANT_NOT_G2_EVIDENCE") {
			t.Fatalf("%s: a different-script look-alike must not be folded: %+v", claim, r)
		}
		if has(r, "E2_HOST_AUTHORITY_PROVENANCE_REQUIRED") != wantRequired {
			t.Fatalf("%s: E2_HOST_AUTHORITY_PROVENANCE_REQUIRED=%v expected, got %+v", claim, wantRequired, r)
		}
	}
}

func TestReviewCandidateBindingMustMatchAcceptedCandidate(t *testing.T) {
	p := fixture()
	p.Claim = "E1"
	p.Acceptance.ReviewCandidateFingerprint = "sha256:" + strings.Repeat("f", 64)
	r := Audit(p, h, m)
	if !has(r, "ACCEPTANCE_REVIEW_CANDIDATE_BINDING_MISSING_OR_MISMATCH") ||
		r.Status != "INCOMPLETE" {
		t.Fatalf("review identity did not match the accepted candidate: %+v", r)
	}
}

func TestMissingReviewCandidateBindingCannotPassPreflight(t *testing.T) {
	p := fixture()
	p.Claim = "E1"
	p.Acceptance.ReviewCandidateFingerprint = ""
	r := Audit(p, h, m)
	if !has(r, "ACCEPTANCE_REVIEW_CANDIDATE_BINDING_MISSING_OR_MISMATCH") ||
		r.ProductGate != "UNQUALIFIED" {
		t.Fatalf("review-to-candidate identity was not required: %+v", r)
	}
}

func TestDistinctReviewAndCandidateDigestsAreValid(t *testing.T) {
	p := fixture()
	p.Claim = "E1"
	p.Web.ProviderKind = "synthetic"
	p.Physical = Physical{}
	if p.Acceptance.CandidateFingerprint == p.Acceptance.ReviewFingerprint {
		t.Fatal("test fixture wrongly reuses candidate digest as review digest")
	}
	r := Audit(p, h, m)
	if r.Status != "REVIEW_REQUIRED" || r.ProductGate != "UNQUALIFIED" {
		t.Fatalf("legitimate distinct candidate and review digests were rejected: %+v", r)
	}
}

func TestReviewDigestCannotSubstituteCandidateLink(t *testing.T) {
	p := fixture()
	p.Claim = "E1"
	p.Acceptance.ReviewCandidateFingerprint = p.Acceptance.ReviewFingerprint
	r := Audit(p, h, m)
	if !has(r, "ACCEPTANCE_REVIEW_CANDIDATE_BINDING_MISSING_OR_MISMATCH") {
		t.Fatalf("review digest was confused with candidate identity: %+v", r)
	}
}

func TestSyntheticWebCannotClaimE2(t *testing.T) {
	p := fixture()
	p.Web.ProviderKind = "synthetic"
	result := Audit(p, h, m)
	want := []string{"E2_ACCEPTANCE_KIND_REQUIRED", "SYNTHETIC_WEB_PROVIDER_CANNOT_CLAIM_E2"}
	if !reflect.DeepEqual(result.Diagnostics, want) || result.Status != "INCOMPLETE" {
		t.Fatalf("%+v", result)
	}
}

// An empty or unknown providerKind names no Provider at all. It is reported
// as unresolved, and must not also be labeled "synthetic": that diagnosis is
// false when the packet never declared a synthetic Provider. Either way the
// packet stays INCOMPLETE.
func TestUnresolvedWebProviderKindIsNotLabeledSynthetic(t *testing.T) {
	for _, kind := range []string{"", "synthetic-db", "odbc", " actual-db "} {
		p := fixture()
		p.Web.ProviderKind = kind
		result := Audit(p, h, m)
		if has(result, "SYNTHETIC_WEB_PROVIDER_CANNOT_CLAIM_E2") ||
			!has(result, "WEB_PROVIDER_KIND_UNRESOLVED") || result.Status != "INCOMPLETE" {
			t.Fatalf("%q: %+v", kind, result)
		}
	}
}
func TestE1WithSyntheticWebIsOnlyReviewable(t *testing.T) {
	p := fixture()
	p.Claim = "E1"
	p.Web.ProviderKind = "synthetic"
	p.Physical = Physical{}
	result := Audit(p, h, m)
	if result.Status != "REVIEW_REQUIRED" || result.ProductGate != "UNQUALIFIED" {
		t.Fatalf("%+v", result)
	}
}
func TestRuleWithoutExplicitApplicationUseIsRejected(t *testing.T) {
	p := fixture()
	p.Claim = "E1"
	p.Acceptance.Delta = []string{"add-validation"}
	if !has(Audit(p, h, m), "VALIDATION_AND_APPLICATION_USE_NOT_ATOMIC") {
		t.Fatal("missing atomic-use diagnostic")
	}
}
func TestRevisionAndStaleQualificationFail(t *testing.T) {
	p := fixture()
	p.Acceptance.R1 = p.Acceptance.R0
	r := Audit(p, h, "cccccccccccccccccccccccccccccccccccccccc")
	if !has(r, "REVISION_QUALIFICATION_STALE_OR_MISSING") || !has(r, "ACCEPTED_R0_R1_LINEAGE_MISSING") || !has(r, "WEB_NOT_BOUND_TO_ACCEPTED_R1_RULE_SELECTION") {
		t.Fatalf("%+v", r)
	}
}
func TestMissingReadAndNegativeControlsFail(t *testing.T) {
	p := fixture()
	p.Physical.IndependentReadReceipt = p.Physical.CommandReceipt
	p.Physical.SchemaFailureControl = ""
	p.Physical.HostFailureControl = ""
	p.Web.UnobservedControlReceipt = ""
	r := Audit(p, h, m)
	for _, c := range []string{"E2_COMMAND_AND_INDEPENDENT_READ_NOT_CORRELATED", "S02_NEGATIVE_CONTROL_MISSING", "S05_NEGATIVE_CONTROL_MISSING", "N11_UNOBSERVED_CONTROL_MISSING"} {
		if !has(r, c) {
			t.Fatalf("missing %s in %+v", c, r)
		}
	}
}

// N11 is an explicit *unobserved* read-control result. Reusing the
// target-launch or workspace evidence does not establish that control.
func TestN11ControlMustBeSeparateFromOtherWebEvidence(t *testing.T) {
	cases := map[string]func(*Packet){
		"blank control": func(p *Packet) {
			p.Web.UnobservedControlReceipt = "  "
		},
		"reused workspace revision": func(p *Packet) {
			p.Web.UnobservedControlReceipt = p.Web.WorkspaceRevisionReceipt
		},
		"reused target launch": func(p *Packet) {
			p.Web.UnobservedControlReceipt = p.Web.TargetLaunchReceipt
		},
		"reused decision": func(p *Packet) {
			p.Web.UnobservedControlReceipt = p.Acceptance.DecisionReceipt
		},
		"reused post-accept reload": func(p *Packet) {
			p.Web.UnobservedControlReceipt = p.Acceptance.ReloadReceipt
		},
		"reused repository health": func(p *Packet) {
			p.Web.UnobservedControlReceipt = p.Qualification.Receipt
		},
		"reused positive response": func(p *Packet) {
			p.Web.UnobservedControlReceipt = p.Web.PositiveReceipt
		},
		"reused negative response": func(p *Packet) {
			p.Web.UnobservedControlReceipt = p.Web.NegativeReceipt
		},
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			p := fixture()
			p.Claim = "E1"
			mutate(&p)
			r := Audit(p, h, m)
			if !has(r, "N11_UNOBSERVED_CONTROL_MISSING") ||
				r.Status != "INCOMPLETE" || r.ProductGate != "UNQUALIFIED" {
				t.Fatalf("N11 was satisfied by unrelated evidence: %+v", r)
			}
		})
	}
}

func TestE2N11MustNotRecycleCommandOrIndependentReadEvidence(t *testing.T) {
	for name, mutate := range map[string]func(*Packet){
		"command reused": func(p *Packet) {
			p.Web.UnobservedControlReceipt = p.Physical.CommandReceipt
		},
		"independent read reused": func(p *Packet) {
			p.Web.UnobservedControlReceipt = p.Physical.IndependentReadReceipt
		},
	} {
		t.Run(name, func(t *testing.T) {
			p := fixture()
			mutate(&p)
			r := Audit(p, h, m)
			if !has(r, "N11_UNOBSERVED_CONTROL_MISSING") ||
				r.Status != "INCOMPLETE" || r.ProductGate != "UNQUALIFIED" {
				t.Fatalf("E2 used an unrelated Command/Observation as N11: %+v", r)
			}
		})
	}
}

func TestE2RejectsWhitespaceOnlyCommandOrReadReceipt(t *testing.T) {
	for name, mutate := range map[string]func(*Packet){
		"command": func(p *Packet) { p.Physical.CommandReceipt = "  " },
		"read":    func(p *Packet) { p.Physical.IndependentReadReceipt = "  " },
	} {
		t.Run(name, func(t *testing.T) {
			p := fixture()
			mutate(&p)
			if !has(Audit(p, h, m), "E2_COMMAND_AND_INDEPENDENT_READ_NOT_CORRELATED") {
				t.Fatal("E2 accepted a whitespace-only evidence receipt")
			}
		})
	}
}

func TestN11DiagnosticAppearsOnlyOnceForMultipleMissingStages(t *testing.T) {
	p := fixture()
	p.Web.UnobservedControlReceipt = ""
	p.Physical.CommandReceipt = ""
	p.Physical.IndependentReadReceipt = ""
	r := Audit(p, h, m)
	count := 0
	for _, code := range r.Diagnostics {
		if code == "N11_UNOBSERVED_CONTROL_MISSING" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("one missing N11 control yielded %d diagnostics: %+v", count, r)
	}
}

func TestEmptyWebReceiptsDoNotPassE1(t *testing.T) {
	for name, change := range map[string]func(*Packet){
		"blank positive": func(p *Packet) { p.Web.PositiveReceipt = "  " },
		"blank negative": func(p *Packet) { p.Web.NegativeReceipt = "  " },
	} {
		t.Run(name, func(t *testing.T) {
			p := fixture()
			p.Claim = "E1"
			change(&p)
			if !has(Audit(p, h, m), "ACTUAL_WEB_POSITIVE_AND_NEGATIVE_RECEIPTS_REQUIRED") {
				t.Fatal("blank Web receipt was accepted")
			}
		})
	}
}

func TestHealthNotRunCannotQualify(t *testing.T) {
	p := fixture()
	p.Qualification.FullHealth = "NOT_RUN"
	p.Qualification.CleanAfter = false
	if !has(Audit(p, h, m), "TRUSTED_FULL_HEALTH_NOT_RECORDED") {
		t.Fatal("lost trusted gate")
	}
}
func TestE0RejectsEmptyProducerConsumerEvidence(t *testing.T) {
	p := fixture()
	p.Claim = "E0"
	p.Contract = Contract{}
	if !has(Audit(p, h, m), "E0_CONTRACT_OR_POSITIVE_NEGATIVE_TEST_MISSING") {
		t.Fatal("empty E0 evidence can pass preflight")
	}
}
func TestE1HandlerOnlyUnitPriceNegativeIsNotLiveEvidence(t *testing.T) {
	p := fixture()
	p.Claim = "E1"
	// Existing unit-price rejection is a Go HTTP handler test; it does not
	// exercise a separately running Web Target with an invalid unit price.
	p.Web.NegativeEvidenceMode = "handler-only"
	r := Audit(p, h, m)
	if !has(r, "E1_NEGATIVE_SELECTED_RULE_RUNNING_WEB_EVIDENCE_REQUIRED") ||
		r.Status != "INCOMPLETE" || r.ProductGate != "UNQUALIFIED" {
		t.Fatalf("%+v", r)
	}
}

func TestE1OtherValidationCannotStandInForUnitPriceNegative(t *testing.T) {
	p := fixture()
	p.Claim = "E1"
	// The default Web fixture's rejected order has invalid quantity,
	// which does not demonstrate the newly accepted unit-price guard.
	p.Web.NegativeValidation = "sales.order_line.quantity_positive"
	r := Audit(p, h, m)
	if !has(r, "E1_NEGATIVE_SELECTED_RULE_RUNNING_WEB_EVIDENCE_REQUIRED") ||
		r.Status != "INCOMPLETE" {
		t.Fatalf("%+v", r)
	}
}

func TestE1RejectsUnboundWebWorkspaceRevision(t *testing.T) {
	for name, edit := range map[string]func(*Packet){
		"missing workspace revision":    func(p *Packet) { p.Web.WorkspaceRevisionReceipt = "" },
		"missing target launch":         func(p *Packet) { p.Web.TargetLaunchReceipt = "" },
		"reused positive result":        func(p *Packet) { p.Web.WorkspaceRevisionReceipt = p.Web.PositiveReceipt },
		"same evidence for both stages": func(p *Packet) { p.Web.TargetLaunchReceipt = p.Web.WorkspaceRevisionReceipt },
	} {
		t.Run(name, func(t *testing.T) {
			p := fixture()
			p.Claim = "E1"
			edit(&p)
			r := Audit(p, h, m)
			if !has(r, "E1_WEB_WORKSPACE_R1_AND_TARGET_LAUNCH_PROVENANCE_MISSING") ||
				r.Status != "INCOMPLETE" || r.ProductGate != "UNQUALIFIED" {
				t.Fatalf("%+v", r)
			}
		})
	}
}

func TestHandlerOnlyIsNotARealWebRun(t *testing.T) {
	p := fixture()
	p.Claim = "E1"
	p.Web.RunMode = "handler-only"
	if !has(Audit(p, h, m), "E1_RUNNING_WEB_TARGET_RECEIPT_REQUIRED") {
		t.Fatal("handler-only test promoted to E1")
	}
}

func TestStrictInputRejectsUnknownAndTrailing(t *testing.T) {
	p := fixture()
	dir := t.TempDir()
	path := filepath.Join(dir, "packet.json")
	b, _ := json.Marshal(p)
	if err := os.WriteFile(path, append(b, []byte(`{"schema":"duplicate"}`)...), 0600); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{"-packet", path, "-head", h, "-main", m}, new(bytes.Buffer)); err == nil {
		t.Fatal("trailing values accepted")
	}
	var obj map[string]any
	if err := json.Unmarshal(b, &obj); err != nil {
		t.Fatal(err)
	}
	obj["extra"] = "not authorized"
	b, _ = json.Marshal(obj)
	if err := os.WriteFile(path, b, 0600); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{"-packet", path, "-head", h, "-main", m}, new(bytes.Buffer)); err == nil {
		t.Fatal("unknown input accepted")
	}
}
func TestDeterministicDiagnosticsOrder(t *testing.T) {
	p := fixture()
	p.Claim = "E2"
	p.Web.ProviderKind = "synthetic"
	p.Physical.SchemaCompatibility = "unresolved"
	p.Qualification.FullHealth = "NOT_RUN"
	a := Audit(p, h, m)
	b := Audit(p, h, m)
	if !reflect.DeepEqual(a, b) {
		t.Fatal("unstable audit")
	}
	if a.Status != "INCOMPLETE" || a.ProductGate != "UNQUALIFIED" {
		t.Fatalf("%+v", a)
	}
}

func TestClaimedE1E2RejectsSourceOnlyEvidenceClass(t *testing.T) {
	for _, tier := range []string{"E1", "E2"} {
		p := fixture()
		p.Claim = tier
		for _, class := range []string{"source-only", "synthetic-tested"} {
			p.Contract.EvidenceClass = class
			r := Audit(p, h, m)
			if r.Status != "INCOMPLETE" || !has(r, "E1_E2_ACTUAL_EXECUTION_CLASS_REQUIRED") {
				t.Fatalf("%s %s was promoted: %+v", tier, class, r)
			}
		}
	}
	p := fixture()
	p.Claim = "E0"
	p.Contract.EvidenceClass = "source-only"
	if r := Audit(p, h, m); r.Status != "REVIEW_REQUIRED" || r.ProductGate != "UNQUALIFIED" {
		t.Fatalf("E0 should permit source-only, but never qualify Product Gate: %+v", r)
	}
}

// E1/E2 require actual-observed. An empty or unknown class is the same class
// defect as source-only for those claims: one diagnosis, not also the E0
// "contract or test missing" report. E0 keeps the class-validity diagnosis,
// and E1/E2 still report genuinely missing contract fields.
func TestClaimedE1E2EmptyEvidenceClassHasOneDiagnosis(t *testing.T) {
	for _, tier := range []string{"E1", "E2"} {
		for _, class := range []string{"", "bogus"} {
			p := fixture()
			p.Claim = tier
			p.Contract.EvidenceClass = class
			r := Audit(p, h, m)
			if !has(r, "E1_E2_ACTUAL_EXECUTION_CLASS_REQUIRED") ||
				has(r, "E0_CONTRACT_OR_POSITIVE_NEGATIVE_TEST_MISSING") ||
				r.Status != "INCOMPLETE" {
				t.Fatalf("%s %q: %+v", tier, class, r)
			}
		}
	}
	p := fixture()
	p.Claim = "E0"
	p.Contract.EvidenceClass = ""
	if r := Audit(p, h, m); !has(r, "E0_CONTRACT_OR_POSITIVE_NEGATIVE_TEST_MISSING") {
		t.Fatalf("E0 lost the invalid-class diagnosis: %+v", r)
	}
	p = fixture()
	p.Claim = "E1"
	p.Contract.ProducerPath = ""
	if r := Audit(p, h, m); !has(r, "E0_CONTRACT_OR_POSITIVE_NEGATIVE_TEST_MISSING") {
		t.Fatalf("E1 lost the missing-contract-field diagnosis: %+v", r)
	}
}

func TestStrictInputRejectsDuplicateKeysAndInvalidUTF8(t *testing.T) {
	p := fixture()
	b, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "packet.json")
	test := func(label string, data []byte) {
		t.Helper()
		if err := os.WriteFile(path, data, 0600); err != nil {
			t.Fatal(err)
		}
		if err := run([]string{"-packet", path, "-head", h, "-main", m}, new(bytes.Buffer)); err == nil {
			t.Fatal(label, "accepted")
		}
	}
	for label, data := range map[string][]byte{
		"top-level duplicate": bytes.Replace(b, []byte(`"claim":"E2"`), []byte(`"claim":"E0","claim":"E2"`), 1),
		"case-folded alias":   bytes.Replace(b, []byte(`"claim":"E2"`), []byte(`"claim":"E0","Claim":"E2"`), 1),
		"nested duplicate":    bytes.Replace(b, []byte(`"runMode":"running-server"`), []byte(`"runMode":"handler-only","runMode":"running-server"`), 1),
		"invalid UTF-8":       append(append([]byte{}, b...), 0xff),
	} {
		test(label, data)
	}
}

func TestStrictInputCannotHideTrailingBytesAfterOneMiBLimit(t *testing.T) {
	e1 := fixture()
	e1.Claim = "E1" // no E2 packet can reach REVIEW_REQUIRED without a formal Acceptance kind
	b, err := json.Marshal(e1)
	if err != nil {
		t.Fatal(err)
	}
	if len(b) >= (1 << 20) {
		t.Fatal("fixture unexpectedly large")
	}
	data := append(b, bytes.Repeat([]byte(" "), (1<<20)-len(b))...)
	data = append(data, []byte(`{"claim":"E0"}`)...)
	path := filepath.Join(t.TempDir(), "packet.json")
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{"-packet", path, "-head", h, "-main", m}, new(bytes.Buffer)); err == nil {
		t.Fatal("over-limit packet was silently truncated at artificial EOF")
	}
}

func TestJSONDepthBoundRejectsPathologicalInput(t *testing.T) {
	data := []byte(strings.Repeat("[", 80) + "0" + strings.Repeat("]", 80))
	if err := rejectDuplicateJSONKeys(data); err == nil {
		t.Fatal("pathological nesting was accepted")
	}
}

func TestCLIReviewRequiredRemainsUnqualified(t *testing.T) {
	e1 := fixture()
	e1.Claim = "E1" // no E2 packet can reach REVIEW_REQUIRED without a formal Acceptance kind
	b, err := json.Marshal(e1)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "packet.json")
	if err := os.WriteFile(path, b, 0600); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := run([]string{"-packet", path, "-head", h, "-main", m}, &out); err != nil {
		t.Fatalf("reviewable packet should emit a report: %v", err)
	}
	var result Report
	if err := json.Unmarshal(out.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Status != "REVIEW_REQUIRED" || result.ProductGate != "UNQUALIFIED" {
		t.Fatalf("CLI accidentally granted Product Gate: %+v", result)
	}
}

func TestCLIIncompleteEmitsDiagnosticAndFails(t *testing.T) {
	p := fixture()
	p.Claim = "E1"
	p.Contract.EvidenceClass = "source-only"
	p.Qualification.FullHealth = "NOT_RUN"
	p.Web.RunMode = "handler-only"
	b, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "packet.json")
	if err := os.WriteFile(path, b, 0600); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := run([]string{"-packet", path, "-head", h, "-main", m}, &out); err == nil {
		t.Fatal("incomplete claim did not fail")
	}
	var result Report
	if err := json.Unmarshal(out.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Status != "INCOMPLETE" || result.ProductGate != "UNQUALIFIED" ||
		!has(result, "E1_E2_ACTUAL_EXECUTION_CLASS_REQUIRED") ||
		!has(result, "E1_RUNNING_WEB_TARGET_RECEIPT_REQUIRED") ||
		!has(result, "TRUSTED_FULL_HEALTH_NOT_RECORDED") {
		t.Fatalf("CLI omitted essential incompleteness: %+v", result)
	}
}

func TestCurrentP0FixtureCannotBeSubmittedAsEvidence(t *testing.T) {
	data, err := os.ReadFile("testdata/p0-order-unit-price-incomplete.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := rejectDuplicateJSONKeys(data); err != nil {
		t.Fatal(err)
	}
	var p Packet
	if err := json.Unmarshal(data, &p); err != nil {
		t.Fatal(err)
	}
	result := Audit(p, h, m)
	if result.Status != "INCOMPLETE" || result.ProductGate != "UNQUALIFIED" {
		t.Fatalf("existing source-based tests were promoted to actual E1: %+v", result)
	}
	for _, code := range []string{
		"E1_E2_ACTUAL_EXECUTION_CLASS_REQUIRED",
		"ACCEPTED_R0_R1_LINEAGE_MISSING",
		"E1_RUNNING_WEB_TARGET_RECEIPT_REQUIRED",
		"ACCEPTANCE_REVIEW_CANDIDATE_BINDING_MISSING_OR_MISMATCH",
		"E1_WEB_WORKSPACE_R1_AND_TARGET_LAUNCH_PROVENANCE_MISSING",
		"TRUSTED_FULL_HEALTH_NOT_RECORDED",
	} {
		if !has(result, code) {
			t.Fatalf("P0 coverage fixture lost expected gap %q: %+v", code, result)
		}
	}
}

// The preflight receives untrusted packets. Keep the duplicate-key scan linear
// in the number of distinct keys (the input is also bounded to 1 MiB by run).
func TestStrictJSONManyDistinctKeysWithoutQuadraticScan(t *testing.T) {
	var doc strings.Builder
	doc.WriteByte('{')
	for i := 0; i < 10000; i++ {
		if i != 0 {
			doc.WriteByte(',')
		}
		doc.WriteString(strconv.Quote("field" + strconv.Itoa(i)))
		doc.WriteString(":null")
	}
	doc.WriteByte('}')
	if err := rejectDuplicateJSONKeys([]byte(doc.String())); err != nil {
		t.Fatalf("distinct keys rejected: %v", err)
	}
}

func TestStrictJSONUnicodeFoldedAliasRejected(t *testing.T) {
	if !strings.EqualFold("Key", "key") {
		t.Fatal("unexpected Go unicode folding semantics")
	}
	if err := rejectDuplicateJSONKeys([]byte(`{"Key":1,"key":2}`)); err == nil {
		t.Fatal("Unicode folded key shadowing was accepted")
	}
}

func TestProvisionalAcceptanceIsNeverFormalEvidence(t *testing.T) {
	zeroWidth, bidi, nl := string(rune(0x200b)), string(rune(0x202e)), string(rune(10))
	for _, label := range []string{"provisional", "Provisional", " PROVISIONAL" + "	", "prov" + zeroWidth + "isional", "provisional" + bidi, "unspecified", "Unspecified" + nl, "UNSPECIFIED"} {
		for _, claim := range []string{"E0", "E1", "E2"} {
			p := fixture()
			p.Claim = claim
			p.Acceptance.AcceptanceKind = label
			r := Audit(p, h, m)
			if !has(r, "PROVISIONAL_ACCEPTANCE_NOT_FORMAL_EVIDENCE") || r.Status != "INCOMPLETE" || r.ProductGate != "UNQUALIFIED" {
				t.Fatalf("%s %q: %+v", claim, label, r)
			}
		}
	}
	for _, label := range []string{"formal", "Formal", "formal ", "provisional2", "x"} {
		p := fixture()
		p.Acceptance.AcceptanceKind = label
		if r := Audit(p, h, m); !has(r, "ACCEPTANCE_KIND_UNRECOGNIZED") || r.Status != "INCOMPLETE" {
			t.Fatalf("%q: %+v", label, r)
		}
	}
	// Omitted (legacy packet): E0/E1 stay reviewable; E2 is incomplete.
	for _, claim := range []string{"E0", "E1"} {
		p := fixture()
		p.Claim = claim
		if r := Audit(p, h, m); r.Status != "REVIEW_REQUIRED" || r.ProductGate != "UNQUALIFIED" || has(r, "E2_ACCEPTANCE_KIND_REQUIRED") {
			t.Fatalf("%s: %+v", claim, r)
		}
	}
	// The strict packet decoder accepts the field and the CLI still fails.
	raw, _ := json.Marshal(func() Packet { p := fixture(); p.Acceptance.AcceptanceKind = "provisional"; return p }())
	path := filepath.Join(t.TempDir(), "p.json")
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := run([]string{"-packet", path, "-head", h, "-main", m}, &out); err == nil || !strings.Contains(out.String(), "PROVISIONAL_ACCEPTANCE_NOT_FORMAL_EVIDENCE") {
		t.Fatalf("err=%v out=%s", err, out.String())
	}
}

func TestE2ProviderGuaranteeMustBeAtomicWithCommit(t *testing.T) {
	const want = "E2_PROVIDER_GUARANTEE_NOT_ATOMIC"
	zeroWidth := string(rune(0x200b))
	for name, g := range map[string]*Guarantee{
		"omitted":         nil,
		"undeclared":      {Level: "undeclared"},
		"not-atomic":      {Level: "not-atomic", Reason: "x"},
		"empty":           {},
		"unknown":         {Level: "exactly-once"},
		"case":            {Level: "Atomic-With-Commit"},
		"underscore":      {Level: "atomic_with_commit"},
		"trailing space":  {Level: "atomic-with-commit "},
		"zero width":      {Level: "atomic-with-" + zeroWidth + "commit"},
		"provisional lvl": {Level: "provisional-atomic"},
	} {
		p := fixture()
		p.Physical.Guarantee = g
		r := Audit(p, h, m)
		if !has(r, want) || r.Status != "INCOMPLETE" || r.ProductGate != "UNQUALIFIED" {
			t.Fatalf("%s: %+v", name, r)
		}
	}
	// E0/E1 packets without the field are unchanged.
	for _, claim := range []string{"E0", "E1"} {
		p := fixture()
		p.Claim = claim
		p.Web.ProviderKind = "synthetic"
		p.Physical = Physical{}
		if r := Audit(p, h, m); has(r, want) || has(r, "PROVISIONAL_GUARANTEE_NOT_G2_EVIDENCE") {
			t.Fatalf("%s: %+v", claim, r)
		}
	}
}

func TestProvisionalGuaranteeIsNeverG2Evidence(t *testing.T) {
	const want = "PROVISIONAL_GUARANTEE_NOT_G2_EVIDENCE"
	zeroWidth, bidi, nl := string(rune(0x200b)), string(rune(0x202e)), string(rune(10))
	reasons := []string{"provisional D10: unverified", "Provisional", " PROVISIONAL	", "provisional_d10", "PROVISIONAL-D10",
		"prov" + zeroWidth + "isional D10", "provisional" + bidi, "x " + nl + "Provisional" + nl, "declared (provisional)"}
	for _, claim := range []string{"E0", "E1", "E2"} {
		for _, reason := range reasons {
			p := fixture()
			p.Claim = claim
			p.Physical.Guarantee = &Guarantee{Level: "atomic-with-commit", Reason: reason}
			r := Audit(p, h, m)
			if !has(r, want) || r.Status != "INCOMPLETE" || r.ProductGate != "UNQUALIFIED" {
				t.Fatalf("%s reason %q: %+v", claim, reason, r)
			}
			// A well-formed atomic level is not reported as merely non-atomic.
			if has(r, "E2_PROVIDER_GUARANTEE_NOT_ATOMIC") {
				t.Fatalf("%s reason %q: level is atomic: %+v", claim, reason, r)
			}
		}
	}
	// A provisional level fails two independent requirements at once: the
	// exact atomic level, and G2 evidence. Neither diagnosis implies the
	// other, so both are reported for the same declaration.
	for _, level := range []string{"provisional", "Provisional_Atomic"} {
		p := fixture()
		p.Physical.Guarantee = &Guarantee{Level: level}
		r := Audit(p, h, m)
		if !has(r, want) || !has(r, "E2_PROVIDER_GUARANTEE_NOT_ATOMIC") {
			t.Fatalf("%q: %+v", level, r)
		}
	}
	// Negative controls: each diagnosis occurs without the other.
	p := fixture()
	p.Physical.Guarantee = &Guarantee{Level: "atomic-with-commit", Reason: "provisional D10: unverified"}
	if r := Audit(p, h, m); !has(r, want) || has(r, "E2_PROVIDER_GUARANTEE_NOT_ATOMIC") {
		t.Fatalf("provisional reason with an atomic level: %+v", r)
	}
	p = fixture()
	p.Physical.Guarantee = &Guarantee{Level: "not-atomic", Reason: "declared by the Provider"}
	if r := Audit(p, h, m); has(r, want) || !has(r, "E2_PROVIDER_GUARANTEE_NOT_ATOMIC") {
		t.Fatalf("non-atomic level with an ordinary reason: %+v", r)
	}
	// An ordinary reason passes; the strict CLI accepts the field.
	p = fixture()
	raw, _ := json.Marshal(func() Packet {
		p.Physical.Guarantee = &Guarantee{Level: "atomic-with-commit", Reason: "provisional D10"}
		return p
	}())
	path := filepath.Join(t.TempDir(), "p.json")
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := run([]string{"-packet", path, "-head", h, "-main", m}, &out); err == nil || !strings.Contains(out.String(), want) {
		t.Fatalf("err=%v out=%s", err, out.String())
	}
	if r := Audit(fixture(), h, m); !reviewableExceptKind(r) {
		t.Fatalf("%+v", r)
	}
	// Duplicate keys and unknown members are still refused.
	for _, doc := range []string{`{"physical":{"guarantee":{"level":"a","level":"b"}}}`, `{"physical":{"guarantee":{"levl":"a"}}}`} {
		path := filepath.Join(t.TempDir(), "q.json")
		if err := os.WriteFile(path, []byte(doc), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := run([]string{"-packet", path, "-head", h, "-main", m}, &bytes.Buffer{}); err == nil {
			t.Fatalf("accepted %s", doc)
		}
	}
}

func TestE2MustNameTheAcceptanceKind(t *testing.T) {
	const want = "E2_ACCEPTANCE_KIND_REQUIRED"
	zeroWidth, bidi, bom, nl := string(rune(0x200b)), string(rune(0x202e)), string(rune(0xfeff)), string(rune(10))
	// Omitted, blank-only and invisible-only values name no kind.
	for name, kind := range map[string]string{
		"omitted": "", "space": " ", "tab/newline": "	" + nl, "zero width": zeroWidth, "bidi": bidi,
		"bom": bom, "mixed invisible": " " + zeroWidth + bidi + nl,
	} {
		p := fixture()
		p.Acceptance.AcceptanceKind = kind
		r := Audit(p, h, m)
		if !has(r, want) || r.Status != "INCOMPLETE" || r.ProductGate != "UNQUALIFIED" {
			t.Fatalf("%s: %+v", name, r)
		}
		// Treated exactly like an omitted field: one diagnosis, not also
		// "unrecognized".
		if has(r, "ACCEPTANCE_KIND_UNRECOGNIZED") {
			t.Fatalf("%s: blank-only kind is reported as unrecognized as well: %v", name, r.Diagnostics)
		}
		// E0/E1 are unchanged (legacy shape).
		for _, claim := range []string{"E0", "E1"} {
			p.Claim = claim
			if r := Audit(p, h, m); has(r, want) {
				t.Fatalf("%s %s: %+v", name, claim, r)
			}
		}
	}
	// A named kind is never reported as missing; each kind keeps its own diagnosis.
	for kind, other := range map[string]string{
		"provisional": "PROVISIONAL_ACCEPTANCE_NOT_FORMAL_EVIDENCE", "unspecified": "PROVISIONAL_ACCEPTANCE_NOT_FORMAL_EVIDENCE",
		"Prov" + zeroWidth + "isional": "PROVISIONAL_ACCEPTANCE_NOT_FORMAL_EVIDENCE",
		"formal":                       "ACCEPTANCE_KIND_UNRECOGNIZED", "x": "ACCEPTANCE_KIND_UNRECOGNIZED",
	} {
		p := fixture()
		p.Acceptance.AcceptanceKind = kind
		r := Audit(p, h, m)
		if has(r, want) || !has(r, other) || r.Status != "INCOMPLETE" || r.ProductGate != "UNQUALIFIED" {
			t.Fatalf("%q: %+v", kind, r)
		}
	}
	// The strict CLI reports it and fails.
	raw, _ := json.Marshal(fixture())
	path := filepath.Join(t.TempDir(), "p.json")
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	if err := run([]string{"-packet", path, "-head", h, "-main", m}, &out); err == nil || !strings.Contains(out.String(), want) {
		t.Fatalf("err=%v out=%s", err, out.String())
	}
}
