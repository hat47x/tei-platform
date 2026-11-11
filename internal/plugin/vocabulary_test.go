package plugin

import "testing"

// BuildOutputKind says what a build produced. A consumer routes the artifact by
// this value, so the set is part of this package's contract (#4032).
func TestBuildOutputKindsAreCovered(t *testing.T) {
	kinds := []BuildOutputKind{BuildOutputRuntimeArtifact, BuildOutputPluginPackage, BuildOutputBuilderArtifact}
	want := []string{"runtime-artifact", "plugin-package", "builder-artifact"}
	if len(kinds) != len(want) {
		t.Fatalf("an output kind was dropped: %d values", len(kinds))
	}
	for i, kind := range kinds {
		if string(kind) != want[i] {
			t.Fatalf("kinds[%d]=%q want %q", i, kind, want[i])
		}
	}
	if BuildOutputBuilderArtifact == BuildOutputRuntimeArtifact {
		t.Fatal("a builder artifact is indistinguishable from a runtime artifact")
	}
}
