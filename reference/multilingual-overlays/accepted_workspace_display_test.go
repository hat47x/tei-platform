package multilingualoverlays

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/sourceacceptance"
	"tei/internal/sourceindex"
	"tei/internal/workspace"
)

type fixtureCurrentReader struct {
	record sourceacceptance.Record
	err    error
	calls  int
	after  func()
}

func (r *fixtureCurrentReader) ReadCurrentAcceptedSource(_ context.Context, _ string) (sourceacceptance.Record, error) {
	r.calls++
	if r.after != nil {
		r.after()
	}
	return r.record, r.err
}

func sourceAcceptedForTest(t *testing.T, root string) sourceacceptance.Record {
	t.Helper()
	loaded, err := workspace.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	idx, _, err := workspace.IndexedLoadedSnapshot(root, loaded)
	if err != nil {
		t.Fatal(err)
	}
	return sourceacceptance.Record{
		ApplicationID:     "order_app",
		DecisionRef:       "trusted-host-decision-test",
		AuthorityRevision: "accepted-r1",
		ModelRevision:     sourceindex.ModelRevision(idx),
		Artifacts:         append([]sourceindex.Artifact(nil), idx.Artifacts...),
	}
}

func TestO2cAcceptedSourceAndDisplayPolicyBothRequired(t *testing.T) {
	root, lock, cfg := makeSourceWorkspace(t)
	reader := &fixtureCurrentReader{record: sourceAcceptedForTest(t, root)}
	calls := 0
	m, witness, err := BuildAcceptedWorkspaceDisplay(
		context.Background(), reader, "order_app", root, lock, cfg,
		func(r AcceptedDisplayReview) error {
			calls++
			if r.Acceptance.DecisionRef != "trusted-host-decision-test" ||
				r.Acceptance.ModelRevision != reader.record.ModelRevision ||
				r.Display.Source.ArtifactPath != "semantic/sales/order.tei" ||
				r.Display.Source.ArtifactFingerprint == "" ||
				r.Display.Source.Original != "Order" ||
				r.Display.Environment != cfg.Environment ||
				r.Display.Surface.Target != cfg.Owner ||
				len(r.Display.Packs) != 1 || len(r.Display.Grants) != 1 {
				return errors.New("invalid accepted review")
			}
			return nil
		})
	if err != nil || reader.calls != 1 || calls != 1 ||
		m.Text != "注文" || witness.Original != "Order" ||
		m.LockFingerprint != witness.LockFingerprint {
		t.Fatalf("accepted Source and display policy not composed: m=%+v witness=%+v calls=%d/%d err=%v",
			m, witness, reader.calls, calls, err)
	}
}

func TestO2cMissingHostDependenciesFailBeforeReader(t *testing.T) {
	root, lock, cfg := makeSourceWorkspace(t)
	reader := &fixtureCurrentReader{record: sourceAcceptedForTest(t, root)}
	for _, tc := range []struct {
		name   string
		ctx    context.Context
		reader sourceacceptance.Reader
		gate   AcceptedDisplayGate
		want   string
	}{
		{"nil context", nil, reader, func(AcceptedDisplayReview) error { return nil }, "context-required"},
		{"nil reader", context.Background(), nil, func(AcceptedDisplayReview) error { return nil }, "host-reader-required"},
		{"nil gate", context.Background(), reader, nil, "host-display-gate-required"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m, s, err := BuildAcceptedWorkspaceDisplay(tc.ctx, tc.reader,
				"order_app", root, lock, cfg, tc.gate)
			if err == nil || !strings.Contains(err.Error(), tc.want) ||
				m.Kind != "" || s.Original != "" {
				t.Fatalf("%s not rejected: %+v %+v %v", tc.want, m, s, err)
			}
		})
	}
	if reader.calls != 0 {
		t.Fatalf("Host was consulted before required dependencies: %d", reader.calls)
	}
}

func TestO2cSourceAcceptanceFailuresNeverReachDisplayGate(t *testing.T) {
	for _, tc := range []struct {
		name string
		edit func(*testing.T, string, *fixtureCurrentReader, *string)
	}{
		{"wrong application", func(_ *testing.T, _ string, _ *fixtureCurrentReader, app *string) {
			*app = "another_application"
		}},
		{"unaccepted source", func(_ *testing.T, _ string, r *fixtureCurrentReader, _ *string) {
			r.record.DecisionRef = ""
		}},
		{"revoked reader", func(_ *testing.T, _ string, r *fixtureCurrentReader, _ *string) {
			r.err = errors.New("private-host-token-cannot-leak")
		}},
		{"stale Source revision", func(t *testing.T, root string, _ *fixtureCurrentReader, _ *string) {
			p := filepath.Join(root, "semantic", "sales", "order.tei")
			if err := os.WriteFile(p, []byte(referenceSemantic+"\n# edit after approval\n"), 0600); err != nil {
				t.Fatal(err)
			}
		}},
		{"changed managed membership", func(t *testing.T, root string, _ *fixtureCurrentReader, _ *string) {
			p := filepath.Join(root, "semantic", "extra.tei")
			if err := os.WriteFile(p, []byte("namespace extra\n\ndata Extra {\n    attribute extraId type std.identifier\n}\n"), 0600); err != nil {
				t.Fatal(err)
			}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root, lock, cfg := makeSourceWorkspace(t)
			r := &fixtureCurrentReader{record: sourceAcceptedForTest(t, root)}
			app := "order_app"
			tc.edit(t, root, r, &app)
			called := false
			m, s, err := BuildAcceptedWorkspaceDisplay(context.Background(), r,
				app, root, lock, cfg, func(AcceptedDisplayReview) error {
					called = true
					return nil
				})
			if err == nil || !strings.Contains(err.Error(), "current-acceptance-verification-failed") ||
				called || m.Kind != "" || s.Original != "" ||
				strings.Contains(err.Error(), "private-host-token") {
				t.Fatalf("invalid acceptance reached display gate: m=%+v source=%+v gate=%v error=%v", m, s, called, err)
			}
		})
	}
}

