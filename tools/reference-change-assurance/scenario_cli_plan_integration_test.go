package main

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// Validate the real CLI producer, not just a Plan reconstructed by this
// Reference. Generation is read-only and does not run a business operation.
func TestScenarioBindingConsumesActualCLIGeneratedPlan(t *testing.T) {
	beforeRoot, afterRoot, scenarioPath, planPath := prepareScenarioPlanBinding(t, false)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	cmd := exec.CommandContext(ctx, "go", "run", "./cmd/tei", "scenario", "generate",
		"-scenario", scenarioPath, "-workspace", afterRoot)
	cmd.Dir = filepath.Join("..", "..")
	var out, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("actual tei scenario generate failed: %v; stderr=%s", err, stderr.String())
	}
	if err := os.WriteFile(planPath, out.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	result, err := reviewScenarioPlanDiff(beforeRoot, afterRoot, "order-entry-reference",
		workspaceKey("application", "order_entry"), scenarioPath, planPath)
	if err != nil {
		t.Fatalf("actual generated CLI Plan does not match read-only binding: %v", err)
	}
	if !result.PlanMatches || result.ScenarioStepCount != 6 || result.PlanPinsSourceRevision ||
		result.Qualification != "unresolved" || result.ExecutionAllowed {
		t.Fatalf("actual CLI generation was misqualified: %+v", result)
	}
}
