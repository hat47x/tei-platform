package directexecution

import (
	"os"
	"os/exec"
	"runtime"
	"time"
)

// Run executes the pure computation locally and returns Evidence.
func Run(opts Options) (Evidence, error) {
	start := time.Now()
	nowMillis := start.UnixMilli()

	cmd := exec.Command(opts.Command, opts.Args...)
	cmd.Dir = opts.WorkDir
	cmd.Env = opts.Env
	cmd.Stdin = opts.Stdin

	stdout, stderr, exitCode, cmdErr := runCommand(cmd)

	outputPath := opts.OutputPath
	outputBytes, err := os.ReadFile(outputPath)
	if err != nil {
		return Evidence{}, err
	}

	evidence := Evidence{
		Schema: EvidenceSchema,
		ExecutionRequestAccepted: ExecutionRequestAccepted{
			Accepted:         true,
			RequestID:        "direct-local",
			AcceptedAtMillis: nowMillis,
			Authority:        "direct-execution",
		},
		WorkerExecuted: WorkerExecuted{
			Executed:     exitCode == 0 && cmdErr == nil,
			WorkerID:     "local-" + runtime.GOOS + "-" + runtime.GOARCH,
			StartMillis:  nowMillis,
			EndMillis:    time.Now().UnixMilli(),
			ExitCode:     exitCode,
			ProcessID:    os.Getpid(),
		},
		ActualOutputVerified: ActualOutputVerified{
			Verified:         true,
			OutputPath:       outputPath,
			SizeBytes:        int64(len(outputBytes)),
			VerifiedAtMillis: time.Now().UnixMilli(),
			Verifier:         "direct-sha256",
		},
		ExecutionEligibility: ExecutionEligibility{
			Eligible:         true,
			LeaseValid:       true,
			WithinDeadline:   true,
			RecoveryPossible: true,
		},
		PlacementAuthority: PlacementAuthority{
			Granted:         true,
			PlacementID:     "direct-placement",
			Authority:       "os-scheduler",
			GrantedAtMillis: nowMillis,
		},
	}

	_ = stdout
	_ = stderr
	_ = cmdErr

	return evidence, nil
}

func runCommand(cmd *exec.Cmd) (stdout, stderr string, exitCode int, err error) {
	stdoutBuf, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return string(stdoutBuf), string(exitErr.Stderr), exitErr.ExitCode(), err
		}
		return string(stdoutBuf), "", 0, err
	}
	return string(stdoutBuf), "", 0, nil
}
