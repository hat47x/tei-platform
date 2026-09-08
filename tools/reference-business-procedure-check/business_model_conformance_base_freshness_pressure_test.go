package main

import (
	"fmt"
	"path/filepath"
	"testing"

	"tei/internal/application"
	"tei/internal/workspace"
)

type businessAcceptancePreviewBaseProbe struct {
	Sources         businessSourceFreshnessSnapshotProbe
	ConformanceBase string
}

func TestBusinessAcceptancePreviewBaseRequiresBothBusinessSourcesAndWorldToRemainFresh(t *testing.T) {
	root := repoPath()
	bundle := currentBusinessAuthoringBundleProbe(t, root)
	sources, err := businessSourceFreshnessSnapshotForBundleProbe(bundle)
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := workspace.Load(filepath.Join(root, "reference", "workspace"))
	if err != nil {
		t.Fatal(err)
	}
	worldFingerprint, err := jsonContentFingerprintProbe(loaded.World)
	if err != nil {
		t.Fatal(err)
	}
	preview := businessAcceptancePreviewBaseProbe{
		Sources:         sources,
		ConformanceBase: worldFingerprint,
	}

	if err := requireBusinessAcceptancePreviewBaseProbe(preview, sources, worldFingerprint); err != nil {
		t.Fatalf("unchanged Business sources / Worldをstaleと判定しました: %v", err)
	}

	changedWorld := loaded.World
	changedWorld.Applications = append([]application.Application(nil), loaded.World.Applications...)
	removed := false
	for i := range changedWorld.Applications {
		if string(changedWorld.Applications[i].ID) != "order_entry" {
			continue
		}
		original := changedWorld.Applications[i].Patterns
		patterns := make([]application.Pattern, 0, len(original))
		for _, pattern := range original {
			if string(pattern.ID) == "order_entry.submit_order" {
				removed = true
				continue
			}
			patterns = append(patterns, pattern)
		}
		changedWorld.Applications[i].Patterns = patterns
	}
	if !removed {
		t.Fatal("test setup: order_entry.submit_order Patternがありません")
	}
	changedWorldFingerprint, err := jsonContentFingerprintProbe(changedWorld)
	if err != nil {
		t.Fatal(err)
	}
	if preview.ConformanceBase == changedWorldFingerprint {
		t.Fatal("World meaning changeをConformance base fingerprintが見逃しました")
	}

	// Business authoring sourceが一切変化していなくても、Preview時に
	// cross-view validationしたWorldが変わった場合はAccept前提がstaleである。
	if err := requireBusinessAcceptancePreviewBaseProbe(preview, sources, changedWorldFingerprint); err == nil {
		t.Fatal("Business sourcesがfreshというだけでstale WorldへAcceptできてしまいます")
	}
}

func TestBusinessAcceptancePreviewBaseStillRejectsBusinessSourceChangeWhenWorldIsFresh(t *testing.T) {
	root := repoPath()
	beforeBundle := currentBusinessAuthoringBundleProbe(t, root)
	beforeSources, err := businessSourceFreshnessSnapshotForBundleProbe(beforeBundle)
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := workspace.Load(filepath.Join(root, "reference", "workspace"))
	if err != nil {
		t.Fatal(err)
	}
	worldFingerprint, err := jsonContentFingerprintProbe(loaded.World)
	if err != nil {
		t.Fatal(err)
	}
	preview := businessAcceptancePreviewBaseProbe{Sources: beforeSources, ConformanceBase: worldFingerprint}

	afterBundle := currentBusinessAuthoringBundleProbe(t, root)
	changed := false
	for p := range afterBundle.Procedures {
		if afterBundle.Procedures[p].Procedure.ID != "order-entry.order-handling" {
			continue
		}
		for s := range afterBundle.Procedures[p].Steps {
			if afterBundle.Procedures[p].Steps[s].ID == "BP-04" {
				afterBundle.Procedures[p].Steps[s].BusinessAction = "changed after preview while World stayed the same"
				changed = true
			}
		}
	}
	if !changed {
		t.Fatal("test setup: BP-04がありません")
	}
	afterSources, err := businessSourceFreshnessSnapshotForBundleProbe(afterBundle)
	if err != nil {
		t.Fatal(err)
	}
	if err := requireBusinessAcceptancePreviewBaseProbe(preview, afterSources, worldFingerprint); err == nil {
		t.Fatal("Worldがfreshというだけでstale Business sourceをAcceptできてしまいます")
	}
}

func requireBusinessAcceptancePreviewBaseProbe(
	preview businessAcceptancePreviewBaseProbe,
	currentSources businessSourceFreshnessSnapshotProbe,
	currentConformanceBase string,
) error {
	if err := requireBusinessSourceFreshnessProbe(preview.Sources, currentSources); err != nil {
		return err
	}
	if preview.ConformanceBase != currentConformanceBase {
		return fmt.Errorf("cross-view Conformance base changed after preview")
	}
	return nil
}
