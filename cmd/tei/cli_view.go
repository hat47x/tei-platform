package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"strings"

	"tei/internal/applicationview"
	"tei/internal/core"
	"tei/internal/workspace"
)

type cliViewReport struct {
	SchemaVersion string                `json:"schemaVersion"`
	OK            bool                  `json:"ok"`
	Result        *applicationview.View `json:"result,omitempty"`
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
	format := fs.String("format", "text", "text or json")
	if err := fs.Parse(args); err != nil {
		return fail("INVALID_ARGUMENT", err.Error(), 2)
	}
	if fs.NArg() != 0 || *root == "" || *applicationID == "" || *patternID == "" ||
		*root != strings.TrimSpace(*root) || *applicationID != strings.TrimSpace(*applicationID) ||
		*patternID != strings.TrimSpace(*patternID) {
		return fail("INVALID_ARGUMENT", "view requires exact -workspace, -application and -pattern, without positional arguments", 2)
	}
	if *format != "text" && *format != "json" {
		return fail("INVALID_ARGUMENT", "view -format must be text or json", 2)
	}

	loaded, err := workspace.Load(*root)
	if err != nil {
		return fail("WORKSPACE_LOAD_FAILED", err.Error(), 2)
	}
	view, err := applicationview.Build(loaded.World, core.CanonicalID(*applicationID), core.CanonicalID(*patternID))
	if err != nil {
		return fail("VIEW_FAILED", err.Error(), 1)
	}
	report.OK = true
	report.Result = &view
	if *format == "json" {
		return writeViewReport(out, report, 0)
	}
	return writeViewText(out, view)
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
		return 0, nil
	}
	if _, err := fmt.Fprintln(out, "navigations:"); err != nil {
		return 2, err
	}
	for _, navigation := range view.Navigations {
		if _, err := fmt.Fprintf(out, "  - %s (%s) data=%s\n", navigation.From, navigation.Kind, navigation.Data); err != nil {
			return 2, err
		}
	}
	return 0, nil
}
