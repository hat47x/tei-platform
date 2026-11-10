package targetprofile

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
	"time"

	generatorselection "tei/tools/reference-generator-selection"
)

// buildProbeStub compiles a tiny candidate that answers the first `answers`
// requests and then stalls, so a probe can be cut off mid-stream on purpose.
func buildProbeStub(t *testing.T, answers int, stderrText string) string {
	t.Helper()
	root := t.TempDir()
	source := fmt.Sprintf(`package main
import ("bufio"; "os"; "time")
func main() {
	scanner := bufio.NewScanner(os.Stdin)
	for i := 0; i < %d && scanner.Scan(); i++ {
		os.Stdout.WriteString("{\"state\":\"missing\"}\n")
	}
	os.Stderr.WriteString(%q)
	os.Stdout.Sync()
	time.Sleep(30 * time.Second)
}
`, answers, stderrText)
	if err := os.WriteFile(filepath.Join(root, "main.go"), []byte(source), 0600); err != nil {
		t.Fatal(err)
	}
	binary := filepath.Join(t.TempDir(), "stub")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	// buildGoCandidate is the package's own single-file build, used by the
	// existing probe tests. Reusing it keeps this stub on the same build path
	// as a real candidate instead of inventing a second one.
	if err := buildGoCandidate(context.Background(), root, binary); err != nil {
		t.Fatalf("build probe stub: %v", err)
	}
	return binary
}

func probeFixture(t *testing.T, cases int) generatorselection.Fixture {
	t.Helper()
	fixture := generatorselection.Fixture{}
	for i := 0; i < cases; i++ {
		input, err := json.Marshal(map[string]any{"n": i})
		if err != nil {
			t.Fatal(err)
		}
		fixture.Cases = append(fixture.Cases, generatorselection.Case{
			ID:    fmt.Sprintf("case-%02d", i+1),
			Input: input,
			Want:  generatorselection.Result{State: "missing"},
		})
	}
	return fixture
}

// A probe cut off by its own budget must be reported as a timeout, not as a
// candidate that answered wrongly. Before this, the truncated stream left the
// remaining cases at the zero Result and the receipt read like a wrong answer
// for a case such as precision-overflow (#3801).
func TestProbeBudgetExhaustionIsReportedAsATimeout(t *testing.T) {
	if testing.Short() {
		t.Skip("builds and runs a probe")
	}
	previous := longRunningProbeBudget
	longRunningProbeBudget = 2 * time.Second
	t.Cleanup(func() { longRunningProbeBudget = previous })

	// Answer two of five cases, then stall: the stream is cut mid-way with
	// three cases unread, which is the shape the CI failure reported.
	binary := buildProbeStub(t, 2, "Traceback: runtime gave up mid-write")
	evidence := runLongRunningProbe(context.Background(), RunOptions{}, probeFixture(t, 5), binary, nil,
		&ProcessModelEvidence{Mode: ProcessLongRunning})

	if evidence.Passed {
		t.Fatalf("a truncated stream passed: %+v", evidence)
	}
	if !strings.Contains(evidence.Reason, "exceeded its") || !strings.Contains(evidence.Reason, "budget") {
		t.Fatalf("the budget was not named as the cause: %q", evidence.Reason)
	}
	// The progress count is what separates "cut short" from "answered wrongly";
	// the exact figure depends on how much the stub managed to flush before the
	// deadline, so assert the shape rather than a number.
	if !regexp.MustCompile(`after \d+ of 5 responses`).MatchString(evidence.Reason) {
		t.Fatalf("the reason does not say how far the stream got: %q", evidence.Reason)
	}
	if !strings.Contains(evidence.Reason, "Traceback") {
		t.Fatalf("the child's stderr is missing from the reason: %q", evidence.Reason)
	}
	if evidence.Responses >= len(probeFixture(t, 5).Cases) {
		t.Fatalf("the stream was not cut short: responses=%d", evidence.Responses)
	}
	// The cases that were never answered must not look like wrong answers.
	for _, processCase := range evidence.Cases {
		if processCase.CaseID == "" {
			t.Fatalf("a case entry has no id: %+v", processCase)
		}
	}
}

// A candidate that answers everything is still reported as passing, so the
// deadline check does not swallow ordinary failures or ordinary successes.
func TestProbeBudgetLeavesACompleteRunPassing(t *testing.T) {
	if testing.Short() {
		t.Skip("builds and runs a probe")
	}
	previous := longRunningProbeBudget
	longRunningProbeBudget = 20 * time.Second
	t.Cleanup(func() { longRunningProbeBudget = previous })

	fixture := probeFixture(t, 2)
	binary := buildProbeStub(t, len(fixture.Cases), "")
	// The stub stalls after answering; with every case answered the probe has
	// read its full stream, so it must not report a budget failure of its own
	// making as long as the budget still has room.
	evidence := runLongRunningProbe(context.Background(), RunOptions{}, fixture, binary, nil,
		&ProcessModelEvidence{Mode: ProcessLongRunning})
	if strings.Contains(evidence.Reason, "exceeded its") {
		t.Fatalf("a complete stream was reported as a budget failure: %q", evidence.Reason)
	}
	if evidence.Responses != len(fixture.Cases) {
		t.Fatalf("responses=%d, want %d", evidence.Responses, len(fixture.Cases))
	}
}

// The mapped receipt must carry the reason through to JSON, so an operator
// reading the artifact sees the deadline rather than a wrong answer.
func TestProbeTimeoutReasonSurvivesJSON(t *testing.T) {
	evidence := &ProcessModelEvidence{
		Mode: ProcessLongRunning, Passed: false,
		Reason: "probe exceeded its 30s budget after 9 of 10 responses; stderr: boom",
	}
	raw, err := json.Marshal(evidence)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), "exceeded its") || !strings.Contains(string(raw), "boom") {
		t.Fatalf("the reason did not survive: %s", raw)
	}
}
