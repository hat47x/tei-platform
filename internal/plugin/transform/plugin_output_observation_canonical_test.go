package transform_test

import (
	"bytes"
	"strings"
	"testing"

	"tei/internal/plugin"
	"tei/internal/plugin/inputprofile"
	"tei/internal/plugin/transform"
)

// The accepted SB-1 bytes are a canonical artifact, not merely a loosely
// decoded manifest that happens to repeat the admitted identity fields.
func TestHostOutputObservationRequiresCanonicalManifestBytes(t *testing.T) {
	def, plan, source, selection, raw := admissionFixture(t)
	prepared, err := inputprofile.PreparePluginPackageInputs(selection, source, plan, raw)
	if err != nil {
		t.Fatal(err)
	}
	result, err := transform.Evaluate(source, prepared.Inputs)
	if err != nil {
		t.Fatal(err)
	}
	oracle, err := plugin.MaterializePluginPackage(def, plan)
	if err != nil || !bytes.Equal(oracle, result.Output.Bytes) {
		t.Fatalf("reference oracle and TEI bytes differ: %v", err)
	}
	if _, err := inputprofile.ObservePackageOutput(prepared.Evidence, result); err != nil {
		t.Fatalf("canonical SB-1 package rejected: %v", err)
	}

	const prefix = `{"formatVersion":1,`
	if !bytes.HasPrefix(oracle, []byte(prefix)) {
		t.Fatal("canonical fixture changed its manifest field order")
	}
	// Inject an identical duplicate. The ordinary typed Go decoder would
	// accept this and keep the exact same manifest and fingerprints.
	duplicate := append([]byte(`{"formatVersion":1,`), oracle[1:]...)

	// Move the first field to the last position, preserving all member
	// values and spelling. JSON order does not change the typed manifest.
	reordered := append([]byte("{"), oracle[len(prefix):len(oracle)-2]...)
	reordered = append(reordered, []byte(",\"formatVersion\":1}\n")...)

	// Keep original members unchanged; the extra field must never be silently
	// ignored when attaching a digest to execution Evidence.
	extra := append(append([]byte(nil), oracle[:len(oracle)-2]...), []byte(",\"unrecognized\":\"data\"}\n")...)
	spaced := append([]byte(" "), oracle...)
	spaced = append(spaced, '\n')
	cases := []struct {
		name  string
		bytes []byte
		want  string
	}{
		{name: "duplicate top-level key", bytes: duplicate, want: "ambiguous or invalid"},
		{name: "unknown output field", bytes: extra, want: "not canonical"},
		{name: "reordered output members", bytes: reordered, want: "not canonical"},
		{name: "whitespace surrounding output", bytes: spaced, want: "not canonical"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			modified := result
			modified.Output = transform.Bytes(tc.bytes)
			if _, err := inputprofile.ObservePackageOutput(prepared.Evidence, modified); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("invalid SB-1 package observed as output Evidence: %v; expected %q", err, tc.want)
			}
		})
	}
}

