package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"tei/internal/application"
	"tei/internal/applicationview"
	"tei/internal/command"
	"tei/internal/command/lifecyclesqlite"
	"tei/internal/core"
	"tei/internal/hook"
	"tei/internal/operations"
	"tei/internal/periodstate"
	"tei/internal/semantic"
	"tei/internal/workspace"
)

type referenceCommandRun struct {
	SchemaVersion string                      `json:"schemaVersion"`
	Mode          string                      `json:"mode"`
	Plan          applicationview.CommandPlan `json:"plan"`
	Outcome       *semantic.Outcome           `json:"outcome,omitempty"`
	// OutcomeUndetermined states why no outcome was returned although the
	// capability declares some, so an absent outcome is not read as a decision.
	OutcomeUndetermined string               `json:"outcomeUndetermined,omitempty"`
	Effect              *semantic.DataEffect `json:"effect,omitempty"`
	// Committed reports that the configured UnitOfWork returned from Commit.
	// It is not evidence that data was persisted: a reference run uses
	// MemoryUnitOfWork, which counts the call without writing anything. Read it
	// together with Persisted and Durability (#2998).
	Committed bool `json:"committed"`
	// Persisted is true only when an independent observer saw the declared
	// effect after Commit. A reference run has no observer, so it is false.
	Persisted bool `json:"persisted"`
	// Persistence separates "not applicable" (reference) from "not observed"
	// (durable Commit without an independent read) and "observed".
	Persistence string `json:"persistence"`
	// Durability names the storage boundary: "none" for the reference executor.
	Durability string `json:"durability"`
	// Differences keeps the applied operations the declaration does not
	// describe, so a reader of the run output can see the gap without reading
	// the operation log.
	Differences []command.Difference `json:"differences,omitempty"`
	// AppliedUnobserved is true when the executor did not enumerate its
	// operations, so `differences` being empty is "not observed" rather than
	// "none found".
	AppliedUnobserved bool `json:"appliedUnobserved,omitempty"`
	// EffectReverted is true when the effect was applied and then put back
	// because an after-change phase refused. It is the one case where the run
	// applied something and the reported state is the state before the run.
	EffectReverted bool `json:"effectReverted,omitempty"`
	// ReportedEffectMissing is true when the capability declared an effect but
	// the executor reported none. The run is not rejected, but a reader must not
	// treat the absence as proof the declared effect was applied.
	ReportedEffectMissing bool `json:"reportedEffectMissing,omitempty"`
	// AuthorizationMode names how the run was authorized:
	// reference-allow-all, reference-allow-list, or external. It is evidence
	// about which check ran, not a grant of authority (#2997).
	AuthorizationMode string `json:"authorizationMode,omitempty"`
	// Hooks records the pattern hooks this run consulted before the command.
	// A refusal here is why the command did not run; an after-change hook is
	// named as skipped, because the reference run has no after-change point.
	Hooks *cliHookReport `json:"hooks,omitempty"`
}

type cliViewReport struct {
	SchemaVersion string `json:"schemaVersion"`
	OK            bool   `json:"ok"`
	Result        any    `json:"result,omitempty"`
	// WorldFingerprint and SourceIndexFingerprint are the same read-only
	// observations `model catalog`, `inspect`, `validate -json` and `trace`
	// report. `result.semanticFingerprint` is a different value: it covers only
	// the namespaces the Application uses, so the two must not be compared
	// (#3345). They are filled only for -format json.
	WorldFingerprint       string                `json:"worldFingerprint,omitempty"`
	SourceIndexFingerprint string                `json:"sourceIndexFingerprint,omitempty"`
	Error                  *cliInspectionFailure `json:"error,omitempty"`
	// Workspace identifies which Workspace the view was read from (#3455).
	Workspace *workspace.WorkspaceIdentity `json:"workspace,omitempty"`
}

// runViewCommandは、Application Patternの対話契約を解決し、textまたはjsonで返す。
// 意味の解決は applicationview に置き、ここでは引数・出力・終了コードだけを扱う。
// commandFailure maps an execution failure to the CLI code a caller needs in
// order to decide what to do next. A failure whose data effect may already be
// applied must not read like a clean rejection, and a lifecycle refusal is a
// replay decision rather than an execution error.
func commandFailure(err error) (string, string, bool) {
	// An ExecutionError already carries the machine-readable reason the caller
	// branches on, and the remedy differs by code: a missing input, a closed
	// period, a refused authorization and a failed commit all need different
	// handling. Report the code as it is.
	//
	// This used to be a hand-written allow-list, which silently flattened 15 of
	// the 21 codes internal/command returns into COMMAND_FAILED (#3807). A list
	// kept per entrance also drifted: the CLI knew seven, the Studio one.
	if code, err2, ok := command.ReportedFailure(err); ok {
		return code, err2, true
	}
	return "", "", false
}

