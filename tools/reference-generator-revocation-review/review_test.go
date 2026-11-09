package generatorrevocationreview

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("test source path is unavailable")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func paths(t *testing.T) (registry, contract string) {
	t.Helper()
	base := filepath.Join(repoRoot(t), "tools", "reference-generator-selection", "testdata")
	return filepath.Join(base, "registry-lifecycle.json"), filepath.Join(base, "line-amount", "contract.json")
}

func approvalsPath(t *testing.T, name string) string {
	t.Helper()
	return filepath.Join("testdata", name)
}

func TestApprovedRevocationPassesSyntacticReview(t *testing.T) {
	registry, contract := paths(t)
	receipt, err := Review(registry, contract, approvalsPath(t, "approvals.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !receipt.Passed || len(receipt.Approved) != 1 || receipt.Approved[0] != "dangerous@1" {
		t.Fatalf("approved revocation was not accepted: %+v", receipt)
	}
	if receipt.GrantsAuthority || receipt.VerifiesIssuerTrust {
		t.Fatal("review claimed authority or issuer trust")
	}
	for _, entry := range receipt.Entries {
		if entry.ID == "dangerous" && (entry.State != "approved-syntax-only" || entry.Authority != "release-board") {
			t.Fatalf("unexpected revoked entry: %+v", entry)
		}
	}
}

func TestMissingApprovalFailsClosed(t *testing.T) {
	registry, contract := paths(t)
	receipt, err := Review(registry, contract, approvalsPath(t, "approvals-missing.json"))
	if err != nil {
		t.Fatal(err)
	}
	if receipt.Passed || len(receipt.Unapproved) != 1 || receipt.Unapproved[0] != "dangerous@1" {
		t.Fatalf("missing approval was accepted: %+v", receipt)
	}
}

func TestWrongTargetDigestFailsClosed(t *testing.T) {
	registry, contract := paths(t)
	receipt, err := Review(registry, contract, approvalsPath(t, "approvals-wrong-target.json"))
	if err != nil {
		t.Fatal(err)
	}
	if receipt.Passed || len(receipt.Unapproved) != 1 {
		t.Fatalf("approval for the wrong source was accepted: %+v", receipt)
	}
	for _, entry := range receipt.Entries {
		if entry.ID == "dangerous" && entry.Reason == "" {
			t.Fatalf("wrong-target reason is missing: %+v", entry)
		}
	}
}

func TestApprovalForNonRevokedEntryIsInvalid(t *testing.T) {
	registry, contract := paths(t)
	receipt, err := Review(registry, contract, approvalsPath(t, "approvals-active.json"))
	if err != nil {
		t.Fatal(err)
	}
	if receipt.Passed || len(receipt.Invalid) != 1 || receipt.Invalid[0] != "template@2" {
		t.Fatalf("approval for an active entry was accepted: %+v", receipt)
	}
}

func TestApprovalForUnknownTargetIsInvalid(t *testing.T) {
	registry, contract := paths(t)
	receipt, err := Review(registry, contract, approvalsPath(t, "approvals-unknown-target.json"))
	if err != nil {
		t.Fatal(err)
	}
	if receipt.Passed || len(receipt.Invalid) != 1 || receipt.Invalid[0] != "ghost@9" {
		t.Fatalf("approval for an unknown target was accepted: %+v", receipt)
	}
}

func TestDuplicateApprovalIsRejected(t *testing.T) {
	registry, contract := paths(t)
	raw, err := os.ReadFile(approvalsPath(t, "approvals.json"))
	if err != nil {
		t.Fatal(err)
	}
	var approvals Approvals
	if err := json.Unmarshal(raw, &approvals); err != nil {
		t.Fatal(err)
	}
	approvals.Approvals = append(approvals.Approvals, approvals.Approvals[0])
	changed, err := json.MarshalIndent(approvals, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "duplicate.json")
	if err := os.WriteFile(path, append(changed, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Review(registry, contract, path); err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("duplicate approval was accepted: %v", err)
	}
}

func TestApprovalWithoutAuthorityIsRejected(t *testing.T) {
	registry, contract := paths(t)
	raw, err := os.ReadFile(approvalsPath(t, "approvals.json"))
	if err != nil {
		t.Fatal(err)
	}
	var approvals Approvals
	if err := json.Unmarshal(raw, &approvals); err != nil {
		t.Fatal(err)
	}
	approvals.Approvals[0].Authority = ""
	changed, err := json.MarshalIndent(approvals, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "missing-authority.json")
	if err := os.WriteFile(path, append(changed, '\n'), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Review(registry, contract, path); err == nil || !strings.Contains(err.Error(), "authority") {
		t.Fatalf("approval without authority was accepted: %v", err)
	}
}
