package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"
)

// TestBusinessDFDUnacceptedProposalReprojection exercises a complete,
// temporary reference-review round trip. It never accepts or writes Canonical
// Source, and the resulting projection remains derived-read-only.
func TestBusinessDFDUnacceptedProposalReprojection(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		if os.Getenv("GITHUB_ACTIONS") == "true" {
			t.Fatal("DFD scratch reprojection requires Node.js in GitHub Actions")
		}
		t.Skip("DFD scratch reprojection not executed: Node.js is unavailable")
	}

	root := filepath.Join("..", "..", "reference", "business-system-design")
	procedurePath := filepath.Join(root, "dfd-order-approval", "order-approval-procedure.json")
	reviewPath := filepath.Join(root, "dfd-order-approval", "review-proposal.json")
	setPath := filepath.Join(root, "dfd-order-approval", "procedure-set.json")
	workspaceDir := filepath.Join("..", "..", "reference", "workspace")

	before, err := run(procedurePath, reviewPath, setPath, workspaceDir)
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range before.Diagnostics {
		if d.Severity == "error" {
			t.Fatalf("original DFD invalid: %+v", d)
		}
	}
	projectionBytes, err := json.Marshal(before)
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
	var original review
	if err := json.Unmarshal(reviewBytes, &original); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name      string
		kind      string
		id        string
		operation map[string]any
		apply     func(*review)
	}{
		{
			name: "trigger_business_description_only",
			kind: "trigger", id: "TR-ORDER",
			operation: map[string]any{
				"kind": "replace-trigger-label", "trigger": "TR-ORDER",
				"label": "受注を確認するための業務契機",
			},
			apply: func(r *review) { r.Triggers[0].Label = "受注を確認するための業務契機" },
		},
		{
			name: "manual_step_business_role_not_rbac",
			kind: "step", id: "APP-10",
			operation: map[string]any{
				"kind": "assign-business-role", "step": "APP-10", "role": "approver",
			},
			apply: func(r *review) {
				r.Responsibilities = append(r.Responsibilities, stepResponsibility{Step: "APP-10", Role: "approver"})
			},
		},
		{
			name: "conditional_exception_not_workflow",
			kind: "step", id: "APP-10",
			operation: map[string]any{
				"kind": "add-control-transition", "from": "APP-10", "to": "APP-40",
				"transitionKind": "exception", "condition": "担当者が例外的に保留した場合",
			},
			apply: func(r *review) {
				r.Flows = append(r.Flows, reviewFlow{
					From: "APP-10", To: "APP-40", Kind: "exception",
					Condition: "担当者が例外的に保留した場合",
				})
			},
		},
		{
			name: "incoming_party_information_not_runtime",
			kind: "party", id: "supplier",
			operation: map[string]any{
				"kind": "add-information-flow", "id": "INFO-SCRATCH-IN",
				"from":    map[string]string{"kind": "party", "id": "supplier"},
				"to":      map[string]string{"kind": "step", "id": "APP-20"},
				"payload": "在庫の確認結果",
			},
			apply: func(r *review) {
				r.InformationFlows = append(r.InformationFlows, informationFlow{
					ID:      "INFO-SCRATCH-IN",
					From:    informationEndpoint{Kind: "party", ID: "supplier"},
					To:      informationEndpoint{Kind: "step", ID: "APP-20"},
					Payload: "在庫の確認結果",
				})
			},
		},
		{
			name: "outgoing_store_information_not_database_write",
			kind: "step", id: "APP-10",
			operation: map[string]any{
				"kind": "add-information-flow", "id": "INFO-SCRATCH-OUT",
				"from":    map[string]string{"kind": "step", "id": "APP-10"},
				"to":      map[string]string{"kind": "store", "id": "orders"},
				"payload": "注文確認の参考情報",
			},
			apply: func(r *review) {
				r.InformationFlows = append(r.InformationFlows, informationFlow{
					ID:      "INFO-SCRATCH-OUT",
					From:    informationEndpoint{Kind: "step", ID: "APP-10"},
					To:      informationEndpoint{Kind: "store", ID: "orders"},
					Payload: "注文確認の参考情報",
				})
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			files := map[string]string{
				"projection": filepath.Join(dir, "projection.json"),
				"procedure":  filepath.Join(dir, "procedure.json"),
				"review":     filepath.Join(dir, "review.json"),
				"binding":    filepath.Join(dir, "binding.json"),
			}
			inputs := map[string][]byte{
				"projection": projectionBytes,
				"procedure":  procedureBytes,
				"review":     reviewBytes,
			}
			for name, data := range inputs {
				if err := os.WriteFile(files[name], data, 0600); err != nil {
					t.Fatal(err)
				}
			}
			binding := map[string]any{
				"schema":           "tei.reference.business-dfd-selected-edit/v0",
				"id":               "scratch-reprojection-" + tc.name,
				"origin":           "cli",
				"intent":           "一時領域で未受理の業務レビュー案を確認する",
				"procedureSHA256":  dfdTestDigest(procedureBytes),
				"reviewSHA256":     dfdTestDigest(reviewBytes),
				"projectionSHA256": dfdTestDigest(projectionBytes),
				"selection":        map[string]string{"kind": tc.kind, "id": tc.id},
				"operation":        tc.operation,
			}
			bindingBytes, err := json.Marshal(binding)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(files["binding"], bindingBytes, 0600); err != nil {
				t.Fatal(err)
			}

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			runNode := func(reviewFile string) ([]byte, []byte, error) {
				cmd := exec.CommandContext(ctx, node, filepath.Join(root, "dfd-selected-edit-cli.mjs"),
					"-projection", files["projection"], "-procedure", files["procedure"],
					"-review", reviewFile, "-binding", files["binding"])
				var stdout, stderr bytes.Buffer
				cmd.Stdout, cmd.Stderr = &stdout, &stderr
				err := cmd.Run()
				if ctx.Err() != nil {
					t.Fatalf("Node review timed out: %v", ctx.Err())
				}
				return stdout.Bytes(), stderr.Bytes(), err
			}
			stdout, stderr, err := runNode(files["review"])
			if err != nil {
				t.Fatalf("Node selected-edit failed: %v\n%s", err, stderr)
			}
			var packet struct {
				Authority                 string            `json:"authority"`
				ReadyForAcceptance        bool              `json:"readyForAcceptance"`
				SharedAcceptanceCandidate *json.RawMessage  `json:"sharedAcceptanceCandidate"`
				SourceWrites              []json.RawMessage `json:"sourceWrites"`
				InputSnapshotsVerified    bool              `json:"inputSnapshotsVerified"`
				ProposedReview            json.RawMessage   `json:"proposedReview"`
				Selected                  struct {
					ID               string `json:"id"`
					AcceptedIdentity string `json:"acceptedIdentity"`
				} `json:"selected"`
			}
			if err := json.Unmarshal(stdout, &packet); err != nil {
				t.Fatalf("invalid proposal packet: %v\n%s", err, stdout)
			}
			if packet.Authority != "unaccepted-proposal-only" ||
				packet.ReadyForAcceptance || packet.SharedAcceptanceCandidate != nil ||
				len(packet.SourceWrites) != 0 || !packet.InputSnapshotsVerified ||
				packet.Selected.ID != tc.id || packet.Selected.AcceptedIdentity != "unverified" {
				t.Fatalf("proposal escalated authority or lost selection: %+v", packet)
			}
			var actual review
			if err := json.Unmarshal(packet.ProposedReview, &actual); err != nil {
				t.Fatal(err)
			}
			var expected review
			if err := json.Unmarshal(reviewBytes, &expected); err != nil {
				t.Fatal(err)
			}
			tc.apply(&expected)
			if !reflect.DeepEqual(actual, expected) {
				t.Fatalf("review contains changes beyond the requested edit\nactual: %+v\nexpected: %+v", actual, expected)
			}
			if actual.Authority != "proposal" || actual.Procedure != original.Procedure {
				t.Fatal("review silently gained accepted Procedure authority")
			}

			// Re-derive the changed DFD using the *real* Go producer, with the
			// candidate review isolated in a temporary file.
			nextFile := filepath.Join(dir, "next-unaccepted-review.json")
			if err := os.WriteFile(nextFile, packet.ProposedReview, 0600); err != nil {
				t.Fatal(err)
			}
			after, err := run(procedurePath, nextFile, setPath, workspaceDir)
			if err != nil {
				t.Fatalf("Go reprojection failed: %v", err)
			}
			for _, d := range after.Diagnostics {
				if d.Severity == "error" {
					t.Fatalf("Go rejected proposed review: %+v", d)
				}
			}
			if after.Authority != "derived-read-only" || after.CanonicalProcedureUsed ||
				after.RoleAuthorizationBinding != "not-inferred" ||
				after.ProcedureID != before.ProcedureID ||
				!reflect.DeepEqual(before.Steps, after.Steps) ||
				!reflect.DeepEqual(before.Handoffs, after.Handoffs) {
				t.Fatal("scratch reprojection altered Canonical/Step/Handoff authority")
			}
			if len(after.Triggers) != len(actual.Triggers) ||
				len(after.Flows) != len(actual.Flows) ||
				len(after.Information.Responsibilities) != len(actual.Responsibilities) ||
				len(after.Information.InformationFlows) != len(actual.InformationFlows) {
				t.Fatal("scratch reprojection lost a reviewed business element")
			}
			for i, trigger := range after.Triggers {
				r := actual.Triggers[i]
				if trigger.ID != r.ID || trigger.Label != r.Label ||
					trigger.Kind != r.Kind || trigger.TargetStep != r.TargetStep ||
					trigger.Authority != "proposal" || trigger.RuntimeBinding != "unbound" ||
					trigger.Source.File != nextFile {
					t.Fatalf("trigger not reprojected as proposal: %+v", trigger)
				}
			}
			for i, flow := range after.Flows {
				r := actual.Flows[i]
				if flow.From != r.From || flow.To != r.To || flow.Kind != r.Kind ||
					flow.Condition != r.Condition || flow.Authority != "proposal" ||
					flow.Source.File != nextFile {
					t.Fatalf("control flow mismatch after reprojection: %+v", flow)
				}
			}
			for i, responsibility := range after.Information.Responsibilities {
				r := actual.Responsibilities[i]
				if responsibility.Step != r.Step || responsibility.Role != r.Role ||
					responsibility.Authority != "proposal" || responsibility.Source.File != nextFile {
					t.Fatalf("business responsibility mismatch: %+v", responsibility)
				}
			}
			for i, info := range after.Information.InformationFlows {
				r := actual.InformationFlows[i]
				if info.ID != r.ID || info.From != r.From || info.To != r.To ||
					info.Payload != r.Payload || info.Authority != "proposal" ||
					info.Source.File != nextFile {
					t.Fatalf("information transfer mismatch: %+v", info)
				}
			}
			for name, originalBytes := range inputs {
				nextBytes, err := os.ReadFile(files[name])
				if err != nil || !bytes.Equal(nextBytes, originalBytes) {
					t.Fatalf("original %s snapshot was modified: %v", name, err)
				}
			}
			nextOriginal, err := os.ReadFile(reviewPath)
			if err != nil || !bytes.Equal(nextOriginal, reviewBytes) {
				t.Fatal("original repository review was modified")
			}

			// The old binding cannot be reused against new review bytes.
			replayed, replayErrors, err := runNode(nextFile)
			if err == nil || len(replayed) != 0 ||
				!strings.Contains(string(replayErrors), "STALE_OR_UNVERIFIED_DFD_EDIT_INPUT_SNAPSHOT") {
				t.Fatalf("stale review replay must fail closed: err=%v stdout=%s stderr=%s",
					err, replayed, replayErrors)
			}
		})
	}
}