func runViewCommand(args []string, out io.Writer) (int, error) {
	report := cliViewReport{SchemaVersion: "tei.cli.view/v0"}
	fail := func(code, message string, exit int) (int, error) {
		report.Error = &cliInspectionFailure{Code: code, Message: message}
		return writeViewReport(out, report, exit)
	}

	fs := flag.NewFlagSet("view", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	root := fs.String("workspace", "", "TEI workspace directory")
	applicationID := fs.String("application", "", "exact application ID")
	patternID := fs.String("pattern", "", "exact application pattern ID")
	packageID := fs.String("package", "", "exact application package ID")
	commandID := fs.String("command", "", "command capability to plan")
	execute := fs.Bool("execute", false, "reference execution of the planned command")
	referenceStore := fs.Bool("reference-store", false, "with -execute, apply the declared effect to an in-memory reference store and run the after-change hooks; the store is not persistence")
	periodStateFile := fs.String("period-state", "", "period state JSON for -execute")
	operationLogFile := fs.String("operation-log", "", "write reference operation records as JSON")
	lifecycleState := fs.String("lifecycle-state", "", "SQLite state file for restart-safe command lifecycle")
	lifecycleCommand := fs.String("lifecycle-command", "", "stable command id recorded in the lifecycle state")
	lifecycleOwner := fs.String("lifecycle-owner", "cli", "owner recorded in the lifecycle state")
	lifecycleGeneration := fs.Int("lifecycle-generation", 1, "generation recorded in the lifecycle state")
	var allowedCapabilities inputFlag
	fs.Var(&allowedCapabilities, "allow-capability", "capability allowed by the reference authorizer; repeatable")
	var inputPairs inputFlag
	fs.Var(&inputPairs, "input", "command or browse-query input as name=value; repeatable")
	environmentID := fs.String("environment", "", "environment ID that binds the browse pattern's datasource (with -execute and -pattern)")
	format := fs.String("format", "text", "text or json")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printHelp(out, fs, "view")
			return 0, nil
		}
		return fail("INVALID_ARGUMENT", err.Error(), 2)
	}
	if *referenceStore && !*execute {
		return fail("INVALID_ARGUMENT", newArgumentError("view requires -execute with -reference-store").Error(), 2)
	}
	if fs.NArg() != 0 || *root == "" || *applicationID == "" ||
		(*patternID != "" && *packageID != "") ||
		(*commandID != "" && *patternID == "") ||
		// -execute runs a command (-command) or reads a browse pattern (-pattern with
		// -environment); a bare -execute names neither.
		(*execute && *commandID == "" && *patternID == "") ||
		(*execute && *commandID == "" && *environmentID == "") ||
		(*environmentID != "" && (!*execute || *commandID != "")) ||
		*environmentID != strings.TrimSpace(*environmentID) ||
		(*periodStateFile != "" && !*execute) ||
		(*lifecycleState != "" && (!*execute || *lifecycleCommand == "")) ||
		(*lifecycleState == "" && *lifecycleCommand != "") ||
		(*lifecycleGeneration < 1) ||
		(len(allowedCapabilities) > 0 && !*execute) ||
		(*operationLogFile != "" && !*execute) ||
		// -input only feeds the command plan, which exists only with -command.
		// Without this, `view -application X -input order=ORD-1` succeeded and
		// dropped the value (and never parsed it), so a caller could believe it
		// had supplied input to a command that was never planned (#3778).
		(len(inputPairs) > 0 && *commandID == "" && !(*execute && *environmentID != "")) ||
		*root != strings.TrimSpace(*root) || *applicationID != strings.TrimSpace(*applicationID) ||
		*patternID != strings.TrimSpace(*patternID) || *packageID != strings.TrimSpace(*packageID) ||
		*commandID != strings.TrimSpace(*commandID) {
		return fail("INVALID_ARGUMENT", "view requires exact -workspace and -application, at most one of -pattern, -package, or -command, without positional arguments", 2)
	}
	if *execute && *format != "text" && *format != "json" {
		return fail("INVALID_ARGUMENT", "view -execute supports -format text or json", 2)
	}
	if *execute && *commandID == "" && *format != "json" {
		return fail("INVALID_ARGUMENT", "view -execute on a browse pattern supports -format json only", 2)
	}
	if *format != "text" && *format != "json" && *format != "html" && *format != "print" {
		return fail("INVALID_ARGUMENT", "view -format must be text, json, html, or print", 2)
	}

	loaded, err := workspace.Load(*root)
	if err != nil {
		return fail("WORKSPACE_LOAD_FAILED", err.Error(), 2)
	}
	// Every view shape is a read of this Workspace, so the identity belongs on
	// the report next to the fingerprints (#3455).
	report.Workspace = workspaceIdentityPointer(loaded)
	if *format == "json" {
		// The JSON projection reports the same read-only observations the other
		// read commands report (#3345). `result.semanticFingerprint` stays as it
		// is: it covers only the namespaces this Application uses, so it is a
		// narrower value, not a different name for worldFingerprint.
		//
		// A Workspace whose Source Index cannot be built keeps returning the
		// view: the projection is of the loaded World, and refusing here would
		// make `view` fail where it previously succeeded. The two fields are
		// omitted in that case rather than filled with a value the call did not
		// observe.
		if idx, indexErr := workspace.BuildSourceIndex(*root, loaded); indexErr == nil {
			if worldFP, sourceFP, fingerprintErr := operations.WorkspaceProjectionFingerprints(loaded.World, idx); fingerprintErr == nil {
				report.WorldFingerprint = worldFP
				report.SourceIndexFingerprint = sourceFP
			}
		}
	}
	if *execute && *commandID == "" {
		if *referenceStore {
			// -reference-store changes what a command execution does; a browse read
			// has no such effect, so asking for both is an argument problem.
			return fail("INVALID_ARGUMENT", "view -reference-store applies to -command only", 2)
		}
		pairs, err := parseInputPairs(inputPairs)
		if err != nil {
			return fail("INVALID_ARGUMENT", err.Error(), 2)
		}
		return runViewBrowseRead(loaded.World, core.CanonicalID(*applicationID), core.CanonicalID(*patternID), core.CanonicalID(*environmentID), pairs, report, out, fail)
	}
	if *patternID == "" && *packageID == "" && *commandID == "" {
		if *format != "text" && *format != "json" {
			return fail("INVALID_ARGUMENT", "view -application supports -format text or json", 2)
		}
		view, err := applicationview.BuildApplication(loaded.World, core.CanonicalID(*applicationID))
		if err != nil {
			return fail("VIEW_FAILED", err.Error(), 1)
		}
		report.OK = true
		report.Result = view
		if *format == "json" {
			return writeViewReport(out, report, 0)
		}
		return writeApplicationText(out, view)
	}
	if *packageID != "" {
		if *format != "text" && *format != "json" {
			return fail("INVALID_ARGUMENT", "view -package supports -format text or json", 2)
		}
		if *commandID != "" {
			return fail("INVALID_ARGUMENT", "view -package does not accept -command", 2)
		}
		view, err := applicationview.BuildPackage(loaded.World, core.CanonicalID(*applicationID), core.CanonicalID(*packageID))
		if err != nil {
			return fail("VIEW_FAILED", err.Error(), 1)
		}
		report.OK = true
		report.Result = view
		if *format == "json" {
			return writeViewReport(out, report, 0)
		}
		return writePackageText(out, view)
	}
	if *commandID != "" {
		if *format != "text" && *format != "json" {
			return fail("INVALID_ARGUMENT", "view -command supports -format text or json", 2)
		}
		values, err := parseInputPairs(inputPairs)
		if err != nil {
			return fail("INVALID_ARGUMENT", err.Error(), 2)
		}
		plan, err := applicationview.BuildCommandPlan(loaded.World, core.CanonicalID(*applicationID), core.CanonicalID(*patternID), core.CanonicalID(*commandID), values)
		if err != nil {
			return fail("VIEW_FAILED", err.Error(), 1)
		}
		if *execute {
			// A declared hook runs before the command when the reference run
			// can reach it: a before-change phase whose declared reads the
			// caller supplied. Everything else is reported as skipped or not
			// evaluated with a reason, so a phase that did not run never reads
			// as a phase that passed.
			// A run that applies its change consults the before-* phases first, then
			// applies, then consults the after-* phases: the change belongs between
			// the halves, and one pass has no point in between. A reference run keeps
			// the single pass, because it has no change to sit there.
			var hookReport *cliHookReport
			var hookErr error
			if !*referenceStore {
				hookReport, hookErr = runDeclaredHooks(context.Background(), loaded.World, core.CanonicalID(*applicationID), core.CanonicalID(*patternID), values)
			} else {
				hookReport, hookErr = runDeclaredHooksHalf(context.Background(), loaded.World, core.CanonicalID(*applicationID), core.CanonicalID(*patternID), values, hook.ReferenceRule{}, hook.ReferenceReach{}, hookHalfBefore)
			}
			if hookErr != nil {
				if hookReport != nil {
					run := referenceCommandRun{
						SchemaVersion: applicationview.SchemaVersion,
						Mode:          "reference",
						Plan:          plan,
						Hooks:         hookReport,
					}
					report.OK = false
					report.Result = run
					if *format == "json" {
						return writeViewReport(out, report, 1)
					}
					return writeReferenceRunText(out, run)
				}
				return fail("COMMAND_FAILED", hookErr.Error(), 1)
			}
			// The store is created per run and dropped at the end. It makes the change
			// something the after-* phases can be consulted about; it is not
			// persistence, and the response says so.
			referenceStoreInstance := command.NewMemoryStore()
			var storeSnapshot command.StoreSnapshot
			var executor command.Executor = command.ReferenceExecutor{World: loaded.World}
			if *referenceStore {
				executor = command.ReferenceDataExecutor{World: loaded.World, Store: referenceStoreInstance}
			}
			invoker := command.Invoker{
				Authorizer: command.ReferenceAuthorizer{},
				World:      loaded.World,
				Executor:   executor,
				UnitOfWork: &command.MemoryUnitOfWork{},
			}
			if len(allowedCapabilities) > 0 {
				authorizer := make(command.AllowListAuthorizer, 0, len(allowedCapabilities))
				for _, capability := range allowedCapabilities {
					authorizer = append(authorizer, core.CanonicalID(capability))
				}
				invoker.Authorizer = authorizer
			}
			if *periodStateFile != "" {
				states, err := periodstate.File{Path: *periodStateFile}.PeriodStates(context.Background(), *applicationID)
				if err != nil {
					return fail("PERIOD_STATE_LOAD_FAILED", err.Error(), 2)
				}
				invoker.Periods = command.NewMapPeriodState(states)
			}
			recorder := &command.MemoryRecorder{}
			invoker.Recorder = recorder
			request := command.Request{
				Application: core.CanonicalID(*applicationID),
				Pattern:     core.CanonicalID(*patternID),
				Capability:  core.CanonicalID(*commandID),
				Inputs:      values,
			}
			var result command.Result
			if *lifecycleState != "" {
				store, openErr := lifecyclesqlite.Open(*lifecycleState)
				if openErr != nil {
					return fail("LIFECYCLE_STATE_FAILED", openErr.Error(), 1)
				}
				defer store.Close()
				tracked := command.LifecycleInvoker{Invoker: invoker, Lifecycle: store}
				result, err = tracked.Invoke(context.Background(), request, command.DispatchOptions{
					CommandID:  *lifecycleCommand,
					Owner:      *lifecycleOwner,
					Generation: *lifecycleGeneration,
				})
			} else {
				result, err = invoker.Invoke(context.Background(), request)
			}
			if err != nil {
				// A reference refusal is a business answer, not an execution
				// fault: name the outcome so a caller can branch on it.
				if outcome, ok := command.FailureOutcome(err); ok {
					return fail("COMMAND_REFUSED", outcome+": "+err.Error(), 1)
				}
				if code, message, ok := commandFailure(err); ok {
					return fail(code, message, 1)
				}
				return fail("COMMAND_FAILED", err.Error(), 1)
			}
			if *referenceStore {
				// The change is applied, so the after-* phases have a point to run at.
				// Their report is joined with the before-* one so the printed run shows
				// both halves.
				storeSnapshot = referenceStoreInstance.Snapshot()
				storeSnapshot = referenceStoreInstance.Snapshot()
				var reverted bool
				var afterErr error
				hookReport, reverted, afterErr = runAfterHalfRevertingOnRefusal(context.Background(), loaded.World, core.CanonicalID(*applicationID), core.CanonicalID(*patternID), values, hook.ReferenceRule{}, referenceStoreInstance, storeSnapshot, hookReport)
				if afterErr != nil {
					run := referenceCommandRun{
						SchemaVersion:  applicationview.SchemaVersion,
						Mode:           "reference",
						Plan:           plan,
						Hooks:          hookReport,
						EffectReverted: reverted,
					}
					report.OK = false
					report.Result = run
					if *format == "json" {
						return writeViewReport(out, report, 1)
					}
					return writeReferenceRunText(out, run)
				}
			}
			if *operationLogFile != "" {
				encoded, encodeErr := json.MarshalIndent(recorder.Records, "", "  ")
				if encodeErr != nil {
					return fail("OPERATION_LOG_WRITE_FAILED", encodeErr.Error(), 1)
				}
				if writeErr := os.WriteFile(*operationLogFile, append(encoded, '\n'), 0o600); writeErr != nil {
					return fail("OPERATION_LOG_WRITE_FAILED", writeErr.Error(), 1)
				}
			}
			var differences []command.Difference
			var reportedEffectMissing bool
			// Only a declared effect gives the check something to compare. For
			// a command without one, an empty record list is "nothing to check",
			// not "the executor could not say".
			appliedUnobserved := plan.Effect != nil
			if len(recorder.Records) > 0 {
				last := recorder.Records[len(recorder.Records)-1]
				differences = last.Differences
				appliedUnobserved = last.AppliedUnobserved
				reportedEffectMissing = last.ReportedEffectMissing
			}
			run := referenceCommandRun{
				SchemaVersion:         applicationview.SchemaVersion,
				Mode:                  "reference",
				Plan:                  plan,
				Outcome:               result.Outcome,
				OutcomeUndetermined:   result.OutcomeUndetermined,
				Effect:                result.Effect,
				Committed:             true,
				Persisted:             result.Persistence == command.PersistenceResourceObserved,
				Persistence:           string(result.Persistence),
				Durability:            "none",
				Differences:           differences,
				AppliedUnobserved:     appliedUnobserved,
				ReportedEffectMissing: reportedEffectMissing,
				AuthorizationMode:     string(result.AuthorizationMode),
				Hooks:                 hookReport,
			}
			report.OK = true
			report.Result = run
			if *format == "json" {
				return writeViewReport(out, report, 0)
			}
			return writeReferenceRunText(out, run)
		}
		report.OK = true
		report.Result = plan
		if *format == "json" {
			return writeViewReport(out, report, 0)
		}
		return writeCommandPlanText(out, plan)
	}
	kind, err := applicationview.PatternKind(loaded.World, core.CanonicalID(*applicationID), core.CanonicalID(*patternID))
	if err != nil {
		return fail("VIEW_FAILED", err.Error(), 1)
	}
	var result any
	switch kind {
	case application.PatternRecord:
		view, err := applicationview.Build(loaded.World, core.CanonicalID(*applicationID), core.CanonicalID(*patternID))
		if err != nil {
			return fail("VIEW_FAILED", err.Error(), 1)
		}
		result = view
	case application.PatternHeaderDetail:
		view, err := applicationview.BuildHeaderDetail(loaded.World, core.CanonicalID(*applicationID), core.CanonicalID(*patternID))
		if err != nil {
			return fail("VIEW_FAILED", err.Error(), 1)
		}
		result = view
	case application.PatternBrowse:
		view, err := applicationview.BuildBrowse(loaded.World, core.CanonicalID(*applicationID), core.CanonicalID(*patternID))
		if err != nil {
			return fail("VIEW_FAILED", err.Error(), 1)
		}
		result = view
	case application.PatternCommand:
		if *format == "html" || *format == "print" {
			return fail("INVALID_ARGUMENT", "view -format html/print does not support command patterns", 2)
		}
		view, err := applicationview.BuildCommandPattern(loaded.World, core.CanonicalID(*applicationID), core.CanonicalID(*patternID))
		if err != nil {
			return fail("VIEW_FAILED", err.Error(), 1)
		}
		result = view
	default:
		return fail("VIEW_UNSUPPORTED", fmt.Sprintf("view does not support pattern kind %s", kind), 1)
	}
	report.OK = true
	report.Result = result
	switch *format {
	case "json":
		return writeViewReport(out, report, 0)
	case "html", "print":
		var html string
		switch value := result.(type) {
		case applicationview.View:
			if *format == "print" {
				html, err = applicationview.RenderPrintHTML(value)
			} else {
				html, err = applicationview.RenderHTML(value)
			}
		case applicationview.HeaderDetailView:
			if *format == "print" {
				html, err = applicationview.RenderHeaderDetailPrintHTML(value)
			} else {
				html, err = applicationview.RenderHeaderDetailHTML(value)
			}
		case applicationview.BrowseView:
			if *format == "print" {
				html, err = applicationview.RenderBrowsePrintHTML(value)
			} else {
				html, err = applicationview.RenderBrowseHTML(value)
			}
		}
		if err != nil {
			return fail("VIEW_FAILED", err.Error(), 1)
		}
		if _, err := fmt.Fprint(out, html); err != nil {
			return 2, err
		}
		return 0, nil
	default:
		switch value := result.(type) {
		case applicationview.View:
			return writeViewText(out, value)
		case applicationview.HeaderDetailView:
			return writeHeaderDetailText(out, value)
		case applicationview.BrowseView:
			return writeBrowseText(out, value)
		case applicationview.CommandPatternView:
			return writeCommandPatternText(out, value)
		}
		return 2, fmt.Errorf("unhandled view result %T", result)
	}
}

