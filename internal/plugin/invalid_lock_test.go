package plugin

import (
	"errors"
	"testing"

	"tei/internal/core"
	"tei/internal/environment"
)

// TestInvalidLockIsDistinguishedFromAMismatchedLock pins #3184 at the source:
// the same unusable lock must be classifiable the same way whoever reads it.
// A lock whose bytes are unusable is an InvalidLockError; a well-formed lock
// that does not match the environment or catalog is not, so callers can map
// the first to an input failure and keep the second as a planning outcome.
func TestInvalidLockIsDistinguishedFromAMismatchedLock(t *testing.T) {
	badFormat := Lock{FormatVersion: 0}
	if _, err := (Catalog{}).ResolveEnvironmentWithLock(environment.Environment{ID: "test"}, badFormat); !IsInvalidLock(err) {
		t.Fatalf("format 0 must be an invalid lock, got %v", err)
	}
	if _, err := EnvironmentLockFingerprint(badFormat, core.CanonicalID("test")); !IsInvalidLock(err) {
		t.Fatalf("format 0 fingerprint must be an invalid lock, got %v", err)
	}

	wellFormed := Lock{FormatVersion: LockFormatVersion, Environments: []EnvironmentLock{{Environment: "test"}}}
	if _, err := (Catalog{}).ResolveEnvironmentWithLock(environment.Environment{ID: "other"}, wellFormed); err == nil {
		t.Fatal("a lock without the requested environment must still fail")
	} else if IsInvalidLock(err) {
		t.Fatalf("a lock missing one environment is a mismatch, not an invalid lock: %v", err)
	}

	// A build-plan fingerprint in a format-2 lock cannot be used either.
	misplaced := Lock{FormatVersion: LockFormatVersion, Environments: []EnvironmentLock{{
		Environment: "test",
		Plugins:     []LockedPlugin{{ID: "x", Version: "1.0.0", Fingerprint: "sha256:00", ArtifactDigest: "sha256:00", BuildPlanFingerprint: "sha256:00"}},
	}}}
	if err := validateLockBuildTrace(misplaced); err == nil {
		t.Fatal("format-2 lock with a build plan fingerprint must be rejected")
	}
	if _, err := EnvironmentLockFingerprint(misplaced, core.CanonicalID("test")); !IsInvalidLock(err) {
		t.Fatalf("misplaced build plan fingerprint must be an invalid lock, got %v", err)
	}

	var notInvalid *InvalidLockError
	if errors.As(nil, &notInvalid) {
		t.Fatal("nil error must not read as an invalid lock")
	}
}
