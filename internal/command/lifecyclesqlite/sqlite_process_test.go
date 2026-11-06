package lifecyclesqlite

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/command"
)

// TestSQLiteLifecycleHelperProcess is the child side of the multi-process test.
// It waits for the parent to close stdin (the barrier), runs one stage
// operation, prints OK or REFUSED and exits 0 either way.
func TestSQLiteLifecycleHelperProcess(t *testing.T) {
	if os.Getenv("TEI_SQLITE_HELPER") != "1" {
		t.Skip("helper process only")
	}
	store, err := Open(os.Getenv("TEI_SQLITE_DB"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	_, _ = io.Copy(io.Discard, os.Stdin)
	commandID := os.Getenv("TEI_SQLITE_COMMAND")
	owner := os.Getenv("TEI_SQLITE_OWNER")
	generation := 1
	_, _ = fmt.Sscanf(os.Getenv("TEI_SQLITE_GENERATION"), "%d", &generation)

	switch os.Getenv("TEI_SQLITE_OP") {
	case "dispatch":
		_, err = store.Dispatch(commandID, owner, generation)
	case "takeover":
		_, err = store.Takeover(commandID, owner, generation)
	default:
		err = fmt.Errorf("unknown operation")
	}
	if err != nil {
		fmt.Printf("REFUSED: %v\n", err)
		return
	}
	fmt.Println("OK")
}

type raceResult struct {
	ok     int
	output []string
}

// raceProcesses starts one child per env set, releases them together by closing
// stdin, and counts the successes.
func raceProcesses(t *testing.T, database string, racers []map[string]string) raceResult {
	t.Helper()
	commands := make([]*exec.Cmd, 0, len(racers))
	outputs := make([]*bytes.Buffer, 0, len(racers))
	writers := make([]io.WriteCloser, 0, len(racers))
	for _, env := range racers {
		command := exec.Command(os.Args[0], "-test.run=^TestSQLiteLifecycleHelperProcess$")
		environment := append(os.Environ(), "TEI_SQLITE_HELPER=1", "TEI_SQLITE_DB="+database)
		for key, value := range env {
			environment = append(environment, key+"="+value)
		}
		command.Env = environment
		writer, err := command.StdinPipe()
		if err != nil {
			t.Fatal(err)
		}
		buffer := &bytes.Buffer{}
		command.Stdout = buffer
		command.Stderr = buffer
		if err := command.Start(); err != nil {
			t.Fatal(err)
		}
		commands = append(commands, command)
		outputs = append(outputs, buffer)
		writers = append(writers, writer)
	}
	for _, writer := range writers {
		_ = writer.Close()
	}
	result := raceResult{}
	for i, command := range commands {
		if err := command.Wait(); err != nil {
			t.Fatalf("child %d failed: %v\n%s", i, err, outputs[i].String())
		}
		text := outputs[i].String()
		result.output = append(result.output, text)
		if strings.Contains(text, "OK") {
			result.ok++
		}
	}
	return result
}

func TestTwoProcessesCannotBothDispatch(t *testing.T) {
	database := filepath.Join(t.TempDir(), "lifecycle.db")
	result := raceProcesses(t, database, []map[string]string{
		{"TEI_SQLITE_OP": "dispatch", "TEI_SQLITE_COMMAND": "cmd-race", "TEI_SQLITE_OWNER": "owner-a", "TEI_SQLITE_GENERATION": "1"},
		{"TEI_SQLITE_OP": "dispatch", "TEI_SQLITE_COMMAND": "cmd-race", "TEI_SQLITE_OWNER": "owner-a", "TEI_SQLITE_GENERATION": "1"},
	})
	if result.ok != 1 {
		t.Fatalf("successful dispatches=%d, want exactly 1:\n%s", result.ok, strings.Join(result.output, "\n"))
	}
	if !strings.Contains(strings.Join(result.output, "\n"), "REFUSED") {
		t.Fatalf("the losing process did not report a refusal:\n%s", strings.Join(result.output, "\n"))
	}
	store, err := Open(database)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	attempt, ok := store.Attempt("cmd-race")
	if !ok || attempt.Stage != command.StageDispatched || attempt.Owner != "owner-a" || attempt.Generation != 1 {
		t.Fatalf("final attempt=%+v ok=%v", attempt, ok)
	}
}

func TestTwoProcessesCannotBothTakeOver(t *testing.T) {
	database := filepath.Join(t.TempDir(), "lifecycle.db")
	store, err := Open(database)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.Dispatch("cmd-race", "owner-a", 1); err != nil {
		t.Fatal(err)
	}
	if err := store.MarkUnknown("cmd-race", "owner-a", 1, "response lost"); err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}

	result := raceProcesses(t, database, []map[string]string{
		{"TEI_SQLITE_OP": "takeover", "TEI_SQLITE_COMMAND": "cmd-race", "TEI_SQLITE_OWNER": "owner-b", "TEI_SQLITE_GENERATION": "2"},
		{"TEI_SQLITE_OP": "takeover", "TEI_SQLITE_COMMAND": "cmd-race", "TEI_SQLITE_OWNER": "owner-c", "TEI_SQLITE_GENERATION": "2"},
	})
	if result.ok != 1 {
		t.Fatalf("successful takeovers=%d, want exactly 1:\n%s", result.ok, strings.Join(result.output, "\n"))
	}
	store, err = Open(database)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	attempt, ok := store.Attempt("cmd-race")
	if !ok || attempt.Stage != command.StageDispatched || attempt.TakeoverFrom != 1 ||
		(attempt.Owner != "owner-b" && attempt.Owner != "owner-c") || attempt.Generation != 2 {
		t.Fatalf("final attempt=%+v ok=%v", attempt, ok)
	}
	if _, err := store.Takeover("cmd-race", "owner-d", 2); err == nil {
		t.Fatal("a second takeover at the same generation was accepted")
	}
}