func writeViewReport(out io.Writer, report cliViewReport, exitCode int) (int, error) {
	encoder := json.NewEncoder(out)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(report); err != nil {
		return 2, err
	}
	return exitCode, nil
}

func writeViewText(out io.Writer, view applicationview.View) (int, error) {
	if _, err := fmt.Fprintf(out, "pattern %s (%s)\n", view.Pattern, view.Kind); err != nil {
		return 2, err
	}
	if _, err := fmt.Fprintf(out, "data %s\n", view.Data); err != nil {
		return 2, err
	}
	if len(view.Packages) > 0 {
		packages := make([]string, 0, len(view.Packages))
		for _, pkg := range view.Packages {
			packages = append(packages, fmt.Sprintf("%s ^%s", pkg.ID, pkg.Version))
		}
		if _, err := fmt.Fprintf(out, "packages: %s\n", strings.Join(packages, ", ")); err != nil {
			return 2, err
		}
	}
	if len(view.Items) == 0 {
		if _, err := fmt.Fprintln(out, "items: (none)"); err != nil {
			return 2, err
		}
	} else {
		if _, err := fmt.Fprintln(out, "items:"); err != nil {
			return 2, err
		}
		for i, item := range view.Items {
			line := fmt.Sprintf("  %d. %s : %s", i+1, item.Attribute, item.Type)
			if item.Lookup != nil {
				line += fmt.Sprintf(" lookup %s (%s) -> %s : %s",
					item.Lookup.Pattern, item.Lookup.Kind, item.Lookup.TargetAttribute, item.Lookup.TargetType)
			}
			if _, err := fmt.Fprintln(out, line); err != nil {
				return 2, err
			}
		}
	}
	if len(view.Navigations) == 0 {
		if _, err := fmt.Fprintln(out, "navigations: (none)"); err != nil {
			return 2, err
		}
	} else {
		if _, err := fmt.Fprintln(out, "navigations:"); err != nil {
			return 2, err
		}
		for _, navigation := range view.Navigations {
			if _, err := fmt.Fprintf(out, "  - %s (%s) data=%s\n", navigation.From, navigation.Kind, navigation.Data); err != nil {
				return 2, err
			}
		}
	}
	if len(view.Unused) > 0 {
		if _, err := fmt.Fprintln(out, "unused:"); err != nil {
			return 2, err
		}
		for _, item := range view.Unused {
			if _, err := fmt.Fprintf(out, "  - %s (%s)\n", item.Label, item.Type); err != nil {
				return 2, err
			}
		}
	}
	if len(view.Hooks) > 0 {
		if _, err := fmt.Fprintln(out, "hooks:"); err != nil {
			return 2, err
		}
		for _, hook := range view.Hooks {
			line := fmt.Sprintf("  - %s", hook.Phase)
			// A declared scope is printed beside the phase; an absent one means the
			// pattern's own unit and is left implicit.
			if hook.Scope != "" {
				line += fmt.Sprintf(" scope %s", hook.Scope)
			}
			if len(hook.Reads) > 0 {
				line += fmt.Sprintf(" reads %s", joinIDs(hook.Reads))
			}
			if len(hook.Uses) > 0 {
				line += fmt.Sprintf(" uses %s", joinIDs(hook.Uses))
			}
			if len(hook.Produces) > 0 {
				line += fmt.Sprintf(" produces %s", joinIDs(hook.Produces))
			}
			if hook.Failure != "" {
				line += fmt.Sprintf(" on-failure %s", hook.Failure)
			}
			if _, err := fmt.Fprintln(out, line); err != nil {
				return 2, err
			}
		}
	}
	if len(view.Commands) > 0 {
		if _, err := fmt.Fprintln(out, "commands:"); err != nil {
			return 2, err
		}
		for _, command := range view.Commands {
			if _, err := fmt.Fprintf(out, "  - %s (%s)\n", command.Label, command.Capability); err != nil {
				return 2, err
			}
			if len(command.Inputs) > 0 {
				inputs := make([]string, 0, len(command.Inputs))
				for _, input := range command.Inputs {
					inputs = append(inputs, fmt.Sprintf("%s (%s)%s", input.Name, input.Type, bindLabel(input.Label, input.Attribute)))
				}
				if _, err := fmt.Fprintf(out, "      入力: %s\n", strings.Join(inputs, ", ")); err != nil {
					return 2, err
				}
			}
			if len(command.Outcomes) > 0 {
				outcomes := make([]string, 0, len(command.Outcomes))
				for _, outcome := range command.Outcomes {
					outcomes = append(outcomes, fmt.Sprintf("%s → %s", outcome.Name, outcome.Payload))
				}
				if _, err := fmt.Fprintf(out, "      結末: %s\n", strings.Join(outcomes, ", ")); err != nil {
					return 2, err
				}
			}
			if command.Effect != nil {
				if _, err := fmt.Fprintf(out, "      効果: %s %s\n", command.Effect.Kind, command.Effect.Data); err != nil {
					return 2, err
				}
			}
		}
	}
	return 0, nil
}

