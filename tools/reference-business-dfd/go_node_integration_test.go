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
			"schema":           "tei.reference.business-dfd-selected-edit/v0",
			"id":               "go-node-reference-review",
			"origin":           "cli",
			"intent":           "業務上の起動契機の説明だけを見直す",
			"procedureSHA256":  dfdTestDigest(procedureInput),
			"reviewSHA256":     selectedReviewDigest,
			"projectionSHA256": dfdTestDigest(projectionInput),
			"selection":        map[string]string{"kind": "trigger", "id": "TR-ORDER"},
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
	t.Run("swapped_selected_trigger_source_pointer_fails", func(t *testing.T) {
		var p map[string]any
		if err := json.Unmarshal(projectionBytes, &p); err != nil {
			t.Fatal(err)
		}
		triggers := p["triggers"].([]any)
		source := triggers[0].(map[string]any)["source"].(map[string]any)
		source["pointer"] = "/triggers/1"
		modified, err := json.Marshal(p)
		if err != nil {
			t.Fatal(err)
		}
		runCase(t, modified, procedureBytes, reviewBytes, dfdTestDigest(reviewBytes),
			"DFD_SELECTED_REFERENCE_LOCATOR_MISMATCH")
	})
	t.Run("swapped_selected_trigger_source_file_fails", func(t *testing.T) {
		var p map[string]any
		if err := json.Unmarshal(projectionBytes, &p); err != nil {
			t.Fatal(err)
		}
		triggers := p["triggers"].([]any)
		source := triggers[0].(map[string]any)["source"].(map[string]any)
		source["file"] = "unrelated-review.json"
		modified, err := json.Marshal(p)
		if err != nil {
			t.Fatal(err)
		}
		runCase(t, modified, procedureBytes, reviewBytes, dfdTestDigest(reviewBytes),
			"DFD_SELECTED_REFERENCE_FILE_MISMATCH")
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

// TestBusinessDFDGoProjectionDrivesView exercises the actual DFD browser
// module with Go-produced projections through its file-picker interface.
// The DOM is mocked by Node; this is not a real-browser acceptance test.
func TestBusinessDFDGoProjectionDrivesView(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		if os.Getenv("GITHUB_ACTIONS") == "true" {
			t.Fatal("DFD Go-to-view integration requires Node.js on GitHub Actions")
		}
		t.Skip("DFD Go-to-view integration not executed: Node.js unavailable")
	}
	root := filepath.Join("..", "..", "reference", "business-system-design")
	cases := []struct {
		name, procedure, review, set string
	}{
		{"direct-validation", "order-entry-business-procedure.json",
			"order-entry-dfd-review-proposal.json", "order-entry-procedure-reference-set.json"},
		{"order-approval", filepath.Join("dfd-order-approval", "order-approval-procedure.json"),
			filepath.Join("dfd-order-approval", "review-proposal.json"),
			filepath.Join("dfd-order-approval", "procedure-set.json")},
	}
	dir := t.TempDir()
	paths := make([]string, 0, len(cases))
	for _, sample := range cases {
		p, err := run(filepath.Join(root, sample.procedure), filepath.Join(root, sample.review),
			filepath.Join(root, sample.set), filepath.Join("..", "..", "reference", "workspace"))
		if err != nil {
			t.Fatalf("real Go projection for %s: %v", sample.name, err)
		}
		if p.Authority != "derived-read-only" || p.CanonicalProcedureUsed {
			t.Fatalf("real Go projection wrongly asserts authority for %s", sample.name)
		}
		for _, d := range p.Diagnostics {
			if d.Severity == "error" {
				t.Fatalf("%s has error diagnostic: %+v", sample.name, d)
			}
		}
		data, err := json.Marshal(p)
		if err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(dir, sample.name+".json")
		if err := os.WriteFile(path, data, 0600); err != nil {
			t.Fatal(err)
		}
		paths = append(paths, path)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, node,
		"--test", "--test-name-pattern=real Go projection crosses browser view boundary",
		filepath.Join(root, "dfd-view-studio.test.mjs"))
	cmd.Env = append(os.Environ(),
		"TEI_DFD_REAL_GO_DIRECT_PROJECTION="+paths[0],
		"TEI_DFD_REAL_GO_APPROVAL_PROJECTION="+paths[1])
	output, err := cmd.CombinedOutput()
	if ctx.Err() != nil {
		t.Fatalf("Go-produced DFD view integration timed out: %v\n%s", ctx.Err(), output)
	}
	if err != nil || !strings.Contains(string(output), "# fail 0") ||
		!strings.Contains(string(output), "# pass 1") {
		t.Fatalf("Go-produced DFD view integration failed: %v\n%s", err, output)
	}
	t.Logf("Real Go DFD projection exercised by Node DOM view test (no browser Acceptance):\n%s", output)
}
