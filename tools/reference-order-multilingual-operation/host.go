package ordermultilingualoperation

import (
	"context"
	"errors"
	"fmt"

	"tei/internal/sourceacceptance"
	"tei/internal/sourceindex"
	localization "tei/reference/message-localization"
)

// Host is the sample's synthetic host. It holds one current acceptance of
// (workspace source, source declarations, environment lock) and applies the
// same check before a message translation and before a display bundle. It is
// not a product acceptance mechanism: the decision reference is synthetic.
type Host struct {
	u                 Unified
	record            sourceacceptance.Record
	declarationDigest string
	lockFP            string
	revoked           bool
	Reads             int
}

// NewHost creates a host for the unified application.
func NewHost(u Unified) *Host { return &Host{u: u} }

// Accept records the current acceptance of the workspace load, the source
// release and the lock fingerprint.
func (h *Host) Accept(ws Workspace, rel SourceRelease, decision string) {
	h.record = sourceacceptance.Record{
		ApplicationID:     string(h.u.Application),
		DecisionRef:       decision,
		AuthorityRevision: "synthetic-host-" + rel.Release,
		ModelRevision:     ws.ModelRevision,
		Artifacts:         append([]sourceindex.Artifact(nil), ws.Index.Artifacts...),
	}
	h.declarationDigest = rel.Digest
	h.lockFP = ws.LockFP
	h.revoked = false
}

// Revoke withdraws the current acceptance.
func (h *Host) Revoke() { h.revoked = true }

// ReadCurrentAcceptedSource implements sourceacceptance.Reader.
func (h *Host) ReadCurrentAcceptedSource(_ context.Context, app string) (sourceacceptance.Record, error) {
	h.Reads++
	if h.revoked {
		return sourceacceptance.Record{}, errors.New("acceptance revoked")
	}
	if h.record.DecisionRef == "" {
		return sourceacceptance.Record{}, errors.New("no acceptance")
	}
	return h.record, nil
}

// Check verifies the current acceptance against this exact load (through the
// core sourceacceptance.VerifyCurrent), the source declarations and the lock.
func (h *Host) Check(ctx context.Context, ws Workspace, rel SourceRelease) error {
	if _, err := sourceacceptance.VerifyCurrent(ctx, h, string(h.u.Application), ws.Index); err != nil {
		return fmt.Errorf("host-acceptance: %v", err)
	}
	if rel.Digest != h.declarationDigest {
		return fmt.Errorf("host-acceptance: source declarations differ from the accepted release")
	}
	if ws.LockFP != h.lockFP {
		return fmt.Errorf("host-acceptance: environment lock differs from the accepted lock")
	}
	return nil
}

// MessageGate is the ReviewGate given to reference/message-localization.
func (h *Host) MessageGate(ctx context.Context, ws Workspace, rel SourceRelease) localization.ReviewGate {
	return func(in localization.ReviewInput) error {
		if err := h.Check(ctx, ws, rel); err != nil {
			return err
		}
		if in.LockFingerprint != h.lockFP {
			return fmt.Errorf("review lock %s is not the accepted lock", in.LockFingerprint)
		}
		return nil
	}
}