func bindLabel(label string, attribute core.CanonicalID) string {
	if attribute == "" {
		return ""
	}
	return fmt.Sprintf(" ← %s", label)
}

func joinIDs(ids []core.CanonicalID) string {
	parts := make([]string, 0, len(ids))
	for _, id := range ids {
		parts = append(parts, string(id))
	}
	return strings.Join(parts, ", ")
}

func writeHeaderDetailText(out io.Writer, view applicationview.HeaderDetailView) (int, error) {
	if _, err := fmt.Fprintf(out, "pattern %s (%s)\n", view.Pattern, view.Kind); err != nil {
		return 2, err
	}
	if _, err := fmt.Fprintf(out, "relation %s — %s -> %s\n", view.Relation, view.Header, view.Detail); err != nil {
		return 2, err
	}
	if len(view.Packages) > 0 {
		packages := make([]string, 0, len(view.Packages))
		for _, pkg := range view.Packages {
			packages = append(packages, fmt.Sprintf("%s ^%s", pkg.ID, pkg.Version))
		}
		if _, err := fmt.Fprintf(out, "packages: %s\n", strings.Join(packages, ", ")); err != nil {
			return 2, err
		}
	}
	for _, section := range []struct {
		label string
		items []applicationview.Item
	}{
		{"header", view.HeaderItems},
		{"detail", view.DetailColumns},
	} {
		if _, err := fmt.Fprintf(out, "%s:\n", section.label); err != nil {
			return 2, err
		}
		for i, item := range section.items {
			line := fmt.Sprintf("  %d. %s : %s", i+1, item.Label, item.Type)
			if len(item.Values) > 0 {
				line += fmt.Sprintf(" [%s]", strings.Join(item.Values, "|"))
			}
			if _, err := fmt.Fprintln(out, line); err != nil {
				return 2, err
			}
		}
	}
	if len(view.Aggregates) > 0 {
		if _, err := fmt.Fprintln(out, "aggregates:"); err != nil {
			return 2, err
		}
		for _, aggregate := range view.Aggregates {
			if _, err := fmt.Fprintf(out, "  - %s: %s %s (%s)\n", aggregate.Name, aggregate.Function, aggregate.Label, aggregate.Type); err != nil {
				return 2, err
			}
		}
	}
	return 0, nil
}

