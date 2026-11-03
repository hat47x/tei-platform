//go:build !tei_order_only && !tei_repair_only

package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"tei/internal/ir"
	"tei/internal/security/effectcommit"
	"tei/internal/security/legacyauthorization"
	"tei/internal/security/provisionaleffectgrant"
)

// This file wires the DEVELOPMENT- AND VERIFICATION-ONLY provisional effect
// Grant (provisional owner decision D8, tools/reference-host-effect-grant) into
// the Web Target. It is NOT a production permission issuer:
//
//   - It is selected only when all three flags are given together
//     (-provisional-effect-grant, -provisional-effect-grant-trust and the
//     explicit -provisional-effect-grant-development-only). Without them the
//     shipped CLI is unchanged: no effect Grant Reader exists and accepted
//     Commands are refused before dispatch.
//   - It cannot be combined with an embedding Host's own Reader or with the
//     unaccepted-Source diagnostic.
//   - Every effect re-reads and re-verifies the signed Grant and the trust file
//     (revocation, key window, Grant period) for the exact authenticated subject
//     and resource. A failure is the same redacted errHostEffectGrantUnavailable.
//   - Evidence built from it carries effectcommit.ProvenanceProvisionalGrant and
//     is never counted for G2.

const provisionalCommandCapability = "order_entry.submit_order"

type provisionalEffectGrantFlags struct {
	grantPath       string
	trustPath       string
	developmentOnly bool
}

func (f provisionalEffectGrantFlags) any() bool {
	return f.grantPath != "" || f.trustPath != "" || f.developmentOnly
}

// provisionalEffectGrantReader adapts FileSource to HostEffectGrantReader.
type provisionalEffectGrantReader struct {
	source provisionaleffectgrant.FileSource
	notice string
}

// announce prints the development-only warning. The caller invokes it only
// after Source verification and every startup gate have passed, immediately
// before the Target is published, so a start that fails never claims to be active.
func (r provisionalEffectGrantReader) announce() {
	log.Print(r.notice)
}

func (r provisionalEffectGrantReader) ReadCurrentEffectGrant(ctx context.Context, q HostEffectGrantQuery) (HostEffectGrantRecord, error) {
	v, err := r.source.Read(ctx, provisionaleffectgrant.Query{
		ApplicationID: q.ApplicationID, Capability: q.ActionID, SubjectID: q.SubjectID, Resource: q.ResourceID,
	})
	if err != nil {
		// Fixed codes only; never file content. The caller still redacts.
		log.Printf("PROVISIONAL_EFFECT_GRANT_REFUSED: %v", err)
		return HostEffectGrantRecord{}, errHostEffectGrantUnavailable
	}
	e := v.Grant.Epochs
	if e == nil {
		return HostEffectGrantRecord{}, errHostEffectGrantUnavailable
	}
	return HostEffectGrantRecord{
		Query:                  q,
		GrantRef:               v.GrantRef,
		GrantAuthorityRevision: fmt.Sprintf("%s/epochs:%s/%d/%d/%d", provisionaleffectgrant.MarkProvisional, e.Source, e.SubjectLink, e.Membership, e.Policy),
	}, nil
}

// selectProvisionalEffectGrant returns the Reader selected by the flags, or nil
// when none of them is set. Any partial, conflicting or invalid selection is an
// error, and it verifies the Grant once at startup so a bad file fails early.
func selectProvisionalEffectGrant(f provisionalEffectGrantFlags, applicationID string, embedded HostEffectGrantReader, diagnostic bool, now func() time.Time) (HostEffectGrantReader, error) {
	if !f.any() {
		return nil, nil
	}
	if f.grantPath == "" || f.trustPath == "" || !f.developmentOnly {
		return nil, errors.New("PROVISIONAL_EFFECT_GRANT_FLAGS: -provisional-effect-grant, -provisional-effect-grant-trust and -provisional-effect-grant-development-only are required together")
	}
	if embedded != nil {
		return nil, errors.New("PROVISIONAL_EFFECT_GRANT_CONFLICT: the embedding Host already supplied an effect Grant Reader")
	}
	if diagnostic {
		return nil, errors.New("SOURCE_ACCEPTANCE_DIAGNOSTIC_CONFLICT: Host Readers cannot be bypassed")
	}
	source := provisionaleffectgrant.FileSource{GrantPath: f.grantPath, TrustPath: f.trustPath, Now: now}
	// The startup probe asks for the Grant's own binding: it proves the files
	// verify now, not that any particular request will be allowed.
	raw, err := readSmallFile(f.grantPath)
	if err != nil {
		return nil, errors.New("PROVISIONAL_GRANT_UNREADABLE")
	}
	g, err := provisionaleffectgrant.Parse(raw)
	if err != nil {
		return nil, err
	}
	if g.ApplicationID != applicationID || g.Capability != provisionalCommandCapability {
		return nil, errors.New("PROVISIONAL_EFFECT_GRANT_BINDING: the Grant is not for this Application's submit Command")
	}
	v, err := source.Read(context.Background(), provisionaleffectgrant.Query{
		ApplicationID: g.ApplicationID, Capability: g.Capability, SubjectID: g.SubjectID, Resource: g.Resource,
	})
	if err != nil {
		return nil, err
	}
	notice := fmt.Sprintf("PROVISIONAL_EFFECT_GRANT_ACTIVE: development-only (D8), not a production permission; application=%s capability=%s notAfter=%s; Evidence is marked %q and is not G2 evidence",
		g.ApplicationID, g.Capability, v.NotAfter.Format(time.RFC3339), provisionaleffectgrant.MarkProvisional)
	return provisionalEffectGrantReader{source: source, notice: notice}, nil
}

