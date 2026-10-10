//go:build !tei_order_only && !tei_repair_only

package main

import (
	"bytes"
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

// Opt-in integration: compile and boot all three REAL Go executables from the
// current checkout. Compare same-B/R/T HTTP result bytes against the original
// combined First Web Target. This uses synthetic Reference fixtures; it does
// not establish production database or authorization equivalence.
func TestTaggedBinariesPreserveFirstWebBusinessHTTPParity(t *testing.T) {
	if os.Getenv("TEI_E1_BINARY_PARITY") != "1" {
		t.Skip("set TEI_E1_BINARY_PARITY=1 to build/run combined, order-only and repair-only executables")
	}
	root := filepath.Dir(filepath.Dir(referenceWorkspacePath(t)))
	startHead, err := requireCleanCheckoutAtHead(root)
	if err != nil {
		t.Fatalf("binary parity requires a clean, identified checkout: %v", err)
	}
	outputDir := t.TempDir()
	build := func(name, tags string) string {
		t.Helper()
		target := filepath.Join(outputDir, name)
		args := []string{"build", "-trimpath", "-ldflags=-buildid=", "-o", target}
		if tags != "" {
			args = append(args, "-tags="+tags)
		}
		args = append(args, "./tools/reference-first-web-target")
		command := exec.Command("go", args...)
		command.Dir = root
		command.Env = append(os.Environ(), "GOENV=off", "GOWORK=off", "GOFLAGS=", "CGO_ENABLED=0")
		output, err := command.CombinedOutput()
		if err != nil {
			t.Fatalf("build %s (tags=%q): %v\n%s", name, tags, err, output)
		}
		return target
	}
	combined := build("combined", "")
	orderOnly := build("order", "tei_order_only")
	repairOnly := build("repair", "tei_repair_only")
	for _, testCase := range []struct {
		app         string
		taggedBinary string
		requests    []taggedParityRequest
		foreign     []taggedParityRequest
	}{
		{
			app: "order_entry", taggedBinary: orderOnly,
			requests: []taggedParityRequest{
				{http.MethodGet, "/manifest.json", "", http.StatusOK},
				{http.MethodGet, "/scenario/read?from=2026-09-18T08:30:00Z&row=0", "", http.StatusOK},
				{http.MethodGet, "/scenario/read?row=-1", "", http.StatusBadRequest},
				{http.MethodPost, "/scenario/submit", `{"loginId":"legacy-order-editor","credential":"editor-secret","from":"2026-09-18T08:30:00Z","row":0}`, http.StatusOK},
				{http.MethodPost, "/scenario/submit", `{"loginId":"legacy-order-viewer","credential":"viewer-secret","from":"2026-09-18T08:30:00Z","row":0}`, http.StatusOK},
				{http.MethodPost, "/scenario/submit", `{"loginId":"legacy-order-editor","credential":"wrong-secret","from":"2026-09-18T08:30:00Z","row":0}`, http.StatusOK},
				{http.MethodPost, "/scenario/submit", `{"loginId":"legacy-order-editor","credential":"editor-secret","from":"2026-09-18T07:00:00Z","row":2}`, http.StatusUnprocessableEntity},
				{http.MethodPost, "/scenario/submit", `{"unknown":"field"}`, http.StatusBadRequest},
			},
			foreign: []taggedParityRequest{
				{http.MethodGet, "/scenario/customer-lookup?customerId=11", "", http.StatusNotFound},
				{http.MethodGet, "/customer-lookup", "", http.StatusNotFound},
			},
		},
		{
			app: "repair_intake", taggedBinary: repairOnly,
			requests: []taggedParityRequest{
				{http.MethodGet, "/manifest.json", "", http.StatusOK},
				{http.MethodGet, "/scenario/customer-lookup?customerId=11", "", http.StatusOK},
				{http.MethodGet, "/scenario/customer-lookup?customerId=abc", "", http.StatusBadRequest},
				{http.MethodGet, "/scenario/customer-lookup", "", http.StatusBadRequest},
				{http.MethodPost, "/scenario/customer-lookup?customerId=11", "", http.StatusMethodNotAllowed},
			},
			foreign: []taggedParityRequest{
				{http.MethodGet, "/scenario/read", "", http.StatusNotFound},
				{http.MethodPost, "/scenario/submit", `{}`, http.StatusNotFound},
				{http.MethodGet, "/custom-submit", "", http.StatusNotFound},
			},
		},
	} {
		t.Run(testCase.app, func(t *testing.T) {
			combinedBase := launchTaggedParityBinary(t, root, combined, testCase.app)
			taggedBase := launchTaggedParityBinary(t, root, testCase.taggedBinary, testCase.app)
			for _, request := range testCase.requests {
				t.Run(request.method+" "+request.path+" "+request.body, func(t *testing.T) {
					legacy := requestTaggedParityHTTP(t, combinedBase, request)
					tagged := requestTaggedParityHTTP(t, taggedBase, request)
					if legacy.status != request.status || tagged.status != request.status {
						t.Fatalf("%s %s: old status=%d, tagged status=%d, expected=%d",
							request.method, request.path, legacy.status, tagged.status, request.status)
					}
					if legacy.contentType != tagged.contentType || !bytes.Equal(legacy.body, tagged.body) {
						t.Fatalf("%s %s result/authorization/observation drift:\nold=%s\ntagged=%s",
							request.method, request.path, legacy.body, tagged.body)
					}
					assertTaggedBusinessSemantics(t, request, tagged)
					for _, secret := range []string{"editor-secret", "viewer-secret", "wrong-secret"} {
						if bytes.Contains(tagged.body, []byte(secret)) {
							t.Fatal("credential leaked into tagged business response")
						}
					}
				})
			}
			for _, request := range testCase.foreign {
				response := requestTaggedParityHTTP(t, taggedBase, request)
				if response.status != http.StatusNotFound {
					t.Errorf("foreign Application route %s %s returned %d, expected 404",
						request.method, request.path, response.status)
				}
			}
		})
	}
	endHead, err := requireCleanCheckoutAtHead(root)
	if err != nil {
		t.Fatalf("source checkout changed during business parity: %v", err)
	}
	if endHead != startHead {
		t.Fatalf("binary parity crossed source revisions: before=%s after=%s", startHead, endHead)
	}
}


// Body parity alone could allow an identical wrong result in both executables.
// These assertions anchor the accepted synthetic Reference scenario semantics
// independently of the new tagged entry implementation.
func assertTaggedBusinessSemantics(t *testing.T, request taggedParityRequest, response taggedParityResponse) {
	t.Helper()
	if request.path == "/manifest.json" {
		var manifest struct {
			ApplicationID              string `json:"applicationId"`
			HandoverBaselineID         string `json:"handoverBaselineId"`
			BaselineRepositoryRevision string `json:"baselineRepositoryRevision"`
			TargetBuildRevision        string `json:"targetBuildRevision"`
		}
		if err := json.Unmarshal(response.body, &manifest); err != nil {
			t.Fatal(err)
		}
		if manifest.HandoverBaselineID != testBaselineID ||
			manifest.BaselineRepositoryRevision != testBaselineRevision ||
			manifest.TargetBuildRevision != testTargetRevision ||
			(manifest.ApplicationID != "order_entry" && manifest.ApplicationID != "repair_intake") {
			t.Fatalf("tagged manifest lost accepted B/R/T or Application: %+v", manifest)
		}
	}
	if request.method == http.MethodPost && request.path == "/scenario/submit" {
		if response.status == http.StatusBadRequest {
			return
		}
		var command struct {
			Result struct {
				ScenarioID     string           `json:"scenarioId"`
				ApplicationID  string           `json:"applicationId"`
				Stage          string           `json:"stage"`
				Authentication *json.RawMessage `json:"authentication"`
				Authorization  *json.RawMessage `json:"authorization"`
				Command        *json.RawMessage `json:"command"`
				Observation    *json.RawMessage `json:"observation"`
			} `json:"result"`
			ExecutionError string `json:"executionError"`
		}
		if err := json.Unmarshal(response.body, &command); err != nil {
			t.Fatal(err)
		}
		if command.Result.ApplicationID != "order_entry" ||
			command.Result.ScenarioID != "order-entry.submit-order-web-slice" {
			t.Fatalf("wrong command business identity: %+v", command.Result)
		}
		if response.status == http.StatusUnprocessableEntity {
			if command.ExecutionError == "" {
				t.Fatal("Execution Error was silently converted into a Business Result")
			}
			return
		}
		expectedStage := "completed"
		switch {
		case strings.Contains(request.body, "wrong-secret"):
			expectedStage = "authentication-denied"
		case strings.Contains(request.body, "legacy-order-viewer"):
			expectedStage = "authorization-denied"
		}
		if command.Result.Stage != expectedStage || command.ExecutionError != "" {
			t.Fatalf("command business stage=%q, expected=%q, executionError=%q",
				command.Result.Stage, expectedStage, command.ExecutionError)
		}
		if expectedStage == "completed" {
			if command.Result.Command == nil || command.Result.Observation == nil ||
				command.Result.Authorization == nil || command.Result.Authentication == nil {
				t.Fatal("completed Command omitted business/auth/observation boundaries")
			}
		} else if command.Result.Command != nil || command.Result.Observation != nil {
			t.Fatal("denied Command exposed command/observation business result")
		}
	}
	if request.method == http.MethodGet &&
		strings.HasPrefix(request.path, "/scenario/customer-lookup?customerId=11") {
		var lookup struct {
			Result struct {
				ApplicationID string `json:"applicationId"`
				ScenarioID    string `json:"scenarioId"`
				CustomerRows  []any  `json:"customerRows"`
			} `json:"result"`
		}
		if err := json.Unmarshal(response.body, &lookup); err != nil {
			t.Fatal(err)
		}
		if lookup.Result.ApplicationID != "repair_intake" ||
			lookup.Result.ScenarioID != "repair-intake.customer-lookup-web-slice" ||
			len(lookup.Result.CustomerRows) != 1 {
			t.Fatalf("repair lookup lost Driver-result semantics: %+v", lookup.Result)
		}
	}
}

type taggedParityRequest struct {
	method, path, body string
	status             int
}

type taggedParityResponse struct {
	status      int
	contentType string
	body        []byte
}

func requestTaggedParityHTTP(t *testing.T, base string, input taggedParityRequest) taggedParityResponse {
	t.Helper()
	request, err := http.NewRequest(input.method, base+input.path, strings.NewReader(input.body))
	if err != nil {
		t.Fatal(err)
	}
	if input.body != "" {
		request.Header.Set("Content-Type", "application/json")
	}
	client := &http.Client{Timeout: 5 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	}}
	response, err := client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	payload, err := io.ReadAll(io.LimitReader(response.Body, 4<<20+1))
	if err != nil {
		t.Fatal(err)
	}
	if len(payload) > 4<<20 {
		t.Fatal("business HTTP response exceeds bounded parity evidence size")
	}
	return taggedParityResponse{
		status: response.StatusCode, contentType: response.Header.Get("Content-Type"), body: payload,
	}
}

