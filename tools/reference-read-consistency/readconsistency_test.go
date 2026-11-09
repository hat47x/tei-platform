package readconsistency

import "testing"

// The fixtures follow Data Derivation D11 (#1708): 在庫・引当済 are updated in
// one DB transaction.
//
//	R0: 在庫120, 引当済100, 引当可能20
//	R1: 在庫110, 引当済90,  引当可能20
//
// Mixing R0 在庫 with R1 引当済 yields 30, which is in neither revision.

func reads(stock, stockRev, reserved, reservedRev string) []Read {
	return []Read{
		{InputID: "在庫", Revision: stockRev, State: ReadPresent, Value: stock},
		{InputID: "引当済", Revision: reservedRev, State: ReadPresent, Value: reserved},
	}
}

func TestFixedSnapshotReturnsTheRevisionValue(t *testing.T) {
	for name, rev := range map[string][2]string{"R0": {"120", "100"}, "R1": {"110", "90"}} {
		in := Input{
			RequiredLevel:     LevelSameTransactionSnapshot,
			ProviderGuarantee: LevelSameTransactionSnapshot,
			Operation:         OpSubtract,
			Reads:             reads(rev[0], name, rev[1], name),
		}
		report := Evaluate(in)
		if !report.Accepted || report.Code != CodeOK || report.Result != "20" {
			t.Fatalf("%s report=%+v want RESULT_OK 20", name, report)
		}
	}
}

func TestMixedSnapshotIsRejectedUnderStrongRequirement(t *testing.T) {
	in := Input{
		RequiredLevel:     LevelSameTransactionSnapshot,
		ProviderGuarantee: LevelSameTransactionSnapshot,
		Operation:         OpSubtract,
		Reads:             reads("120", "R0", "90", "R1"),
	}
	report := Evaluate(in)
	if report.Accepted || report.Code != CodeMixedSnapshot {
		t.Fatalf("report=%+v want MIXED_SNAPSHOT", report)
	}
	if report.Mixed != "30" {
		t.Fatalf("mixedValue=%q want 30", report.Mixed)
	}
}

func TestBestEffortAcceptsANonAtomicRead(t *testing.T) {
	in := Input{
		RequiredLevel:    LevelBestEffort,
		AcceptBestEffort: true,
		Operation:        OpSubtract,
		Reads:            reads("120", "R0", "90", "R1"),
	}
	report := Evaluate(in)
	if !report.Accepted || report.Code != CodeNonAtomicRead || report.Result != "30" {
		t.Fatalf("report=%+v want NON_ATOMIC_READ 30", report)
	}
}

func TestBestEffortWithoutCaseAcceptanceIsRejected(t *testing.T) {
	in := Input{
		RequiredLevel: LevelBestEffort,
		Operation:     OpSubtract,
		Reads:         reads("120", "R0", "90", "R1"),
	}
	report := Evaluate(in)
	if report.Accepted || report.Code != CodeBestEffortNotAccepted {
		t.Fatalf("report=%+v want BEST_EFFORT_NOT_ACCEPTED", report)
	}
}

func TestCrossProviderStrongGuaranteeIsUnsupported(t *testing.T) {
	in := Input{
		RequiredLevel:     LevelSameTransactionSnapshot,
		ProviderGuarantee: LevelSameTransactionSnapshot,
		MultiProvider:     true,
		Operation:         OpSubtract,
		Reads:             reads("120", "R0", "100", "R0"),
	}
	report := Evaluate(in)
	if report.Accepted || report.Code != CodeUnsupported {
		t.Fatalf("report=%+v want UNSUPPORTED_CONSISTENCY", report)
	}
}

func TestProviderWithoutTheRequiredGuaranteeIsUnsupported(t *testing.T) {
	in := Input{
		RequiredLevel:     LevelVersionAligned,
		ProviderGuarantee: LevelSameRecord,
		Operation:         OpSubtract,
		Reads:             reads("120", "R0", "100", "R0"),
	}
	report := Evaluate(in)
	if report.Accepted || report.Code != CodeUnsupported {
		t.Fatalf("report=%+v want UNSUPPORTED_CONSISTENCY", report)
	}
}

func TestLostSnapshotIsNotZeroOrNull(t *testing.T) {
	in := Input{
		RequiredLevel:     LevelSameTransactionSnapshot,
		ProviderGuarantee: LevelSameTransactionSnapshot,
		SnapshotTokenLost: true,
		Operation:         OpSubtract,
		Reads:             reads("120", "R0", "100", "R0"),
	}
	report := Evaluate(in)
	if report.Accepted || report.Code != CodeInputUnavailable {
		t.Fatalf("report=%+v want INPUT_UNAVAILABLE", report)
	}
}

func TestRetryWithChangedAuthorizationRequiresReauth(t *testing.T) {
	in := Input{
		RequiredLevel:               LevelSameTransactionSnapshot,
		ProviderGuarantee:           LevelSameTransactionSnapshot,
		Retry:                       true,
		AuthorizationVersionChanged: true,
		RetryReauthorized:           false,
		Operation:                   OpSubtract,
		Reads:                       reads("120", "R0", "100", "R0"),
	}
	report := Evaluate(in)
	if report.Accepted || report.Code != CodeRetryRequiresReauth {
		t.Fatalf("report=%+v want RETRY_REQUIRES_REAUTHORIZATION", report)
	}
}

func TestMissingAndNullDiagnosticsStayDistinct(t *testing.T) {
	base := Input{
		RequiredLevel:     LevelSameTransactionSnapshot,
		ProviderGuarantee: LevelSameTransactionSnapshot,
		Operation:         OpSubtract,
	}
	base.Reads = []Read{
		{InputID: "在庫", Revision: "R0", State: ReadPresent, Value: "120"},
		{InputID: "引当済", Revision: "R0", State: ReadMissing},
	}
	if report := Evaluate(base); report.Code != CodeInputMissing {
		t.Fatalf("report=%+v want INPUT_MISSING", report)
	}
	base.Reads[1].State = ReadNull
	if report := Evaluate(base); report.Code != CodeNullInput {
		t.Fatalf("report=%+v want NULL_INPUT", report)
	}
}

func TestAuthorizationIsDelegated(t *testing.T) {
	in := Input{
		RequiredLevel:       LevelSameTransactionSnapshot,
		ProviderGuarantee:   LevelSameTransactionSnapshot,
		AuthorizationDenied: true,
		Operation:           OpSubtract,
		Reads:               reads("120", "R0", "100", "R0"),
	}
	report := Evaluate(in)
	if report.Accepted || report.Code != CodeAuthorizationDelegated {
		t.Fatalf("report=%+v want AUTHORIZATION_DELEGATED", report)
	}
}
