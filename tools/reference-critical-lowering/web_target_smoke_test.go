package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Opt-in black-box evidence path against the EXISTING Web Target binary.
// This does not put the synthetic lowering Trust into the Web's Host Authority.
// Run with TEI_LOWERING_WEB_SMOKE=1 go test -count=1 -run TestWebTargetHTTPBlackBoxTrace ./tools/reference-critical-lowering
func TestWebTargetHTTPBlackBoxTrace(t *testing.T) {
	if os.Getenv("TEI_LOWERING_WEB_SMOKE") != "1" {
		t.Skip("set TEI_LOWERING_WEB_SMOKE=1 for the actual Web HTTP black-box trace")
	}

	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	headCmd := exec.Command("git", "rev-parse", "HEAD")
	headCmd.Dir = root
	headOut, err := headCmd.Output()
	if err != nil {
		t.Fatalf("exact checkout HEAD unavailable: %v", err)
	}
	head := strings.TrimSpace(string(headOut))
	if len(head) != 40 {
		t.Fatalf("invalid checkout revision %q", head)
	}

	buildCtx, cancelBuild := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancelBuild()
	binary := filepath.Join(t.TempDir(), "first-web-target")
	build := exec.CommandContext(buildCtx, "go", "build", "-o", binary, "./tools/reference-first-web-target")
	build.Dir = root
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build existing Web Target: %v\n%s", err, output)
	}

	// A loopback-only, ephemeral port minimizes interference with other lanes.
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	address := listener.Addr().String()
	if err := listener.Close(); err != nil {
		t.Fatal(err)
	}

	runCtx, cancelRun := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancelRun()
	proc := exec.CommandContext(runCtx, binary,
		"-workspace", "reference/workspace",
		"-application", "order_entry",
		"-baseline-id", "B",
		"-baseline-revision", head,
		"-target-revision", head,
		"-legacy-users", "reference/legacy-auth/users.json",
		"-legacy-authorization", "reference/legacy-auth/authorization.json",
		"-listen", address)
	proc.Dir = root
	var serverLog bytes.Buffer
	proc.Stdout = &serverLog
	proc.Stderr = &serverLog
	if err := proc.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		cancelRun()
		_ = proc.Wait()
	}()

	client := &http.Client{Timeout: 3 * time.Second}
	base := "http://" + address
	ready := false
	for deadline := time.Now().Add(15 * time.Second); time.Now().Before(deadline); {
		resp, requestErr := client.Get(base + "/healthz")
		if requestErr == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				ready = true
				break
			}
		}
		select {
		case <-runCtx.Done():
			t.Fatalf("Web Target exited or timed out before readiness: %v\n%s", runCtx.Err(), serverLog.String())
		case <-time.After(125 * time.Millisecond):
		}
	}
	if !ready {
		t.Fatalf("Web Target did not become healthy\n%s", serverLog.String())
	}

	get := func(path string) ([]byte, int) {
		t.Helper()
		resp, err := client.Get(base + path)
		if err != nil {
			t.Fatalf("GET %s: %v", path, err)
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
		if err != nil {
			t.Fatal(err)
		}
		return body, resp.StatusCode
	}
	var manifest struct {
		ApplicationID string `json:"applicationId"`
		TargetBuildRevision string `json:"targetBuildRevision"`
		Patterns []struct {
			ID string `json:"id"`
			Kind string `json:"kind"`
			Capability string `json:"capability"`
			Data string `json:"data"`
		} `json:"patterns"`
	}
	manifestBytes, status := get("/manifest.json")
	if status != http.StatusOK {
		t.Fatalf("manifest status=%d", status)
	}
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.ApplicationID != "order_entry" || manifest.TargetBuildRevision != head {
		t.Fatalf("Web Target manifest identity mismatch: %+v", manifest)
	}
	// Compare real Web projection to the same semantic subject references
	// used by the synthetic lowering contract; the Acceptance receipt remains a stub.
	f := orderFixture("tei.language.dsl")
	neutral, err := Neutralize(f.c, f.a, stubTrust{})
	if err != nil {
		t.Fatal(err)
	}
	if neutral.Snapshot != f.a.Snapshot || len(f.c.Subjects) != 3 {
		t.Fatal("lowering scenario semantic boundary lost")
	}
	businessID := f.c.Subjects[0].Ref.ID
	dataID := f.c.Subjects[1].Ref.ID
	functionID := f.c.Subjects[2].Ref.ID
	seen := map[string]string{}
	for _, pattern := range manifest.Patterns {
		seen[pattern.ID] = pattern.Kind
		if pattern.ID == businessID && pattern.Capability != functionID {
			t.Fatalf("Source->Web command capability changed: %+v", pattern)
		}
		if pattern.ID == "order_entry.recent_orders" && pattern.Data != dataID {
			t.Fatalf("Source->Web data identity changed: %+v", pattern)
		}
	}
	for id, kind := range map[string]string{
		"order_entry.submit_order": "command",
		"order_entry.recent_orders": "browse",
		"order_entry.customer_reference": "reference",
		"order_entry.order_lines": "header-detail",
	} {
		if seen[id] != kind {
			t.Fatalf("Source->Web pattern %s=%q, expected %q", id, seen[id], kind)
		}
	}

	post := func(payload string, expectedStatus int) map[string]json.RawMessage {
		t.Helper()
		req, err := http.NewRequest(http.MethodPost, base+"/scenario/submit", strings.NewReader(payload))
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		data, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != expectedStatus {
			t.Fatalf("submit expected HTTP %d, got %d: %s", expectedStatus, resp.StatusCode, data)
		}
		if bytes.Contains(data, []byte("editor-secret")) || bytes.Contains(data, []byte("wrong-secret")) {
			t.Fatal("runtime credential leaked into HTTP evidence")
		}
		var envelope map[string]json.RawMessage
		if err := json.Unmarshal(data, &envelope); err != nil {
			t.Fatal(err)
		}
		if string(envelope["targetBuildRevision"]) != fmt.Sprintf("%q", head) {
			t.Fatalf("wrong Target revision evidence: %s", envelope["targetBuildRevision"])
		}
		return envelope
	}

	success := post(`{"loginId":"legacy-order-editor","credential":"editor-secret","from":"2026-09-18T08:30:00Z","row":0}`, http.StatusOK)
	var result struct {
		Stage string `json:"stage"`
		Authentication *struct { Outcome string `json:"outcome"` } `json:"authentication"`
		Validations []struct { Result string `json:"result"` } `json:"validations"`
		Authorization *struct { Outcome string `json:"outcome"` } `json:"authorization"`
		Command *struct { Capability string `json:"capability"` } `json:"command"`
		Observation *struct { Nature string `json:"nature"` } `json:"observation"`
	}
	if err := json.Unmarshal(success["result"], &result); err != nil {
		t.Fatal(err)
	}
	if result.Stage != "completed" || result.Authentication == nil ||
		result.Authentication.Outcome != "authenticated" || len(result.Validations) != 3 ||
		result.Authorization == nil || result.Authorization.Outcome != "allowed" ||
		result.Command == nil || result.Command.Capability != "sales.order.submit" ||
		result.Observation == nil || result.Observation.Nature != "synthetic" {
		t.Fatalf("Web Command stage trace incomplete: %+v", result)
	}
	for _, validation := range result.Validations {
		if validation.Result != "satisfied" {
			t.Fatalf("validation stage lost: %+v", validation)
		}
	}

	denied := post(`{"loginId":"legacy-order-editor","credential":"wrong-secret","from":"2026-09-18T08:30:00Z","row":0}`, http.StatusOK)
	var denial struct {
		Stage string `json:"stage"`
		Command json.RawMessage `json:"command"`
		Observation json.RawMessage `json:"observation"`
	}
	if err := json.Unmarshal(denied["result"], &denial); err != nil {
		t.Fatal(err)
	}
	if denial.Stage != "authentication-denied" || len(denial.Command) != 0 || len(denial.Observation) != 0 {
		t.Fatalf("business denial continued into command: %+v", denial)
	}

	failed := post(`{"loginId":"legacy-order-editor","credential":"editor-secret","from":"2026-09-18T07:00:00Z","row":2}`, http.StatusUnprocessableEntity)
	var errorResult struct { Stage string `json:"stage"` }
	if err := json.Unmarshal(failed["result"], &errorResult); err != nil {
		t.Fatal(err)
	}
	if errorResult.Stage != "command" || len(failed["executionError"]) == 0 {
		t.Fatalf("execution error lost its stage / evidence: %+v", failed)
	}
}
