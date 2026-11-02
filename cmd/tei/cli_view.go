package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"strings"

	"tei/internal/application"
	"tei/internal/applicationview"
	"tei/internal/core"
	"tei/internal/workspace"
)

type cliViewReport struct {
	SchemaVersion string                `json:"schemaVersion"`
	OK            bool                  `json:"ok"`
	Result        any                   `json:"result,omitempty"`
	Error         *cliInspectionFailure `json:"error,omitempty"`
}

// runViewCommandは、Application Patternの対話契約を解決し、textまたはjsonで返す。
// 意味の解決は applicationview に置き、ここでは引数・出力・終了コードだけを扱う。
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
	commandID := fs.String("command", "", "command capability to plan")
	var inputPairs inputFlag
	fs.Var(&inputPairs, "input", "command input as name=value; repeatable")
	format := fs.String("format", "text", "text or json")
	if err := fs.Parse(args); err != nil {
		return fail("INVALID_ARGUMENT", err.Error(), 2)
	}
	if fs.NArg() != 0 || *root == "" || *applicationID == "" || *patternID == "" ||
		*root != strings.TrimSpace(*root) || *applicationID != strings.TrimSpace(*applicationID) ||
		*patternID != strings.TrimSpace(*patternID) {
		return fail("INVALID_ARGUMENT", "view requires exact -workspace, -application and -pattern, without positional arguments", 2)
	}
	if *format != "text" && *format != "json" && *format != "html" && *format != "print" {
		return fail("INVALID_ARGUMENT", "view -format must be text, json, html, or print", 2)
	}

	loaded, err := workspace.Load(*root)
	if err != nil {
		return fail("WORKSPACE_LOAD_FAILED", err.Error(), 2)
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
