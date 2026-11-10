package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// §4.48: a module context that points outside itself cannot be reproduced from a
// staged tree, so it is refused rather than built and reported as if it were.
// replace sends the build somewhere the plan does not name; vendor holds the
// dependencies beside the module instead of in the cache.

func writeModule(t *testing.T, dir, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestCheckModuleContextRefusesAReplaceDirective(t *testing.T) {
	dir := t.TempDir()
	writeModule(t, dir, "module probe\n\ngo 1.25\n\nreplace example.com/x => ../x\n")
	err := checkModuleContextIsReproducible(dir)
	if err == nil {
		t.Fatal("a replace directive was accepted")
	}
	if !strings.Contains(err.Error(), "replace") {
		t.Fatalf("the refusal does not name the directive: %v", err)
	}
}

// The block form is the same statement spelled differently, so it is refused too.
func TestCheckModuleContextRefusesAReplaceBlock(t *testing.T) {
	dir := t.TempDir()
	writeModule(t, dir, "module probe\n\ngo 1.25\n\nreplace (\n\texample.com/x => ../x\n)\n")
	if err := checkModuleContextIsReproducible(dir); err == nil {
		t.Fatal("a replace block was accepted")
	}
}

func TestCheckModuleContextRefusesAVendorDirectory(t *testing.T) {
	dir := t.TempDir()
	writeModule(t, dir, "module probe\n\ngo 1.25\n")
	if err := os.MkdirAll(filepath.Join(dir, "vendor"), 0o755); err != nil {
		t.Fatal(err)
	}
	err := checkModuleContextIsReproducible(dir)
	if err == nil {
		t.Fatal("a vendor directory was accepted")
	}
	if !strings.Contains(err.Error(), "vendor") {
		t.Fatalf("the refusal does not name the directory: %v", err)
	}
}

// A plain module context is what the isolated build can reproduce.
func TestCheckModuleContextAcceptsAPlainModule(t *testing.T) {
	dir := t.TempDir()
	writeModule(t, dir, "module probe\n\ngo 1.25\n\nrequire example.com/x v1.0.0\n")
	if err := checkModuleContextIsReproducible(dir); err != nil {
		t.Fatalf("a plain module context was refused: %v", err)
	}
}

// No module file at all is not this check's business: the build reports it.
func TestCheckModuleContextIgnoresAMissingModuleFile(t *testing.T) {
	if err := checkModuleContextIsReproducible(t.TempDir()); err != nil {
		t.Fatalf("a missing go.mod was refused by this check: %v", err)
	}
}
