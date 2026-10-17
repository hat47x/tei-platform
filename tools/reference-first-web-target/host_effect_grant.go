//go:build !tei_order_only && !tei_repair_only

package main

import (
	"context"
	"errors"
	"strings"

	"tei/internal/sourceacceptance"
	"tei/internal/sourceindex"
)

// HostEffectGrantQuery binds a requested business effect to the authenticated
// subject, exact Action, and the Host Accepted Source used to build this World.
// No B/R/T label or Shared Acceptance Decision file is an effect permission.
type HostEffectGrantQuery struct {
	ApplicationID           string
	SubjectID               string
	ActionID                string
	SourceDecisionRef       string
	SourceAuthorityRevision string
	ModelRevision           string
}

// HostEffectGrantRecord is only a read-only view of the current Host grant.
// It is NOT an independently accepted grant or a replacement Authority store.
type HostEffectGrantRecord struct {
	Query                  HostEffectGrantQuery
	GrantRef               string
	GrantAuthorityRevision string
}

// HostEffectGrantReader must be injected by a trusted Host and consult its
// current subject/action/Source-specific permission on every effect invocation.
// It must check revocation and return an error on missing, expired or denied
// permission. User JSON, test fixtures and Source Acceptance do not implement
// a trustworthy production Grant issuer.
type HostEffectGrantReader interface {
	ReadCurrentEffectGrant(context.Context, HostEffectGrantQuery) (HostEffectGrantRecord, error)
}

var errHostEffectGrantUnavailable = errors.New("HOST_EFFECT_GRANT_UNAVAILABLE")

// requireCurrentHostEffectGrant is called inside the concrete Command Adapter,
// immediately before it changes state. Recheck current Source in addition to
// the HTTP admission, since authentication/validation can take time. Then ask
// the independent Host permission authority for this exact subject and Action.
//
// This does not lock the Host's grant across the physical write. A genuine
// Provider must join its permission check and mutation to its owner-specific
// atomic effect boundary (e.g. #1174); this synthetic runner cannot prove it.
func requireCurrentHostEffectGrant(
	ctx context.Context,
	sourceReader sourceacceptance.Reader,
	grantReader HostEffectGrantReader,
	applicationID, subjectID, actionID string,
	index sourceindex.Index,
	accepted sourceacceptance.Record,
) error {
	if ctx == nil || grantReader == nil || sourceReader == nil ||
		strings.TrimSpace(subjectID) == "" || strings.TrimSpace(actionID) == "" {
		return errHostEffectGrantUnavailable
	}
	if ctx.Err() != nil {
		return errHostEffectGrantUnavailable
	}
	current, err := sourceacceptance.VerifyCurrent(ctx, sourceReader, applicationID, index)
	if err != nil || current.DecisionRef != accepted.DecisionRef ||
		current.AuthorityRevision != accepted.AuthorityRevision ||
		current.ModelRevision != accepted.ModelRevision {
		return errHostEffectGrantUnavailable
	}
	query := HostEffectGrantQuery{
		ApplicationID:           applicationID,
		SubjectID:               subjectID,
		ActionID:                actionID,
		SourceDecisionRef:       current.DecisionRef,
		SourceAuthorityRevision: current.AuthorityRevision,
		ModelRevision:           current.ModelRevision,
	}
	grant, err := grantReader.ReadCurrentEffectGrant(ctx, query)
	// An injected Reader may ignore cancellation; never allow the Command
	// Adapter to proceed with that stale result.
	if err != nil || ctx.Err() != nil || grant.Query != query ||
		strings.TrimSpace(grant.GrantRef) == "" ||
		strings.TrimSpace(grant.GrantAuthorityRevision) == "" {
		return errHostEffectGrantUnavailable
	}
	return nil
}
