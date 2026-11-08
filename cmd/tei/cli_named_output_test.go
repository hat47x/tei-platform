package main

import (
	"testing"
)

// The discovery entry names the flags that make a command write files, so a
// caller reading `effect: read-only` can see exactly which outputs are
// caller-named (#3182). The Workspace identity (#3181) is already implemented
// on main and tested in internal/workspace and internal/operations.
func TestCommandsNameCallerNamedOutputFlags(t *testing.T) {
	catalog := commandCatalog()
	var view *cliCommandRecord
	for i := range catalog.Commands {
		if catalog.Commands[i].Name == "view" {
			view = &catalog.Commands[i]
		}
	}
	if view == nil {
		t.Fatal("view is not advertised")
	}
	if view.Effect != "read-only" {
		t.Fatalf("view effect = %q", view.Effect)
	}
	if len(view.WritesToNamedOutput) == 0 {
		t.Fatal("view advertises read-only but names no output-writing flags")
	}
	want := map[string]bool{"-operation-log": true, "-lifecycle-state": true}
	for _, name := range view.WritesToNamedOutput {
		if !want[name] {
			t.Fatalf("unexpected output flag %q", name)
		}
		delete(want, name)
	}
	if len(want) != 0 {
		t.Fatalf("missing output flags: %v", want)
	}
	// Every named flag must be one the command actually advertises as optional.
	optional := map[string]bool{}
	for _, item := range view.Optional {
		optional[item] = true
	}
	for _, name := range view.WritesToNamedOutput {
		matched := false
		for item := range optional {
			if len(item) >= len(name) && item[:len(name)] == name {
				matched = true
				break
			}
		}
		if !matched {
			t.Fatalf("%s writes %s but does not advertise it", view.Name, name)
		}
	}
	// Commands that write nothing by themselves must not claim otherwise.
	for _, cmd := range catalog.Commands {
		if cmd.Name == "view" {
			continue
		}
		if len(cmd.WritesToNamedOutput) != 0 {
			t.Fatalf("%s unexpectedly names output flags %v", cmd.Name, cmd.WritesToNamedOutput)
		}
	}
}
