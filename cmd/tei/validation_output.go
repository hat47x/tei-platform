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
		// The validation load collects independent syntax findings instead of
		// stopping at the first one, so every collected finding is printed. The
		// JSON form already reports all of them with `incomplete`; printing only
		// Direct read as "there is one problem" (#4178).
		directs := append([]workspace.SourceDiagnostic{*result.Direct}, result.AdditionalDirect...)
		for _, direct := range directs {
			printSourceDiagnostic(out, direct)
		}
		if result.InspectionIncomplete {
			// Collected run: the findings above are every file that failed to
			// parse. What stays unverified is syntax coverage beyond those files
			// (imports and sources depending on the failed load) and semantic
			// validation, which never ran because no World loaded. Saying "other
			// files were not examined" here would be false, because they were.
			fmt.Fprintf(out, "%s\t%s\t%s\n",
				"note", "VALIDATION_INCOMPLETE",
				fmt.Sprintf("%d件の構文の指摘を同じ読み込みで集めました。構文検査の網羅は未確認で、意味検証は実行していません",
					len(directs)))
		} else {
			// Fail-fast projection: this run really did stop at the one file, so
			// the original sentence stays true.
			fmt.Fprintf(out, "%s\t%s\t%s\n",
				"note", "VALIDATION_INCOMPLETE",
				fmt.Sprintf("the run stopped at the first syntax error in %s; files other than that one were not examined",
					result.Direct.Source.ArtifactPath))
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
