package main

import (
	"strings"
	"testing"
)

// TestCatalogueFlagsAreFlagsTheCommandAccepts pins #3956: the catalogue described
// `validate` with `repeatableFlags: [-analysis, -schema, -plan, -evidence]`, but
// validate refuses all four (text mode and -json mode alike). A reader planning
// work from `tei commands` was told about a grammar that does not exist.
//
// Every flag the catalogue names for a command must be a flag that command
// reads, so the advertisement and the argument guard cannot drift apart.
func TestCatalogueFlagsAreFlagsTheCommandAccepts(t *testing.T) {
	catalog := commandCatalog()
	for _, record := range catalog.Commands {
		allowed, ok := selectedReadOnlyFlagAllowlist[record.Name]
		if !ok {
			// The command parses its own arguments or is not governed by the
			// shared allowlist; its advertisement is checked elsewhere.
			continue
		}
		accepted := map[string]bool{}
		for _, name := range allowed {
			accepted[name] = true
		}
		named := map[string][]string{}
		add := func(source string, flags []string) {
			for _, flag := range flags {
				named[strings.TrimPrefix(flag, "-")] = append(named[strings.TrimPrefix(flag, "-")], source)
			}
		}
		add("repeatableFlags", record.RepeatableFlags)
		add("writeFlags", record.WriteFlags)
		for flag := range record.ArgumentValues {
			named[flag] = append(named[flag], "argumentValues")
		}
		for flag := range record.ArgumentType {
			named[flag] = append(named[flag], "argumentType")
		}
		for flag := range record.ArgumentDefaults {
			named[flag] = append(named[flag], "argumentDefaults")
		}
		for flag, sources := range named {
			if !accepted[flag] {
				t.Fatalf("%q advertises -%s via %v, but its accepted flags are %v",
					record.Name, flag, sources, allowed)
			}
		}
	}
}

// The four flags validate used to advertise are read by other commands, which is
// why they appeared in the shared FlagSet's help. They must stay out of
// validate's advertisement.
func TestValidateDoesNotAdvertiseOtherCommandsFlags(t *testing.T) {
	catalog := commandCatalog()
	var validate *cliCommandRecord
	for i := range catalog.Commands {
		if catalog.Commands[i].Name == "validate" {
			validate = &catalog.Commands[i]
		}
	}
	if validate == nil {
		t.Fatal("validate is not advertised")
	}
	for _, flag := range []string{"-analysis", "-schema", "-plan", "-evidence"} {
		for _, advertised := range validate.RepeatableFlags {
			if advertised == flag {
				t.Fatalf("validate still advertises %s", flag)
			}
		}
	}
	// And the repeat guard must not treat them as repeatable for validate,
	// otherwise a repeated -analysis would be accepted by the guard and then
	// refused by the argument check, which is two different answers.
	for _, flag := range repeatableFlags["validate"] {
		_ = flag
		t.Fatalf("validate's repeat map still names a repeatable flag")
	}
}
