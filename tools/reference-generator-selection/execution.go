package generatorselection

import (
	"context"
	"os/exec"
)

// ExecutionOptions prefixes candidate execution with an explicit executable and
// argv. Compilation and interpreter discovery are outside this boundary.
// The caller must establish what the wrapper enforces; a prefix grants no authority.
type ExecutionOptions struct {
	CommandPrefix []string `json:"commandPrefix,omitempty"`
	// MinimalEnvironment runs the command with a short environment; see
	// withMinimalEnvironment.
	MinimalEnvironment bool `json:"-"`
}

func (options ExecutionOptions) Command(ctx context.Context, name string, args ...string) *exec.Cmd {
	if len(options.CommandPrefix) == 0 {
		return exec.CommandContext(ctx, name, args...)
	}
	wrapped := append([]string(nil), options.CommandPrefix[1:]...)
	wrapped = append(wrapped, name)
	wrapped = append(wrapped, args...)
	return exec.CommandContext(ctx, options.CommandPrefix[0], wrapped...)
}

// withMinimalEnvironment asks the next command to start with a short
// environment instead of the host's. The Go wasm runtime copies every variable
// into the instance, and an inherited block from a CI runner overflows that
// buffer (#3971).
func (options ExecutionOptions) withMinimalEnvironment() ExecutionOptions {
	options.MinimalEnvironment = true
	return options
}
