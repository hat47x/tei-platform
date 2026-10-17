//go:build !tei_order_only && !tei_repair_only

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
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

	"tei/internal/sourceacceptance"
	"tei/internal/sourceindex"
	"tei/internal/workspace"
)

// This Reader exists ONLY in the test executable. Its file is a synthetic Host
// publication, not a trusted ledger, a human decision, or an effect grant.
// The shipped CLI cannot select this Reader with an environment variable.
type syntheticStartupReader struct {
	publication string
	revoked     bool
}

func (r syntheticStartupReader) ReadCurrentAcceptedSource(ctx context.Context, _ string) (sourceacceptance.Record, error) {
	if err := ctx.Err(); err != nil {
		return sourceacceptance.Record{}, err
	}
	if r.revoked {
		return sourceacceptance.Record{}, errors.New("test-only Host acceptance revoked")
	}
	raw, err := os.ReadFile(r.publication)
	if err != nil {
		return sourceacceptance.Record{}, err
	}
	var record sourceacceptance.Record
	err = json.Unmarshal(raw, &record)
	return record, err
}

// Re-exec the test binary in a separate OS process, using exactly the combined
// Web startup/Runner/HTTP code used by main. This is NOT the shipped CLI with
// an authoritative Host: only the injected Reader is test-only synthetic.
func TestCombinedWebTargetHostReaderChild(t *testing.T) {
	mode := os.Getenv("TEI_TEST_WEB_SOURCE_READER")
	if mode == "" {
		return
	}
	var reader sourceacceptance.Reader
	if mode != "missing" {
		reader = syntheticStartupReader{
			publication: os.Getenv("TEI_TEST_WEB_SOURCE_PUBLICATION"),
			revoked:     mode == "revoked",
		}
	}
	separator := -1
	for i, arg := range os.Args {
		if arg == "--" {
			separator = i
			break
		}
	}
	if separator < 0 {
		os.Exit(3)
	}
	var effects HostEffectGrantReader
	if mode == "current" {
		// Test-only source acceptance and a *separate* synthetic Host Grant.
		// A real shipping binary cannot obtain either from environment.
		effects = &syntheticEffectReader{current: true}
	}
	if err := runCombinedWebTargetWithEffectGrant(context.Background(), reader, effects, os.Args[separator+1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
}

func TestCombinedWebTargetSourceGateLiveProcesses(t *testing.T) {
	// R1 is produced by the real Shared Acceptance CLI on scratch Source.
	// The helper explicitly uses test-only acceptance, without human authority.
	r1 := workspaceWithUnitPriceRule(t)
	r0 := filepath.Join(t.TempDir(), "r0")
	if err := os.CopyFS(r0, os.DirFS(referenceWorkspacePath(t))); err != nil {
		t.Fatal(err)
	}
	r2 := filepath.Join(t.TempDir(), "r2")
	if err := os.CopyFS(r2, os.DirFS(r1)); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(r2, "applications", "order-entry", "application.tei")
	file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	_, writeErr := file.WriteString("\n")
	closeErr := file.Close()
	if writeErr != nil || closeErr != nil {
		t.Fatal(writeErr, closeErr)
	}
	loadIndex := func(root string) (workspace.Result, sourceindex.Index) {
		t.Helper()
		loaded, err := workspace.Load(root)
		if err != nil {
			t.Fatal(err)
		}
		index, err := workspace.BuildSourceIndex(root, loaded)
		if err != nil {
			t.Fatal(err)
		}
		return loaded, index
	}
	world1, index1 := loadIndex(r1)
	world2, index2 := loadIndex(r2)
	_, index0 := loadIndex(r0)
	meaning1, err := json.Marshal(world1.World)
	if err != nil {
		t.Fatal(err)
	}
	meaning2, err := json.Marshal(world2.World)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(meaning1, meaning2) || sourceindex.ModelRevision(index1) == sourceindex.ModelRevision(index2) ||
		sourceindex.ModelRevision(index1) == sourceindex.ModelRevision(index0) {
		t.Fatal("R0/R1/R2 controls do not distinguish Source bytes from equivalent World meaning")
	}
	record := sourceacceptance.Record{
		ApplicationID: "order_entry", DecisionRef: "test-only-synthetic-decision",
		AuthorityRevision: "test-only-no-human-or-host-authority",
		ModelRevision:     sourceindex.ModelRevision(index1), Artifacts: index1.Artifacts,
	}
	publication := filepath.Join(t.TempDir(), "synthetic-host-publication.json")
	raw, err := json.Marshal(record)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(publication, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	repoRoot := filepath.Dir(filepath.Dir(referenceWorkspacePath(t)))
	for _, tc := range []struct {
		name, root, mode, refusal string
		diagnostic                bool
	}{
		{name: "R1", root: r1, mode: "current"},
		{name: "R1-no-effect-grant", root: r1, mode: "no-effect-grant"},
		{name: "R0", root: r0, mode: "current", refusal: "SOURCE_FINGERPRINT_MISMATCH"},
		{name: "meaning-equivalent-R2", root: r2, mode: "current", refusal: "SOURCE_FINGERPRINT_MISMATCH"},
		{name: "revoked", root: r1, mode: "revoked", refusal: "SOURCE_ACCEPTANCE_CURRENT_UNAVAILABLE"},
		{name: "missing-reader", root: "does-not-exist", mode: "missing", refusal: "SOURCE_ACCEPTANCE_READER_REQUIRED"},
		{name: "unavailable-publication", root: r1, mode: "unavailable", refusal: "SOURCE_ACCEPTANCE_CURRENT_UNAVAILABLE"},
		{name: "diagnostic-cannot-bypass-reader", root: r1, mode: "current", diagnostic: true, refusal: "SOURCE_ACCEPTANCE_DIAGNOSTIC_CONFLICT"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			reservation, err := net.Listen("tcp4", "127.0.0.1:0")
			if err != nil {
				t.Fatal(err)
			}
			address := reservation.Addr().String()
			if err := reservation.Close(); err != nil {
				t.Fatal(err)
			}
			args := []string{"-test.run=^TestCombinedWebTargetHostReaderChild$", "--",
				"-workspace", tc.root, "-application", "order_entry",
				// Identical caller-supplied labels across R0/R1/R2 cannot authorize Source.
				"-baseline-id", "B", "-baseline-revision", testBaselineRevision,
				"-target-revision", testTargetRevision, "-listen", address}
			if tc.diagnostic {
				args = append(args, "-diagnostic-unaccepted-source")
			}
			if tc.refusal != "" {
				// Source refusal must precede loading any Runner credential fixture.
				args = append(args, "-legacy-users", "does-not-exist", "-legacy-authorization", "does-not-exist")
			}
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()
			command := exec.CommandContext(ctx, executable, args...)
			command.Dir = repoRoot
			selectedPublication := publication
			if tc.mode == "unavailable" {
				selectedPublication += ".missing"
			}
			command.Env = append(os.Environ(), "TEI_TEST_WEB_SOURCE_READER="+tc.mode,
				"TEI_TEST_WEB_SOURCE_PUBLICATION="+selectedPublication)
			base := "http://" + address
			client := &http.Client{Timeout: 300 * time.Millisecond}
			if tc.refusal != "" {
				output, err := command.CombinedOutput()
				var exit *exec.ExitError
				if !errors.As(err, &exit) || exit.ExitCode() != 2 || !bytes.Contains(output, []byte(tc.refusal)) ||
					bytes.Contains(output, []byte("TEI first Web Target projection:")) {
					t.Fatalf("expected refusal %s before HTTP/Runner publication; err=%v output=%s", tc.refusal, err, output)
				}
				if response, err := client.Get(base + "/healthz"); err == nil {
					response.Body.Close()
					t.Fatal("refused Source published HTTP")
				}
				return
			}
			logfile, err := os.CreateTemp(t.TempDir(), "child-*.log")
			if err != nil {
				t.Fatal(err)
			}
			defer logfile.Close()
			command.Stdout, command.Stderr = logfile, logfile
			if err := command.Start(); err != nil {
				t.Fatal(err)
			}
			done := make(chan error, 1)
			go func() { done <- command.Wait() }()
			defer func() { cancel(); <-done }()
			for deadline := time.Now().Add(10 * time.Second); ; {
				response, err := client.Get(base + "/healthz")
				if err == nil {
					response.Body.Close()
					if response.StatusCode == http.StatusOK {
						break
					}
				}
				if time.Now().After(deadline) {
					t.Fatal("R1 did not publish HTTP within the bounded deadline")
				}
				time.Sleep(50 * time.Millisecond)
			}
			response, err := client.Post(base+"/scenario/submit", "application/json", strings.NewReader(
				`{"loginId":"legacy-order-editor","credential":"editor-secret","from":"2026-09-18T08:30:00Z","row":0}`))
			if err != nil {
				t.Fatal(err)
			}
			defer response.Body.Close()
			body, err := io.ReadAll(io.LimitReader(response.Body, 2<<20))
			if err != nil {
				t.Fatal(err)
			}
			var observed struct {
				Result         commandScenarioResult `json:"result"`
				ExecutionError string                `json:"executionError"`
			}
			if err := json.Unmarshal(body, &observed); err != nil {
				t.Fatal(err)
			}
			if tc.name == "R1-no-effect-grant" {
				if response.StatusCode != http.StatusUnprocessableEntity ||
					!strings.Contains(observed.ExecutionError, "HOST_EFFECT_GRANT_UNAVAILABLE") ||
					observed.Result.Command != nil || observed.Result.Observation != nil ||
					bytes.Contains(body, []byte("editor-secret")) {
					t.Fatalf("accepted Source without independent effect Grant dispatched Command: status=%d body=%s", response.StatusCode, body)
				}
				return
			}
			if response.StatusCode != http.StatusOK || observed.Result.Stage != "completed" ||
				observed.Result.ApplicationID != "order_entry" || observed.Result.Command == nil ||
				observed.Result.Observation == nil || observed.ExecutionError != "" ||
				len(observed.Result.Validations) != 4 || bytes.Contains(body, []byte("editor-secret")) {
				t.Fatal("R1 did not retain the four selected Validations, Command, synthetic Observation, or credential boundary")
			}
			foundPriceRule := false
			for _, result := range observed.Result.Validations {
				if result.Result != "satisfied" {
					t.Fatal("R1 Validation was not satisfied")
				}
				foundPriceRule = foundPriceRule || result.Validation == "sales.order_line.unit_price_positive"
			}
			if !foundPriceRule {
				t.Fatal("Runner used a World other than the verified R1")
			}
			if tc.name == "R1" {
				// This is an actual running HTTP process. The synthetic Host
				// publication disappears AFTER startup, before the next Command.
				// The child must refuse the new request instead of dispatching.
				if err := os.Remove(publication); err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() { _ = os.WriteFile(publication, raw, 0o600) })
				blocked, err := client.Post(base+"/scenario/submit", "application/json",
					strings.NewReader(`{"loginId":"legacy-order-editor","credential":"editor-secret","from":"2026-09-18T08:30:00Z","row":0}`))
				if err != nil {
					t.Fatal(err)
				}
				blockedBody, readErr := io.ReadAll(io.LimitReader(blocked.Body, 1024))
				closeErr := blocked.Body.Close()
				if readErr != nil || closeErr != nil || blocked.StatusCode != http.StatusServiceUnavailable ||
					!bytes.Contains(blockedBody, []byte("SOURCE_ACCEPTANCE_CURRENT_UNAVAILABLE")) ||
					bytes.Contains(blockedBody, []byte("editor-secret")) {
					t.Fatalf("revoked current Host was not refused before running Command: status=%d body=%q read=%v close=%v",
						blocked.StatusCode, blockedBody, readErr, closeErr)
				}
				if err := os.WriteFile(publication, raw, 0o600); err != nil {
					t.Fatal(err)
				}
				restored, err := client.Get(base + "/healthz")
				if err != nil {
					t.Fatal(err)
				}
				restored.Body.Close()
				if restored.StatusCode != http.StatusOK {
					t.Fatalf("restored current Host acceptance was refused: %d", restored.StatusCode)
				}
			}
		})
	}
}

func TestCombinedWebCLIRefusesUntrustedReaderEnvironment(t *testing.T) {
	root := filepath.Dir(filepath.Dir(referenceWorkspacePath(t)))
	binary := filepath.Join(t.TempDir(), "first-web-target")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	build := exec.CommandContext(ctx, "go", "build", "-o", binary, "./tools/reference-first-web-target")
	build.Dir = root
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build shipped combined executable: %v\n%s", err, output)
	}
	command := exec.CommandContext(ctx, binary, "-workspace", "does-not-exist",
		"-baseline-id", "B", "-baseline-revision", testBaselineRevision, "-target-revision", testTargetRevision)
	command.Dir = root
	command.Env = append(os.Environ(), "TEI_TEST_WEB_SOURCE_READER=current",
		"TEI_TEST_WEB_SOURCE_PUBLICATION=caller-invented-decision.json")
	output, err := command.CombinedOutput()
	var exit *exec.ExitError
	if !errors.As(err, &exit) || exit.ExitCode() != 2 || !bytes.Contains(output, []byte("SOURCE_ACCEPTANCE_READER_REQUIRED")) ||
		bytes.Contains(output, []byte("TEI first Web Target projection:")) {
		t.Fatalf("shipped CLI did not refuse before loading Source or publishing HTTP: err=%v output=%s", err, output)
	}
}
