// Test-only H3-A0 uses the actual TEI and SEI CLI binaries. The consumer
// records a recommendation, not an accepted meaning or a resolved inquiry.
// Run with TEI_H3A0_BIN and SEI_H3A0_BIN; no product code is modified.
package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

type h3A0Reply struct {
	SchemaVersion string          `json:"schemaVersion"`
	OK            bool            `json:"ok"`
	Mode          string          `json:"mode"`
	Result        json.RawMessage `json:"result"`
}

type h3A0Subject struct {
	Kind string `json:"kind"`
	ID   string `json:"id"`
}

type h3A0Index struct {
	References []struct {
		Subject h3A0Subject `json:"subject"`
	} `json:"references"`
}

type h3A0SourceView struct {
	Subject h3A0Subject `json:"subject"`
}

type h3A0Resume struct {
	SnapshotDigest string `json:"snapshot_digest"`
	RecordCount    int    `json:"record_count"`
	Open           []struct {
		ID        string `json:"id"`
		Statement string `json:"statement"`
	} `json:"recommendations_not_yet_relied_on"`
}

type h3A0Original struct {
	SnapshotDigest string `json:"snapshot_digest"`
	SourceRecord   struct {
		RecordID string `json:"record_id"`
	} `json:"source_record"`
	Record json.RawMessage `json:"record"`
}

func h3A0Invoke(t *testing.T, ctx context.Context, bin string, args ...string) ([]byte, []byte) {
	t.Helper()
	command := exec.CommandContext(ctx, bin, args...)
	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		t.Fatalf("actual CLI invocation %q failed: %v\nstdout=%s\nstderr=%s",
			filepath.Base(bin), err, stdout.String(), stderr.String())
	}
	return stdout.Bytes(), stderr.Bytes()
}

func h3A0JSON(t *testing.T, body []byte, dest any) {
	t.Helper()
	if !json.Valid(body) || !bytes.HasSuffix(body, []byte("\n")) {
		t.Fatalf("CLI result must be a complete newline-terminated JSON document: %q", body)
	}
	if err := json.Unmarshal(body, dest); err != nil {
		t.Fatalf("decode actual CLI result: %v", err)
	}
}

func h3A0SHA(body []byte) string {
	return fmt.Sprintf("sha256:%x", sha256.Sum256(body))
}

