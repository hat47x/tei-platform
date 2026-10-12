package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func dfdTestDigest(data []byte) string {
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

// TestBusinessDFDGoProjectionToNodeReview checks the real Go-produced
// projection against the separate Node review adapter. Neither operation
// grants Source Acceptance or writes back to the reference sources.
func TestBusinessDFDGoProjectionToNodeReview(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		if os.Getenv("GITHUB_ACTIONS") == "true" {
			t.Fatal("DFD end-to-end review test requires Node.js on GitHub Actions")
		}
		t.Skip("DFD Go-to-Node review test not executed: Node.js is unavailable")
	}

	root := filepath.Join("..", "..", "reference", "business-system-design")
	procedurePath := filepath.Join(root, "dfd-order-approval", "order-approval-procedure.json")
	reviewPath := filepath.Join(root, "dfd-order-approval", "review-proposal.json")
	setPath := filepath.Join(root, "dfd-order-approval", "procedure-set.json")
	projection, err := run(procedurePath, reviewPath, setPath, filepath.Join("..", "..", "reference", "workspace"))
	if err != nil {
		t.Fatalf("Go DFD projection: %v", err)
	}
	for _, d := range projection.Diagnostics {
		if d.Severity == "error" {
			t.Fatalf("Go DFD projection has unresolved error: %+v", d)
		}
	}
	projectionBytes, err := json.Marshal(projection)
	if err != nil {
		t.Fatal(err)
	}
	procedureBytes, err := os.ReadFile(procedurePath)
	if err != nil {
		t.Fatal(err)
	}
	reviewBytes, err := os.ReadFile(reviewPath)
	if err != nil {
		t.Fatal(err)
	}

	runCase := func(t *testing.T, projectionInput, procedureInput, reviewInput []byte, selectedReviewDigest string, expectError string) {
		t.Helper()
		dir := t.TempDir()
		paths := map[string]string{
			"projection": filepath.Join(dir, "projection.json"),
			"procedure":  filepath.Join(dir, "procedure.json"),
			"review":     filepath.Join(dir, "review.json"),
			"binding":    filepath.Join(dir, "binding.json"),
		}
		inputs := map[string][]byte{
			"projection": projectionInput, "procedure": procedureInput, "review": reviewInput,
		}
		for key, data := range inputs {
			if err := os.WriteFile(paths[key], data, 0600); err != nil {
				t.Fatal(err)
			}
		}
		binding := map[string]any{
			"schema": "tei.reference.business-dfd-selected-edit/v0",
			"id": "go-node-reference-review",
			"origin": "cli",
			"intent": "業務上の起動契機の説明だけを見直す",
			"procedureSHA256": dfdTestDigest(procedureInput),
			"reviewSHA256": selectedReviewDigest,
			"projectionSHA256": dfdTestDigest(projectionInput),
			"selection": map[string]string{"kind": "trigger", "id": "TR-ORDER"},
			"operation": map[string]string{
				"kind": "replace-trigger-label", "trigger": "TR-ORDER",
				"label": "受注内容を担当者が確認する",
			},
		}
		bindingBytes, err := json.Marshal(binding)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(paths["binding"], bindingBytes, 0600); err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		cmd := exec.CommandContext(ctx, node,
			filepath.Join(root, "dfd-selected-edit-cli.mjs"),
			"-projection", paths["projection"], "-procedure", paths["procedure"],
			"-review", paths["review"], "-binding", paths["binding"],
		)
		var stdout, stderr bytes.Buffer
		cmd.Stdout, cmd.Stderr = &stdout, &stderr
		err = cmd.Run()
		if ctx.Err() != nil {
			t.Fatalf("Node selected-edit CLI timed out: %v", ctx.Err())
		}
		if expectError != "" {
			if err == nil || stdout.Len() != 0 || !strings.Contains(stderr.String(), expectError) {
				t.Fatalf("expected rejection %q, got err=%v stdout=%s stderr=%s",
					expectError, err, stdout.String(), stderr.String())
			}
			return
		}
		if err != nil {
			t.Fatalf("real Go projection rejected by Node CLI: %v\n%s", err, stderr.String())
		}
		var result struct {
			Authority                 string `json:"authority"`
			ReadyForAcceptance        bool   `json:"readyForAcceptance"`
			SharedAcceptanceCandidate *any   `json:"sharedAcceptanceCandidate"`
			SourceWrites              []any  `json:"sourceWrites"`
			InputSnapshotsVerified    bool   `json:"inputSnapshotsVerified"`
			Selected                  struct {
				ID string `json:"id"`
			} `json:"selected"`
			ProposedReview struct {
				Triggers []struct {
					ID    string `json:"id"`
					Label string `json:"label"`
				} `json:"triggers"`
			} `json:"proposedReview"`
		}
		if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
			t.Fatalf("unreadable Node review packet: %v\n%s", err, stdout.String())
		}
		if result.Authority != "unaccepted-proposal-only" ||
			result.ReadyForAcceptance || result.SharedAcceptanceCandidate != nil ||
			len(result.SourceWrites) != 0 || !result.InputSnapshotsVerified ||
			result.Selected.ID != "TR-ORDER" ||
			len(result.ProposedReview.Triggers) == 0 ||
			result.ProposedReview.Triggers[0].Label != "受注内容を担当者が確認する" {
			t.Fatalf("Node review silently promoted or lost business meaning: %+v", result)
		}
		for key, original := range inputs {
			after, err := os.ReadFile(paths[key])
			if err != nil || !bytes.Equal(original, after) {
				t.Fatalf("Node altered %s: %v", key, err)
			}
		}
	}

	t.Run("exact_go_projection_is_unaccepted_review", func(t *testing.T) {
		runCase(t, projectionBytes, procedureBytes, reviewBytes, dfdTestDigest(reviewBytes), "")
	})
	t.Run("stale_review_bytes_fail_closed", func(t *testing.T) {
		changed := append(bytes.Clone(reviewBytes), '\n')
		runCase(t, projectionBytes, procedureBytes, changed, dfdTestDigest(reviewBytes),
			"STALE_OR_UNVERIFIED_DFD_EDIT_INPUT_SNAPSHOT")
	})
	t.Run("forged_trigger_runtime_status_fails", func(t *testing.T) {
		var p map[string]any
		if err := json.Unmarshal(projectionBytes, &p); err != nil {
			t.Fatal(err)
		}
		triggers := p["triggers"].([]any)
		triggers[0].(map[string]any)["runtimeBinding"] = "cron"
		modified, err := json.Marshal(p)
		if err != nil {
			t.Fatal(err)
		}
		runCase(t, modified, procedureBytes, reviewBytes, dfdTestDigest(reviewBytes),
			"DFD_PRODUCER_AUTHORITY_BOUNDARY_MISMATCH")
	})
}
