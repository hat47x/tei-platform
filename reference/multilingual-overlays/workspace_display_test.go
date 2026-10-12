package multilingualoverlays

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/core"
	"tei/internal/environment"
	"tei/internal/plugin"
	"tei/internal/modelref"
	"tei/internal/subject"
	"tei/internal/workspace"
)

const referenceSemantic = "namespace sales\n\ndata Order {\n    attribute orderId type std.identifier\n}\n"

func makeSourceWorkspace(t *testing.T) (string, plugin.Lock, WorkspaceDisplayConfig) {
	t.Helper()
	root := t.TempDir()
	files := map[string]string{
		"tei.workspace.yaml": "workspace: test\nsources:\n  - plugins\n  - semantic\n  - environments\n",
		"plugins/tei-order-ui/1.0.0/plugin.tei": "plugin tei.order.ui v1.0.0 {\n    runtime builtin\n}\n",
		"plugins/tei-locale-ja/1.0.0/plugin.tei": "plugin tei.locale.ja v1.0.0 {\n    runtime builtin\n}\n",
		"semantic/sales/order.tei": referenceSemantic,
		"environments/development/environment.tei": "environment development {\n    plugin tei.order.ui ^1\n    plugin tei.locale.ja ^1\n}\n",
	}
	for name, content := range files {
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	ordinary := plugin.Identity{ID: core.CanonicalID("tei.order.ui"), Version: "1.0.0"}
	localizer := plugin.Identity{ID: core.CanonicalID("tei.locale.ja"), Version: "1.0.0"}
	// This test uses a lock from the actual parser and existing resolver.
	loaded, err := workspaceForTest(root)
	if err != nil {
		t.Fatal(err)
	}
	env := loaded.World.Environments[0]
	lock, err := plugin.BuildLock(plugin.Catalog{Plugins: loaded.World.Plugins, Bundles: loaded.World.Bundles}, []environment.Environment{env})
	if err != nil {
		t.Fatal(err)
	}
	cfg := WorkspaceDisplayConfig{
		Environment: core.CanonicalID("development"),
		Scope: "workspace.sales",
		Data: core.CanonicalID("sales.order"),
		Owner: ordinary,
		SourceLocale: "en-US",
		Locale: "ja-JP",
	}
	key := resourceKey(Surface{
		Target: ordinary,
		Subject: typedDataKey(cfg),
		Field: "display-name",
	})
	cfg.Packs = []Pack{{
		Provider: localizer, Target: ordinary, Key: key, Locale: "ja-JP",
		Text: "注文", BasedOn: Fingerprint("Order"), Review: "confirmed",
	}}
	cfg.Grants = []Grant{{
		Environment: "development", Provider: localizer, Target: ordinary,
		Key: key, Locale: "ja-JP",
	}}
	return root, lock, cfg
}

// The fixture does not interpret source content: production Workspace.Load
// supplies the same World used to create the pinned Plugin Lock.
func workspaceForTest(root string) (workspace.Result, error) {
	return workspace.Load(root)
}

func typedDataKey(cfg WorkspaceDisplayConfig) modelref.Key {
	return modelref.Key{Scope: cfg.Scope, Subject: subject.Subject{Kind: "semantic-data", ID: string(cfg.Data)}}
}

func TestO2bWorkspaceLoaderSourceIndexAndHostGate(t *testing.T) {
	root, lock, cfg := makeSourceWorkspace(t)
	calls := 0
	gate := func(review ReviewRequest) error {
		calls++
		if review.Environment != "development" || review.Source.Original != "Order" ||
			review.Source.Subject != typedDataKey(cfg) ||
			review.Source.ArtifactPath != "semantic/sales/order.tei" ||
			review.Source.ArtifactFingerprint == "" ||
			review.Source.OriginalFingerprint != Fingerprint("Order") ||
			review.Source.LockFingerprint == "" ||
			review.Surface.Target != cfg.Owner ||
			len(review.Packs) != 1 || review.Packs[0].Text != "注文" ||
			len(review.Grants) != 1 || review.Requested.Locale != "ja-JP" {
			return errors.New("unexpected review input")
		}
		return nil
	}
	manifest, witness, err := BuildWorkspaceDisplay(root, lock, cfg, gate)
	if err != nil {
		t.Fatal(err)
	}
	if calls != 1 || witness.ArtifactPath != "semantic/sales/order.tei" ||
		manifest.Locale != "ja-JP" || manifest.Text != "注文" ||
		manifest.LockFingerprint != witness.LockFingerprint ||
		manifest.SourceRevision != Fingerprint("Order") {
		t.Fatalf("source-bound result: %+v %+v calls=%d", manifest, witness, calls)
	}
	// Removing the localization plugin is not a requirement on the ordinary
	// semantic Data; the default label remains present in the parsed model.
	cfg.Packs, cfg.Grants, cfg.Locale = nil, nil, "en-US"
	original, _, err := BuildWorkspaceDisplay(root, lock, cfg, gateForSource("Order"))
	if err != nil || original.Text != "Order" || !original.FromSource {
		t.Fatalf("source language damaged: %+v %v", original, err)
	}
}

func gateForSource(label string) ReviewGate {
	return func(input ReviewRequest) error {
		if input.Source.Original != label {
			return errors.New("unexpected source label")
		}
		return nil
	}
}

func TestO2bNoGateNeverUsesParsedWorkspaceAsAcceptance(t *testing.T) {
	root, lock, cfg := makeSourceWorkspace(t)
	display, source, err := BuildWorkspaceDisplay(root, lock, cfg, nil)
	if err == nil || !strings.Contains(err.Error(), "review-gate-required") ||
		display.Kind != "" || source.Original != "" {
		t.Fatalf("unreviewed source was accepted: %+v %+v %v", display, source, err)
	}
}

func TestO2bDeniedOrRevokedReviewBlocksDisplay(t *testing.T) {
	root, lock, cfg := makeSourceWorkspace(t)
	reviewCalled := false
	_, _, err := BuildWorkspaceDisplay(root, lock, cfg, func(ReviewRequest) error {
		reviewCalled = true
		return errors.New("authorization revoked")
	})
	if !reviewCalled || err == nil || !strings.Contains(err.Error(), "review-denied") {
		t.Fatalf("revoked grant was ignored: %v", err)
	}
}

func TestO2bSourceEditRequiresFreshAcceptanceForNewFingerprint(t *testing.T) {
	root, lock, cfg := makeSourceWorkspace(t)
	var baseline SourceWitness
	_, _, err := BuildWorkspaceDisplay(root, lock, cfg, func(r ReviewRequest) error {
		baseline = r.Source
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "semantic", "sales", "order.tei")
	if err := os.WriteFile(path, []byte(referenceSemantic+"\n# Reviewed later\n"), 0600); err != nil {
		t.Fatal(err)
	}
	_, witness, err := BuildWorkspaceDisplay(root, lock, cfg, func(r ReviewRequest) error {
		if r.Source.ArtifactFingerprint != baseline.ArtifactFingerprint {
			return errors.New("source changed since acceptance")
		}
		return nil
	})
	if err == nil || !strings.Contains(err.Error(), "review-denied") || witness.Original != "" {
		t.Fatalf("stale review reused after source change: %+v, %v", witness, err)
	}
	// Approved current content can be used again without changing the Data ID.
	manifest, next, err := BuildWorkspaceDisplay(root, lock, cfg, gateForSource("Order"))
	if err != nil || manifest.Text != "注文" ||
		next.ArtifactFingerprint == baseline.ArtifactFingerprint {
		t.Fatalf("fresh source index not observed: %+v %+v %v", manifest, next, err)
	}
}

func TestO2bRejectsDifferentEnvironmentAndUnselectedLocalization(t *testing.T) {
	root, lock, cfg := makeSourceWorkspace(t)
	cfg.Environment = "production"
	if _, _, err := BuildWorkspaceDisplay(root, lock, cfg, gateForSource("Order")); err == nil ||
		!strings.Contains(err.Error(), "unknown environment") {
		t.Fatalf("unknown environment was used: %v", err)
	}
	cfg.Environment = "development"
	cfg.Grants = nil
	if _, _, err := BuildWorkspaceDisplay(root, lock, cfg, gateForSource("Order")); err == nil ||
		!strings.Contains(err.Error(), "overlay-not-permitted") {
		t.Fatalf("unapproved overlay used: %v", err)
	}
}

func TestO2bDoesNotInventUnindexedSemanticData(t *testing.T) {
	root, lock, cfg := makeSourceWorkspace(t)
	cfg.Data = "sales.not_present"
	gateCalled := false
	_, _, err := BuildWorkspaceDisplay(root, lock, cfg, func(ReviewRequest) error {
		gateCalled = true
		return nil
	})
	if gateCalled || err == nil || !strings.Contains(err.Error(), "data-not-in-typed-index") {
		t.Fatalf("unknown meaning became a display: %v", err)
	}
}

func TestO2bCannotReuseForeignWorkspaceLock(t *testing.T) {
	root, lock, cfg := makeSourceWorkspace(t)
	lock.Environments[0].Environment = "production"
	gateCalled := false
	_, _, err := BuildWorkspaceDisplay(root, lock, cfg, func(ReviewRequest) error {
		gateCalled = true
		return nil
	})
	if gateCalled || err == nil || !strings.Contains(err.Error(), "plugin-lock") {
		t.Fatalf("unlocked plugin set was accepted: %v", err)
	}
}

func TestO2bMalformedSourceFailsBeforeHostReview(t *testing.T) {
	root, lock, cfg := makeSourceWorkspace(t)
	file := filepath.Join(root, "semantic", "sales", "order.tei")
	if err := os.WriteFile(file, []byte("garbage no data"), 0600); err != nil {
		t.Fatal(err)
	}
	called := false
	_, _, err := BuildWorkspaceDisplay(root, lock, cfg, func(ReviewRequest) error {
		called = true
		return nil
	})
	if err == nil || called {
		t.Fatalf("malformed source promoted into review: %v", err)
	}
}