func readSmallFile(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return io.ReadAll(io.LimitReader(f, 64<<10+1))
}

// newProvisionalSubmitGrantSource gives an embedding Host the Grant to hand to
// the effectcommit.Runner for one order, re-verified at each call. subjectID is
// the authenticated subject the Host has already established for the request.
func newProvisionalSubmitGrantSource(source provisionaleffectgrant.FileSource, applicationID, subjectID string) submitGrantSource {
	return func(ctx context.Context, orderID int64) (legacyauthorization.AuthorizationGrant, error) {
		v, err := source.Read(ctx, provisionaleffectgrant.Query{
			ApplicationID: applicationID, Capability: provisionalCommandCapability,
			SubjectID: subjectID, Resource: fmt.Sprintf("order/%d", orderID),
		})
		if err != nil {
			return legacyauthorization.AuthorizationGrant{}, errHostEffectGrantUnavailable
		}
		return v.Grant, nil
	}
}

// provisionalAuditConfig is the only Audit setting a store fed by a provisional
// Grant may use: the Provenance is fixed to provisional-grant.
func provisionalAuditConfig(sink effectcommit.EvidenceSink) effectcommit.AuditConfig {
	return effectcommit.AuditConfig{Sink: sink, Provenance: effectcommit.ProvenanceProvisionalGrant}
}

// newProvisionalRunnerOrderSubmitStore is the ONLY way this package builds a
// store fed by a provisional Grant: it fixes the Runner's Evidence Provenance to
// provisional-grant, so a Host cannot pair a provisional Grant with real-pg,
// synthetic or unspecified Provenance (a test checks that nothing else calls
// newProvisionalSubmitGrantSource).
func newProvisionalRunnerOrderSubmitStore(
	provider effectcommit.Provider,
	source provisionaleffectgrant.FileSource,
	applicationID, subjectID string,
	sink effectcommit.EvidenceSink,
	schemaCheck func(context.Context) error,
	status orderStatusReader,
) (*runnerOrderSubmitStore, error) {
	if provider == nil || sink == nil {
		return nil, errors.New("provisional runner order submit store requires a provider and an Evidence sink")
	}
	return newRunnerOrderSubmitStore(
		effectcommit.Runner{Provider: provider, Audit: provisionalAuditConfig(sink)},
		newProvisionalSubmitGrantSource(source, applicationID, subjectID),
		schemaCheck, status)
}

// newPostgresProvisionalRunnerOrderSubmitStore is the PostgreSQL form of
// newProvisionalRunnerOrderSubmitStore (same fixed Provenance).
func newPostgresProvisionalRunnerOrderSubmitStore(
	ctx context.Context,
	writeDSN, readDSN, effectSchema string,
	world ir.World,
	source provisionaleffectgrant.FileSource,
	applicationID, subjectID string,
	sink effectcommit.EvidenceSink,
) (*runnerOrderSubmitStore, error) {
	if sink == nil {
		return nil, errors.New("provisional runner order submit store requires an Evidence sink")
	}
	return newPostgresRunnerOrderSubmitStore(ctx, writeDSN, readDSN, effectSchema, world,
		newProvisionalSubmitGrantSource(source, applicationID, subjectID),
		withRunnerAudit(provisionalAuditConfig(sink)))
}