func TestH3A0ActualTEIToSEIRecommendationResume(t *testing.T) {
	teiBin, seiBin := os.Getenv("TEI_H3A0_BIN"), os.Getenv("SEI_H3A0_BIN")
	if teiBin == "" || seiBin == "" {
		if os.Getenv("TEI_H3A0_REQUIRED") == "1" {
			t.Fatal("H3-A0 requires both built CLI binaries in CI")
		}
		t.Skip("requires actual built TEI/SEI CLI binaries")
	}
	for _, bin := range []string{teiBin, seiBin} {
		if !filepath.IsAbs(bin) {
			t.Fatalf("CLI binary must be an absolute path: %s", bin)
		}
		if _, err := os.Stat(bin); err != nil {
			t.Fatalf("CLI binary is not present: %v", err)
		}
	}
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("test source location not available")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	workspace := filepath.Join(repoRoot, "reference", "workspace")
	if _, err := os.Stat(workspace); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	// Freeze three public obligations independently of any source-index output.
	required := []string{
		"sales.order_line.quantity_positive",
		"sales.order.lines.line_number_unique",
		"sales.order_line.line_number_positive",
	}
	rule := h3A0Subject{Kind: "semantic-validation", ID: required[2]}
	indexRaw, indexErr := h3A0Invoke(t, ctx, teiBin, "source-index", "-workspace", workspace)
	subjectRaw, subjectErr := h3A0Invoke(t, ctx, teiBin, "source-index", "-workspace", workspace,
		"-subject-kind", rule.Kind, "-subject-id", rule.ID)
	if len(indexErr) != 0 || len(subjectErr) != 0 {
		t.Fatalf("unexpected diagnostics for successful CLI calls: index=%q subject=%q", indexErr, subjectErr)
	}
	var indexReply, subjectReply h3A0Reply
	h3A0JSON(t, indexRaw, &indexReply)
	h3A0JSON(t, subjectRaw, &subjectReply)
	if indexReply.SchemaVersion != "tei.cli.source-index/v0" || !indexReply.OK ||
		indexReply.Mode != "index" || subjectReply.SchemaVersion != indexReply.SchemaVersion ||
		!subjectReply.OK || subjectReply.Mode != "subject" {
		t.Fatalf("source-index mode or schema mismatch: index=%+v subject=%+v", indexReply, subjectReply)
	}
	var full h3A0Index
	var scoped h3A0SourceView
	h3A0JSON(t, append(append([]byte(nil), indexReply.Result...), '\n'), &full)
	h3A0JSON(t, append(append([]byte(nil), subjectReply.Result...), '\n'), &scoped)
	if scoped.Subject != rule {
		t.Fatalf("unexpected subject-mode source: got=%+v want=%+v", scoped.Subject, rule)
	}
	observed := map[string]bool{}
	for _, entry := range full.References {
		if entry.Subject.Kind == rule.Kind {
			observed[entry.Subject.ID] = true
		}
	}
	for _, id := range required {
		if !observed[id] {
			t.Fatalf("fixed required validation is absent from observed index: %s", id)
		}
	}
	// Subject mode returns one requested subject, not the independent
	// Required list. This is why the recorded advice asks for index mode.
	if len(required) != 3 {
		t.Fatal("invalid independently fixed required list")
	}

	// SEI uses a new, isolated, throwaway store, not TEI's workspace.
	dataRoot := filepath.Join(t.TempDir(), "sei-record-store")
	initRaw, _ := h3A0Invoke(t, ctx, seiBin, "init", "--data-root", dataRoot)
	var initialized map[string]string
	h3A0JSON(t, initRaw, &initialized)
	if initialized["bundle_id"] == "" {
		t.Fatal("SEI did not initialize a new record bundle")
	}
	id := "h3-a0-required-rule-index"
	statement := "Inspect the complete index-mode source index for three fixed validations; a single subject is insufficient."
	addRaw, _ := h3A0Invoke(t, ctx, seiBin, "add", "recommendation",
		"--data-root", dataRoot, "--id", id, "--statement", statement,
		"--rationale", "The one-subject response cannot establish the independent required set.",
		"--source", "h3-a0-fixture-evaluator")
	var added map[string]string
	h3A0JSON(t, addRaw, &added)
	if added["recorded"] != "recommendation" || added["id"] != id {
		t.Fatalf("SEI did not acknowledge the recommendation: %+v", added)
	}

	resumeRaw, _ := h3A0Invoke(t, ctx, seiBin, "resume", "--data-root", dataRoot, "--json")
	var resumed h3A0Resume
	h3A0JSON(t, resumeRaw, &resumed)
	if resumed.RecordCount != 1 || len(resumed.Open) != 1 ||
		resumed.Open[0].ID != id || resumed.Open[0].Statement != statement ||
		!strings.HasPrefix(resumed.SnapshotDigest, "sha256:") {
		t.Fatalf("SEI resume did not return the original recommendation: %+v", resumed)
	}
	recordRaw, _ := h3A0Invoke(t, ctx, seiBin, "resume", "--data-root", dataRoot,
		"--json", "--record-id", id, "--snapshot-digest", resumed.SnapshotDigest)
	var original h3A0Original
	h3A0JSON(t, recordRaw, &original)
	if original.SourceRecord.RecordID != id || original.SnapshotDigest != resumed.SnapshotDigest ||
		!bytes.Contains(original.Record, []byte(statement)) {
		t.Fatalf("SEI source-record/digest/statement is inconsistent: %+v", original)
	}

	// "--unresolved" is a decision-only flag. A failed attempt must not
	// persist a fictitious decision or silently mark the inquiry resolved.
	invalid := exec.CommandContext(ctx, seiBin, "add", "recommendation", "--data-root", dataRoot,
		"--id", "h3-a0-unsupported", "--statement", "placeholder",
		"--unresolved", "question:h3-a0-unknown")
	if err := invalid.Run(); err == nil {
		t.Fatal("recommendation incorrectly accepted decision-only --unresolved")
	}
	afterRaw, _ := h3A0Invoke(t, ctx, seiBin, "resume", "--data-root", dataRoot, "--json")
	var after h3A0Resume
	h3A0JSON(t, afterRaw, &after)
	if after.SnapshotDigest != resumed.SnapshotDigest || after.RecordCount != 1 {
		t.Fatalf("failed recommendation attempt changed SEI store: %+v", after)
	}
	t.Logf("H3-A0 actual local record round-trip: TEI index=%s subject=%s; SEI store=%s; no Method/Authority/Grant claimed",
		h3A0SHA(indexRaw), h3A0SHA(subjectRaw), resumed.SnapshotDigest)
}
