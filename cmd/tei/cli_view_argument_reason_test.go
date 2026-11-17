package main

import (
	"strings"
	"testing"
)

type writerFunc func(string)

func (f writerFunc) Write(p []byte) (int, error) {
	f(string(p))
	return len(p), nil
}

// The argument guard is a chain of many conditions. One shared message over it
// told a caller who had passed exactly one selector that they had passed more
// than one, which is the opposite of their argv. Each refusal now names the
// flag that is wrong.
func TestViewRefusalsNameTheFlagThatIsWrong(t *testing.T) {
	cases := []struct {
		args []string
		want string
	}{
		{[]string{"-workspace", "w", "-application", "a", "-pattern", "p", "-execute"}, "-execute requires -command"},
		{[]string{"-workspace", "w", "-application", "a", "-pattern", "p", "-period-state", "x"}, "-period-state requires -execute"},
		{[]string{"-workspace", "w", "-application", "a", "-allow-capability", "c"}, "-allow-capability requires -execute"},
		{[]string{"-workspace", "w", "-application", "a", "-operation-log", "f"}, "-operation-log requires -execute"},
		{[]string{"-workspace", "w", "-application", "a", "stray"}, "does not take positional arguments"},
		{[]string{"-application", "a"}, "requires -workspace and -application"},
		{[]string{"-workspace", "w", "-application", "a", "-command", "c"}, "-command requires -pattern"},
		{[]string{"-workspace", "w", "-application", "a", "-pattern", "p", "-format", "xml"}, "-format must be text, json, html, or print"},
	}
	for _, tc := range cases {
		var message string
		code, err := runViewCommand(tc.args, writerFunc(func(text string) { message = text }))
		if err != nil {
			t.Fatalf("%v: %v", tc.args, err)
		}
		if code == 0 {
			t.Errorf("%v was accepted", tc.args)
			continue
		}
		if !strings.Contains(message, tc.want) {
			t.Errorf("%v: message does not say %q: %s", tc.args, tc.want, message)
		}
	}
}