func TestO2cReviewRevocationAndLockInvalidation(t *testing.T) {
	root, lock, cfg := makeSourceWorkspace(t)
	reader := &fixtureCurrentReader{record: sourceAcceptedForTest(t, root)}
	denied := false
	_, _, err := BuildAcceptedWorkspaceDisplay(context.Background(), reader, "order_app",
		root, lock, cfg, func(AcceptedDisplayReview) error {
			denied = true
			return errors.New("locale grant revoked")
		})
	if !denied || err == nil || !strings.Contains(err.Error(), "review-denied") {
		t.Fatalf("display grant revocation ignored: %v", err)
	}
	cfg.Grants = nil
	m, s, err := BuildAcceptedWorkspaceDisplay(context.Background(), reader, "order_app",
		root, lock, cfg, func(AcceptedDisplayReview) error { return nil })
	if err == nil || !strings.Contains(err.Error(), "overlay-not-permitted") ||
		m.Kind != "" || s.Original != "" {
		t.Fatalf("plugin overlay used without exact environment grant: %v", err)
	}
	cfg.Grants = []Grant{{
		Environment: "development", Provider: cfg.Packs[0].Provider,
		Target: cfg.Owner, Key: cfg.Packs[0].Key, Locale: "ja-JP",
	}}
	lock.Environments[0].Environment = "foreign"
	reviewCalled := false
	_, _, err = BuildAcceptedWorkspaceDisplay(context.Background(), reader, "order_app",
		root, lock, cfg, func(AcceptedDisplayReview) error {
			reviewCalled = true
			return nil
		})
	if err == nil || !strings.Contains(err.Error(), "plugin-lock") || reviewCalled {
		t.Fatalf("foreign lock reached display review: %v", err)
	}
}

func TestO2cCancellationClosesBeforeAndAfterHostGate(t *testing.T) {
	root, lock, cfg := makeSourceWorkspace(t)
	t.Run("already cancelled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		reader := &fixtureCurrentReader{record: sourceAcceptedForTest(t, root)}
		_, _, err := BuildAcceptedWorkspaceDisplay(ctx, reader, "order_app", root,
			lock, cfg, func(AcceptedDisplayReview) error { return nil })
		if err == nil || !strings.Contains(err.Error(), "context-cancelled") || reader.calls != 0 {
			t.Fatalf("pre-cancelled call used Host: %v", err)
		}
	})
	t.Run("cancelled in reader", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		reader := &fixtureCurrentReader{record: sourceAcceptedForTest(t, root)}
		reader.after = cancel
		called := false
		_, _, err := BuildAcceptedWorkspaceDisplay(ctx, reader, "order_app", root,
			lock, cfg, func(AcceptedDisplayReview) error {
				called = true
				return nil
			})
		if err == nil || called {
			t.Fatalf("cancelled Host read reached display gate: %v", err)
		}
	})
	t.Run("cancelled in gate", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		reader := &fixtureCurrentReader{record: sourceAcceptedForTest(t, root)}
		_, _, err := BuildAcceptedWorkspaceDisplay(ctx, reader, "order_app",
			root, lock, cfg, func(AcceptedDisplayReview) error {
				cancel()
				return nil
			})
		if err == nil || !strings.Contains(err.Error(), "context-cancelled") {
			t.Fatalf("gate cancelled but returned a display: %v", err)
		}
	})
}

func TestO2cSingleParsedSourceSnapshotAcrossHostCheck(t *testing.T) {
	root, lock, cfg := makeSourceWorkspace(t)
	reader := &fixtureCurrentReader{record: sourceAcceptedForTest(t, root)}
	reader.after = func() {
		// A concurrent edit after Load must not make the reviewed display
		// silently switch to a second Workspace Load. This does not prove
		// the filesystem remains current; the Host must recheck at dispatch.
		p := filepath.Join(root, "semantic", "sales", "order.tei")
		if err := os.WriteFile(p, []byte(referenceSemantic+"\n# later edit\n"), 0600); err != nil {
			t.Error(err)
		}
	}
	m, witness, err := BuildAcceptedWorkspaceDisplay(context.Background(), reader,
		"order_app", root, lock, cfg, func(review AcceptedDisplayReview) error {
			if review.Display.Source.ArtifactFingerprint !=
				findTestArtifactFingerprint(reader.record, "semantic/sales/order.tei") {
				return errors.New("different parsed Source after Host verification")
			}
			return nil
		})
	if err != nil || m.Text != "注文" ||
		witness.ArtifactFingerprint != findTestArtifactFingerprint(reader.record, "semantic/sales/order.tei") {
		t.Fatalf("model read twice or changed after verification: %+v %+v %v", m, witness, err)
	}
}

func findTestArtifactFingerprint(record sourceacceptance.Record, path string) string {
	for _, a := range record.Artifacts {
		if a.Path == path {
			return a.Fingerprint
		}
	}
	return ""
}
