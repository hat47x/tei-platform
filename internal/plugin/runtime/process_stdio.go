package runtime

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

// StdioProcessLauncher is the smallest real OS-process implementation of
// ProcessLauncher. Command is treated as one executable name/path exactly as
// supplied. It is never passed through a shell and is never whitespace-split
// into argv elements.
//
// This launcher intentionally models a one-shot local child process. It does
// not claim that an OS PID is a durable runtime handle and does not provide
// host-restart or cross-host reattach semantics.
type StdioProcessLauncher struct {
	// Env, when non-nil, replaces the child environment in the same way as
	// exec.Cmd.Env. Leave nil to inherit the host process environment.
	Env []string
	// Dir, when non-empty, sets the child working directory. It is host
	// configuration and is not derived from plugin request payloads.
	Dir string
}

func (l StdioProcessLauncher) Invoke(ctx context.Context, command string, req Request) (Response, error) {
	if ctx == nil {
		return Response{}, fmt.Errorf("process plugin invocation requires context")
	}
	if strings.TrimSpace(command) == "" {
		return Response{}, fmt.Errorf("process plugin command is empty")
	}

	input, err := json.Marshal(req)
	if err != nil {
		return Response{}, fmt.Errorf("encode process plugin request: %w", err)
	}
	input = append(input, '\n')

	cmd := exec.CommandContext(ctx, command)
	cmd.Stdin = bytes.NewReader(input)
	if l.Env != nil {
		cmd.Env = append([]string(nil), l.Env...)
	}
	if l.Dir != "" {
		cmd.Dir = l.Dir
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return Response{}, fmt.Errorf("process plugin %q canceled: %w", command, ctxErr)
		}
		return Response{}, processExecutionError(command, err, stderr.Bytes())
	}

	dec := json.NewDecoder(bytes.NewReader(stdout.Bytes()))
	var resp Response
	if err := dec.Decode(&resp); err != nil {
		return Response{}, fmt.Errorf("decode process plugin response from %q: %w", command, err)
	}
	var trailing any
	if err := dec.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return Response{}, fmt.Errorf("process plugin %q returned more than one JSON value", command)
		}
		return Response{}, fmt.Errorf("decode trailing process plugin output from %q: %w", command, err)
	}
	return resp, nil
}

func processExecutionError(command string, err error, stderr []byte) error {
	// Plugin stderr may contain concrete secret material that the child learned
	// through a host injection mechanism. Keep diagnostics useful without
	// copying external-process text into TEI errors/logs.
	sum := sha256.Sum256(stderr)
	return fmt.Errorf("process plugin %q failed: %w (stderrBytes=%d stderrSHA256=%s)", command, err, len(stderr), hex.EncodeToString(sum[:]))
}
