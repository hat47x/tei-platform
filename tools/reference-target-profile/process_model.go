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
	CaseID string                    `json:"caseId"`
	Passed bool                      `json:"passed"`
	Got    generatorselection.Result `json:"got"`
	Want   generatorselection.Result `json:"want"`
}

// ProcessModelEvidence records whether the harness executed the selected
// candidate according to the profile's declared process model. It is not a
// measurement of a real target's process supervisor.
type ProcessModelEvidence struct {
	Mode         string                              `json:"mode"`
	Runtime      *generatorselection.RuntimeEvidence `json:"runtime,omitempty"`
	Processes    int                                 `json:"processes"`
	Requests     int                                 `json:"requests"`
	Responses    int                                 `json:"responses"`
	Passed       bool                                `json:"passed"`
	ExitCode     int                                 `json:"exitCode,omitempty"`
	StderrDigest string                              `json:"stderrDigest,omitempty"`
	Reason       string                              `json:"reason,omitempty"`
	Cases        []ProcessCase                       `json:"cases,omitempty"`
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
		return longRunningEvidence(ctx, profile, opts)
	case ProcessBatch:
		return &ProcessModelEvidence{
			Mode: ProcessBatch, Passed: false,
			Reason: "batch process model is not implemented by this reference",
		}
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
		if !result.Passed {
			evidence.Passed = false
		}
		evidence.Cases = append(evidence.Cases, processCase)
	}
	return evidence
}

func longRunningEvidence(ctx context.Context, profile Profile, opts RunOptions) *ProcessModelEvidence {
	evidence := &ProcessModelEvidence{Mode: ProcessLongRunning}
	if strings.TrimSpace(profile.GeneratorID) == "" {
		evidence.Reason = "long-running profile has no generatorId"
		return evidence
	}
	candidateDir := filepath.Join(opts.CandidatesDir, profile.GeneratorID)
	var manifest generatorselection.CandidateManifest
	manifestRaw, err := os.ReadFile(filepath.Join(candidateDir, "candidate.json"))
	if err != nil {
		evidence.Reason = "candidate manifest is unreadable: " + err.Error()
		return evidence
	}
	if err := json.Unmarshal(manifestRaw, &manifest); err != nil {
		evidence.Reason = "candidate manifest is invalid: " + err.Error()
		return evidence
	}
	language := manifest.Language
	if language == "" {
		language = generatorselection.LanguageGo
	}
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
	binaryPath := filepath.Join(tempDir, "candidate")
	if runtime.GOOS == "windows" {
		binaryPath += ".exe"
	}
	executable := binaryPath
	var args []string
	switch language {
	case generatorselection.LanguageGo:
		if err := buildGoCandidate(ctx, candidateDir, binaryPath); err != nil {
			evidence.Reason = "candidate build failed: " + err.Error()
			return evidence
		}
	case generatorselection.LanguagePython, generatorselection.LanguageJavaScript:
		executable, evidence.Runtime = generatorselection.ResolveScriptRuntime(language)
		if !evidence.Runtime.Available {
			evidence.Reason = "runtime unavailable: " + language + " interpreter was not found"
			return evidence
		}
		sourcePath, err := filepath.Abs(filepath.Join(candidateDir, filepath.FromSlash(manifest.Source)))
		if err != nil {
			evidence.Reason = err.Error()
			return evidence
		}
		args = []string{sourcePath}
	default:
		evidence.Reason = "unsupported long-running candidate language: " + language
		return evidence
	}
	runCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(runCtx, executable, args...)
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
		if evidence.Responses > len(fixture.Cases) {
			evidence.Reason = "candidate returned extra responses"
			continue
		}
		tc := fixture.Cases[evidence.Responses-1]
		processCase := ProcessCase{CaseID: tc.ID, Got: got, Want: tc.Want, Passed: reflect.DeepEqual(got, tc.Want)}
		evidence.Cases = append(evidence.Cases, processCase)
	}
	waitErr := cmd.Wait()
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

func buildGoCandidate(ctx context.Context, dir, binaryPath string) error {
	build := exec.CommandContext(ctx, "go", "build", "-o", binaryPath, ".")
	build.Dir = dir
	out, err := build.CombinedOutput()
	if err == nil {
		return nil
	}
	if strings.Contains(string(out), "go.mod file not found") {
		fallback := exec.CommandContext(ctx, "go", "build", "-o", binaryPath, "main.go")
		fallback.Dir = dir
		fallback.Env = append(os.Environ(), "GO111MODULE=off")
		if fallbackOut, fallbackErr := fallback.CombinedOutput(); fallbackErr == nil {
			return nil
		} else {
			return fmt.Errorf("%v: %s", fallbackErr, fallbackOut)
		}
	}
	return fmt.Errorf("%v: %s", err, out)
}

func digestBytes(content []byte) string {
	sum := sha256.Sum256(content)
	return "sha256:" + hex.EncodeToString(sum[:])
}
