package analyzer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
)

// RunPythonWorker invokes the detachable Editor Intelligence worker over a
// stable JSON protocol. SQLGlot's AST is intentionally not part of the wire
// format.
func RunPythonWorker(ctx context.Context, python, workerPath string, req Request) (Result, error) {
	payload, err := json.Marshal(req)
	if err != nil {
		return Result{}, err
	}
	cmd := exec.CommandContext(ctx, python, workerPath)
	cmd.Stdin = bytes.NewReader(payload)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return Result{}, fmt.Errorf("sql analyzer worker: %w: %s", err, stderr.String())
	}
	var out Result
	if err := json.Unmarshal(stdout.Bytes(), &out); err != nil {
		return Result{}, fmt.Errorf("decode sql analyzer response: %w", err)
	}
	return out, nil
}
