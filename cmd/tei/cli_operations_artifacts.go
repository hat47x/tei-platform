package main

import (
	"fmt"

	"tei/internal/artifact"
	"tei/internal/operations"
)

// runOperationsCommand prints the derived operation surface. Gaps are part of
// the output: an operation that could not be derived is shown as a gap, not as
// a success.
func runOperationsCommand(workspacePath, applicationID string) error {
	if workspacePath == "" {
		return fmt.Errorf("operations requires -workspace")
	}
	surface, err := operations.SurfaceWorkspace(workspacePath, applicationID)
	if err != nil {
		return err
	}
	writeJSON(surface)
	return nil
}

// runArtifactsCommand prints the derived artifacts and their manifest. Findings
// from the consistency check are part of the output; an incomplete set is
// reported as such.
func runArtifactsCommand(workspacePath, applicationID string) error {
	if workspacePath == "" {
		return fmt.Errorf("artifacts requires -workspace")
	}
	if applicationID == "" {
		return fmt.Errorf("artifacts requires -application")
	}
	set, err := artifact.RenderWorkspace(workspacePath, applicationID)
	if err != nil {
		return err
	}
	writeJSON(set)
	return nil
}
