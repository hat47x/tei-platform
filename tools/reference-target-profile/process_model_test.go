package targetprofile

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	generatorselection "tei/tools/reference-generator-selection"
)

func TestLongRunningEvidenceRequiresCompleteResponseStream(t *testing.T) {
	_, _, fixturePath, _ := selectionPaths(t)
	raw, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatal(err)
	}
	var fixture generatorselection.Fixture
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatal(err)
	}
	var valid strings.Builder
	for _, tc := range fixture.Cases {
		result, err := json.Marshal(tc.Want)
		if err != nil {
			t.Fatal(err)
		}
		valid.Write(result)
		valid.WriteByte('\n')
	}
	cases := []struct {
		name, output, reason string
		exit                 int
		responses            int
	}{
		{name: "complete", output: valid.String() + " \n\t", responses: len(fixture.Cases)},
		{name: "extra", output: valid.String() + `{}`, reason: "extra responses", responses: len(fixture.Cases) + 1},
		{name: "malformed-tail", output: valid.String() + strings.Repeat("!", 256*1024), reason: "reading response failed", responses: len(fixture.Cases)},
		{name: "truncated", output: valid.String()[:strings.LastIndex(strings.TrimSpace(valid.String()), "\n")+1], reason: "responses", responses: len(fixture.Cases) - 1},
		{name: "mismatch", output: strings.Replace(valid.String(), `"state":"present"`, `"state":"wrong"`, 1), reason: "fixture mismatch", responses: len(fixture.Cases)},
		{name: "nonzero-exit", output: valid.String(), exit: 7, reason: "candidate wait failed", responses: len(fixture.Cases)},
		{name: "no-response", reason: "responses"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			candidate := filepath.Join(root, "probe")
			if err := os.Mkdir(candidate, 0700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(candidate, "candidate.json"), []byte(`{"language":"go"}`), 0600); err != nil {
				t.Fatal(err)
			}
			// Consume all requests before writing, exercising concurrent stdin/stdout.
			source := fmt.Sprintf(`package main
import ("io"; "os"; "fmt")
func main() { io.Copy(io.Discard, os.Stdin); fmt.Print(%s); os.Exit(%d) }
`, strconv.Quote(tc.output), tc.exit)
			if err := os.WriteFile(filepath.Join(candidate, "main.go"), []byte(source), 0600); err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			binary := filepath.Join(t.TempDir(), "probe")
			if runtime.GOOS == "windows" {
				binary += ".exe"
			}
			if err := buildGoCandidate(ctx, candidate, binary); err != nil {
				t.Fatal(err)
			}
			evidence := runLongRunningProbe(ctx, RunOptions{}, fixture, binary, nil, &ProcessModelEvidence{Mode: ProcessLongRunning})
			m := evidence.Measurement
			if m == nil || m.ElapsedNanos <= 0 || m.UserCPUNanos < 0 || m.SystemCPUNanos < 0 || m.GOOS != runtime.GOOS {
				t.Fatalf("missing process measurement: %+v", m)
			}
			if tc.responses == 0 {
				if evidence.FirstResponseNanos != nil {
					t.Fatal("invented first response latency without a response")
				}
			} else if evidence.FirstResponseNanos == nil || *evidence.FirstResponseNanos <= 0 || *evidence.FirstResponseNanos > m.ElapsedNanos {
				t.Fatalf("first response latency is outside execution: %+v", evidence)
			}
			if evidence.Passed != (tc.reason == "") || !strings.Contains(evidence.Reason, tc.reason) {
				t.Fatalf("unexpected pass/reason: %+v", evidence)
			}
			if evidence.Processes != 1 || evidence.Requests != len(fixture.Cases) || evidence.Responses != tc.responses || evidence.ExitCode != tc.exit || evidence.StderrDigest == "" {
				t.Fatalf("incomplete execution evidence: %+v", evidence)
			}
		})
	}
}

func TestLongRunningStartFailureHasNoMeasurements(t *testing.T) {
	evidence := runLongRunningProbe(context.Background(), RunOptions{}, generatorselection.Fixture{}, filepath.Join(t.TempDir(), "missing"), nil, &ProcessModelEvidence{Mode: ProcessLongRunning})
	if evidence.Passed || evidence.Processes != 0 || evidence.Measurement != nil || evidence.FirstResponseNanos != nil {
		t.Fatalf("start failure invented measurements: %+v", evidence)
	}
}
