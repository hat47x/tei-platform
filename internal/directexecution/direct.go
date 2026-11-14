package directexecution

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// Run executes a local command and records only observations made here.
// A previously existing output is refused. This is not an OS-level fence:
// another process may still change the path while the command runs.
func Run(opts Options) (Evidence, error) {
	if opts.Command == "" || opts.OutputPath == "" {
		return Evidence{}, errors.New("command and output path are required")
	}
	outputPath := opts.OutputPath
	if !filepath.IsAbs(outputPath) && opts.WorkDir != "" {
		outputPath = filepath.Join(opts.WorkDir, outputPath)
	}
	if _, err := os.Lstat(outputPath); err == nil {
		return Evidence{}, errors.New("output path already exists")
	} else if !os.IsNotExist(err) {
		return Evidence{}, fmt.Errorf("inspect output path: %w", err)
	}
	start := time.Now()
	cmd := exec.Command(opts.Command, opts.Args...)
	cmd.Dir, cmd.Env, cmd.Stdin = opts.WorkDir, opts.Env, opts.Stdin
	if err := cmd.Run(); err != nil {
		return Evidence{}, fmt.Errorf("direct execution failed: %w", err)
	}
	outputBytes, err := os.ReadFile(outputPath)
	if err != nil {
		return Evidence{}, fmt.Errorf("read output: %w", err)
	}
	digest := sha256.Sum256(outputBytes)
	digestHex := hex.EncodeToString(digest[:])
	verified := false
	if opts.ExpectedSHA256 != "" {
		if opts.ExpectedSHA256 != digestHex {
			return Evidence{}, errors.New("output digest does not match expected SHA-256")
		}
		verified = true
	}
	output := ActualOutputVerified{Verified: verified, OutputPath: outputPath,
		SizeBytes: int64(len(outputBytes)), SHA256Hex: digestHex}
	if verified {
		output.VerifiedAtMillis = time.Now().UnixMilli()
		output.Verifier = "caller-expected-sha256"
	}
	return Evidence{
		Schema:                   EvidenceSchema,
		ExecutionRequestAccepted: ExecutionRequestAccepted{Accepted: true, AcceptedAtMillis: start.UnixMilli()},
		WorkerExecuted: WorkerExecuted{Executed: true, StartMillis: start.UnixMilli(),
			EndMillis: time.Now().UnixMilli(), ExitCode: cmd.ProcessState.ExitCode(), ProcessID: cmd.Process.Pid},
		ActualOutputVerified: output,
	}, nil
}
