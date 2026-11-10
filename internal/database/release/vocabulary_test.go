package release

import "testing"

// DataReadinesses is quoted by INVALID_DATABASE_RELEASE_DATA_READINESS (#4094).
func TestDataReadinessesAndValidAgree(t *testing.T) {
	declared := []DataReadiness{DataReady, DataPreparing, DataNotReady, DataUnknown}
	if len(declared) != len(DataReadinesses) {
		t.Fatalf("DataReadinesses has %d entries but %d are declared", len(DataReadinesses), len(declared))
	}
	for _, v := range DataReadinesses {
		if !v.Valid() {
			t.Errorf("DataReadinesses lists %q but Valid() rejects it", v)
		}
	}
	for _, bad := range []DataReadiness{"", "Ready", "pending"} {
		if bad.Valid() {
			t.Errorf("Valid() accepted data readiness %q", bad)
		}
	}
}
