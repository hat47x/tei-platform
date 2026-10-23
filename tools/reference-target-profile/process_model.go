package targetprofile

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"time"

	generatorselection "tei/tools/reference-generator-selection"
)

type ProcessCase struct {
	Measurement *generatorselection.ProcessMeasurement `json:"measurement,omitempty"`
	CaseID      string                                 `json:"caseId"`
	Passed      bool                                   `json:"passed"`
	Got         generatorselection.Result              `json:"got"`
	Want        generatorselection.Result              `json:"want"`
}

// ProcessModelEvidence records whether the harness executed the selected
// candidate according to the profile's declared process model. It is not a
// measurement of a real target's process supervisor.
type ProcessModelEvidence struct {
	Measurement        *generatorselection.ProcessMeasurement `json:"measurement,omitempty"`
	FirstResponseNanos *int64                                 `json:"firstResponseNanos,omitempty"`
	CandidateID        string                                 `json:"candidateId,omitempty"`
	SourceDigest       string                                 `json:"sourceDigest,omitempty"`
	BinaryDigest       string                                 `json:"binaryDigest,omitempty"`
	Binding            string                                 `json:"binding,omitempty"`
	Mode               string                                 `json:"mode"`
	Runtime            *generatorselection.RuntimeEvidence    `json:"runtime,omitempty"`
	Processes          int                                    `json:"processes"`
	Requests           int                                    `json:"requests"`
	Responses          int                                    `json:"responses"`
	Passed             bool                                   `json:"passed"`
	ExitCode           int                                    `json:"exitCode,omitempty"`
	StderrDigest       string                                 `json:"stderrDigest,omitempty"`
	Reason             string                                 `json:"reason,omitempty"`
	Cases              []ProcessCase                          `json:"cases,omitempty"`
}

func enforceProcessModel(ctx context.Context, profile Profile, opts RunOptions, selection generatorselection.Receipt) *ProcessModelEvidence {
	switch profile.ProcessModel {
	case ProcessOneShot:
		return oneshotEvidence(profile, selection)
	case ProcessLongRunning:
		candidate := selectedCandidate(Receipt{Profile: profile, Selection: selection})
		if candidate == nil || !candidate.Passed {
			return &ProcessModelEvidence{Mode: ProcessLongRunning, Reason: "selected candidate did not pass selection"}
		}
		return longRunningEvidence(ctx, opts, *candidate, ProcessLongRunning)
	case ProcessBatch:
		candidate := selectedCandidate(Receipt{Profile: profile, Selection: selection})
		if candidate == nil || !candidate.Passed {
			return &ProcessModelEvidence{Mode: ProcessBatch, Reason: "selected candidate did not pass selection"}
		}
		return longRunningEvidence(ctx, opts, *candidate, ProcessBatch)
	default:
		return &ProcessModelEvidence{Mode: profile.ProcessModel, Passed: false, Reason: "unknown process model"}
	}
}

func oneshotEvidence(profile Profile, selection generatorselection.Receipt) *ProcessModelEvidence {
	evidence := &ProcessModelEvidence{Mode: ProcessOneShot}
	candidate := selectedCandidate(Receipt{Profile: profile, Selection: selection})
	if candidate == nil || len(candidate.Results) == 0 {
		evidence.Reason = "no selected candidate execution evidence"
		return evidence
	}
	evidence.Processes = len(candidate.Results)
	evidence.Requests = len(candidate.Results)
	evidence.Responses = len(candidate.Results)
	evidence.Passed = true
	for _, result := range candidate.Results {
		processCase := ProcessCase{CaseID: result.CaseID, Passed: result.Passed, Got: result.Got, Want: result.Want}
		if result.Execution != nil {
			processCase.Measurement = result.Execution.Measurement
		}
		if !result.Passed {
			evidence.Passed = false
		}
		evidence.Cases = append(evidence.Cases, processCase)
	}
	return evidence
}

