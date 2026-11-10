package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestThreeElementValidateCommand(t *testing.T) {
	root := t.TempDir()
	modelPath := filepath.Join(root, "model.json")
	if err := os.WriteFile(modelPath, []byte(fixtureBytes), 0o644); err != nil {
		t.Fatal(err)
	}

	var out strings.Builder
	code, err := runThreeElementCommand([]string{"validate", "-model", modelPath}, &out)
	if err != nil {
		t.Fatalf("command error: %v", err)
	}
	if code != 0 {
		t.Fatalf("expected exit 0, got %d: %s", code, out.String())
	}

	var resp threeElementResponse
	if err := json.Unmarshal([]byte(out.String()), &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if !resp.OK {
		t.Fatalf("expected OK validation, got %+v", resp.Result)
	}
}

func TestThreeElementCRUDCommand(t *testing.T) {
	root := t.TempDir()
	modelPath := filepath.Join(root, "model.json")
	if err := os.WriteFile(modelPath, []byte(fixtureBytes), 0o644); err != nil {
		t.Fatal(err)
	}

	var out strings.Builder
	code, err := runThreeElementCommand([]string{"crud", "-model", modelPath}, &out)
	if err != nil {
		t.Fatalf("command error: %v", err)
	}
	if code != 0 {
		t.Fatalf("expected exit 0, got %d: %s", code, out.String())
	}

	var resp threeElementResponse
	if err := json.Unmarshal([]byte(out.String()), &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if !resp.OK {
		t.Fatalf("expected OK response, got %+v", resp.Result)
	}
}

func TestThreeElementDiffCommand(t *testing.T) {
	root := t.TempDir()
	before := filepath.Join(root, "before.json")
	after := filepath.Join(root, "after.json")
	if err := os.WriteFile(before, []byte(fixtureBytes), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(after, []byte(fixtureBytes), 0o644); err != nil {
		t.Fatal(err)
	}

	var out strings.Builder
	code, err := runThreeElementCommand([]string{"diff", "-before", before, "-after", after}, &out)
	if err != nil {
		t.Fatalf("command error: %v", err)
	}
	if code != 0 {
		t.Fatalf("expected exit 0, got %d: %s", code, out.String())
	}

	var resp threeElementResponse
	if err := json.Unmarshal([]byte(out.String()), &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if !resp.OK {
		t.Fatalf("expected OK response, got %+v", resp.Result)
	}
}
