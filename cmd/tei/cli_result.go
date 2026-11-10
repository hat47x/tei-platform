package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"tei/internal/core"
	"tei/internal/dependency"
	"tei/internal/knowledgecontext"
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
	// Workspace identifies the Workspace the result was read from (#3455).
	// It sits at the top level, next to schemaVersion, so every read command
	// answers "which Workspace" in the same place. Older result-level
	// `workspace` fields remain for readers that already unwrap them.
	Workspace *workspace.WorkspaceIdentity `json:"workspace,omitempty"`
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
	resultCodeKnowledgeFailed     = "KNOWLEDGE_BUILD_FAILED"
)

const (
	resultModeIndex         = "index"
	resultModeSubject       = "subject"
	resultModeContext       = "context"
	resultModeDisclosure    = "disclosure"
	resultModeQualification = "qualification"
	resultModePlan          = "plan"
	resultModeHandoff       = "handoff"
	resultModeReview        = "review"
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
		"plugin-lock", "plugin-resolve", "plugin-runtime-plan",
		"knowledge-context", "knowledge-qualify",
		"knowledge-review-plan", "knowledge-review-handoff",
		"knowledge-review-disclose", "knowledge-data-design-candidate-review":
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

// resultEnvelopeWithWorkspace is resultEnvelope plus the Workspace identity
// that the read observed (#3455).
func resultEnvelopeWithWorkspace(command, mode string, result any, identity *workspace.WorkspaceIdentity) cliResultEnvelope {
	envelope := resultEnvelope(command, mode, result)
	envelope.Workspace = identity
	return envelope
}

// knowledgeFailure maps a knowledge-* build error to its envelope. A missing
// or malformed argument is the caller's input problem and answers
// INVALID_ARGUMENT; anything else stays KNOWLEDGE_BUILD_FAILED (#3733).
func knowledgeFailure(command string, cause error) cliResultEnvelope {
	code := resultCodeKnowledgeFailed
	var argument *argumentError
	if errors.As(cause, &argument) {
		code = resultCodeInvalidArgument
	}
	return resultFailure(command, code, cause)
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
	return emitResult(out, resultEnvelope("workspace-files", "", workspaceFilesValue{
		ManagedFiles:       r.ManagedFiles,
		SourceRootFindings: r.SourceRootFindings,
		Workspace:          workspaceIdentityPointer(r),
	}), 0)
}

// workspaceIdentityPointer returns the Workspace identity as a pointer so the
// JSON field is omitted only when the manifest was not read at all (#3181).
func workspaceIdentityPointer(r workspace.Result) *workspace.WorkspaceIdentity {
	identity := r.Identity()
	return &identity
}

// workspaceFilesValue is the workspace-files projection. ReadByLoader on each
// ManagedFile separates managed-but-not-interpreted files from files a domain
// loader read (#2823). SourceRootFindings names unknown or missing roots.
type workspaceFilesValue struct {
	ManagedFiles       []workspace.ManagedFile                `json:"managedFiles"`
	SourceRootFindings []workspace.WorkspaceSourceRootFinding `json:"sourceRootFindings,omitempty"`
	// Workspace identifies which Workspace the listed files came from (#3181).
	Workspace *workspace.WorkspaceIdentity `json:"workspace,omitempty"`
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
func resultCommandJSON(out io.Writer, command, workspacePath, worldPath, environmentID, pluginLockPath string, diagnostic, environmentSpecified bool, extraArgs int) (int, error) {
	if (workspacePath == "") == (worldPath == "") {
		return resultInvalidArgument(out, command, command+" requires exactly one of -workspace or -world")
	}
	if extraArgs != 0 {
		return resultInvalidArgument(out, command, command+" does not accept positional arguments")
	}
	w, identity, err := loadWorldWithIdentity(worldPath, workspacePath)
	if err != nil {
		return resultLoadFailure(out, command, err)
	}
	ds := validation.Validate(w)
	if diagnostic && command == "plugin-resolve" {
		report := cliResultEnvelope{SchemaVersion: resultSchemaVersion(command), Mode: "unvalidated-candidates", Error: &cliFailure{Code: "UNVALIDATED_CANDIDATES", Message: "diagnostic only; not validated or authorized for execution"}}
		detail := struct {
			WorkspaceValidated bool                    `json:"workspaceValidated"`
			Diagnostics        []validation.Diagnostic `json:"diagnostics"`
			Candidates         *plugin.ResolvedSet     `json:"candidates,omitempty"`
			SelectionError     *cliFailure             `json:"selectionError,omitempty"`
		}{Diagnostics: ds}
		fail := func(code, message string) {
			detail.SelectionError = &cliFailure{Code: code, Message: message}
		}
		id, idErr := core.ParseCanonicalID(environmentID)
		if environmentID == "" {
			fail(resultCodeInvalidArgument, "plugin-resolve requires -environment")
		} else if idErr != nil {
			fail(resultCodeInvalidArgument, idErr.Error())
		} else if env, ok := findEnvironment(w, id); !ok {
			fail(resultCodeUnknownEnvironment, "unknown environment "+environmentID)
		} else {
			catalog := plugin.Catalog{Plugins: w.Plugins, Bundles: w.Bundles}
			var resolved plugin.ResolvedSet
			var resolveErr error
			if pluginLockPath != "" {
				lock, decodeErr := decodePluginLockFile(pluginLockPath)
				if decodeErr != nil {
					fail(resultCodeInvalidPluginLock, decodeErr.Error())
				} else {
					resolved, resolveErr = catalog.ResolveEnvironmentWithLock(env, lock)
				}
			} else {
				resolved, resolveErr = catalog.ResolveEnvironment(env)
			}
			if resolveErr != nil {
				// Same rule as plugin-runtime-plan: an unusable lock is an input
				// failure with INVALID_PLUGIN_LOCK, not a resolve outcome (#3184).
				if plugin.IsInvalidLock(resolveErr) {
					fail(resultCodeInvalidPluginLock, resolveErr.Error())
				} else {
					fail(resultCodePluginResolveFailed, resolveErr.Error())
				}
			} else if detail.SelectionError == nil {
				detail.Candidates = &resolved
			}
		}
		report.Result = detail
		return emitResult(out, report, 1)
	}
	if validation.HasErrors(ds) {
		return resultValidationFailure(out, command, ds)
	}
	switch command {
	case "requirements":
		return emitResult(out, resultEnvelopeWithWorkspace(command, "", dependency.Derive(w), identity), 0)
	case "plugin-lock":
		environments := w.Environments
		mode := ""
		if environmentSpecified {
			id, err := core.ParseCanonicalID(environmentID)
			if err != nil {
				return resultInvalidArgument(out, command, "invalid -environment: "+err.Error())
			}
			env, ok := findEnvironment(w, id)
			if !ok {
				return emitResult(out, resultFailure(command, resultCodeUnknownEnvironment, fmt.Errorf("unknown environment %s", environmentID)), 2)
			}
			environments = append(environments[:0:0], env)
			mode = "environment"
		}
		lock, err := plugin.BuildLock(plugin.Catalog{Plugins: w.Plugins, Bundles: w.Bundles}, environments)
		if err != nil {
			return emitResult(out, resultFailure(command, resultCodePluginLockFailed, err), 2)
		}
		return emitResult(out, resultEnvelopeWithWorkspace(command, mode, lock, identity), 0)
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
				lock, lockErr := decodePluginLockFile(pluginLockPath)
				if lockErr != nil {
					return emitResult(out, resultFailure(command, resultCodeInvalidPluginLock, lockErr), 2)
				}
				r, err = catalog.ResolveEnvironmentWithLock(env, lock)
			} else {
				r, err = catalog.ResolveEnvironment(env)
			}
			if err != nil {
				if plugin.IsInvalidLock(err) {
					return emitResult(out, resultFailure(command, resultCodeInvalidPluginLock, err), 2)
				}
				if plugin.IsLockMismatch(err) {
					return emitResult(out, resultFailure(command, resultCodeInvalidPluginLock, err), 2)
				}
				return emitResult(out, resultFailure(command, resultCodePluginResolveFailed, err), 2)
			}
			return emitResult(out, resultEnvelopeWithWorkspace(command, "", r, identity), 0)
		}
		lock, lockErr := decodePluginLockFile(pluginLockPath)
		if lockErr != nil {
			return emitResult(out, resultFailure(command, resultCodeInvalidPluginLock, lockErr), 2)
		}
		plan, err := pluginruntime.BuildPlan(catalog, env, lock)
		if err != nil {
			// A lock whose bytes are unusable is an input failure (exit 2), the
			// same as the decode and resolve paths of plugin-resolve. A
			// well-formed lock that does not match this environment or catalog
			// is also the caller's input problem: the only fix is another lock.
			// Both answer INVALID_PLUGIN_LOCK with exit 2, so the same bad lock
			// is classified the same way whichever command reads it (#3765,
			// #3184).
			if plugin.IsInvalidLock(err) || plugin.IsLockMismatch(err) {
				return emitResult(out, resultFailure(command, resultCodeInvalidPluginLock, err), 2)
			}
			return emitResult(out, resultFailure(command, resultCodePluginPlanFailed, err), 1)
		}
		return emitResult(out, resultEnvelopeWithWorkspace(command, "", plan, identity), 0)
	}
	return resultInvalidArgument(out, command, "unsupported command "+command)
}

