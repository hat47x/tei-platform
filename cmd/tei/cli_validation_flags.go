package main

import (
	"flag"
	"fmt"
	"strconv"
	"strings"
)

// wantsJSONValidation selects the structured failure path before FlagSet.Parse.
// Otherwise an unknown flag or a malformed value terminates the process with
// a stderr-only diagnostic, even when the caller requested -json.
func wantsJSONValidation(args []string) bool {
	requested := false
	for _, arg := range args {
		switch {
		case arg == "-json", arg == "--json":
			requested = true
		case strings.HasPrefix(arg, "-json="), strings.HasPrefix(arg, "--json="):
			_, value, _ := strings.Cut(arg, "=")
			enabled, err := strconv.ParseBool(value)
			requested = err != nil || enabled
		}
	}
	return requested
}

// The shared FlagSet defines flags for many unrelated commands. Reject their
// use in JSON validation rather than silently ignoring them.
func validateJSONValidationFlags(fs *flag.FlagSet) error {
	var invalid error
	fs.Visit(func(option *flag.Flag) {
		switch option.Name {
		case "workspace", "json":
			// Supported by the JSON validation adapter.
		default:
			if invalid == nil {
				invalid = fmt.Errorf("validate -json does not accept -%s", option.Name)
			}
		}
	})
	return invalid
}
