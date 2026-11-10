package main

import (
	"flag"
	"fmt"
)

// emptyStringFlagsNotAllowed lists, per command, the string flags whose empty
// value is a mistake rather than "the caller omitted the flag".
//
// The list is deliberately narrow. Many flags default to "" and mean "no input"
// when the caller leaves them out; their empty value is the same as omitted, so
// rejecting it would add noise without changing behaviour. The flags below are
// different: each has a nonempty default or a nonempty-only meaning, and an
// empty value silently selects a third outcome that matches neither the
// advertised default nor "absent" (#3605). The MCP reference adapter already
// rejects these as empty strings, so listing them here is what makes the CLI
// and MCP answer the same way.
var emptyStringFlagsNotAllowed = map[string][]string{
	"trace":          {"mode", "direction", "expected-fingerprint"},
	"model catalog":  {"namespace"},
	"model locate":   {"workspace", "subject-kind", "subject-id"},
	"model overview": {"workspace"},
	"inspect":        {"subject-kind"},
}

// emptyStringFlag returns the first flag in names that the caller passed with
// an empty value, or "" when every one of them carries a value. A flag the
// caller left out keeps its declared default and is never reported.
func emptyStringFlag(fs *flag.FlagSet, names []string) string {
	passed := map[string]bool{}
	fs.Visit(func(f *flag.Flag) { passed[f.Name] = true })
	for _, name := range names {
		if !passed[name] {
			continue
		}
		f := fs.Lookup(name)
		if f == nil {
			continue
		}
		getter, ok := f.Value.(flag.Getter)
		if !ok {
			continue
		}
		if value, ok := getter.Get().(string); ok && value == "" {
			return name
		}
	}
	return ""
}

// invalidEmptyStringArgument rejects an explicitly empty value for the flags
// this command cannot interpret. Returning nil means the caller either omitted
// the flag or gave it a value.
func invalidEmptyStringArgument(fs *flag.FlagSet, command string) error {
	name := emptyStringFlag(fs, emptyStringFlagsNotAllowed[command])
	if name == "" {
		return nil
	}
	return fmt.Errorf("%s: -%s must not be an empty string; omit the flag to use its default", command, name)
}
