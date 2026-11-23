package goemit

import "testing"

// Formatting failures return an error; Emit turns it into a diagnostic
// instead of panicking. Input validation keeps Emit from reaching this path,
// so the function is called directly.
func TestWithHeaderReturnsAnErrorForSourceThatDoesNotFormat(t *testing.T) {
	if _, err := withHeader("// header", []byte("package p\n\nfunc (")); err == nil {
		t.Fatal("unformattable source gave no error")
	}
	if _, err := standardPart("result.go", "main", "// header"); err != nil {
		t.Fatal(err)
	}
}
