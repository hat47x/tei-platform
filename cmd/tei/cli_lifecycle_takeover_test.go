package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
)

// buildCLIForTakeover builds the CLI once for the takeover cases.
func buildCLIForTakeover(t *testing.T) string {
	t.Helper()
	binary := filepath.Join(t.TempDir(), "tei")
	if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	build := exec.Command("go", "build", "-o", binary, ".")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build CLI: %v\n%s", err, out)
	}
	return binary
}

// runTakeoverCLI runs one invocation and returns the decoded envelope.
func runTakeoverCLI(t *testing.T, binary string, args ...string) (map[string]any, int) {
	t.Helper()
	command := exec.Command(binary, args...)
	var stdout, stderr bytes.Buffer
	command.Stdout, command.Stderr = &stdout, &stderr
	code := 0
	if err := command.Run(); err != nil {
		var exit *exec.ExitError
		if !errors.As(err, &exit) {
			t.Fatalf("CLI execution: %v", err)
		}
		code = exit.ExitCode()
	}
	var doc map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &doc); err != nil {
		t.Fatalf("not one JSON document: %v (%s)", err, stdout.String())
	}
	return doc, code
}

// takeoverArgs returns the arguments for one lifecycle-tracked reference run.
func takeoverArgs(state string, owner string, generation int, takeover bool) []string {
	args := []string{"view",
		"-workspace", filepath.Join("..", "..", "reference", "workspace"),
		"-application", "order_entry",
		"-pattern", "order_entry.submit_order",
		"-command", "sales.order.submit",
		"-input", "order=ORD-1",
		"-execute", "-format", "json",
		"-lifecycle-state", state,
		"-lifecycle-command", "cmd-takeover",
		"-lifecycle-owner", owner,
		"-lifecycle-generation", strconv.Itoa(generation),
	}
	if takeover {
		args = append(args, "-lifecycle-takeover")
	}
	return args
}

// TestLifecycleTakeoverRecoversAnUnknownOutcome checks the operator path AC-5
// left open: a command whose outcome is unknown can be picked up by a new owner
// at a larger generation, instead of staying stuck.
//
// The state file is written directly, because the CLI has no way to produce an
// unknown outcome by itself: a reference run either observes an outcome or
// fails. That is what the stage store is for, and it is the same store the CLI
// opens.
func TestLifecycleTakeoverRecoversAnUnknownOutcome(t *testing.T) {
	binary := buildCLIForTakeover(t)
	state := filepath.Join(t.TempDir(), "lifecycle.db")
	prepareUnknownAttempt(t, state, "cmd-takeover")

	doc, code := runTakeoverCLI(t, binary, takeoverArgs(state, "owner-b", 2, true)...)
	if code != 0 {
		t.Fatalf("takeover refused the recovery: exit=%d doc=%+v", code, doc)
	}
	if ok, _ := doc["ok"].(bool); !ok {
		t.Fatalf("takeover did not run the command: %+v", doc)
	}

	// The attempt is now observed, so a second takeover must refuse: recovery is
	// not a way to re-run a finished command.
	again, secondCode := runTakeoverCLI(t, binary, takeoverArgs(state, "owner-c", 3, true)...)
	if secondCode == 0 {
		t.Fatalf("a second takeover ran the command again: %+v", again)
	}
	errorInfo, _ := again["error"].(map[string]any)
	if errorInfo == nil || errorInfo["code"] != "COMMAND_LIFECYCLE_REFUSED" {
		t.Fatalf("second takeover error=%+v", again)
	}
}

// TestLifecycleTakeoverRefusesAClaimedCommand checks that the same flag cannot
// be used to re-run a command that is not unknown. Without this, the flag would
// be a way around the observed-once rule.
func TestLifecycleTakeoverRefusesAClaimedCommand(t *testing.T) {
	binary := buildCLIForTakeover(t)
	state := filepath.Join(t.TempDir(), "lifecycle.db")

	first, code := runTakeoverCLI(t, binary, takeoverArgs(state, "owner-a", 1, false)...)
	if code != 0 {
		t.Fatalf("first run: exit=%d doc=%+v", code, first)
	}

	doc, secondCode := runTakeoverCLI(t, binary, takeoverArgs(state, "owner-b", 2, true)...)
	if secondCode == 0 {
		t.Fatalf("takeover re-ran an observed command: %+v", doc)
	}
	errorInfo, _ := doc["error"].(map[string]any)
	if errorInfo == nil || errorInfo["code"] != "COMMAND_LIFECYCLE_REFUSED" {
		t.Fatalf("error=%+v", doc)
	}
	message, _ := errorInfo["message"].(string)
	if !strings.Contains(message, "takeover requires an unknown outcome") {
		t.Fatalf("unexpected refusal: %s", message)
	}
}

// prepareUnknownAttempt writes a dispatched-then-unknown attempt into the stage
// store the CLI opens, so the test starts from the state a lost response leaves.
func prepareUnknownAttempt(t *testing.T, state string, commandID string) {
	t.Helper()
	// The helper imports an internal package, so it has to live inside this
	// module's tree; a temp directory outside it is refused by the toolchain.
	source := filepath.Join("testdata", "takeover-prepare")
	if err := os.MkdirAll(source, 0o700); err != nil {
		t.Fatal(err)
	}
	program := `package main

import (
	"fmt"
	"os"

	"tei/internal/command/lifecyclesqlite"
)

func main() {
	store, err := lifecyclesqlite.Open(os.Args[1])
	if err != nil {
		panic(err)
	}
	defer store.Close()
	if _, err := store.Dispatch(os.Args[2], "owner-a", 1); err != nil {
		panic(err)
	}
	if err := store.MarkUnknown(os.Args[2], "owner-a", 1, "response lost"); err != nil {
		panic(err)
	}
	fmt.Println("prepared")
}
`
	path := filepath.Join(source, "main.go")
	if err := os.WriteFile(path, []byte(program), 0o600); err != nil {
		t.Fatal(err)
	}
	run := exec.Command("go", "run", path, state, commandID)
	if out, err := run.CombinedOutput(); err != nil {
		t.Fatalf("prepare unknown attempt: %v\n%s", err, out)
	}
}
