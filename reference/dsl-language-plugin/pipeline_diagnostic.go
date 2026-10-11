package languagereference

import "fmt"

// PipelineDiagnostic is a reference-only source-located diagnostic.
// Span offsets are UTF-8 byte positions, with an exclusive end.
// Step is -1 for input/pipeline errors or the zero-based operator index.
// It does not replace the shared Syntax and Source Index diagnostic ABI.
type PipelineDiagnostic struct {
	Span  PipelineSpan
	Step  int
	Cause error
}

func (e *PipelineDiagnostic) Error() string {
	if e.Step < 0 {
		return fmt.Sprintf("pipeline bytes %d:%d: %v", e.Span.Start, e.Span.End, e.Cause)
	}
	return fmt.Sprintf("pipeline step %d bytes %d:%d: %v", e.Step, e.Span.Start, e.Span.End, e.Cause)
}

func (e *PipelineDiagnostic) Unwrap() error {
	return e.Cause
}
