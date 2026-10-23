package main

import (
	"bytes"
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"
)

const notExecutedFixture = "testdata/p0-order-entry-e2-not-executed.json"

// 実runの前に、#1163の一改訂packetが「何がまだ無いか」を機械可読に示す。
// 値を持つ欄を足すと、実行していない証跡を取得済みと読ませる危険があるため、
// 未実行の欄は空文字のままにする(非空のlocatorは存在の申告として扱われる)。
func TestNotExecutedE2FixtureListsEveryMissingHandoff(t *testing.T) {
	var out bytes.Buffer
	err := run([]string{"-packet", notExecutedFixture, "-head", h, "-main", m}, &out)
	if err == nil {
		t.Fatal("a packet with no executed evidence must exit non-zero")
	}
	var result Report
	if err := json.Unmarshal(out.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Status != "INCOMPLETE" || result.ProductGate != "UNQUALIFIED" || result.Requested != "E2" {
		t.Fatalf("not-executed packet was promoted: %+v", result)
	}
	want := []string{
		"ACCEPTANCE_REVIEW_AND_RELOAD_EVIDENCE_MISSING",
		"ACCEPTANCE_REVIEW_CANDIDATE_BINDING_MISSING_OR_MISMATCH",
		"ACCEPTED_R0_R1_LINEAGE_MISSING",
		"ACTUAL_WEB_POSITIVE_AND_NEGATIVE_RECEIPTS_REQUIRED",
		"E1_E2_ACTUAL_EXECUTION_CLASS_REQUIRED",
		"E1_NEGATIVE_SELECTED_RULE_RUNNING_WEB_EVIDENCE_REQUIRED",
		"E1_RUNNING_WEB_TARGET_RECEIPT_REQUIRED",
		"E1_WEB_WORKSPACE_R1_AND_TARGET_LAUNCH_PROVENANCE_MISSING",
		"E2_COMMAND_AND_INDEPENDENT_READ_NOT_CORRELATED",
		"N11_UNOBSERVED_CONTROL_MISSING",
		"REVISION_QUALIFICATION_STALE_OR_MISSING",
		"S02_NEGATIVE_CONTROL_MISSING",
		"S02_REAL_DATASOURCE_SCHEMA_READINESS_MISSING",
		"S05_CURRENT_HOST_AUTHORITY_MISSING",
		"S05_NEGATIVE_CONTROL_MISSING",
		"SYNTHETIC_WEB_PROVIDER_CANNOT_CLAIM_E2",
		"TRUSTED_FULL_HEALTH_NOT_RECORDED",
		"WEB_PROVIDER_KIND_UNRESOLVED",
	}
	if !reflect.DeepEqual(result.Diagnostics, want) {
		t.Fatalf("missing-handoff list changed:\n got %v\nwant %v", result.Diagnostics, want)
	}
}

// fixtureに実行結果の欄が埋まっていれば、このfixtureは「未実行」ではなくなる。
func TestNotExecutedE2FixtureCarriesNoEvidenceValues(t *testing.T) {
	data, err := os.ReadFile(notExecutedFixture)
	if err != nil {
		t.Fatal(err)
	}
	if err := rejectDuplicateJSONKeys(data); err != nil {
		t.Fatal(err)
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	var p Packet
	if err := dec.Decode(&p); err != nil {
		t.Fatal(err)
	}
	for name, value := range map[string]string{
		"qualification.head": p.Qualification.Head, "qualification.main": p.Qualification.Main,
		"qualification.receipt": p.Qualification.Receipt,
		"acceptance.r0":         p.Acceptance.R0, "acceptance.r1": p.Acceptance.R1,
		"acceptance.candidateFingerprint": p.Acceptance.CandidateFingerprint,
		"acceptance.reviewFingerprint":    p.Acceptance.ReviewFingerprint,
		"acceptance.reviewCandidate":      p.Acceptance.ReviewCandidateFingerprint,
		"acceptance.decisionReceipt":      p.Acceptance.DecisionReceipt,
		"acceptance.reloadReceipt":        p.Acceptance.ReloadReceipt,
		"web.acceptedRevision":            p.Web.AcceptedRevision,
		"web.negativeValidation":          p.Web.NegativeValidation,
		"web.negativeEvidenceMode":        p.Web.NegativeEvidenceMode,
		"web.providerKind":                p.Web.ProviderKind, "web.runMode": p.Web.RunMode,
		"web.workspaceRevisionReceipt": p.Web.WorkspaceRevisionReceipt,
		"web.targetLaunchReceipt":      p.Web.TargetLaunchReceipt,
		"web.positiveReceipt":          p.Web.PositiveReceipt, "web.negativeReceipt": p.Web.NegativeReceipt,
		"web.unobservedControlReceipt": p.Web.UnobservedControlReceipt,
		"physical.acceptedRevision":    p.Physical.AcceptedRevision, "physical.datasource": p.Physical.Datasource,
		"physical.schemaEvidence":         p.Physical.SchemaEvidence,
		"physical.hostAuthorityEvidence":  p.Physical.HostAuthorityEvidence,
		"physical.commandReceipt":         p.Physical.CommandReceipt,
		"physical.independentReadReceipt": p.Physical.IndependentReadReceipt,
		"physical.recordId":               p.Physical.RecordID,
		"physical.schemaFailureControl":   p.Physical.SchemaFailureControl,
		"physical.hostFailureControl":     p.Physical.HostFailureControl,
	} {
		if strings.TrimSpace(value) != "" {
			t.Errorf("%s holds %q: a not-executed packet must leave unexecuted fields empty", name, value)
		}
	}
	if p.Qualification.FullHealth != "NOT_RUN" || p.Qualification.CleanBefore || p.Qualification.CleanAfter ||
		p.Physical.SchemaCompatibility != "unresolved" || p.Contract.EvidenceClass != "source-only" {
		t.Fatalf("fixture claims an executed or observed state: %+v", p)
	}
	lowered := strings.ToLower(string(data))
	for _, forbidden := range []string{"postgres://", "postgresql://", "password", "secret", "token", "dsn"} {
		if strings.Contains(lowered, forbidden) {
			t.Errorf("fixture contains %q; packets must not carry connection strings or credentials", forbidden)
		}
	}
}
