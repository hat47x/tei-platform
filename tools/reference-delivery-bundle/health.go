package deliverybundle

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"time"
)

type HealthReceipt struct {
	Catalog      string `json:"catalog"`
	Digest       string `json:"digest"`
	Command      string `json:"command"`
	ExitCode     int    `json:"exitCode"`
	Healthy      bool   `json:"healthy"`
	DurationMs   int64  `json:"durationMs"`
	OutputDigest string `json:"outputDigest"`
	Error        string `json:"error,omitempty"`
}

// Health runs a trusted health-check command against the active deployment.
// The command is a user-supplied adapter; TEI does not start or stop a service.
func Health(statePath, command string, timeout time.Duration) (HealthReceipt, error) {
	if command == "" {
		return HealthReceipt{}, fmt.Errorf("health command is required")
	}
	state, err := Status(statePath)
	if err != nil {
		return HealthReceipt{}, err
	}
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	started := time.Now()
	cmd := exec.CommandContext(ctx, "bash", "-c", command)
	cmd.Env = append(os.Environ(),
		"TEI_CATALOG="+state.Active.Catalog,
		"TEI_ACTIVE_DIGEST="+state.Active.Digest,
		"TEI_STATE="+statePath,
	)
	output, runErr := cmd.CombinedOutput()
	receipt := HealthReceipt{
		Catalog: state.Active.Catalog, Digest: state.Active.Digest, Command: command,
		Healthy: runErr == nil, DurationMs: time.Since(started).Milliseconds(),
		OutputDigest: digestBytes(output),
	}
	if runErr == nil {
		receipt.ExitCode = 0
		return receipt, nil
	}
	var exit *exec.ExitError
	if errors.As(runErr, &exit) {
		receipt.ExitCode = exit.ExitCode()
	} else {
		receipt.ExitCode = -1
	}
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		receipt.Error = "timeout"
	} else {
		receipt.Error = runErr.Error()
	}
	return receipt, nil
}
