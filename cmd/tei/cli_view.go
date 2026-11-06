package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"tei/internal/application"
	"tei/internal/applicationview"
	"tei/internal/command"
	"tei/internal/command/lifecyclesqlite"
	"tei/internal/core"
	"tei/internal/periodstate"
	"tei/internal/semantic"
	"tei/internal/workspace"
)

type referenceCommandRun struct {
	SchemaVersion string                      `json:"schemaVersion"`
	Mode          string                      `json:"mode"`
	Plan          applicationview.CommandPlan `json:"plan"`
	Outcome       *semantic.Outcome           `json:"outcome,omitempty"`
	Effect        *semantic.DataEffect        `json:"effect,omitempty"`
	Committed     bool                        `json:"committed"`
}

type cliViewReport struct {
	SchemaVersion string                `json:"schemaVersion"`
	OK            bool                  `json:"ok"`
	Result        any                   `json:"result,omitempty"`
	Error         *cliInspectionFailure `json:"error,omitempty"`
}

// runViewCommandは、Application Patternの対話契約を解決し、textまたはjsonで返す。
// 意味の解決は applicationview に置き、ここでは引数・出力・終了コードだけを扱う。
// commandFailure maps an execution failure to the CLI code a caller needs in
// order to decide what to do next. A failure whose data effect may already be
// applied must not read like a clean rejection, and a lifecycle refusal is a
// replay decision rather than an execution error.
func commandFailure(err error) (string, string, bool) {
	var execution *command.ExecutionError
	if errors.As(err, &execution) && execution.Code == "COMMAND_LIFECYCLE_REFUSED" {
		return "COMMAND_LIFECYCLE_REFUSED", execution.Message, true
	}
	if command.EffectMayBeUnknown(err) {
		return "COMMAND_EFFECT_UNKNOWN", err.Error(), true
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
	periodStateFile := fs.String("period-state", "", "period state JSON for -execute")
	operationLogFile := fs.String("operation-log", "", "write reference operation records as JSON")
	lifecycleState := fs.String("lifecycle-state", "", "SQLite state file for restart-safe command lifecycle")
	lifecycleCommand := fs.String("lifecycle-command", "", "stable command id recorded in the lifecycle state")
	lifecycleOwner := fs.String("lifecycle-owner", "cli", "owner recorded in the lifecycle state")
	lifecycleGeneration := fs.Int("lifecycle-generation", 1, "generation recorded in the lifecycle state")
	var allowedCapabilities inputFlag
	fs.Var(&allowedCapabilities, "allow-capability", "capability allowed by the reference authorizer; repeatable")
	var inputPairs inputFlag
	fs.Var(&inputPairs, "input", "command input as name=value; repeatable")
	format := fs.String("format", "text", "text or json")
	if err := fs.Parse(args); err != nil {
		return fail("INVALID_ARGUMENT", err.Error(), 2)
	}
	if fs.NArg() != 0 || *root == "" || *applicationID == "" ||
		(*patternID != "" && *packageID != "") ||
		(*commandID != "" && *patternID == "") ||
		(*execute && *commandID == "") ||
		(*periodStateFile != "" && !*execute) ||
		(*lifecycleState != "" && (!*execute || *lifecycleCommand == "")) ||
		(*lifecycleState == "" && *lifecycleCommand != "") ||
		(*lifecycleGeneration < 1) ||
		(len(allowedCapabilities) > 0 && !*execute) ||
		(*operationLogFile != "" && !*execute) ||
		*root != strings.TrimSpace(*root) || *applicationID != strings.TrimSpace(*applicationID) ||
		*patternID != strings.TrimSpace(*patternID) || *packageID != strings.TrimSpace(*packageID) ||
		*commandID != strings.TrimSpace(*commandID) {
		return fail("INVALID_ARGUMENT", "view requires exact -workspace and -application, at most one of -pattern, -package, or -command, without positional arguments", 2)
	}
	if *execute && *format != "text" && *format != "json" {
		return fail("INVALID_ARGUMENT", "view -execute supports -format text or json", 2)
	}
	if *format != "text" && *format != "json" && *format != "html" && *format != "print" {
		return fail("INVALID_ARGUMENT", "view -format must be text, json, html, or print", 2)
	}

	loaded, err := workspace.Load(*root)
	if err != nil {
		return fail("WORKSPACE_LOAD_FAILED", err.Error(), 2)
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
			invoker := command.Invoker{
				Authorizer: command.ReferenceAuthorizer{},
				World:      loaded.World,
				Executor:   command.ReferenceExecutor{World: loaded.World},
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
			var recorder *command.MemoryRecorder
			if *operationLogFile != "" {
				recorder = &command.MemoryRecorder{}
				invoker.Recorder = recorder
			}
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
				if code, message, ok := commandFailure(err); ok {
					return fail(code, message, 1)
				}
				return fail("COMMAND_FAILED", err.Error(), 1)
			}
			if recorder != nil {
				encoded, encodeErr := json.MarshalIndent(recorder.Records, "", "  ")
				if encodeErr != nil {
					return fail("OPERATION_LOG_WRITE_FAILED", encodeErr.Error(), 1)
				}
				if writeErr := os.WriteFile(*operationLogFile, append(encoded, '\n'), 0o600); writeErr != nil {
					return fail("OPERATION_LOG_WRITE_FAILED", writeErr.Error(), 1)
				}
			}
			run := referenceCommandRun{
				SchemaVersion: applicationview.SchemaVersion,
				Mode:          "reference",
				Plan:          plan,
				Outcome:       result.Outcome,
				Effect:        result.Effect,
				Committed:     true,
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
	}
	for _, use := range view.SemanticUses {
		if _, err := fmt.Fprintf(out, "uses %s\n", use); err != nil {
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

func writeReferenceRunText(out io.Writer, run referenceCommandRun) (int, error) {
	if _, err := fmt.Fprintln(out, "mode: reference (no durable effects)"); err != nil {
		return 2, err
	}
	if code, err := writeCommandPlanText(out, run.Plan); err != nil {
		return code, err
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
	if _, err := fmt.Fprintf(out, "committed: %t\n", run.Committed); err != nil {
		return 2, err
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
