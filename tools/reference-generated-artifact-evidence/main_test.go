package main

import (
	"reflect"
	"testing"
)

func TestCheckOnlyCommandsNeverUseWriteMode(t *testing.T) {
	got := checkOnlyCommands()
	want := []commandSpec{
		{
			Name: "world-plugin-lock-golden",
			Args: []string{"run", "./tools/reference-golden"},
		},
		{
			Name: "durable-run-golden",
			Args: []string{"run", "./tools/reference-durable-run-golden"},
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("check-only command=%#v want=%#v", got, want)
	}
	for _, spec := range got {
		for _, arg := range spec.Args {
			if arg == "-write" {
				t.Fatalf("完了Evidence runnerが修復操作-writeを含んでいます: %#v", spec)
			}
		}
	}
}

func TestCommandDisplayKeepsArgumentBoundaries(t *testing.T) {
	spec := commandSpec{Name: "example", Args: []string{"run", "./tools/reference-golden"}}
	got := commandDisplay(spec)
	want := []string{"go", "run", "./tools/reference-golden"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("command display=%#v want=%#v", got, want)
	}
}
