package cognitionport

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func testdata(t *testing.T, parts ...string) string {
	t.Helper()
	base := filepath.Join("testdata")
	return filepath.Join(append([]string{base}, parts...)...)
}

func writeTempJSON(t *testing.T, value any) string {
	t.Helper()
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "value.json")
	if err := os.WriteFile(path, append(raw, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLocalProviderProposesWithoutAuthority(t *testing.T) {
	receipt, err := Propose(context.Background(), testdata(t, "request.json"), testdata(t, "providers", "local-rules-a.json"), "")
	if err != nil {
		t.Fatal(err)
	}
	if receipt.State != StateProposed {
		t.Fatalf("unexpected state: %+v", receipt)
	}
	if receipt.Proposal == nil || len(receipt.Proposal.Candidates) != 1 {
		t.Fatalf("missing proposal: %+v", receipt.Proposal)
	}
	candidate := receipt.Proposal.Candidates[0]
	if candidate.Subject != "sales.order_line.quantity" || len(candidate.Basis) == 0 {
		t.Fatalf("unexpected candidate: %+v", candidate)
	}
	if receipt.GrantsAuthority || receipt.Accepted || receipt.Applied {
		t.Fatal("proposal claimed authority, acceptance, or application")
	}
	if receipt.RequestDigest == "" || receipt.ContextDigest == "" || receipt.ProviderDigest == "" || receipt.InputFingerprint == "" || receipt.OutputDigest == "" {
		t.Fatalf("receipt lost provenance: %+v", receipt)
	}
	if receipt.Provider.Model != "deterministic-rules" || receipt.Provider.Execution != ExecutionLocal || receipt.Provider.Network != NetworkNone {
		t.Fatalf("unexpected provider identity: %+v", receipt.Provider)
	}
}

func TestSameContextCanBeReusedAcrossProviders(t *testing.T) {
	first, err := Propose(context.Background(), testdata(t, "request.json"), testdata(t, "providers", "local-rules-a.json"), "")
	if err != nil {
		t.Fatal(err)
	}
	second, err := Propose(context.Background(), testdata(t, "request.json"), testdata(t, "providers", "local-rules-b.json"), "")
	if err != nil {
		t.Fatal(err)
	}
	if first.InputFingerprint != second.InputFingerprint || first.ContextDigest != second.ContextDigest {
		t.Fatal("the same request produced different context fingerprints")
	}
	if first.ProviderDigest == second.ProviderDigest {
		t.Fatal("different providers were not distinguished")
	}
	if first.State != StateProposed || second.State != StateProposed {
		t.Fatalf("unexpected states: %s / %s", first.State, second.State)
	}
	if len(first.Proposal.Candidates) != 1 || len(second.Proposal.Candidates) != 1 {
		t.Fatalf("unexpected candidate counts: %d / %d", len(first.Proposal.Candidates), len(second.Proposal.Candidates))
	}
	left, right := first.Proposal.Candidates[0], second.Proposal.Candidates[0]
	if left.ID != right.ID || left.Subject != right.Subject || !reflect.DeepEqual(left.Payload, right.Payload) {
		t.Fatalf("provider swap changed the candidate: %+v vs %+v", left, right)
	}
}

func TestOfflineRequestBlocksNetworkProvider(t *testing.T) {
	receipt, err := Propose(context.Background(), testdata(t, "request.json"), testdata(t, "providers", "external-cloud.json"), "")
	if err != nil {
		t.Fatal(err)
	}
	if receipt.State != StateBlocked || receipt.Proposal != nil {
		t.Fatalf("offline request did not block the network provider: %+v", receipt)
	}
	if len(receipt.Diagnostics) == 0 || !strings.Contains(receipt.Diagnostics[0], "offline") {
		t.Fatalf("unexpected diagnostics: %+v", receipt.Diagnostics)
	}
}

func TestUnsupportedCapabilityIsNotSuccess(t *testing.T) {
	var request Request
	raw, err := os.ReadFile(testdata(t, "request.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &request); err != nil {
		t.Fatal(err)
	}
	request.Kind = "deployment-plan"
	requestPath := writeTempJSON(t, request)
	receipt, err := Propose(context.Background(), requestPath, testdata(t, "providers", "external-cloud.json"), "")
	if err != nil {
		t.Fatal(err)
	}
	if receipt.State != StateUnsupported || receipt.Proposal != nil {
		t.Fatalf("unsupported capability was not preserved: %+v", receipt)
	}
}

func TestUnknownProposalIsPreserved(t *testing.T) {
	var request Request
	raw, err := os.ReadFile(testdata(t, "request.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &request); err != nil {
		t.Fatal(err)
	}
	request.Context.Observations = []Observation{{ID: "obs-1", Statement: "nothing looks wrong", Reference: "sales.order_line.quantity"}}
	requestPath := writeTempJSON(t, request)
	receipt, err := Propose(context.Background(), requestPath, testdata(t, "providers", "local-rules-a.json"), "")
	if err != nil {
		t.Fatal(err)
	}
	if receipt.State != StateUnknown || receipt.Proposal == nil || len(receipt.Proposal.Candidates) != 0 {
		t.Fatalf("unknown was collapsed: %+v", receipt)
	}
	if len(receipt.Diagnostics) == 0 {
		t.Fatal("unknown state has no diagnostic")
	}
}

func TestExternalCommandProposalAndFailure(t *testing.T) {
	fixture := testdata(t, "proposals", "validation.json")
	command := fmt.Sprintf("cp %q \"$TEI_COGNITION_OUTPUT\"", fixture)
	receipt, err := Propose(context.Background(), testdata(t, "request.json"), testdata(t, "providers", "external-local.json"), command)
	if err != nil {
		t.Fatal(err)
	}
	if receipt.State != StateProposed || receipt.OutputDigest == "" {
		t.Fatalf("external proposal was not recorded: %+v", receipt)
	}
	failed, err := Propose(context.Background(), testdata(t, "request.json"), testdata(t, "providers", "external-local.json"), "exit 7")
	if err != nil {
		t.Fatal(err)
	}
	if failed.State != StateFailed || len(failed.Diagnostics) == 0 {
		t.Fatalf("external failure was not preserved: %+v", failed)
	}
}

func TestProposalCannotAcceptItself(t *testing.T) {
	raw, err := os.ReadFile(testdata(t, "proposals", "validation.json"))
	if err != nil {
		t.Fatal(err)
	}
	var proposal map[string]any
	if err := json.Unmarshal(raw, &proposal); err != nil {
		t.Fatal(err)
	}
	proposal["accepted"] = true
	path := filepath.Join(t.TempDir(), "accepted.json")
	changed, err := json.Marshal(proposal)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, changed, 0o600); err != nil {
		t.Fatal(err)
	}
	receipt, err := Propose(context.Background(), testdata(t, "request.json"), testdata(t, "providers", "external-local.json"), fmt.Sprintf("cp %q \"$TEI_COGNITION_OUTPUT\"", path))
	if err != nil {
		t.Fatal(err)
	}
	if receipt.State != StateFailed || len(receipt.Diagnostics) == 0 {
		t.Fatalf("proposal with acceptance was not failed closed: %+v", receipt)
	}
}

func TestAbsentBasisIsRejected(t *testing.T) {
	raw, err := os.ReadFile(testdata(t, "proposals", "validation.json"))
	if err != nil {
		t.Fatal(err)
	}
	var proposal Proposal
	if err := json.Unmarshal(raw, &proposal); err != nil {
		t.Fatal(err)
	}
	proposal.Candidates[0].Basis = []string{"observation:obs-9"}
	path := writeTempJSON(t, proposal)
	receipt, err := Propose(context.Background(), testdata(t, "request.json"), testdata(t, "providers", "external-local.json"), fmt.Sprintf("cp %q \"$TEI_COGNITION_OUTPUT\"", path))
	if err != nil {
		t.Fatal(err)
	}
	if receipt.State != StateFailed || !strings.Contains(strings.Join(receipt.Diagnostics, " "), "absent basis") {
		t.Fatalf("absent basis was not rejected: %+v", receipt)
	}
}

func TestContextLimitIsFailedClosed(t *testing.T) {
	var provider Provider
	raw, err := os.ReadFile(testdata(t, "providers", "local-rules-a.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &provider); err != nil {
		t.Fatal(err)
	}
	provider.Limits.MaxContextBytes = 16
	providerPath := writeTempJSON(t, provider)
	receipt, err := Propose(context.Background(), testdata(t, "request.json"), providerPath, "")
	if err != nil {
		t.Fatal(err)
	}
	if receipt.State != StateFailed || !strings.Contains(strings.Join(receipt.Diagnostics, " "), "context is") {
		t.Fatalf("context limit was not enforced: %+v", receipt)
	}
}
