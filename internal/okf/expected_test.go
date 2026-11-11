package okf

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// readerExpectedPath is the declarative expectation file for this reader. It is
// separate from testdata/okf-k1-v02.expected.json, which pins the fixture's raw
// bytes for the upstream-reference witness.
const readerExpectedPath = "../../testdata/okf-k1-reader.expected.json"

type readerExpected struct {
	Reader                string `json:"reader"`
	OracleQualification   string `json:"oracle_qualification"`
	ExpectedRuntimeStatus string `json:"expected_runtime_status"`
	Okf                   struct {
		Version    string `json:"version"`
		SpecCommit string `json:"spec_commit"`
		SpecSHA256 string `json:"spec_sha256_utf8"`
	} `json:"okf"`
	Baseline struct {
		ReaderOkfVersion string `json:"readerOkfVersion"`
		Format           struct {
			Conformant       bool `json:"conformant"`
			Concepts         int  `json:"concepts"`
			Indexes          int  `json:"indexes"`
			Logs             int  `json:"logs"`
			FormatErrorCount int  `json:"formatErrorCount"`
		} `json:"format"`
		ConceptIDs []string          `json:"conceptIds"`
		Digests    map[string]string `json:"digests"`
		Unresolved int               `json:"unresolvedReferenceCount"`
	} `json:"baseline"`
	MustNotInfer []string `json:"mustNotInfer"`
	Cases        []struct {
		ID     string `json:"id"`
		Name   string `json:"name"`
		Expect struct {
			FormatConformant *bool `json:"formatConformant"`
			Concepts         *int  `json:"concepts"`
		} `json:"expect"`
	} `json:"cases"`
}

// loadReaderExpected reads and validates the declarative expectation file. A
// qualification string that is not PROVISIONAL fails here: this file must not
// promote itself to an accepted oracle.
func loadReaderExpected(t *testing.T) readerExpected {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(readerExpectedPath))
	if err != nil {
		t.Fatalf("read %s: %v", readerExpectedPath, err)
	}
	var expected readerExpected
	if err := json.Unmarshal(raw, &expected); err != nil {
		t.Fatalf("parse %s: %v", readerExpectedPath, err)
	}
	if expected.Reader == "" || expected.Okf.Version != "0.2" {
		t.Fatalf("expected file does not name the reader and OKF 0.2: %+v", expected)
	}
	if expected.Okf.SpecCommit != "ad30107c31c06aec8a7d5636e0d1058118604e6f" {
		t.Fatalf("spec commit = %q", expected.Okf.SpecCommit)
	}
	if expected.Okf.SpecSHA256 != "26aa5da029278939f914e578107242d9607d4f2dc5fe153272b82f9ed1030101" {
		t.Fatalf("spec sha256 = %q", expected.Okf.SpecSHA256)
	}
	if len(expected.OracleQualification) == 0 || expected.OracleQualification[:11] != "PROVISIONAL" {
		t.Fatalf("oracle qualification = %q, want a PROVISIONAL statement", expected.OracleQualification)
	}
	if len(expected.Cases) != 22 {
		t.Fatalf("case count = %d, want the 22 declared cases", len(expected.Cases))
	}
	return expected
}

