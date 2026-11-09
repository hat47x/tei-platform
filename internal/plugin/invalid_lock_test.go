package plugin

import (
	"errors"
	"testing"

	"tei/internal/core"
	"tei/internal/environment"
)

// TestInvalidLockCoversByteFormAndDisagreement pins #3184 and #3765 at the
// source: the same supplied lock must be classifiable the same way whoever
// reads it. #3184 covered only the byte form. #3765 showed that a well-formed
// lock disagreeing with the declaration (an environment it does not pin, a
// version the catalog does not offer, a diverged fingerprint) is also something
// only the caller can fix, and that leaving it as an ordinary resolve failure
// made plugin-resolve answer exit 2 while plugin-runtime-plan answered exit 1
// for the same file. Both kinds are InvalidLockError now.
func TestInvalidLockCoversByteFormAndDisagreement(t *testing.T) {
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
	} else if !IsInvalidLock(err) {
		t.Fatalf("a lock that does not pin the requested environment is a supplied input the caller can fix: %v", err)
	}
	// The same rule holds where the fingerprint is computed rather than resolved.
	if _, err := EnvironmentLockFingerprint(wellFormed, core.CanonicalID("other")); !IsInvalidLock(err) {
		t.Fatalf("a missing environment must be an invalid lock on the fingerprint path too: %v", err)
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
