package syntax_test

import (
	"testing"

	"tei/internal/syntax"
)

func TestProcessRuntimeRejectsLegacyWhitespaceCommand(t *testing.T) {
	_, err := syntax.NewParser(`plugin tei.process.reference v1.0.0 {
        runtime process "python3 -m worker" sandbox required
    }`).ParsePlugin()
	if err == nil {
		t.Fatal("legacy whitespace command was accepted instead of requiring explicit args")
	}
}

func TestProcessRuntimePreservesArgumentBoundaries(t *testing.T) {
	p, err := syntax.NewParser(`plugin tei.process.reference v1.0.0 {
        runtime process "python3" args "-m" "worker module" "--flag=a b" sandbox required
    }`).ParsePlugin()
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"-m", "worker module", "--flag=a b"}
	if p.Runtime.Command != "python3" || len(p.Runtime.Args) != len(want) {
		t.Fatalf("runtime=%#v", p.Runtime)
	}
	for i := range want {
		if p.Runtime.Args[i] != want[i] {
			t.Fatalf("args[%d]=%q want=%q", i, p.Runtime.Args[i], want[i])
		}
	}
}
