package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"testing"
)

type businessSourceFreshnessSnapshotProbe struct {
	Sources []businessSourceFreshnessEntryProbe `json:"sources"`
}

type businessSourceFreshnessEntryProbe struct {
	LogicalKey  string `json:"logicalKey"`
	Fingerprint string `json:"fingerprint"`
}

func TestBusinessSourceFreshnessSnapshotDetectsAnyAcceptedSourceChange(t *testing.T) {
	root := repoPath()
	baselineBundle := currentBusinessAuthoringBundleProbe(t, root)
	baseline, err := businessSourceFreshnessSnapshotForBundleProbe(baselineBundle)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("procedure wording", func(t *testing.T) {
		bundle := currentBusinessAuthoringBundleProbe(t, root)
		changed := false
		for p := range bundle.Procedures {
			if bundle.Procedures[p].Procedure.ID != "order-entry.order-handling" {
				continue
			}
			for s := range bundle.Procedures[p].Steps {
				if bundle.Procedures[p].Steps[s].ID == "BP-04" {
					bundle.Procedures[p].Steps[s].BusinessAction = "changed after preview"
					changed = true
				}
			}
		}
		if !changed {
			t.Fatal("test setup: BP-04がありません")
		}
		current, err := businessSourceFreshnessSnapshotForBundleProbe(bundle)
		if err != nil {
			t.Fatal(err)
		}
		if err := requireBusinessSourceFreshnessProbe(baseline, current); err == nil {
			t.Fatal("Procedure source changeをstaleとして検出できません")
		}
	})

	t.Run("protected target", func(t *testing.T) {
		bundle := currentBusinessAuthoringBundleProbe(t, root)
		changed := false
		for p := range bundle.Procedures {
			if bundle.Procedures[p].Procedure.ID != "order-entry.order-handling" {
				continue
			}
			for s := range bundle.Procedures[p].Steps {
				if bundle.Procedures[p].Steps[s].ID == "BP-06" {
					bundle.Procedures[p].Steps[s].ProtectedApplicationPattern = "order_entry.order_record"
					changed = true
				}
			}
		}
		if !changed {
			t.Fatal("test setup: BP-06がありません")
		}
		current, err := businessSourceFreshnessSnapshotForBundleProbe(bundle)
		if err != nil {
			t.Fatal(err)
		}
		if err := requireBusinessSourceFreshnessProbe(baseline, current); err == nil {
			t.Fatal("inline O1 source changeをstaleとして検出できません")
		}
	})

	t.Run("handoff", func(t *testing.T) {
		bundle := currentBusinessAuthoringBundleProbe(t, root)
		if len(bundle.Handoffs.Handoffs) == 0 {
			t.Fatal("test setup: Handoffがありません")
		}
		bundle.Handoffs.Handoffs = append([]referenceSetHandoff(nil), bundle.Handoffs.Handoffs...)
		bundle.Handoffs.Handoffs[0].Condition = "changed after preview"
		current, err := businessSourceFreshnessSnapshotForBundleProbe(bundle)
		if err != nil {
			t.Fatal(err)
		}
		if err := requireBusinessSourceFreshnessProbe(baseline, current); err == nil {
			t.Fatal("standalone Handoff source changeをstaleとして検出できません")
		}
	})
}

func TestBusinessSourceFreshnessSnapshotIgnoresProcedureEnumerationOrder(t *testing.T) {
	root := repoPath()
	beforeBundle := currentBusinessAuthoringBundleProbe(t, root)
	before, err := businessSourceFreshnessSnapshotForBundleProbe(beforeBundle)
	if err != nil {
		t.Fatal(err)
	}

	afterBundle := currentBusinessAuthoringBundleProbe(t, root)
	for left, right := 0, len(afterBundle.Procedures)-1; left < right; left, right = left+1, right-1 {
		afterBundle.Procedures[left], afterBundle.Procedures[right] = afterBundle.Procedures[right], afterBundle.Procedures[left]
	}
	after, err := businessSourceFreshnessSnapshotForBundleProbe(afterBundle)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("source enumeration orderだけでfreshness snapshotが変化しました: before=%+v after=%+v", before, after)
	}
}

func TestBusinessSourceFreshnessSnapshotUsesLogicalKeysNotPaths(t *testing.T) {
	root := repoPath()
	bundle := currentBusinessAuthoringBundleProbe(t, root)
	snapshot, err := businessSourceFreshnessSnapshotForBundleProbe(bundle)
	if err != nil {
		t.Fatal(err)
	}

	want := []string{
		"handoffs:default",
		"procedure:order-entry.follow-up",
		"procedure:order-entry.order-handling",
	}
	got := make([]string, 0, len(snapshot.Sources))
	for _, source := range snapshot.Sources {
		got = append(got, source.LogicalKey)
		if source.Fingerprint == "" {
			t.Fatalf("source fingerprintがありません: %+v", source)
		}
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("freshness logical keys=%v want=%v", got, want)
	}
}

func businessSourceFreshnessSnapshotForBundleProbe(bundle businessAuthoringBundleProbe) (businessSourceFreshnessSnapshotProbe, error) {
	entries := make([]businessSourceFreshnessEntryProbe, 0, len(bundle.Procedures)+1)
	seen := map[string]struct{}{}

	for _, source := range bundle.Procedures {
		key := "procedure:" + source.Procedure.ID
		if source.Procedure.ID == "" {
			return businessSourceFreshnessSnapshotProbe{}, fmt.Errorf("Business Procedure source has empty identity")
		}
		if _, duplicate := seen[key]; duplicate {
			return businessSourceFreshnessSnapshotProbe{}, fmt.Errorf("duplicate Business source logical key %q", key)
		}
		seen[key] = struct{}{}
		fingerprint, err := jsonContentFingerprintProbe(source)
		if err != nil {
			return businessSourceFreshnessSnapshotProbe{}, err
		}
		entries = append(entries, businessSourceFreshnessEntryProbe{LogicalKey: key, Fingerprint: fingerprint})
	}

	handoffKey := "handoffs:default"
	if _, duplicate := seen[handoffKey]; duplicate {
		return businessSourceFreshnessSnapshotProbe{}, fmt.Errorf("duplicate Business source logical key %q", handoffKey)
	}
	fingerprint, err := jsonContentFingerprintProbe(bundle.Handoffs)
	if err != nil {
		return businessSourceFreshnessSnapshotProbe{}, err
	}
	entries = append(entries, businessSourceFreshnessEntryProbe{LogicalKey: handoffKey, Fingerprint: fingerprint})

	sort.Slice(entries, func(i, j int) bool { return entries[i].LogicalKey < entries[j].LogicalKey })
	return businessSourceFreshnessSnapshotProbe{Sources: entries}, nil
}

func requireBusinessSourceFreshnessProbe(base, current businessSourceFreshnessSnapshotProbe) error {
	if !reflect.DeepEqual(base, current) {
		return fmt.Errorf("Business authoring sources changed after preview")
	}
	return nil
}

func jsonContentFingerprintProbe(value any) (string, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:]), nil
}
