package workspace

import (
	"encoding/json"
	"fmt"
	"sort"
	"testing"
)

// referenceLoadFingerprint is a stable digest of what Load produced for a
// Workspace: the World as JSON, the managed-file inventory, the Source map and
// the Source Contributions. Every list is emitted in a deterministic order, so
// the value changes only when the loaded content or its discovery order does.
func referenceLoadFingerprint(t *testing.T, r Result) string {
	t.Helper()
	world, err := json.Marshal(r.World)
	if err != nil {
		t.Fatal(err)
	}
	managed, err := json.Marshal(r.ManagedFiles)
	if err != nil {
		t.Fatal(err)
	}
	// encoding/json writes map keys in sorted order.
	sources, err := json.Marshal(r.Sources)
	if err != nil {
		t.Fatal(err)
	}
	implementationSources, err := json.Marshal(r.PluginImplementationSources)
	if err != nil {
		t.Fatal(err)
	}
	contributions := make([]string, 0, len(r.SourceContributions))
	for _, c := range r.SourceContributions {
		contributions = append(contributions, fmt.Sprintf("%s|%s|%s|%s", c.Subject.Kind, c.Subject.ID, c.File, c.Role))
	}
	sort.Strings(contributions)
	encodedContributions, err := json.Marshal(contributions)
	if err != nil {
		t.Fatal(err)
	}
	// Discovery order of the World lists is part of the observable result, so
	// the World JSON above is deliberately not re-sorted.
	return sha256Hex([]byte(fmt.Sprintf("world=%s\nmanaged=%s\nsources=%s\nimplSources=%s\ncontributions=%s\n",
		world, managed, sources, implementationSources, encodedContributions)))
}

// referenceLoadFingerprintOnOriginMain was computed ONCE by running this same
// test on the unmodified origin/main loader (before any production change in
// this slice): `go test -run ^TestLoadReferenceWorkspaceFingerprintIsStable$`
// with the constant empty prints the computed digest in the failure message.
// It is a computed value, not a golden file. If a legitimate change to the
// reference Workspace or to the World schema alters it, recompute it and say so
// in the change that does it.
const referenceLoadFingerprintOnOriginMain = "859b1313bfd151bd99162716bb7545612a3f32ff729f32c534d1664e6f4541ca"

func TestLoadReferenceWorkspaceFingerprintIsStable(t *testing.T) {
	root := copyReferenceWorkspace(t)
	result, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	got := referenceLoadFingerprint(t, result)
	if got != referenceLoadFingerprintOnOriginMain {
		t.Fatalf("reference Workspace fingerprint = %s, want %s", got, referenceLoadFingerprintOnOriginMain)
	}
	if len(result.ManagedFiles) == 0 || len(result.Sources) == 0 {
		t.Fatalf("fingerprint over an empty result proves nothing: managed=%d sources=%d", len(result.ManagedFiles), len(result.Sources))
	}
}