func writeBrowseText(out io.Writer, view applicationview.BrowseView) (int, error) {
	if _, err := fmt.Fprintf(out, "pattern %s (%s)\n", view.Pattern, view.Kind); err != nil {
		return 2, err
	}
	if _, err := fmt.Fprintf(out, "data %s — query %s\n", view.Data, view.Query); err != nil {
		return 2, err
	}
	if len(view.Packages) > 0 {
		packages := make([]string, 0, len(view.Packages))
		for _, pkg := range view.Packages {
			packages = append(packages, fmt.Sprintf("%s ^%s", pkg.ID, pkg.Version))
		}
		if _, err := fmt.Fprintf(out, "packages: %s\n", strings.Join(packages, ", ")); err != nil {
			return 2, err
		}
	}
	if view.SemanticFingerprint != "" {
		if _, err := fmt.Fprintf(out, "semantics: %s\n", view.SemanticFingerprint); err != nil {
			return 2, err
		}
	}
	if view.PageSize > 0 {
		if _, err := fmt.Fprintf(out, "page-size: %d\\n", view.PageSize); err != nil {
			return 2, err
		}
	}
	if len(view.Sorts) > 0 {
		if _, err := fmt.Fprintln(out, "sorts:"); err != nil {
			return 2, err
		}
		for _, sort := range view.Sorts {
			if _, err := fmt.Fprintf(out, "  - %s %s\\n", sort.Label, sort.Direction); err != nil {
				return 2, err
			}
		}
	}
	if len(view.Filters) > 0 {
		if _, err := fmt.Fprintln(out, "filters:"); err != nil {
			return 2, err
		}
		for _, filter := range view.Filters {
			if _, err := fmt.Fprintf(out, "  - %s: %s\n", filter.Label, strings.Join(filter.Operators, ", ")); err != nil {
				return 2, err
			}
		}
	}
	if _, err := fmt.Fprintln(out, "columns:"); err != nil {
		return 2, err
	}
	for i, column := range view.Columns {
		if _, err := fmt.Fprintf(out, "  %d. %s : %s\n", i+1, column.Label, column.Type); err != nil {
			return 2, err
		}
	}
	return 0, nil
}