func longRunningEvidence(ctx context.Context, opts RunOptions, selected generatorselection.CandidateReceipt, mode string) *ProcessModelEvidence {
	evidence := &ProcessModelEvidence{Mode: mode, CandidateID: selected.ID}
	fixtureRaw, err := os.ReadFile(opts.FixturePath)
	if err != nil {
		evidence.Reason = "fixture is unreadable: " + err.Error()
		return evidence
	}
	var fixture generatorselection.Fixture
	if err := json.Unmarshal(fixtureRaw, &fixture); err != nil {
		evidence.Reason = "fixture is invalid: " + err.Error()
		return evidence
	}
	if len(fixture.Cases) == 0 {
		evidence.Reason = "fixture has no cases"
		return evidence
	}
	tempDir, err := os.MkdirTemp("", "tei-target-profile-long-running-*")
	if err != nil {
		evidence.Reason = err.Error()
		return evidence
	}
	defer os.RemoveAll(tempDir)
	manifest, err := snapshotProbeSource(opts.CandidatesDir, tempDir, selected)
	if err != nil {
		evidence.Reason = "candidate binding failed: " + err.Error()
		return evidence
	}
	evidence.SourceDigest = manifest.SourceDigest
	evidence.Binding = "source-digest"
	language := selected.Language
	binaryPath := filepath.Join(tempDir, "candidate")
	if runtime.GOOS == "windows" {
		binaryPath += ".exe"
	}
	executable := binaryPath
	var args []string
	switch language {
	case generatorselection.LanguageGo:
		if err := buildGoCandidate(ctx, tempDir, binaryPath); err != nil {
			evidence.Reason = "candidate build failed: " + err.Error()
			return evidence
		}
		binary, err := os.ReadFile(binaryPath)
		if err != nil {
			evidence.Reason = "probe executable is unreadable: " + err.Error()
			return evidence
		}
		evidence.BinaryDigest = digestBytes(binary)
		if selected.BinaryDigest == "" || evidence.BinaryDigest != selected.BinaryDigest {
			evidence.Reason = "probe binary digest differs from selection"
			return evidence
		}
		evidence.Binding = "source-and-binary-digest"
	case generatorselection.LanguagePython, generatorselection.LanguageJavaScript:
		executable, evidence.Runtime = generatorselection.ResolveScriptRuntime(language)
		if !evidence.Runtime.Available {
			evidence.Reason = "runtime unavailable: " + language + " interpreter was not found"
			return evidence
		}
		sourcePath, err := filepath.Abs(filepath.Join(tempDir, manifest.Source))
		if err != nil {
			evidence.Reason = err.Error()
			return evidence
		}
		args = []string{sourcePath}
	case generatorselection.LanguageJava:
		javac, java, version, available := findJavaForProbe()
		if !available {
			evidence.Reason = "runtime unavailable: javac/java was not found"
			return evidence
		}
		classesDir := filepath.Join(tempDir, "classes")
		if err := os.MkdirAll(classesDir, 0o755); err != nil {
			evidence.Reason = err.Error()
			return evidence
		}
		compile := exec.CommandContext(ctx, javac, "-d", classesDir, manifest.Source)
		compile.Dir = tempDir
		if out, err := compile.CombinedOutput(); err != nil {
			evidence.Reason = "java compile failed: " + err.Error() + ": " + string(out)
			return evidence
		}
		classBytes, err := os.ReadFile(filepath.Join(classesDir, "Main.class"))
		if err != nil {
			evidence.Reason = "compiled class is unreadable: " + err.Error()
			return evidence
		}
		evidence.BinaryDigest = digestBytes(classBytes)
		if selected.BinaryDigest == "" || evidence.BinaryDigest != selected.BinaryDigest {
			evidence.Reason = "probe class digest differs from selection"
			return evidence
		}
		evidence.Binding = "source-and-binary-digest"
		evidence.Runtime = &generatorselection.RuntimeEvidence{
			Language: generatorselection.LanguageJava, Command: java + " -cp <classes> Main",
			Version: version, Available: true,
		}
		executable = java
		args = []string{"-cp", classesDir, "Main"}
	case generatorselection.LanguageWasm:
		node, wasmExec, version, available := findWasmForProbe()
		if !available {
			evidence.Reason = "runtime unavailable: node or Go wasm_exec_node.js was not found"
			return evidence
		}
		wasmPath := filepath.Join(tempDir, "candidate.wasm")
		build := exec.CommandContext(ctx, "go", "build", "-trimpath", "-buildvcs=false", "-o", wasmPath, "main.go")
		build.Dir = tempDir
		build.Env = append(os.Environ(), "GOOS=js", "GOARCH=wasm")
		if out, err := build.CombinedOutput(); err != nil {
			evidence.Reason = "wasm build failed: " + err.Error() + ": " + string(out)
			return evidence
		}
		wasmBytes, err := os.ReadFile(wasmPath)
		if err != nil {
			evidence.Reason = "wasm binary is unreadable: " + err.Error()
			return evidence
		}
		evidence.BinaryDigest = digestBytes(wasmBytes)
		if selected.BinaryDigest == "" || evidence.BinaryDigest != selected.BinaryDigest {
			evidence.Reason = "probe wasm digest differs from selection"
			return evidence
		}
		evidence.Binding = "source-and-binary-digest"
		evidence.Runtime = &generatorselection.RuntimeEvidence{
			Language: generatorselection.LanguageWasm, Command: node + " wasm_exec_node.js <wasm>",
			Version: version, Available: true,
		}
		executable = node
		args = []string{wasmExec, wasmPath}
	default:
		evidence.Reason = "unsupported long-running candidate language: " + language
		return evidence
	}
	return runLongRunningProbe(ctx, opts, fixture, executable, args, evidence)
}