// requireKnowledgeSubject reports a missing -subject-kind or -subject-id as an
// input error. Before this the missing flag reached the build and came back as
// KNOWLEDGE_BUILD_FAILED, the same code a genuine construction failure uses, so
// a caller could not tell a forgotten flag from an unreadable Workspace
// (#3733).
func requireKnowledgeSubject(out io.Writer, command, subjectKind, subjectID string) (int, error, bool) {
	if strings.TrimSpace(subjectKind) == "" || strings.TrimSpace(subjectID) == "" {
		code, err := resultInvalidArgument(out, command, command+" requires -subject-kind and -subject-id")
		return code, err, true
	}
	return 0, nil, false
}

// knowledgeContextJSON wraps knowledge-context in the command envelope. The
// pre-envelope output had two shapes: the whole Context, or (with -disclose) a
// DisclosureProjection. `mode` distinguishes them: context | disclosure.
// Exit codes stay as they were: 2 for argument/build failures, 1 for a
// fingerprint mismatch the caller asked to enforce.
func knowledgeContextJSON(out io.Writer, workspacePath, subjectKind, subjectID, disclosure, expectedDataDesignFingerprint, expectedDisclosureFingerprint string, extraArgs int) (int, error) {
	if workspacePath == "" {
		return resultInvalidArgument(out, "knowledge-context", "knowledge-context requires -workspace")
	}
	if extraArgs != 0 {
		return resultInvalidArgument(out, "knowledge-context", "knowledge-context does not accept positional arguments")
	}
	if code, err, missing := requireKnowledgeSubject(out, "knowledge-context", subjectKind, subjectID); missing {
		return code, err
	}
	if disclosure == "" && expectedDisclosureFingerprint != "" {
		return resultInvalidArgument(out, "knowledge-context", "-expected-disclosure-fingerprint requires -disclose")
	}
	context, identity, err := buildKnowledgeContextWithIdentity(workspacePath, subjectKind, subjectID)
	if err != nil {
		return emitResult(out, knowledgeFailure("knowledge-context", err), 2)
	}
	if err := checkDataDesignFingerprint(context, expectedDataDesignFingerprint); err != nil {
		return emitResult(out, resultFailure("knowledge-context", resultCodeKnowledgeFailed, err), 1)
	}
	if disclosure != "" {
		projection, err := buildKnowledgeDisclosure(context, disclosure)
		if err != nil {
			return emitResult(out, resultFailure("knowledge-context", resultCodeKnowledgeFailed, err), 2)
		}
		if err := checkKnowledgeDisclosureFingerprint(projection, expectedDisclosureFingerprint); err != nil {
			return emitResult(out, resultFailure("knowledge-context", resultCodeKnowledgeFailed, err), 1)
		}
		return emitResult(out, resultEnvelopeWithWorkspace("knowledge-context", resultModeDisclosure, projection, identity), 0)
	}
	return emitResult(out, resultEnvelopeWithWorkspace("knowledge-context", resultModeContext, context, identity), 0)
}

