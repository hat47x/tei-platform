package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"tei/internal/validation"
	"tei/internal/workspace"
)

// This file implements the thin entry SB-C0 §3.4 selected: the CLI does not
// build in-process (it cannot import tools/internal/gobuilder), it launches an
// explicitly named child builder and records that it did so.
//
// It does not advertise itself in `tei commands` (SB-C0 §4.5, D-3): the catalog
// is scoped to read-only commands and SC0-11/SC0-12 fix that. `tei self-build
// -h` explains its own flags instead.
//
// Every accepted output keeps qualified=false. A produced artifact is not an
// acceptance, not L3, and not build or runtime authority.

const selfBuildSchema = "tei.cli.self-build/v0"

const (
	selfBuildCodeInvalidArgument = "INVALID_ARGUMENT"
	selfBuildCodePrecondition    = "SELF_BUILD_PRECONDITION_FAILED"
	selfBuildCodeChildFailed     = "SELF_BUILD_CHILD_FAILED"
	selfBuildBoundary            = "one child build launched by this binary; no closure qualification, no acceptance, no runtime authority"
	selfBuildNature              = "thin CLI entry that launches an explicit child builder; the CLI does not build in-process"
)

// selfBuildReport is the machine-readable result. It deliberately has no field
// that could be read as success-of-build-system: the only success it reports is
// that the child ran and its artifact bytes match the bytes on disk.
type selfBuildReport struct {
	SchemaVersion string             `json:"schemaVersion"`
	OK            bool               `json:"ok"`
	Nature        string             `json:"nature"`
	Workspace     string             `json:"workspace"`
	Intent        string             `json:"intent,omitempty"`
	SupplyRoot    string             `json:"supplyRoot,omitempty"`
	StateDir      string             `json:"buildState"`
	Output        string             `json:"output"`
	Child         *selfBuildChild    `json:"child,omitempty"`
	Artifact      *selfBuildArtifact `json:"artifact,omitempty"`
	Qualified     bool               `json:"qualified"`
	Boundary      string             `json:"boundary"`
	Error         *cliFailure        `json:"error,omitempty"`
}

type selfBuildChild struct {
	Executable       string   `json:"executable"`
	ExecutableDigest string   `json:"executableDigest"`
	Argv             []string `json:"argv"`
	ExitCode         int      `json:"exitCode"`
	StdoutDigest     string   `json:"stdoutDigest"`
	Stderr           string   `json:"stderr,omitempty"`
}

type selfBuildArtifact struct {
	Path         string `json:"path"`
	Digest       string `json:"digest"`
	Bytes        int64  `json:"bytes"`
	ChildDigest  string `json:"childReportedDigest,omitempty"`
	ChildMatches bool   `json:"childReportedDigestMatches"`
}

