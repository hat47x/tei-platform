package generatorselection

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// The Java target must carry the same three failure axes the Go runtime carries,
// or a caller has to keep a per-language list of code strings. This test reads
// the generated source and runs the compiled program, because either alone would
// miss the case where the contract is declared but never used.
func TestJavaTargetCarriesTheUnifiedFailureAxes(t *testing.T) {
	if _, _, _, ok := findJava(); !ok {
		t.Skip("javac/java is not available")
	}
	_, contract, _, _ := paths(t)
	generated, err := GenerateVersion(contract, "java-template", "1", filepath.Join(t.TempDir(), "java-template"))
	if err != nil {
		t.Fatal(err)
	}
	source, err := os.ReadFile(filepath.Join(generated.Path, generated.Manifest.Source))
	if err != nil {
		t.Fatal(err)
	}
	text := string(source)

	// The contract itself is emitted, so a caller in the same package can catch
	// one type and read all four fields.
	for _, want := range []string{"class TeiFailure", "enum Kind", "enum RetryPolicy", "effectUnknown"} {
		if !strings.Contains(text, want) {
			t.Fatalf("generated Java does not declare %q", want)
		}
	}
	// The wire values are the closed sets the Go runtime uses, not Java-flavoured
	// spellings.
	for _, want := range []string{
		`BUSINESS_REJECTION("business_rejection")`,
		`TECHNICAL_FAILURE("technical_failure")`,
		`PARTIAL_SUCCESS("partial_success")`,
		`RETRYABLE("retryable")`,
		`NOT_RETRYABLE("not_retryable")`,
		`LOOKUP_FIRST("lookup_first")`,
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("generated Java does not declare the wire value %q", want)
		}
	}

	// Now the behaviour: a business rejection and a technical failure must come
	// out with different axes, and neither may be reported as retryable-with-
	// unknown-effect at the same time.
	workDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(workDir, "Main.java"), source, 0o600); err != nil {
		t.Fatal(err)
	}
	classes := filepath.Join(workDir, "classes")
	if err := compileJavaCandidate(context.Background(), javacPathFor(t), workDir, classes); err != nil {
		t.Fatalf("compile generated java: %v", err)
	}

	business := runJava(t, classes, `{"quantity":"2","unitPrice":"3","currency":"USD","unit":"EA","perUnit":"EA"}`)
	if business.State != "error" || business.Code != "CURRENCY_MISMATCH" {
		t.Fatalf("unexpected business rejection output: %+v", business)
	}
	if business.Kind != "business_rejection" || business.Retryable != "not_retryable" || business.EffectUnknown {
		t.Fatalf("business rejection axes are wrong: %+v", business)
	}

	// The currency and unit have to be right, or the run is refused as a
	// business rejection before it reaches the decimal conversion this case is
	// about.
	technical := runJava(t, classes, `{"quantity":"x","unitPrice":"3","currency":"JPY","unit":"EA","perUnit":"EA"}`)
	if technical.State != "error" || technical.Code != "DECIMAL_INVALID" {
		t.Fatalf("unexpected technical failure output: %+v", technical)
	}
	if technical.Kind != "technical_failure" {
		t.Fatalf("technical failure kind is wrong: %+v", technical)
	}
	// The effect may have been applied, so the caller must look the result up
	// before resending rather than send again blindly.
	if technical.Retryable != "lookup_first" || !technical.EffectUnknown {
		t.Fatalf("technical failure axes are wrong: %+v", technical)
	}
}

// javaFailureOutput decodes the fields the unified contract puts on the wire.
type javaFailureOutput struct {
	State         string `json:"state"`
	Code          string `json:"code"`
	Kind          string `json:"kind"`
	Retryable     string `json:"retryable"`
	EffectUnknown bool   `json:"effectUnknown"`
}

func runJava(t *testing.T, classes, stdin string) javaFailureOutput {
	t.Helper()
	cmd := exec.CommandContext(context.Background(), javaPathFor(t), "-cp", classes, "Main")
	cmd.Stdin = strings.NewReader(stdin)
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("run generated java with %s: %v (%s)", stdin, err, out)
	}
	var decoded javaFailureOutput
	if err := json.Unmarshal(out, &decoded); err != nil {
		t.Fatalf("generated java output is not one JSON document: %v (%q)", err, out)
	}
	return decoded
}

func javacPathFor(t *testing.T) string {
	t.Helper()
	javac, _, _, ok := findJava()
	if !ok {
		t.Skip("javac/java is not available")
	}
	return javac
}

func javaPathFor(t *testing.T) string {
	t.Helper()
	_, java, _, ok := findJava()
	if !ok {
		t.Skip("javac/java is not available")
	}
	return java
}