type inputFlag []string

func (f *inputFlag) String() string { return strings.Join(*f, ",") }

func (f *inputFlag) Set(value string) error {
	*f = append(*f, value)
	return nil
}

func parseInputPairs(pairs inputFlag) (map[string]string, error) {
	values := map[string]string{}
	for _, pair := range pairs {
		name, value, ok := strings.Cut(pair, "=")
		if !ok || name == "" {
			return nil, fmt.Errorf("-input must be name=value")
		}
		if _, exists := values[name]; exists {
			return nil, fmt.Errorf("-input %s is given more than once", name)
		}
		values[name] = value
	}
	return values, nil
}

func writeApplicationText(out io.Writer, view applicationview.ApplicationView) (int, error) {
	if _, err := fmt.Fprintf(out, "application %s\n", view.Application); err != nil {
		return 2, err
	}
	for _, period := range view.Periods {
		if _, err := fmt.Fprintf(out, "period %s data %s states %s\n", period.ID, period.Data, strings.Join(period.States, ",")); err != nil {
			return 2, err
		}
		// A period that declares a calendar names the rule its boundary follows;
		// printing it beside the period keeps that visible without a default.
		if period.Calendar != "" {
			if _, err := fmt.Fprintf(out, "period %s calendar %s\n", period.ID, period.Calendar); err != nil {
				return 2, err
			}
		}
		// A closing day splits the month into two segments; printing it shows the
		// period is narrower than a month without the reader inferring a default.
		if period.ClosingDay != 0 {
			if _, err := fmt.Fprintf(out, "period %s closes on day %d\n", period.ID, period.ClosingDay); err != nil {
				return 2, err
			}
		}
	}
	for _, use := range view.SemanticUses {
		if _, err := fmt.Fprintf(out, "uses %s\n", use); err != nil {
			return 2, err
		}
	}
	// Standard business Data is shown with the attributes its binder reads, so
	// a reader can see which Data the roles are on without opening the files.
	for _, ref := range view.StandardBusiness {
		if _, err := fmt.Fprintf(out, "standard-business %s %s (%s)\n", ref.Kind, ref.Data, strings.Join(canonicalIDs(ref.Attributes), " ")); err != nil {
			return 2, err
		}
	}
	for _, pattern := range view.Patterns {
		if _, err := fmt.Fprintf(out, "pattern %s (%s)\n", pattern.ID, pattern.Kind); err != nil {
			return 2, err
		}
	}
	for _, pkg := range view.Packages {
		if _, err := fmt.Fprintf(out, "package %s ^%s (bound: %t)\n", pkg.ID, pkg.Version, pkg.Bound); err != nil {
			return 2, err
		}
	}
	if _, err := fmt.Fprintf(out, "semantics %s\n", view.SemanticFingerprint); err != nil {
		return 2, err
	}
	return 0, nil
}

