package directexecution

import "io"

// Options configure a Direct execution.
type Options struct {
	Command         string
	Args            []string
	InputPath       string
	OutputPath      string
	WorkDir         string
	Env             []string
	Stdin           io.Reader
	FunctionID      string
	SourceRevision  string
	Toolchain       string
}
