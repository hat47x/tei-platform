package plugindocreference

import (
	"bytes"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	graphreader "tei/reference/tei-md-graph-reader"
)

// This test links the PDC-0 document/graph fixture to the *existing*
// restricted Graph/Rule generator. Its candidate shape is not a Source
// fingerprint, a disclosure grant, or a product Plugin/Provider identity.
func TestPDC0ReferenceGraphShapeDoesNotReplaceSourceOrDisclosure(t *testing.T) {
	fixture := readFixture(t)
	if fixture.Adoption != "proposal-only" ||
		fixture.Documents.Public.SourceRefStatus != "candidate-unbound" ||
		fixture.Documents.Private.SourceRefStatus != "candidate-unbound" {
		t.Fatal("candidate data was unexpectedly promoted to a product document contract")
	}

	load := func(name string) ([]byte, string, []byte) {
		t.Helper()
		raw, err := os.ReadFile(filepath.Join("..", "..", "testdata", "tei-md-graph", name))
		if err != nil {
			t.Fatal(err)
		}
		candidate, err := graphreader.ParseRuleCandidate(raw)
		if err != nil {
			t.Fatal(err)
		}
		shape, err := graphreader.CandidateShapeFingerprint(candidate)
		if err != nil {
			t.Fatal(err)
		}
		generated, err := graphreader.GenerateGo(candidate)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Contains(generated, []byte("const candidateShapeDigest = "+strconv.Quote(shape))) {
			t.Fatal("the generated Go did not embed its own reference candidate shape")
		}
		return raw, shape, generated
	}

	const baselineName = "baseline.tei.md"
	if fixture.Graph.Source != "testdata/tei-md-graph/"+baselineName {
		t.Fatalf("candidate PDC-0 graph source changed: %q", fixture.Graph.Source)
	}
	baseRaw, baseShape, baseGo := load(baselineName)
	if fixtureDigest(baseRaw) != fixture.Graph.SourceDigest {
		t.Fatal("PDC-0 source-byte pin does not match the Graph/Rule input")
	}

	layoutRaw, layoutShape, layoutGo := load("layout-only.tei.md")
	if bytes.Equal(baseRaw, layoutRaw) || fixtureDigest(baseRaw) == fixtureDigest(layoutRaw) {
		t.Fatal("layout-only source should have its own authored-byte identity")
	}
	if layoutShape != baseShape || !bytes.Equal(layoutGo, baseGo) {
		t.Fatal("layout-only change altered the generated Graph/Rule behavior")
	}

	for _, name := range []string{"edge-reversed.tei.md", "rule-by-target.tei.md"} {
		t.Run(name, func(t *testing.T) {
			_, changedShape, changedGo := load(name)
			if changedShape == baseShape || bytes.Equal(changedGo, baseGo) {
				t.Fatalf("%s changed a Graph/Rule condition without changing generated logic", name)
			}
		})
	}

	// Only prose changes: the old Source pin must not become current even
	// though the interpreted candidate and generated executable stay equal.
	proseEdited := append(bytes.Clone(baseRaw), []byte("\n\n# Additional explanation\n\nDocumentation revision only.\n")...)
	if fixtureDigest(proseEdited) == fixture.Graph.SourceDigest {
		t.Fatal("authored Source revision failed to invalidate the old byte pin")
	}
	editedCandidate, err := graphreader.ParseRuleCandidate(proseEdited)
	if err != nil {
		t.Fatal(err)
	}
	editedShape, err := graphreader.CandidateShapeFingerprint(editedCandidate)
	if err != nil {
		t.Fatal(err)
	}
	editedGo, err := graphreader.GenerateGo(editedCandidate)
	if err != nil {
		t.Fatal(err)
	}
	if editedShape != baseShape || !bytes.Equal(editedGo, baseGo) {
		t.Fatal("prose-only edit changed the executable candidate structure")
	}

	publicManual, err := os.ReadFile(filepath.Join("..", "..", filepath.FromSlash(fixture.Documents.Public.Path)))
	if err != nil {
		t.Fatal(err)
	}
	if fixtureDigest(publicManual) != fixture.Documents.Public.SourceDigest {
		t.Fatal("the public Manual candidate Source pin is stale")
	}
	changedManual := append(bytes.Clone(publicManual), []byte("\nAdditional usage note.\n")...)
	if fixtureDigest(changedManual) == fixture.Documents.Public.SourceDigest {
		t.Fatal("edited Manual reused the old Document Source pin")
	}
	// The Graph remains byte-identical when an independent Manual changes.
	// A real Product read must *separately* check owner, exact selected
	// Contract/Lock, current audience/Disclosure and requested consumer.
	if fixtureDigest(baseRaw) != fixture.Graph.SourceDigest {
		t.Fatal("independent Manual edit unexpectedly changed the Graph source")
	}
}
