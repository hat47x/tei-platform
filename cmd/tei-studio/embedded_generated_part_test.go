package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"regexp"
	"testing"

	"tei/internal/plugin/transform"
)

// TestEmbeddedWorkbenchCarriesTheCurrentGeneratedInspectionReader reads the
// bytes this binary embeds, not the files next to the source.
//
// The Inspector reader is generated from a contract Source. Three different
// things can go wrong between that Source and the Desktop, and each must fail
// here rather than look like a working Inspector:
//   - the generated part is not in the embedded bundle at all;
//   - the bundle carries a part generated from an older contract revision;
//   - the bundle carries a second, hand-written reader next to the part.
//
// The digest is recomputed from the contract with the transform package's
// canonical form, independently of the generator tool and of the part's text.
// This shows inclusion in the build. It does not show that a WebView ran it;
// the browser and Desktop smokes observe that separately.
func TestEmbeddedWorkbenchCarriesTheCurrentGeneratedInspectionReader(t *testing.T) {
	bundle, err := assets.ReadFile("assets/workbench/main.js")
	if err != nil {
		t.Fatal(err)
	}
	contract, err := os.ReadFile("../../reference/tei-studio-workbench/inspection-reader.contract.json")
	if err != nil {
		t.Fatal(err)
	}
	value, err := transform.ParseJSON(contract)
	if err != nil {
		t.Fatal(err)
	}
	canonical, err := value.JSON()
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(canonical)
	current := "sha256:" + hex.EncodeToString(sum[:])

	carried := regexp.MustCompile(`contractDigest: "(sha256:[0-9a-f]{64})"`).FindAllSubmatch(bundle, -1)
	if len(carried) == 0 {
		t.Fatal("embedded Workbench bundle does not contain the generated Inspector reader")
	}
	if len(carried) != 1 {
		t.Fatalf("embedded Workbench bundle contains %d generated Inspector readers", len(carried))
	}
	if got := string(carried[0][1]); got != current {
		t.Fatalf("embedded Inspector reader was generated from contract %s; the contract Source is now %s", got, current)
	}
	if !bytes.Contains(bundle, []byte("// inspection-reader.generated.ts\n")) {
		t.Fatal("embedded Workbench bundle was not built from inspection-reader.generated.ts")
	}
	// One reader only: esbuild keeps function names, so a hand-written
	// readInspection beside the generated one would appear as a second
	// declaration (renamed readInspection2) or as the Stage0 oracle.
	if n := len(regexp.MustCompile(`function readInspection\d*\(`).FindAll(bundle, -1)); n != 1 {
		t.Fatalf("embedded Workbench bundle declares %d Inspector readers, want exactly the generated one", n)
	}
	if bytes.Contains(bundle, []byte("seedReadInspection")) {
		t.Fatal("embedded Workbench bundle contains the test-only Stage0 oracle")
	}
	// The use marker the smokes read must be in the embedded bytes too.
	if !bytes.Contains(bundle, []byte("teiInspectionReaderCalls")) {
		t.Fatal("embedded Inspector reader does not record its use")
	}
}
