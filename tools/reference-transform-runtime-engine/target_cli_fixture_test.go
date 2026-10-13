package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"tei/internal/plugin/transform"
)

// Export the *same* request used by the in-process conformance test so the
// real native and js/wasm CLI binaries can be exercised after Host acceptance.
func TestExportTargetCLIConformanceFixture(t *testing.T) {
	directory := os.Getenv("TEI_TARGET_CLI_FIXTURE_OUT")
	if directory == "" {
		t.Skip("requires an explicit fixture output directory")
	}
	source := referenceMetaBuilderSource(t)
	value, err := transform.SourceValue(source)
	if err != nil {
		t.Fatal(err)
	}
	wire, err := encodeWireValue(value, source.Limits, 0)
	if err != nil {
		t.Fatal(err)
	}
	sourceBytes, _, err := transform.MaterializeSource(source)
	if err != nil {
		t.Fatal(err)
	}
	requestBytes := requestBytes(t, engineRequest{
		Schema: engineRequestSchema,
		Inputs: []wireInput{{Name: "transformSource", Value: wire}},
	})
	response, err := execute(source, requestBytes)
	if err != nil {
		t.Fatal(err)
	}
	expected, err := json.Marshal(response)
	if err != nil {
		t.Fatal(err)
	}
	for _, fixture := range []struct {
		name  string
		bytes []byte
	}{
		{"cli-source.json", sourceBytes},
		{"cli-request.json", requestBytes},
		{"cli-expected-response.json", expected},
	} {
		if err := os.WriteFile(filepath.Join(directory, fixture.name), fixture.bytes, 0o600); err != nil {
			t.Fatal(err)
		}
	}
}
