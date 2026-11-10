package syntax

import (
	"strings"
	"testing"
)

// A contract header is `contract <id> <version> { ... }` with three bare
// tokens, so writing the brace straight after the id is the obvious mistake.
// The generic `unexpected token "{"` left the author to guess the shape (#4169).
func TestContractWithoutVersionNamesTheShape(t *testing.T) {
	for _, src := range []string{
		"contract sales_core.probe {\n}\n",
		"contract sales_core.probe ^1 {\n}\n",
	} {
		_, err := NewParser(src).ParseContract()
		if err == nil {
			t.Fatalf("%s: expected a parse error", src)
		}
		message, ok := ErrorMessage(err)
		if !ok {
			t.Fatalf("%s: error has no message", src)
		}
		if !strings.Contains(message, "expected a version") {
			t.Errorf("message does not ask for a version: %s", message)
		}
		if !strings.Contains(message, "sales_core.probe") {
			t.Errorf("message does not show the id in a correct form: %s", message)
		}
		if strings.Contains(message, `unexpected token "{"`) ||
			strings.Contains(message, `unexpected token "^"`) {
			t.Errorf("message is still the generic one: %s", message)
		}
	}
}

// A version token that parses as an invalid version keeps its own message: it is
// present, so this is a different mistake and must not be reported as missing.
func TestContractWithAnInvalidVersionTokenKeepsItsMessage(t *testing.T) {
	_, err := NewParser("contract sales_core.probe version 1 {\n}\n").ParseContract()
	if err == nil {
		t.Fatal("expected a parse error")
	}
	message, ok := ErrorMessage(err)
	if !ok {
		t.Fatal("error has no message")
	}
	if !strings.Contains(message, "invalid version") {
		t.Errorf("message is %q, want it to say the version is invalid", message)
	}
	if strings.Contains(message, "expected a version") {
		t.Errorf("a present-but-invalid version must not be reported as missing: %s", message)
	}
}

// The new check must not swallow the id position or the body errors.
func TestContractHeaderControlsAreUnchanged(t *testing.T) {
	cases := []struct {
		src  string
		want string
	}{
		{"contract {\n}\n", `unexpected token "{"`},
		{"contract sales_core.probe v1 {\n    frobnicate\n}\n", "expected read, write, or query"},
	}
	for _, c := range cases {
		_, err := NewParser(c.src).ParseContract()
		if err == nil {
			t.Fatalf("%s: expected a parse error", c.src)
		}
		message, ok := ErrorMessage(err)
		if !ok || !strings.Contains(message, c.want) {
			t.Errorf("%s: message=%q want it to contain %q", c.src, message, c.want)
		}
	}
}

// The error range must point at the token that ended the header, not at the id.
func TestContractMissingVersionRangePointsAtTheBrace(t *testing.T) {
	src := "contract sales_core.probe {\n}\n"
	_, err := NewParser(src).ParseContract()
	if err == nil {
		t.Fatal("expected a parse error")
	}
	start, end, ok := ErrorRange(err)
	if !ok {
		t.Fatalf("error has no source range: %T %v", err, err)
	}
	if got := src[start.Offset:end.Offset]; got != "{" {
		t.Fatalf("range text=%q want %q", got, "{")
	}
}
