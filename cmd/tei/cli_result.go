package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"tei/internal/core"
	"tei/internal/dependency"
	"tei/internal/plugin"
	pluginruntime "tei/internal/plugin/runtime"
	"tei/internal/validation"
	"tei/internal/workspace"
)

// The result envelope wraps the JSON outputs of the read-only commands that
// previously printed their internal structures bare (#2354). It is additive:
// the previous output is carried in `result`, so readers that unwrapped the
// old shape keep working one level down. `source-index` sets `mode` because
// its pre-envelope output had two shapes (whole index, or one subject's
// sources) selected by the presence of -subject-kind/-subject-id.
type cliResultEnvelope struct {
	SchemaVersion string      `json:"schemaVersion"`
	OK            bool        `json:"ok"`
	Mode          string      `json:"mode,omitempty"`
	Result        any         `json:"result,omitempty"`
	Error         *cliFailure `json:"error,omitempty"`
}

type cliFailure struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

const (
	resultCodeInvalidArgument     = "INVALID_ARGUMENT"
	resultCodeWorkspaceLoadFailed = "WORKSPACE_LOAD_FAILED"
	resultCodeSourceIndexFailed   = "SOURCE_INDEX_FAILED"
	resultCodeValidationFailed    = "WORKSPACE_VALIDATION_FAILED"
	resultCodePluginLockFailed    = "PLUGIN_LOCK_FAILED"
	resultCodePluginResolveFailed = "PLUGIN_RESOLVE_FAILED"
	resultCodePluginPlanFailed    = "PLUGIN_RUNTIME_PLAN_FAILED"
	resultCodeUnknownEnvironment  = "UNKNOWN_ENVIRONMENT"
	resultCodeInvalidPluginLock   = "INVALID_PLUGIN_LOCK"
)

const (
	resultModeIndex   = "index"
	resultModeSubject = "subject"
)

// isResultEnvelopeCommand reports whether the command wraps its output in the
// #2354 result envelope. The command catalogue keeps advertising the same
// flags; only the output shape and the failure path change. `world` stays a
// bare ir.World write for now: the critical-path evidence audit
// (tools/reference-critical-path-evidence-audit/live-r1-smoke.mjs) digests its
// stdout bytes against the accepted derived World, so wrapping it is a
// separate contract decision recorded on #2354.
func isResultEnvelopeCommand(command string) bool {
	switch command {
	case "workspace-files", "source-index", "requirements",
		"plugin-lock", "plugin-resolve", "plugin-runtime-plan":
		return true
	}
	return false
}

func resultSchemaVersion(command string) string {
	return "tei.cli." + command + "/v0"
}

func resultEnvelope(command, mode string, result any) cliResultEnvelope {
	return cliResultEnvelope{
		SchemaVersion: resultSchemaVersion(command),
		OK:            true,
		Mode:          mode,
		Result:        result,
	}
}

func resultFailure(command, code string, cause error) cliResultEnvelope {
	return cliResultEnvelope{
		SchemaVersion: resultSchemaVersion(command),
		Error:         &cliFailure{Code: code, Message: cause.Error()},
	}
}

// emitResult writes one envelope and returns the process exit code. It follows
// the C1 convention: a single JSON diagnostic on stdout, no second stderr
// message that a caller would have to reconcile.
func emitResult(out io.Writer, report cliResultEnvelope, exitCode int) (int, error) {
	enc := json.NewEncoder(out)
	enc.SetIndent("", "  ")
	if err := enc.Encode(report); err != nil {
		return exitCode, err
	}
	return exitCode, nil
}

func resultInvalidArgument(out io.Writer, command, message string) (int, error) {
	return emitResult(out, resultFailure(command, resultCodeInvalidArgument, errors.New(message)), 2)
}

func resultLoadFailure(out io.Writer, command string, cause error) (int, error) {
	return emitResult(out, resultFailure(command, resultCodeWorkspaceLoadFailed, cause), 2)
}

// resultValidationFailure keeps the pre-envelope exit semantics of
// validateOrExit: semantic errors end the command with exit code 1. The
// envelope carries the first error so callers see one machine-readable cause;
// full diagnostics remain the job of validate -json.
func resultValidationFailure(out io.Writer, command string, ds []validation.Diagnostic) (int, error) {
	var cause error
	for _, d := range ds {
		if d.Severity == validation.Error {
			cause = fmt.Errorf("%s: %s", d.Code, d.Message)
			break
		}
	}
	return emitResult(out, resultFailure(command, resultCodeValidationFailed, cause), 1)
}

// workspaceFilesJSON wraps the managed file list in the command envelope.
func workspaceFilesJSON(out io.Writer, workspacePath, worldPath string, extraArgs int) (int, error) {
	if workspacePath == "" || worldPath != "" {
		return resultInvalidArgument(out, "workspace-files", "workspace-files requires -workspace and does not accept -world")
	}
	if extraArgs != 0 {
		return resultInvalidArgument(out, "workspace-files", "workspace-files does not accept positional arguments")
	}
	r, err := workspace.Load(workspacePath)
	if err != nil {
		return resultLoadFailure(out, "workspace-files", err)
	}
	return emitResult(out, resultEnvelope("workspace-files", "", r.ManagedFiles), 0)
}

