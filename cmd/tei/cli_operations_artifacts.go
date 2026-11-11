package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"tei/internal/artifact"
	"tei/internal/core"
	"tei/internal/operations"
	"tei/internal/periodstate"
)

// runOperationsCommand prints the derived operation surface. Gaps are part of
// the output: an operation that could not be derived is shown as a gap, not as
// a success.
func runOperationsCommand(workspacePath, applicationID, outPath string) error {
	if workspacePath == "" {
		return fmt.Errorf("operations requires -workspace")
	}
	surface, err := operations.SurfaceWorkspace(workspacePath, applicationID)
	if err != nil {
		return err
	}
	if outPath == "" {
		writeJSON(surface)
		return nil
	}
	encoded, err := json.MarshalIndent(surface, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(outPath, append(encoded, byte(10)), 0o644)
}

// runArtifactsCommand prints the derived artifacts and their manifest. Findings
// from the consistency check are part of the output; an incomplete set is
// reported as such. With -out, one file per artifact and a manifest file are
// written instead, and the manifest is printed.
func runArtifactsCommand(workspacePath, applicationID, outDir string, check bool) error {
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
	if check && len(set.Findings) > 0 {
		writeJSON(set.Findings)
		return fmt.Errorf("the artifact set is inconsistent: %d findings", len(set.Findings))
	}
	if outDir == "" {
		writeJSON(set)
		return nil
	}
	if err := writeArtifactFiles(outDir, set); err != nil {
		return err
	}
	writeJSON(set.Manifest)
	return nil
}

// writeArtifactFiles writes one file per artifact plus the manifest. Names come
// from the content identity, so the same meaning writes the same file names and
// an unchanged artifact keeps its file.
func writeArtifactFiles(outDir string, set artifact.WorkspaceSet) error {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}
	names := map[string]string{}
	for _, item := range set.Artifacts {
		name, body := artifactFileName(item)
		if previous, duplicated := names[name]; duplicated && previous != item.Digest {
			return fmt.Errorf("two artifacts want the same file name %s; refusing to overwrite", name)
		}
		names[name] = item.Digest
		if err := os.WriteFile(filepath.Join(outDir, name), []byte(body), 0o644); err != nil {
			return err
		}
	}
	manifest, err := json.MarshalIndent(struct {
		Manifest artifact.Manifest  `json:"manifest"`
		Findings []artifact.Finding `json:"findings"`
	}{Manifest: set.Manifest, Findings: set.Findings}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(outDir, "manifest.json"), append(manifest, '\n'), 0o644)
}

// artifactFileName names the file after what the artifact describes. A table
// definition is named by its table, a function definition by its application and
// the operation surface by its kind.
func artifactFileName(item artifact.Artifact) (string, string) {
	header := strings.TrimSpace(strings.SplitN(item.Content, "\n", 2)[0])
	switch artifact.Kind(item.Kind) {
	case artifact.KindTableDefinition:
		if name := strings.TrimPrefix(header, "table "); name != header && name != "" {
			return "table-" + slug(name) + ".md", item.Content
		}
	case artifact.KindFunctionDefinition:
		if name := strings.TrimPrefix(header, "application "); name != header && name != "" {
			return "function-" + slug(name) + ".md", item.Content
		}
	case artifact.KindOperationSurface:
		return "operation-surface.json", item.Content
	}
	digest := strings.TrimPrefix(item.Digest, "sha256:")
	if len(digest) > 8 {
		digest = digest[:8]
	}
	return slug(item.Kind) + "-" + slug(digest) + ".txt", item.Content
}

func slug(value string) string {
	replaced := strings.NewReplacer(".", "-", "/", "-", " ", "-").Replace(value)
	return strings.ToLower(replaced)
}

