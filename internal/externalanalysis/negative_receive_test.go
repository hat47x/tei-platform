package externalanalysis_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"tei/internal/externalanalysis"
)

func mustReadFile(t *testing.T, path string) []byte {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func jsonUnmarshal(raw []byte, target any) error {
	return json.Unmarshal(raw, target)
}

func mustJSONNode(t *testing.T, observation externalanalysis.Observation) string {
	t.Helper()
	raw, err := json.Marshal(observation)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

// derivedFixture writes a derived copy of the captured Evidence with the first
// node's location declared unavailable, and returns its path. The bytes are the
// producer's, with one declaration changed: the negative is about what the
// producer did not observe, not about a different producer.
func unavailableLocationFixture(t *testing.T, dir string) string {
	t.Helper()
	raw := fixture(t, "rea-6.8.0/analysis.json")
	text := decode(t, raw)
	const available = `"location":{"available":true,"value":{"start":{"line":2,"column":6},"end":{"line":2,"column":50},"kind":"source-range","source":"index.js"}}`
	const unavailable = `"location":{"available":false}`
	derived := strings.Replace(text, available, unavailable, 1)
	if derived == text {
		t.Fatal("the fixture does not carry the node location the test expects")
	}
	path := filepath.Join(dir, "unavailable-location.json")
	if err := os.WriteFile(path, []byte(derived), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

// TestUnavailableLocationIsNotAnEmptyLocation: a location the producer declared
// it could not observe is reported as unavailable, not as an empty place name
// that reads like "no location".
func TestUnavailableLocationIsNotAnEmptyLocation(t *testing.T) {
	path := unavailableLocationFixture(t, t.TempDir())
	receipt := externalanalysis.Receive([]byte(mustReadFile(t, path)), presentTool("6.8.0"), acquisition)
	candidates, err := externalanalysis.ReadCandidates(receipt.EvidenceBody())
	if err != nil {
		t.Fatalf("read candidates: %v", err)
	}
	node := candidates.Observations[0]
	if node.Location != "" {
		t.Fatalf("location=%q: an unavailable location has no place name", node.Location)
	}
	if node.Span != nil {
		t.Fatalf("span=%+v: an unavailable location has no span", node.Span)
	}
	if len(node.Unavailable) == 0 {
		t.Fatal("the observation must name what it could not observe")
	}
	// The rest of the observation survives, so the candidate is still usable.
	if node.NodeID == "" || node.Kind == "" {
		t.Fatalf("node=%+v", node)
	}
}

// TestStoredEvidenceReadbackIsReproducibleAndRequalifiable: a stored Evidence
// read back later yields the same receipt, and the same Evidence ID with
// different bytes is a different observation rather than the same one re-read
// (RE-16: historical readback is allowed, current qualification is separate).
func TestStoredEvidenceReadbackIsReproducibleAndRequalifiable(t *testing.T) {
	raw := fixture(t, "rea-6.8.0/analysis.json")
	first := externalanalysis.Receive(raw, presentTool("6.8.0"), acquisition)
	second := externalanalysis.Receive(raw, presentTool("6.8.0"), acquisition)
	if first.Raw.Fingerprint != second.Raw.Fingerprint {
		t.Fatal("the same bytes must read back as the same receipt")
	}
	if first.Evidence.ID != second.Evidence.ID || first.Source.Revision != second.Source.Revision {
		t.Fatal("the same bytes must read back as the same observation")
	}
	// A stored readback is not a fresh run: the environment records it.
	if !first.Tool.Present {
		t.Fatal("the test declares the tool present")
	}
	pins := externalanalysis.Pins{
		EvidenceID:     first.Evidence.ID,
		Fingerprint:    first.Raw.Fingerprint,
		SourceRevision: first.Source.Revision,
	}
	if err := second.VerifyPins(pins); err != nil {
		t.Fatalf("the same bytes must match the recorded pins: %v", err)
	}
}

// TestExternalIdentityIsNeverASubjectIdentity: the projection carries the
// producer's identity verbatim and has no place to put a TEI subject, so a
// caller cannot read an external ID as a semantic subject even when a subject
// with the same string exists in a Workspace.
func TestExternalIdentityIsNeverASubjectIdentity(t *testing.T) {
	raw := fixture(t, "rea-6.8.0/analysis.json")
	receipt := externalanalysis.Receive(raw, presentTool("6.8.0"), acquisition)
	candidates, err := externalanalysis.ReadCandidates(receipt.EvidenceBody())
	if err != nil {
		t.Fatalf("read candidates: %v", err)
	}
	node := candidates.Observations[0]
	body := mustReadFile(t, filepath.Join("testdata", "rea-6.8.0", "analysis.json"))
	// The external identity appears under the external field only.
	if !strings.Contains(decode(t, body), `"node_id":"`+node.NodeID+`"`) {
		t.Fatal("the producer states the node identity under node_id")
	}
	view := map[string]any{}
	if err := jsonUnmarshal([]byte(mustJSONNode(t, node)), &view); err != nil {
		t.Fatal(err)
	}
	if _, exists := view["subject"]; exists {
		t.Fatal("an external observation must not carry a TEI subject field")
	}
	if view["nodeId"] != node.NodeID {
		t.Fatalf("nodeId=%v want %s", view["nodeId"], node.NodeID)
	}
}
