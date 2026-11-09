package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

// writeRelationlessWorkspace creates a Workspace whose subject has no declared
// references, so a full scan produces zero rows.
func writeRelationlessWorkspace(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeWorkspaceFile(t, root, "tei.workspace.yaml", "workspace: c2c\nformatVersion: 1\nsources:\n  - semantic\n")
	writeWorkspaceFile(t, root, "semantic/sales.tei", `namespace sales

data order {
    attribute orderId type std.text
}

data customer {
    attribute customerId type std.text
}
`)
	return root
}

// TestTraceFullWritesRowsEvenWhenZero pins #3693 at the CLI boundary: a reader
// must be able to tell "this subject declares no references" from "this response
// has no rows field".
func TestTraceFullWritesRowsEvenWhenZero(t *testing.T) {
	root := writeRelationlessWorkspace(t)
	runTrace := func(mode string) map[string]json.RawMessage {
		t.Helper()
		var out bytes.Buffer
		code, err := traceWorkspaceJSON(&out, root, "", "c2c", "semantic-data", "sales.order",
			"both", modeOrDefault(mode), "", 0, 0, 0)
		if err != nil || code != 0 {
			t.Fatalf("%s: code=%d err=%v out=%q", mode, code, err, out.String())
		}
		var envelope struct {
			Result struct {
				Trace map[string]json.RawMessage `json:"trace"`
			} `json:"result"`
		}
		if err := json.Unmarshal(out.Bytes(), &envelope); err != nil {
			t.Fatalf("%s: %v (%q)", mode, err, out.String())
		}
		return envelope.Result.Trace
	}

	full := runTrace("full")
	rows, present := full["rows"]
	if !present {
		t.Fatalf("full mode omitted rows for zero references: %v", keysOf(full))
	}
	if string(rows) != "[]" {
		t.Fatalf("rows = %s, want []", rows)
	}

	summary := runTrace("summary")
	if _, present := summary["rows"]; present {
		t.Fatalf("summary carried rows: %v", keysOf(summary))
	}
	// Summary still reports the count, which is how it communicates emptiness.
	if _, present := summary["total"]; !present {
		t.Fatalf("summary lost total: %v", keysOf(summary))
	}

	// The default mode is summary, so it must behave like an explicit summary.
	if _, present := runTrace("")["rows"]; present {
		t.Fatal("the default mode carried rows; the advertised default is summary")
	}
}

func modeOrDefault(mode string) string {
	if mode == "" {
		return "summary"
	}
	return mode
}

func keysOf(m map[string]json.RawMessage) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	return keys
}

// The built binary must behave the same way, so the field is not an artefact of
// calling the projection directly.
func TestTraceFullWritesRowsInTheBuiltProcess(t *testing.T) {
	if testing.Short() {
		t.Skip("builds the CLI")
	}
	binary := filepath.Join(t.TempDir(), "tei")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	if output, err := exec.Command("go", "build", "-o", binary, ".").CombinedOutput(); err != nil {
		t.Fatalf("build CLI: %v\n%s", err, output)
	}
	root := writeRelationlessWorkspace(t)

	for _, tc := range []struct {
		mode    string
		wantRow bool
	}{{"full", true}, {"summary", false}} {
		t.Run(tc.mode, func(t *testing.T) {
			command := exec.Command(binary, "trace", "-workspace", root, "-scope", "c2c",
				"-subject-kind", "semantic-data", "-subject-id", "sales.order", "-mode", tc.mode)
			command.Env = append(os.Environ(), "GIT_DIR=")
			var out bytes.Buffer
			command.Stdout = &out
			if err := command.Run(); err != nil {
				t.Fatalf("trace: %v (%q)", err, out.String())
			}
			var envelope struct {
				Result struct {
					Trace map[string]json.RawMessage `json:"trace"`
				} `json:"result"`
			}
			if err := json.Unmarshal(out.Bytes(), &envelope); err != nil {
				t.Fatalf("answer is not JSON: %v (%q)", err, out.String())
			}
			rows, present := envelope.Result.Trace["rows"]
			if present != tc.wantRow {
				t.Fatalf("rows present=%v, want %v: %v", present, tc.wantRow, keysOf(envelope.Result.Trace))
			}
			if tc.wantRow && string(rows) != "[]" {
				t.Fatalf("rows = %s, want []", rows)
			}
		})
	}
}
