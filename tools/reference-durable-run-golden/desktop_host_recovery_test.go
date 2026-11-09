package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"tei/internal/deployment"
	"tei/internal/deployment/durable"
	"tei/internal/deployment/executor"
)

// The child process receives only the persisted Durable Run and the
// independent observed-state fixture. No previous Host memory, command object,
// executor callback, or retry authorization is carried across the restart.
// This is a reference test for already-recorded acknowledgements, not a
// production Host Run store or a crash-atomic dispatch protocol.
func TestDesktopHostReopensUnknownRunWithoutCommandReplay(t *testing.T) {
	if os.Getenv("TEI_DESKTOP_DURABLE_RECOVERY_CHILD") == "yes" {
		runFile := os.Getenv("TEI_DESKTOP_DURABLE_RUN_FILE")
		observedFile := os.Getenv("TEI_DESKTOP_DURABLE_OBSERVED_FILE")
		expected := executor.Outcome(os.Getenv("TEI_DESKTOP_DURABLE_EXPECTED_OUTCOME"))
		if !filepath.IsAbs(runFile) || !filepath.IsAbs(observedFile) {
			t.Fatal("child requires Host-provided absolute fixture paths")
		}
		raw, err := os.ReadFile(runFile)
		if err != nil {
			t.Fatal(err)
		}
		var run durable.Run
		if err := json.Unmarshal(raw, &run); err != nil {
			t.Fatal(err)
		}
		if len(run.Attempts) != 1 || run.Attempts[0].Outcome != expected ||
			run.Attempts[0].AttemptID != "desktop-attempt-1" ||
			run.Status != durable.RunWaitingObservation ||
			len(run.WaitingActions) != 1 {
			t.Fatalf("persisted command acknowledgement was not restored: %+v", run)
		}
		if cmds := durable.ReadyCommands(run); len(cmds) != 0 {
			t.Fatalf("restarted Host would replay an uncertain command: %+v", cmds)
		}

		// Reload observed state independently of the serialized Run. Only the
		// existing Durable Run reconciler decides if an action completed.
		observedBytes, err := os.ReadFile(observedFile)
		if err != nil {
			t.Fatal(err)
		}
		var observed deployment.Snapshot
		if err := json.Unmarshal(observedBytes, &observed); err != nil {
			t.Fatal(err)
		}
		checked := durable.Reconcile(run, observed, nil)
		if checked.Status != durable.RunWaitingObservation ||
			len(checked.Attempts) != 1 || len(durable.ReadyCommands(checked)) != 0 {
			t.Fatalf("fresh observation wrongly made uncertain command dispatchable: %+v", checked)
		}
		// Resume alone must not convert a waiting-observation Run into a
		// dispatch instruction, even after Host restart.
		resumed := durable.Resume(checked)
		if resumed.Status != durable.RunWaitingObservation ||
			len(durable.ReadyCommands(resumed)) != 0 {
			t.Fatalf("resume bypassed observation/retry boundary: %+v", resumed)
		}
		fmt.Println("TEI_DESKTOP_DURABLE_NO_REPLAY:" + string(expected))
		return
	}

	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate reference fixtures")
	}
	repositoryRoot := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", ".."))
	observedFile := filepath.Join(repositoryRoot, "reference", "observed", "production.deployment-state.json")
	generated, err := generateDurableRun(
		filepath.Join(repositoryRoot, defaultWorkspacePath),
		observedFile,
		filepath.Join(repositoryRoot, defaultPluginLockPath),
		defaultSolutionID,
		defaultVersion,
		"desktop-recovery-reference",
	)
	if err != nil {
		t.Fatal(err)
	}
	var initial durable.Run
	if err := json.Unmarshal(generated, &initial); err != nil {
		t.Fatal(err)
	}
	ready := durable.ReadyCommands(initial)
	if len(ready) != 1 {
		t.Fatalf("reference fixture needs one initial command: %+v", ready)
	}

	for _, outcome := range []executor.Outcome{executor.OutcomeUnknown, executor.OutcomeAccepted} {
		t.Run(string(outcome), func(t *testing.T) {
			result := executor.Result{
				ExecutorID:     "reference-executor",
				AttemptID:      "desktop-attempt-1",
				ActionID:       ready[0].ActionID,
				IdempotencyKey: ready[0].IdempotencyKey,
				Outcome:        outcome,
				ExternalRef:    "job://pending-observation",
			}
			recorded := durable.RecordExecutorResult(initial, result)
			if recorded.Status != durable.RunWaitingObservation ||
				len(durable.ReadyCommands(recorded)) != 0 {
				t.Fatalf("executor acknowledgement did not fence replay: %+v", recorded)
			}
			data, err := json.Marshal(recorded)
			if err != nil {
				t.Fatal(err)
			}
			// Simulate a Host-owned checkpoint outside Canonical Workspace.
			// Neither candidate content nor external command is re-executed.
			runFile := filepath.Join(t.TempDir(), "recorded-run.json")
			if err := os.WriteFile(runFile, data, 0o600); err != nil {
				t.Fatal(err)
			}
			cmd := exec.Command(os.Args[0], "-test.run=^TestDesktopHostReopensUnknownRunWithoutCommandReplay$", "-test.v")
			cmd.Env = append(os.Environ(),
				"TEI_DESKTOP_DURABLE_RECOVERY_CHILD=yes",
				"TEI_DESKTOP_DURABLE_RUN_FILE="+runFile,
				"TEI_DESKTOP_DURABLE_OBSERVED_FILE="+observedFile,
				"TEI_DESKTOP_DURABLE_EXPECTED_OUTCOME="+string(outcome),
			)
			var output bytes.Buffer
			cmd.Stdout, cmd.Stderr = &output, &output
			if err := cmd.Run(); err != nil {
				t.Fatalf("fresh Host process failed: %v\n%s", err, output.String())
			}
			if !strings.Contains(output.String(), "TEI_DESKTOP_DURABLE_NO_REPLAY:"+string(outcome)) {
				t.Fatalf("fresh Host process did not confirm no replay: %s", output.String())
			}
		})
	}
}