// knowledgeQualifyJSON wraps knowledge-qualify in the command envelope. The
// pre-envelope output was one QualificationResult; the envelope has no mode.
func knowledgeQualifyJSON(out io.Writer, workspacePath, subjectKind, subjectID, requirementsPath string, extraArgs int) (int, error) {
	if workspacePath == "" {
		return resultInvalidArgument(out, "knowledge-qualify", "knowledge-qualify requires -workspace and does not accept -world")
	}
	if extraArgs != 0 {
		return resultInvalidArgument(out, "knowledge-qualify", "knowledge-qualify does not accept positional arguments")
	}
	if code, err, missing := requireKnowledgeSubject(out, "knowledge-qualify", subjectKind, subjectID); missing {
		return code, err
	}
	qualification, err := buildKnowledgeQualificationFromFile(workspacePath, subjectKind, subjectID, requirementsPath)
	if err != nil {
		return emitResult(out, knowledgeFailure("knowledge-qualify", err), 2)
	}
	return emitResult(out, resultEnvelope("knowledge-qualify", resultModeQualification, qualification), 0)
}

// knowledgeReviewPlanJSON wraps knowledge-review-plan in the command envelope.
// The pre-envelope output was one BuilderReviewPlan; the envelope has no flag-
// selected second shape, and `mode` names the result type the way the other
// knowledge commands do.
func knowledgeReviewPlanJSON(out io.Writer, workspacePath, subjectKind, subjectID, expectedReviewPlanFingerprint string, extraArgs int) (int, error) {
	if workspacePath == "" {
		return resultInvalidArgument(out, "knowledge-review-plan", "knowledge-review-plan requires -workspace and does not accept -world")
	}
	if extraArgs != 0 {
		return resultInvalidArgument(out, "knowledge-review-plan", "knowledge-review-plan does not accept positional arguments")
	}
	if code, err, missing := requireKnowledgeSubject(out, "knowledge-review-plan", subjectKind, subjectID); missing {
		return code, err
	}
	plan, err := buildBuilderReviewPlan(workspacePath, subjectKind, subjectID)
	if err != nil {
		return emitResult(out, knowledgeFailure("knowledge-review-plan", err), 2)
	}
	if err := checkBuilderReviewPlanFingerprint(plan, expectedReviewPlanFingerprint); err != nil {
		return emitResult(out, resultFailure("knowledge-review-plan", resultCodeKnowledgeFailed, err), 1)
	}
	return emitResult(out, resultEnvelope("knowledge-review-plan", resultModePlan, plan), 0)
}

