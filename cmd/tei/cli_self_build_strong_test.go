package main

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

// These tests pin the opt-in flag SB-C0 §4.25 left unnamed and §4.30 names.
//
// The flag's whole value is that it is thin: it changes no default, invents no
// input, and refuses rather than returning a strong name over a weak attempt.
// Each of those properties is tested, because each is easy to lose by accident.

// Without the flag nothing changes: no second build, no record.
func TestStrongChangesNothingWhenAbsent(t *testing.T) {
	rebuild, record, err := applyStrong(false, false, false)
	if err != nil {
		t.Fatalf("absence was refused: %v", err)
	}
	if rebuild {
		t.Fatal("the absent flag turned on the reproduction check")
	}
	if record.Requested {
		t.Fatal("the absent flag recorded itself as requested")
	}
	if len(record.Implied) != 0 {
		t.Fatalf("the absent flag implied observations: %v", record.Implied)
	}
}

// With the flag and the in-process route, the reproduction check is turned on and
// the record names it as implied rather than caller-requested.
func TestStrongImpliesReproduction(t *testing.T) {
	rebuild, record, err := applyStrong(true, false, true)
	if err != nil {
		t.Fatalf("a well-formed -strong was refused: %v", err)
	}
	if !rebuild {
		t.Fatal("-strong did not turn on the reproduction check")
	}
	if !record.Requested {
		t.Fatal("-strong did not record itself as requested")
	}
	if len(record.Implied) != 1 || record.Implied[0] != strongImpliedRebuildCheck {
		t.Fatalf("the record does not name what it implied: %v", record.Implied)
	}
	if record.Route != derivedByEntryInProcess {
		t.Fatalf("the record does not name the required route: %q", record.Route)
	}
	if record.Boundary == "" {
		t.Fatal("the record has no boundary")
	}
}

// When the caller already asked for reproduction, the flag adds nothing to the
// implied list: it did not turn that on.
func TestStrongDoesNotClaimCreditForAnExplicitChoice(t *testing.T) {
	rebuild, record, err := applyStrong(true, true, true)
	if err != nil {
		t.Fatalf("refused: %v", err)
	}
	if !rebuild {
		t.Fatal("the reproduction check was turned off")
	}
	if len(record.Implied) != 0 {
		t.Fatalf("the flag claimed to imply what the caller asked for: %v", record.Implied)
	}
}

// Without the in-process route the flag refuses. Returning a strong name over an
// attempt that cannot support the claim is the failure the unmet list exists to
// prevent, so this is a refusal rather than a weaker success.
func TestStrongRefusesWithoutTheInProcessRoute(t *testing.T) {
	_, _, err := applyStrong(true, false, false)
	if err == nil {
		t.Fatal("-strong was accepted without the in-process route")
	}
	msg := err.Error()
	for _, want := range []string{"-build-catalog", "resolver child"} {
		if !strings.Contains(msg, want) {
			t.Fatalf("the refusal does not mention %q: %s", want, msg)
		}
	}
}

// The CLI refuses end to end: no artifact is produced, and the reason names the
// missing route.
func TestStrongIsRefusedByTheCommandWithoutACatalog(t *testing.T) {
	dir := t.TempDir()
	child := buildSelfBuildChild(t, dir)
	workspace := outerWorkspace(t, dir)
	output := filepath.Join(dir, "artifact")

	var out bytes.Buffer
	code, err := runSelfBuildCommand([]string{
		"-child", child,
		"-workspace", workspace,
		"-intent-file", outerIntentFile(t, dir),
		"-state", filepath.Join(dir, "state"),
		"-output", output,
		"-strong",
	}, &out)
	if code != 2 {
		t.Fatalf("-strong without a catalog was accepted: code=%d out=%s", code, out.String())
	}
	if err == nil || !strings.Contains(err.Error(), "-strong") {
		t.Fatalf("the refusal does not name the flag: %v", err)
	}
}

// The record appears in the report, so a reader can tell the second build ran
// because of the flag.
func TestStrongIsRecordedInTheReport(t *testing.T) {
	dir := t.TempDir()
	child := buildSelfBuildChild(t, dir)
	workspace := outerWorkspace(t, dir)
	output := filepath.Join(dir, "artifact")
	catalog := "../../tools/reference-build-authoring-intent/testdata/catalog.json"
	srcRoot := "../../tools/reference-build-authoring-intent/testdata/src"
	intent := "../../tools/reference-build-authoring-intent/testdata/intent.json"

	digest, digestErr := digestOfFile(catalog)
	if digestErr != nil {
		t.Fatalf("catalog digest: %v", digestErr)
	}

	var out bytes.Buffer
	code, _ := runSelfBuildCommand([]string{
		"-child", child,
		"-workspace", workspace,
		"-intent-file", intent,
		"-build-catalog", catalog,
		"-build-catalog-digest", digest,
		"-build-source-root", srcRoot,
		"-state", filepath.Join(dir, "state"),
		"-output", output,
		"-strong",
		"-child-arg", childArgs(t,
			"-artifact", output,
			"-digest", digestOfBytes([]byte("artifact-bytes"))),
	}, &out)

	var report struct {
		Strong *struct {
			Requested bool     `json:"requested"`
			Implied   []string `json:"implied"`
			Route     string   `json:"routeRequired"`
			Boundary  string   `json:"boundary"`
		} `json:"strong"`
		ArtifactReproducibility *struct {
			State string `json:"state"`
		} `json:"artifactReproducibility"`
	}
	if err := json.Unmarshal(out.Bytes(), &report); err != nil {
		t.Fatalf("decode: %v: %s", err, out.String())
	}
	if report.Strong == nil {
		t.Fatalf("-strong produced no record (code=%d): %s", code, out.String())
	}
	if !report.Strong.Requested || report.Strong.Route != derivedByEntryInProcess {
		t.Fatalf("the record is incomplete: %+v", report.Strong)
	}
	if report.Strong.Boundary == "" {
		t.Fatal("the record has no boundary")
	}
	// The flag must actually have turned the reproduction on: a record that says
	// it implied the check while the check did not run would be a lie.
	if report.ArtifactReproducibility == nil {
		t.Fatal("no reproducibility record although -strong implies it")
	}
	if report.ArtifactReproducibility.State == reproducibilityUnperformed {
		t.Fatalf("the reproduction check did not run: %+v", report.ArtifactReproducibility)
	}
}
