package runtime

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"hash"
	"io"
	"os/exec"
	"strings"
)

const defaultProcessResponseLimit = 8 << 20 // 8 MiB

// StdioProcessLauncher is the smallest real OS-process implementation of
// ProcessLauncher. Executable and argv are kept separate exactly as supplied.
// They are never passed through a shell and are never re-split or re-quoted.
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
	// MaxResponseBytes bounds stdout captured from an untrusted process. Zero
	// uses a conservative Reference default rather than disabling the limit.
	MaxResponseBytes int
}

var _ ProcessLauncher = StdioProcessLauncher{}
var _ SandboxedProcessRunner = StdioProcessLauncher{}

// InvokeSandboxed only provides the process transport. It does not install OS
// sandbox policy itself. HostSandboxLauncher must apply the required network,
// filesystem and secret adapters before calling this method.
func (l StdioProcessLauncher) InvokeSandboxed(ctx context.Context, command ProcessCommand, req Request) (Response, error) {
	return l.Invoke(ctx, command, req)
}

func (l StdioProcessLauncher) Invoke(ctx context.Context, command ProcessCommand, req Request) (Response, error) {
	if ctx == nil {
		return Response{}, fmt.Errorf("process plugin invocation requires context")
	}
	if strings.TrimSpace(command.Executable) == "" {
		return Response{}, fmt.Errorf("process plugin executable is empty")
	}
	if strings.ContainsAny(command.Executable, " \t\r\n") {
		return Response{}, fmt.Errorf("process plugin executable must be a single name/path; use argv for arguments")
	}

	input, err := json.Marshal(req)
	if err != nil {
		return Response{}, fmt.Errorf("encode process plugin request: %w", err)
	}
	input = append(input, '\n')

	limit := l.MaxResponseBytes
	if limit <= 0 {
		limit = defaultProcessResponseLimit
	}
	stdout := &boundedCapture{limit: limit}
	stderr := newDigestWriter()
	cmd := exec.CommandContext(ctx, command.Executable, command.Args...)
	cmd.Stdin = bytes.NewReader(input)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	if l.Env != nil {
		cmd.Env = append([]string(nil), l.Env...)
	}
	if l.Dir != "" {
		cmd.Dir = l.Dir
	}

	if err := cmd.Run(); err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return Response{}, fmt.Errorf("process plugin executable %q canceled: %w", command.Executable, ctxErr)
		}
		return Response{}, processExecutionError(command.Executable, err, stderr)
	}
	if stdout.overflow {
		return Response{}, fmt.Errorf("process plugin executable %q response exceeds %d bytes", command.Executable, limit)
	}

	dec := json.NewDecoder(bytes.NewReader(stdout.buf.Bytes()))
	var resp Response
	if err := dec.Decode(&resp); err != nil {
		return Response{}, fmt.Errorf("decode process plugin response from executable %q: %w", command.Executable, err)
	}
	var trailing any
	if err := dec.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return Response{}, fmt.Errorf("process plugin executable %q returned more than one JSON value", command.Executable)
		}
		return Response{}, fmt.Errorf("decode trailing process plugin output from executable %q: %w", command.Executable, err)
	}
	return resp, nil
}

type boundedCapture struct {
	limit    int
	buf      bytes.Buffer
	overflow bool
}

func (w *boundedCapture) Write(p []byte) (int, error) {
	n := len(p)
	remaining := w.limit - w.buf.Len()
	if remaining > 0 {
		take := len(p)
		if take > remaining {
			take = remaining
		}
		_, _ = w.buf.Write(p[:take])
	}
	if n > remaining {
		w.overflow = true
	}
	return n, nil
}

type digestWriter struct {
	n int64
	h hash.Hash
}

func newDigestWriter() *digestWriter { return &digestWriter{h: sha256.New()} }
func (w *digestWriter) Write(p []byte) (int, error) {
	w.n += int64(len(p))
	return w.h.Write(p)
}
func (w *digestWriter) hex() string { return hex.EncodeToString(w.h.Sum(nil)) }

func processExecutionError(command string, err error, stderr *digestWriter) error {
	// Plugin stderr may contain concrete secret material that the child learned
	// through a host injection mechanism. Keep diagnostics useful without
	// copying external-process text into TEI errors/logs.
	return fmt.Errorf("process plugin %q failed: %w (stderrBytes=%d stderrSHA256=%s)", command, err, stderr.n, stderr.hex())
}