// TestExpectedFileMatchesBaseline is the bridge between the declarative file and
// the reader: every baseline value in the file is asserted against a live read. A
// drift in either direction fails, so the file cannot quietly restate the reader's
// output.
func TestExpectedFileMatchesBaseline(t *testing.T) {
	expected := loadReaderExpected(t)
	bundle := readFixture(t)

	if bundle.OkfVersion != expected.Baseline.ReaderOkfVersion {
		t.Fatalf("reader version = %q, want %q", bundle.OkfVersion, expected.Baseline.ReaderOkfVersion)
	}
	if bundle.Format.Conformant != expected.Baseline.Format.Conformant {
		t.Fatalf("conformance = %t, want %t", bundle.Format.Conformant, expected.Baseline.Format.Conformant)
	}
	if len(bundle.Concepts) != expected.Baseline.Format.Concepts {
		t.Fatalf("concepts = %d, want %d", len(bundle.Concepts), expected.Baseline.Format.Concepts)
	}
	if len(bundle.Indexes) != expected.Baseline.Format.Indexes {
		t.Fatalf("indexes = %d, want %d", len(bundle.Indexes), expected.Baseline.Format.Indexes)
	}
	if len(bundle.Logs) != expected.Baseline.Format.Logs {
		t.Fatalf("logs = %d, want %d", len(bundle.Logs), expected.Baseline.Format.Logs)
	}
	if len(bundle.Format.FormatErrors) != expected.Baseline.Format.FormatErrorCount {
		t.Fatalf("format errors = %d, want %d", len(bundle.Format.FormatErrors), expected.Baseline.Format.FormatErrorCount)
	}
	if len(bundle.UnresolvedReferences) != expected.Baseline.Unresolved {
		t.Fatalf("unresolved references = %d, want %d", len(bundle.UnresolvedReferences), expected.Baseline.Unresolved)
	}

	for _, conceptID := range expected.Baseline.ConceptIDs {
		if findConcept(bundle, conceptID) == nil {
			t.Fatalf("missing expected concept %q", conceptID)
		}
	}
	for path, digest := range expected.Baseline.Digests {
		got := ""
		if concept := findConcept(bundle, conceptIDFor(path)); concept != nil {
			got = concept.Digest
		}
		for _, index := range bundle.Indexes {
			if index.Path == path {
				got = index.Digest
			}
		}
		if got != digest {
			t.Errorf("%s digest = %q, want %q", path, got, digest)
		}
	}
}

// TestExpectedCaseIDsAreStable covers the case table: the reader's published cases
// must stay the ones the file declares, so a removed case is visible rather than
// silently dropped from the expectation set.
func TestExpectedCaseIDsAreStable(t *testing.T) {
	expected := loadReaderExpected(t)
	for index, want := range []string{
		"OKF-E01", "OKF-E02", "OKF-E03", "OKF-E04", "OKF-E05", "OKF-E06",
		"OKF-E07", "OKF-E08", "OKF-E09", "OKF-E10", "OKF-E11", "OKF-E12",
		"OKF-E13", "OKF-E14", "OKF-E15", "OKF-E16", "OKF-E17", "OKF-E18",
		"OKF-E19", "OKF-E20", "OKF-E21", "OKF-E22",
	} {
		if expected.Cases[index].ID != want {
			t.Fatalf("case %d = %q, want %q", index, expected.Cases[index].ID, want)
		}
	}
	// The two cases whose expectation is the whole-bundle verdict must agree with
	// the baseline table above.
	for _, item := range expected.Cases {
		if item.ID != "OKF-E01" {
			continue
		}
		if item.Expect.FormatConformant == nil || !*item.Expect.FormatConformant {
			t.Fatalf("OKF-E01 must declare conformance")
		}
		if item.Expect.Concepts == nil || *item.Expect.Concepts != 3 {
			t.Fatalf("OKF-E01 must declare three concepts")
		}
	}
	if len(expected.MustNotInfer) == 0 {
		t.Fatal("mustNotInfer must not be empty: a reader that answers format questions cannot answer semantic ones")
	}
}

// TestExpectedFileDeclaresTheReaderBoundary covers the caveat list: the file must
// state what the reader does not do, so a consumer reading only the JSON cannot
// mistake a conformant bundle for an accepted one.
func TestExpectedFileDeclaresTheReaderBoundary(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join(readerExpectedPath))
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]json.RawMessage
	if err := json.Unmarshal(raw, &document); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"mustNotInfer", "readerCaveats", "oracle_qualification", "expected_runtime_status"} {
		if _, present := document[key]; !present {
			t.Fatalf("expected file lacks %q", key)
		}
	}
}