// runPeriodsCommand reports the operations that may not run because their period
// is not open. The period state is runtime information: the caller supplies it,
// and an unspecified period is not treated as closed.
func runPeriodsCommand(workspacePath, applicationID, statesFlag, stateFile, advance, closeCapability string) error {
	if workspacePath == "" {
		return fmt.Errorf("periods requires -workspace")
	}
	states, err := parsePeriodStates(statesFlag)
	if err != nil {
		return err
	}
	if stateFile != "" {
		fileStates, err := periodstate.File{Path: stateFile}.PeriodStates(context.Background(), applicationID)
		if err != nil {
			return err
		}
		for period, state := range fileStates {
			states[period] = state
		}
	}
	surface, err := operations.SurfaceWorkspace(workspacePath, applicationID)
	if err != nil {
		return err
	}
	if advance != "" {
		next, err := advancePeriod(advance, states[advance], stateFile)
		if err != nil {
			return err
		}
		// A period that declares close-requires must have run that capability before
		// it closes. The declaration was validated but never read when the period was
		// closed, so a close could name a capability and then close without it (D-9).
		//
		// The CLI itself runs no capability: it only records state. So a period with a
		// requirement is refused here rather than closed as if the capability had run.
		// A caller that can run the capability (the Command boundary) supplies
		// `-close-capability` once it has.
		requirements, err := operations.PeriodRequirementsFromWorkspace(workspacePath, applicationID)
		if err != nil {
			return err
		}
		if requirement, ok := requirements[core.CanonicalID(advance)]; ok {
			current := states[advance]
			if current == "" {
				current = operations.PeriodOpen
			}
			if err := operations.CheckCloseRequirement(requirement, current, next, core.CanonicalID(closeCapability)); err != nil {
				return err
			}
			// Reopening a closed period is a correction after closing, so the
			// period's declared correction capability must have run. The command
			// path asks for it already; this path advanced the state without
			// checking, so the declaration held on one entrance and not the other.
			if err := operations.CheckReopenRequirement(requirement, current, next, core.CanonicalID(closeCapability)); err != nil {
				return err
			}
		}
		if err := (periodstate.Writer{Path: stateFile}).Write(context.Background(), map[string]operations.PeriodState{
			advance: next,
		}); err != nil {
			return err
		}
		states[advance] = next
	}
	rejections := operations.CheckPeriods(surface, states)
	if rejections == nil {
		rejections = []operations.PeriodRejection{}
	}
	writeJSON(struct {
		Application string                            `json:"application,omitempty"`
		States      map[string]operations.PeriodState `json:"states"`
		Rejections  []operations.PeriodRejection      `json:"rejections"`
	}{Application: applicationID, States: states, Rejections: rejections})
	if len(rejections) > 0 {
		return fmt.Errorf("%d operation(s) are refused by the period state", len(rejections))
	}
	return nil
}

// advancePeriod returns the state one closing step past the current one, or an
// error when the step cannot be taken.
//
// A period absent from the states is treated as open, because an unstated period
// has never been closed. The write needs a file: a state given only on the command
// line would be forgotten, and the next run would see the old state and close the
// period twice.
func advancePeriod(period string, current operations.PeriodState, stateFile string) (operations.PeriodState, error) {
	if stateFile == "" {
		return "", fmt.Errorf("-advance requires -period-state-file, so the new state is kept")
	}
	if period == "" {
		return "", fmt.Errorf("-advance needs a period ID")
	}
	if current == "" {
		current = operations.PeriodOpen
	}
	step, ok := operations.CloseStep(current)
	if !ok {
		return "", fmt.Errorf("period %s is %s and cannot be closed further", period, current)
	}
	next, err := operations.NextState(current, step.To)
	if err != nil {
		return "", err
	}
	return next, nil
}

// parsePeriodStates reads "period=state,period=state". An empty flag yields no
// state, so nothing is refused by assumption.
func parsePeriodStates(value string) (map[string]operations.PeriodState, error) {
	states := map[string]operations.PeriodState{}
	if strings.TrimSpace(value) == "" {
		return states, nil
	}
	for _, pair := range strings.Split(value, ",") {
		period, state, ok := strings.Cut(strings.TrimSpace(pair), "=")
		if !ok || period == "" || state == "" {
			return nil, fmt.Errorf("period state %q must be period=state", pair)
		}
		switch operations.PeriodState(state) {
		case operations.PeriodOpen, operations.PeriodClosing, operations.PeriodClosed:
		default:
			return nil, fmt.Errorf("period state %q must be open, closing, or closed", state)
		}
		states[period] = operations.PeriodState(state)
	}
	return states, nil
}
