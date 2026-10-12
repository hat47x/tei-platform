package effectfence

import "testing"

func TestContractDoesNotClaimProductionGuaranteeFromMemory(t *testing.T) {
	c := NewMemoryStore().BoundaryContract()
	if got := AssessBoundary(c, GuaranteeProcess); !got.Allowed || got.Available != GuaranteeProcess {
		t.Fatalf("synthetic in-process proof was rejected: %+v", got)
	}
	if got := AssessBoundary(c, GuaranteeDurable); got.Allowed || got.Available == GuaranteeDurable {
		t.Fatalf("memory unexpectedly certified as durable: %+v", got)
	}
}

func TestContractAdmissionFailClosed(t *testing.T) {
	valid := BoundaryContract{
		Boundary:                BoundaryTransaction,
		AuthorityCheckedAtWrite: true,
		ResourceCheckedAtWrite:  true,
		ReceiptStoredWithWrite:  true,
		SubjectLinksCovered:     true,
		DurableReceipt:          true,
		RecoveryLookup:          true,
		Proof:                   ProofProvider,
		ProofReference:          "evidence/actual-provider-commit",
	}
	if a := AssessBoundary(valid, GuaranteeDurable); a.Allowed || a.Reason != "provider-evidence-not-attested" {
		t.Fatalf("untrusted provider self-declaration accepted: %+v", a)
	}
	verify := func(c BoundaryContract) bool { return c.ProofReference == "evidence/actual-provider-commit" }
	if a := AssessVerifiedBoundary(valid, GuaranteeDurable, verify); !a.Allowed || a.Available != GuaranteeDurable {
		t.Fatalf("validated provider declaration not admitted: %+v", a)
	}
	cases := []struct {
		name   string
		modify func(*BoundaryContract)
	}{
		{"authority-preflight-only", func(c *BoundaryContract) { c.AuthorityCheckedAtWrite = false }},
		{"resource-not-checked", func(c *BoundaryContract) { c.ResourceCheckedAtWrite = false }},
		{"receipt-not-atomic", func(c *BoundaryContract) { c.ReceiptStoredWithWrite = false }},
		{"subject-link-not-guarded", func(c *BoundaryContract) { c.SubjectLinksCovered = false }},
		{"post-commit-external-effect", func(c *BoundaryContract) { c.RemoteEffect = true }},
		{"receipt-not-durable", func(c *BoundaryContract) { c.DurableReceipt = false }},
		{"recovery-unavailable", func(c *BoundaryContract) { c.RecoveryLookup = false }},
		{"synthetic-evidence-only", func(c *BoundaryContract) { c.Proof = ProofSynthetic }},
		{"unverified", func(c *BoundaryContract) { c.Proof = ProofUnverified }},
		{"missing-evidence-reference", func(c *BoundaryContract) { c.ProofReference = "" }},
		{"external-provider", func(c *BoundaryContract) { c.Boundary = BoundaryIndependent }},
		{"unknown-boundary", func(c *BoundaryContract) { c.Boundary = "invented" }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := valid
			tc.modify(&c)
			if got := AssessVerifiedBoundary(c, GuaranteeDurable, func(BoundaryContract) bool { return true }); got.Allowed || got.Reason == "" {
				t.Fatalf("unsafe declaration admitted: %+v", got)
			}
		})
	}
	if got := AssessBoundary(valid, ""); got.Allowed || got.Reason != "invalid-requirement" {
		t.Fatalf("unspecified policy admitted: %+v", got)
	}
	if got := AssessBoundary(BoundaryContract{}, GuaranteeDurable); got.Allowed {
		t.Fatalf("zero contract admitted: %+v", got)
	}
}
