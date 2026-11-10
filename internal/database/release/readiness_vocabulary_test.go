package release_test

import (
	"testing"

	"tei/internal/database/release"
)

// The validator quotes DataReadinesses in its diagnostic, so the slice and Valid
// must accept exactly the same set (#4094).
func TestDataReadinessesSliceAndValidAgree(t *testing.T) {
	for _, value := range release.DataReadinesses {
		if !value.Valid() {
			t.Errorf("DataReadinesses lists %q but Valid rejects it", value)
		}
	}
	for _, invalid := range []release.DataReadiness{"", "readiness-ready", "READY", "partial"} {
		if invalid.Valid() {
			t.Errorf("Valid accepts %q, which is not in DataReadinesses", invalid)
		}
	}
}
