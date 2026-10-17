// Package hostdraftrecovery is a read-only, synthetic reference for the
// Desktop Studio Host issues (#1529). It distinguishes the accepted
// modelRevision from the accepted Source byte fingerprint, and decides how a
// Desktop draft behaves across an external edit, a host or WebView restart,
// and an interrupted side effect.
//
// It is not a Source Authority, an Application Service, a Plugin Resolver, a
// second acceptance controller, or an accepted API. It writes nothing: an
// accepted Source byte and revision stay unchanged on Reject and Preview. A
// restored draft is only ever displayed again; it never promotes itself to
// the accepted source, and an external side effect with an unknown outcome is
// never resent.
package hostdraftrecovery

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

// Result codes.
const (
	CodeAcceptAllowed       = "ACCEPT_ALLOWED"
	CodeDraftDisplayOnly    = "DRAFT_DISPLAY_ONLY"
	CodeRevisionStale       = "REVISION_STALE"
	CodeSourceBytesMismatch = "SOURCE_BYTES_MISMATCH"
	CodeWriteRecheckMissing = "WRITE_RECHECK_MISSING"
	CodeConcurrentWriter    = "CONCURRENT_WRITER_REJECTED"
	CodeDraftNotPromoted    = "DRAFT_NOT_PROMOTED"
	CodeSideEffectNoResend  = "SIDE_EFFECT_UNKNOWN_NO_RESEND"
	CodeVerifyObservation   = "VERIFY_FROM_OBSERVATION"
	CodeInvalidRequest      = "INVALID_REQUEST"
)

// Workspace is the accepted state of the workspace as the Host reads it now.
// The fingerprint is computed from the bytes, never trusted from a label.
type Workspace struct {
	WorkspaceID string `json:"workspaceId"`
	// Revision is the current accepted modelRevision.
	Revision string `json:"revision"`
	// SourceBytes are the current accepted Source bytes.
	SourceBytes string `json:"sourceBytes"`
}

// Draft is one undelegated Desktop candidate or draft kept separate from the
// accepted Source.
type Draft struct {
	// BaseRevision is the modelRevision the draft was based on.
	BaseRevision string `json:"baseRevision"`
	// BaseSourceFingerprint is the byte fingerprint recorded when the draft was
	// created, in the `sha256:<hex>` form.
	BaseSourceFingerprint string `json:"baseSourceFingerprint"`
	// Content is the persisted draft body. Restoring it only displays it again.
	Content string `json:"content,omitempty"`
	// ExplicitReviewDone is true only after a reviewer completed an explicit
	// review following the last restore or reconnect.
	ExplicitReviewDone bool `json:"explicitReviewDone"`
}

// Input is the whole synthetic request.
type Input struct {
	Workspace *Workspace `json:"workspace"`
	Draft     *Draft     `json:"draft"`
	// WantAccept proposes promoting the restored draft through the existing
	// acceptance entrance.
	WantAccept bool `json:"wantAccept"`
	// WriteRecheck is the result of re-checking the workspace fingerprint just
	// before a promotion is written. Watcher events can be lost, duplicated or
	// reordered, so this recheck is the protection.
	WriteRecheck bool `json:"writeRecheck"`
	// TwoWriters is true when another window or process edits the same
	// workspace.
	TwoWriters bool `json:"twoWriters"`
	// ResumedAfterRestart is true after a host or WebView restart or a
	// disconnect; the accepted source must be reread and the draft must not
	// promote itself.
	ResumedAfterRestart bool `json:"resumedAfterRestart"`
	// SideEffectPending is true when an external side effect had been started
	// and its outcome is unknown after a host terminate.
	SideEffectPending bool `json:"sideEffectPending"`
	// ObservationFound is true when an existing Run or Observation records the
	// outcome of the pending side effect.
	ObservationFound bool `json:"observationFound"`
}

// Report is the read-only result.
type Report struct {
	Code    string `json:"code"`
	Allowed bool   `json:"allowed"`
	// Fingerprint is the computed byte fingerprint of the accepted source.
	Fingerprint string `json:"fingerprint,omitempty"`
	Message     string `json:"message,omitempty"`
}

func reject(code, message string, fingerprint string) Report {
	return Report{Code: code, Fingerprint: fingerprint, Message: message}
}

func fingerprintOf(sourceBytes string) string {
	sum := sha256.Sum256([]byte(sourceBytes))
	return "sha256:" + hex.EncodeToString(sum[:])
}

func fingerprintWellFormed(value string) bool {
	if !strings.HasPrefix(value, "sha256:") {
		return false
	}
	return len(value) == len("sha256:")+64
}

// Decide checks one Host draft situation. It never writes anything and never
// resends a side effect.
func Decide(in Input) Report {
	if in.Workspace == nil || in.Draft == nil {
		return reject(CodeInvalidRequest, "workspace と draft の双方が必要。", "")
	}
	if strings.TrimSpace(in.Workspace.WorkspaceID) == "" {
		return reject(CodeInvalidRequest, "workspaceId が未指定。", "")
	}
	ws := strings.TrimSpace(in.Workspace.Revision)
	fp := fingerprintOf(in.Workspace.SourceBytes)

	if in.SideEffectPending {
		if !in.ObservationFound {
			return reject(CodeSideEffectNoResend,
				"外部副作用の完了が不明。既存 Run／Observation に問い合わせるまで再送しない。", fp)
		}
		return reject(CodeVerifyObservation,
			"既存 Observation で結果を照合する。推測で再送も昇格もしない。", fp)
	}

	if !in.WantAccept {
		return Report{Code: CodeDraftDisplayOnly, Allowed: true, Fingerprint: fp,
			Message: "未受理Draftは表示・保存・再読込のみ。Acceptedへは明示Reviewが終わるまで昇格しない。"}
	}

	if ws != strings.TrimSpace(in.Draft.BaseRevision) {
		return reject(CodeRevisionStale,
			fmt.Sprintf("modelRevision が不一致（draft基点 %s、現在 %s）。fail closed。", in.Draft.BaseRevision, ws), fp)
	}
	if !fingerprintWellFormed(in.Draft.BaseSourceFingerprint) {
		return reject(CodeInvalidRequest,
			fmt.Sprintf("draft基点のfingerprint %q が sha256:<hex> 形式でない。", in.Draft.BaseSourceFingerprint), fp)
	}
	if fp != strings.TrimSpace(in.Draft.BaseSourceFingerprint) {
		return reject(CodeSourceBytesMismatch,
			fmt.Sprintf("revision が一致しても Source バイトが不一致（現在 %s）。外部編集後の旧Preview受理を拒否する。", fp), fp)
	}
	if in.TwoWriters && !in.WriteRecheck {
		return reject(CodeConcurrentWriter,
			"別プロセス・別ウィンドウが同じWorkspaceを編集中。書込み直前の再検査が無い限り拒否する。", fp)
	}
	if !in.WriteRecheck {
		return reject(CodeWriteRecheckMissing,
			"Warehouse/Watcher通知の欠落・重複・順序逆転に依存せず、保存時の再検査で保護する。", fp)
	}
	if in.ResumedAfterRestart && !in.Draft.ExplicitReviewDone {
		return reject(CodeDraftNotPromoted,
			"再起動・切断後のDraftは明示Reviewが終わるまで Accepted へ昇格しない。Accepted Source を読み直す。", fp)
	}
	return Report{Code: CodeAcceptAllowed, Allowed: true, Fingerprint: fp,
		Message: "revision・Sourceバイトfingerprint・書込み時の再検査・明示Reviewの全てが確認できた。受理入口（#1124/#267）へ移譲する。本参照は書込みを行わない。"}
}