// knowledgeReviewHandoffJSON wraps knowledge-review-handoff. The pre-envelope
// output was one BuilderReviewHandoff; `mode=handoff` lets a caller tell a
// handoff file from a qualification file when both feed the same decoder.
func knowledgeReviewHandoffJSON(out io.Writer, workspacePath, subjectKind, subjectID, requirementsPath, expectedReviewHandoffFingerprint string, extraArgs int) (int, error) {
	if workspacePath == "" {
		return resultInvalidArgument(out, "knowledge-review-handoff", "knowledge-review-handoff requires -workspace and does not accept -world")
	}
	if extraArgs != 0 {
		return resultInvalidArgument(out, "knowledge-review-handoff", "knowledge-review-handoff does not accept positional arguments")
	}
	if code, err, missing := requireKnowledgeSubject(out, "knowledge-review-handoff", subjectKind, subjectID); missing {
		return code, err
	}
	handoff, err := buildBuilderReviewHandoffFromFile(workspacePath, subjectKind, subjectID, requirementsPath)
	if err != nil {
		return emitResult(out, knowledgeFailure("knowledge-review-handoff", err), 2)
	}
	if err := checkBuilderReviewHandoffFingerprint(handoff, expectedReviewHandoffFingerprint); err != nil {
		return emitResult(out, resultFailure("knowledge-review-handoff", resultCodeKnowledgeFailed, err), 1)
	}
	return emitResult(out, resultEnvelope("knowledge-review-handoff", resultModeHandoff, handoff), 0)
}