func runSelfBuildCommand(args []string, out io.Writer) (int, error) {
	fs := flag.NewFlagSet("self-build", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	workspacePath := fs.String("workspace", "", "TEI workspace directory that supplies the build intent")
	intent := fs.String("intent", "", "build intent ID or path; not resolved by this entry")
	supply := fs.String("supply", "", "offline module supply directory passed to the child")
	buildState := fs.String("build-state", "", "isolated state directory for this attempt (required; no default)")
	output := fs.String("output", "", "artifact output path; the entry refuses to overwrite it")
	builder := fs.String("builder", "", "exact child builder executable (required; never resolved from PATH)")
	childArgs := fs.String("child-args", "", "JSON array of extra arguments for the child")
	if err := fs.Parse(args); err != nil {
		return selfBuildFailure(out, selfBuildCodeInvalidArgument, err)
	}

	report := selfBuildReport{
		SchemaVersion: selfBuildSchema,
		Nature:        selfBuildNature,
		Workspace:     *workspacePath,
		Intent:        *intent,
		SupplyRoot:    *supply,
		StateDir:      *buildState,
		Output:        *output,
		Qualified:     false,
		Boundary:      selfBuildBoundary,
	}

	// Preconditions are checked before anything is started. Each one states why
	// no build began, so a caller cannot read a missing flag as a failed build.
	if *workspacePath == "" {
		report.Error = &cliFailure{Code: selfBuildCodeInvalidArgument, Message: "-workspace is required"}
		return emitSelfBuild(out, report, 2)
	}
	if *buildState == "" {
		return selfBuildPrecondition(out, report, "-build-state is required: a shared state directory would make two attempts indistinguishable")
	}
	if *output == "" {
		return selfBuildPrecondition(out, report, "-output is required")
	}
	if *builder == "" {
		return selfBuildPrecondition(out, report, "-builder is required: the entry never resolves a child from PATH")
	}
	if _, err := os.Stat(*output); err == nil {
		return selfBuildPrecondition(out, report, "output already exists; this entry does not overwrite ("+*output+")")
	} else if !os.IsNotExist(err) {
		return selfBuildPrecondition(out, report, "cannot inspect the output path: "+err.Error())
	}

	// The builder is an argument of this command, not a property of the
	// workspace, so its exact path is settled before the workspace is read. A
	// bare name is refused here rather than after a load, so the caller learns
	// that the flag is wrong before any workspace problem can mask it (SC0-09).
	exactBuilder, err := exactExecutablePath(*builder)
	if err != nil {
		return selfBuildPrecondition(out, report, err.Error())
	}
	builderDigest, err := fileDigest(exactBuilder)
	if err != nil {
		return selfBuildPrecondition(out, report, "cannot digest the child builder: "+err.Error())
	}

	// The workspace must be readable and must not have validation errors. A
	// syntactic or semantic error is a precondition failure, not a build failure.
	loaded, err := workspace.Load(*workspacePath)
	if err != nil {
		return selfBuildPrecondition(out, report, "workspace load failed: "+err.Error())
	}
	for _, diagnostic := range validation.Validate(loaded.World) {
		if diagnostic.Severity == validation.Error {
			return selfBuildPrecondition(out, report, "workspace has validation errors at "+diagnostic.Code+": "+diagnostic.Message)
		}
	}

	var extraArgs []string
	if strings.TrimSpace(*childArgs) != "" {
		if err := json.Unmarshal([]byte(*childArgs), &extraArgs); err != nil {
			return selfBuildFailure(out, selfBuildCodeInvalidArgument, fmt.Errorf("-child-args is not a JSON array of strings: %w", err))
		}
	}
	argv := append([]string{}, extraArgs...)
	if *supply != "" {
		argv = append(argv, "-supply", *supply)
	}
	// The child contract is the one the existing reference builder already
	// accepts: -artifact-out names the artifact, -state isolates the attempt.
	argv = append(argv, "-artifact-out", *output, "-state", *buildState)
	report.Child = &selfBuildChild{
		Executable:       exactBuilder,
		ExecutableDigest: builderDigest,
		Argv:             append([]string{exactBuilder}, argv...),
	}

	cmd := exec.Command(exactBuilder, argv...)
	var stdoutBuf, stderrBuf strings.Builder
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf
	runErr := cmd.Run()
	exitCode := 0
	if runErr != nil {
		exitCode = -1
		var exitErr *exec.ExitError
		if errors.As(runErr, &exitErr) {
			exitCode = exitErr.ExitCode()
		}
	}
	report.Child.ExitCode = exitCode
	report.Child.StdoutDigest = digestOfBytes([]byte(stdoutBuf.String()))
	report.Child.Stderr = strings.TrimSpace(stderrBuf.String())

	if exitCode != 0 {
		report.Error = &cliFailure{Code: selfBuildCodeChildFailed, Message: "child builder exited unsuccessfully"}
		return emitSelfBuild(out, report, 1)
	}

	digest, size, err := fileDigestAndSize(*output)
	if err != nil {
		report.Error = &cliFailure{Code: selfBuildCodeChildFailed, Message: "child reported success but no artifact could be read: " + err.Error()}
		return emitSelfBuild(out, report, 1)
	}
	report.Artifact = &selfBuildArtifact{Path: *output, Digest: digest, Bytes: size}
	// The child may report its own digest; the entry compares it with the bytes
	// on disk rather than trusting the report.
	if reported := childReportedDigest(stdoutBuf.String()); reported != "" {
		report.Artifact.ChildDigest = reported
		report.Artifact.ChildMatches = reported == digest
	}
	report.OK = true
	return emitSelfBuild(out, report, 0)
}

func selfBuildFailure(out io.Writer, code string, cause error) (int, error) {
	report := selfBuildReport{
		SchemaVersion: selfBuildSchema,
		Nature:        selfBuildNature,
		Qualified:     false,
		Boundary:      selfBuildBoundary,
		Error:         &cliFailure{Code: code, Message: cause.Error()},
	}
	return emitSelfBuild(out, report, 2)
}

func selfBuildPrecondition(out io.Writer, report selfBuildReport, message string) (int, error) {
	report.Error = &cliFailure{Code: selfBuildCodePrecondition, Message: message}
	return emitSelfBuild(out, report, 2)
}

func emitSelfBuild(out io.Writer, report selfBuildReport, exitCode int) (int, error) {
	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	if err := enc.Encode(report); err != nil {
		return exitCode, err
	}
	return exitCode, nil
}

// exactExecutablePath refuses a bare name so a PATH entry cannot silently stand
// in for the builder the caller meant (SB-C0 §4.2 condition 7, SC0-09).
func exactExecutablePath(argument string) (string, error) {
	trimmed := strings.TrimSpace(argument)
	if trimmed == "" {
		return "", errors.New("empty builder path")
	}
	if !strings.Contains(trimmed, "/") && !strings.Contains(trimmed, string(filepath.Separator)) {
		return "", fmt.Errorf("-builder %q is a bare name; pass an absolute or relative path so its digest can be recorded", trimmed)
	}
	absolute, err := filepath.Abs(trimmed)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(absolute)
	if err != nil {
		return "", fmt.Errorf("-builder is not readable: %w", err)
	}
	if !info.Mode().IsRegular() {
		return "", errors.New("-builder is not a regular file")
	}
	return absolute, nil
}

func fileDigest(path string) (string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return digestOfBytes(raw), nil
}

func fileDigestAndSize(path string) (string, int64, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", 0, err
	}
	return digestOfBytes(raw), int64(len(raw)), nil
}

func digestOfBytes(raw []byte) string {
	sum := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(sum[:])
}

// childReportedDigest reads only a documented field; an unparseable or absent
// report yields "", which the caller records as "not reported" rather than as a
// mismatch.
func childReportedDigest(stdout string) string {
	decoder := json.NewDecoder(strings.NewReader(stdout))
	var child struct {
		ArtifactDigest string `json:"artifactDigest"`
	}
	if err := decoder.Decode(&child); err != nil {
		return ""
	}
	return child.ArtifactDigest
}

// selfBuildHelpText describes the entry's own flags. It is reached through the
// existing `-h` path so the command stays discoverable without adding a
// non-read-only entry to the catalog (SB-C0 §4.5, D-3).
func selfBuildHelpText() string {
	return "usage: tei self-build -workspace <path> -builder <path> -build-state <dir> -output <path> [options]\n" +
		"\tthe CLI does not build in-process; it launches the named child builder and records that it did\n" +
		"\tqualified is always false: an artifact is not acceptance, L3, or runtime authority\n"
}
