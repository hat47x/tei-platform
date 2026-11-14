package directexecution

import "io"

// Options configure a Direct execution.
type Options struct {
	Command    string
	Args       []string
	OutputPath string
	// ExpectedSHA256 is an optional caller-supplied lowercase hex digest.
	ExpectedSHA256 string
	WorkDir        string
	Env            []string
	Stdin          io.Reader
	// Reserved for future adapters. Run does not read, bind, or verify these.
	InputPath      string
	FunctionID     string
	SourceRevision string
	Toolchain      string
}
