//go:build tei_frozen_world && tei_api_only && tei_order_only && !tei_repair_only

package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFrozenWorldRejectsUnpinnedOrChangedInputs(t *testing.T) {
	dir := t.TempDir()
	name := filepath.Join(dir, "frozen-world.json")
	snapshot := frozenWorldEnvelope{
		Schema:                     frozenWorldSchema,
		ApplicationID:              "order_entry",
		HandoverBaselineID:         "B",
		BaselineRepositoryRevision: strings.Repeat("a", 40),
		TargetBuildRevision:        strings.Repeat("b", 40),
	}
	source, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	previous := frozenWorldSHA256
	t.Cleanup(func() { frozenWorldSHA256 = previous })
	save := func(data []byte) {
		t.Helper()
		if err := os.WriteFile(name, data, 0600); err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256(data)
		frozenWorldSHA256 = hex.EncodeToString(sum[:])
	}
	load := func(b, r, rev string) error {
		t.Helper()
		_, err := loadSingleApplicationWorld(dir, b, r, rev)
		return err
	}
	frozenWorldSHA256 = ""
	if err := os.WriteFile(name, source, 0600); err != nil {
		t.Fatal(err)
	}
	if load("B", snapshot.BaselineRepositoryRevision, snapshot.TargetBuildRevision) == nil {
		t.Fatal("missing compiled digest accepted")
	}
	save(source)
	if err := load("B", snapshot.BaselineRepositoryRevision, snapshot.TargetBuildRevision); err != nil {
		t.Fatal("matching frozen source rejected:", err)
	}
	for _, tc := range []struct{ b, r, rev string }{
		{"other", snapshot.BaselineRepositoryRevision, snapshot.TargetBuildRevision},
		{"B", strings.Repeat("c", 40), snapshot.TargetBuildRevision},
		{"B", snapshot.BaselineRepositoryRevision, strings.Repeat("d", 40)},
	} {
		if load(tc.b, tc.r, tc.rev) == nil {
			t.Fatal("B/R/T mismatch accepted", tc)
		}
	}
	if err := os.WriteFile(name, append(append([]byte(nil), source...), '\n'), 0600); err != nil {
		t.Fatal(err)
	}
	if load("B", snapshot.BaselineRepositoryRevision, snapshot.TargetBuildRevision) == nil {
		t.Fatal("changed World bytes accepted")
	}
	save(append(append([]byte(nil), source...), []byte("{}")...))
	if load("B", snapshot.BaselineRepositoryRevision, snapshot.TargetBuildRevision) == nil {
		t.Fatal("extra JSON document accepted")
	}
	snapshot.ApplicationID = "repair_intake"
	changed, err := json.Marshal(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	save(changed)
	if load("B", snapshot.BaselineRepositoryRevision, snapshot.TargetBuildRevision) == nil {
		t.Fatal("different Application accepted even with pinned digest")
	}
}