func canonicalIDs(ids []core.CanonicalID) []string {
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		out = append(out, string(id))
	}
	return out
}

func writePackageText(out io.Writer, view applicationview.PackageView) (int, error) {
	if _, err := fmt.Fprintf(out, "package %s ^%s (bound: %t)\n", view.Package, view.Version, view.Bound); err != nil {
		return 2, err
	}
	if _, err := fmt.Fprintf(out, "application %s\n", view.Application); err != nil {
		return 2, err
	}
	for _, member := range view.Members {
		if _, err := fmt.Fprintf(out, "member %s %s\n", member.Kind, member.Pattern); err != nil {
			return 2, err
		}
	}
	for _, requirement := range view.Requires {
		if _, err := fmt.Fprintf(out, "requires %s %s\n", requirement.Kind, requirement.ID); err != nil {
			return 2, err
		}
	}
	if view.Projection != "" {
		if _, err := fmt.Fprintf(out, "projection %s\n", view.Projection); err != nil {
			return 2, err
		}
	}
	if view.Conformance != "" {
		if _, err := fmt.Fprintf(out, "conformance %s\n", view.Conformance); err != nil {
			return 2, err
		}
	}
	return 0, nil
}

func writeCommandPatternText(out io.Writer, view applicationview.CommandPatternView) (int, error) {
	if _, err := fmt.Fprintf(out, "command %s (%s)\n", view.Capability, view.Label); err != nil {
		return 2, err
	}
	for _, input := range view.Inputs {
		if _, err := fmt.Fprintf(out, "input %s (%s)\n", input.Name, input.Type); err != nil {
			return 2, err
		}
	}
	for _, outcome := range view.Outcomes {
		if _, err := fmt.Fprintf(out, "outcome %s -> %s\n", outcome.Name, outcome.Payload); err != nil {
			return 2, err
		}
	}
	if view.Effect != nil {
		if _, err := fmt.Fprintf(out, "effect: %s %s\n", view.Effect.Kind, view.Effect.Data); err != nil {
			return 2, err
		}
	}
	return 0, nil
}

// writeHookReportText prints which declared hooks ran, which were skipped, and
// why. A skipped phase is printed with its reason so the absence of a phase is
// not read as a pass.
// writeProposalText prints the values a phase proposed, in attribute order so
// the same proposal always prints the same way.
func writeProposalText(out io.Writer, proposed hook.Produced) (int, error) {
	if len(proposed) == 0 {
		return 0, nil
	}
	attributes := make([]string, 0, len(proposed))
	for attribute := range proposed {
		attributes = append(attributes, string(attribute))
	}
	sort.Strings(attributes)
	for _, attribute := range attributes {
		if _, err := fmt.Fprintf(out, "    proposes %s = %s (candidate, not adopted)\n", attribute, proposed[core.CanonicalID(attribute)]); err != nil {
			return 2, err
		}
	}
	return 0, nil
}

func writeHookReportText(out io.Writer, report cliHookReport) (int, error) {
	if _, err := fmt.Fprintf(out, "hooks: %s (%d executed, %d warned, %d skipped)\n", report.Pattern, len(report.Executed), len(report.Warned), len(report.Skipped)); err != nil {
		return 2, err
	}
	for _, phase := range report.Executed {
		if _, err := fmt.Fprintf(out, "  hook %s: executed\n", phase); err != nil {
			return 2, err
		}
		// A proposal is a candidate value, not a settled one. Printing it beside
		// the phase keeps a reader from treating `executed` as "the hook changed
		// nothing" when it in fact proposed a value the command did not adopt.
		if code, err := writeProposalText(out, report.Proposed(phase)); err != nil {
			return code, err
		}
	}
	for _, phase := range report.Warned {
		if _, err := fmt.Fprintf(out, "  hook %s: warned (on-failure warn)\n", phase); err != nil {
			return 2, err
		}
	}
	for _, phase := range report.NotEvaluated {
		if _, err := fmt.Fprintf(out, "  hook %s: not evaluated (%s)\n", phase, report.Reasons[phase]); err != nil {
			return 2, err
		}
		// A declared `uses` that was never called is the difference between
		// "the hook passed" and "the hook's call did not happen". Print it.
		if uncalled := report.Uncalled[phase]; len(uncalled) > 0 {
			for _, capability := range uncalled {
				if _, err := fmt.Fprintf(out, "    uncalled %s\n", capability); err != nil {
					return 2, err
				}
			}
		}
	}
	for _, phase := range report.Skipped {
		if hookReportHolds(report.NotEvaluated, phase) {
			continue
		}
		if _, err := fmt.Fprintf(out, "  hook %s: skipped (%s)\n", phase, report.Reasons[phase]); err != nil {
			return 2, err
		}
	}
	if report.Refused != "" {
		if _, err := fmt.Fprintf(out, "  hook %s: refused", report.Refused); err != nil {
			return 2, err
		}
		if report.CommandOutcome != "" {
			// Name both: the hook's own outcome and the Command name it
			// corresponds to, so the correspondence is visible rather than
			// silently applied.
			if _, err := fmt.Fprintf(out, " (%s, command outcome %s)", report.Outcome, report.CommandOutcome); err != nil {
				return 2, err
			}
		} else if report.Outcome != "" {
			if _, err := fmt.Fprintf(out, " (%s)", report.Outcome); err != nil {
				return 2, err
			}
		}
		if _, err := fmt.Fprintln(out); err != nil {
			return 2, err
		}
	}
	return 0, nil
}

