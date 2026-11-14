package studiohost

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

const localREAChildReceiptPrefix = "REA_LOCAL_HOST_CHILD_RECEIPT="

// localREAChildReceipt is a deliberately minimal, test-only acknowledgment of
// an actual different OS process. It is NOT a public Delivery/Disclosure
// receipt: source paths, raw Evidence and private errors never enter it.
type localREAChildReceipt struct {
	PID          int    `json:"pid"`
	OK           bool   `json:"ok"`
	EvidenceID   string `json:"evidenceId,omitempty"`
	Digest       string `json:"digest,omitempty"`
	Observation  string `json:"observationId,omitempty"`
	Relation     string `json:"relationId,omitempty"`
	ReadAt       string `json:"readAt,omitempty"`
	UnknownCount int    `json:"unknownCount,omitempty"`
	FailureCode  string `json:"failureCode,omitempty"`
}

// TestLocalREAReaderActualChildProcessWitness proves the Host reader is
// invocable and observable through a separate real OS process, not merely
// through a second function call in the parent test. This test binary is only
// an independent test harness, NOT a Studio/Wails/MCP product consumer.
func TestLocalREAReaderActualChildProcessWitness(t *testing.T) {
	binary, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name     string
		mode     string
		wantOK   bool
		wantCode string
	}{
		{name: "selected-invoked-received", mode: "allowed", wantOK: true},
		{name: "unknown-selector", mode: "unbound", wantCode: "OPERATION_FAILED"},
		{name: "caller-path-in-selector", mode: "path", wantCode: "OPERATION_FAILED"},
		{name: "changed-saved-evidence", mode: "mutated", wantCode: "OPERATION_FAILED"},
		{name: "other-workspace", mode: "other-workspace", wantCode: "WORKSPACE_NOT_ALLOWED"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cmd := exec.Command(binary, "-test.run=^TestLocalREAReaderChildProcessHelper$", "-test.v")
			cmd.Env = append(os.Environ(),
				"TEI_REA_LOCAL_CHILD_HELPER=1",
				"TEI_REA_LOCAL_CHILD_MODE="+tc.mode,
			)
			output, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("actual Host child process failed: %v; output=%s", err, output)
			}
			var receipts []localREAChildReceipt
			for _, line := range strings.Split(string(output), "\n") {
				line = strings.TrimSpace(line)
				if !strings.HasPrefix(line, localREAChildReceiptPrefix) {
					continue
				}
				var receipt localREAChildReceipt
				if err := json.Unmarshal([]byte(strings.TrimPrefix(line, localREAChildReceiptPrefix)), &receipt); err != nil {
					t.Fatalf("malformed child process receipt: %v; output=%s", err, output)
				}
				receipts = append(receipts, receipt)
			}
			if len(receipts) != 1 {
				t.Fatalf("actual child receipt missing or repeated (%d): %s", len(receipts), output)
			}
			got := receipts[0]
			if got.PID == 0 || got.PID == os.Getpid() || cmd.Process == nil || got.PID != cmd.Process.Pid {
				t.Fatalf("child is not an independently invoked OS process: %+v (parent=%d)", got, os.Getpid())
			}
			if got.OK != tc.wantOK || got.FailureCode != tc.wantCode {
				t.Fatalf("child process receipt was unexpected: %+v", got)
			}
			if tc.wantOK {
				if got.EvidenceID != localREAEvidenceID || got.Digest != localREADigest ||
					got.Observation != localREANodeID || got.Relation != localREARelationID ||
					got.ReadAt == "" || got.UnknownCount == 0 {
					t.Fatalf("child did not actually receive selected/partial REA Evidence: %+v", got)
				}
			} else if got.EvidenceID != "" || got.Digest != "" || got.Observation != "" ||
				got.Relation != "" || got.ReadAt != "" || got.UnknownCount != 0 {
				t.Fatalf("rejected child receipt disclosed protected Evidence metadata: %+v", got)
			}
			if strings.Contains(string(output), "/private/rea-executable") || strings.Contains(string(output), "private-not-json") {
				t.Fatalf("child leaked private Host information: %s", output)
			}
		})
	}
}

// TestLocalREAReaderChildProcessHelper executes only when the parent launches
// the *real Go test executable*. All source binding and path choice stay in
// this child Host; the parent chooses a test case but cannot supply a path.
func TestLocalREAReaderChildProcessHelper(t *testing.T) {
	if os.Getenv("TEI_REA_LOCAL_CHILD_HELPER") != "1" {
		return
	}
	host, binding, _ := newLocalREAHostFixture(t)
	reader, err := NewLocalREAReader(host, []LocalREABinding{binding})
	if err != nil {
		t.Fatal(err)
	}
	workspaceID, selector := "workspace-1", "review-one"
	switch os.Getenv("TEI_REA_LOCAL_CHILD_MODE") {
	case "allowed":
	case "unbound":
		selector = "not-configured"
	case "path":
		selector = "../../private/evidence.json"
	case "other-workspace":
		workspaceID = "workspace-2"
	case "mutated":
		if err := os.WriteFile(filepath.Join(host.roots["workspace-1"], binding.File), []byte("private-not-json"), 0600); err != nil {
			t.Fatal(err)
		}
	default:
		t.Fatal("unsupported test-only child mode")
	}
	result := reader.Read(workspaceID, selector, localREANodeID, localREARelationID)
	receipt := localREAChildReceipt{PID: os.Getpid(), OK: result.OK}
	if result.OK {
		if result.Value == nil || result.Failure != nil ||
			result.Value.Observation == nil || result.Value.Relation == nil {
			t.Fatal("successful child returned an incomplete Host projection")
		}
		receipt.EvidenceID = result.Value.Attempt.EvidenceID
		receipt.Digest = result.Value.Attempt.ReceivedDigest
		receipt.Observation = result.Value.Observation.NodeID
		receipt.Relation = result.Value.Relation.RelationID
		receipt.ReadAt = result.Value.Attempt.ReadAt
		receipt.UnknownCount = len(result.Value.Unknowns)
	} else {
		if result.Failure == nil || result.Value != nil {
			t.Fatal("denied child returned no typed failure or an unexpected value")
		}
		receipt.FailureCode = result.Failure.Code
	}
	payload, err := json.Marshal(receipt)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(localREAChildReceiptPrefix + string(payload))
}
