//go:build !tei_order_only && !tei_repair_only

package main

import (
	"context"
	"errors"
	"strings"
	"testing"

	"tei/internal/sourceacceptance"
	"tei/internal/sourceindex"
)

// This grants ONLY synthetic test admission. The real Host must inject its
// own current, authenticated issuer and revocation source.
type syntheticEffectReader struct {
	current bool
	calls   int
	err     error
	tamper  func(*HostEffectGrantRecord)
	onRead  func()
}

func (r *syntheticEffectReader) ReadCurrentEffectGrant(ctx context.Context, query HostEffectGrantQuery) (HostEffectGrantRecord, error) {
	r.calls++
	if r.onRead != nil {
		r.onRead()
	}
	if r.err != nil {
		return HostEffectGrantRecord{}, r.err
	}
	if !r.current {
		return HostEffectGrantRecord{}, errors.New("test-only grant revoked")
	}
	record := HostEffectGrantRecord{
		Query: query, GrantRef: "test-only/no-host-effect-permission",
		GrantAuthorityRevision: "test-only-no-issuer",
	}
	if r.tamper != nil {
		r.tamper(&record)
	}
	return record, nil
}

func TestCurrentHostEffectGrantRechecksSourceAndExactSubject(t *testing.T) {
	member, err := sourceindex.NewArtifact("applications/order-entry/application.tei", []byte("application order_entry {}\n"))
	if err != nil {
		t.Fatal(err)
	}
	index := sourceindex.Index{Artifacts: []sourceindex.Artifact{member}}
	accepted := sourceacceptance.Record{
		ApplicationID: "order_entry",
		DecisionRef: "test-only-source-decision",
		AuthorityRevision: "test-only-source-authority",
		ModelRevision: sourceindex.ModelRevision(index),
		Artifacts: index.Artifacts,
	}
	source := &requestSourceReader{record: accepted}
	grants := &syntheticEffectReader{current: true}
	check := func(ctx context.Context, effect HostEffectGrantReader) error {
		return requireCurrentHostEffectGrant(ctx, source, effect,
			"order_entry", "test-only-logged-in-subject", "order_entry.submit_order", index, accepted)
	}
	if err := check(context.Background(), grants); err != nil ||
		source.calls != 1 || grants.calls != 1 {
		t.Fatalf("exact current Source and Grant should pass only test admission: %v, Source=%d, Grant=%d", err, source.calls, grants.calls)
	}

	grants.current = false
	if err := check(context.Background(), grants); !errors.Is(err, errHostEffectGrantUnavailable) {
		t.Fatal("revoked effect Grant was admitted")
	}
	grants.current = true
	grants.tamper = func(r *HostEffectGrantRecord) { r.Query.SubjectID = "another-user" }
	if err := check(context.Background(), grants); !errors.Is(err, errHostEffectGrantUnavailable) {
		t.Fatal("Grant for another authenticated subject was admitted")
	}
	grants.tamper = func(r *HostEffectGrantRecord) { r.Query.ActionID = "other-action" }
	if err := check(context.Background(), grants); !errors.Is(err, errHostEffectGrantUnavailable) {
		t.Fatal("Grant for another action was admitted")
	}
	grants.tamper = func(r *HostEffectGrantRecord) { r.Query.ModelRevision = sourceindex.Fingerprint([]byte("different")) }
	if err := check(context.Background(), grants); !errors.Is(err, errHostEffectGrantUnavailable) {
		t.Fatal("Grant for another Source revision was admitted")
	}
	grants.tamper = func(r *HostEffectGrantRecord) { r.GrantAuthorityRevision = "" }
	if err := check(context.Background(), grants); !errors.Is(err, errHostEffectGrantUnavailable) {
		t.Fatal("Grant without Host authority provenance was admitted")
	}
	grants.tamper = nil
	grants.err = errors.New("Host issuer secret-token=SHOULD_NOT_LEAK")
	if err := check(context.Background(), grants); !errors.Is(err, errHostEffectGrantUnavailable) ||
		strings.Contains(err.Error(), "SHOULD_NOT_LEAK") {
		t.Fatalf("private Host effect error must be redacted: %v", err)
	}
	grants.err = nil
	callsBefore := grants.calls
	source.record.AuthorityRevision = "newer-source-decision"
	if err := check(context.Background(), grants); !errors.Is(err, errHostEffectGrantUnavailable) ||
		grants.calls != callsBefore {
		t.Fatal("stale Source authority reached effect Grant reader")
	}
	source.record = accepted
	source.record.Artifacts = append([]sourceindex.Artifact(nil), accepted.Artifacts...)
	source.record.Artifacts[0].Fingerprint = sourceindex.Fingerprint([]byte("application order_entry {}\n\n"))
	if err := check(context.Background(), grants); !errors.Is(err, errHostEffectGrantUnavailable) ||
		grants.calls != callsBefore {
		t.Fatal("same-meaning R2 reached effect Grant reader")
	}
	source.record = accepted
	if err := check(context.Background(), nil); !errors.Is(err, errHostEffectGrantUnavailable) ||
		grants.calls != callsBefore {
		t.Fatal("missing independent Host effect Grant was admitted")
	}
	if err := check(context.Background(), grants); err != nil {
		t.Fatalf("test Host original R1 and Grant were not recovered: %v", err)
	}
}

func TestCurrentHostEffectGrantContextRefusal(t *testing.T) {
	member, _ := sourceindex.NewArtifact("one.tei", []byte("x\n"))
	index := sourceindex.Index{Artifacts: []sourceindex.Artifact{member}}
	accepted := sourceacceptance.Record{
		ApplicationID: "order_entry", DecisionRef: "synthetic",
		AuthorityRevision: "synthetic", ModelRevision: sourceindex.ModelRevision(index),
		Artifacts: index.Artifacts,
	}
	source := &requestSourceReader{record: accepted}
	grants := &syntheticEffectReader{current: true}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err := requireCurrentHostEffectGrant(ctx, source, grants, "order_entry", "subject", "action", index, accepted)
	if !errors.Is(err, errHostEffectGrantUnavailable) || source.calls != 0 || grants.calls != 0 {
		t.Fatal("cancelled context queried Source/Grant or admitted effect")
	}
	ctx, cancel = context.WithCancel(context.Background())
	grants.onRead = cancel
	err = requireCurrentHostEffectGrant(ctx, source, grants, "order_entry", "subject", "action", index, accepted)
	if !errors.Is(err, errHostEffectGrantUnavailable) || grants.calls != 1 {
		t.Fatal("Host grant reader ignored cancellation at final effect boundary")
	}
}
