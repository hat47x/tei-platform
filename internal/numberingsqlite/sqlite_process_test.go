package numberingsqlite

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/numbering"
)

// TestNumberingHelperProcess is the child side of the multi-process test. It waits
// for the parent to close stdin (the barrier), reserves one number from the shared
// counter, and prints OK or REFUSED. It exits 0 either way, so the parent counts
// the outcomes rather than the exit codes.
func TestNumberingHelperProcess(t *testing.T) {
	if os.Getenv("TEI_NUMBERING_HELPER") != "1" {
		t.Skip("helper process only")
	}
	store, err := Open(os.Getenv("TEI_NUMBERING_DB"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	_, _ = io.Copy(io.Discard, os.Stdin)
	contract := numbering.Contract{Digits: 6}
	reservation, err := contract.ReserveInTransaction(context.Background(), store, helperUnitOfWork{}, os.Getenv("TEI_NUMBERING_RULE"))
	if err != nil {
		fmt.Printf("REFUSED: %v\n", err)
		return
	}
	if err := reservation.Commit(context.Background()); err != nil {
		fmt.Printf("REFUSED: %v\n", err)
		return
	}
	fmt.Printf("OK: %s\n", reservation.Number())
}

// helperUnitOfWork is a transaction that does nothing, so the reservation's own
// store write is what the race observes.
type helperUnitOfWork struct{}

func (helperUnitOfWork) Begin(context.Context) (numbering.Transaction, error) {
	return helperTransaction{}, nil
}

type helperTransaction struct{}

func (helperTransaction) Commit(context.Context) error   { return nil }
func (helperTransaction) Rollback(context.Context) error { return nil }

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
		command := exec.Command(os.Args[0], "-test.run=^TestNumberingHelperProcess$")
		environment := append(os.Environ(), "TEI_NUMBERING_HELPER=1", "TEI_NUMBERING_DB="+database)
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
		if strings.Contains(text, "OK:") {
			result.ok++
		}
	}
	return result
}

// TestTwoProcessesNeverIssueTheSameNumber checks the invariant that matters: no
// two processes get the same number, and the counter ends at the number of
// successes.
//
// Two processes can both succeed when their load and save do not overlap: SQLite
// serializes the immediate write transaction, so the second reads the first's
// committed counter. A refusal appears when the versions do overlap. Both outcomes
// are correct; a shared number is not.
//
// X-TEA wrote the counter with an optimistic update whose result it never inspected
// (D:Session.java L1995–2002), so two concurrent callers could both believe they
// issued the next number and one issuance vanished.
func TestTwoProcessesNeverIssueTheSameNumber(t *testing.T) {
	database := filepath.Join(t.TempDir(), "counter.db")
	result := raceProcesses(t, database, []map[string]string{
		{"TEI_NUMBERING_RULE": "order"},
		{"TEI_NUMBERING_RULE": "order"},
	})
	issued := map[string]bool{}
	for _, output := range result.output {
		for _, line := range strings.Split(output, "\n") {
			if !strings.HasPrefix(line, "OK: ") {
				continue
			}
			number := strings.TrimSpace(strings.TrimPrefix(line, "OK: "))
			if issued[number] {
				t.Fatalf("number %s was issued twice:\n%s", number, strings.Join(result.output, "\n"))
			}
			issued[number] = true
		}
	}
	if len(issued) != result.ok || result.ok == 0 {
		t.Fatalf("issued=%v ok=%d:\n%s", issued, result.ok, strings.Join(result.output, "\n"))
	}
	store, err := Open(database)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	state, version, err := store.Load(context.Background(), "order")
	if err != nil {
		t.Fatal(err)
	}
	// The counter equals the number of successes, and the version moved once per
	// successful write. A lost update would leave the counter below the issuances.
	if state.Current != result.ok || version != result.ok {
		t.Fatalf("state=%+v version=%d ok=%d", state, version, result.ok)
	}
}

// TestTwoProcessesIssueDistinctNumbers checks that the two processes end up with the
// numbers 1 and 2 when they run one after the other, so a refusal does not lose an
// issuance in the counter.
func TestTwoProcessesIssueDistinctNumbers(t *testing.T) {
	database := filepath.Join(t.TempDir(), "counter.db")
	contract := numbering.Contract{Digits: 6}
	ctx := context.Background()

	store, err := Open(database)
	if err != nil {
		t.Fatal(err)
	}
	reservation, err := contract.ReserveInTransaction(ctx, store, helperUnitOfWork{}, "order")
	if err != nil {
		t.Fatal(err)
	}
	if err := reservation.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}

	result := raceProcesses(t, database, []map[string]string{{"TEI_NUMBERING_RULE": "order"}})
	if result.ok != 1 {
		t.Fatalf("successful issuance=%d:\n%s", result.ok, strings.Join(result.output, "\n"))
	}
	if reservation.Number() != "000001" {
		t.Fatalf("first=%q", reservation.Number())
	}
	if !strings.Contains(result.output[0], "000002") {
		t.Fatalf("the second process did not take the next number:\n%s", result.output[0])
	}
}
