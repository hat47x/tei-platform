package main

import (
	"fmt"
	"io"

	"tei/internal/operations"
	"tei/internal/validation"
	"tei/internal/workspace"
)

func validateWorkspace(out io.Writer, root string) (int, error) {
	result, err := operations.ValidateWorkspace(root)
	if err != nil {
		return 2, err
	}
	if result.Direct != nil {
		printSourceDiagnostic(out, *result.Direct)
		for _, additional := range result.AdditionalDirect {
			printSourceDiagnostic(out, additional)
		}
		// The JSON form carries this distinction in `incomplete`. The text form
		// must say the same thing, and must not read as "there is one problem"
		// (#4178).
		if result.InspectionIncomplete {
			if len(result.AdditionalDirect) == 0 {
				fmt.Fprintf(out, "%s\t%s\t%s\n",
					"note", "VALIDATION_INCOMPLETE",
					fmt.Sprintf("the run stopped at the first syntax error in %s; files other than that one were not examined",
						result.Direct.Source.ArtifactPath))
			} else {
				fmt.Fprintf(out, "%s\t%s\t%s\n",
					"note", "VALIDATION_INCOMPLETE",
					fmt.Sprintf("\u610f\u5473\u691c\u8a3c\u306f\u5b9f\u884c\u3057\u3066\u3044\u307e\u305b\u3093: %d files have syntax errors and the World is partial",
						len(result.AdditionalDirect)+1))
			}
		}
		return 1, nil
	}
	hasSourceRootError := false
	for _, finding := range result.SourceRootFindings {
		fmt.Fprintf(out, "%s\tWORKSPACE_SOURCE_ROOT\t%s\n", finding.Severity, finding.Message)
		if finding.Severity == "error" {
			hasSourceRootError = true
		}
	}
	if printProjectedDiagnostics(out, result.Diagnostics) {
		return 1, nil
	}
	if hasSourceRootError {
		return 1, nil
	}
	fmt.Fprintln(out, "ok")
	return 0, nil
}

func printProjectedDiagnostics(out io.Writer, diagnostics []workspace.ProjectedDiagnostic) bool {
	hasError := false
	for _, projected := range diagnostics {
		diagnostic := projected.Diagnostic
		// Duplicate candidates are a set, not an indication that the first
		// source is the declaration to reject. Text output has only one prefix.
		if len(projected.Sources) > 0 && diagnostic.Code != "DUPLICATE_SEMANTIC_ELEMENT" {
			source := projected.Sources[0]
			fmt.Fprintf(
				out,
				"%s:%d:%d\t%s\t%s\t%s\n",
				source.ArtifactPath,
				source.Range.Start.Line,
				source.Range.Start.Column,
				diagnostic.Severity,
				diagnostic.Code,
				diagnostic.Message,
			)
		} else {
			fmt.Fprintf(out, "%s\t%s\t%s\n", diagnostic.Severity, diagnostic.Code, diagnostic.Message)
		}
		if diagnostic.Severity == validation.Error {
			hasError = true
		}
	}
	return hasError
}

func printSourceDiagnostic(out io.Writer, diagnostic workspace.SourceDiagnostic) {
	fmt.Fprintf(
		out,
		"%s\t%s\t%s\t%s\n",
		fmt.Sprintf("%s:%d:%d", diagnostic.Source.ArtifactPath, diagnostic.Source.Range.Start.Line, diagnostic.Source.Range.Start.Column),
		diagnostic.Severity,
		diagnostic.Code,
		diagnostic.Message,
	)
}
