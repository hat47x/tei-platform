package plugin_test

import (
	"reflect"
	"strings"
	"testing"

	"tei/internal/plugin"
)

// buildLockFixture returns a lock whose builds are in an order that is not
// sorted, so normalization must reorder them. Each build differs in the first
// key that matters (plugin id), so reversing them changes the order.
func buildLockFixture() plugin.BuildArtifactLock {
	return plugin.BuildArtifactLock{
		FormatVersion: plugin.BuildArtifactLockFormatVersion,
		Builds: []plugin.LockedBuild{
			{
				Plugin:          plugin.Identity{ID: cid("tei.z"), Version: "1.0.0"},
				OutputKind:      plugin.BuildOutputBuilderArtifact,
				Target:          "linux/amd64",
				PlanFingerprint: strings.Repeat("b", 64),
				ArtifactDigest:  "sha256:" + strings.Repeat("e", 64),
			},
			{
				Plugin:          plugin.Identity{ID: cid("tei.a"), Version: "1.0.0"},
				OutputKind:      plugin.BuildOutputBuilderArtifact,
				Target:          "linux/amd64",
				PlanFingerprint: strings.Repeat("a", 64),
				ArtifactDigest:  "sha256:" + strings.Repeat("f", 64),
			},
		},
	}
}

// The build artifact lock fingerprint is what a self-build pins, so two locks
// that name the same builds must fingerprint the same regardless of the order
// they were recorded in. Normalization provides that, and nothing pinned it.
func TestBuildArtifactLockFingerprintIsOrderInvariant(t *testing.T) {
	base := buildLockFixture()
	want, err := plugin.BuildArtifactLockFingerprint(base)
	if err != nil {
		t.Fatalf("fingerprint: %v", err)
	}

	reversed := buildLockFixture()
	reverse(reversed.Builds)
	// The reversal must actually change the order, or this case proves nothing.
	if reflect.DeepEqual(reversed.Builds, buildLockFixture().Builds) {
		t.Fatal("reversing the builds changed nothing")
	}

	got, err := plugin.BuildArtifactLockFingerprint(reversed)
	if err != nil {
		t.Fatalf("fingerprint of the reversed lock: %v", err)
	}
	if got != want {
		t.Fatalf("build order changed the fingerprint:\n got  %s\n want %s", got, want)
	}
}

// Ordering inside a second key must not matter either: two builds of the same
// plugin that differ in output kind or target are ordered by those fields next.
func TestBuildArtifactLockFingerprintIgnoresTieBreakOrder(t *testing.T) {
	base := plugin.BuildArtifactLock{
		FormatVersion: plugin.BuildArtifactLockFormatVersion,
		Builds: []plugin.LockedBuild{
			{Plugin: plugin.Identity{ID: cid("tei.a"), Version: "1.0.0"},
				OutputKind: plugin.BuildOutputBuilderArtifact, Target: "linux/amd64",
				PlanFingerprint: strings.Repeat("1", 64), ArtifactDigest: "sha256:" + strings.Repeat("1", 64)},
			{Plugin: plugin.Identity{ID: cid("tei.a"), Version: "1.0.0"},
				OutputKind: plugin.BuildOutputBuilderArtifact, Target: "windows/amd64",
				PlanFingerprint: strings.Repeat("2", 64), ArtifactDigest: "sha256:" + strings.Repeat("2", 64)},
		},
	}
	want, err := plugin.BuildArtifactLockFingerprint(base)
	if err != nil {
		t.Fatalf("fingerprint: %v", err)
	}

	reversed := base
	reversed.Builds = append([]plugin.LockedBuild(nil), base.Builds...)
	reverse(reversed.Builds)

	got, err := plugin.BuildArtifactLockFingerprint(reversed)
	if err != nil {
		t.Fatalf("fingerprint of the reversed lock: %v", err)
	}
	if got != want {
		t.Fatalf("tie-break order changed the fingerprint:\n got  %s\n want %s", got, want)
	}
}

// Normalizing must not reorder the caller's slice. A normalizer that sorted in
// place would change a value the caller still holds, which is the kind of
// aliasing bug the copy at the top of the function exists to prevent.
func TestNormalizeBuildArtifactLockDoesNotMutateItsInput(t *testing.T) {
	original := buildLockFixture()
	before := append([]plugin.LockedBuild(nil), original.Builds...)

	_ = plugin.NormalizeBuildArtifactLock(original)

	if !reflect.DeepEqual(original.Builds, before) {
		t.Fatalf("normalizing reordered the caller's builds:\n got  %+v\n want %+v", original.Builds, before)
	}

	// And the returned value must be sorted, so the copy is not the only thing
	// that happened.
	normalized := plugin.NormalizeBuildArtifactLock(original)
	if normalized.Builds[0].Plugin.ID != cid("tei.a") {
		t.Fatalf("normalized builds are not in canonical order: %+v", normalized.Builds)
	}
}

// A real difference in the lock must still change the fingerprint, so the
// invariance above is not "the fingerprint ignores the builds".
func TestBuildArtifactLockFingerprintChangesWithTheLock(t *testing.T) {
	base := buildLockFixture()
	want, err := plugin.BuildArtifactLockFingerprint(base)
	if err != nil {
		t.Fatalf("fingerprint: %v", err)
	}

	cases := []struct {
		name string
		edit func(*plugin.BuildArtifactLock)
	}{
		{name: "artifact digest", edit: func(l *plugin.BuildArtifactLock) {
			l.Builds[0].ArtifactDigest = "sha256:" + strings.Repeat("9", 64)
		}},
		{name: "plan fingerprint", edit: func(l *plugin.BuildArtifactLock) {
			l.Builds[0].PlanFingerprint = strings.Repeat("9", 64)
		}},
		{name: "target", edit: func(l *plugin.BuildArtifactLock) {
			l.Builds[0].Target = "darwin/arm64"
		}},
		{name: "output kind", edit: func(l *plugin.BuildArtifactLock) {
			l.Builds[0].OutputKind = plugin.BuildOutputRuntimeArtifact
		}},
		{name: "plugin version", edit: func(l *plugin.BuildArtifactLock) {
			l.Builds[0].Plugin.Version = "1.0.1"
		}},
		{name: "dropped build", edit: func(l *plugin.BuildArtifactLock) {
			l.Builds = l.Builds[:1]
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			changed := buildLockFixture()
			tc.edit(&changed)
			got, err := plugin.BuildArtifactLockFingerprint(changed)
			if err != nil {
				t.Fatalf("fingerprint: %v", err)
			}
			if got == want {
				t.Fatalf("changing the %s did not change the fingerprint", tc.name)
			}
		})
	}
}

// An unsupported format is refused before any fingerprint is produced, so a
// caller cannot pin bytes the lock type does not describe.
func TestBuildArtifactLockFingerprintRefusesAnUnsupportedFormat(t *testing.T) {
	lock := buildLockFixture()
	lock.FormatVersion = 99
	if _, err := plugin.BuildArtifactLockFingerprint(lock); err == nil {
		t.Fatal("an unsupported format must be refused")
	}
}

// The empty lock is the common case for a workspace with no builds, and it must
// have a fingerprint rather than an error.
func TestAnEmptyBuildArtifactLockHasAFingerprint(t *testing.T) {
	empty := plugin.BuildArtifactLock{FormatVersion: plugin.BuildArtifactLockFormatVersion}
	got, err := plugin.BuildArtifactLockFingerprint(empty)
	if err != nil {
		t.Fatalf("an empty lock must be fingerprintable: %v", err)
	}
	if got == "" {
		t.Fatal("an empty lock produced no fingerprint")
	}
}
