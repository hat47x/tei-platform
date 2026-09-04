package main

import (
	"fmt"
	"io"

	"tei/internal/validation"
	"tei/internal/workspace"
)

func validateWorkspace(out io.Writer, root string) (int, error) {
	result, err := workspace.Load(root)
	if err != nil {
		if diagnostic, ok := workspace.SourceDiagnosticFromError(root, err); ok {
			printSourceDiagnostic(out, diagnostic)
			return 1, nil
		}
		return 2, err
	}

	index, err := workspace.BuildSourceIndex(root, result)
	if err != nil {
		return 2, err
	}
	projected := workspace.ProjectDiagnostics(validation.Validate(result.World), index)
	if printProjectedDiagnostics(out, projected) {
		return 1, nil
	}
	fmt.Fprintln(out, "ok")
	return 0, nil
}

func printProjectedDiagnostics(out io.Writer, diagnostics []workspace.ProjectedDiagnostic) bool {
	hasError := false
	for _, projected := range diagnostics {
		diagnostic := projected.Diagnostic
		if len(projected.Sources) > 0 {
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
