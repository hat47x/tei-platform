package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"tei/internal/plugin"
	"tei/internal/syntax"
)

// reference-plugin-package-build-child is the P-1 child of SB-C0 §4.10: it
// consumes an exact BuildPlan, materializes the Plugin package from it, writes
// the artifact where `tei self-build` expects it, and reports the plan
// fingerprint it actually used.
//
// It exists because the earlier survey (§4.10.1) found that no tool both
// satisfied the child grammar (-artifact-out, -state, one JSON on stdout) and
// reported a plan fingerprint. Without such a child the cross-check can only be
// `unperformed`.
//
// The report is a claim about the plan this process loaded and validated, not
// about the meaning that produced it and not about the artifact being that
// plan's output. The boundary string says so.

const (
	childSchema   = "tei.reference.plugin-package-build-child/v0"
	childBoundary = "one child process that loaded and validated an exact BuildPlan and materialized its Plugin package; this is not acceptance, not L3, and not proof that the plan came from any particular declaration"
)

type childReport struct {
	SchemaVersion   string `json:"schemaVersion"`
	Nature          string `json:"nature"`
	PlanPath        string `json:"planPath"`
	PlanBytesDigest string `json:"planBytesDigest"`
	PlanFingerprint string `json:"planFingerprint"`
	PluginID        string `json:"pluginId"`
	PluginVersion   string `json:"pluginVersion"`
	OutputKind      string `json:"outputKind"`
	ArtifactPath    string `json:"artifactPath,omitempty"`
	ArtifactDigest  string `json:"artifactDigest,omitempty"`
	StateDir        string `json:"stateDir,omitempty"`
	Boundary        string `json:"claimBoundary"`
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("reference-plugin-package-build-child", flag.ContinueOnError)
	fs.SetOutput(stderr)
	planPath := fs.String("plan", "", "exact BuildPlan JSON to consume (required)")
	pluginPath := fs.String("plugin", "", "Plugin definition (.tei) the plan names (required)")
	artifactOut := fs.String("artifact-out", "", "where to write the package artifact (required)")
	stateDir := fs.String("state", "", "isolated state directory for this attempt")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if *planPath == "" || *pluginPath == "" || *artifactOut == "" {
		fmt.Fprintln(stderr, "-plan, -plugin and -artifact-out are required")
		return 2
	}

	report, err := materialize(*planPath, *pluginPath, *artifactOut, *stateDir)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	encoder := json.NewEncoder(stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(report); err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	return 0
}

// materialize reads the plan as bytes, hashes those bytes, decodes and
// validates the plan, then materializes the package. The byte digest and the
// fingerprint are recorded separately: the first is what this process held, the
// second is what internal/plugin derives from the normalized plan.
func materialize(planPath, pluginPath, artifactOut, stateDir string) (childReport, error) {
	planBytes, err := os.ReadFile(planPath)
	if err != nil {
		return childReport{}, fmt.Errorf("read plan: %w", err)
	}
	var plan plugin.BuildPlan
	decoder := json.NewDecoder(bytes.NewReader(planBytes))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&plan); err != nil {
		return childReport{}, fmt.Errorf("decode plan: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return childReport{}, errors.New("plan contains multiple JSON values")
		}
		return childReport{}, fmt.Errorf("decode trailing plan content: %w", err)
	}
	// Validate before touching the artifact: a plan that does not validate must
	// not produce output, so a failure cannot be read as "the build worked".
	if err := plan.Validate(); err != nil {
		return childReport{}, fmt.Errorf("plan does not validate: %w", err)
	}
	fingerprint, err := plugin.BuildPlanFingerprint(plan)
	if err != nil {
		return childReport{}, fmt.Errorf("plan fingerprint: %w", err)
	}

	definitionSource, err := os.ReadFile(pluginPath)
	if err != nil {
		return childReport{}, fmt.Errorf("read plugin definition: %w", err)
	}
	definition, err := syntax.NewParser(string(definitionSource)).ParsePlugin()
	if err != nil {
		return childReport{}, fmt.Errorf("parse plugin definition: %w", err)
	}
	artifact, err := plugin.MaterializePluginPackage(definition, plan)
	if err != nil {
		return childReport{}, err
	}

	if err := os.MkdirAll(filepath.Dir(artifactOut), 0o755); err != nil {
		return childReport{}, fmt.Errorf("prepare artifact directory: %w", err)
	}
	if err := os.WriteFile(artifactOut, artifact, 0o644); err != nil {
		return childReport{}, fmt.Errorf("write artifact: %w", err)
	}
	if stateDir != "" {
		if err := os.MkdirAll(stateDir, 0o755); err != nil {
			return childReport{}, fmt.Errorf("prepare state directory: %w", err)
		}
	}

	report := childReport{
		SchemaVersion:   childSchema,
		Nature:          "one child process that consumes an exact BuildPlan and materializes its Plugin package",
		PlanPath:        planPath,
		PlanBytesDigest: digestOf(planBytes),
		PlanFingerprint: fingerprint,
		PluginID:        string(plan.Plugin.ID),
		PluginVersion:   plan.Plugin.Version,
		OutputKind:      string(plan.OutputKind),
		ArtifactPath:    artifactOut,
		ArtifactDigest:  digestOf(artifact),
		StateDir:        stateDir,
		Boundary:        childBoundary,
	}
	return report, nil
}

func digestOf(raw []byte) string {
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:])
}