func writeReferenceRunText(out io.Writer, run referenceCommandRun) (int, error) {
	if _, err := fmt.Fprintln(out, "mode: reference (no durable effects)"); err != nil {
		return 2, err
	}
	if run.AuthorizationMode != "" {
		if _, err := fmt.Fprintf(out, "authorization: %s\n", run.AuthorizationMode); err != nil {
			return 2, err
		}
	}
	if code, err := writeCommandPlanText(out, run.Plan); err != nil {
		return code, err
	}
	if run.EffectReverted {
		// The change was applied and then put back. Say so where the run is
		// printed, so a reader never takes the reported state for a change that
		// survived the after-change phases.
		if _, err := fmt.Fprintln(out, "effect: reverted (an after-change phase refused after the effect was applied)"); err != nil {
			return 2, err
		}
	}
	if run.OutcomeUndetermined != "" {
		// An outcome nobody determined must not read as "the capability has no
		// outcome". Print the reason where the outcome would have been.
		if _, err := fmt.Fprintf(out, "outcome: not determined (%s)\n", run.OutcomeUndetermined); err != nil {
			return 2, err
		}
	}
	if run.Outcome != nil {
		if _, err := fmt.Fprintf(out, "outcome: %s -> %s\n", run.Outcome.Name, run.Outcome.Payload); err != nil {
			return 2, err
		}
	}
	if run.Effect != nil {
		if _, err := fmt.Fprintf(out, "effect: %s %s\n", run.Effect.Kind, run.Effect.Data); err != nil {
			return 2, err
		}
	}
	if run.Hooks != nil {
		if code, err := writeHookReportText(out, *run.Hooks); err != nil {
			return code, err
		}
	}
	if _, err := fmt.Fprintf(out, "committed: %t (reference unit of work)\n", run.Committed); err != nil {
		return 2, err
	}
	if _, err := fmt.Fprintf(out, "persisted: %t (persistence: %s, durability: %s)\n", run.Persisted, run.Persistence, run.Durability); err != nil {
		return 2, err
	}
	// An unresolved difference is shown even when the run succeeded: the
	// declaration did not describe the whole effect, and the reader should not
	// have to open the operation log to notice.
	// Only a declared effect gives the check something to compare against; a
	// command without one has no difference to look for.
	if run.ReportedEffectMissing {
		if _, err := fmt.Fprintln(out, "reported-effect: missing (the capability declares an effect but the executor reported none)"); err != nil {
			return 2, err
		}
	}
	if run.AppliedUnobserved && run.Plan.Effect != nil {
		if _, err := fmt.Fprintln(out, "difference-check: not observed (this executor does not enumerate its operations)"); err != nil {
			return 2, err
		}
	}
	for _, difference := range run.Differences {
		if _, err := fmt.Fprintf(out, "difference: %s (%s %s beyond %s %s)\n",
			difference.Code, difference.Applied.Kind, difference.Applied.Data,
			difference.Declared.Kind, difference.Declared.Data); err != nil {
			return 2, err
		}
	}
	return 0, nil
}

func writeCommandPlanText(out io.Writer, plan applicationview.CommandPlan) (int, error) {
	if _, err := fmt.Fprintf(out, "command %s (%s)\n", plan.Capability, plan.Label); err != nil {
		return 2, err
	}
	if _, err := fmt.Fprintln(out, "inputs:"); err != nil {
		return 2, err
	}
	for _, input := range plan.Inputs {
		line := fmt.Sprintf("  - %s (%s)", input.Name, input.Type)
		if input.Attribute != "" {
			line += fmt.Sprintf(" ← %s", input.Label)
		}
		if input.Provided {
			line += fmt.Sprintf(" = %s", input.Value)
		} else {
			line += " (missing)"
		}
		if _, err := fmt.Fprintln(out, line); err != nil {
			return 2, err
		}
	}
	for _, outcome := range plan.Outcomes {
		if _, err := fmt.Fprintf(out, "outcome %s -> %s\n", outcome.Name, outcome.Payload); err != nil {
			return 2, err
		}
	}
	if plan.Effect != nil {
		if _, err := fmt.Fprintf(out, "effect: %s %s\n", plan.Effect.Kind, plan.Effect.Data); err != nil {
			return 2, err
		}
	}
	if plan.Period != nil {
		if _, err := fmt.Fprintf(out, "period: %s states=%s closed-rejects=%s\n", plan.Period.ID, strings.Join(plan.Period.States, ","), strings.Join(plan.Period.ClosedRejects, ",")); err != nil {
			return 2, err
		}
	}
	if len(plan.Invalid) > 0 {
		if _, err := fmt.Fprintln(out, "invalid:"); err != nil {
			return 2, err
		}
		for _, issue := range plan.Invalid {
			if _, err := fmt.Fprintf(out, "  - %s: %s\n", issue.Input, issue.Message); err != nil {
				return 2, err
			}
		}
	}
	if _, err := fmt.Fprintf(out, "ready: %t\n", plan.Ready); err != nil {
		return 2, err
	}
	if len(plan.Missing) > 0 {
		if _, err := fmt.Fprintf(out, "missing: %s\n", strings.Join(plan.Missing, ", ")); err != nil {
			return 2, err
		}
	}
	return 0, nil
}
