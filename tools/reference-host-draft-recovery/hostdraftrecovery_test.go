package hostdraftrecovery

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

func workspace(revision, bytes string) *Workspace {
	return &Workspace{WorkspaceID: "ws-repair", Revision: revision, SourceBytes: bytes}
}

func fingerprint(bytes string) string {
	sum := sha256.Sum256([]byte(bytes))
	return "sha256:" + hex.EncodeToString(sum[:])
}

func draft(revision, bytes string) *Draft {
	return &Draft{BaseRevision: revision, BaseSourceFingerprint: fingerprint(bytes), Content: "draft body"}
}

func acceptInput(revision, bytes string) Input {
	return Input{
		Workspace:    workspace(revision, bytes),
		Draft:        draft(revision, bytes),
		WantAccept:   true,
		WriteRecheck: true,
	}
}

func TestFreshDraftAfterExplicitReviewIsAccepted(t *testing.T) {
	report := Decide(acceptInput("r7", "accepted source bytes"))
	if !report.Allowed || report.Code != CodeAcceptAllowed {
		t.Fatalf("report=%+v want ACCEPT_ALLOWED", report)
	}
	if report.Fingerprint != fingerprint("accepted source bytes") {
		t.Fatalf("fingerprint=%s want the computed value", report.Fingerprint)
	}
}

func TestStaleRevisionFailsClosed(t *testing.T) {
	// Workspace moved to r8 while the draft was still based on r7.
	in := Input{
		Workspace:    workspace("r8", "accepted source bytes"),
		Draft:        draft("r7", "accepted source bytes"),
		WantAccept:   true,
		WriteRecheck: true,
	}
	if report := Decide(in); report.Allowed || report.Code != CodeRevisionStale {
		t.Fatalf("report=%+v want REVISION_STALE", report)
	}
}

func TestSameRevisionWithDifferentBytesRefusesTheOldPreview(t *testing.T) {
	// The draft recorded r7; an external edit kept the revision but changed the
	// accepted source bytes.
	in := Input{
		Workspace:    workspace("r7", "externally edited bytes"),
		Draft:        draft("r7", "accepted source bytes"),
		WantAccept:   true,
		WriteRecheck: true,
	}
	report := Decide(in)
	if report.Allowed || report.Code != CodeSourceBytesMismatch {
		t.Fatalf("report=%+v want SOURCE_BYTES_MISMATCH", report)
	}
}

func TestWatcherLossIsNotTrustedWithoutTheSaveTimeRecheck(t *testing.T) {
	in := acceptInput("r7", "accepted source bytes")
	in.WriteRecheck = false
	if report := Decide(in); report.Allowed || report.Code != CodeWriteRecheckMissing {
		t.Fatalf("report=%+v want WRITE_RECHECK_MISSING", report)
	}
}

func TestConcurrentWriterFailsClosedWithoutTheRecheck(t *testing.T) {
	in := acceptInput("r7", "accepted source bytes")
	in.TwoWriters = true
	in.WriteRecheck = false
	if report := Decide(in); report.Code != CodeConcurrentWriter {
		t.Fatalf("report=%+v want CONCURRENT_WRITER_REJECTED", report)
	}
}

func TestRestartedHostDoesNotSelfPromoteTheDraft(t *testing.T) {
	in := acceptInput("r7", "accepted source bytes")
	in.ResumedAfterRestart = true
	if report := Decide(in); report.Allowed || report.Code != CodeDraftNotPromoted {
		t.Fatalf("report=%+v want DRAFT_NOT_PROMOTED", report)
	}
	// After the reviewer completes an explicit review the promotion may pass.
	in.Draft.ExplicitReviewDone = true
	if report := Decide(in); !report.Allowed || report.Code != CodeAcceptAllowed {
		t.Fatalf("report=%+v want ACCEPT_ALLOWED after explicit review", report)
	}
}

func TestPendingSideEffectIsNeverResentWithoutEvidence(t *testing.T) {
	in := acceptInput("r7", "accepted source bytes")
	in.SideEffectPending = true
	report := Decide(in)
	if report.Allowed || report.Code != CodeSideEffectNoResend {
		t.Fatalf("report=%+v want SIDE_EFFECT_UNKNOWN_NO_RESEND", report)
	}
}

func TestImpossibleSideEffectOutcomeIsVerifiedThroughObservation(t *testing.T) {
	in := acceptInput("r7", "accepted source bytes")
	in.SideEffectPending = true
	in.ObservationFound = true
	if report := Decide(in); report.Allowed || report.Code != CodeVerifyObservation {
		t.Fatalf("report=%+v want VERIFY_FROM_OBSERVATION", report)
	}
}

func TestDisplayOnlyNeverPromotes(t *testing.T) {
	in := acceptInput("r7", "accepted source bytes")
	in.WantAccept = false
	report := Decide(in)
	if !report.Allowed || report.Code != CodeDraftDisplayOnly {
		t.Fatalf("report=%+v want DRAFT_DISPLAY_ONLY", report)
	}
}

func TestMalformedFingerprintIsRejected(t *testing.T) {
	in := acceptInput("r7", "accepted source bytes")
	in.Draft.BaseSourceFingerprint = "r7"
	if report := Decide(in); report.Code != CodeInvalidRequest {
		t.Fatalf("report=%+v want INVALID_REQUEST", report)
	}
}

func TestMissingWorkspaceOrDraftIsRejected(t *testing.T) {
	if report := Decide(Input{}); report.Code != CodeInvalidRequest {
		t.Fatalf("report=%+v want INVALID_REQUEST", report)
	}
}