// knowledgeReviewDiscloseJSON wraps knowledge-review-disclose. The pre-envelope
// output was one DisclosureProjection; `mode=disclosure` matches the value
// knowledge-context -disclose uses for the same result shape.
func knowledgeReviewDiscloseJSON(out io.Writer, workspacePath, subjectKind, subjectID, reviewHandoffPath, expectedReviewHandoffFingerprint, expectedDisclosureFingerprint string, extraArgs int) (int, error) {
	if workspacePath == "" {
		return resultInvalidArgument(out, "knowledge-review-disclose", "knowledge-review-disclose requires -workspace and does not accept -world")
	}
	if extraArgs != 0 {
		return resultInvalidArgument(out, "knowledge-review-disclose", "knowledge-review-disclose does not accept positional arguments")
	}
	if code, err, missing := requireKnowledgeSubject(out, "knowledge-review-disclose", subjectKind, subjectID); missing {
		return code, err
	}
	disclosure, handoff, err := buildBuilderReviewDisclosureFromFile(workspacePath, subjectKind, subjectID, reviewHandoffPath)
	if err != nil {
		return emitResult(out, knowledgeFailure("knowledge-review-disclose", err), 2)
	}
	if err := checkBuilderReviewHandoffFingerprint(handoff, expectedReviewHandoffFingerprint); err != nil {
		return emitResult(out, resultFailure("knowledge-review-disclose", resultCodeKnowledgeFailed, err), 1)
	}
	if err := checkKnowledgeDisclosureFingerprint(disclosure, expectedDisclosureFingerprint); err != nil {
		return emitResult(out, resultFailure("knowledge-review-disclose", resultCodeKnowledgeFailed, err), 1)
	}
	return emitResult(out, resultEnvelope("knowledge-review-disclose", resultModeDisclosure, disclosure), 0)
}

// knowledgeDataDesignCandidateReviewJSON wraps
// knowledge-data-design-candidate-review. The pre-envelope output was one
// candidate review packet.
func knowledgeDataDesignCandidateReviewJSON(out io.Writer, workspacePath, subjectKind, subjectID, candidateKind, candidateFingerprint, expectedDataDesignFingerprint, expectedEvidenceRequirementFingerprint, expectedReviewFingerprint string, extraArgs int) (int, error) {
	if workspacePath == "" {
		return resultInvalidArgument(out, "knowledge-data-design-candidate-review", "knowledge-data-design-candidate-review requires -workspace and does not accept -world")
	}
	if extraArgs != 0 {
		return resultInvalidArgument(out, "knowledge-data-design-candidate-review", "knowledge-data-design-candidate-review does not accept positional arguments")
	}
	if strings.TrimSpace(candidateKind) == "" || strings.TrimSpace(candidateFingerprint) == "" {
		return resultInvalidArgument(out, "knowledge-data-design-candidate-review",
			"knowledge-data-design-candidate-review requires -data-design-candidate-kind and -data-design-candidate-fingerprint")
	}
	context, identity, err := buildKnowledgeContextWithIdentity(workspacePath, subjectKind, subjectID)
	if err != nil {
		return emitResult(out, resultFailure("knowledge-data-design-candidate-review", resultCodeKnowledgeFailed, err), 2)
	}
	if err := checkDataDesignFingerprint(context, expectedDataDesignFingerprint); err != nil {
		return emitResult(out, resultFailure("knowledge-data-design-candidate-review", resultCodeKnowledgeFailed, err), 1)
	}
	packet, err := knowledgecontext.BuildDataDesignCandidateReview(context, candidateKind, candidateFingerprint, expectedEvidenceRequirementFingerprint)
	if err != nil {
		return emitResult(out, resultFailure("knowledge-data-design-candidate-review", resultCodeKnowledgeFailed, err), 1)
	}
	if err := checkDataDesignCandidateReviewFingerprint(packet, expectedReviewFingerprint); err != nil {
		return emitResult(out, resultFailure("knowledge-data-design-candidate-review", resultCodeKnowledgeFailed, err), 1)
	}
	return emitResult(out, resultEnvelopeWithWorkspace("knowledge-data-design-candidate-review", resultModeReview, packet, identity), 0)
}

func flagPart(command string) string {
	if command == "plugin-runtime-plan" {
		return " and -plugin-lock"
	}
	return ""
}
