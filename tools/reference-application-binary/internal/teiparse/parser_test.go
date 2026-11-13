package teiparse

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseWorldReferenceWorkspace(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	root := filepath.Join(wd, "..", "..", "..", "..", "reference", "workspace")
	world, err := ParseWorld(root)
	if err != nil {
		t.Fatalf("parse world: %v", err)
	}
	t.Logf("databases=%d migrations=%d", len(world.Databases), len(world.Migrations))
	for _, db := range world.Databases {
		t.Logf("database name=%q kind=%q", db.Name, db.Kind)
	}
	for _, m := range world.Migrations {
		t.Logf("migration id=%q database=%q schema=%d data=%d verify=%d", m.ID, m.Database, len(m.SchemaSQL), len(m.DataSQL), len(m.VerifySQL))
	}
	if len(world.Databases) == 0 {
		t.Fatalf("expected databases, got 0")
	}
	if len(world.Migrations) == 0 {
		t.Fatalf("expected migrations, got 0")
	}
}