// sourceIndexJSON wraps the Source Index inspection. The whole-index form and
// the single-subject form share one schema version and are distinguished by
// `mode`, because both are the same computation over the same index.
func sourceIndexJSON(out io.Writer, workspacePath, worldPath, subjectKind, subjectID string, extraArgs int) (int, error) {
	if workspacePath == "" || worldPath != "" {
		return resultInvalidArgument(out, "source-index", "source-index requires -workspace and does not accept -world")
	}
	if extraArgs != 0 {
		return resultInvalidArgument(out, "source-index", "source-index does not accept positional arguments")
	}
	if (subjectKind == "") != (subjectID == "") {
		return resultInvalidArgument(out, "source-index", "-subject-kind and -subject-id must be supplied together")
	}
	inspection, mode, err := buildSourceIndexInspection(workspacePath, subjectKind, subjectID)
	if err != nil {
		var stage *sourceIndexStageError
		code := resultCodeSourceIndexFailed
		if errors.As(err, &stage) && stage.Stage == sourceIndexStageLoad {
			code = resultCodeWorkspaceLoadFailed
		}
		return emitResult(out, resultFailure("source-index", code, err), 2)
	}
	return emitResult(out, resultEnvelope("source-index", mode, inspection), 0)
}

// resultCommandJSON serves requirements, plugin-lock, plugin-resolve
// and plugin-runtime-plan. Their pre-envelope dispatch exited from five places
// with stderr text; every failure here is one envelope instead, with the same
// exit codes (2 for argument/load/resolve failures, 1 for semantic validation
// and plugin runtime plan build failures).
func resultCommandJSON(out io.Writer, command, workspacePath, worldPath, environmentID, pluginLockPath string, extraArgs int) (int, error) {
	if (workspacePath == "") == (worldPath == "") {
		return resultInvalidArgument(out, command, command+" requires exactly one of -workspace or -world")
	}
	if extraArgs != 0 {
		return resultInvalidArgument(out, command, command+" does not accept positional arguments")
	}
	w, err := loadWorld(worldPath, workspacePath)
	if err != nil {
		return resultLoadFailure(out, command, err)
	}
	ds := validation.Validate(w)
	if validation.HasErrors(ds) {
		return resultValidationFailure(out, command, ds)
	}
	switch command {
	case "requirements":
		return emitResult(out, resultEnvelope(command, "", dependency.Derive(w)), 0)
	case "plugin-lock":
		lock, err := plugin.BuildLock(plugin.Catalog{Plugins: w.Plugins, Bundles: w.Bundles}, w.Environments)
		if err != nil {
			return emitResult(out, resultFailure(command, resultCodePluginLockFailed, err), 2)
		}
		return emitResult(out, resultEnvelope(command, "", lock), 0)
	case "plugin-resolve", "plugin-runtime-plan":
		if environmentID == "" || (command == "plugin-runtime-plan" && pluginLockPath == "") {
			return resultInvalidArgument(out, command, command+" requires -environment"+flagPart(command))
		}
		env, ok := findEnvironment(w, core.MustCanonicalID(environmentID))
		if !ok {
			return emitResult(out, resultFailure(command, resultCodeUnknownEnvironment, fmt.Errorf("unknown environment %s", environmentID)), 2)
		}
		catalog := plugin.Catalog{Plugins: w.Plugins, Bundles: w.Bundles}
		if command == "plugin-resolve" {
			var r plugin.ResolvedSet
			if pluginLockPath != "" {
				var lock plugin.Lock
				if err := decodeJSONFile(pluginLockPath, &lock); err != nil {
					return emitResult(out, resultFailure(command, resultCodeInvalidPluginLock, err), 2)
				}
				r, err = catalog.ResolveEnvironmentWithLock(env, lock)
			} else {
				r, err = catalog.ResolveEnvironment(env)
			}
			if err != nil {
				return emitResult(out, resultFailure(command, resultCodePluginResolveFailed, err), 2)
			}
			return emitResult(out, resultEnvelope(command, "", r), 0)
		}
		var lock plugin.Lock
		if err := decodeJSONFile(pluginLockPath, &lock); err != nil {
			return emitResult(out, resultFailure(command, resultCodeInvalidPluginLock, err), 2)
		}
		plan, err := pluginruntime.BuildPlan(catalog, env, lock)
		if err != nil {
			return emitResult(out, resultFailure(command, resultCodePluginPlanFailed, err), 1)
		}
		return emitResult(out, resultEnvelope(command, "", plan), 0)
	}
	return resultInvalidArgument(out, command, "unsupported command "+command)
}

func flagPart(command string) string {
	if command == "plugin-runtime-plan" {
		return " and -plugin-lock"
	}
	return ""
}
