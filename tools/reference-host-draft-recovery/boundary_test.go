package hostdraftrecovery

import (
	"reflect"
	"testing"
)

func acceptedWorkspace() *Workspace {
	return &Workspace{WorkspaceID: "ws-repair", Revision: "r7", SourceBytes: "accepted source"}
}

func baseDraft() *Draft {
	return &Draft{
		WorkspaceID:           "ws-repair",
		BaseRevision:          "r7",
		BaseSourceFingerprint: fingerprintOf("accepted source"),
	}
}

// Preview and Reject never rewrite the accepted Source bytes or revision.
func TestPreviewAndRejectLeaveAcceptedSourceUnchanged(t *testing.T) {
	ws := acceptedWorkspace()
	draft := baseDraft()

	preview := Decide(Input{Workspace: ws, Draft: draft, WantAccept: false})
	if preview.Code != CodeDraftDisplayOnly || !preview.Allowed {
		t.Fatalf("preview=%+v", preview)
	}
	if ws.Revision != "r7" || ws.SourceBytes != "accepted source" {
		t.Fatalf("preview changed the accepted source: %+v", ws)
	}

	staleDraft := baseDraft()
	staleDraft.BaseRevision = "r6"
	stale := Decide(Input{Workspace: ws, Draft: staleDraft, WantAccept: true, WriteRecheck: true})
	if stale.Code != CodeRevisionStale || stale.Allowed {
		t.Fatalf("stale=%+v", stale)
	}
	if ws.Revision != "r7" || ws.SourceBytes != "accepted source" {
		t.Fatalf("reject changed the accepted source: %+v", ws)
	}

	mismatchDraft := baseDraft()
	mismatchDraft.BaseSourceFingerprint = fingerprintOf("other bytes")
	mismatch := Decide(Input{Workspace: ws, Draft: mismatchDraft, WantAccept: true, WriteRecheck: true})
	if mismatch.Code != CodeSourceBytesMismatch || mismatch.Allowed {
		t.Fatalf("mismatch=%+v", mismatch)
	}
	if ws.Revision != "r7" || ws.SourceBytes != "accepted source" {
		t.Fatalf("mismatch changed the accepted source: %+v", ws)
	}
}

func TestDraftFromAnotherWorkspaceFailsClosed(t *testing.T) {
	ws := acceptedWorkspace()
	draft := baseDraft()
	draft.WorkspaceID = "ws-other"
	report := Decide(Input{Workspace: ws, Draft: draft, WantAccept: true, WriteRecheck: true})
	if report.Code != CodeWrongWorkspace || report.Allowed {
		t.Fatalf("report=%+v want WRONG_WORKSPACE", report)
	}
}

func TestRevokedAuthorityFailsClosedEvenForPreview(t *testing.T) {
	ws := acceptedWorkspace()
	draft := baseDraft()
	report := Decide(Input{Workspace: ws, Draft: draft, WantAccept: false, AuthorityRevoked: true})
	if report.Code != CodeAuthorityRevoked || report.Allowed {
		t.Fatalf("report=%+v want AUTHORITY_REVOKED", report)
	}
	if ws.Revision != "r7" || ws.SourceBytes != "accepted source" {
		t.Fatalf("revocation changed the accepted source: %+v", ws)
	}
}

func TestDraftWithoutWorkspaceIDStillWorks(t *testing.T) {
	// Older drafts may not carry a workspace id; an empty value is not a
	// mismatch, and the remaining checks still apply.
	ws := acceptedWorkspace()
	draft := baseDraft()
	draft.WorkspaceID = ""
	report := Decide(Input{Workspace: ws, Draft: draft, WantAccept: true, WriteRecheck: true})
	if report.Code != CodeAcceptAllowed || !report.Allowed {
		t.Fatalf("report=%+v", report)
	}
}

func TestDecideDoesNotMutateTheInput(t *testing.T) {
	ws := acceptedWorkspace()
	draft := baseDraft()
	beforeWS := *ws
	beforeDraft := *draft
	_ = Decide(Input{Workspace: ws, Draft: draft, WantAccept: true, WriteRecheck: true})
	if !reflect.DeepEqual(*ws, beforeWS) || !reflect.DeepEqual(*draft, beforeDraft) {
		t.Fatalf("input mutated: ws=%+v draft=%+v", ws, draft)
	}
}