func launchTaggedParityBinary(t *testing.T, root, binary, app string) string {
	t.Helper()
	reservation, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	address := reservation.Addr().String()
	if err := reservation.Close(); err != nil {
		t.Fatal(err)
	}
	logFile, err := os.CreateTemp(t.TempDir(), "target-*.log")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = logFile.Close() })
	command := exec.Command(binary,
		"-application", app,
		"-baseline-id", testBaselineID,
		"-baseline-revision", testBaselineRevision,
		"-target-revision", testTargetRevision,
		"-workspace", "reference/workspace",
		"-legacy-users", "reference/legacy-auth/users.json",
		"-legacy-authorization", "reference/legacy-auth/authorization.json",
		"-listen", address,
	)
	command.Dir = root
	command.Stdout, command.Stderr = logFile, logFile
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if command.Process != nil {
			_ = command.Process.Kill()
			_ = command.Wait()
		}
	})
	base := "http://" + address
	deadline := time.Now().Add(15 * time.Second)
	client := &http.Client{Timeout: 200 * time.Millisecond}
	for time.Now().Before(deadline) {
		reply, err := client.Get(base + "/healthz")
		if err == nil {
			_ = reply.Body.Close()
			if reply.StatusCode == http.StatusOK {
				return base
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	logData, _ := os.ReadFile(logFile.Name())
	t.Fatalf("binary %s (%s) never became healthy: %s", binary, app, logData)
	return ""
}

func requireCleanCheckoutAtHead(root string) (string, error) {
	var head string
	for _, args := range [][]string{
		{"rev-parse", "--verify", "HEAD"},
		{"status", "--porcelain", "--untracked-files=all"},
	} {
		cmd := exec.Command("git", args...)
		cmd.Dir = root
		output, err := cmd.CombinedOutput()
		if err != nil {
			return "", fmt.Errorf("git %v: %w: %s", args, err, output)
		}
		if args[0] == "status" && len(bytes.TrimSpace(output)) != 0 {
			return "", fmt.Errorf("dirty checkout cannot be used for binary parity evidence")
		}
		if args[0] == "rev-parse" {
			head = strings.TrimSpace(string(output))
		}
	}
	if len(head) != 40 {
		return "", fmt.Errorf("unexpected Git revision %q", head)
	}
	return head, nil
}