// runLongRunningProbe handles the stream after the executable/source binding
// has been checked. Protocol regression tests use synthetic executables here.
func runLongRunningProbe(ctx context.Context, opts RunOptions, fixture generatorselection.Fixture, executable string, args []string, evidence *ProcessModelEvidence) *ProcessModelEvidence {
	runCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	cmd := opts.execution.Command(runCtx, executable, args...)
	var input bytes.Buffer
	encoder := json.NewEncoder(&input)
	for _, tc := range fixture.Cases {
		if err := encoder.Encode(tc.Input); err != nil {
			evidence.Reason = "encoding request failed: " + err.Error()
			return evidence
		}
	}
	// os/exec writes input concurrently with stdout reads, avoiding pipe deadlock.
	cmd.Stdin = &input
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		evidence.Reason = err.Error()
		return evidence
	}
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	started := time.Now()
	if err := cmd.Start(); err != nil {
		evidence.Reason = "candidate start failed: " + err.Error()
		return evidence
	}
	evidence.Processes = 1
	evidence.Requests = len(fixture.Cases)
	decoder := json.NewDecoder(stdout)
	for {
		var got generatorselection.Result
		if err := decoder.Decode(&got); err != nil {
			if err != io.EOF {
				evidence.Reason = "reading response failed: " + err.Error()
				// Drain malformed output so the child can exit before Wait.
				_, _ = io.Copy(io.Discard, stdout)
			}
			break
		}
		evidence.Responses++
		if evidence.FirstResponseNanos == nil {
			firstResponse := time.Since(started).Nanoseconds()
			evidence.FirstResponseNanos = &firstResponse
		}
		if evidence.Responses > len(fixture.Cases) {
			evidence.Reason = "candidate returned extra responses"
			continue
		}
		tc := fixture.Cases[evidence.Responses-1]
		processCase := ProcessCase{CaseID: tc.ID, Got: got, Want: tc.Want, Passed: reflect.DeepEqual(got, tc.Want)}
		evidence.Cases = append(evidence.Cases, processCase)
	}
	waitErr := cmd.Wait()
	evidence.Measurement = generatorselection.MeasureProcess(cmd.ProcessState, time.Since(started))
	if cmd.ProcessState != nil {
		evidence.ExitCode = cmd.ProcessState.ExitCode()
	}
	evidence.StderrDigest = digestBytes(stderr.Bytes())
	if waitErr != nil && evidence.Reason == "" {
		evidence.Reason = "candidate wait failed: " + waitErr.Error()
	}
	evidence.Passed = evidence.Responses == evidence.Requests && evidence.Reason == ""
	for _, processCase := range evidence.Cases {
		if !processCase.Passed {
			evidence.Passed = false
			if evidence.Reason == "" {
				evidence.Reason = "fixture mismatch: " + processCase.CaseID
			}
		}
	}
	if evidence.Responses != evidence.Requests && evidence.Reason == "" {
		evidence.Reason = fmt.Sprintf("candidate returned %d of %d responses", evidence.Responses, evidence.Requests)
	}
	return evidence
}

func findJavaForProbe() (javacPath, javaPath, version string, available bool) {
	javac, err := exec.LookPath("javac")
	if err != nil {
		return "", "", "", false
	}
	java, err := exec.LookPath("java")
	if err != nil {
		return "", "", "", false
	}
	out, err := exec.Command(javac, "-version").CombinedOutput()
	if err != nil {
		return "", "", "", false
	}
	return javac, java, strings.TrimSpace(string(out)), true
}

func findWasmForProbe() (nodePath, wasmExecPath, version string, available bool) {
	node, err := exec.LookPath("node")
	if err != nil {
		return "", "", "", false
	}
	out, err := exec.Command(node, "--version").CombinedOutput()
	if err != nil {
		return "", "", "", false
	}
	gorootOut, err := exec.Command("go", "env", "GOROOT").Output()
	if err != nil {
		return "", "", "", false
	}
	wasmExec := filepath.Join(strings.TrimSpace(string(gorootOut)), "lib", "wasm", "wasm_exec_node.js")
	if _, err := os.Stat(wasmExec); err != nil {
		return "", "", "", false
	}
	return node, wasmExec, strings.TrimSpace(string(out)), true
}

func buildGoCandidate(ctx context.Context, dir, binaryPath string) error {
	// Match the selection reference's single-file, path-independent build.
	build := exec.CommandContext(ctx, "go", "build", "-trimpath", "-buildvcs=false", "-o", binaryPath, "main.go")
	build.Dir = dir
	out, err := build.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%v: %s", err, out)
	}
	return nil
}

func digestBytes(content []byte) string {
	sum := sha256.Sum256(content)
	return "sha256:" + hex.EncodeToString(sum[:])
}
